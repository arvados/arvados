// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package costanalyzer

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/sirupsen/logrus"
)

const timestampFormat = "2006-01-02T15:04:05"

type nodeInfo struct {
	// Legacy (records created by Arvados Node Manager with Arvados <= 1.4.3)
	Properties struct {
		CloudNode struct {
			Price float64
			Size  string
		} `json:"cloud_node"`
	}
	// Modern
	ProviderType string
	Price        float64
	Preemptible  bool
}

type consumption struct {
	cost     float64
	duration float64
}

func (c *consumption) Add(n consumption) {
	c.cost += n.cost
	c.duration += n.duration
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	for _, s := range strings.Split(value, ",") {
		*i = append(*i, s)
	}
	return nil
}

func (c *command) parseFlags(prog string, args []string, logger *logrus.Logger, stderr io.Writer) (exitCode int, err error) {
	var beginStr, endStr string
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), `
Usage:
  %s [options ...] [UUID ...]

	This program analyzes the cost of Arvados container requests and calculates
	the total cost across all requests. At least one UUID or a timestamp range
	must be specified.

	When the '-output' option is specified, a set of CSV files with cost details
	will be written to the provided directory. Each file is a CSV report that lists
	all the containers used to fulfill the container request, together with the
	machine type and cost of each container.

	When supplied with the UUID of a container request, it will calculate the
	cost of that container request and all its children.

	When supplied with the UUID of a collection, it will see if there is a
	container_request UUID in the properties of the collection, and if so, it
	will calculate the cost of that container request and all its children.

	When supplied with a project UUID or when supplied with multiple container
	request or collection UUIDs, it will calculate the total cost for all
	supplied UUIDs.

	When supplied with a 'begin' and 'end' timestamp (format:
	%s), it will calculate the cost for all top-level container
	requests whose containers finished during the specified interval.

	The total cost calculation takes container reuse into account: if a container
	was reused between several container requests, its cost will only be counted
	once.

	Caveats:

	- This program uses the cost data from config.yml at the time of the
	execution of the container, stored in the 'node.json' file in its log
	collection. If the cost data was not correctly configured at the time the
	container was executed, the output from this program will be incorrect.

	- If a container was run on a preemptible ("spot") instance, the cost data
	reported by this program may be wildly inaccurate, because it does not have
	access to the spot pricing in effect for the node then the container ran. The
	UUID report file that is generated when the '-output' option is specified has
	a column that indicates the preemptible state of the instance that ran the
	container.

	- This program does not take into account overhead costs like the time spent
	starting and stopping compute nodes that run containers, the cost of the
	permanent cloud nodes that provide the Arvados services, the cost of data
	stored in Arvados, etc.

	- When provided with a project UUID, subprojects will not be considered.

	In order to get the data for the UUIDs supplied, the ARVADOS_API_HOST and
	ARVADOS_API_TOKEN environment variables must be set.

	This program prints the total dollar amount from the aggregate cost
	accounting across all provided UUIDs on stdout.

Options:
`, prog, timestampFormat)
		flags.PrintDefaults()
	}
	loglevel := flags.String("log-level", "info", "logging `level` (debug, info, ...)")
	flags.StringVar(&c.resultsDir, "output", "", "output `directory` for the CSV reports")
	flags.StringVar(&beginStr, "begin", "", fmt.Sprintf("timestamp `begin` for date range operation (format: %s)", timestampFormat))
	flags.StringVar(&endStr, "end", "", fmt.Sprintf("timestamp `end` for date range operation (format: %s)", timestampFormat))
	flags.BoolVar(&c.cache, "cache", true, "create and use a local disk cache of Arvados objects")
	err = flags.Parse(args)
	if err == flag.ErrHelp {
		err = nil
		exitCode = 1
		return
	} else if err != nil {
		exitCode = 2
		return
	}
	c.uuids = flags.Args()

	if (len(beginStr) != 0 && len(endStr) == 0) || (len(beginStr) == 0 && len(endStr) != 0) {
		flags.Usage()
		err = fmt.Errorf("When specifying a date range, both begin and end must be specified")
		exitCode = 2
		return
	}

	if len(beginStr) != 0 {
		var errB, errE error
		c.begin, errB = time.Parse(timestampFormat, beginStr)
		c.end, errE = time.Parse(timestampFormat, endStr)
		if (errB != nil) || (errE != nil) {
			flags.Usage()
			err = fmt.Errorf("When specifying a date range, both begin and end must be of the format %s %+v, %+v", timestampFormat, errB, errE)
			exitCode = 2
			return
		}
	}

	if (len(c.uuids) < 1) && (len(beginStr) == 0) {
		flags.Usage()
		err = fmt.Errorf("error: no uuid(s) provided")
		exitCode = 2
		return
	}

	lvl, err := logrus.ParseLevel(*loglevel)
	if err != nil {
		exitCode = 2
		return
	}
	logger.SetLevel(lvl)
	if !c.cache {
		logger.Debug("Caching disabled")
	}
	return
}

