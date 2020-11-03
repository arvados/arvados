// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package costanalyzer

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Dict is a helper type so we don't have to write out 'map[string]interface{}' every time.
type Dict map[string]interface{}

// LegacyNodeInfo is a struct for records created by Arvados Node Manager (Arvados <= 1.4.3)
// Example:
// {
//    "total_cpu_cores":2,
//    "total_scratch_mb":33770,
//    "cloud_node":
//      {
//        "price":0.1,
//        "size":"m4.large"
//      },
//     "total_ram_mb":7986
// }
type LegacyNodeInfo struct {
	CPUCores  int64           `json:"total_cpu_cores"`
	ScratchMb int64           `json:"total_scratch_mb"`
	RAMMb     int64           `json:"total_ram_mb"`
	CloudNode LegacyCloudNode `json:"cloud_node"`
}

// LegacyCloudNode is a struct for records created by Arvados Node Manager (Arvados <= 1.4.3)
type LegacyCloudNode struct {
	Price float64 `json:"price"`
	Size  string  `json:"size"`
}

// Node is a struct for records created by Arvados Dispatch Cloud (Arvados >= 2.0.0)
// Example:
// {
//    "Name": "Standard_D1_v2",
//    "ProviderType": "Standard_D1_v2",
//    "VCPUs": 1,
//    "RAM": 3584000000,
//    "Scratch": 50000000000,
//    "IncludedScratch": 50000000000,
//    "AddedScratch": 0,
//    "Price": 0.057,
//    "Preemptible": false
//}
type Node struct {
	VCPUs        int64
	Scratch      int64
	RAM          int64
	Price        float64
	Name         string
	ProviderType string
	Preemptible  bool
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func parseFlags(prog string, args []string, loader *config.Loader, logger *logrus.Logger, stderr io.Writer) (exitCode int, uuids arrayFlags, resultsDir string) {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), `
Usage:
  %s [options ...]

	This program analyzes the cost of Arvados container requests. For each uuid
	supplied, it creates a CSV report that lists all the containers used to
	fulfill the container request, together with the machine type and cost of
	each container.

	When supplied with the uuid of a container request, it will calculate the
	cost of that container request and all its children. When suplied with a
	project uuid or when supplied with multiple container request uuids, it will
	create a CSV report for each supplied uuid, as well as a CSV file with
	aggregate cost accounting for all supplied uuids. The aggregate cost report
	takes container reuse into account: if a container was reused between several
	container requests, its cost will only be counted once.

	To get the node costs, the progam queries the Arvados API for current cost
	data for each node type used. This means that the reported cost always
	reflects the cost data as currently defined in the Arvados API configuration
	file.

	Caveats:
	- the Arvados API configuration cost data may be out of sync with the cloud
	provider.
	- when generating reports for older container requests, the cost data in the
	Arvados API configuration file may have changed since the container request
	was fulfilled. This program uses the cost data stored at the time of the
	execution of the container, stored in the 'node.json' file in its log
	collection.

	In order to get the data for the uuids supplied, the ARVADOS_API_HOST and
	ARVADOS_API_TOKEN environment variables must be set.

Options:
`, prog)
		flags.PrintDefaults()
	}
	loglevel := flags.String("log-level", "info", "logging level (debug, info, ...)")
	resultsDir = *flags.String("output", "results", "output directory for the CSV reports")
	flags.Var(&uuids, "uuid", "Toplevel project or container request uuid. May be specified more than once.")
	err := flags.Parse(args)
	if err == flag.ErrHelp {
		exitCode = 1
		return
	} else if err != nil {
		exitCode = 2
		return
	}

	if len(uuids) < 1 {
		logger.Errorf("Error: no uuid(s) provided")
		flags.Usage()
		exitCode = 2
		return
	}

	lvl, err := logrus.ParseLevel(*loglevel)
	if err != nil {
		exitCode = 2
		return
	}
	logger.SetLevel(lvl)
	return
}

func ensureDirectory(logger *logrus.Logger, dir string) (err error) {
	statData, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			logger.Errorf("Error creating directory %s: %s\n", dir, err.Error())
			return
		}
	} else {
		if !statData.IsDir() {
			logger.Errorf("The path %s is not a directory\n", dir)
			return
		}
	}
	return
}

