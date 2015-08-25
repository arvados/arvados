package main

import (
	"log"
	"net"
	"net/http"
	"net/http/cgi"
)

// gitHandler is an http.Handler that invokes git-http-backend (or
// whatever backend is configured) via CGI, with appropriate
// environment variables in place for git-http-backend or
// gitolite-shell.
type gitHandler struct {
	cgi.Handler
}

func newGitHandler() http.Handler {
	return &gitHandler{
		Handler: cgi.Handler{
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
		},
	}
}

func (h *gitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteHost, remotePort, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		log.Printf("Internal error: SplitHostPort(r.RemoteAddr==%q): %s", r.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Copy the wrapped cgi.Handler, so these request-specific
	// variables don't leak into the next request.
	handlerCopy := h.Handler
	handlerCopy.Env = append(handlerCopy.Env,
		// In Go1.5 we can skip this, net/http/cgi will do it for us:
		"REMOTE_HOST="+remoteHost,
		"REMOTE_ADDR="+remoteHost,
		"REMOTE_PORT="+remotePort,
		// Ideally this would be a real username:
		"REMOTE_USER="+r.RemoteAddr,
	)
	handlerCopy.ServeHTTP(w, r)
}
