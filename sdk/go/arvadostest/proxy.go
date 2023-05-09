// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"gopkg.in/check.v1"
)

type Proxy struct {
	*httptest.Server

	// URL where the proxy is listening. Same as Server.URL, but
	// with parsing already done for you.
	URL *url.URL

	// A dump of each request that has been proxied.
	RequestDumps [][]byte

	// If non-nil, func will be called on each incoming request
	// before proxying it.
	Director func(*http.Request)
}

// NewProxy returns a new Proxy that saves a dump of each reqeust
// before forwarding to the indicated service.
func NewProxy(c *check.C, svc arvados.Service) *Proxy {
	var target url.URL
	c.Assert(svc.InternalURLs, check.HasLen, 1)
	for u := range svc.InternalURLs {
		target = url.URL(u)
		break
	}
	rp := httputil.NewSingleHostReverseProxy(&target)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		dump, _ := httputil.DumpRequest(r, false)
		c.Logf("arvadostest.Proxy ErrorHandler(%s): %s\n%s", r.URL, err, dump)
		http.Error(w, err.Error(), http.StatusBadGateway)
	}
	rp.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	srv := httptest.NewServer(rp)
	u, err := url.Parse(srv.URL)
	c.Assert(err, check.IsNil)
	proxy := &Proxy{
		Server: srv,
		URL:    u,
	}
	rp.Director = func(r *http.Request) {
		if proxy.Director != nil {
			proxy.Director(r)
		}
		dump, _ := httputil.DumpRequest(r, true)
		proxy.RequestDumps = append(proxy.RequestDumps, dump)
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
	}
	return proxy
}
