// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"fmt"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	log "github.com/Sirupsen/logrus"
)

var version = "dev"

func main() {
	configFile := flag.String("config", arvados.DefaultConfigFile, "`path` to arvados configuration file")
	getVersion := flag.Bool("version", false, "Print version information and exit.")
	flag.Parse()

	// Print version information if requested
	if *getVersion {
		fmt.Printf("arvados-health %s\n", version)
		return
	}

	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000000Z07:00",
	})
	log.Printf("arvados-health %s started", version)

	cfg, err := arvados.GetConfig(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	clusterCfg, err := cfg.GetCluster("")
	if err != nil {
		log.Fatal(err)
	}
	nodeCfg, err := clusterCfg.GetNodeProfile("")
	if err != nil {
		log.Fatal(err)
	}

	log := log.WithField("Service", "Health")
	srv := &httpserver.Server{
		Addr: nodeCfg.Health.Listen,
		Server: http.Server{
			Handler: &health.Aggregator{
				Config: cfg,
				Log: func(req *http.Request, err error) {
					log.WithField("RemoteAddr", req.RemoteAddr).
						WithField("Path", req.URL.Path).
						WithError(err).
						Info("HTTP request")
				},
			},
		},
	}
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
	log.WithField("Listen", srv.Addr).Info("listening")
	if err := srv.Wait(); err != nil {
		log.Fatal(err)
	}
}
