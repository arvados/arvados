// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"database/sql"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type session interface {
	// Receive processes a message received from the client. If a
	// non-nil error is returned, the connection will be
	// terminated.
	Receive([]byte) error

	// Filter returns true if the event should be queued for
	// sending to the client. It should return as fast as
	// possible, and must not block.
	Filter(*event) bool

	// EventMessage encodes the given event (from the front of the
	// queue) into a form suitable to send to the client. If a
	// non-nil error is returned, the connection is terminated. If
	// the returned buffer is empty, nothing is sent to the client
	// and the event is not counted in statistics.
	//
	// Unlike Filter, EventMessage can block without affecting
	// other connections. If EventMessage is slow, additional
	// incoming events will be queued. If the event queue fills
	// up, the connection will be dropped.
	EventMessage(*event) ([]byte, error)
}

type sessionFactory func(wsConn, chan<- interface{}, *sql.DB, permChecker, *arvados.Client) (session, error)
