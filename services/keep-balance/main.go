// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"github.com/sirupsen/logrus"
)

var debugf = func(string, ...interface{}) {}

func main() {
	var cfg Config
	var runOptions RunOptions

	configPath := flag.String("config", defaultConfigPath,
		"`path` of JSON or YAML configuration file")
	serviceListPath := flag.String("config.KeepServiceList", "",
		"`path` of JSON or YAML file with list of keep services to balance, as given by \"arv keep_service list\" "+
			"(default: config[\"KeepServiceList\"], or if none given, get all available services and filter by config[\"KeepServiceTypes\"])")
	flag.BoolVar(&runOptions.Once, "once", false,
		"balance once and then exit")
	flag.BoolVar(&runOptions.CommitPulls, "commit-pulls", false,
		"send pull requests (make more replicas of blocks that are underreplicated or are not in optimal rendezvous probe order)")
	flag.BoolVar(&runOptions.CommitTrash, "commit-trash", false,
		"send trash requests (delete unreferenced old blocks, and excess replicas of overreplicated blocks)")
	dumpConfig := flag.Bool("dump-config", false, "write current configuration to stdout and exit")
	dumpFlag := flag.Bool("dump", false, "dump details for each block to stdout")
	debugFlag := flag.Bool("debug", false, "enable debug messages")
	getVersion := flag.Bool("version", false, "Print version information and exit.")
	flag.Usage = usage
	flag.Parse()

	// Print version information if requested
	if *getVersion {
		fmt.Printf("keep-balance %s\n", version)
		return
	}

	mustReadConfig(&cfg, *configPath)
	if *serviceListPath != "" {
		mustReadConfig(&cfg.KeepServiceList, *serviceListPath)
	}

	if *dumpConfig {
		log.Fatal(config.DumpAndExit(cfg))
	}

	to := time.Duration(cfg.RequestTimeout)
	if to == 0 {
		to = 30 * time.Minute
	}
	arvados.DefaultSecureClient.Timeout = to
	arvados.InsecureHTTPClient.Timeout = to
	http.DefaultClient.Timeout = to

	log.Printf("keep-balance %s started", version)

	if *debugFlag {
		debugf = log.Printf
		if j, err := json.Marshal(cfg); err != nil {
			log.Fatal(err)
		} else {
			log.Printf("config is %s", j)
		}
	}
	if *dumpFlag {
		dumper := logrus.New()
		dumper.Out = os.Stdout
		dumper.Formatter = &logrus.TextFormatter{}
		runOptions.Dumper = dumper
	}
	srv, err := NewServer(cfg, runOptions)
	if err != nil {
		// (don't run)
	} else if runOptions.Once {
		_, err = srv.Run()
	} else {
		err = srv.RunForever(nil)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func mustReadConfig(dst interface{}, path string) {
	if err := config.LoadFile(dst, path); err != nil {
		log.Fatal(err)
	}
}
