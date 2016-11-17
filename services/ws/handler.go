package main

import (
	"io"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type handler struct {
	Client      arvados.Client
	PingTimeout time.Duration
	QueueSize   int
	NewSession  func(wsConn, chan<- interface{}) (session, error)
}

type handlerStats struct {
	QueueDelayNs time.Duration
	WriteDelayNs time.Duration
	EventBytes   uint64
	EventCount   uint64
}

func (h *handler) Handle(ws wsConn, incoming <-chan *event) (stats handlerStats) {
	log := logger(ws.Request().Context())
	queue := make(chan interface{}, h.QueueSize)
	sess, err := h.NewSession(ws, queue)
	if err != nil {
		log.WithError(err).Error("NewSession failed")
		return
	}

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
			buf := buf[:n]
			log.WithField("frame", string(buf[:n])).Debug("received frame")
			if err == nil && n == cap(buf) {
				err = errFrameTooBig
			}
			if err != nil {
				if err != io.EOF {
					log.WithError(err).Info("read error")
				}
				stop <- err
				return
			}
			err = sess.Receive(buf)
			if err != nil {
				stop <- err
				return
			}
		}
	}()

	go func() {
		for data := range queue {
			var e *event
			var buf []byte
			var err error
			log := log

			switch data := data.(type) {
			case []byte:
				buf = data
			case *event:
				e = data
				log = log.WithField("serial", e.Serial)
				buf, err = sess.EventMessage(e)
				if err != nil {
					log.WithError(err).Error("EventMessage failed")
					stop <- err
					break
				} else if len(buf) == 0 {
					log.Debug("skip")
					continue
				}
			default:
				log.WithField("data", data).Error("bad object in client queue")
				continue
			}

			log.WithField("frame", string(buf)).Debug("send event")
			ws.SetWriteDeadline(time.Now().Add(h.PingTimeout))
			t0 := time.Now()
			_, err = ws.Write(buf)
			if err != nil {
				log.WithError(err).Error("write failed")
				stop <- err
				break
			}
			log.Debug("sent")

			if e != nil {
				stats.QueueDelayNs += t0.Sub(e.Received)
			}
			stats.WriteDelayNs += time.Since(t0)
			stats.EventBytes += uint64(len(buf))
			stats.EventCount++
		}
		for _ = range queue {
			// Ensure queue can't fill up and block other
			// goroutines after we hit a write error.
		}
	}()

	// Filter incoming events against the current subscription
	// list, and forward matching events to the outgoing message
	// queue. Close the queue and return when the "stopped"
	// channel closes or the incoming event stream ends. Shut down
	// the handler if the outgoing queue fills up.
	go func() {
		ticker := time.NewTicker(h.PingTimeout)
		defer ticker.Stop()

		for {
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
			case e, ok := <-incoming:
				if !ok {
					close(queue)
					return
				}
				if !sess.Filter(e) {
					continue
				}
				select {
				case queue <- e:
				default:
					stop <- errQueueFull
				}
			}
		}
	}()

	<-stop
	close(stopped)

	return
}
