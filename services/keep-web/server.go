package main

import (
	"flag"
	"net/http"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

var address string

func init() {
	flag.StringVar(&address, "listen", ":80",
		"Address to listen on: \"host:port\", or \":port\" to listen on all interfaces.")
}

type server struct {
	httpserver.Server
}

func (srv *server) Start() error {
	mux := http.NewServeMux()
	mux.Handle("/", &handler{})
	srv.Handler = mux
	srv.Addr = address
	return srv.Server.Start()
}
