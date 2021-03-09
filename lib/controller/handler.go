// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/controller/api"
	"git.arvados.org/arvados.git/lib/controller/federation"
	"git.arvados.org/arvados.git/lib/controller/localdb"
	"git.arvados.org/arvados.git/lib/controller/railsproxy"
	"git.arvados.org/arvados.git/lib/controller/router"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/jmoiron/sqlx"

	// sqlx needs lib/pq to talk to PostgreSQL
	_ "github.com/lib/pq"
)

type Handler struct {
	Cluster *arvados.Cluster

	setupOnce      sync.Once
	handlerStack   http.Handler
	proxy          *proxy
	secureClient   *http.Client
	insecureClient *http.Client
	pgdb           *sqlx.DB
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
	if h.Cluster.API.RequestTimeout > 0 {
		ctx, cancel := context.WithDeadline(req.Context(), time.Now().Add(time.Duration(h.Cluster.API.RequestTimeout)))
		req = req.WithContext(ctx)
		defer cancel()
	}

	h.handlerStack.ServeHTTP(w, req)
}

func (h *Handler) CheckHealth() error {
	h.setupOnce.Do(h.setup)
	_, err := h.db(context.TODO())
	if err != nil {
		return err
	}
	_, _, err = railsproxy.FindRailsAPI(h.Cluster)
	return err
}

func (h *Handler) Done() <-chan struct{} {
	return nil
}

func neverRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

func (h *Handler) setup() {
	mux := http.NewServeMux()
	mux.Handle("/_health/", &health.Handler{
		Token:  h.Cluster.ManagementToken,
		Prefix: "/_health/",
		Routes: health.Routes{"ping": func() error { _, err := h.db(context.TODO()); return err }},
	})

	oidcAuthorizer := localdb.OIDCAccessTokenAuthorizer(h.Cluster, h.db)
	rtr := router.New(federation.New(h.Cluster), api.ComposeWrappers(ctrlctx.WrapCallsInTransactions(h.db), oidcAuthorizer.WrapCalls))
	mux.Handle("/arvados/v1/config", rtr)
	mux.Handle("/"+arvados.EndpointUserAuthenticate.Path, rtr)

	if !h.Cluster.ForceLegacyAPI14 {
		mux.Handle("/arvados/v1/collections", rtr)
		mux.Handle("/arvados/v1/collections/", rtr)
		mux.Handle("/arvados/v1/users", rtr)
		mux.Handle("/arvados/v1/users/", rtr)
		mux.Handle("/arvados/v1/connect/", rtr)
		mux.Handle("/arvados/v1/container_requests", rtr)
		mux.Handle("/arvados/v1/container_requests/", rtr)
		mux.Handle("/arvados/v1/groups", rtr)
		mux.Handle("/arvados/v1/groups/", rtr)
		mux.Handle("/login", rtr)
		mux.Handle("/logout", rtr)
	}

	hs := http.NotFoundHandler()
	hs = prepend(hs, h.proxyRailsAPI)
	hs = h.setupProxyRemoteCluster(hs)
	hs = prepend(hs, oidcAuthorizer.Middleware)
	mux.Handle("/", hs)
	h.handlerStack = mux

	sc := *arvados.DefaultSecureClient
	sc.CheckRedirect = neverRedirect
	h.secureClient = &sc

	ic := *arvados.InsecureHTTPClient
	ic.CheckRedirect = neverRedirect
	h.insecureClient = &ic

	h.proxy = &proxy{
		Name: "arvados-controller",
	}
}

var errDBConnection = errors.New("database connection error")

func (h *Handler) db(ctx context.Context) (*sqlx.DB, error) {
	h.pgdbMtx.Lock()
	defer h.pgdbMtx.Unlock()
	if h.pgdb != nil {
		return h.pgdb, nil
	}

	db, err := sqlx.Open("postgres", h.Cluster.PostgreSQL.Connection.String())
	if err != nil {
		ctxlog.FromContext(ctx).WithError(err).Error("postgresql connect failed")
		return nil, errDBConnection
	}
	if p := h.Cluster.PostgreSQL.ConnectionPool; p > 0 {
		db.SetMaxOpenConns(p)
	}
	if err := db.Ping(); err != nil {
		ctxlog.FromContext(ctx).WithError(err).Error("postgresql connect succeeded but ping failed")
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

func (h *Handler) localClusterRequest(req *http.Request) (*http.Response, error) {
	urlOut, insecure, err := railsproxy.FindRailsAPI(h.Cluster)
	if err != nil {
		return nil, err
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
	return h.proxy.Do(req, urlOut, client)
}

func (h *Handler) proxyRailsAPI(w http.ResponseWriter, req *http.Request, next http.Handler) {
	resp, err := h.localClusterRequest(req)
	n, err := h.proxy.ForwardResponse(w, resp, err)
	if err != nil {
		httpserver.Logger(req).WithError(err).WithField("bytesCopied", n).Error("error copying response body")
	}
}

// Use a localhost entry from Services.RailsAPI.InternalURLs if one is
// present, otherwise choose an arbitrary entry.
func findRailsAPI(cluster *arvados.Cluster) (*url.URL, bool, error) {
	var best *url.URL
	for target := range cluster.Services.RailsAPI.InternalURLs {
		target := url.URL(target)
		best = &target
		if strings.HasPrefix(target.Host, "localhost:") || strings.HasPrefix(target.Host, "127.0.0.1:") || strings.HasPrefix(target.Host, "[::1]:") {
			break
		}
	}
	if best == nil {
		return nil, false, fmt.Errorf("Services.RailsAPI.InternalURLs is empty")
	}
	return best, cluster.TLS.Insecure, nil
}
