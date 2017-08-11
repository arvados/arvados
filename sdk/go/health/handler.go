// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
)

// Func is a health-check function: it returns nil when healthy, an
// error when not.
type Func func() error

// Routes is a map of URI path to health-check function.
type Routes map[string]Func

// Handler is an http.Handler that responds to authenticated
// health-check requests with JSON responses like {"health":"OK"} or
// {"health":"ERROR","error":"error text"}.
//
// Fields of a Handler should not be changed after the Handler is
// first used.
type Handler struct {
	setupOnce sync.Once
	mux       *http.ServeMux

	// Authentication token. If empty, all requests will return 404.
	Token string

	// Route prefix, typically "/_health/".
	Prefix string

	// Map of URI paths to health-check Func. The prefix is
	// omitted: Routes["foo"] is the health check invoked by a
	// request to "{Prefix}/foo".
	//
	// If "ping" is not listed here, it will be added
	// automatically and will always return a "healthy" response.
	Routes Routes

	// If non-nil, Log is called after handling each request. The
	// error argument is nil if the request was successfully
	// authenticated and served, even if the health check itself
	// failed.
	Log func(*http.Request, error)
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.setupOnce.Do(h.setup)
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) setup() {
	h.mux = http.NewServeMux()
	prefix := h.Prefix
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	for name, fn := range h.Routes {
		h.mux.Handle(prefix+name, h.healthJSON(fn))
	}
	if _, ok := h.Routes["ping"]; !ok {
		h.mux.Handle(prefix+"ping", h.healthJSON(func() error { return nil }))
	}
}

var (
	healthyBody     = []byte(`{"health":"OK"}` + "\n")
	errNotFound     = errors.New(http.StatusText(http.StatusNotFound))
	errUnauthorized = errors.New(http.StatusText(http.StatusUnauthorized))
	errForbidden    = errors.New(http.StatusText(http.StatusForbidden))
)

func (h *Handler) healthJSON(fn Func) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		defer func() {
			if h.Log != nil {
				h.Log(r, err)
			}
		}()
		if h.Token == "" {
			http.Error(w, "disabled", http.StatusNotFound)
			err = errNotFound
		} else if ah := r.Header.Get("Authorization"); ah == "" {
			http.Error(w, "authorization required", http.StatusUnauthorized)
			err = errUnauthorized
		} else if ah != "Bearer "+h.Token {
			http.Error(w, "authorization error", http.StatusForbidden)
			err = errForbidden
		} else if err = fn(); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Write(healthyBody)
		} else {
			w.Header().Set("Content-Type", "application/json")
			enc := json.NewEncoder(w)
			err = enc.Encode(map[string]string{
				"health": "ERROR",
				"error":  err.Error(),
			})
		}
	})
}
