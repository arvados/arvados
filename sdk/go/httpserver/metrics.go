// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/stats"
	"github.com/Sirupsen/logrus"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler interface {
	http.Handler

	// Returns an http.Handler that serves the Handler's metrics
	// data at /metrics and /metrics.json, and passes other
	// requests through to next.
	ServeAPI(next http.Handler) http.Handler
}

type metrics struct {
	next         http.Handler
	logger       *logrus.Logger
	registry     *prometheus.Registry
	reqDuration  *prometheus.SummaryVec
	timeToStatus *prometheus.SummaryVec
	exportProm   http.Handler
}

func (*metrics) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire implements logrus.Hook in order to collect data points from
// request logs.
func (m *metrics) Fire(ent *logrus.Entry) error {
	if tts, ok := ent.Data["timeToStatus"].(stats.Duration); !ok {
	} else if method, ok := ent.Data["reqMethod"].(string); !ok {
	} else if code, ok := ent.Data["respStatusCode"].(int); !ok {
	} else {
		m.timeToStatus.WithLabelValues(strconv.Itoa(code), strings.ToLower(method)).Observe(time.Duration(tts).Seconds())
	}
	return nil
}

func (m *metrics) exportJSON(w http.ResponseWriter, req *http.Request) {
	jm := jsonpb.Marshaler{Indent: "  "}
	mfs, _ := m.registry.Gather()
	w.Write([]byte{'['})
	for i, mf := range mfs {
		if i > 0 {
			w.Write([]byte{','})
		}
		jm.Marshal(w, mf)
	}
	w.Write([]byte{']'})
}

// ServeHTTP implements http.Handler.
func (m *metrics) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m.next.ServeHTTP(w, req)
}

// ServeAPI returns a new http.Handler that serves current data at
// metrics API endpoints (currently "GET /metrics(.json)?") and passes
// other requests through to next.
//
// Typical example:
//
//	m := Instrument(...)
//	srv := http.Server{Handler: m.ServeAPI(m)}
func (m *metrics) ServeAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch {
		case req.Method != "GET" && req.Method != "HEAD":
			next.ServeHTTP(w, req)
		case req.URL.Path == "/metrics.json":
			m.exportJSON(w, req)
		case req.URL.Path == "/metrics":
			m.exportProm.ServeHTTP(w, req)
		default:
			next.ServeHTTP(w, req)
		}
	})
}

// Instrument returns a new Handler that passes requests through to
// the next handler in the stack, and tracks metrics of those
// requests.
//
// For the metrics to be accurate, the caller must ensure every
// request passed to the Handler also passes through
// LogRequests(logger, ...), and vice versa.
//
// If registry is nil, a new registry is created.
//
// If logger is nil, logrus.StandardLogger() is used.
func Instrument(registry *prometheus.Registry, logger *logrus.Logger, next http.Handler) Handler {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	if registry == nil {
		registry = prometheus.NewRegistry()
	}
	reqDuration := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "request_duration_seconds",
		Help: "Summary of request duration.",
	}, []string{"code", "method"})
	timeToStatus := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "time_to_status_seconds",
		Help: "Summary of request TTFB.",
	}, []string{"code", "method"})
	registry.MustRegister(timeToStatus)
	registry.MustRegister(reqDuration)
	m := &metrics{
		next:         promhttp.InstrumentHandlerDuration(reqDuration, next),
		logger:       logger,
		registry:     registry,
		reqDuration:  reqDuration,
		timeToStatus: timeToStatus,
		exportProm: promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			ErrorLog: logger,
		}),
	}
	m.logger.AddHook(m)
	return m
}
