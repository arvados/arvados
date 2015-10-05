package main

import (
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"log"
	"strings"
)

// keep-rsync arguments
var (
	srcConfig            map[string]string
	destConfig           map[string]string
	srcKeepServicesJSON  string
	destKeepServicesJSON string
	replications         int
	prefix               string
)

func main() {
	var srcConfigFile string
	var destConfigFile string

	flag.StringVar(
		&srcConfigFile,
		"src-config-file",
		"",
		"Source configuration filename with full path that contains "+
			"an ARVADOS_API_TOKEN which is a valid datamanager token recognized by the source keep servers, "+
			"ARVADOS_API_HOST, ARVADOS_API_HOST_INSECURE, and ARVADOS_BLOB_SIGNING_KEY.")

	flag.StringVar(
		&destConfigFile,
		"dest-config-file",
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
		&destKeepServicesJSON,
		"dest-keep-services-json",
		"",
		"An optional list of available destination keepservices. "+
			"If not provided, this list is obtained from api server configured in dest-config-file.")

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

	if srcConfigFile == "" {
		log.Fatal("-src-config-file must be specified.")
	}
	srcConfig, err = readConfigFromFile(srcConfigFile)
	if err != nil {
		log.Fatal("Error reading source configuration: %s", err.Error())
	}

	if destConfigFile == "" {
		log.Fatal("-dest-config-file must be specified.")
	}
	destConfig, err = readConfigFromFile(destConfigFile)
	if err != nil {
		log.Fatal("Error reading destination configuration: %s", err.Error())
	}

	err = initializeKeepRsync()
	if err != nil {
		log.Fatal("Error configurating keep-rsync: %s", err.Error())
	}
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
	arvSrc  arvadosclient.ArvadosClient
	arvDest arvadosclient.ArvadosClient
	kcSrc   *keepclient.KeepClient
	kcDest  *keepclient.KeepClient
)

// Initializes keep-rsync using the config provided
func initializeKeepRsync() (err error) {
	arvSrc, err = arvadosclient.MakeArvadosClientWithConfig(srcConfig)
	if err != nil {
		return
	}

	arvDest, err = arvadosclient.MakeArvadosClientWithConfig(destConfig)
	if err != nil {
		return
	}

	kcSrc, err = keepclient.MakeKeepClient(&arvSrc)
	if err != nil {
		return
	}

	kcDest, err = keepclient.MakeKeepClient(&arvDest)

	return
}
