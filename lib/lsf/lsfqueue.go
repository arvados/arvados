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

	initOnce   sync.Once
	mutex      sync.Mutex
	needUpdate chan bool
	updated    *sync.Cond
	latest     map[string]bjobsEntry
}

// JobID waits for the next queue update (so even a job that was only
// submitted a nanosecond ago will show up) and then returns the LSF
// job ID corresponding to the given container UUID.
func (q *lsfqueue) JobID(uuid string) (int, bool) {
	q.initOnce.Do(q.init)
	q.mutex.Lock()
	defer q.mutex.Unlock()
	select {
	case q.needUpdate <- true:
	default:
		// an update is already pending
	}
	q.updated.Wait()
	ent, ok := q.latest[uuid]
	q.logger.Debugf("JobID(%q) == %d", uuid, ent.id)
	return ent.id, ok
}

func (q *lsfqueue) SetPriority(uuid string, priority int64) {
	q.initOnce.Do(q.init)
	q.logger.Debug("SetPriority is not implemented")
}

func (q *lsfqueue) init() {
	q.updated = sync.NewCond(&q.mutex)
	q.needUpdate = make(chan bool, 1)
	ticker := time.NewTicker(time.Second)
	go func() {
		for range q.needUpdate {
			q.logger.Debug("running bjobs")
			ents, err := q.lsfcli.Bjobs()
			if err != nil {
				q.logger.Warnf("bjobs: %s", err)
				// Retry on the next tick, don't wait
				// for another new call to JobID().
				select {
				case q.needUpdate <- true:
				default:
				}
				<-ticker.C
				continue
			}
			next := make(map[string]bjobsEntry, len(ents))
			for _, ent := range ents {
				next[ent.name] = ent
			}
			q.mutex.Lock()
			q.latest = next
			q.updated.Broadcast()
			q.logger.Debugf("waking up waiters with latest %v", q.latest)
			q.mutex.Unlock()
			// Limit "bjobs" invocations to 1 per second
			<-ticker.C
		}
	}()
}
