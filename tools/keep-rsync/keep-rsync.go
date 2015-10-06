package main

import (
	"bytes"
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"log"
	"strings"
)

// keep-rsync arguments
var (
	srcConfig           map[string]string
	dstConfig           map[string]string
	srcKeepServicesJSON string
	dstKeepServicesJSON string
	replications        int
	prefix              string
)

func main() {
	var srcConfigFile string
	var dstConfigFile string

	flag.StringVar(
		&srcConfigFile,
		"src-config-file",
		"",
		"Source configuration filename with full path that contains "+
			"an ARVADOS_API_TOKEN which is a valid datamanager token recognized by the source keep servers, "+
			"ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE, and ARVADOS_BLOB_SIGNING_KEY.")

	flag.StringVar(
		&dstConfigFile,
		"dst-config-file",
		"",
		"Destination configuration filename with full path that contains "+
			"an ARVADOS_API_TOKEN which is a valid datamanager token recognized by the destination keep servers, "+
			"ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE, and ARVADOS_BLOB_SIGNING_KEY.")

	flag.StringVar(
		&srcKeepServicesJSON,
		"src-keep-services-json",
		"",
		"An optional list of available source keepservices. "+
			"If not provided, this list is obtained from api server configured in src-config-file.")

	flag.StringVar(
		&dstKeepServicesJSON,
		"dst-keep-services-json",
		"",
		"An optional list of available destination keepservices. "+
			"If not provided, this list is obtained from api server configured in dst-config-file.")

	flag.IntVar(
		&replications,
		"replications",
		3,
		"Number of replications to write to the destination.")

	flag.StringVar(
		&prefix,
		"prefix",
		"",
		"Index prefix")

	flag.Parse()

	var err error

	// Load config
	if srcConfigFile == "" {
		log.Fatal("-src-config-file must be specified.")
	}
	srcConfig, err = readConfigFromFile(srcConfigFile)
	if err != nil {
		log.Fatal("Error reading source configuration: %s", err.Error())
	}

	if dstConfigFile == "" {
		log.Fatal("-dst-config-file must be specified.")
	}
	dstConfig, err = readConfigFromFile(dstConfigFile)
	if err != nil {
		log.Fatal("Error reading destination configuration: %s", err.Error())
	}

	// Initialize keep-rsync
	err = initializeKeepRsync()
	if err != nil {
		log.Fatal("Error configurating keep-rsync: %s", err.Error())
	}

	// Copy blocks not found in dst from src
	performKeepRsync()
}

// Reads config from file
func readConfigFromFile(filename string) (map[string]string, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	config := make(map[string]string)
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		kv := strings.Split(line, "=")
		config[kv[0]] = kv[1]
	}
	return config, nil
}

// keep-rsync source and destination clients
var (
	arvSrc arvadosclient.ArvadosClient
	arvDst arvadosclient.ArvadosClient
	kcSrc  *keepclient.KeepClient
	kcDst  *keepclient.KeepClient
)

// Initializes keep-rsync using the config provided
func initializeKeepRsync() (err error) {
	// arvSrc from srcConfig
	arvSrc, err = arvadosclient.MakeArvadosClientWithConfig(srcConfig)
	if err != nil {
		return
	}

	// arvDst from dstConfig
	arvDst, err = arvadosclient.MakeArvadosClientWithConfig(dstConfig)
	if err != nil {
		return
	}

	// if srcKeepServicesJSON is provided, use it to load services; else, use DiscoverKeepServers
	if srcKeepServicesJSON == "" {
		kcSrc, err = keepclient.MakeKeepClient(&arvSrc)
		if err != nil {
			return
		}
	} else {
		kcSrc, err = keepclient.MakeKeepClientFromJSON(&arvSrc, srcKeepServicesJSON)
		if err != nil {
			return
		}
	}

	// if dstKeepServicesJSON is provided, use it to load services; else, use DiscoverKeepServers
	if dstKeepServicesJSON == "" {
		kcDst, err = keepclient.MakeKeepClient(&arvDst)
		if err != nil {
			return
		}
	} else {
		kcDst, err = keepclient.MakeKeepClientFromJSON(&arvDst, dstKeepServicesJSON)
		if err != nil {
			return
		}
	}
	kcDst.Want_replicas = replications

	return
}

// Get unique block locators from src and dst
// Copy any blocks missing in dst
func performKeepRsync() error {
	// Get unique locators from src
	srcIndex, err := getUniqueLocators(kcSrc, prefix)
	if err != nil {
		return err
	}

	// Get unique locators from dst
	dstIndex, err := getUniqueLocators(kcDst, prefix)
	if err != nil {
		return err
	}

	// Get list of locators found in src, but missing in dst
	toBeCopied := getMissingLocators(srcIndex, dstIndex)

	// Copy each missing block to dst
	copyBlocksToDst(toBeCopied)

	return nil
}

// Get list of unique locators from the specified cluster
func getUniqueLocators(kc *keepclient.KeepClient, prefix string) (map[string]bool, error) {
	var indexBytes []byte

	for uuid := range kc.LocalRoots() {
		reader, err := kc.GetIndex(uuid, prefix)
		if err != nil {
			return nil, err
		}

		var readBytes []byte
		readBytes, err = ioutil.ReadAll(reader)
		if err != nil {
			return nil, err
		}

		indexBytes = append(indexBytes, readBytes...)
	}

	// Got index; Now dedup it
	locators := bytes.Split(indexBytes, []byte("\n"))

	uniqueLocators := map[string]bool{}
	for _, loc := range locators {
		if len(loc) == 0 {
			continue
		}

		locator := string(bytes.Split(loc, []byte(" "))[0])
		if _, ok := uniqueLocators[locator]; !ok {
			uniqueLocators[locator] = true
		}
	}
	return uniqueLocators, nil
}

// Get list of locators that are in src but not in dst
func getMissingLocators(srcLocators map[string]bool, dstLocators map[string]bool) []string {
	var missingLocators []string
	for locator := range srcLocators {
		if _, ok := dstLocators[locator]; !ok {
			missingLocators = append(missingLocators, locator)
		}
	}
	return missingLocators
}

// Copy blocks from src to dst; only those that are missing in dst are copied
func copyBlocksToDst(toBeCopied []string) {
	done := 0
	total := len(toBeCopied)
	var failed []string

	for _, locator := range toBeCopied {
		log.Printf("Getting block %d of %d", done+1, total)

		log.Printf("Getting block: %v", locator)

		reader, _, _, err := kcSrc.Get(locator)
		if err != nil {
			log.Printf("Error getting block: %q %v", locator, err)
			failed = append(failed, locator)
			continue
		}
		data, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Printf("Error reading block data: %q %v", locator, err)
			failed = append(failed, locator)
			continue
		}

		log.Printf("Copying block: %q", locator)
		_, rep, err := kcDst.PutB(data)
		if err != nil {
			log.Printf("Error putting block data: %q %v", locator, err)
			failed = append(failed, locator)
			continue
		}
		if rep != replications {
			log.Printf("Failed to put enough number of replicas. Wanted: %d; Put: %d", replications, rep)
			failed = append(failed, locator)
			continue
		}

		done++
		log.Printf("%.2f%% done", float64(done)/float64(total)*100)
	}

	log.Printf("Successfully copied to destination %d and failed %d out of a total of %d", done, len(failed), total)
	log.Printf("Failed blocks %v", failed)
}
