// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type Handler struct {
	Cluster     *arvados.Cluster
	NodeProfile *arvados.NodeProfile

	setupOnce    sync.Once
	handlerStack http.Handler
	proxyClient  *arvados.Client
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.setupOnce.Do(h.setup)
	h.handlerStack.ServeHTTP(w, req)
}

func (h *Handler) CheckHealth() error {
	h.setupOnce.Do(h.setup)
	_, err := findRailsAPI(h.Cluster, h.NodeProfile)
	return err
}

func (h *Handler) setup() {
	mux := http.NewServeMux()
	mux.Handle("/_health/", &health.Handler{
		Token:  h.Cluster.ManagementToken,
		Prefix: "/_health/",
	})
	hs := http.NotFoundHandler()
	hs = prepend(hs, h.proxyRailsAPI)
	hs = prepend(hs, h.proxyRemoteCluster)
	mux.Handle("/", hs)
	h.handlerStack = mux
}

type middlewareFunc func(http.ResponseWriter, *http.Request, http.Handler)

func prepend(next http.Handler, middleware middlewareFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		middleware(w, req, next)
	})
}

func (h *Handler) proxyRailsAPI(w http.ResponseWriter, req *http.Request, next http.Handler) {
	urlOut, err := findRailsAPI(h.Cluster, h.NodeProfile)
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	urlOut = &url.URL{
		Scheme:   urlOut.Scheme,
		Host:     urlOut.Host,
		Path:     req.URL.Path,
		RawPath:  req.URL.RawPath,
		RawQuery: req.URL.RawQuery,
	}
	h.proxy(w, req, urlOut)
}

// For now, findRailsAPI always uses the rails API running on this
// node.
func findRailsAPI(cluster *arvados.Cluster, np *arvados.NodeProfile) (*url.URL, error) {
	hostport := np.RailsAPI.Listen
	if len(hostport) > 1 && hostport[0] == ':' && strings.TrimRight(hostport[1:], "0123456789") == "" {
		// ":12345" => connect to indicated port on localhost
		hostport = "localhost" + hostport
	} else if _, _, err := net.SplitHostPort(hostport); err == nil {
		// "[::1]:12345" => connect to indicated address & port
	} else {
		return nil, err
	}
	proto := "http"
	if np.RailsAPI.TLS {
		proto = "https"
	}
	return url.Parse(proto + "://" + hostport)
}
