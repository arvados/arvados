// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"database/sql"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	_ "github.com/lib/pq"
)

type Handler struct {
	Cluster     *arvados.Cluster
	NodeProfile *arvados.NodeProfile

	setupOnce      sync.Once
	handlerStack   http.Handler
	proxy          *proxy
	secureClient   *http.Client
	insecureClient *http.Client
	pgdb           *sql.DB
	pgdbMtx        sync.Mutex
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h.setupOnce.Do(h.setup)
	if req.Method != "GET" && req.Method != "HEAD" {
		// http.ServeMux returns 301 with a cleaned path if
		// the incoming request has a double slash. Some
		// clients (including the Go standard library) change
		// the request method to GET when following a 301
		// redirect if the original method was not HEAD
		// (RFC7231 6.4.2 specifically allows this in the case
		// of POST). Thus "POST //foo" gets misdirected to
		// "GET /foo". To avoid this, eliminate double slashes
		// before passing the request to ServeMux.
		for strings.Contains(req.URL.Path, "//") {
			req.URL.Path = strings.Replace(req.URL.Path, "//", "/", -1)
		}
	}
	h.handlerStack.ServeHTTP(w, req)
}

func (h *Handler) CheckHealth() error {
	h.setupOnce.Do(h.setup)
	_, _, err := findRailsAPI(h.Cluster, h.NodeProfile)
	return err
}

func neverRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

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

	sc := *arvados.DefaultSecureClient
	sc.Timeout = time.Duration(h.Cluster.HTTPRequestTimeout)
	sc.CheckRedirect = neverRedirect
	h.secureClient = &sc

	ic := *arvados.InsecureHTTPClient
	ic.Timeout = time.Duration(h.Cluster.HTTPRequestTimeout)
	ic.CheckRedirect = neverRedirect
	h.insecureClient = &ic

	h.proxy = &proxy{
		Name:           "arvados-controller",
		RequestTimeout: time.Duration(h.Cluster.HTTPRequestTimeout),
	}
}

var errDBConnection = errors.New("database connection error")

func (h *Handler) db(req *http.Request) (*sql.DB, error) {
	h.pgdbMtx.Lock()
	defer h.pgdbMtx.Unlock()
	if h.pgdb != nil {
		return h.pgdb, nil
	}

	db, err := sql.Open("postgres", h.Cluster.PostgreSQL.Connection.String())
	if err != nil {
		httpserver.Logger(req).WithError(err).Error("postgresql connect failed")
		return nil, errDBConnection
	}
	if p := h.Cluster.PostgreSQL.ConnectionPool; p > 0 {
		db.SetMaxOpenConns(p)
	}
	if err := db.Ping(); err != nil {
		httpserver.Logger(req).WithError(err).Error("postgresql connect succeeded but ping failed")
		return nil, errDBConnection
	}
	h.pgdb = db
	return db, nil
}

type middlewareFunc func(http.ResponseWriter, *http.Request, http.Handler)

func prepend(next http.Handler, middleware middlewareFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		middleware(w, req, next)
	})
}

func (h *Handler) proxyRailsAPI(w http.ResponseWriter, req *http.Request, next http.Handler) {
	urlOut, insecure, err := findRailsAPI(h.Cluster, h.NodeProfile)
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
	client := h.secureClient
	if insecure {
		client = h.insecureClient
	}
	h.proxy.Do(w, req, urlOut, client)
}

// For now, findRailsAPI always uses the rails API running on this
// node.
func findRailsAPI(cluster *arvados.Cluster, np *arvados.NodeProfile) (*url.URL, bool, error) {
	hostport := np.RailsAPI.Listen
	if len(hostport) > 1 && hostport[0] == ':' && strings.TrimRight(hostport[1:], "0123456789") == "" {
		// ":12345" => connect to indicated port on localhost
		hostport = "localhost" + hostport
	} else if _, _, err := net.SplitHostPort(hostport); err == nil {
		// "[::1]:12345" => connect to indicated address & port
	} else {
		return nil, false, err
	}
	proto := "http"
	if np.RailsAPI.TLS {
		proto = "https"
	}
	url, err := url.Parse(proto + "://" + hostport)
	return url, np.RailsAPI.Insecure, err
}
