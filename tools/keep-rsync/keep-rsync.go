package main

import (
	"flag"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
)

// keep-rsync arguments
var (
	srcConfig           arvadosclient.APIConfig
	dstConfig           arvadosclient.APIConfig
	blobSigningKey      string
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
}

var matchTrue = regexp.MustCompile("^(?i:1|yes|true)$")

// Reads config from file
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
		kv := strings.Split(line, "=")

		switch kv[0] {
		case "ARVADOS_API_TOKEN":
			config.APIToken = kv[1]
		case "ARVADOS_API_HOST":
			config.APIHost = kv[1]
		case "ARVADOS_API_HOST_INSECURE":
			config.APIHostInsecure = matchTrue.MatchString(kv[1])
		case "ARVADOS_EXTERNAL_CLIENT":
			config.ExternalClient = matchTrue.MatchString(kv[1])
		case "ARVADOS_BLOB_SIGNING_KEY":
			blobSigningKey = kv[1]
		}
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
	arvSrc, err = arvadosclient.New(srcConfig)
	if err != nil {
		return
	}

	// arvDst from dstConfig
	arvDst, err = arvadosclient.New(dstConfig)
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