func addContainerLine(logger *logrus.Logger, node interface{}, cr Dict, container Dict) (csv string, cost float64) {
	csv = cr["uuid"].(string) + ","
	csv += cr["name"].(string) + ","
	csv += container["uuid"].(string) + ","
	csv += container["state"].(string) + ","
	if container["started_at"] != nil {
		csv += container["started_at"].(string) + ","
	} else {
		csv += ","
	}

	var delta time.Duration
	if container["finished_at"] != nil {
		csv += container["finished_at"].(string) + ","
		finishedTimestamp, err := time.Parse("2006-01-02T15:04:05.000000000Z", container["finished_at"].(string))
		if err != nil {
			fmt.Println(err)
		}
		startedTimestamp, err := time.Parse("2006-01-02T15:04:05.000000000Z", container["started_at"].(string))
		if err != nil {
			fmt.Println(err)
		}
		delta = finishedTimestamp.Sub(startedTimestamp)
		csv += strconv.FormatFloat(delta.Seconds(), 'f', 0, 64) + ","
	} else {
		csv += ",,"
	}
	var price float64
	var size string
	switch n := node.(type) {
	case Node:
		price = n.Price
		size = n.ProviderType
	case LegacyNodeInfo:
		price = n.CloudNode.Price
		size = n.CloudNode.Size
	default:
		logger.Warn("WARNING: unknown node type found!")
	}
	cost = delta.Seconds() / 3600 * price
	csv += size + "," + strconv.FormatFloat(price, 'f', 8, 64) + "," + strconv.FormatFloat(cost, 'f', 8, 64) + "\n"
	return
}

func loadCachedObject(logger *logrus.Logger, file string, uuid string) (reload bool, object Dict) {
	reload = true
	// See if we have a cached copy of this object
	if _, err := os.Stat(file); err == nil {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			logger.Errorf("error reading %q: %s", file, err)
			return
		}
		err = json.Unmarshal(data, &object)
		if err != nil {
			logger.Errorf("failed to unmarshal json: %s: %s", data, err)
			return
		}

		// See if it is in a final state, if that makes sense
		// Projects (j7d0g) do not have state so they should always be reloaded
		if !strings.Contains(uuid, "-j7d0g-") {
			if object["state"].(string) == "Complete" || object["state"].(string) == "Failed" {
				reload = false
				logger.Debugf("Loaded object %s from local cache (%s)\n", uuid, file)
				return
			}
		}
	}
	return
}

// Load an Arvados object.
func loadObject(logger *logrus.Logger, arv *arvadosclient.ArvadosClient, path string, uuid string) (object Dict, err error) {
	err = ensureDirectory(logger, path)
	if err != nil {
		return
	}

	file := path + "/" + uuid + ".json"

	var reload bool
	reload, object = loadCachedObject(logger, file, uuid)

	if reload {
		var err error
		if strings.Contains(uuid, "-j7d0g-") {
			err = arv.Get("groups", uuid, nil, &object)
		} else if strings.Contains(uuid, "-xvhdp-") {
			err = arv.Get("container_requests", uuid, nil, &object)
		} else if strings.Contains(uuid, "-dz642-") {
			err = arv.Get("containers", uuid, nil, &object)
		} else {
			err = arv.Get("jobs", uuid, nil, &object)
		}
		if err != nil {
			logger.Fatalf("Error loading object with UUID %q:\n  %s\n", uuid, err)
		}
		encoded, err := json.MarshalIndent(object, "", " ")
		if err != nil {
			logger.Fatalf("Error marshaling object with UUID %q:\n  %s\n", uuid, err)
		}
		err = ioutil.WriteFile(file, encoded, 0644)
		if err != nil {
			logger.Fatalf("Error writing file %s:\n  %s\n", file, err)
		}
	}
	return
}

