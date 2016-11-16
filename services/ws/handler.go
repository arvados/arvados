package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type wsConn interface {
	io.ReadWriter
	Request() *http.Request
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

type handler struct {
	Client      arvados.Client
	PingTimeout time.Duration
	QueueSize   int
	NewSession  func(wsConn) (session, error)
}

type handlerStats struct {
	QueueDelay time.Duration
	WriteDelay time.Duration
	EventBytes uint64
	EventCount uint64
}

func (h *handler) Handle(ws wsConn, events <-chan *event) (stats handlerStats) {
	sess, err := h.NewSession(ws)
	if err != nil {
		log.Printf("%s NewSession: %s", ws.Request().RemoteAddr, err)
		return
	}

	queue := make(chan interface{}, h.QueueSize)

	stopped := make(chan struct{})
	stop := make(chan error, 5)

	go func() {
		buf := make([]byte, 2<<20)
		for {
			select {
			case <-stopped:
				return
			default:
			}
			ws.SetReadDeadline(time.Now().Add(24 * 365 * time.Hour))
			n, err := ws.Read(buf)
			sess.debugLogf("received frame: %q", buf[:n])
			if err == nil && n == len(buf) {
				err = errFrameTooBig
			}
			if err != nil {
				if err != io.EOF {
					sess.debugLogf("handler: read: %s", err)
				}
				stop <- err
				return
			}
			msg := make(map[string]interface{})
			err = json.Unmarshal(buf[:n], &msg)
			if err != nil {
				sess.debugLogf("handler: unmarshal: %s", err)
				stop <- err
				return
			}
			for _, buf := range sess.Receive(msg, buf[:n]) {
				sess.debugLogf("handler: to queue: %s", string(buf))
				queue <- buf
			}
		}
	}()

	go func() {
		for e := range queue {
			if buf, ok := e.([]byte); ok {
				ws.SetWriteDeadline(time.Now().Add(h.PingTimeout))
				sess.debugLogf("handler: send msg: %s", string(buf))
				_, err := ws.Write(buf)
				if err != nil {
					sess.debugLogf("handler: write {}: %s", err)
					stop <- err
					break
				}
				continue
			}
			e := e.(*event)

			buf, err := sess.EventMessage(e)
			if err != nil {
				sess.debugLogf("EventMessage %d: err %s", err)
				stop <- err
				break
			} else if len(buf) == 0 {
				sess.debugLogf("EventMessage %d: skip", e.Serial)
				continue
			}

			sess.debugLogf("handler: send event %d: %q", e.Serial, buf)
			ws.SetWriteDeadline(time.Now().Add(h.PingTimeout))
			t0 := time.Now()
			_, err = ws.Write(buf)
			if err != nil {
				sess.debugLogf("handler: write: %s", err)
				stop <- err
				break
			}
			sess.debugLogf("handler: sent event %d", e.Serial)
			stats.WriteDelay += time.Since(t0)
			stats.QueueDelay += t0.Sub(e.Received)
			stats.EventBytes += uint64(len(buf))
			stats.EventCount++
		}
		for _ = range queue {
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

		ticker := time.NewTicker(h.PingTimeout)
		defer ticker.Stop()

		for {
			var e *event
			var ok bool
			select {
			case <-stopped:
				close(queue)
				return
			case <-ticker.C:
				// If the outgoing queue is empty,
				// send an empty message. This can
				// help detect a disconnected network
				// socket, and prevent an idle socket
				// from being closed.
				if len(queue) == 0 {
					queue <- []byte(`{}`)
				}
				continue
			case e, ok = <-events:
				if !ok {
					close(queue)
					return
				}
			}
			if sess.Filter(e) {
				send(e)
			}
		}
	}()

	<-stop
	close(stopped)

	return
}
