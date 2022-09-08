// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
)

var version = "dev"

func main() {
	os.Exit(doMain(os.Args[1:], os.Stderr))
}

func doMain(args []string, stderr io.Writer) int {
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

	blobSignatureTTLFlag := flags.Duration(
		"blob-signature-ttl",
		0,
		"Lifetime of blob permission signatures on the keepservers. If not provided, this will be retrieved from the API server's discovery document.")

	verbose := flags.Bool(
		"v",
		false,
		"Log progress of each block verification")

	getVersion := flags.Bool(
		"version",
		false,
		"Print version information and exit.")

	if ok, code := cmd.ParseFlags(flags, os.Args[0], args, "", stderr); !ok {
		return code
	} else if *getVersion {
		fmt.Printf("%s %s\n", os.Args[0], version)
		return 0
	}

	config, blobSigningKey, err := loadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(stderr, "Error loading configuration from file: %s\n", err)
		return 1
	}

	// get list of block locators to be checked
	blockLocators, err := getBlockLocators(*locatorFile, *prefix)
	if err != nil {
		fmt.Fprintf(stderr, "Error reading block hashes to be checked from file: %s\n", err)
		return 1
	}

	// setup keepclient
	kc, blobSignatureTTL, err := setupKeepClient(config, *keepServicesJSON, *blobSignatureTTLFlag)
	if err != nil {
		fmt.Fprintf(stderr, "Error configuring keepclient: %s\n", err)
		return 1
	}

	err = performKeepBlockCheck(kc, blobSignatureTTL, blobSigningKey, blockLocators, *verbose)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

type apiConfig struct {
	APIToken        string
	APIHost         string
	APIHostInsecure bool
}

// Load config from given file
func loadConfig(configFile string) (config apiConfig, blobSigningKey string, err error) {
	if configFile == "" {
		err = errors.New("Client config file not specified")
		return
	}

	config, blobSigningKey, err = readConfigFromFile(configFile)
	return
}

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
		if len(kv) == 2 {
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])

			switch key {
			case "ARVADOS_API_TOKEN":
				config.APIToken = value
			case "ARVADOS_API_HOST":
				config.APIHost = value
			case "ARVADOS_API_HOST_INSECURE":
				config.APIHostInsecure = arvadosclient.StringBool(value)
			case "ARVADOS_BLOB_SIGNING_KEY":
				blobSigningKey = value
			}
		}
	}

	return
}

// setup keepclient using the config provided
func setupKeepClient(config apiConfig, keepServicesJSON string, blobSignatureTTL time.Duration) (kc *keepclient.KeepClient, ttl time.Duration, err error) {
	arv := arvadosclient.ArvadosClient{
		ApiToken:    config.APIToken,
		ApiServer:   config.APIHost,
		ApiInsecure: config.APIHostInsecure,
		Client: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: config.APIHostInsecure}}},
	}

	// If keepServicesJSON is provided, use it instead of service discovery
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

	// Get if blobSignatureTTL is not provided
	ttl = blobSignatureTTL
	if blobSignatureTTL == 0 {
		value, err := arv.Discovery("blobSignatureTtl")
		if err == nil {
			ttl = time.Duration(int(value.(float64))) * time.Second
		} else {
			return nil, 0, err
		}
	}

	return
}

// Get list of unique block locators from the given file
func getBlockLocators(locatorFile, prefix string) (locators []string, err error) {
	if locatorFile == "" {
		err = errors.New("block-hash-file not specified")
		return
	}

	content, err := ioutil.ReadFile(locatorFile)
	if err != nil {
		return
	}

	locatorMap := make(map[string]bool)
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, prefix) || locatorMap[line] {
			continue
		}
		locators = append(locators, line)
		locatorMap[line] = true
	}

	return
}

// Get block headers from keep. Log any errors.
func performKeepBlockCheck(kc *keepclient.KeepClient, blobSignatureTTL time.Duration, blobSigningKey string, blockLocators []string, verbose bool) error {
	totalBlocks := len(blockLocators)
	notFoundBlocks := 0
	current := 0
	for _, locator := range blockLocators {
		current++
		if verbose {
			log.Printf("Verifying block %d of %d: %v", current, totalBlocks, locator)
		}
		getLocator := locator
		if blobSigningKey != "" {
			expiresAt := time.Now().AddDate(0, 0, 1)
			getLocator = keepclient.SignLocator(locator, kc.Arvados.ApiToken, expiresAt, blobSignatureTTL, []byte(blobSigningKey))
		}

		_, _, err := kc.Ask(getLocator)
		if err != nil {
			notFoundBlocks++
			log.Printf("Error verifying block %v: %v", locator, err)
		}
	}

	log.Printf("Verify block totals: %d attempts, %d successes, %d errors", totalBlocks, totalBlocks-notFoundBlocks, notFoundBlocks)

	if notFoundBlocks > 0 {
		return fmt.Errorf("Block verification failed for %d out of %d blocks with matching prefix", notFoundBlocks, totalBlocks)
	}

	return nil
}