func getNode(arv *arvadosclient.ArvadosClient, arv2 *arvados.Client, kc *keepclient.KeepClient, itemMap Dict) (node interface{}, err error) {
	if _, ok := itemMap["log_uuid"]; ok {
		if itemMap["log_uuid"] == nil {
			err = errors.New("No log collection")
			return
		}

		var collection arvados.Collection
		err = arv.Get("collections", itemMap["log_uuid"].(string), nil, &collection)
		if err != nil {
			err = fmt.Errorf("Error getting collection: %s", err)
			return
		}

		var fs arvados.CollectionFileSystem
		fs, err = collection.FileSystem(arv2, kc)
		if err != nil {
			err = fmt.Errorf("Error opening collection as filesystem: %s", err)
			return
		}
		var f http.File
		f, err = fs.Open("node.json")
		if err != nil {
			err = fmt.Errorf("Error opening file 'node.json' in collection %s: %s", itemMap["log_uuid"].(string), err)
			return
		}

		var nodeDict Dict
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(f)
		if err != nil {
			err = fmt.Errorf("Error reading file 'node.json' in collection %s: %s", itemMap["log_uuid"].(string), err)
			return
		}
		contents := buf.String()
		f.Close()

		err = json.Unmarshal([]byte(contents), &nodeDict)
		if err != nil {
			err = fmt.Errorf("Error unmarshalling: %s", err)
			return
		}
		if val, ok := nodeDict["properties"]; ok {
			var encoded []byte
			encoded, err = json.MarshalIndent(val, "", " ")
			if err != nil {
				err = fmt.Errorf("Error marshalling: %s", err)
				return
			}
			// node is type LegacyNodeInfo
			var newNode LegacyNodeInfo
			err = json.Unmarshal(encoded, &newNode)
			if err != nil {
				err = fmt.Errorf("Error unmarshalling: %s", err)
				return
			}
			node = newNode
		} else {
			// node is type Node
			var newNode Node
			err = json.Unmarshal([]byte(contents), &newNode)
			if err != nil {
				err = fmt.Errorf("Error unmarshalling: %s", err)
				return
			}
			node = newNode
		}
	}
	return
}

func handleProject(logger *logrus.Logger, uuid string, arv *arvadosclient.ArvadosClient, arv2 *arvados.Client, kc *keepclient.KeepClient, resultsDir string) (cost map[string]float64) {

	cost = make(map[string]float64)

	project, err := loadObject(logger, arv, resultsDir+"/"+uuid, uuid)
	if err != nil {
		logger.Fatalf("Error loading object %s: %s\n", uuid, err.Error())
	}

	// arv -f uuid container_request list --filters '[["owner_uuid","=","<someuuid>"],["requesting_container_uuid","=",null]]'

	// Now find all container requests that have the container we found above as requesting_container_uuid
	var childCrs map[string]interface{}
	filterset := []arvados.Filter{
		{
			Attr:     "owner_uuid",
			Operator: "=",
			Operand:  project["uuid"].(string),
		},
		{
			Attr:     "requesting_container_uuid",
			Operator: "=",
			Operand:  nil,
		},
	}
	err = arv.List("container_requests", arvadosclient.Dict{"filters": filterset, "limit": 10000}, &childCrs)
	if err != nil {
		logger.Fatalf("Error querying container_requests: %s\n", err.Error())
	}
	if value, ok := childCrs["items"]; ok {
		logger.Infof("Collecting top level container requests in project %s\n", uuid)
		items := value.([]interface{})
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			for k, v := range generateCrCsv(logger, itemMap["uuid"].(string), arv, arv2, kc, resultsDir) {
				cost[k] = v
			}
		}
	} else {
		logger.Infof("No top level container requests found in project %s\n", uuid)
	}
	return
}

