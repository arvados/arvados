package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

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
			log.Printf("%v accepted", ws.Request().RemoteAddr)
			sink := rtr.eventSource.NewSink()
			handler.Handle(ws, sink.Channel())
			sink.Stop()
			ws.Close()
			log.Printf("%v disconnected", ws.Request().RemoteAddr)
		}),
	}
}

func (rtr *router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rtr.setupOnce.Do(rtr.setup)
	rtr.mux.ServeHTTP(resp, req)
	j, err := json.Marshal(map[string]interface{}{
		"req": fmt.Sprintf("%+v", req),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Print(string(j))
}
