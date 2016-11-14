package main

import (
	"database/sql"
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

	pqListener *pq.Listener
	sinks      map[*pgEventSink]bool
	setupOnce  sync.Once
	mtx        sync.Mutex
}

func (ps *pgEventSource) setup() {
	ps.sinks = make(map[*pgEventSink]bool)
	go ps.run()
}

func (ps *pgEventSource) run() {
	db, err := sql.Open("postgres", ps.DataSource)
	if err != nil {
		log.Fatalf("sql.Open: %s", err)
	}
	if err = db.Ping(); err != nil {
		log.Fatalf("db.Ping: %s", err)
	}

	listener := pq.NewListener(ps.DataSource, time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			// Until we have a mechanism for catching up
			// on missed events, we cannot recover from a
			// dropped connection without breaking our
			// promises to clients.
			log.Fatalf("pgEventSource listener problem: %s", err)
		}
	})
	err = listener.Listen("logs")
	if err != nil {
		log.Fatal(err)
	}

	debugLogf("pgEventSource listening")

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
		case <-ticker.C:
			debugLogf("pgEventSource listener ping")
			listener.Ping()

		case pqEvent, ok := <-listener.Notify:
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
				db:       db,
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
