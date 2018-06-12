// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"io"
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
	Cluster *arvados.Cluster
	Node    *arvados.SystemNode

	setupOnce    sync.Once
	handlerStack http.Handler
	proxyClient  *arvados.Client
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.setupOnce.Do(h.setup)
	h.handlerStack.ServeHTTP(w, req)
}

func (h *Handler) setup() {
	mux := http.NewServeMux()
	mux.Handle("/_health/", &health.Handler{
		Token:  h.Cluster.ManagementToken,
		Prefix: "/_health/",
	})
	mux.Handle("/", http.HandlerFunc(h.proxyRailsAPI))
	h.handlerStack = mux
}

func (h *Handler) proxyRailsAPI(w http.ResponseWriter, reqIn *http.Request) {
	urlOut, err := findRailsAPI(h.Cluster, h.Node)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	urlOut = &url.URL{
		Scheme:   urlOut.Scheme,
		Host:     urlOut.Host,
		Path:     reqIn.URL.Path,
		RawPath:  reqIn.URL.RawPath,
		RawQuery: reqIn.URL.RawQuery,
	}

	// Copy headers from incoming request, then add/replace proxy
	// headers like Via and X-Forwarded-For.
	hdrOut := http.Header{}
	for k, v := range reqIn.Header {
		hdrOut[k] = v
	}
	xff := reqIn.RemoteAddr
	if xffIn := reqIn.Header.Get("X-Forwarded-For"); xffIn != "" {
		xff = xffIn + "," + xff
	}
	hdrOut.Set("X-Forwarded-For", xff)
	hdrOut.Add("Via", reqIn.Proto+" arvados-controller")

	reqOut := (&http.Request{
		Method: reqIn.Method,
		URL:    urlOut,
		Header: hdrOut,
	}).WithContext(reqIn.Context())
	resp, err := arvados.InsecureHTTPClient.Do(reqOut)
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
	n, err := io.Copy(w, resp.Body)
	if err != nil {
		httpserver.Logger(reqIn).WithError(err).WithField("bytesCopied", n).Error("error copying response body")
	}
}

// For now, findRailsAPI always uses the rails API running on this
// node.
func findRailsAPI(cluster *arvados.Cluster, node *arvados.SystemNode) (*url.URL, error) {
	hostport := node.RailsAPI.Listen
	if len(hostport) > 1 && hostport[0] == ':' && strings.TrimRight(hostport[1:], "0123456789") == "" {
		// ":12345" => connect to indicated port on localhost
		hostport = "localhost" + hostport
	} else if _, _, err := net.SplitHostPort(hostport); err == nil {
		// "[::1]:12345" => connect to indicated address & port
	} else {
		return nil, err
	}
	proto := "http"
	if node.RailsAPI.TLS {
		proto = "https"
	}
	return url.Parse(proto + "://" + hostport)
}