func generateCrCsv(logger *logrus.Logger, uuid string, arv *arvadosclient.ArvadosClient, arv2 *arvados.Client, kc *keepclient.KeepClient, resultsDir string) (cost map[string]float64) {

	cost = make(map[string]float64)

	csv := "CR UUID,CR name,Container UUID,State,Started At,Finished At,Duration in seconds,Compute node type,Hourly node cost,Total cost\n"
	var tmpCsv string
	var tmpTotalCost float64
	var totalCost float64

	// This is a container request, find the container
	cr, err := loadObject(logger, arv, resultsDir+"/"+uuid, uuid)
	if err != nil {
		log.Fatalf("Error loading object %s: %s", uuid, err)
	}
	container, err := loadObject(logger, arv, resultsDir+"/"+uuid, cr["container_uuid"].(string))
	if err != nil {
		log.Fatalf("Error loading object %s: %s", cr["container_uuid"].(string), err)
	}

	topNode, err := getNode(arv, arv2, kc, cr)
	if err != nil {
		log.Fatalf("Error getting node %s: %s\n", cr["uuid"], err)
	}
	tmpCsv, totalCost = addContainerLine(logger, topNode, cr, container)
	csv += tmpCsv
	totalCost += tmpTotalCost

	cost[container["uuid"].(string)] = totalCost

	// Now find all container requests that have the container we found above as requesting_container_uuid
	var childCrs map[string]interface{}
	filterset := []arvados.Filter{
		{
			Attr:     "requesting_container_uuid",
			Operator: "=",
			Operand:  container["uuid"].(string),
		}}
	err = arv.List("container_requests", arvadosclient.Dict{"filters": filterset, "limit": 10000}, &childCrs)
	if err != nil {
		log.Fatal("error querying container_requests", err.Error())
	}
	if value, ok := childCrs["items"]; ok {
		logger.Infof("Collecting child containers for container request %s", uuid)
		items := value.([]interface{})
		for _, item := range items {
			logger.Info(".")
			itemMap := item.(map[string]interface{})
			node, err := getNode(arv, arv2, kc, itemMap)
			if err != nil {
				log.Fatalf("Error getting node %s: %s\n", itemMap["uuid"], err)
			}
			logger.Debug("\nChild container: " + itemMap["container_uuid"].(string) + "\n")
			c2, err := loadObject(logger, arv, resultsDir+"/"+uuid, itemMap["container_uuid"].(string))
			if err != nil {
				log.Fatalf("Error loading object %s: %s", cr["container_uuid"].(string), err)
			}
			tmpCsv, tmpTotalCost = addContainerLine(logger, node, itemMap, c2)
			cost[itemMap["container_uuid"].(string)] = tmpTotalCost
			csv += tmpCsv
			totalCost += tmpTotalCost
		}
	}
	logger.Info(" done\n")

	csv += "TOTAL,,,,,,,,," + strconv.FormatFloat(totalCost, 'f', 8, 64) + "\n"

	// Write the resulting CSV file
	fName := resultsDir + "/" + uuid + ".csv"
	err = ioutil.WriteFile(fName, []byte(csv), 0644)
	if err != nil {
		logger.Errorf("Error writing file with path %s: %s\n", fName, err.Error())
		os.Exit(1)
	}

	return
}

func costanalyzer(prog string, args []string, loader *config.Loader, logger *logrus.Logger, stdout, stderr io.Writer) (exitcode int) {
	exitcode, uuids, resultsDir := parseFlags(prog, args, loader, logger, stderr)
	if exitcode != 0 {
		return
	}
	err := ensureDirectory(logger, resultsDir)
	if err != nil {
		exitcode = 3
		return
	}

	// Arvados Client setup
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		logger.Errorf("error creating Arvados object: %s", err)
		os.Exit(1)
	}
	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		logger.Errorf("error creating Keep object: %s", err)
		os.Exit(1)
	}

	arv2 := arvados.NewClientFromEnv()

	cost := make(map[string]float64)
	for _, uuid := range uuids {
		if strings.Contains(uuid, "-j7d0g-") {
			// This is a project (group)
			for k, v := range handleProject(logger, uuid, arv, arv2, kc, resultsDir) {
				cost[k] = v
			}
		} else if strings.Contains(uuid, "-xvhdp-") {
			// This is a container request
			for k, v := range generateCrCsv(logger, uuid, arv, arv2, kc, resultsDir) {
				cost[k] = v
			}
		} else if strings.Contains(uuid, "-tpzed-") {
			// This is a user. The "Home" project for a user is not a real project.
			// It is identified by the user uuid. As such, cost analysis for the
			// "Home" project is not supported by this program.
			logger.Errorf("Cost analysis is not supported for the 'Home' project: %s", uuid)
		}
	}

	logger.Info("\n")
	for k := range cost {
		logger.Infof("Uuid report in %s/%s.csv\n", resultsDir, k)
	}

	if len(cost) == 0 {
		logger.Info("Nothing to do!\n")
		return
	}

	var csv string

	csv = "# Aggregate cost accounting for uuids:\n"
	for _, uuid := range uuids {
		csv += "# " + uuid + "\n"
	}

	var total float64
	for k, v := range cost {
		csv += k + "," + strconv.FormatFloat(v, 'f', 8, 64) + "\n"
		total += v
	}

	csv += "TOTAL," + strconv.FormatFloat(total, 'f', 8, 64) + "\n"

	// Write the resulting CSV file
	aFile := resultsDir + "/" + time.Now().Format("2006-01-02-15-04-05") + "-aggregate-costaccounting.csv"
	err = ioutil.WriteFile(aFile, []byte(csv), 0644)
	if err != nil {
		logger.Errorf("Error writing file with path %s: %s\n", aFile, err.Error())
		os.Exit(1)
	} else {
		logger.Infof("\nAggregate cost accounting for all supplied uuids in %s\n", aFile)
	}
	return
}
