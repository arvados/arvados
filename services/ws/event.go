// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"database/sql"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
)

type eventSink interface {
	Channel() <-chan *event
	Stop()
}

type eventSource interface {
	NewSink() eventSink
	DB() *sql.DB
	DBHealth() error
}

type event struct {
	LogID    uint64
	Received time.Time
	Ready    time.Time
	Serial   uint64

	db     *sql.DB
	logRow *arvados.Log
	err    error
	mtx    sync.Mutex
}

// Detail returns the database row corresponding to the event. It can
// be called safely from multiple goroutines. Only one attempt will be
// made. If the database row cannot be retrieved, Detail returns nil.
func (e *event) Detail() *arvados.Log {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	if e.logRow != nil || e.err != nil {
		return e.logRow
	}
	var logRow arvados.Log
	var propYAML []byte
	e.err = e.db.QueryRow(`SELECT id, uuid, object_uuid, COALESCE(object_owner_uuid,''), COALESCE(event_type,''), event_at, created_at, properties FROM logs WHERE id = $1`, e.LogID).Scan(
		&logRow.ID,
		&logRow.UUID,
		&logRow.ObjectUUID,
		&logRow.ObjectOwnerUUID,
		&logRow.EventType,
		&logRow.EventAt,
		&logRow.CreatedAt,
		&propYAML)
	if e.err != nil {
		logger(nil).WithField("LogID", e.LogID).WithError(e.err).Error("QueryRow failed")
		return nil
	}
	e.err = yaml.Unmarshal(propYAML, &logRow.Properties)
	if e.err != nil {
		logger(nil).WithField("LogID", e.LogID).WithError(e.err).Error("yaml decode failed")
		return nil
	}
	e.logRow = &logRow
	return e.logRow
}
