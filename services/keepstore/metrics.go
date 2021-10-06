// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

type nodeMetrics struct {
	reg *prometheus.Registry
}

func (m *nodeMetrics) setupBufferPoolMetrics(b *bufferPool) {
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "bufferpool_allocated_bytes",
			Help:      "Number of bytes allocated to buffers",
		},
		func() float64 { return float64(b.Alloc()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "bufferpool_max_buffers",
			Help:      "Maximum number of buffers allowed",
		},
		func() float64 { return float64(b.Cap()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "bufferpool_inuse_buffers",
			Help:      "Number of buffers in use",
		},
		func() float64 { return float64(b.Len()) },
	))
}

func (m *nodeMetrics) setupWorkQueueMetrics(q *WorkQueue, qName string) {
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      fmt.Sprintf("%s_queue_inprogress_entries", qName),
			Help:      fmt.Sprintf("Number of %s requests in progress", qName),
		},
		func() float64 { return float64(getWorkQueueStatus(q).InProgress) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      fmt.Sprintf("%s_queue_pending_entries", qName),
			Help:      fmt.Sprintf("Number of queued %s requests", qName),
		},
		func() float64 { return float64(getWorkQueueStatus(q).Queued) },
	))
}

type volumeMetricsVecs struct {
	ioBytes     *prometheus.CounterVec
	errCounters *prometheus.CounterVec
	opsCounters *prometheus.CounterVec
}

func newVolumeMetricsVecs(reg *prometheus.Registry) *volumeMetricsVecs {
	m := &volumeMetricsVecs{}
	m.opsCounters = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_operations",
			Help:      "Number of volume operations",
		},
		[]string{"device_id", "operation"},
	)
	reg.MustRegister(m.opsCounters)
	m.errCounters = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_errors",
			Help:      "Number of volume errors",
		},
		[]string{"device_id", "error_type"},
	)
	reg.MustRegister(m.errCounters)
	m.ioBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_bytes",
			Help:      "Volume I/O traffic in bytes",
		},
		[]string{"device_id", "direction"},
	)
	reg.MustRegister(m.ioBytes)

	return m
}

func (vm *volumeMetricsVecs) getCounterVecsFor(lbls prometheus.Labels) (opsCV, errCV, ioCV *prometheus.CounterVec) {
	opsCV = vm.opsCounters.MustCurryWith(lbls)
	errCV = vm.errCounters.MustCurryWith(lbls)
	ioCV = vm.ioBytes.MustCurryWith(lbls)
	return
}
