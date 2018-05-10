// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/config"
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-systemd/daemon"
)

var (
	defaultConfigPath = "/etc/arvados/keep-web/keep-web.yml"
	version           = "dev"
)

// Config specifies server configuration.
type Config struct {
	Client arvados.Client

	Listen string

	AnonymousTokens    []string
	AttachmentOnlyHost string
	TrustAllContent    bool

	Cache cache

	// Hack to support old command line flag, which is a bool
	// meaning "get actual token from environment".
	deprecatedAllowAnonymous bool

	//Authorization token to be included in all health check requests.
	ManagementToken string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Listen: ":80",
		Cache: cache{
			TTL:                  arvados.Duration(5 * time.Minute),
			UUIDTTL:              arvados.Duration(5 * time.Second),
			MaxCollectionEntries: 1000,
			MaxCollectionBytes:   100000000,
			MaxPermissionEntries: 1000,
			MaxUUIDEntries:       1000,
		},
	}
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
	cfg := DefaultConfig()

	var configPath string
	deprecated := " (DEPRECATED -- use config file instead)"
	flag.StringVar(&configPath, "config", defaultConfigPath,
		"`path` to JSON or YAML configuration file")
	flag.StringVar(&cfg.Listen, "listen", "",
		"address:port or :port to listen on"+deprecated)
	flag.BoolVar(&cfg.deprecatedAllowAnonymous, "allow-anonymous", false,
		"Load an anonymous token from the ARVADOS_API_TOKEN environment variable"+deprecated)
	flag.StringVar(&cfg.AttachmentOnlyHost, "attachment-only-host", "",
		"Only serve attachments at the given `host:port`"+deprecated)
	flag.BoolVar(&cfg.TrustAllContent, "trust-all-content", false,
		"Serve non-public content from a single origin. Dangerous: read docs before using!"+deprecated)
	flag.StringVar(&cfg.ManagementToken, "management-token", "",
		"Authorization token to be included in all health check requests.")

	dumpConfig := flag.Bool("dump-config", false,
		"write current configuration to stdout and exit")
	getVersion := flag.Bool("version", false,
		"print version information and exit.")
	flag.Usage = usage
	flag.Parse()

	// Print version information if requested
	if *getVersion {
		fmt.Printf("keep-web %s\n", version)
		return
	}

	if err := config.LoadFile(cfg, configPath); err != nil {
		if h := os.Getenv("ARVADOS_API_HOST"); h != "" && configPath == defaultConfigPath {
			log.Printf("DEPRECATED: Using ARVADOS_API_HOST environment variable. Use config file instead.")
			cfg.Client.APIHost = h
		} else {
			log.Fatal(err)
		}
	}
	if cfg.deprecatedAllowAnonymous {
		log.Printf("DEPRECATED: Using -allow-anonymous command line flag with ARVADOS_API_TOKEN environment variable. Use config file instead.")
		cfg.AnonymousTokens = []string{os.Getenv("ARVADOS_API_TOKEN")}
	}

	if *dumpConfig {
		log.Fatal(config.DumpAndExit(cfg))
	}

	log.Printf("keep-web %s started", version)

	os.Setenv("ARVADOS_API_HOST", cfg.Client.APIHost)
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
