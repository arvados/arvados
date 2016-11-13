package main

import (
	"io"
	"log"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

type router struct {
	EventSource <-chan event
	mux         *http.ServeMux
	setupOnce   sync.Once
}

func (rtr *router) setup() {
	rtr.mux = http.NewServeMux()
	rtr.mux.Handle("/websocket", makeServer(handlerV0))
	rtr.mux.Handle("/arvados/v1/events.ws", makeServer(handlerV1))
}

func makeServer(handler func(io.ReadWriter)) websocket.Server {
	return websocket.Server{
		Handshake: func(c *websocket.Config, r *http.Request) error {
			return nil
		},
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			log.Printf("socket request: %+v", ws.Request())
			handler(ws)
			ws.Close()
			log.Printf("socket disconnect: %+v", ws.Request().RemoteAddr)
		}),
	}
}

func (rtr *router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	rtr.setupOnce.Do(rtr.setup)
	rtr.mux.ServeHTTP(resp, req)
}
