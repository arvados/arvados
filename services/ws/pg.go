package main

import (
	"database/sql"
	"fmt"
	"log"
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
		log.Fatalf("sql.Open: %s", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("db.Ping: %s", err)
	}
	ps.db = db

	ps.pqListener = pq.NewListener(ps.DataSource, time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			// Until we have a mechanism for catching up
			// on missed events, we cannot recover from a
			// dropped connection without breaking our
			// promises to clients.
			ps.shutdown <- fmt.Errorf("pgEventSource listener problem: %s", err)
		}
	})
	err = ps.pqListener.Listen("logs")
	if err != nil {
		log.Fatal(err)
	}
	debugLogf("pgEventSource listening")

	go ps.run()
}

func (ps *pgEventSource) run() {
	eventQueue := make(chan *event, ps.QueueSize)

	go func() {
		for e := range eventQueue {
			// Wait for the "select ... from logs" call to
			// finish. This limits max concurrent queries
			// to ps.QueueSize. Without this, max
			// concurrent queries would be bounded by
			// client_count X client_queue_size.
			e.Detail()
			debugLogf("event %d detail %+v", e.Serial, e.Detail())
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
				debugLogf("shutdown on error: %s", err)
			}
			close(eventQueue)
			return

		case <-ticker.C:
			debugLogf("pgEventSource listener ping")
			ps.pqListener.Ping()

		case pqEvent, ok := <-ps.pqListener.Notify:
			if !ok {
				close(eventQueue)
				return
			}
			if pqEvent.Channel != "logs" {
				continue
			}
			logID, err := strconv.ParseUint(pqEvent.Extra, 10, 64)
			if err != nil {
				log.Printf("bad notify payload: %+v", pqEvent)
				continue
			}
			serial++
			e := &event{
				LogID:    logID,
				Received: time.Now(),
				Serial:   serial,
				db:       ps.db,
			}
			debugLogf("event %d %+v", e.Serial, e)
			eventQueue <- e
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
