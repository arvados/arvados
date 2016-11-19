package main

import (
	"database/sql"
	"strconv"
	"strings"
	"sync"
	"time"

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
	DataSource string
	QueueSize  int

	db         *sql.DB
	pqListener *pq.Listener
	queue      chan *event
	sinks      map[*pgEventSink]bool
	setupOnce  sync.Once
	mtx        sync.Mutex
	shutdown   chan error
}

func (ps *pgEventSource) setup() {
	ps.shutdown = make(chan error, 1)
	ps.sinks = make(map[*pgEventSink]bool)

	db, err := sql.Open("postgres", ps.DataSource)
	if err != nil {
		logger(nil).WithError(err).Fatal("sql.Open failed")
	}
	if err = db.Ping(); err != nil {
		logger(nil).WithError(err).Fatal("db.Ping failed")
	}
	ps.db = db

	ps.pqListener = pq.NewListener(ps.DataSource, time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			// Until we have a mechanism for catching up
			// on missed events, we cannot recover from a
			// dropped connection without breaking our
			// promises to clients.
			logger(nil).WithError(err).Error("listener problem")
			ps.shutdown <- err
		}
	})
	err = ps.pqListener.Listen("logs")
	if err != nil {
		logger(nil).WithError(err).Fatal("pq Listen failed")
	}
	logger(nil).Debug("pgEventSource listening")

	go ps.run()
}

func (ps *pgEventSource) run() {
	ps.queue = make(chan *event, ps.QueueSize)

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

			ps.mtx.Lock()
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
		case err, ok := <-ps.shutdown:
			if ok {
				logger(nil).WithError(err).Info("shutdown")
			}
			close(ps.queue)
			return

		case <-ticker.C:
			logger(nil).Debug("listener ping")
			ps.pqListener.Ping()

		case pqEvent, ok := <-ps.pqListener.Notify:
			if !ok {
				close(ps.queue)
				return
			}
			if pqEvent.Channel != "logs" {
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
	ps.setupOnce.Do(ps.setup)
	sink := &pgEventSink{
		channel: make(chan *event, 1),
		source:  ps,
	}
	ps.mtx.Lock()
	ps.sinks[sink] = true
	ps.mtx.Unlock()
	return sink
}

func (ps *pgEventSource) DB() *sql.DB {
	ps.setupOnce.Do(ps.setup)
	return ps.db
}

func (ps *pgEventSource) Status() interface{} {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	blocked := 0
	for sink := range ps.sinks {
		blocked += len(sink.channel)
	}
	return map[string]interface{}{
		"Queue":        len(ps.queue),
		"QueueMax":     cap(ps.queue),
		"Sinks":        len(ps.sinks),
		"SinksBlocked": blocked,
	}
}

type pgEventSink struct {
	channel chan *event
	source  *pgEventSource
}

func (sink *pgEventSink) Channel() <-chan *event {
	return sink.channel
}

func (sink *pgEventSink) Stop() {
	go func() {
		// Ensure this sink cannot fill up and block the
		// server-side queue (which otherwise could in turn
		// block our mtx.Lock() here)
		for _ = range sink.channel {
		}
	}()
	sink.source.mtx.Lock()
	delete(sink.source.sinks, sink)
	sink.source.mtx.Unlock()
	close(sink.channel)
}
