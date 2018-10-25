// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/worker"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Sirupsen/logrus"
)

// FixStaleLocks waits for any already-locked containers (i.e., locked
// by a prior dispatcher process) to appear on workers as the worker
// pool recovers its state. It unlocks any that still remain when all
// workers are recovered or shutdown, or its timer expires.
func FixStaleLocks(logger logrus.FieldLogger, queue ContainerQueue, pool WorkerPool, limit time.Duration) {
	wp := pool.Subscribe()
	defer pool.Unsubscribe(wp)
	timeout := time.NewTimer(limit)
waiting:
	for {
		unlock := false
		select {
		case <-wp:
			// If all workers have been contacted, unlock
			// containers that aren't claimed by any
			// worker.
			unlock = pool.Workers()[worker.StateUnknown] == 0
		case <-timeout.C:
			// Give up and unlock the containers, even
			// though they might be working.
			unlock = true
		}

		running := pool.Running()
		qEntries, _ := queue.Entries()
		for uuid, ent := range qEntries {
			if ent.Container.State != arvados.ContainerStateLocked {
				continue
			}
			if _, running := running[uuid]; running {
				continue
			}
			if !unlock {
				continue waiting
			}
			err := queue.Unlock(uuid)
			if err != nil {
				logger.Warnf("Unlock %s: %s", uuid, err)
			}
		}
		return
	}
}
