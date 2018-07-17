// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type server struct {
	httpserver.Server
	Config *Config
}

func (srv *server) Start() error {
	srv.Handler = httpserver.AddRequestIDs(httpserver.LogRequests(nil, &handler{Config: srv.Config}))
	srv.Addr = srv.Config.Listen
	return srv.Server.Start()
}
