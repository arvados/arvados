package main

import (
	"encoding/json"
	"io"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	log "github.com/Sirupsen/logrus"
)

type handler struct {
	Client      arvados.Client
	PingTimeout time.Duration
	QueueSize   int
	NewSession  func(wsConn) (session, error)
}

type handlerStats struct {
	QueueDelayNs time.Duration
	WriteDelayNs time.Duration
	EventBytes   uint64
	EventCount   uint64
}

func (h *handler) Handle(ws wsConn, events <-chan *event) (stats handlerStats) {
	ctx := contextWithLogger(ws.Request().Context(), log.WithFields(log.Fields{
		"RemoteAddr": ws.Request().RemoteAddr,
	}))
	sess, err := h.NewSession(ws)
	if err != nil {
		logger(ctx).WithError(err).Error("NewSession failed")
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
			logger(ctx).WithField("frame", string(buf[:n])).Debug("received frame")
			if err == nil && n == len(buf) {
				err = errFrameTooBig
			}
			if err != nil {
				if err != io.EOF {
					logger(ctx).WithError(err).Info("read error")
				}
				stop <- err
				return
			}
			msg := make(map[string]interface{})
			err = json.Unmarshal(buf[:n], &msg)
			if err != nil {
				logger(ctx).WithError(err).Info("invalid json from client")
				stop <- err
				return
			}
			for _, buf := range sess.Receive(msg, buf[:n]) {
				logger(ctx).WithField("frame", string(buf)).Debug("queued message from sess.Receive")
				queue <- buf
			}
		}
	}()

	go func() {
		for e := range queue {
			if buf, ok := e.([]byte); ok {
				ws.SetWriteDeadline(time.Now().Add(h.PingTimeout))
				logger(ctx).WithField("frame", string(buf)).Debug("send msg buf")
				_, err := ws.Write(buf)
				if err != nil {
					logger(ctx).WithError(err).Error("write failed")
					stop <- err
					break
				}
				continue
			}
			e := e.(*event)
			log := logger(ctx).WithField("serial", e.Serial)

			buf, err := sess.EventMessage(e)
			if err != nil {
				log.WithError(err).Error("EventMessage failed")
				stop <- err
				break
			} else if len(buf) == 0 {
				log.Debug("skip")
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
