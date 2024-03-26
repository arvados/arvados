// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"github.com/prometheus/client_golang/prometheus"
)

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
