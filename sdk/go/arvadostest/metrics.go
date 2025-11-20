// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadostest

import (
	"bytes"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/check.v1"
)

func GatherMetricsAsString(reg *prometheus.Registry) string {
	buf := bytes.NewBuffer(nil)
	enc := expfmt.NewEncoder(buf, expfmt.NewFormat(expfmt.TypeTextPlain))
	got, _ := reg.Gather()
	for _, mf := range got {
		enc.Encode(mf)
	}
	return buf.String()
}

// GetMetricValue returns the current value of the indicated metric.
// Metric parameter names and values are given in labels, as in:
//
//	GetMetricValue(c, reg, "arvados_metric_name", "label1", "value1", "label2", "value2")
func GetMetricValue(c *check.C, reg *prometheus.Registry, name string, labels ...string) float64 {
	gather, _ := reg.Gather()
	for _, mf := range gather {
		if mf.Name != nil && *mf.Name == name {
		metric:
			for _, m := range mf.Metric {
				if 2*len(m.Label) != len(labels) {
					continue metric
				}
				for i, lp := range m.Label {
					if lp.Name == nil ||
						*lp.Name != labels[i*2] ||
						lp.Value == nil ||
						*lp.Value != labels[i*2+1] {
						continue metric
					}
				}
				if m.GetCounter() != nil {
					return *m.GetCounter().Value
				}
				if m.GetGauge() != nil {
					return *m.GetGauge().Value
				}
				if m.GetUntyped() != nil {
					return *m.GetUntyped().Value
				}
				c.Fatalf("GetMetricValue: unsupported metric type: %s", m)
				return -1
			}
		}
	}
	c.Fatalf("metric not found: %s %v", name, labels)
	return -1
}
