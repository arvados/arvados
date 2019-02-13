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
	errors      prometheus.Counter
	inBytes     prometheus.Counter
	outBytes    prometheus.Counter
	errCounters *prometheus.CounterVec

	ErrorCodes map[string]uint64 `json:",omitempty"`
	lock       sync.Mutex
}

func (s *statsTicker) setup(m *volumeMetrics) {
	s.errors = m.Errors
	s.errCounters = m.ErrorCodes
	s.inBytes = m.InBytes
	s.outBytes = m.OutBytes
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
	s.errors.Inc()
	s.Tick(&s.Errors)

	s.lock.Lock()
	if s.ErrorCodes == nil {
		s.ErrorCodes = make(map[string]uint64)
	}
	s.ErrorCodes[errType]++
	s.lock.Unlock()
	s.errCounters.WithLabelValues(errType).Inc()
}

// TickInBytes increments the incoming byte counter by n.
func (s *statsTicker) TickInBytes(n uint64) {
	s.inBytes.Add(float64(n))
	atomic.AddUint64(&s.InBytes, n)
}

// TickOutBytes increments the outgoing byte counter by n.
func (s *statsTicker) TickOutBytes(n uint64) {
	s.outBytes.Add(float64(n))
	atomic.AddUint64(&s.OutBytes, n)
}
