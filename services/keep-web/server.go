// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"github.com/prometheus/client_golang/prometheus"
)

type server struct {
	httpserver.Server
	Config *Config
}

func (srv *server) Start() error {
	h := &handler{Config: srv.Config}
	reg := prometheus.NewRegistry()
	h.Config.Cache.registry = reg
	mh := httpserver.Instrument(reg, nil, httpserver.AddRequestIDs(httpserver.LogRequests(nil, h)))
	h.MetricsAPI = mh.ServeAPI(http.NotFoundHandler())
	srv.Handler = mh
	srv.Addr = srv.Config.Listen
	return srv.Server.Start()
}
