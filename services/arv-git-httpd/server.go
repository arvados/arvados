package main

import (
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type server struct {
	httpserver.Server
}

func (srv *server) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/", &authHandler{handler: newGitHandler()})
	srv.Handler = mux
	srv.Addr = theConfig.Listen
	return srv.Server.Start()
}
