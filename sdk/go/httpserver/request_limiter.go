// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"net/http"
)

// RequestCounter is an http.Handler that tracks the number of
// requests in progress.
type RequestCounter interface {
	http.Handler

	// Current() returns the number of requests in progress.
	Current() int

	// Max() returns the maximum number of concurrent requests
	// that will be accepted.
	Max() int
}

type limiterHandler struct {
	requests chan struct{}
	handler  http.Handler
}

// NewRequestLimiter returns a RequestCounter that delegates up to
// maxRequests at a time to the given handler, and responds 503 to all
// incoming requests beyond that limit.
func NewRequestLimiter(maxRequests int, handler http.Handler) RequestCounter {
	return &limiterHandler{
		requests: make(chan struct{}, maxRequests),
		handler:  handler,
	}
}

func (h *limiterHandler) Current() int {
	return len(h.requests)
}

func (h *limiterHandler) Max() int {
	return cap(h.requests)
}

func (h *limiterHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	select {
	case h.requests <- struct{}{}:
	default:
		// reached max requests
		resp.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	h.handler.ServeHTTP(resp, req)
	<-h.requests
}
