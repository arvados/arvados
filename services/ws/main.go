// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"fmt"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/config"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
)

var logger = ctxlog.FromContext
var version = "dev"

func main() {
	log := logger(nil)

	configPath := flag.String("config", "/etc/arvados/ws/ws.yml", "`path` to config file")
	dumpConfig := flag.Bool("dump-config", false, "show current configuration and exit")
	getVersion := flag.Bool("version", false, "Print version information and exit.")
	cfg := defaultConfig()
	flag.Parse()

	// Print version information if requested
	if *getVersion {
		fmt.Printf("Version: %s\n", version)
		os.Exit(0)
	}

	err := config.LoadFile(&cfg, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	ctxlog.SetLevel(cfg.LogLevel)
	ctxlog.SetFormat(cfg.LogFormat)

	if *dumpConfig {
		txt, err := config.Dump(&cfg)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Print(string(txt))
		return
	}

	log.Printf("arvados-ws %q started", version)

	log.Info("started")
	srv := &server{wsConfig: &cfg}
	log.Fatal(srv.Run())
}
