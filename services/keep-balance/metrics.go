// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type observer interface{ Observe(float64) }
type setter interface{ Set(float64) }

type metrics struct {
	reg         *prometheus.Registry
	statsGauges map[string]setter
	observers   map[string]observer
	setupOnce   sync.Once
	mtx         sync.Mutex
}

func newMetrics(registry *prometheus.Registry) *metrics {
	return &metrics{
		reg:         registry,
		statsGauges: map[string]setter{},
		observers:   map[string]observer{},
	}
}

func (m *metrics) DurationObserver(name, help string) observer {
	m.mtx.Lock()
	defer m.mtx.Unlock()
	if obs, ok := m.observers[name]; ok {
		return obs
	}
	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace: "arvados",
		Name:      name,
		Subsystem: "keepbalance",
		Help:      help,
	})
	m.reg.MustRegister(summary)
	m.observers[name] = summary
	return summary
}

// UpdateStats updates prometheus metrics using the given
// balancerStats. It creates and registers the needed gauges on its
// first invocation.
func (m *metrics) UpdateStats(s balancerStats) {
	type gauge struct {
		Value interface{}
		Help  string
	}
	s2g := map[string]gauge{
		"total":             {s.current, "current backend storage usage"},
		"garbage":           {s.garbage, "garbage (unreferenced, old)"},
		"transient":         {s.unref, "transient (unreferenced, new)"},
		"overreplicated":    {s.overrep, "overreplicated"},
		"underreplicated":   {s.underrep, "underreplicated"},
		"lost":              {s.lost, "lost"},
		"dedup_byte_ratio":  {s.dedupByteRatio(), "deduplication ratio, bytes referenced / bytes stored"},
		"dedup_block_ratio": {s.dedupBlockRatio(), "deduplication ratio, blocks referenced / blocks stored"},
	}
	m.setupOnce.Do(func() {
		// Register gauge(s) for each balancerStats field.
		addGauge := func(name, help string) {
			g := prometheus.NewGauge(prometheus.GaugeOpts{
				Namespace: "arvados",
				Name:      name,
				Subsystem: "keep",
				Help:      help,
			})
			m.reg.MustRegister(g)
			m.statsGauges[name] = g
		}
		for name, gauge := range s2g {
			switch gauge.Value.(type) {
			case blocksNBytes:
				for _, sub := range []string{"blocks", "bytes", "replicas"} {
					addGauge(name+"_"+sub, sub+" of "+gauge.Help)
				}
			case int, int64, float64:
				addGauge(name, gauge.Help)
			default:
				panic(fmt.Sprintf("bad gauge type %T", gauge.Value))
			}
		}
	})
	// Set gauges to values from s.
	for name, gauge := range s2g {
		switch val := gauge.Value.(type) {
		case blocksNBytes:
			m.statsGauges[name+"_blocks"].Set(float64(val.blocks))
			m.statsGauges[name+"_bytes"].Set(float64(val.bytes))
			m.statsGauges[name+"_replicas"].Set(float64(val.replicas))
		case int:
			m.statsGauges[name].Set(float64(val))
		case int64:
			m.statsGauges[name].Set(float64(val))
		case float64:
			m.statsGauges[name].Set(float64(val))
		default:
			panic(fmt.Sprintf("bad gauge type %T", gauge.Value))
		}
	}
}

func (m *metrics) Handler(log promhttp.Logger) http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{
		ErrorLog: log,
	})
}
