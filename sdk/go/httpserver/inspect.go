// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"encoding/json"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Inspect serves a report of current requests at "GET
// /_inspect/requests", and passes other requests through to the next
// handler.
//
// If registry is not nil, Inspect registers metrics about current
// requests.
func Inspect(registry *prometheus.Registry, authToken string, next http.Handler) http.Handler {
	type ent struct {
		startTime  time.Time
		hangupTime atomic.Value
	}
	current := map[*http.Request]*ent{}
	mtx := sync.Mutex{}
	if registry != nil {
		registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "max_active_request_age_seconds",
				Help:      "Age of oldest active request",
			},
			func() float64 {
				mtx.Lock()
				defer mtx.Unlock()
				earliest := time.Time{}
				any := false
				for _, e := range current {
					if _, ok := e.hangupTime.Load().(time.Time); ok {
						// Don't count abandoned requests here
						continue
					}
					if !any || e.startTime.Before(earliest) {
						any = true
						earliest = e.startTime
					}
				}
				if !any {
					return 0
				}
				return float64(time.Since(earliest).Seconds())
			},
		))
		registry.MustRegister(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      "max_abandoned_request_age_seconds",
				Help:      "Maximum time since client hung up on a request whose processing thread is still running",
			},
			func() float64 {
				mtx.Lock()
				defer mtx.Unlock()
				earliest := time.Time{}
				any := false
				for _, e := range current {
					if hangupTime, ok := e.hangupTime.Load().(time.Time); ok {
						if !any || hangupTime.Before(earliest) {
							any = true
							earliest = hangupTime
						}
					}
				}
				if !any {
					return 0
				}
				return float64(time.Since(earliest).Seconds())
			},
		))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method == "GET" && req.URL.Path == "/_inspect/requests" {
			if authToken == "" || req.Header.Get("Authorization") != "Bearer "+authToken {
				Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			mtx.Lock()
			defer mtx.Unlock()
			type outrec struct {
				RequestID  string
				Method     string
				Host       string
				URL        string
				RemoteAddr string
				Elapsed    float64
			}
			now := time.Now()
			outrecs := []outrec{}
			for req, e := range current {
				outrecs = append(outrecs, outrec{
					RequestID:  req.Header.Get(HeaderRequestID),
					Method:     req.Method,
					Host:       req.Host,
					URL:        req.URL.String(),
					RemoteAddr: req.RemoteAddr,
					Elapsed:    now.Sub(e.startTime).Seconds(),
				})
			}
			sort.Slice(outrecs, func(i, j int) bool { return outrecs[i].Elapsed < outrecs[j].Elapsed })
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(outrecs)
		} else {
			e := ent{startTime: time.Now()}
			mtx.Lock()
			current[req] = &e
			mtx.Unlock()
			go func() {
				<-req.Context().Done()
				e.hangupTime.Store(time.Now())
			}()
			defer func() {
				mtx.Lock()
				defer mtx.Unlock()
				delete(current, req)
			}()
			next.ServeHTTP(w, req)
		}
	})
}
