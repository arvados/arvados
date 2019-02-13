// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

type statsTicker struct {
	Errors   uint64
	InBytes  uint64
	OutBytes uint64

	// Prometheus metrics
	PromErrors     prometheus.Counter
	PromInBytes    prometheus.Counter
	PromOutBytes   prometheus.Counter
	PromErrorCodes *prometheus.CounterVec

	ErrorCodes map[string]uint64 `json:",omitempty"`
	lock       sync.Mutex
}

// Tick increments each of the given counters by 1 using
// atomic.AddUint64.
func (s *statsTicker) Tick(counters ...*uint64) {
	for _, counter := range counters {
		atomic.AddUint64(counter, 1)
	}
}

// TickErr increments the overall error counter, as well as the
// ErrorCodes entry for the given errType. If err is nil, TickErr is a
// no-op.
func (s *statsTicker) TickErr(err error, errType string) {
	if err == nil {
		return
	}
	s.PromErrors.Inc()
	s.Tick(&s.Errors)

	s.lock.Lock()
	if s.ErrorCodes == nil {
		s.ErrorCodes = make(map[string]uint64)
	}
	s.ErrorCodes[errType]++
	s.lock.Unlock()
	s.PromErrorCodes.WithLabelValues(errType).Inc()
}

// TickInBytes increments the incoming byte counter by n.
func (s *statsTicker) TickInBytes(n uint64) {
	s.PromInBytes.Add(float64(n))
	atomic.AddUint64(&s.InBytes, n)
}

// TickOutBytes increments the outgoing byte counter by n.
func (s *statsTicker) TickOutBytes(n uint64) {
	s.PromOutBytes.Add(float64(n))
	atomic.AddUint64(&s.OutBytes, n)
}
