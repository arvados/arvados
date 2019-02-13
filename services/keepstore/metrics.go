// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
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
			Name:      "bufferpool_bytes_allocated",
			Help:      "Number of bytes allocated to buffers",
		},
		func() float64 { return float64(b.Alloc()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "bufferpool_buffers_max",
			Help:      "Maximum number of buffers allowed",
		},
		func() float64 { return float64(b.Cap()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "bufferpool_buffers_in_use",
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
			Name:      fmt.Sprintf("%s_queue_in_progress", qName),
			Help:      fmt.Sprintf("Number of %s requests in progress", qName),
		},
		func() float64 { return float64(getWorkQueueStatus(q).InProgress) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      fmt.Sprintf("%s_queue_queued", qName),
			Help:      fmt.Sprintf("Number of queued %s requests", qName),
		},
		func() float64 { return float64(getWorkQueueStatus(q).Queued) },
	))
}

func (m *nodeMetrics) setupRequestMetrics(rc httpserver.RequestCounter) {
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "requests_current",
			Help:      "Number of requests in progress",
		},
		func() float64 { return float64(rc.Current()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "requests_max",
			Help:      "Maximum number of concurrent requests",
		},
		func() float64 { return float64(rc.Max()) },
	))
}

type volumeMetricsVecs struct {
	BytesFree  *prometheus.GaugeVec
	BytesUsed  *prometheus.GaugeVec
	Errors     *prometheus.CounterVec
	Ops        *prometheus.CounterVec
	CompareOps *prometheus.CounterVec
	GetOps     *prometheus.CounterVec
	PutOps     *prometheus.CounterVec
	TouchOps   *prometheus.CounterVec
	InBytes    *prometheus.CounterVec
	OutBytes   *prometheus.CounterVec
	ErrorCodes *prometheus.CounterVec
}

type volumeMetrics struct {
	BytesFree  prometheus.Gauge
	BytesUsed  prometheus.Gauge
	Errors     prometheus.Counter
	Ops        prometheus.Counter
	CompareOps prometheus.Counter
	GetOps     prometheus.Counter
	PutOps     prometheus.Counter
	TouchOps   prometheus.Counter
	InBytes    prometheus.Counter
	OutBytes   prometheus.Counter
	ErrorCodes *prometheus.CounterVec
}

func newVolumeMetricsVecs(reg *prometheus.Registry) *volumeMetricsVecs {
	m := &volumeMetricsVecs{}
	m.BytesFree = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_bytes_free",
			Help:      "Number of free bytes on the volume",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.BytesFree)
	m.BytesUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_bytes_used",
			Help:      "Number of used bytes on the volume",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.BytesUsed)
	m.Errors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_errors",
			Help:      "Number of volume I/O errors",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.Errors)
	m.Ops = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_ops",
			Help:      "Number of volume I/O operations",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.Ops)
	m.CompareOps = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_compare_ops",
			Help:      "Number of volume I/O compare operations",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.CompareOps)
	m.GetOps = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_get_ops",
			Help:      "Number of volume I/O get operations",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.GetOps)
	m.PutOps = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_put_ops",
			Help:      "Number of volume I/O put operations",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.PutOps)
	m.TouchOps = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_touch_ops",
			Help:      "Number of volume I/O touch operations",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.TouchOps)
	m.InBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_in_bytes",
			Help:      "Number of input bytes",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.InBytes)
	m.OutBytes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_out_bytes",
			Help:      "Number of output bytes",
		},
		[]string{"label", "mount_point", "device_number"},
	)
	reg.MustRegister(m.OutBytes)
	m.ErrorCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "volume_io_error_codes",
			Help:      "Number of I/O errors by error code",
		},
		[]string{"label", "mount_point", "device_number", "error_code"},
	)
	reg.MustRegister(m.ErrorCodes)

	return m
}

func (m *volumeMetricsVecs) curryWith(lbl string, mnt string, dev string) *volumeMetrics {
	lbls := []string{lbl, mnt, dev}
	curried := &volumeMetrics{
		BytesFree:  m.BytesFree.WithLabelValues(lbls...),
		BytesUsed:  m.BytesUsed.WithLabelValues(lbls...),
		Errors:     m.Errors.WithLabelValues(lbls...),
		Ops:        m.Ops.WithLabelValues(lbls...),
		CompareOps: m.CompareOps.WithLabelValues(lbls...),
		GetOps:     m.GetOps.WithLabelValues(lbls...),
		PutOps:     m.PutOps.WithLabelValues(lbls...),
		TouchOps:   m.TouchOps.WithLabelValues(lbls...),
		InBytes:    m.InBytes.WithLabelValues(lbls...),
		OutBytes:   m.OutBytes.WithLabelValues(lbls...),
		ErrorCodes: m.ErrorCodes.MustCurryWith(prometheus.Labels{
			"label":         lbl,
			"mount_point":   mnt,
			"device_number": dev,
		}),
	}
	return curried
}
