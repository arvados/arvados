// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"github.com/Sirupsen/logrus"
	"golang.org/x/net/websocket"
)

type wsConn interface {
	io.ReadWriter
	Request() *http.Request
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type router struct {
	Config         *wsConfig
	eventSource    eventSource
	newPermChecker func() permChecker

	handler   *handler
	mux       *http.ServeMux
	setupOnce sync.Once

	lastReqID  int64
	lastReqMtx sync.Mutex

	status routerDebugStatus
}

type routerDebugStatus struct {
	ReqsReceived int64
	ReqsActive   int64
}

type debugStatuser interface {
	DebugStatus() interface{}
}

func (rtr *router) setup() {
	rtr.handler = &handler{
		PingTimeout: rtr.Config.PingTimeout.Duration(),
		QueueSize:   rtr.Config.ClientEventQueue,
	}
	rtr.mux = http.NewServeMux()
	rtr.mux.Handle("/websocket", rtr.makeServer(newSessionV0))
	rtr.mux.Handle("/arvados/v1/events.ws", rtr.makeServer(newSessionV1))
	rtr.mux.Handle("/debug.json", rtr.jsonHandler(rtr.DebugStatus))
	rtr.mux.Handle("/status.json", rtr.jsonHandler(rtr.Status))

	rtr.mux.Handle("/_health/", &health.Handler{
		Token:  rtr.Config.ManagementToken,
		Prefix: "/_health/",
		Routes: health.Routes{
			"db": rtr.eventSource.DBHealth,
		},
		Log: func(r *http.Request, err error) {
			if err != nil {
				logger(r.Context()).WithError(err).Error("error")
			}
		},
	})
}

func (rtr *router) makeServer(newSession sessionFactory) *websocket.Server {
	return &websocket.Server{
		Handshake: func(c *websocket.Config, r *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			t0 := time.Now()
			log := logger(ws.Request().Context())
			log.Info("connected")

			stats := rtr.handler.Handle(ws, rtr.eventSource,
				func(ws wsConn, sendq chan<- interface{}) (session, error) {
					return newSession(ws, sendq, rtr.eventSource.DB(), rtr.newPermChecker(), &rtr.Config.Client)
				})

			log.WithFields(logrus.Fields{
				"elapsed": time.Now().Sub(t0).Seconds(),
				"stats":   stats,
			}).Info("disconnect")
			ws.Close()
		}),
	}
}

func (rtr *router) newReqID() string {
	rtr.lastReqMtx.Lock()
	defer rtr.lastReqMtx.Unlock()
	id := time.Now().UnixNano()
	if id <= rtr.lastReqID {
		id = rtr.lastReqID + 1
	}
	return strconv.FormatInt(id, 36)
}

func (rtr *router) DebugStatus() interface{} {
	s := map[string]interface{}{
		"HTTP":     rtr.status,
		"Outgoing": rtr.handler.DebugStatus(),
	}
	if es, ok := rtr.eventSource.(debugStatuser); ok {
		s["EventSource"] = es.DebugStatus()
	}
	return s
}

func (rtr *router) Status() interface{} {
	return map[string]interface{}{
		"Clients": atomic.LoadInt64(&rtr.status.ReqsActive),
		"Version": version,
	}
}

func (rtr *router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rtr.setupOnce.Do(rtr.setup)
	atomic.AddInt64(&rtr.status.ReqsReceived, 1)
	atomic.AddInt64(&rtr.status.ReqsActive, 1)
	defer atomic.AddInt64(&rtr.status.ReqsActive, -1)

	logger := logger(req.Context()).
		WithField("RequestID", rtr.newReqID())
	ctx := ctxlog.Context(req.Context(), logger)
	req = req.WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"remoteAddr":      req.RemoteAddr,
		"reqForwardedFor": req.Header.Get("X-Forwarded-For"),
	}).Info("accept request")
	rtr.mux.ServeHTTP(resp, req)
}

func (rtr *router) jsonHandler(fn func() interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger(r.Context())
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		err := enc.Encode(fn())
		if err != nil {
			msg := "encode failed"
			logger.WithError(err).Error(msg)
			http.Error(w, msg, http.StatusInternalServerError)
		}
	})
}
