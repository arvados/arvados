// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-systemd/daemon"
)

type server struct {
	httpServer  *http.Server
	listener    net.Listener
	wsConfig    *wsConfig
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

	ln, err := net.Listen("tcp", srv.wsConfig.Listen)
	if err != nil {
		log.WithField("Listen", srv.wsConfig.Listen).Fatal(err)
	}
	log.WithField("Listen", ln.Addr().String()).Info("listening")

	srv.listener = ln
	srv.eventSource = &pgEventSource{
		DataSource:   srv.wsConfig.Postgres.ConnectionString(),
		MaxOpenConns: srv.wsConfig.PostgresPool,
		QueueSize:    srv.wsConfig.ServerEventQueue,
	}
	srv.httpServer = &http.Server{
		Addr:           srv.wsConfig.Listen,
		ReadTimeout:    time.Minute,
		WriteTimeout:   time.Minute,
		MaxHeaderBytes: 1 << 20,
		Handler: &router{
			Config:         srv.wsConfig,
			eventSource:    srv.eventSource,
			newPermChecker: func() permChecker { return newPermChecker(srv.wsConfig.Client) },
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
