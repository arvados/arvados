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

func (m *metrics) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch {
	case req.Method != "GET" && req.Method != "HEAD":
		m.next.ServeHTTP(w, req)
	case req.URL.Path == "/metrics.json":
		m.exportJSON(w, req)
	case req.URL.Path == "/metrics":
		m.exportProm.ServeHTTP(w, req)
	default:
		m.next.ServeHTTP(w, req)
	}
}

func Instrument(logger *logrus.Logger, next http.Handler) Handler {
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	reqDuration := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "request_duration_seconds",
		Help: "Summary of request duration.",
	}, []string{"code", "method"})
	timeToStatus := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: "time_to_status_seconds",
		Help: "Summary of request TTFB.",
	}, []string{"code", "method"})
	registry := prometheus.NewRegistry()
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
