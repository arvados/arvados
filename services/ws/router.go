package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"golang.org/x/net/websocket"
)

type router struct {
	Config *Config

	eventSource eventSource
	mux         *http.ServeMux
	setupOnce   sync.Once
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
		NewSession:  func(ws wsConn) (session, error) {
			return newSession(ws, rtr.Config.Client, rtr.eventSource.DB())
		},
	}
	return &websocket.Server{
		Handshake: func(c *websocket.Config, r *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			logj("Type", "connect",
				"RemoteAddr", ws.Request().RemoteAddr)
			t0 := time.Now()

			sink := rtr.eventSource.NewSink()
			stats := handler.Handle(ws, sink.Channel())

			logj("Type", "disconnect",
				"RemoteAddr", ws.Request().RemoteAddr,
				"Elapsed", time.Now().Sub(t0).Seconds(),
				"Stats", stats)

			sink.Stop()
			ws.Close()
		}),
	}
}

func (rtr *router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rtr.setupOnce.Do(rtr.setup)
	logj("Type", "request",
		"RemoteAddr", req.RemoteAddr,
		"X-Forwarded-For", req.Header.Get("X-Forwarded-For"))
	rtr.mux.ServeHTTP(resp, req)
}

func reqLog(m map[string]interface{}) {
	j, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(string(j))
}
