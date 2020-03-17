// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"net/http"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type server struct {
	httpserver.Server
	Config *Config
}

func (srv *server) Start() error {
	h := &handler{Config: srv.Config}
	reg := prometheus.NewRegistry()
	h.Config.Cache.registry = reg
	ctx := ctxlog.Context(context.Background(), logrus.StandardLogger())
	mh := httpserver.Instrument(reg, nil, httpserver.HandlerWithContext(ctx, httpserver.AddRequestIDs(httpserver.LogRequests(h))))
	h.MetricsAPI = mh.ServeAPI(h.Config.cluster.ManagementToken, http.NotFoundHandler())
	srv.Handler = mh
	var listen arvados.URL
	for listen = range srv.Config.cluster.Services.WebDAV.InternalURLs {
		break
	}
	if len(srv.Config.cluster.Services.WebDAV.InternalURLs) > 1 {
		logrus.Warn("Services.WebDAV.InternalURLs has more than one key; picked: ", listen)
	}
	srv.Addr = listen.Host
	return srv.Server.Start()
}
