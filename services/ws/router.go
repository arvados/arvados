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
	rtr.mux.HandleFunc("/debug.json", jsonHandler(rtr.DebugStatus))
	rtr.mux.HandleFunc("/status.json", jsonHandler(rtr.Status))
	rtr.mux.HandleFunc("/_health/ping", jsonHandler(rtr.HealthFunc(func() error { return nil })))
	rtr.mux.HandleFunc("/_health/db", jsonHandler(rtr.HealthFunc(rtr.eventSource.DBHealth)))
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

var pingResponseOK = map[string]string{"health": "OK"}

func (rtr *router) HealthFunc(f func() error) func() interface{} {
	return func() interface{} {
		err := f()
		if err == nil {
			return pingResponseOK
		}
		return map[string]string{
			"health": "ERROR",
			"error":  err.Error(),
		}
	}
}

func (rtr *router) Status() interface{} {
	return map[string]interface{}{
		"Clients": atomic.LoadInt64(&rtr.status.ReqsActive),
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

func jsonHandler(fn func() interface{}) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		logger := logger(req.Context())
		resp.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(resp)
		err := enc.Encode(fn())
		if err != nil {
			msg := "encode failed"
			logger.WithError(err).Error(msg)
			http.Error(resp, msg, http.StatusInternalServerError)
		}
	}
}
