package main

import (
	"net/http"
	"net/http/cgi"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type server struct {
	httpserver.Server
}

func (srv *server) Start() error {
	gitHandler := &cgi.Handler{
		Path: theConfig.GitCommand,
		Dir:  theConfig.Root,
		Env: []string{
			"GIT_PROJECT_ROOT=" + theConfig.Root,
			"GIT_HTTP_EXPORT_ALL=",
			"SERVER_ADDR=" + theConfig.Addr,
		},
		InheritEnv: []string{
			"PATH",
			// Needed if GitCommand is gitolite-shell:
			"GITOLITE_HTTP_HOME",
			"GL_BYPASS_ACCESS_CHECKS",
		},
		Args:       []string{"http-backend"},
	}
	mux := http.NewServeMux()
	mux.Handle("/", &authHandler{gitHandler})
	srv.Handler = mux
	srv.Addr = theConfig.Addr
	return srv.Server.Start()
}
