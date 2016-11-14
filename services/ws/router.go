package main

import (
	"log"
	"net/http"
	"sync"

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
	rtr.mux.Handle("/websocket", rtr.makeServer(&handlerV0{
		QueueSize: rtr.Config.ClientEventQueue,
	}))
	rtr.mux.Handle("/arvados/v1/events.ws", rtr.makeServer(&handlerV1{
		QueueSize: rtr.Config.ClientEventQueue,
	}))
}

func (rtr *router) makeServer(handler handler) *websocket.Server {
	return &websocket.Server{
		Handshake: func(c *websocket.Config, r *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			log.Printf("socket request: %+v", ws.Request())
			sink := rtr.eventSource.NewSink(nil)
			handler.Handle(ws, sink.Channel())
			sink.Stop()
			ws.Close()
			log.Printf("socket disconnect: %+v", ws.Request().RemoteAddr)
		}),
	}
}

func (rtr *router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rtr.setupOnce.Do(rtr.setup)
	rtr.mux.ServeHTTP(resp, req)
}
