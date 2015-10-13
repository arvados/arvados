package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"
)

// keep-rsync arguments
var (
	blobSigningKey string
)

func main() {
	var srcConfigFile, dstConfigFile, srcKeepServicesJSON, dstKeepServicesJSON, prefix string
	var replications int

	flag.StringVar(
		&srcConfigFile,
		"src-config-file",
		"",
		"Source configuration filename with full path that contains "+
			"an ARVADOS_API_TOKEN which is a valid datamanager token recognized by the source keep servers, "+
			"ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE, ARVADOS_EXTERNAL_CLIENT and ARVADOS_BLOB_SIGNING_KEY.")

	flag.StringVar(
		&dstConfigFile,
		"dst-config-file",
		"",
		"Destination configuration filename with full path that contains "+
			"an ARVADOS_API_TOKEN which is a valid datamanager token recognized by the destination keep servers, "+
			"ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE, ARVADOS_EXTERNAL_CLIENT and ARVADOS_BLOB_SIGNING_KEY.")

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
		0,
		"Number of replications to write to the destination.")

	flag.StringVar(
		&prefix,
		"prefix",
		"",
		"Index prefix")

	flag.Parse()

	srcConfig, dstConfig, err := loadConfig(srcConfigFile, dstConfigFile)
	if err != nil {
		log.Fatalf("Error loading configuration from files: %s", err.Error())
	}

	// setup src and dst keepclients
	kcSrc, kcDst, err := setupKeepClients(srcConfig, dstConfig, srcKeepServicesJSON, dstKeepServicesJSON, replications)
	if err != nil {
		log.Fatalf("Error configuring keep-rsync: %s", err.Error())
	}

	// Copy blocks not found in dst from src
	err = performKeepRsync(kcSrc, kcDst, prefix)
	if err != nil {
		log.Fatalf("Error while syncing data: %s", err.Error())
	}
}

// Load src and dst config from given files
func loadConfig(srcConfigFile, dstConfigFile string) (srcConfig, dstConfig arvadosclient.APIConfig, err error) {
	if srcConfigFile == "" {
		return srcConfig, dstConfig, errors.New("-src-config-file must be specified")
	}

	srcConfig, err = readConfigFromFile(srcConfigFile)
	if err != nil {
		return srcConfig, dstConfig, fmt.Errorf("Error reading source configuration: %v", err)
	}

	if dstConfigFile == "" {
		return srcConfig, dstConfig, errors.New("-dst-config-file must be specified")
	}
	dstConfig, err = readConfigFromFile(dstConfigFile)
	if err != nil {
		return srcConfig, dstConfig, fmt.Errorf("Error reading destination configuration: %v", err)
	}

	return srcConfig, dstConfig, err
}

var matchTrue = regexp.MustCompile("^(?i:1|yes|true)$")

// Read config from file
func readConfigFromFile(filename string) (arvadosclient.APIConfig, error) {
	var config arvadosclient.APIConfig

	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		kv := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "ARVADOS_API_TOKEN":
			config.APIToken = value
		case "ARVADOS_API_HOST":
			config.APIHost = value
		case "ARVADOS_API_HOST_INSECURE":
			config.APIHostInsecure = matchTrue.MatchString(value)
		case "ARVADOS_EXTERNAL_CLIENT":
			config.ExternalClient = matchTrue.MatchString(value)
		case "ARVADOS_BLOB_SIGNING_KEY":
			blobSigningKey = value
		}
	}
	return config, nil
}

// Initializes keep-rsync using the config provided
func setupKeepClients(srcConfig, dstConfig arvadosclient.APIConfig, srcKeepServicesJSON, dstKeepServicesJSON string, replications int) (kcSrc, kcDst *keepclient.KeepClient, err error) {
	// arvSrc from srcConfig
	arvSrc, err := arvadosclient.New(srcConfig)
	if err != nil {
		return kcSrc, kcDst, err
	}

	// arvDst from dstConfig
	arvDst, err := arvadosclient.New(dstConfig)
	if err != nil {
		return kcSrc, kcDst, err
	}

	// Get default replications value from destination, if it is not already provided
	if replications == 0 {
		value, err := arvDst.Discovery("defaultCollectionReplication")
		if err == nil {
			replications = int(value.(float64))
		} else {
			replications = 2
		}
	}

	// if srcKeepServicesJSON is provided, use it to load services; else, use DiscoverKeepServers
	if srcKeepServicesJSON == "" {
		kcSrc, err = keepclient.MakeKeepClient(&arvSrc)
		if err != nil {
			return nil, nil, err
		}
	} else {
		kcSrc, err = keepclient.MakeKeepClientFromJSON(&arvSrc, srcKeepServicesJSON)
		if err != nil {
			return kcSrc, kcDst, err
		}
	}

	// if dstKeepServicesJSON is provided, use it to load services; else, use DiscoverKeepServers
	if dstKeepServicesJSON == "" {
		kcDst, err = keepclient.MakeKeepClient(&arvDst)
		if err != nil {
			return kcSrc, kcDst, err
		}
	} else {
		kcDst, err = keepclient.MakeKeepClientFromJSON(&arvDst, dstKeepServicesJSON)
		if err != nil {
			return kcSrc, kcDst, err
		}
	}
	kcDst.Want_replicas = replications

	return kcSrc, kcDst, nil
}

// Get unique block locators from src and dst
// Copy any blocks missing in dst
func performKeepRsync(kcSrc, kcDst *keepclient.KeepClient, prefix string) error {
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
	err = copyBlocksToDst(toBeCopied, kcSrc, kcDst)

	return err
}

// Get list of unique locators from the specified cluster
func getUniqueLocators(kc *keepclient.KeepClient, prefix string) (map[string]bool, error) {
	uniqueLocators := map[string]bool{}

	// Get index and dedup
	for uuid := range kc.LocalRoots() {
		reader, err := kc.GetIndex(uuid, prefix)
		if err != nil {
			return uniqueLocators, err
		}
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			uniqueLocators[strings.Split(scanner.Text(), " ")[0]] = true
		}
	}

	return uniqueLocators, nil
}

// Get list of locators that are in src but not in dst
func getMissingLocators(srcLocators, dstLocators map[string]bool) []string {
	var missingLocators []string
	for locator := range srcLocators {
		if _, ok := dstLocators[locator]; !ok {
			missingLocators = append(missingLocators, locator)
		}
	}
	return missingLocators
}

// Copy blocks from src to dst; only those that are missing in dst are copied
func copyBlocksToDst(toBeCopied []string, kcSrc, kcDst *keepclient.KeepClient) error {
	done := 0
	total := len(toBeCopied)

	for _, locator := range toBeCopied {
		log.Printf("Getting block %d of %d: %v", done+1, total, locator)

		getLocator := locator
		expiresAt := time.Now().AddDate(0, 0, 1)
		if blobSigningKey != "" {
			getLocator = keepclient.SignLocator(getLocator, kcSrc.Arvados.ApiToken, expiresAt, []byte(blobSigningKey))
		}

		reader, _, _, err := kcSrc.Get(getLocator)
		if err != nil {
			return fmt.Errorf("Error getting block: %v %v", locator, err)
		}
		data, err := ioutil.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("Error reading block data: %v %v", locator, err)
		}

		log.Printf("Writing block%d of %d: %v", locator)
		_, _, err = kcDst.PutB(data)
		if err != nil {
			return fmt.Errorf("Error putting block data: %v %v", locator, err)
		}

		done++
		log.Printf("%.2f%% done", float64(done)/float64(total)*100)
	}

	log.Printf("Successfully copied to destination %d blocks.", total)
	return nil
}
