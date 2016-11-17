package main

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/websocket"
)

type wsConn interface {
	io.ReadWriter
	Request() *http.Request
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type router struct {
	Config *Config

	eventSource eventSource
	mux         *http.ServeMux
	setupOnce   sync.Once

	lastReqID  int64
	lastReqMtx sync.Mutex
}

func (rtr *router) setup() {
	rtr.mux = http.NewServeMux()
	rtr.mux.Handle("/websocket", rtr.makeServer(NewSessionV0))
	rtr.mux.Handle("/arvados/v1/events.ws", rtr.makeServer(NewSessionV1))
}

func (rtr *router) makeServer(newSession func(wsConn, arvados.Client, *sql.DB) (session, error)) *websocket.Server {
	handler := &handler{
		PingTimeout: rtr.Config.PingTimeout.Duration(),
		QueueSize:   rtr.Config.ClientEventQueue,
		NewSession: func(ws wsConn) (session, error) {
			return newSession(ws, rtr.Config.Client, rtr.eventSource.DB())
		},
	}
	return &websocket.Server{
		Handshake: func(c *websocket.Config, r *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			t0 := time.Now()
			sink := rtr.eventSource.NewSink()
			logger(ws.Request().Context()).Info("connected")

			stats := handler.Handle(ws, sink.Channel())

			logger(ws.Request().Context()).WithFields(log.Fields{
				"Elapsed": time.Now().Sub(t0).Seconds(),
				"Stats":   stats,
			}).Info("disconnect")

			sink.Stop()
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

func (rtr *router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rtr.setupOnce.Do(rtr.setup)
	logger := logger(req.Context()).
		WithField("RequestID", rtr.newReqID())
	ctx := contextWithLogger(req.Context(), logger)
	req = req.WithContext(ctx)
	logger.WithFields(log.Fields{
		"RemoteAddr":      req.RemoteAddr,
		"X-Forwarded-For": req.Header.Get("X-Forwarded-For"),
	}).Info("accept request")
	rtr.mux.ServeHTTP(resp, req)
}
