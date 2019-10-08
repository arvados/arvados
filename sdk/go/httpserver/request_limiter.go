// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"net/http"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
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
	count    int64 // only used if cap(requests)==0
}

// NewRequestLimiter returns a RequestCounter that delegates up to
// maxRequests at a time to the given handler, and responds 503 to all
// incoming requests beyond that limit.
//
// "concurrent_requests" and "max_concurrent_requests" metrics are
// registered with the given reg, if reg is not nil.
func NewRequestLimiter(maxRequests int, handler http.Handler, reg *prometheus.Registry) RequestCounter {
	h := &limiterHandler{
		requests: make(chan struct{}, maxRequests),
		handler:  handler,
	}
	if reg != nil {
		reg.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "concurrent_requests",
				Help:      "Number of requests in progress",
			},
			func() float64 { return float64(h.Current()) },
		))
		reg.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "max_concurrent_requests",
				Help:      "Maximum number of concurrent requests",
			},
			func() float64 { return float64(h.Max()) },
		))
	}
	return h
}

func (h *limiterHandler) Current() int {
	if cap(h.requests) == 0 {
		return int(atomic.LoadInt64(&h.count))
	}
	return len(h.requests)
}

func (h *limiterHandler) Max() int {
	return cap(h.requests)
}

func (h *limiterHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if cap(h.requests) == 0 {
		atomic.AddInt64(&h.count, 1)
		h.handler.ServeHTTP(resp, req)
		atomic.AddInt64(&h.count, -1)
	}
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
