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
	sdkConfig "git.curoverse.com/arvados.git/sdk/go/config"
	"github.com/coreos/go-systemd/daemon"
	log "github.com/sirupsen/logrus"
)

var (
	version = "dev"
)

// Config specifies server configuration.
type Config struct {
	Client  arvados.Client
	Cache   cache
	cluster *arvados.Cluster
}

// DefaultConfig returns the default configuration.
func DefaultConfig(arvCfg *arvados.Config) *Config {
	cfg := Config{}
	var cls *arvados.Cluster
	var err error
	if cls, err = arvCfg.GetCluster(""); err != nil {
		log.Fatal(err)
	}
	cfg.cluster = cls
	cfg.Cache.config = &cfg.cluster.Collections.WebDAVCache
	return &cfg
}

func init() {
	// MakeArvadosClient returns an error if this env var isn't
	// available as a default token (even if we explicitly set a
	// different token before doing anything with the client). We
	// set this dummy value during init so it doesn't clobber the
	// one used by "run test servers".
	if os.Getenv("ARVADOS_API_TOKEN") == "" {
		os.Setenv("ARVADOS_API_TOKEN", "xxx")
	}

	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
	})
}

func main() {
	prog := os.Args[0]
	args := os.Args[1:]
	logger := log.New()

	flags := flag.NewFlagSet(prog, flag.ExitOnError)

	loader := config.NewLoader(os.Stdin, logger)
	loader.SetupFlags(flags)

	dumpConfig := flags.Bool("dump-config", false,
		"write current configuration to stdout and exit")
	getVersion := flags.Bool("version", false,
		"print version information and exit.")

	args = loader.MungeLegacyConfigArgs(logger, args, "-legacy-keepweb-config")
	flags.Parse(args)

	// Print version information if requested
	if *getVersion {
		fmt.Printf("keep-web %s\n", version)
		return
	}

	arvCfg, err := loader.Load()
	if err != nil {
		log.Fatal(err)
	}
	cfg := DefaultConfig(arvCfg)

	if *dumpConfig {
		log.Fatal(sdkConfig.DumpAndExit(cfg.cluster))
	}

	log.Printf("keep-web %s started", version)

	os.Setenv("ARVADOS_API_HOST", cfg.cluster.Services.Controller.ExternalURL.Host)
	srv := &server{Config: cfg}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}
	log.Println("Listening at", srv.Addr)
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
