// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type server struct {
	httpserver.Server
}

func (srv *server) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/", &authHandler{handler: newGitHandler()})
	mux.Handle("/_health/", &health.Handler{
		Token:  theConfig.ManagementToken,
		Prefix: "/_health/",
	})
	srv.Handler = mux
	srv.Addr = theConfig.Listen
	return srv.Server.Start()
}