func ensureDirectory(logger *logrus.Logger, dir string) (err error) {
	statData, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			return fmt.Errorf("error creating directory %s: %s", dir, err.Error())
		}
	} else {
		if !statData.IsDir() {
			return fmt.Errorf("the path %s is not a directory", dir)
		}
	}
	return
}

func addContainerLine(logger *logrus.Logger, node nodeInfo, cr arvados.ContainerRequest, container arvados.Container) (string, consumption) {
	var csv string
	var containerConsumption consumption
	csv = cr.UUID + ","
	csv += cr.Name + ","
	csv += container.UUID + ","
	csv += string(container.State) + ","
	if container.StartedAt != nil {
		csv += container.StartedAt.String() + ","
	} else {
		csv += ","
	}

	var delta time.Duration
	if container.FinishedAt != nil {
		csv += container.FinishedAt.String() + ","
		delta = container.FinishedAt.Sub(*container.StartedAt)
		csv += strconv.FormatFloat(delta.Seconds(), 'f', 3, 64) + ","
	} else {
		csv += ",,"
	}
	var price float64
	var size string
	if node.Properties.CloudNode.Price != 0 {
		price = node.Properties.CloudNode.Price
		size = node.Properties.CloudNode.Size
	} else {
		price = node.Price
		size = node.ProviderType
	}
	containerConsumption.cost = delta.Seconds() / 3600 * price
	containerConsumption.duration = delta.Seconds()
	csv += size + "," + fmt.Sprintf("%+v", node.Preemptible) + "," + strconv.FormatFloat(price, 'f', 8, 64) + "," + strconv.FormatFloat(containerConsumption.cost, 'f', 8, 64) + "\n"
	return csv, containerConsumption
}

func loadCachedObject(logger *logrus.Logger, file string, uuid string, object interface{}) (reload bool) {
	reload = true
	if strings.Contains(uuid, "-j7d0g-") || strings.Contains(uuid, "-4zz18-") {
		// We do not cache projects or collections, they have no final state
		return
	}
	// See if we have a cached copy of this object
	_, err := os.Stat(file)
	if err != nil {
		return
	}
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
	switch v := object.(type) {
	case *arvados.ContainerRequest:
		if v.State == arvados.ContainerRequestStateFinal {
			reload = false
			logger.Debugf("Loaded object %s from local cache (%s)", uuid, file)
		}
	case *arvados.Container:
		if v.State == arvados.ContainerStateComplete || v.State == arvados.ContainerStateCancelled {
			reload = false
			logger.Debugf("Loaded object %s from local cache (%s)", uuid, file)
		}
	}
	return
}

// Load an Arvados object.
func loadObject(logger *logrus.Logger, ac *arvados.Client, path string, uuid string, cache bool, object interface{}) (err error) {
	file := uuid + ".json"

	var reload bool
	var cacheDir string

	if !cache {
		reload = true
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			reload = true
			logger.Info("Unable to determine current user home directory, not using cache")
		} else {
			cacheDir = homeDir + "/.cache/arvados/costanalyzer/"
			err = ensureDirectory(logger, cacheDir)
			if err != nil {
				reload = true
				logger.Infof("Unable to create cache directory at %s, not using cache: %s", cacheDir, err.Error())
			} else {
				reload = loadCachedObject(logger, cacheDir+file, uuid, object)
			}
		}
	}
	if !reload {
		return
	}

	if strings.Contains(uuid, "-j7d0g-") {
		err = ac.RequestAndDecode(&object, "GET", "arvados/v1/groups/"+uuid, nil, nil)
	} else if strings.Contains(uuid, "-xvhdp-") {
		err = ac.RequestAndDecode(&object, "GET", "arvados/v1/container_requests/"+uuid, nil, nil)
	} else if strings.Contains(uuid, "-dz642-") {
		err = ac.RequestAndDecode(&object, "GET", "arvados/v1/containers/"+uuid, nil, nil)
	} else if strings.Contains(uuid, "-4zz18-") {
		err = ac.RequestAndDecode(&object, "GET", "arvados/v1/collections/"+uuid, nil, nil)
	} else {
		err = fmt.Errorf("unsupported object type with UUID %q:\n  %s", uuid, err)
		return
	}
	if err != nil {
		err = fmt.Errorf("error loading object with UUID %q:\n  %s", uuid, err)
		return
	}
	encoded, err := json.MarshalIndent(object, "", " ")
	if err != nil {
		err = fmt.Errorf("error marshaling object with UUID %q:\n  %s", uuid, err)
		return
	}
	if cacheDir != "" {
		err = ioutil.WriteFile(cacheDir+file, encoded, 0644)
		if err != nil {
			err = fmt.Errorf("error writing file %s:\n  %s", file, err)
			return
		}
	}
	return
}

