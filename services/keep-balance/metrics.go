// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type observer interface{ Observe(float64) }
type setter interface{ Set(float64) }

type metrics struct {
	reg            *prometheus.Registry
	statsGauges    map[string]setter
	statsGaugeVecs map[string]*prometheus.GaugeVec
	observers      map[string]observer
	setupOnce      sync.Once
	mtx            sync.Mutex
}

func newMetrics(registry *prometheus.Registry) *metrics {
	return &metrics{
		reg:            registry,
		statsGauges:    map[string]setter{},
		statsGaugeVecs: map[string]*prometheus.GaugeVec{},
		observers:      map[string]observer{},
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
		"unachievable":      {s.unachievable, "unachievable"},
		"balanced":          {s.justright, "optimally balanced"},
		"desired":           {s.desired, "desired"},
		"lost":              {s.lost, "lost"},
		"dedup_byte_ratio":  {s.dedupByteRatio(), "deduplication ratio, bytes referenced / bytes stored"},
		"dedup_block_ratio": {s.dedupBlockRatio(), "deduplication ratio, blocks referenced / blocks stored"},
		"collection_bytes":  {s.collectionBytes, "total apparent size of all collections"},
		"referenced_bytes":  {s.collectionBlockBytes, "total size of unique referenced blocks"},
		"reference_count":   {s.collectionBlockRefs, "block references in all collections"},
		"referenced_blocks": {s.collectionBlocks, "blocks referenced by any collection"},

		"pull_entries_sent_count":      {s.pulls, "total entries sent in pull lists"},
		"pull_entries_deferred_count":  {s.pullsDeferred, "total entries deferred (not sent) in pull lists"},
		"trash_entries_sent_count":     {s.trashes, "total entries sent in trash lists"},
		"trash_entries_deferred_count": {s.trashesDeferred, "total entries deferred (not sent) in trash lists"},

		"replicated_block_count": {s.replHistogram, "blocks with indicated number of replicas at last count"},
		"usage":                  {s.classStats, "stored in indicated storage class"},
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
			case []int:
				// replHistogram
				gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
					Namespace: "arvados",
					Name:      name,
					Subsystem: "keep",
					Help:      gauge.Help,
				}, []string{"replicas"})
				m.reg.MustRegister(gv)
				m.statsGaugeVecs[name] = gv
			case map[string]replicationStats:
				// classStats
				for _, sub := range []string{"blocks", "bytes", "replicas"} {
					name := name + "_" + sub
					gv := prometheus.NewGaugeVec(prometheus.GaugeOpts{
						Namespace: "arvados",
						Name:      name,
						Subsystem: "keep",
						Help:      gauge.Help,
					}, []string{"storage_class", "status"})
					m.reg.MustRegister(gv)
					m.statsGaugeVecs[name] = gv
				}
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
		case []int:
			// replHistogram
			for r, n := range val {
				m.statsGaugeVecs[name].WithLabelValues(strconv.Itoa(r)).Set(float64(n))
			}
			// Record zero for higher-than-max-replication
			// metrics, so we don't incorrectly continue
			// to report stale metrics.
			//
			// For example, if we previously reported n=1
			// for repl=6, but have since restarted
			// keep-balance and the most replicated block
			// now has repl=5, then the repl=6 gauge will
			// still say n=1 until we clear it explicitly
			// here.
			for r := len(val); r < len(val)+4 || r < len(val)*2; r++ {
				m.statsGaugeVecs[name].WithLabelValues(strconv.Itoa(r)).Set(0)
			}
		case map[string]replicationStats:
			// classStats
			for class, cs := range val {
				for label, val := range map[string]blocksNBytes{
					"needed":       cs.needed,
					"unneeded":     cs.unneeded,
					"pulling":      cs.pulling,
					"unachievable": cs.unachievable,
				} {
					m.statsGaugeVecs[name+"_blocks"].WithLabelValues(class, label).Set(float64(val.blocks))
					m.statsGaugeVecs[name+"_bytes"].WithLabelValues(class, label).Set(float64(val.bytes))
					m.statsGaugeVecs[name+"_replicas"].WithLabelValues(class, label).Set(float64(val.replicas))
				}
			}
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
