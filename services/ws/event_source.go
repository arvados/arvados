// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/stats"
	"github.com/lib/pq"
)

type pgConfig map[string]string

func (c pgConfig) ConnectionString() string {
	s := ""
	for k, v := range c {
		s += k
		s += "='"
		s += strings.Replace(
			strings.Replace(v, `\`, `\\`, -1),
			`'`, `\'`, -1)
		s += "' "
	}
	return s
}

type pgEventSource struct {
	DataSource   string
	MaxOpenConns int
	QueueSize    int

	db         *sql.DB
	pqListener *pq.Listener
	queue      chan *event
	sinks      map[*pgEventSink]bool
	mtx        sync.Mutex

	lastQDelay time.Duration
	eventsIn   uint64
	eventsOut  uint64

	cancel func()

	setupOnce sync.Once
	ready     chan bool
}

var _ debugStatuser = (*pgEventSource)(nil)

func (ps *pgEventSource) listenerProblem(et pq.ListenerEventType, err error) {
	if et == pq.ListenerEventConnected {
		logger(nil).Debug("pgEventSource connected")
		return
	}

	// Until we have a mechanism for catching up on missed events,
	// we cannot recover from a dropped connection without
	// breaking our promises to clients.
	logger(nil).
		WithField("eventType", et).
		WithError(err).
		Error("listener problem")
	ps.cancel()
}

func (ps *pgEventSource) setup() {
	ps.ready = make(chan bool)
}

// Close stops listening for new events and disconnects all clients.
func (ps *pgEventSource) Close() {
	ps.WaitReady()
	ps.cancel()
}

// WaitReady returns when the event listener is connected.
func (ps *pgEventSource) WaitReady() {
	ps.setupOnce.Do(ps.setup)
	<-ps.ready
}

// Run listens for event notifications on the "logs" channel and sends
// them to all subscribers.
func (ps *pgEventSource) Run() {
	logger(nil).Debug("pgEventSource Run starting")
	defer logger(nil).Debug("pgEventSource Run finished")

	ps.setupOnce.Do(ps.setup)
	ready := ps.ready
	defer func() {
		if ready != nil {
			close(ready)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	ps.cancel = cancel
	defer cancel()

	defer func() {
		// Disconnect all clients
		ps.mtx.Lock()
		for sink := range ps.sinks {
			close(sink.channel)
		}
		ps.sinks = nil
		ps.mtx.Unlock()
	}()

	db, err := sql.Open("postgres", ps.DataSource)
	if err != nil {
		logger(nil).WithError(err).Error("sql.Open failed")
		return
	}
	if ps.MaxOpenConns <= 0 {
		logger(nil).Warn("no database connection limit configured -- consider setting PostgresPool>0 in arvados-ws configuration file")
	}
	db.SetMaxOpenConns(ps.MaxOpenConns)
	if err = db.Ping(); err != nil {
		logger(nil).WithError(err).Error("db.Ping failed")
		return
	}
	ps.db = db

	ps.pqListener = pq.NewListener(ps.DataSource, time.Second, time.Minute, ps.listenerProblem)
	err = ps.pqListener.Listen("logs")
	if err != nil {
		logger(nil).WithError(err).Error("pq Listen failed")
		return
	}
	defer ps.pqListener.Close()
	logger(nil).Debug("pq Listen setup done")

	close(ready)
	// Avoid double-close in deferred func
	ready = nil

	ps.queue = make(chan *event, ps.QueueSize)
	defer close(ps.queue)

	go func() {
		for e := range ps.queue {
			// Wait for the "select ... from logs" call to
			// finish. This limits max concurrent queries
			// to ps.QueueSize. Without this, max
			// concurrent queries would be bounded by
			// client_count X client_queue_size.
			e.Detail()

			logger(nil).
				WithField("serial", e.Serial).
				WithField("detail", e.Detail()).
				Debug("event ready")
			e.Ready = time.Now()
			ps.lastQDelay = e.Ready.Sub(e.Received)

			ps.mtx.Lock()
			atomic.AddUint64(&ps.eventsOut, uint64(len(ps.sinks)))
			for sink := range ps.sinks {
				sink.channel <- e
			}
			ps.mtx.Unlock()
		}
	}()

	var serial uint64
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger(nil).Debug("ctx done")
			return

		case <-ticker.C:
			logger(nil).Debug("listener ping")
			ps.pqListener.Ping()

		case pqEvent, ok := <-ps.pqListener.Notify:
			if !ok {
				logger(nil).Debug("pqListener Notify chan closed")
				return
			}
			if pqEvent == nil {
				// pq should call listenerProblem
				// itself in addition to sending us a
				// nil event, so this might be
				// superfluous:
				ps.listenerProblem(-1, nil)
				continue
			}
			if pqEvent.Channel != "logs" {
				logger(nil).WithField("pqEvent", pqEvent).Error("unexpected notify from wrong channel")
				continue
			}
			logID, err := strconv.ParseUint(pqEvent.Extra, 10, 64)
			if err != nil {
				logger(nil).WithField("pqEvent", pqEvent).Error("bad notify payload")
				continue
			}
			serial++
			e := &event{
				LogID:    logID,
				Received: time.Now(),
				Serial:   serial,
				db:       ps.db,
			}
			logger(nil).WithField("event", e).Debug("incoming")
			atomic.AddUint64(&ps.eventsIn, 1)
			ps.queue <- e
			go e.Detail()
		}
	}
}

// NewSink subscribes to the event source. NewSink returns an
// eventSink, whose Channel() method returns a channel: a pointer to
// each subsequent event will be sent to that channel.
//
// The caller must ensure events are received from the sink channel as
// quickly as possible because when one sink stops being ready, all
// other sinks block.
func (ps *pgEventSource) NewSink() eventSink {
	sink := &pgEventSink{
		channel: make(chan *event, 1),
		source:  ps,
	}
	ps.mtx.Lock()
	if ps.sinks == nil {
		ps.sinks = make(map[*pgEventSink]bool)
	}
	ps.sinks[sink] = true
	ps.mtx.Unlock()
	return sink
}

func (ps *pgEventSource) DB() *sql.DB {
	ps.WaitReady()
	return ps.db
}

func (ps *pgEventSource) DBHealth() error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	defer cancel()
	var i int
	return ps.db.QueryRowContext(ctx, "SELECT 1").Scan(&i)
}

func (ps *pgEventSource) DebugStatus() interface{} {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	blocked := 0
	for sink := range ps.sinks {
		blocked += len(sink.channel)
	}
	return map[string]interface{}{
		"EventsIn":     atomic.LoadUint64(&ps.eventsIn),
		"EventsOut":    atomic.LoadUint64(&ps.eventsOut),
		"Queue":        len(ps.queue),
		"QueueLimit":   cap(ps.queue),
		"QueueDelay":   stats.Duration(ps.lastQDelay),
		"Sinks":        len(ps.sinks),
		"SinksBlocked": blocked,
		"DBStats":      ps.db.Stats(),
	}
}

type pgEventSink struct {
	channel chan *event
	source  *pgEventSource
}

func (sink *pgEventSink) Channel() <-chan *event {
	return sink.channel
}

// Stop sending events to the sink's channel.
func (sink *pgEventSink) Stop() {
	go func() {
		// Ensure this sink cannot fill up and block the
		// server-side queue (which otherwise could in turn
		// block our mtx.Lock() here)
		for range sink.channel {
		}
	}()
	sink.source.mtx.Lock()
	if _, ok := sink.source.sinks[sink]; ok {
		delete(sink.source.sinks, sink)
		close(sink.channel)
	}
	sink.source.mtx.Unlock()
}
