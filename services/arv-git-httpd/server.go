// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type server struct {
	httpserver.Server
	cluster *arvados.Cluster
}

func (srv *server) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/", &authHandler{handler: newGitHandler(srv.cluster), cluster: srv.cluster})
	mux.Handle("/_health/", &health.Handler{
		Token:  srv.cluster.ManagementToken,
		Prefix: "/_health/",
	})

	var listen arvados.URL
	for listen = range srv.cluster.Services.GitHTTP.InternalURLs {
		break
	}

	srv.Handler = mux
	srv.Addr = listen.Host
	return srv.Server.Start()
}
