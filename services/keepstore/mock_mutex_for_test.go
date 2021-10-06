// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

type MockMutex struct {
	AllowLock   chan struct{}
	AllowUnlock chan struct{}
}

func NewMockMutex() *MockMutex {
	return &MockMutex{
		AllowLock:   make(chan struct{}),
		AllowUnlock: make(chan struct{}),
	}
}

// Lock waits for someone to send to AllowLock.
func (m *MockMutex) Lock() {
	<-m.AllowLock
}

// Unlock waits for someone to send to AllowUnlock.
func (m *MockMutex) Unlock() {
	<-m.AllowUnlock
}
