// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"context"
	"io"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/stats"
	"github.com/sirupsen/logrus"
)

type handler struct {
	Client      arvados.Client
	PingTimeout time.Duration
	QueueSize   int

	mtx       sync.Mutex
	lastDelay map[chan interface{}]stats.Duration
	setupOnce sync.Once
}

type handlerStats struct {
	QueueDelayNs time.Duration
	WriteDelayNs time.Duration
	EventBytes   uint64
	EventCount   uint64
}

func (h *handler) Handle(ws wsConn, logger logrus.FieldLogger, eventSource eventSource, newSession func(wsConn, chan<- interface{}) (session, error)) (hStats handlerStats) {
	h.setupOnce.Do(h.setup)

	ctx, cancel := context.WithCancel(ws.Request().Context())
	defer cancel()

	queue := make(chan interface{}, h.QueueSize)
	h.mtx.Lock()
	h.lastDelay[queue] = 0
	h.mtx.Unlock()
	defer func() {
		h.mtx.Lock()
		delete(h.lastDelay, queue)
		h.mtx.Unlock()
	}()

	sess, err := newSession(ws, queue)
	if err != nil {
		logger.WithError(err).Error("newSession failed")
		return
	}

	// Receive websocket frames from the client and pass them to
	// sess.Receive().
	go func() {
		defer cancel()
		buf := make([]byte, 2<<20)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			ws.SetReadDeadline(time.Now().Add(24 * 365 * time.Hour))
			n, err := ws.Read(buf)
			buf := buf[:n]
			logger.WithField("frame", string(buf[:n])).Debug("received frame")
			if err == nil && n == cap(buf) {
				err = errFrameTooBig
			}
			if err != nil {
				if err != io.EOF && ctx.Err() == nil {
					logger.WithError(err).Info("read error")
				}
				return
			}
			err = sess.Receive(buf)
			if err != nil {
				logger.WithError(err).Error("sess.Receive() failed")
				return
			}
		}
	}()

	// Take items from the outgoing queue, serialize them using
	// sess.EventMessage() as needed, and send them to the client
	// as websocket frames.
	go func() {
		defer cancel()
		for {
			var ok bool
			var data interface{}
			select {
			case <-ctx.Done():
				return
			case data, ok = <-queue:
				if !ok {
					return
				}
			}
			var e *event
			var buf []byte
			var err error
			logger := logger

			switch data := data.(type) {
			case []byte:
				buf = data
			case *event:
				e = data
				logger = logger.WithField("serial", e.Serial)
				buf, err = sess.EventMessage(e)
				if err != nil {
					logger.WithError(err).Error("EventMessage failed")
					return
				} else if len(buf) == 0 {
					logger.Debug("skip")
					continue
				}
			default:
				logger.WithField("data", data).Error("bad object in client queue")
				continue
			}

			logger.WithField("frame", string(buf)).Debug("send event")
			ws.SetWriteDeadline(time.Now().Add(h.PingTimeout))
			t0 := time.Now()
			_, err = ws.Write(buf)
			if err != nil {
				if ctx.Err() == nil {
					logger.WithError(err).Error("write failed")
				}
				return
			}
			logger.Debug("sent")

			if e != nil {
				hStats.QueueDelayNs += t0.Sub(e.Ready)
				h.mtx.Lock()
				h.lastDelay[queue] = stats.Duration(time.Since(e.Ready))
				h.mtx.Unlock()
			}
			hStats.WriteDelayNs += time.Since(t0)
			hStats.EventBytes += uint64(len(buf))
			hStats.EventCount++
		}
	}()

	// Filter incoming events against the current subscription
	// list, and forward matching events to the outgoing message
	// queue. Close the queue and return when the request context
	// is done/cancelled or the incoming event stream ends. Shut
	// down the handler if the outgoing queue fills up.
	go func() {
		defer cancel()
		ticker := time.NewTicker(h.PingTimeout)
		defer ticker.Stop()

		incoming := eventSource.NewSink()
		defer incoming.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// If the outgoing queue is empty,
				// send an empty message. This can
				// help detect a disconnected network
				// socket, and prevent an idle socket
				// from being closed.
				if len(queue) == 0 {
					select {
					case queue <- []byte(`{}`):
					default:
					}
				}
			case e, ok := <-incoming.Channel():
				if !ok {
					return
				}
				if !sess.Filter(e) {
					continue
				}
				select {
				case queue <- e:
				default:
					logger.WithError(errQueueFull).Error("terminate")
					return
				}
			}
		}
	}()

	<-ctx.Done()
	return
}

func (h *handler) DebugStatus() interface{} {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	var s struct {
		QueueCount    int
		QueueMin      int
		QueueMax      int
		QueueTotal    uint64
		QueueDelayMin stats.Duration
		QueueDelayMax stats.Duration
	}
	for q, lastDelay := range h.lastDelay {
		s.QueueCount++
		n := len(q)
		s.QueueTotal += uint64(n)
		if s.QueueMax < n {
			s.QueueMax = n
		}
		if s.QueueMin > n || s.QueueCount == 1 {
			s.QueueMin = n
		}
		if (s.QueueDelayMin > lastDelay || s.QueueDelayMin == 0) && lastDelay > 0 {
			s.QueueDelayMin = lastDelay
		}
		if s.QueueDelayMax < lastDelay {
			s.QueueDelayMax = lastDelay
		}
	}
	return &s
}

func (h *handler) setup() {
	h.lastDelay = make(map[chan interface{}]stats.Duration)
}
