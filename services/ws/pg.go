package main

import (
	"database/sql"
	"log"
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
	PgConfig  pgConfig
	QueueSize int

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
	db, err := sql.Open("postgres", ps.PgConfig.ConnectionString())
	if err != nil {
		log.Fatal(err)
	}

	listener := pq.NewListener(ps.PgConfig.ConnectionString(), time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			// Until we have a mechanism for catching up
			// on missed events, we cannot recover from a
			// dropped connection without breaking our
			// promises to clients.
			log.Fatal(err)
		}
	})
	err = listener.Listen("logs")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for _ = range time.NewTicker(time.Minute).C {
			listener.Ping()
		}
	}()

	eventQueue := make(chan *event, ps.QueueSize)
	go func() {
		for e := range eventQueue {
			// Wait for the "select ... from logs" call to
			// finish. This limits max concurrent queries
			// to ps.QueueSize. Without this, max
			// concurrent queries would be bounded by
			// client_count X client_queue_size.
			e.Detail(db)
			debugLogf("%+v", e)
			ps.mtx.Lock()
			for sink := range ps.sinks {
				sink.channel <- e
			}
			ps.mtx.Unlock()
		}
	}()

	var serial uint64
	for pqEvent := range listener.Notify {
		if pqEvent.Channel != "logs" {
			continue
		}
		serial++
		e := &event{
			LogUUID:  pqEvent.Extra,
			Received: time.Now(),
			Serial:   serial,
		}
		debugLogf("%+v", e)
		eventQueue <- e
		go e.Detail(db)
	}
}

// NewSink subscribes to the event source. If c is not nil, it will be
// used as the event channel. Otherwise, a new channel will be
// created. Either way, the sink channel will be returned by the
// Channel() method of the returned eventSink. All subsequent events
// will be sent to the sink channel. The caller must ensure events are
// received from the sink channel as quickly as possible: when one
// sink blocks, all other sinks also block.
func (ps *pgEventSource) NewSink(c chan *event) eventSink {
	ps.setupOnce.Do(ps.setup)
	if c == nil {
		c = make(chan *event, 1)
	}
	sink := &pgEventSink{
		channel: c,
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
		for _ = range sink.channel {}
	}()
	sink.source.mtx.Lock()
	delete(sink.source.sinks, sink)
	sink.source.mtx.Unlock()
	close(sink.channel)
}
