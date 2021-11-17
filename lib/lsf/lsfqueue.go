// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package lsf

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type lsfqueue struct {
	logger logrus.FieldLogger
	period time.Duration
	lsfcli *lsfcli

	initOnce  sync.Once
	mutex     sync.Mutex
	nextReady chan (<-chan struct{})
	updated   *sync.Cond
	latest    map[string]bjobsEntry
}

// JobID waits for the next queue update (so even a job that was only
// submitted a nanosecond ago will show up) and then returns the LSF
// job ID corresponding to the given container UUID.
func (q *lsfqueue) JobID(uuid string) (string, bool) {
	ent, ok := q.getNext()[uuid]
	return ent.ID, ok
}

// All waits for the next queue update, then returns the names of all
// jobs in the queue. Used by checkLsfQueueForOrphans().
func (q *lsfqueue) All() []string {
	latest := q.getNext()
	names := make([]string, 0, len(latest))
	for name := range latest {
		names = append(names, name)
	}
	return names
}

func (q *lsfqueue) SetPriority(uuid string, priority int64) {
	q.initOnce.Do(q.init)
	q.logger.Debug("SetPriority is not implemented")
}

func (q *lsfqueue) getNext() map[string]bjobsEntry {
	q.initOnce.Do(q.init)
	<-(<-q.nextReady)
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.latest
}

func (q *lsfqueue) init() {
	q.updated = sync.NewCond(&q.mutex)
	q.nextReady = make(chan (<-chan struct{}))
	ticker := time.NewTicker(time.Second)
	go func() {
		for range ticker.C {
			// Send a new "next update ready" channel to
			// the next goroutine that wants one (and any
			// others that have already queued up since
			// the first one started waiting).
			//
			// Below, when we get a new update, we'll
			// signal that to the other goroutines by
			// closing the ready chan.
			ready := make(chan struct{})
			q.nextReady <- ready
			for {
				select {
				case q.nextReady <- ready:
					continue
				default:
				}
				break
			}
			// Run bjobs repeatedly if needed, until we
			// get valid output.
			var ents []bjobsEntry
			for {
				q.logger.Debug("running bjobs")
				var err error
				ents, err = q.lsfcli.Bjobs()
				if err == nil {
					break
				}
				q.logger.Warnf("bjobs: %s", err)
				<-ticker.C
			}
			next := make(map[string]bjobsEntry, len(ents))
			for _, ent := range ents {
				next[ent.Name] = ent
			}
			// Replace q.latest and notify all the
			// goroutines that the "next update" they
			// asked for is now ready.
			q.mutex.Lock()
			q.latest = next
			q.mutex.Unlock()
			close(ready)
		}
	}()
}