func getNode(arv *arvadosclient.ArvadosClient, ac *arvados.Client, kc *keepclient.KeepClient, cr arvados.ContainerRequest) (node nodeInfo, err error) {
	if cr.LogUUID == "" {
		err = errors.New("no log collection")
		return
	}

	var collection arvados.Collection
	err = ac.RequestAndDecode(&collection, "GET", "arvados/v1/collections/"+cr.LogUUID, nil, nil)
	if err != nil {
		err = fmt.Errorf("error getting collection: %s", err)
		return
	}

	var fs arvados.CollectionFileSystem
	fs, err = collection.FileSystem(ac, kc)
	if err != nil {
		err = fmt.Errorf("error opening collection as filesystem: %s", err)
		return
	}
	var f http.File
	f, err = fs.Open("node.json")
	if err != nil {
		err = fmt.Errorf("error opening file 'node.json' in collection %s: %s", cr.LogUUID, err)
		return
	}

	err = json.NewDecoder(f).Decode(&node)
	if err != nil {
		err = fmt.Errorf("error reading file 'node.json' in collection %s: %s", cr.LogUUID, err)
		return
	}
	return
}

func handleProject(logger *logrus.Logger, uuid string, arv *arvadosclient.ArvadosClient, ac *arvados.Client, kc *keepclient.KeepClient, resultsDir string, cache bool) (cost map[string]consumption, err error) {
	cost = make(map[string]consumption)

	var project arvados.Group
	err = loadObject(logger, ac, uuid, uuid, cache, &project)
	if err != nil {
		return nil, fmt.Errorf("error loading object %s: %s", uuid, err.Error())
	}

	var childCrs map[string]interface{}
	filterset := []arvados.Filter{
		{
			Attr:     "owner_uuid",
			Operator: "=",
			Operand:  project.UUID,
		},
		{
			Attr:     "requesting_container_uuid",
			Operator: "=",
			Operand:  nil,
		},
	}
	err = ac.RequestAndDecode(&childCrs, "GET", "arvados/v1/container_requests", nil, map[string]interface{}{
		"filters": filterset,
		"limit":   10000,
	})
	if err != nil {
		return nil, fmt.Errorf("error querying container_requests: %s", err.Error())
	}
	if value, ok := childCrs["items"]; ok {
		logger.Infof("Collecting top level container requests in project %s", uuid)
		items := value.([]interface{})
		for _, item := range items {
			itemMap := item.(map[string]interface{})
			crInfo, err := generateCrInfo(logger, itemMap["uuid"].(string), arv, ac, kc, resultsDir, cache)
			if err != nil {
				return nil, fmt.Errorf("error generating container_request CSV: %s", err.Error())
			}
			for k, v := range crInfo {
				cost[k] = v
			}
		}
	} else {
		logger.Infof("No top level container requests found in project %s", uuid)
	}
	return
}

