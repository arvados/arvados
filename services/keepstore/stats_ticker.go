// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

type statsTicker struct {
	Errors   uint64
	InBytes  uint64
	OutBytes uint64

	ErrorCodes map[string]uint64 `json:",omitempty"`
	lock       sync.Mutex
}

func (s *statsTicker) setupPrometheus(drv string, reg *prometheus.Registry, lbl prometheus.Labels) {
	metrics := map[string][]interface{}{
		"errors":    []interface{}{string("errors"), s.Errors},
		"in_bytes":  []interface{}{string("input bytes"), s.InBytes},
		"out_bytes": []interface{}{string("output bytes"), s.OutBytes},
	}
	for mName, data := range metrics {
		mHelp := data[0].(string)
		mVal := data[1].(uint64)
		reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        fmt.Sprintf("%s_%s", drv, mName),
				Help:        fmt.Sprintf("Number of %s backend %s", drv, mHelp),
				ConstLabels: lbl,
			},
			func() float64 { return float64(mVal) },
		))
	}
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
	s.Tick(&s.Errors)

	s.lock.Lock()
	if s.ErrorCodes == nil {
		s.ErrorCodes = make(map[string]uint64)
	}
	s.ErrorCodes[errType]++
	s.lock.Unlock()
}

// TickInBytes increments the incoming byte counter by n.
func (s *statsTicker) TickInBytes(n uint64) {
	atomic.AddUint64(&s.InBytes, n)
}

// TickOutBytes increments the outgoing byte counter by n.
func (s *statsTicker) TickOutBytes(n uint64) {
	atomic.AddUint64(&s.OutBytes, n)
}
