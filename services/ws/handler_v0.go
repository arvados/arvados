package main

import (
	"encoding/json"
	"log"
	"sync"
)

type handlerV0 struct {
	QueueSize int
}

func (h *handlerV0) debugLogf(ws wsConn, s string, args ...interface{}) {
	args = append([]interface{}{ws.Request().RemoteAddr}, args...)
	debugLogf("%s "+s, args...)
}

func (h *handlerV0) Handle(ws wsConn, events <-chan *event) {
	done := make(chan struct{}, 3)
	queue := make(chan *event, h.QueueSize)
	mtx := sync.Mutex{}
	subscribed := make(map[string]bool)
	go func() {
		buf := make([]byte, 2<<20)
		for {
			n, err := ws.Read(buf)
			h.debugLogf(ws, "received frame: %q", buf[:n])
			if err != nil || n == len(buf) {
				break
			}
			msg := make(map[string]interface{})
			err = json.Unmarshal(buf[:n], &msg)
			if err != nil {
				break
			}
			h.debugLogf(ws, "received message: %+v", msg)
			h.debugLogf(ws, "subscribing to *")
			subscribed["*"] = true
		}
		done <- struct{}{}
	}()
	go func(queue <-chan *event) {
		for e := range queue {
			detail := e.Detail(nil)
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
			if  err != nil {
				h.debugLogf(ws, "handlerV0: write: %s", err)
				break
			}
		}
		done <- struct{}{}
	}(queue)
	go func() {
		send := func(e *event) {
			if queue == nil {
				return
			}
			select {
			case queue <- e:
			default:
				close(queue)
				queue = nil
				done <- struct{}{}
			}
		}
		for e := range events {
			detail := e.Detail(nil)
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
		done <- struct{}{}
	}()
	<-done
}
