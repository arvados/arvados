// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"

	"git.arvados.org/arvados.git/lib/controller/api"
	"git.arvados.org/arvados.git/lib/controller/federation"
	"git.arvados.org/arvados.git/lib/controller/localdb"
	"git.arvados.org/arvados.git/lib/controller/railsproxy"
	"git.arvados.org/arvados.git/lib/controller/router"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/health"
	"git.arvados.org/arvados.git/sdk/go/httpserver"

	// sqlx needs lib/pq to talk to PostgreSQL
	_ "github.com/lib/pq"
)

type Handler struct {
	Cluster           *arvados.Cluster
	BackgroundContext context.Context

	setupOnce      sync.Once
	federation     *federation.Conn
	handlerStack   http.Handler
	proxy          *proxy
	secureClient   *http.Client
	insecureClient *http.Client
	dbConnector    ctrlctx.DBConnector
	limitLogCreate chan struct{}
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
	_, err := h.dbConnector.GetDB(context.TODO())
	if err != nil {
		return err
	}
	_, _, err = railsproxy.FindRailsAPI(h.Cluster)
	if err != nil {
		return err
	}
	if h.Cluster.API.VocabularyPath != "" {
		req, err := http.NewRequest("GET", "/arvados/v1/vocabulary", nil)
		if err != nil {
			return err
		}
		var resp httptest.ResponseRecorder
		h.handlerStack.ServeHTTP(&resp, req)
		if resp.Result().StatusCode != http.StatusOK {
			return fmt.Errorf("%d %s", resp.Result().StatusCode, resp.Result().Status)
		}
	}
	return nil
}

func (h *Handler) Done() <-chan struct{} {
	return nil
}

func neverRedirect(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

func (h *Handler) setup() {
	mux := http.NewServeMux()
	healthFuncs := make(map[string]health.Func)

	h.dbConnector = ctrlctx.DBConnector{PostgreSQL: h.Cluster.PostgreSQL}
	go func() {
		<-h.BackgroundContext.Done()
		h.dbConnector.Close()
	}()
	oidcAuthorizer := localdb.OIDCAccessTokenAuthorizer(h.Cluster, h.dbConnector.GetDB)
	h.federation = federation.New(h.BackgroundContext, h.Cluster, &healthFuncs, h.dbConnector.GetDB)
	rtr := router.New(h.federation, router.Config{
		MaxRequestSize: h.Cluster.API.MaxRequestSize,
		WrapCalls: api.ComposeWrappers(
			ctrlctx.WrapCallsInTransactions(h.dbConnector.GetDB),
			oidcAuthorizer.WrapCalls,
			ctrlctx.WrapCallsWithAuth(h.Cluster)),
	})

	healthRoutes := health.Routes{"ping": func() error { _, err := h.dbConnector.GetDB(context.TODO()); return err }}
	for name, f := range healthFuncs {
		healthRoutes[name] = f
	}
	mux.Handle("/_health/", &health.Handler{
		Token:  h.Cluster.ManagementToken,
		Prefix: "/_health/",
		Routes: healthRoutes,
	})
	mux.Handle("/arvados/v1/config", rtr)
	mux.Handle("/arvados/v1/vocabulary", rtr)
	mux.Handle("/"+arvados.EndpointUserAuthenticate.Path, rtr) // must come before .../users/
	mux.Handle("/arvados/v1/collections", rtr)
	mux.Handle("/arvados/v1/collections/", rtr)
	mux.Handle("/arvados/v1/users", rtr)
	mux.Handle("/arvados/v1/users/", rtr)
	mux.Handle("/arvados/v1/connect/", rtr)
	mux.Handle("/arvados/v1/container_requests", rtr)
	mux.Handle("/arvados/v1/container_requests/", rtr)
	mux.Handle("/arvados/v1/groups", rtr)
	mux.Handle("/arvados/v1/groups/", rtr)
	mux.Handle("/arvados/v1/links", rtr)
	mux.Handle("/arvados/v1/links/", rtr)
	mux.Handle("/login", rtr)
	mux.Handle("/logout", rtr)
	mux.Handle("/arvados/v1/api_client_authorizations", rtr)
	mux.Handle("/arvados/v1/api_client_authorizations/", rtr)

	hs := http.NotFoundHandler()
	hs = prepend(hs, h.proxyRailsAPI)
	hs = prepend(hs, h.routeContainerEndpoints(rtr))
	hs = prepend(hs, h.limitLogCreateRequests)
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

	logCreateLimit := int(float64(h.Cluster.API.MaxConcurrentRequests) * h.Cluster.API.LogCreateRequestFraction)
	if logCreateLimit == 0 && h.Cluster.API.LogCreateRequestFraction > 0 {
		logCreateLimit = 1
	}
	h.limitLogCreate = make(chan struct{}, logCreateLimit)

	h.proxy = &proxy{
		Name: "arvados-controller",
	}

	go h.trashSweepWorker()
	go h.containerLogSweepWorker()
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

// Route /arvados/v1/containers/{uuid}/log*, .../ssh, and
// .../gateway_tunnel to rtr, pass everything else to next.
//
// (http.ServeMux doesn't let us route these without also routing
// everything under /containers/, which we don't want yet.)
func (h *Handler) routeContainerEndpoints(rtr http.Handler) middlewareFunc {
	return func(w http.ResponseWriter, req *http.Request, next http.Handler) {
		trim := strings.TrimPrefix(req.URL.Path, "/arvados/v1/containers/")
		if trim != req.URL.Path && (strings.Index(trim, "/log") == 27 ||
			strings.Index(trim, "/ssh") == 27 ||
			strings.Index(trim, "/gateway_tunnel") == 27) {
			rtr.ServeHTTP(w, req)
		} else {
			next.ServeHTTP(w, req)
		}
	}
}

func (h *Handler) limitLogCreateRequests(w http.ResponseWriter, req *http.Request, next http.Handler) {
	if cap(h.limitLogCreate) > 0 && req.Method == http.MethodPost && strings.HasPrefix(req.URL.Path, "/arvados/v1/logs") {
		select {
		case h.limitLogCreate <- struct{}{}:
			defer func() { <-h.limitLogCreate }()
			next.ServeHTTP(w, req)
		default:
			http.Error(w, "Excess log messages", http.StatusServiceUnavailable)
		}
		return
	}
	next.ServeHTTP(w, req)
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
