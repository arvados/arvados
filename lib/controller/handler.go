// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

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
	mux.Handle("/", http.HandlerFunc(h.proxyRailsAPI))
	h.handlerStack = mux
}

// headers that shouldn't be forwarded when proxying. See
// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers
var dropHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"TE":                true,
	"Trailer":           true,
	"Transfer-Encoding": true,
	"Upgrade":           true,
}

func (h *Handler) proxyRailsAPI(w http.ResponseWriter, reqIn *http.Request) {
	urlOut, err := findRailsAPI(h.Cluster, h.NodeProfile)
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusInternalServerError)
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
		if !dropHeaders[k] {
			hdrOut[k] = v
		}
	}
	xff := reqIn.RemoteAddr
	if xffIn := reqIn.Header.Get("X-Forwarded-For"); xffIn != "" {
		xff = xffIn + "," + xff
	}
	hdrOut.Set("X-Forwarded-For", xff)
	hdrOut.Add("Via", reqIn.Proto+" arvados-controller")

	ctx := reqIn.Context()
	if timeout := h.Cluster.HTTPRequestTimeout; timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(time.Duration(timeout)))
		defer cancel()
	}

	reqOut := (&http.Request{
		Method: reqIn.Method,
		URL:    urlOut,
		Header: hdrOut,
		Body:   reqIn.Body,
	}).WithContext(ctx)
	resp, err := arvados.InsecureHTTPClient.Do(reqOut)
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusInternalServerError)
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
