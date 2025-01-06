// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/websocket"
)

type wsConn interface {
	io.ReadWriter
	Request() *http.Request
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type router struct {
	client         *arvados.Client
	cluster        *arvados.Cluster
	eventSource    eventSource
	newPermChecker func() permChecker

	handler   *handler
	mux       *http.ServeMux
	setupOnce sync.Once
	done      chan struct{}
	reg       *prometheus.Registry
}

func (rtr *router) setup() {
	mSockets := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "ws",
		Name:      "sockets",
		Help:      "Number of connected sockets",
	}, []string{"version"})
	rtr.reg.MustRegister(mSockets)

	rtr.handler = &handler{
		PingTimeout: time.Duration(rtr.cluster.API.SendTimeout),
		QueueSize:   rtr.cluster.API.WebsocketClientEventQueue,
	}
	rtr.mux = http.NewServeMux()
	rtr.mux.Handle("/websocket", rtr.makeServer(newSessionV0, mSockets.WithLabelValues("0")))
	rtr.mux.Handle("/arvados/v1/events.ws", rtr.makeServer(newSessionV1, mSockets.WithLabelValues("1")))
	rtr.mux.Handle("/_health/", &health.Handler{
		Token:  rtr.cluster.ManagementToken,
		Prefix: "/_health/",
		Routes: health.Routes{
			"db": rtr.eventSource.DBHealth,
		},
		Log: func(r *http.Request, err error) {
			if err != nil {
				ctxlog.FromContext(r.Context()).WithError(err).Error("error")
			}
		},
	})
}

func exemptFromDeadline(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		httpserver.ExemptFromDeadline(req)
		h.ServeHTTP(w, req)
	})
}

func (rtr *router) makeServer(newSession sessionFactory, gauge prometheus.Gauge) http.Handler {
	var connected int64
	return exemptFromDeadline(&websocket.Server{
		Handshake: func(c *websocket.Config, r *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			t0 := time.Now()
			logger := ctxlog.FromContext(ws.Request().Context())
			atomic.AddInt64(&connected, 1)
			gauge.Set(float64(atomic.LoadInt64(&connected)))

			stats := rtr.handler.Handle(ws, logger, rtr.eventSource,
				func(ws wsConn, sendq chan<- interface{}) (session, error) {
					return newSession(ws, sendq, rtr.eventSource.DB(), rtr.newPermChecker(), rtr.client)
				})

			logger.WithFields(logrus.Fields{
				"elapsed": time.Now().Sub(t0).Seconds(),
				"stats":   stats,
			}).Info("client disconnected")
			ws.Close()
			atomic.AddInt64(&connected, -1)
			gauge.Set(float64(atomic.LoadInt64(&connected)))
		}),
	})
}

func (rtr *router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rtr.setupOnce.Do(rtr.setup)
	rtr.mux.ServeHTTP(httpserver.ResponseControllerShim{ResponseWriter: resp}, req)
}

func (rtr *router) CheckHealth() error {
	rtr.setupOnce.Do(rtr.setup)
	return rtr.eventSource.DBHealth()
}

func (rtr *router) Done() <-chan struct{} {
	return rtr.done
}
