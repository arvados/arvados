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
	rc  httpserver.RequestCounter
}

func (m *nodeMetrics) setup() {
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "bufferpool_bytes_allocated",
			Help:      "Number of bytes allocated to buffers",
		},
		func() float64 { return float64(bufs.Alloc()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "bufferpool_buffers_max",
			Help:      "Maximum number of buffers allowed",
		},
		func() float64 { return float64(bufs.Cap()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "bufferpool_buffers_in_use",
			Help:      "Number of buffers in use",
		},
		func() float64 { return float64(bufs.Len()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "pull_queue_in_progress",
			Help:      "Number of pull requests in progress",
		},
		func() float64 { return float64(getWorkQueueStatus(pullq).InProgress) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "pull_queue_queued",
			Help:      "Number of queued pull requests",
		},
		func() float64 { return float64(getWorkQueueStatus(pullq).Queued) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "trash_queue_in_progress",
			Help:      "Number of trash requests in progress",
		},
		func() float64 { return float64(getWorkQueueStatus(trashq).InProgress) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "trash_queue_queued",
			Help:      "Number of queued trash requests",
		},
		func() float64 { return float64(getWorkQueueStatus(trashq).Queued) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "requests_current",
			Help:      "Number of requests in progress",
		},
		func() float64 { return float64(m.rc.Current()) },
	))
	m.reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "keepstore",
			Name:      "requests_max",
			Help:      "Maximum number of concurrent requests",
		},
		func() float64 { return float64(m.rc.Max()) },
	))
	// Register individual volume's metrics
	vols := KeepVM.AllReadable()
	for _, vol := range vols {
		labels := prometheus.Labels{
			"label":         vol.String(),
			"mount_point":   vol.Status().MountPoint,
			"device_number": fmt.Sprintf("%d", vol.Status().DeviceNum),
		}
		if vol, ok := vol.(InternalMetricser); ok {
			// Per-driver internal metrics
			vol.SetupInternalMetrics(m.reg, labels)
		}
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_bytes_free",
				Help:        "Number of free bytes on the volume",
				ConstLabels: labels,
			},
			func() float64 { return float64(vol.Status().BytesFree) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_bytes_used",
				Help:        "Number of used bytes on the volume",
				ConstLabels: labels,
			},
			func() float64 { return float64(vol.Status().BytesUsed) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_io_errors",
				Help:        "Number of I/O errors",
				ConstLabels: labels,
			},
			func() float64 { return float64(KeepVM.VolumeStats(vol).Errors) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_io_ops",
				Help:        "Number of I/O operations",
				ConstLabels: labels,
			},
			func() float64 { return float64(KeepVM.VolumeStats(vol).Ops) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_io_compare_ops",
				Help:        "Number of I/O compare operations",
				ConstLabels: labels,
			},
			func() float64 { return float64(KeepVM.VolumeStats(vol).CompareOps) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_io_get_ops",
				Help:        "Number of I/O get operations",
				ConstLabels: labels,
			},
			func() float64 { return float64(KeepVM.VolumeStats(vol).GetOps) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_io_put_ops",
				Help:        "Number of I/O put operations",
				ConstLabels: labels,
			},
			func() float64 { return float64(KeepVM.VolumeStats(vol).PutOps) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_io_touch_ops",
				Help:        "Number of I/O touch operations",
				ConstLabels: labels,
			},
			func() float64 { return float64(KeepVM.VolumeStats(vol).TouchOps) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_io_input_bytes",
				Help:        "Number of input bytes",
				ConstLabels: labels,
			},
			func() float64 { return float64(KeepVM.VolumeStats(vol).InBytes) },
		))
		m.reg.Register(prometheus.NewGaugeFunc(
			prometheus.GaugeOpts{
				Namespace:   "arvados",
				Subsystem:   "keepstore",
				Name:        "volume_io_output_bytes",
				Help:        "Number of output bytes",
				ConstLabels: labels,
			},
			func() float64 { return float64(KeepVM.VolumeStats(vol).OutBytes) },
		))
	}
}
