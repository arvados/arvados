package main

import (
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

func (rtr *router) makeServer(newSession func(wsConn, arvados.Client) (session, error)) *websocket.Server {
	handler := &handler{
		Client:      rtr.Config.Client,
		PingTimeout: rtr.Config.PingTimeout.Duration(),
		QueueSize:   rtr.Config.ClientEventQueue,
		NewSession:  newSession,
	}
	return &websocket.Server{
		Handshake: func(c *websocket.Config, r *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			sink := rtr.eventSource.NewSink()
			handler.Handle(ws, sink.Channel())
			sink.Stop()
			ws.Close()
		}),
	}
}

func (rtr *router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rtr.setupOnce.Do(rtr.setup)
	t0 := time.Now()
	reqLog(map[string]interface{}{
		"Connect":         req.RemoteAddr,
		"RemoteAddr":      req.RemoteAddr,
		"X-Forwarded-For": req.Header.Get("X-Forwarded-For"),
		"Time":            t0.UTC(),
	})
	rtr.mux.ServeHTTP(resp, req)
	t1 := time.Now()
	reqLog(map[string]interface{}{
		"Disconnect":      req.RemoteAddr,
		"RemoteAddr":      req.RemoteAddr,
		"X-Forwarded-For": req.Header.Get("X-Forwarded-For"),
		"Time":            t1.UTC(),
		"Elapsed":         time.Now().Sub(t0).Seconds(),
	})
}

func reqLog(m map[string]interface{}) {
	j, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(string(j))
}
