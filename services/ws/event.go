package main

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type eventSink interface {
	Channel() <-chan *event
	Stop()
}

type eventSource interface {
	NewSink(chan *event) eventSink
}

type event struct {
	LogUUID  string
	Received time.Time
	Serial   uint64

	logRow *arvados.Log
	err error
	mtx sync.Mutex
}

// Detail returns the database row corresponding to the event. It can
// be called safely from multiple goroutines. Only one attempt will be
// made. If the database row cannot be retrieved, Detail returns nil.
func (e *event) Detail(db *sql.DB) *arvados.Log {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if e.logRow != nil || e.err != nil {
		return e.logRow
	}
	var logRow arvados.Log
	var oldAttrs, newAttrs []byte
	e.err = db.QueryRow(`SELECT uuid, object_uuid, object_owner_uuid, event_type, created_at, old_attributes, new_attributes FROM logs WHERE uuid = ?`, e.LogUUID).Scan(
		&logRow.UUID,
		&logRow.ObjectUUID,
		&logRow.ObjectOwnerUUID,
		&logRow.EventType,
		&logRow.CreatedAt,
		&oldAttrs,
		&newAttrs)
	if e.err != nil {
		log.Printf("retrieving log row %s: %s", e.LogUUID, e.err)
	}
	return e.logRow
}
