// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	HeaderRequestID = "X-Request-Id"
)

// IDGenerator generates alphanumeric strings suitable for use as
// unique IDs (a given IDGenerator will never return the same ID
// twice).
type IDGenerator struct {
	// Prefix is prepended to each returned ID.
	Prefix string

	mtx sync.Mutex
	src rand.Source
}

// Next returns a new ID string. It is safe to call Next from multiple
// goroutines.
func (g *IDGenerator) Next() string {
	g.mtx.Lock()
	defer g.mtx.Unlock()
	if g.src == nil {
		g.src = rand.NewSource(time.Now().UnixNano())
	}
	a, b := g.src.Int63(), g.src.Int63()
	id := strconv.FormatInt(a, 36) + strconv.FormatInt(b, 36)
	for len(id) > 20 {
		id = id[:20]
	}
	return g.Prefix + id
}

// AddRequestIDs wraps an http.Handler, adding an X-Request-Id header
// to each request that doesn't already have one.
func AddRequestIDs(h http.Handler) http.Handler {
	gen := &IDGenerator{Prefix: "req-"}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get(HeaderRequestID) == "" {
			if req.Header == nil {
				req.Header = http.Header{}
			}
			req.Header.Set(HeaderRequestID, gen.Next())
		}
		w.Header().Set("X-Request-Id", req.Header.Get("X-Request-Id"))
		h.ServeHTTP(w, req)
	})
}
