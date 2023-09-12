// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/stats"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	listenerPingInterval = time.Minute
	testSlowPing         = false
)

type pgEventSource struct {
	DataSource   string
	MaxOpenConns int
	QueueSize    int
	Logger       logrus.FieldLogger
	Reg          *prometheus.Registry

	db         *sql.DB
	pqListener *pq.Listener
	queue      chan *event
	sinks      map[*pgEventSink]bool
	mtx        sync.Mutex

	lastQDelay time.Duration
	eventsIn   prometheus.Counter
	eventsOut  prometheus.Counter

	cancel func()

	setupOnce sync.Once
	ready     chan bool
}

func (ps *pgEventSource) listenerProblem(et pq.ListenerEventType, err error) {
	if et == pq.ListenerEventConnected {
		ps.Logger.Debug("pgEventSource connected")
		return
	}

	// Until we have a mechanism for catching up on missed events,
	// we cannot recover from a dropped connection without
	// breaking our promises to clients.
	ps.Logger.
		WithField("eventType", et).
		WithError(err).
		Error("listener problem")
	ps.cancel()
}

func (ps *pgEventSource) setup() {
	ps.ready = make(chan bool)
	ps.Reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "ws",
			Name:      "queue_len",
			Help:      "Current number of events in queue",
		}, func() float64 { return float64(len(ps.queue)) }))
	ps.Reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "ws",
			Name:      "queue_cap",
			Help:      "Event queue capacity",
		}, func() float64 { return float64(cap(ps.queue)) }))
	ps.Reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "ws",
			Name:      "queue_delay",
			Help:      "Queue delay of the last emitted event",
		}, func() float64 { return ps.lastQDelay.Seconds() }))
	ps.Reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "ws",
			Name:      "sinks",
			Help:      "Number of active sinks (connections)",
		}, func() float64 { return float64(len(ps.sinks)) }))
	ps.Reg.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Namespace: "arvados",
			Subsystem: "ws",
			Name:      "sinks_blocked",
			Help:      "Number of sinks (connections) that are busy and blocking the main event stream",
		}, func() float64 {
			ps.mtx.Lock()
			defer ps.mtx.Unlock()
			blocked := 0
			for sink := range ps.sinks {
				blocked += len(sink.channel)
			}
			return float64(blocked)
		}))
	ps.eventsIn = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "ws",
		Name:      "events_in",
		Help:      "Number of events received from postgresql notify channel",
	})
	ps.Reg.MustRegister(ps.eventsIn)
	ps.eventsOut = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "ws",
		Name:      "events_out",
		Help:      "Number of events sent to client sessions (before filtering)",
	})
	ps.Reg.MustRegister(ps.eventsOut)

	maxConnections := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "ws",
		Name:      "db_max_connections",
		Help:      "Maximum number of open connections to the database",
	})
	ps.Reg.MustRegister(maxConnections)
	openConnections := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "ws",
		Name:      "db_open_connections",
		Help:      "Open connections to the database",
	}, []string{"inuse"})
	ps.Reg.MustRegister(openConnections)

	updateDBStats := func() {
		stats := ps.db.Stats()
		maxConnections.Set(float64(stats.MaxOpenConnections))
		openConnections.WithLabelValues("0").Set(float64(stats.Idle))
		openConnections.WithLabelValues("1").Set(float64(stats.InUse))
	}
	go func() {
		<-ps.ready
		if ps.db == nil {
			return
		}
		updateDBStats()
		for range time.Tick(time.Second) {
			updateDBStats()
		}
	}()
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
	ps.Logger.Debug("pgEventSource Run starting")
	defer ps.Logger.Debug("pgEventSource Run finished")

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
		ps.Logger.WithError(err).Error("sql.Open failed")
		return
	}
	if ps.MaxOpenConns <= 0 {
		ps.Logger.Warn("no database connection limit configured -- consider setting PostgreSQL.ConnectionPool>0 in arvados-ws configuration file")
	}
	db.SetMaxOpenConns(ps.MaxOpenConns)
	if err = db.Ping(); err != nil {
		ps.Logger.WithError(err).Error("db.Ping failed")
		return
	}
	ps.db = db

	ps.pqListener = pq.NewListener(ps.DataSource, time.Second, time.Minute, ps.listenerProblem)
	err = ps.pqListener.Listen("logs")
	if err != nil {
		ps.Logger.WithError(err).Error("pq Listen failed")
		return
	}
	defer ps.pqListener.Close()
	ps.Logger.Debug("pq Listen setup done")

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

			ps.Logger.
				WithField("serial", e.Serial).
				WithField("detail", e.Detail()).
				Debug("event ready")
			e.Ready = time.Now()
			ps.lastQDelay = e.Ready.Sub(e.Received)

			ps.mtx.Lock()
			for sink := range ps.sinks {
				sink.channel <- e
				ps.eventsOut.Inc()
			}
			ps.mtx.Unlock()
		}
	}()

	var serial uint64

	go func() {
		ticker := time.NewTicker(listenerPingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				ps.Logger.Debug("ctx done")
				return

			case <-ticker.C:
				ps.Logger.Debug("listener ping")
				if testSlowPing {
					time.Sleep(time.Second / 2)
				}
				err := ps.pqListener.Ping()
				if err != nil {
					ps.listenerProblem(-1, fmt.Errorf("pqListener ping failed: %s", err))
					continue
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			ps.Logger.Debug("ctx done")
			return

		case pqEvent, ok := <-ps.pqListener.Notify:
			if !ok {
				ps.Logger.Error("pqListener Notify chan closed")
				return
			}
			if pqEvent == nil {
				// pq should call listenerProblem
				// itself in addition to sending us a
				// nil event, so this might be
				// superfluous:
				ps.listenerProblem(-1, errors.New("pqListener Notify chan received nil event"))
				continue
			}
			if pqEvent.Channel != "logs" {
				ps.Logger.WithField("pqEvent", pqEvent).Error("unexpected notify from wrong channel")
				continue
			}
			logID, err := strconv.ParseInt(pqEvent.Extra, 10, 64)
			if err != nil {
				ps.Logger.WithField("pqEvent", pqEvent).Error("bad notify payload")
				continue
			}
			serial++
			e := &event{
				LogID:    logID,
				Received: time.Now(),
				Serial:   serial,
				db:       ps.db,
				logger:   ps.Logger,
			}
			ps.Logger.WithField("event", e).Debug("incoming")
			ps.eventsIn.Inc()
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
	if ps.db == nil {
		return errors.New("database not connected")
	}
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
