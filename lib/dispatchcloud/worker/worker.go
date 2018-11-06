// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"time"
)

// State indicates whether a worker is available to do work, and (if
// not) whether/when it is expected to become ready.
type State int

const (
	StateUnknown  State = iota // might be running a container already
	StateBooting               // instance is booting
	StateIdle                  // instance booted, no containers are running
	StateRunning               // instance is running one or more containers
	StateShutdown              // worker has stopped monitoring the instance
	StateHold                  // running, but not available to run new containers
)

const (
	// TODO: configurable
	maxPingFailTime = 10 * time.Minute
)

var stateString = map[State]string{
	StateUnknown:  "unknown",
	StateBooting:  "booting",
	StateIdle:     "idle",
	StateRunning:  "running",
	StateShutdown: "shutdown",
	StateHold:     "hold",
}

// String implements fmt.Stringer.
func (s State) String() string {
	return stateString[s]
}

// MarshalText implements encoding.TextMarshaler so a JSON encoding of
// map[State]anything uses the state's string representation.
func (s State) MarshalText() ([]byte, error) {
	return []byte(stateString[s]), nil
}
