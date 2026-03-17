// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// fixStaleLocks waits for any already-locked containers (i.e., locked
// by a prior dispatcher process) to appear on workers as the worker
// pool recovers its state. It unlocks any that still remain when all
// workers are recovered or shutdown, or its timer (StaleLockTimeout)
// expires.
func (sch *Scheduler) fixStaleLocks() {
	wp := sch.pool.Subscribe()
	defer sch.pool.Unsubscribe(wp)

	var stale []string
	timeout := time.NewTimer(time.Duration(sch.cluster.Containers.StaleLockTimeout))
waiting:
	for sch.pool.CountWorkers()[worker.StateUnknown] > 0 {
		running := sch.pool.Running()
		qEntries, _ := sch.queue.Entries()

		stale = nil
		for uuid, ent := range qEntries {
			if ent.Container.State != arvados.ContainerStateLocked {
				continue
			}
			if _, running := running[uuid]; running {
				continue
			}
			stale = append(stale, uuid)
		}
		if len(stale) == 0 {
			return
		}

		select {
		case <-wp:
		case <-timeout.C:
			// Give up.
			break waiting
		}
	}

	for _, uuid := range stale {
		err := sch.queue.Unlock(uuid)
		if err != nil {
			sch.logger.Warnf("Unlock %s: %s", uuid, err)
		}
	}
}