func generateCrInfo(logger *logrus.Logger, uuid string, arv *arvadosclient.ArvadosClient, ac *arvados.Client, kc *keepclient.KeepClient, resultsDir string, cache bool) (cost map[string]consumption, err error) {

	cost = make(map[string]consumption)

	csv := "CR UUID,CR name,Container UUID,State,Started At,Finished At,Duration in seconds,Compute node type,Preemptible,Hourly node cost,Total cost\n"
	var tmpCsv string
	var total, tmpTotal consumption
	logger.Debugf("Processing %s", uuid)

	var crUUID = uuid
	if strings.Contains(uuid, "-4zz18-") {
		// This is a collection, find the associated container request (if any)
		var c arvados.Collection
		err = loadObject(logger, ac, uuid, uuid, cache, &c)
		if err != nil {
			return nil, fmt.Errorf("error loading collection object %s: %s", uuid, err)
		}
		value, ok := c.Properties["container_request"]
		if !ok {
			return nil, fmt.Errorf("error: collection %s does not have a 'container_request' property", uuid)
		}
		crUUID, ok = value.(string)
		if !ok {
			return nil, fmt.Errorf("error: collection %s does not have a 'container_request' property of the string type", uuid)
		}
	}

	// This is a container request, find the container
	var cr arvados.ContainerRequest
	err = loadObject(logger, ac, crUUID, crUUID, cache, &cr)
	if err != nil {
		return nil, fmt.Errorf("error loading cr object %s: %s", uuid, err)
	}
	if len(cr.ContainerUUID) == 0 {
		// Nothing to do! E.g. a CR in 'Uncommitted' state.
		logger.Infof("No container associated with container request %s, skipping", crUUID)
		return nil, nil
	}
	var container arvados.Container
	err = loadObject(logger, ac, crUUID, cr.ContainerUUID, cache, &container)
	if err != nil {
		return nil, fmt.Errorf("error loading container object %s: %s", cr.ContainerUUID, err)
	}

	topNode, err := getNode(arv, ac, kc, cr)
	if err != nil {
		logger.Errorf("Skipping container request %s: error getting node %s: %s", cr.UUID, cr.UUID, err)
		return nil, nil
	}
	tmpCsv, total = addContainerLine(logger, topNode, cr, container)
	csv += tmpCsv
	cost[container.UUID] = total

	// Find all container requests that have the container we found above as requesting_container_uuid
	var childCrs arvados.ContainerRequestList
	filterset := []arvados.Filter{
		{
			Attr:     "requesting_container_uuid",
			Operator: "=",
			Operand:  container.UUID,
		}}
	err = ac.RequestAndDecode(&childCrs, "GET", "arvados/v1/container_requests", nil, map[string]interface{}{
		"filters": filterset,
		"limit":   10000,
	})
	if err != nil {
		return nil, fmt.Errorf("error querying container_requests: %s", err.Error())
	}
	logger.Infof("Collecting child containers for container request %s (%s)", crUUID, container.FinishedAt)
	progressTicker := time.NewTicker(5 * time.Second)
	defer progressTicker.Stop()
	for i, cr2 := range childCrs.Items {
		select {
		case <-progressTicker.C:
			logger.Infof("... %d of %d", i+1, len(childCrs.Items))
		default:
		}
		node, err := getNode(arv, ac, kc, cr2)
		if err != nil {
			logger.Errorf("Skipping container request %s: error getting node %s: %s", cr2.UUID, cr2.UUID, err)
			continue
		}
		logger.Debug("Child container: " + cr2.ContainerUUID)
		var c2 arvados.Container
		err = loadObject(logger, ac, cr.UUID, cr2.ContainerUUID, cache, &c2)
		if err != nil {
			return nil, fmt.Errorf("error loading object %s: %s", cr2.ContainerUUID, err)
		}
		tmpCsv, tmpTotal = addContainerLine(logger, node, cr2, c2)
		cost[cr2.ContainerUUID] = tmpTotal
		csv += tmpCsv
		total.Add(tmpTotal)
	}
	logger.Debug("Done collecting child containers")

	csv += "TOTAL,,,,,," + strconv.FormatFloat(total.duration, 'f', 3, 64) + ",,,," + strconv.FormatFloat(total.cost, 'f', 2, 64) + "\n"

	if resultsDir != "" {
		// Write the resulting CSV file
		fName := resultsDir + "/" + crUUID + ".csv"
		err = ioutil.WriteFile(fName, []byte(csv), 0644)
		if err != nil {
			return nil, fmt.Errorf("error writing file with path %s: %s", fName, err.Error())
		}
		logger.Infof("\nUUID report in %s", fName)
	}

	return
}

