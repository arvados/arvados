// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"io"
	"net/http"
	"net/url"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type Handler struct {
	Cluster *arvados.Cluster

	setupOnce    sync.Once
	mux          http.ServeMux
	handlerStack http.Handler
	proxyClient  *arvados.Client
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.setupOnce.Do(h.setup)
	h.handlerStack.ServeHTTP(w, req)
}

func (h *Handler) setup() {
	h.mux.Handle("/_health/", &health.Handler{
		Token:  h.Cluster.ManagementToken,
		Prefix: "/_health/",
	})
	h.mux.Handle("/", http.HandlerFunc(h.proxyRailsAPI))
	h.handlerStack = httpserver.LogRequests(&h.mux)
}

func (h *Handler) proxyRailsAPI(w http.ResponseWriter, incomingReq *http.Request) {
	url, err := findRailsAPI(h.Cluster)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req := *incomingReq
	req.URL.Host = url.Host
	resp, err := arvados.InsecureHTTPClient.Do(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for k, v := range resp.Header {
		for _, v := range v {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// For now, findRailsAPI always uses the rails API running on this
// node.
func findRailsAPI(cluster *arvados.Cluster) (*url.URL, error) {
	node, err := cluster.GetThisSystemNode()
	if err != nil {
		return nil, err
	}
	return url.Parse("http://" + node.RailsAPI.Listen)
}
