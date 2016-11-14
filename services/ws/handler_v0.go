package main

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"
)

var errQueueFull = errors.New("client queue full")

type handlerV0 struct {
	QueueSize int
}

func (h *handlerV0) debugLogf(ws wsConn, s string, args ...interface{}) {
	args = append([]interface{}{ws.Request().RemoteAddr}, args...)
	debugLogf("%s "+s, args...)
}

func (h *handlerV0) Handle(ws wsConn, events <-chan *event) {
	queue := make(chan *event, h.QueueSize)
	mtx := sync.Mutex{}
	subscribed := make(map[string]bool)

	stopped := make(chan struct{})
	stop := make(chan error, 5)

	go func() {
		buf := make([]byte, 2<<20)
		for {
			n, err := ws.Read(buf)
			h.debugLogf(ws, "received frame: %q", buf[:n])
			if err != nil || n == len(buf) {
				h.debugLogf(ws, "handlerV0: read: %s", err)
				stop <- err
				return
			}
			msg := make(map[string]interface{})
			err = json.Unmarshal(buf[:n], &msg)
			if err != nil {
				h.debugLogf(ws, "handlerV0: unmarshal: %s", err)
				stop <- err
				return
			}
			h.debugLogf(ws, "received message: %+v", msg)
			h.debugLogf(ws, "subscribing to *")
			subscribed["*"] = true
		}
	}()

	go func() {
		for e := range queue {
			if e == nil {
				ws.Write([]byte("{}\n"))
				continue
			}
			detail := e.Detail()
			if detail == nil {
				continue
			}
			// FIXME: check permission
			buf, err := json.Marshal(map[string]interface{}{
				"msgID":             e.Serial,
				"id":                detail.ID,
				"uuid":              detail.UUID,
				"object_uuid":       detail.ObjectUUID,
				"object_owner_uuid": detail.ObjectOwnerUUID,
				"event_type":        detail.EventType,
			})
			if err != nil {
				log.Printf("error encoding: ", err)
				continue
			}
			_, err = ws.Write(append(buf, byte('\n')))
			if err != nil {
				h.debugLogf(ws, "handlerV0: write: %s", err)
				stop <- err
				return
			}
		}
	}()

	// Filter incoming events against the current subscription
	// list, and forward matching events to the outgoing message
	// queue. Close the queue and return when the "stopped"
	// channel closes or the incoming event stream ends. Shut down
	// the handler if the outgoing queue fills up.
	go func() {
		send := func(e *event) {
			select {
			case queue <- e:
			default:
				stop <- errQueueFull
			}
		}

		// Once a minute, if the queue is empty, send an empty
		// message. This can help detect a disconnected
		// network socket.
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			var e *event
			select {
			case <-stopped:
				close(queue)
				return
			case <-ticker.C:
				if len(queue) == 0 {
					send(nil)
				}
				continue
			case e = <-events:
			}
			detail := e.Detail()
			mtx.Lock()
			switch {
			case subscribed["*"]:
				send(e)
			case detail == nil:
			case subscribed[detail.ObjectUUID]:
				send(e)
			case subscribed[detail.ObjectOwnerUUID]:
				send(e)
			}
			mtx.Unlock()
		}
	}()

	<-stop
	close(stopped)
}
