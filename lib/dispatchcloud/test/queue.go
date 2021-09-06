// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package test

import (
	"fmt"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud/container"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

// Queue is a test stub for container.Queue. The caller specifies the
// initial queue state.
type Queue struct {
	// Containers represent the API server database contents.
	Containers []arvados.Container

	// ChooseType will be called for each entry in Containers. It
	// must not be nil.
	ChooseType func(*arvados.Container) (arvados.InstanceType, error)

	Logger logrus.FieldLogger

	entries      map[string]container.QueueEnt
	updTime      time.Time
	subscribers  map[<-chan struct{}]chan struct{}
	stateChanges []QueueStateChange

	mtx sync.Mutex
}

type QueueStateChange struct {
	UUID string
	From arvados.ContainerState
	To   arvados.ContainerState
}

// All calls to Lock/Unlock/Cancel to date.
func (q *Queue) StateChanges() []QueueStateChange {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	return q.stateChanges
}

// Entries returns the containers that were queued when Update was
// last called.
func (q *Queue) Entries() (map[string]container.QueueEnt, time.Time) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	updTime := q.updTime
	r := map[string]container.QueueEnt{}
	for uuid, ent := range q.entries {
		r[uuid] = ent
	}
	return r, updTime
}

// Get returns the container from the cached queue, i.e., as it was
// when Update was last called -- just like a container.Queue does. If
// the state has been changed (via Lock, Unlock, or Cancel) since the
// last Update, the updated state is returned.
func (q *Queue) Get(uuid string) (arvados.Container, bool) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	ent, ok := q.entries[uuid]
	return ent.Container, ok
}

func (q *Queue) Forget(uuid string) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	delete(q.entries, uuid)
}

func (q *Queue) Lock(uuid string) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	return q.changeState(uuid, arvados.ContainerStateQueued, arvados.ContainerStateLocked)
}

func (q *Queue) Unlock(uuid string) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	return q.changeState(uuid, arvados.ContainerStateLocked, arvados.ContainerStateQueued)
}

func (q *Queue) Cancel(uuid string) error {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	return q.changeState(uuid, q.entries[uuid].Container.State, arvados.ContainerStateCancelled)
}

func (q *Queue) Subscribe() <-chan struct{} {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	if q.subscribers == nil {
		q.subscribers = map[<-chan struct{}]chan struct{}{}
	}
	ch := make(chan struct{}, 1)
	q.subscribers[ch] = ch
	return ch
}

func (q *Queue) Unsubscribe(ch <-chan struct{}) {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	delete(q.subscribers, ch)
}

// caller must have lock.
func (q *Queue) notify() {
	for _, ch := range q.subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// caller must have lock.
func (q *Queue) changeState(uuid string, from, to arvados.ContainerState) error {
	ent := q.entries[uuid]
	q.stateChanges = append(q.stateChanges, QueueStateChange{uuid, from, to})
	if ent.Container.State != from {
		return fmt.Errorf("changeState failed: state=%q", ent.Container.State)
	}
	ent.Container.State = to
	q.entries[uuid] = ent
	for i, ctr := range q.Containers {
		if ctr.UUID == uuid {
			q.Containers[i].State = to
			break
		}
	}
	q.notify()
	return nil
}

// Update rebuilds the current entries from the Containers slice.
func (q *Queue) Update() error {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	updTime := time.Now()
	upd := map[string]container.QueueEnt{}
	for _, ctr := range q.Containers {
		_, exists := q.entries[ctr.UUID]
		if !exists && (ctr.State == arvados.ContainerStateComplete || ctr.State == arvados.ContainerStateCancelled) {
			continue
		}
		if ent, ok := upd[ctr.UUID]; ok {
			ent.Container = ctr
			upd[ctr.UUID] = ent
		} else {
			it, _ := q.ChooseType(&ctr)
			upd[ctr.UUID] = container.QueueEnt{
				Container:    ctr,
				InstanceType: it,
			}
		}
	}
	q.entries = upd
	q.updTime = updTime
	q.notify()
	return nil
}

// Notify adds/updates an entry in the Containers slice.  This
// simulates the effect of an API update from someone other than the
// dispatcher -- e.g., crunch-run updating state to "Complete" when a
// container exits.
//
// The resulting changes are not exposed through Get() or Entries()
// until the next call to Update().
//
// Return value is true unless the update is rejected (invalid state
// transition).
func (q *Queue) Notify(upd arvados.Container) bool {
	q.mtx.Lock()
	defer q.mtx.Unlock()
	for i, ctr := range q.Containers {
		if ctr.UUID == upd.UUID {
			if allowContainerUpdate[ctr.State][upd.State] {
				q.Containers[i] = upd
				return true
			}
			if q.Logger != nil {
				q.Logger.WithField("ContainerUUID", ctr.UUID).Infof("test.Queue rejected update from %s to %s", ctr.State, upd.State)
			}
			return false
		}
	}
	q.Containers = append(q.Containers, upd)
	return true
}

var allowContainerUpdate = map[arvados.ContainerState]map[arvados.ContainerState]bool{
	arvados.ContainerStateQueued: {
		arvados.ContainerStateQueued:    true,
		arvados.ContainerStateLocked:    true,
		arvados.ContainerStateCancelled: true,
	},
	arvados.ContainerStateLocked: {
		arvados.ContainerStateQueued:    true,
		arvados.ContainerStateLocked:    true,
		arvados.ContainerStateRunning:   true,
		arvados.ContainerStateCancelled: true,
	},
	arvados.ContainerStateRunning: {
		arvados.ContainerStateRunning:   true,
		arvados.ContainerStateCancelled: true,
		arvados.ContainerStateComplete:  true,
	},
}
