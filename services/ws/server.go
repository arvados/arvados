// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net"
	"net/http"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/coreos/go-systemd/daemon"
)

type server struct {
	httpServer  *http.Server
	listener    net.Listener
	cluster     *arvados.Cluster
	eventSource *pgEventSource
	setupOnce   sync.Once
}

func (srv *server) Close() {
	srv.WaitReady()
	srv.eventSource.Close()
	srv.listener.Close()
}

func (srv *server) WaitReady() {
	srv.setupOnce.Do(srv.setup)
	srv.eventSource.WaitReady()
}

func (srv *server) Run() error {
	srv.setupOnce.Do(srv.setup)
	return srv.httpServer.Serve(srv.listener)
}

func (srv *server) setup() {
	log := logger(nil)

	var listen arvados.URL
	for listen, _ = range srv.cluster.Services.Websocket.InternalURLs {
		break
	}
	ln, err := net.Listen("tcp", listen.Host)
	if err != nil {
		log.WithField("Listen", listen).Fatal(err)
	}
	log.WithField("Listen", ln.Addr().String()).Info("listening")

	client := arvados.Client{}
	client.APIHost = srv.cluster.Services.Controller.ExternalURL.Host
	client.AuthToken = srv.cluster.SystemRootToken
	client.Insecure = srv.cluster.TLS.Insecure

	srv.listener = ln
	srv.eventSource = &pgEventSource{
		DataSource:   srv.cluster.PostgreSQL.Connection.String(),
		MaxOpenConns: srv.cluster.PostgreSQL.ConnectionPool,
		QueueSize:    srv.cluster.API.WebsocketServerEventQueue,
	}

	srv.httpServer = &http.Server{
		Addr:           listen.Host,
		ReadTimeout:    time.Minute,
		WriteTimeout:   time.Minute,
		MaxHeaderBytes: 1 << 20,
		Handler: &router{
			cluster:        srv.cluster,
			client:         client,
			eventSource:    srv.eventSource,
			newPermChecker: func() permChecker { return newPermChecker(client) },
		},
	}

	go func() {
		srv.eventSource.Run()
		log.Info("event source stopped")
		srv.Close()
	}()

	if _, err := daemon.SdNotify(false, "READY=1"); err != nil {
		log.WithError(err).Warn("error notifying init daemon")
	}
}
