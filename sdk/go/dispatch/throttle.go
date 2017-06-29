// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package dispatch

import (
	"sync"
	"time"
)

type throttleEnt struct {
	last time.Time // last attempt that was allowed
}

type throttle struct {
	hold      time.Duration
	seen      map[string]*throttleEnt
	updated   sync.Cond
	setupOnce sync.Once
	mtx       sync.Mutex
}

// Check checks whether there have been too many recent attempts with
// the given uuid, and returns true if it's OK to attempt [again] now.
func (t *throttle) Check(uuid string) bool {
	if t.hold == 0 {
		return true
	}
	t.setupOnce.Do(t.setup)
	t.mtx.Lock()
	defer t.updated.Broadcast()
	defer t.mtx.Unlock()
	ent, ok := t.seen[uuid]
	if !ok {
		t.seen[uuid] = &throttleEnt{last: time.Now()}
		return true
	}
	if time.Since(ent.last) < t.hold {
		return false
	}
	ent.last = time.Now()
	return true
}

func (t *throttle) setup() {
	t.seen = make(map[string]*throttleEnt)
	t.updated.L = &t.mtx
	go func() {
		for range time.NewTicker(t.hold).C {
			t.mtx.Lock()
			for uuid, ent := range t.seen {
				if time.Since(ent.last) >= t.hold {
					delete(t.seen, uuid)
				}
			}
			// don't bother cleaning again until the next update
			t.updated.Wait()
			t.mtx.Unlock()
		}
	}()
}
