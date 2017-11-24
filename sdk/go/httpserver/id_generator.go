// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// IDGenerator generates alphanumeric strings suitable for use as
// unique IDs (a given IDGenerator will never return the same ID
// twice).
type IDGenerator struct {
	// Prefix is prepended to each returned ID.
	Prefix string

	lastID int64
	mtx    sync.Mutex
}

// Next returns a new ID string. It is safe to call Next from multiple
// goroutines.
func (g *IDGenerator) Next() string {
	id := time.Now().UnixNano()
	g.mtx.Lock()
	if id <= g.lastID {
		id = g.lastID + 1
	}
	g.lastID = id
	g.mtx.Unlock()
	return g.Prefix + strconv.FormatInt(id, 36)
}

// AddRequestIDs wraps an http.Handler, adding an X-Request-Id header
// to each request that doesn't already have one.
func AddRequestIDs(h http.Handler) http.Handler {
	gen := &IDGenerator{Prefix: "req-"}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Request-Id") == "" {
			req.Header.Set("X-Request-Id", gen.Next())
		}
		h.ServeHTTP(w, req)
	})
}
