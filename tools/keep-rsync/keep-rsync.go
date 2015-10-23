package main

import (
	"bufio"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {
	err := doMain()
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func doMain() error {
	flags := flag.NewFlagSet("keep-rsync", flag.ExitOnError)

	srcConfigFile := flags.String(
		"src",
		"",
		"Source configuration filename. May be either a pathname to a config file, or (for example) 'foo' as shorthand for $HOME/.config/arvados/foo.conf file. This file is expected to specify the values for ARVADOS_API_TOKEN, ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE, and ARVADOS_BLOB_SIGNING_KEY for the source.")

	dstConfigFile := flags.String(
		"dst",
		"",
		"Destination configuration filename. May be either a pathname to a config file, or (for example) 'foo' as shorthand for $HOME/.config/arvados/foo.conf file. This file is expected to specify the values for ARVADOS_API_TOKEN, ARVADOS_API_HOST, and ARVADOS_API_HOST_INSECURE for the destination.")

	srcKeepServicesJSON := flags.String(
		"src-keep-services-json",
		"",
		"An optional list of available source keepservices. "+
			"If not provided, this list is obtained from api server configured in src-config-file.")

	dstKeepServicesJSON := flags.String(
		"dst-keep-services-json",
		"",
		"An optional list of available destination keepservices. "+
			"If not provided, this list is obtained from api server configured in dst-config-file.")

	replications := flags.Int(
		"replications",
		0,
		"Number of replications to write to the destination. If replications not specified, "+
			"default replication level configured on destination server will be used.")

	prefix := flags.String(
		"prefix",
		"",
		"Index prefix")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	srcConfig, srcBlobSigningKey, err := loadConfig(*srcConfigFile)
	if err != nil {
		return fmt.Errorf("Error loading src configuration from file: %s", err.Error())
	}

	dstConfig, _, err := loadConfig(*dstConfigFile)
	if err != nil {
		return fmt.Errorf("Error loading dst configuration from file: %s", err.Error())
	}

	// setup src and dst keepclients
	kcSrc, err := setupKeepClient(srcConfig, *srcKeepServicesJSON, false, 0)
	if err != nil {
		return fmt.Errorf("Error configuring src keepclient: %s", err.Error())
	}

	kcDst, err := setupKeepClient(dstConfig, *dstKeepServicesJSON, true, *replications)
	if err != nil {
		return fmt.Errorf("Error configuring dst keepclient: %s", err.Error())
	}

	// Copy blocks not found in dst from src
	err = performKeepRsync(kcSrc, kcDst, srcBlobSigningKey, *prefix)
	if err != nil {
		return fmt.Errorf("Error while syncing data: %s", err.Error())
	}

	return nil
}

type apiConfig struct {
	APIToken        string
	APIHost         string
	APIHostInsecure bool
	ExternalClient  bool
}

// Load src and dst config from given files
func loadConfig(configFile string) (config apiConfig, blobSigningKey string, err error) {
	if configFile == "" {
		return config, blobSigningKey, errors.New("config file not specified")
	}

	config, blobSigningKey, err = readConfigFromFile(configFile)
	if err != nil {
		return config, blobSigningKey, fmt.Errorf("Error reading config file: %v", err)
	}

	return
}

var matchTrue = regexp.MustCompile("^(?i:1|yes|true)$")

// Read config from file
func readConfigFromFile(filename string) (config apiConfig, blobSigningKey string, err error) {
	if !strings.Contains(filename, "/") {
		filename = os.Getenv("HOME") + "/.config/arvados/" + filename + ".conf"
	}

	content, err := ioutil.ReadFile(filename)

	if err != nil {
		return config, "", err
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
	return
}

// setup keepclient using the config provided
func setupKeepClient(config apiConfig, keepServicesJSON string, isDst bool, replications int) (kc *keepclient.KeepClient, err error) {
	arv := arvadosclient.ArvadosClient{
		ApiToken:    config.APIToken,
		ApiServer:   config.APIHost,
		ApiInsecure: config.APIHostInsecure,
		Client: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: config.APIHostInsecure}}},
		External: config.ExternalClient,
	}

	// if keepServicesJSON is provided, use it to load services; else, use DiscoverKeepServers
	if keepServicesJSON == "" {
		kc, err = keepclient.MakeKeepClient(&arv)
		if err != nil {
			return nil, err
		}
	} else {
		kc = keepclient.New(&arv)
		err = kc.LoadKeepServicesFromJSON(keepServicesJSON)
		if err != nil {
			return kc, err
		}
	}

	if isDst {
		// Get default replications value from destination, if it is not already provided
		if replications == 0 {
			value, err := arv.Discovery("defaultCollectionReplication")
			if err == nil {
				replications = int(value.(float64))
			} else {
				return nil, err
			}
		}

		kc.Want_replicas = replications
	}

	return kc, nil
}

// Get unique block locators from src and dst
// Copy any blocks missing in dst
func performKeepRsync(kcSrc, kcDst *keepclient.KeepClient, blobSigningKey, prefix string) error {
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
	log.Printf("Before keep-rsync, there are %d blocks in src and %d blocks in dst. Start copying %d blocks from src not found in dst.",
		len(srcIndex), len(dstIndex), len(toBeCopied))

	err = copyBlocksToDst(toBeCopied, kcSrc, kcDst, blobSigningKey)

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
func copyBlocksToDst(toBeCopied []string, kcSrc, kcDst *keepclient.KeepClient, blobSigningKey string) error {
	total := len(toBeCopied)

	startedAt := time.Now()
	for done, locator := range toBeCopied {
		if done == 0 {
			log.Printf("Copying data block %d of %d (%.2f%% done): %v", done+1, total,
				float64(done)/float64(total)*100, locator)
		} else {
			timePerBlock := time.Since(startedAt) / time.Duration(done)
			log.Printf("Copying data block %d of %d (%.2f%% done, %v est. time remaining): %v", done+1, total,
				float64(done)/float64(total)*100, timePerBlock*time.Duration(total-done), locator)
		}

		getLocator := locator
		expiresAt := time.Now().AddDate(0, 0, 1)
		if blobSigningKey != "" {
			getLocator = keepclient.SignLocator(getLocator, kcSrc.Arvados.ApiToken, expiresAt, []byte(blobSigningKey))
		}

		reader, len, _, err := kcSrc.Get(getLocator)
		if err != nil {
			return fmt.Errorf("Error getting block: %v %v", locator, err)
		}

		_, _, err = kcDst.PutHR(getLocator[:32], reader, len)
		if err != nil {
			return fmt.Errorf("Error copying data block: %v %v", locator, err)
		}
	}

	log.Printf("Successfully copied to destination %d blocks.", total)
	return nil
}
