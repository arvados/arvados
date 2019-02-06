// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"sort"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/container"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

func (sch *Scheduler) runQueue() {
	unsorted, _ := sch.queue.Entries()
	sorted := make([]container.QueueEnt, 0, len(unsorted))
	for _, ent := range unsorted {
		sorted = append(sorted, ent)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Container.Priority > sorted[j].Container.Priority
	})

	running := sch.pool.Running()
	unalloc := sch.pool.Unallocated()

	sch.logger.WithFields(logrus.Fields{
		"Containers": len(sorted),
		"Processes":  len(running),
	}).Debug("runQueue")

	dontstart := map[arvados.InstanceType]bool{}
	var overquota []container.QueueEnt // entries that are unmappable because of worker pool quota

tryrun:
	for i, ctr := range sorted {
		ctr, it := ctr.Container, ctr.InstanceType
		logger := sch.logger.WithFields(logrus.Fields{
			"ContainerUUID": ctr.UUID,
			"InstanceType":  it.Name,
		})
		if _, running := running[ctr.UUID]; running || ctr.Priority < 1 {
			continue
		}
		switch ctr.State {
		case arvados.ContainerStateQueued:
			if unalloc[it] < 1 && sch.pool.AtQuota() {
				logger.Debug("not locking: AtQuota and no unalloc workers")
				overquota = sorted[i:]
				break tryrun
			}
			sch.bgLock(logger, ctr.UUID)
			unalloc[it]--
		case arvados.ContainerStateLocked:
			if unalloc[it] > 0 {
				unalloc[it]--
			} else if sch.pool.AtQuota() {
				logger.Debug("not starting: AtQuota and no unalloc workers")
				overquota = sorted[i:]
				break tryrun
			} else {
				logger.Info("creating new instance")
				if !sch.pool.Create(it) {
					// (Note pool.Create works
					// asynchronously and logs its
					// own failures, so we don't
					// need to log this as a
					// failure.)

					sch.queue.Unlock(ctr.UUID)
					// Don't let lower-priority
					// containers starve this one
					// by using keeping idle
					// workers alive on different
					// instance types.  TODO:
					// avoid getting starved here
					// if instances of a specific
					// type always fail.
					overquota = sorted[i:]
					break tryrun
				}
			}

			if dontstart[it] {
				// We already tried & failed to start
				// a higher-priority container on the
				// same instance type. Don't let this
				// one sneak in ahead of it.
			} else if sch.pool.StartContainer(it, ctr) {
				// Success.
			} else {
				dontstart[it] = true
			}
		}
	}

	if len(overquota) > 0 {
		// Unlock any containers that are unmappable while
		// we're at quota.
		for _, ctr := range overquota {
			ctr := ctr.Container
			if ctr.State == arvados.ContainerStateLocked {
				logger := sch.logger.WithField("ContainerUUID", ctr.UUID)
				logger.Debug("unlock because pool capacity is used by higher priority containers")
				err := sch.queue.Unlock(ctr.UUID)
				if err != nil {
					logger.WithError(err).Warn("error unlocking")
				}
			}
		}
		// Shut down idle workers that didn't get any
		// containers mapped onto them before we hit quota.
		for it, n := range unalloc {
			if n < 1 {
				continue
			}
			sch.pool.Shutdown(it)
		}
	}
}

// Start an API call to lock the given container, and return
// immediately while waiting for the response in a new goroutine. Do
// nothing if a lock request is already in progress for this
// container.
func (sch *Scheduler) bgLock(logger logrus.FieldLogger, uuid string) {
	logger.Debug("locking")
	sch.mtx.Lock()
	defer sch.mtx.Unlock()
	if sch.locking[uuid] {
		logger.Debug("locking in progress, doing nothing")
		return
	}
	if ctr, ok := sch.queue.Get(uuid); !ok || ctr.State != arvados.ContainerStateQueued {
		// This happens if the container has been cancelled or
		// locked since runQueue called sch.queue.Entries(),
		// possibly by a bgLock() call from a previous
		// runQueue iteration. In any case, we will respond
		// appropriately on the next runQueue iteration, which
		// will have already been triggered by the queue
		// update.
		logger.WithField("State", ctr.State).Debug("container no longer queued by the time we decided to lock it, doing nothing")
		return
	}
	sch.locking[uuid] = true
	go func() {
		defer func() {
			sch.mtx.Lock()
			defer sch.mtx.Unlock()
			delete(sch.locking, uuid)
		}()
		err := sch.queue.Lock(uuid)
		if err != nil {
			logger.WithError(err).Warn("error locking container")
			return
		}
		logger.Debug("lock succeeded")
		ctr, ok := sch.queue.Get(uuid)
		if !ok {
			logger.Error("(BUG?) container disappeared from queue after Lock succeeded")
		} else if ctr.State != arvados.ContainerStateLocked {
			logger.Warnf("(race?) container has state=%q after Lock succeeded", ctr.State)
		}
	}()
}
