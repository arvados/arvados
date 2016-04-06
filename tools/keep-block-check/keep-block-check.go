package main

import (
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
	flags := flag.NewFlagSet("keep-block-check", flag.ExitOnError)

	configFile := flags.String(
		"config",
		"",
		"Configuration filename. May be either a pathname to a config file, or (for example) 'foo' as shorthand for $HOME/.config/arvados/foo.conf file. This file is expected to specify the values for ARVADOS_API_TOKEN, ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE, and ARVADOS_BLOB_SIGNING_KEY for the source.")

	keepServicesJSON := flags.String(
		"keep-services-json",
		"",
		"An optional list of available keepservices. "+
			"If not provided, this list is obtained from api server configured in config-file.")

	locatorFile := flags.String(
		"block-hash-file",
		"",
		"Filename containing the block hashes to be checked. This is required. "+
			"This file contains the block hashes one per line.")

	prefix := flags.String(
		"prefix",
		"",
		"Block hash prefix. When a prefix is specified, only hashes listed in the file with this prefix will be checked.")

	// Parse args; omit the first arg which is the command name
	flags.Parse(os.Args[1:])

	config, blobSigningKey, err := loadConfig(*configFile)
	if err != nil {
		return fmt.Errorf("Error loading configuration from file: %s", err.Error())
	}

	// get list of block locators to be checked
	blockLocators, err := getBlockLocators(*locatorFile)
	if err != nil {
		return fmt.Errorf("Error reading block hashes to be checked from file: %s", err.Error())
	}

	// setup keepclient
	kc, err := setupKeepClient(config, *keepServicesJSON)
	if err != nil {
		return fmt.Errorf("Error configuring keepclient: %s", err.Error())
	}

	performKeepBlockCheck(kc, blobSigningKey, *prefix, blockLocators)
	return nil
}

type apiConfig struct {
	APIToken        string
	APIHost         string
	APIHostInsecure bool
	ExternalClient  bool
}

// Load config from given file
func loadConfig(configFile string) (config apiConfig, blobSigningKey string, err error) {
	if configFile == "" {
		err = errors.New("API config file not specified")
		return
	}

	config, blobSigningKey, err = readConfigFromFile(configFile)
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
		return
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
func setupKeepClient(config apiConfig, keepServicesJSON string) (kc *keepclient.KeepClient, err error) {
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
			return
		}
	} else {
		kc = keepclient.New(&arv)
		err = kc.LoadKeepServicesFromJSON(keepServicesJSON)
		if err != nil {
			return
		}
	}

	return
}

// Get list of block locators from the given file
func getBlockLocators(locatorFile string) (locators []string, err error) {
	if locatorFile == "" {
		err = errors.New("block-hash-file not specified")
		return
	}

	content, err := ioutil.ReadFile(locatorFile)

	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		locators = append(locators, strings.TrimSpace(line))
	}

	return
}

// Get block headers from keep. Log any errors.
func performKeepBlockCheck(kc *keepclient.KeepClient, blobSigningKey, prefix string, blockLocators []string) {
	for _, locator := range blockLocators {
		if !strings.HasPrefix(locator, prefix) {
			continue
		}
		getLocator := locator
		if blobSigningKey != "" {
			expiresAt := time.Now().AddDate(0, 0, 1)
			getLocator = keepclient.SignLocator(locator, kc.Arvados.ApiToken, expiresAt, []byte(blobSigningKey))
		}

		_, _, err := kc.Ask(getLocator)
		if err != nil {
			log.Printf("Error getting head info for block: %v %v", locator, err)
		}
	}
}
