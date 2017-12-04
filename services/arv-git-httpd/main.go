// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/config"
	"github.com/coreos/go-systemd/daemon"
)

var version = "dev"

// Server configuration
type Config struct {
	Client          arvados.Client
	Listen          string
	GitCommand      string
	RepoRoot        string
	GitoliteHome    string
	ManagementToken string
}

var theConfig = defaultConfig()

func defaultConfig() *Config {
	return &Config{
		Listen:     ":80",
		GitCommand: "/usr/bin/git",
		RepoRoot:   "/var/lib/arvados/git/repositories",
	}
}

func main() {
	const defaultCfgPath = "/etc/arvados/git-httpd/git-httpd.yml"
	const deprecated = " (DEPRECATED -- use config file instead)"
	flag.StringVar(&theConfig.Listen, "address", theConfig.Listen,
		"Address to listen on, \"host:port\" or \":port\"."+deprecated)
	flag.StringVar(&theConfig.GitCommand, "git-command", theConfig.GitCommand,
		"Path to git or gitolite-shell executable. Each authenticated request will execute this program with a single argument, \"http-backend\"."+deprecated)
	flag.StringVar(&theConfig.RepoRoot, "repo-root", theConfig.RepoRoot,
		"Path to git repositories."+deprecated)
	flag.StringVar(&theConfig.GitoliteHome, "gitolite-home", theConfig.GitoliteHome,
		"Value for GITOLITE_HTTP_HOME environment variable. If not empty, GL_BYPASS_ACCESS_CHECKS=1 will also be set."+deprecated)

	cfgPath := flag.String("config", defaultCfgPath, "Configuration file `path`.")
	dumpConfig := flag.Bool("dump-config", false, "write current configuration to stdout and exit (useful for migrating from command line flags to config file)")
	getVersion := flag.Bool("version", false, "print version information and exit.")

	flag.StringVar(&theConfig.ManagementToken, "management-token", theConfig.ManagementToken,
		"Authorization token to be included in all health check requests.")

	flag.Usage = usage
	flag.Parse()

	// Print version information if requested
	if *getVersion {
		fmt.Printf("arv-git-httpd %s\n", version)
		return
	}

	err := config.LoadFile(theConfig, *cfgPath)
	if err != nil {
		h := os.Getenv("ARVADOS_API_HOST")
		if h == "" || !os.IsNotExist(err) || *cfgPath != defaultCfgPath {
			log.Fatal(err)
		}
		log.Print("DEPRECATED: No config file found, but ARVADOS_API_HOST environment variable is set. Please use a config file instead.")
		theConfig.Client.APIHost = h
		if regexp.MustCompile("^(?i:1|yes|true)$").MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE")) {
			theConfig.Client.Insecure = true
		}
		if j, err := json.MarshalIndent(theConfig, "", "    "); err == nil {
			log.Print("Current configuration:\n", string(j))
		}
	}

	if *dumpConfig {
		log.Fatal(config.DumpAndExit(theConfig))
	}

	srv := &server{}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.Printf("Error notifying init daemon: %v", err)
	}
	log.Printf("arv-git-httpd %s started", version)
	log.Println("Listening at", srv.Addr)
	log.Println("Repository root", theConfig.RepoRoot)
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
