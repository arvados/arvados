// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"fmt"
	"os"

	"git.arvados.org/arvados.git/lib/config"
	"github.com/coreos/go-systemd/daemon"
	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
)

var version = "dev"

func main() {
	logger := log.New()
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
	})

	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	loader := config.NewLoader(os.Stdin, logger)
	loader.SetupFlags(flags)

	dumpConfig := flags.Bool("dump-config", false, "write current configuration to stdout and exit (useful for migrating from command line flags to config file)")
	getVersion := flags.Bool("version", false, "print version information and exit.")

	args := loader.MungeLegacyConfigArgs(logger, os.Args[1:], "-legacy-git-httpd-config")
	err := flags.Parse(args)
	if err == flag.ErrHelp {
		return
	} else if err != nil {
		logger.Error(err)
		os.Exit(2)
	} else if flags.NArg() != 0 {
		logger.Errorf("unrecognized command line arguments: %v", flags.Args())
		os.Exit(2)
	}

	if *getVersion {
		fmt.Printf("arv-git-httpd %s\n", version)
		return
	}

	cfg, err := loader.Load()
	if err != nil {
		log.Fatal(err)
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		log.Fatal(err)
	}

	if *dumpConfig {
		out, err := yaml.Marshal(cfg)
		if err != nil {
			log.Fatal(err)
		}
		_, err = os.Stdout.Write(out)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	srv := &server{cluster: cluster}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}
	log.Printf("arv-git-httpd %s started", version)
	log.Println("Listening at", srv.Addr)
	log.Println("Repository root", cluster.Git.Repositories)
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
