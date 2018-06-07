// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"flag"
	"fmt"
	"io"
	"net/http"

	"git.curoverse.com/arvados.git/lib/cmd"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"github.com/Sirupsen/logrus"
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

var Command cmd.Handler = &command{}

type command struct{}

func (*command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	log := logrus.StandardLogger()
	log.Formatter = &logrus.JSONFormatter{
		TimestampFormat: rfc3339NanoFixed,
	}
	log.Out = stderr

	var err error
	defer func() {
		if err != nil {
			log.WithError(err).Info("exiting")
		}
	}()
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configFile := flags.String("config", arvados.DefaultConfigFile, "Site configuration `file`")
	err = flags.Parse(args)
	if err != nil {
		return 2
	}
	cfg, err := arvados.GetConfig(*configFile)
	if err != nil {
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}
	node, err := cluster.GetThisSystemNode()
	if err != nil {
		return 1
	}
	if node.Controller.Listen == "" {
		err = fmt.Errorf("configuration does not run a controller on this host: Clusters[%q].SystemNodes[`hostname` or *].Controller.Listen == \"\"", cluster.ClusterID)
		return 1
	}
	srv := &httpserver.Server{
		Server: http.Server{
			Handler: httpserver.LogRequests(&Handler{
				Cluster: cluster,
			}),
		},
		Addr: node.Controller.Listen,
	}
	err = srv.Start()
	if err != nil {
		return 1
	}
	log.WithField("Listen", srv.Addr).Info("listening")
	err = srv.Wait()
	if err != nil {
		return 1
	}
	return 0
}
