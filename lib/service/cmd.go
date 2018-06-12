// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// package service provides a cmd.Handler that brings up a system service.
package service

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

type NewHandlerFunc func(*arvados.Cluster, *arvados.SystemNode) http.Handler

type command struct {
	newHandler NewHandlerFunc
	svcName    arvados.ServiceName
}

// Command returns a cmd.Handler that loads site config, calls
// newHandler with the current cluster and node configs, and brings up
// an http server with the returned handler.
//
// The handler is wrapped with server middleware (adding X-Request-ID
// headers, logging requests/responses, etc).
func Command(svcName arvados.ServiceName, newHandler NewHandlerFunc) cmd.Handler {
	return &command{
		newHandler: newHandler,
		svcName:    svcName,
	}
}

func (c *command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	log := logrus.New()
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
	if err == flag.ErrHelp {
		err = nil
		return 0
	} else if err != nil {
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
	listen := node.ServicePorts()[c.svcName]
	if listen == "" {
		err = fmt.Errorf("configuration does not enable the %s service on this host", c.svcName)
		return 1
	}
	srv := &httpserver.Server{
		Server: http.Server{
			Handler: httpserver.AddRequestIDs(httpserver.LogRequests(log, c.newHandler(cluster, node))),
		},
		Addr: listen,
	}
	err = srv.Start()
	if err != nil {
		return 1
	}
	log.WithFields(logrus.Fields{
		"Listen":  srv.Addr,
		"Service": c.svcName,
	}).Info("listening")
	err = srv.Wait()
	if err != nil {
		return 1
	}
	return 0
}

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"
