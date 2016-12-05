package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

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
	Config         *Config
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

type DebugStatuser interface {
	DebugStatus() interface{}
}

type sessionFactory func(wsConn, chan<- interface{}, *sql.DB, permChecker) (session, error)

func (rtr *router) setup() {
	rtr.handler = &handler{
		PingTimeout: rtr.Config.PingTimeout.Duration(),
		QueueSize:   rtr.Config.ClientEventQueue,
	}
	rtr.mux = http.NewServeMux()
	rtr.mux.Handle("/websocket", rtr.makeServer(NewSessionV0))
	rtr.mux.Handle("/arvados/v1/events.ws", rtr.makeServer(NewSessionV1))
	rtr.mux.HandleFunc("/debug.json", jsonHandler(rtr.DebugStatus))
	rtr.mux.HandleFunc("/status.json", jsonHandler(rtr.Status))
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
					return newSession(ws, sendq, rtr.eventSource.DB(), rtr.newPermChecker())
				})

			log.WithFields(logrus.Fields{
				"Elapsed": time.Now().Sub(t0).Seconds(),
				"Stats":   stats,
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
	if es, ok := rtr.eventSource.(DebugStatuser); ok {
		s["EventSource"] = es.DebugStatus()
	}
	return s
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
	ctx := contextWithLogger(req.Context(), logger)
	req = req.WithContext(ctx)
	logger.WithFields(logrus.Fields{
		"RemoteAddr":      req.RemoteAddr,
		"X-Forwarded-For": req.Header.Get("X-Forwarded-For"),
	}).Info("accept request")
	rtr.mux.ServeHTTP(resp, req)
}

func jsonHandler(fn func() interface{}) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		logger := logger(req.Context())
		enc := json.NewEncoder(resp)
		err := enc.Encode(fn())
		if err != nil {
			msg := "encode failed"
			logger.WithError(err).Error(msg)
			http.Error(resp, msg, http.StatusInternalServerError)
		}
	}
}