func (c *command) costAnalyzer(prog string, args []string, logger *logrus.Logger, stdout, stderr io.Writer) (exitcode int, err error) {
	exitcode, err = c.parseFlags(prog, args, logger, stderr)

	if exitcode != 0 {
		return
	}
	if c.resultsDir != "" {
		err = ensureDirectory(logger, c.resultsDir)
		if err != nil {
			exitcode = 3
			return
		}
	}

	uuidChannel := make(chan string)

	// Arvados Client setup
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		err = fmt.Errorf("error creating Arvados object: %s", err)
		exitcode = 1
		return
	}
	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		err = fmt.Errorf("error creating Keep object: %s", err)
		exitcode = 1
		return
	}

	ac := arvados.NewClientFromEnv()

	// Populate uuidChannel with the requested uuid list
	go func() {
		defer close(uuidChannel)
		for _, uuid := range c.uuids {
			uuidChannel <- uuid
		}

		if !c.begin.IsZero() {
			initialParams := arvados.ResourceListParams{
				Filters: []arvados.Filter{{"container.finished_at", ">=", c.begin}, {"container.finished_at", "<", c.end}, {"requesting_container_uuid", "=", nil}},
				Order:   "created_at",
			}
			params := initialParams
			for {
				// This list variable must be a new one declared
				// inside the loop: otherwise, items in the API
				// response would get deep-merged into the items
				// loaded in previous iterations.
				var list arvados.ContainerRequestList

				err := ac.RequestAndDecode(&list, "GET", "arvados/v1/container_requests", nil, params)
				if err != nil {
					logger.Errorf("Error getting container request list from Arvados API: %s", err)
					break
				}
				if len(list.Items) == 0 {
					break
				}

				for _, i := range list.Items {
					uuidChannel <- i.UUID
				}
				params.Offset += len(list.Items)
			}

		}
	}()

	cost := make(map[string]consumption)

	for uuid := range uuidChannel {
		logger.Debugf("Considering %s", uuid)
		if strings.Contains(uuid, "-j7d0g-") {
			// This is a project (group)
			cost, err = handleProject(logger, uuid, arv, ac, kc, c.resultsDir, c.cache)
			if err != nil {
				exitcode = 1
				return
			}
			for k, v := range cost {
				cost[k] = v
			}
		} else if strings.Contains(uuid, "-xvhdp-") || strings.Contains(uuid, "-4zz18-") {
			// This is a container request
			var crInfo map[string]consumption
			crInfo, err = generateCrInfo(logger, uuid, arv, ac, kc, c.resultsDir, c.cache)
			if err != nil {
				err = fmt.Errorf("error generating CSV for uuid %s: %s", uuid, err.Error())
				exitcode = 2
				return
			}
			for k, v := range crInfo {
				cost[k] = v
			}
		} else if strings.Contains(uuid, "-tpzed-") {
			// This is a user. The "Home" project for a user is not a real project.
			// It is identified by the user uuid. As such, cost analysis for the
			// "Home" project is not supported by this program. Skip this uuid, but
			// keep going.
			logger.Errorf("cost analysis is not supported for the 'Home' project: %s", uuid)
		} else {
			logger.Errorf("this argument does not look like a uuid: %s", uuid)
			exitcode = 3
			return
		}
	}

	if len(cost) == 0 {
		logger.Info("Nothing to do!")
		return
	}

	var csv string

	csv = "# Aggregate cost accounting for uuids:\n# UUID, Duration in seconds, Total cost\n"
	for _, uuid := range c.uuids {
		csv += "# " + uuid + "\n"
	}

	var total consumption
	for k, v := range cost {
		csv += k + "," + strconv.FormatFloat(v.duration, 'f', 3, 64) + "," + strconv.FormatFloat(v.cost, 'f', 8, 64) + "\n"
		total.Add(v)
	}

	csv += "TOTAL," + strconv.FormatFloat(total.duration, 'f', 3, 64) + "," + strconv.FormatFloat(total.cost, 'f', 2, 64) + "\n"

	if c.resultsDir != "" {
		// Write the resulting CSV file
		aFile := c.resultsDir + "/" + time.Now().Format("2006-01-02-15-04-05") + "-aggregate-costaccounting.csv"
		err = ioutil.WriteFile(aFile, []byte(csv), 0644)
		if err != nil {
			err = fmt.Errorf("error writing file with path %s: %s", aFile, err.Error())
			exitcode = 1
			return
		}
		logger.Infof("Aggregate cost accounting for all supplied uuids in %s", aFile)
	}

	// Output the total dollar amount on stdout
	fmt.Fprintf(stdout, "%s\n", strconv.FormatFloat(total.cost, 'f', 2, 64))

	return
}
