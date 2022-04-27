// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package githttpd

import (
	"context"
	"net"
	"net/http"
	"net/http/cgi"
	"os"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

// gitHandler is an http.Handler that invokes git-http-backend (or
// whatever backend is configured) via CGI, with appropriate
// environment variables in place for git-http-backend or
// gitolite-shell.
type gitHandler struct {
	cgi.Handler
}

func newGitHandler(ctx context.Context, cluster *arvados.Cluster) http.Handler {
	const glBypass = "GL_BYPASS_ACCESS_CHECKS"
	const glHome = "GITOLITE_HTTP_HOME"
	var env []string
	path := os.Getenv("PATH")
	if cluster.Git.GitoliteHome != "" {
		env = append(env,
			glHome+"="+cluster.Git.GitoliteHome,
			glBypass+"=1")
		path = path + ":" + cluster.Git.GitoliteHome + "/bin"
	} else if home, bypass := os.Getenv(glHome), os.Getenv(glBypass); home != "" || bypass != "" {
		env = append(env, glHome+"="+home, glBypass+"="+bypass)
		ctxlog.FromContext(ctx).Printf("DEPRECATED: Passing through %s and %s environment variables. Use GitoliteHome configuration instead.", glHome, glBypass)
	}

	var listen arvados.URL
	for listen = range cluster.Services.GitHTTP.InternalURLs {
		break
	}
	env = append(env,
		"GIT_PROJECT_ROOT="+cluster.Git.Repositories,
		"GIT_HTTP_EXPORT_ALL=",
		"SERVER_ADDR="+listen.Host,
		"PATH="+path)
	return &gitHandler{
		Handler: cgi.Handler{
			Path: cluster.Git.GitCommand,
			Dir:  cluster.Git.Repositories,
			Env:  env,
			Args: []string{"http-backend"},
		},
	}
}

func (h *gitHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	remoteHost, remotePort, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ctxlog.FromContext(r.Context()).Errorf("Internal error: SplitHostPort(r.RemoteAddr==%q): %s", r.RemoteAddr, err)
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
