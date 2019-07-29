// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"fmt"
	"os"

	"git.curoverse.com/arvados.git/lib/config"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

var logger = ctxlog.FromContext
var version = "dev"

func configure(log logrus.FieldLogger, args []string) *arvados.Cluster {
	flags := flag.NewFlagSet(args[0], flag.ExitOnError)
	dumpConfig := flags.Bool("dump-config", false, "show current configuration and exit")
	getVersion := flags.Bool("version", false, "Print version information and exit.")

	loader := config.NewLoader(nil, log)
	loader.SetupFlags(flags)
	args = loader.MungeLegacyConfigArgs(log, args[1:], "-legacy-ws-config")

	flags.Parse(args)

	// Print version information if requested
	if *getVersion {
		fmt.Printf("arvados-ws %s\n", version)
		return nil
	}

	cfg, err := loader.Load()
	if err != nil {
		log.Fatal(err)
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		log.Fatal(err)
	}

	ctxlog.SetLevel(cluster.SystemLogs.LogLevel)
	ctxlog.SetFormat(cluster.SystemLogs.Format)

	if *dumpConfig {
		out, err := yaml.Marshal(cfg)
		if err != nil {
			log.Fatal(err)
		}
		_, err = os.Stdout.Write(out)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	}
	return cluster
}

func main() {
	log := logger(nil)

	cluster := configure(log, os.Args)
	if cluster == nil {
		return
	}

	log.Printf("arvados-ws %s started", version)
	srv := &server{cluster: cluster}
	log.Fatal(srv.Run())
}
