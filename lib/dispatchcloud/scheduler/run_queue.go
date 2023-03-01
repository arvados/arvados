// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"sort"
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud/container"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

var quietAfter503 = time.Minute

func isSupervisor(ctr arvados.Container) bool {
	return (len(ctr.Command) > 0 && ctr.Command[0] == "arvados-cwl-runner") || ctr.SchedulingParameters.Supervisor
}

func (sch *Scheduler) runQueue() {
	unsorted, _ := sch.queue.Entries()
	sorted := make([]container.QueueEnt, 0, len(unsorted))
	for _, ent := range unsorted {
		sorted = append(sorted, ent)
	}
	sort.Slice(sorted, func(i, j int) bool {
		if pi, pj := sorted[i].Container.Priority, sorted[j].Container.Priority; pi != pj {
			return pi > pj
		} else {
			// When containers have identical priority,
			// start them in the order we first noticed
			// them. This avoids extra lock/unlock cycles
			// when we unlock the containers that don't
			// fit in the available pool.
			return sorted[i].FirstSeenAt.Before(sorted[j].FirstSeenAt)
		}
	})

	running := sch.pool.Running()
	unalloc := sch.pool.Unallocated()

	if t := sch.client.Last503(); t.After(sch.last503time) {
		// API has sent an HTTP 503 response since last time
		// we checked. Use current #containers - 1 as
		// maxConcurrency, i.e., try to stay just below the
		// level where we see 503s.
		sch.last503time = t
		if newlimit := len(running) - 1; newlimit < 1 {
			sch.maxConcurrency = 1
		} else {
			sch.maxConcurrency = newlimit
		}
	} else if sch.maxConcurrency > 0 && time.Since(sch.last503time) > quietAfter503 {
		// If we haven't seen any 503 errors lately, raise
		// limit to ~10% beyond the current workload.
		//
		// As we use the added 10% to schedule more
		// containers, len(running) will increase and we'll
		// push the limit up further. Soon enough,
		// maxConcurrency will get high enough to schedule the
		// entire queue, hit pool quota, or get 503s again.
		max := len(running)*11/10 + 1
		if sch.maxConcurrency < max {
			sch.maxConcurrency = max
		}
	}
	if sch.last503time.IsZero() {
		sch.mLast503Time.Set(0)
	} else {
		sch.mLast503Time.Set(float64(sch.last503time.Unix()))
	}
	sch.mMaxContainerConcurrency.Set(float64(sch.maxConcurrency))

	sch.logger.WithFields(logrus.Fields{
		"Containers":     len(sorted),
		"Processes":      len(running),
		"maxConcurrency": sch.maxConcurrency,
	}).Debug("runQueue")

	dontstart := map[arvados.InstanceType]bool{}
	var overquota []container.QueueEnt // entries that are unmappable because of worker pool quota
	var containerAllocatedWorkerBootingCount int

	// trying is #containers running + #containers we're trying to
	// start. We stop trying to start more containers if this
	// reaches the dynamic maxConcurrency limit.
	trying := len(running)

	supervisors := 0

tryrun:
	for i, ctr := range sorted {
		ctr, it := ctr.Container, ctr.InstanceType
		logger := sch.logger.WithFields(logrus.Fields{
			"ContainerUUID": ctr.UUID,
			"InstanceType":  it.Name,
		})
		if isSupervisor(ctr) {
			supervisors += 1
			if sch.maxSupervisors > 0 && supervisors > sch.maxSupervisors {
				continue
			}
		}
		if _, running := running[ctr.UUID]; running || ctr.Priority < 1 {
			continue
		}
		switch ctr.State {
		case arvados.ContainerStateQueued:
			if sch.maxConcurrency > 0 && trying >= sch.maxConcurrency {
				logger.Tracef("not locking: already at maxConcurrency %d", sch.maxConcurrency)
				overquota = sorted[i:]
				break tryrun
			}
			trying++
			if unalloc[it] < 1 && sch.pool.AtQuota() {
				logger.Trace("not locking: AtQuota and no unalloc workers")
				overquota = sorted[i:]
				break tryrun
			}
			if sch.pool.KillContainer(ctr.UUID, "about to lock") {
				logger.Info("not locking: crunch-run process from previous attempt has not exited")
				continue
			}
			go sch.lockContainer(logger, ctr.UUID)
			unalloc[it]--
		case arvados.ContainerStateLocked:
			if sch.maxConcurrency > 0 && trying >= sch.maxConcurrency {
				logger.Debugf("not starting: already at maxConcurrency %d", sch.maxConcurrency)
				overquota = sorted[i:]
				break tryrun
			}
			trying++
			if unalloc[it] > 0 {
				unalloc[it]--
			} else if sch.pool.AtQuota() {
				// Don't let lower-priority containers
				// starve this one by using keeping
				// idle workers alive on different
				// instance types.
				logger.Trace("overquota")
				overquota = sorted[i:]
				break tryrun
			} else if sch.pool.Create(it) {
				// Success. (Note pool.Create works
				// asynchronously and does its own
				// logging about the eventual outcome,
				// so we don't need to.)
				logger.Info("creating new instance")
			} else {
				// Failed despite not being at quota,
				// e.g., cloud ops throttled.  TODO:
				// avoid getting starved here if
				// instances of a specific type always
				// fail.
				logger.Trace("pool declined to create new instance")
				continue
			}

			if dontstart[it] {
				// We already tried & failed to start
				// a higher-priority container on the
				// same instance type. Don't let this
				// one sneak in ahead of it.
			} else if sch.pool.KillContainer(ctr.UUID, "about to start") {
				logger.Info("not restarting yet: crunch-run process from previous attempt has not exited")
			} else if sch.pool.StartContainer(it, ctr) {
				logger.Trace("StartContainer => true")
				// Success.
			} else {
				logger.Trace("StartContainer => false")
				containerAllocatedWorkerBootingCount += 1
				dontstart[it] = true
			}
		}
	}

	sch.mContainersAllocatedNotStarted.Set(float64(containerAllocatedWorkerBootingCount))
	sch.mContainersNotAllocatedOverQuota.Set(float64(len(overquota)))

	if len(overquota) > 0 {
		// Unlock any containers that are unmappable while
		// we're at quota (but if they have already been
		// scheduled and they're loading docker images etc.,
		// let them run).
		for _, ctr := range overquota {
			ctr := ctr.Container
			_, toolate := running[ctr.UUID]
			if ctr.State == arvados.ContainerStateLocked && !toolate {
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

// Lock the given container. Should be called in a new goroutine.
func (sch *Scheduler) lockContainer(logger logrus.FieldLogger, uuid string) {
	if !sch.uuidLock(uuid, "lock") {
		return
	}
	defer sch.uuidUnlock(uuid)
	if ctr, ok := sch.queue.Get(uuid); !ok || ctr.State != arvados.ContainerStateQueued {
		// This happens if the container has been cancelled or
		// locked since runQueue called sch.queue.Entries(),
		// possibly by a lockContainer() call from a previous
		// runQueue iteration. In any case, we will respond
		// appropriately on the next runQueue iteration, which
		// will have already been triggered by the queue
		// update.
		logger.WithField("State", ctr.State).Debug("container no longer queued by the time we decided to lock it, doing nothing")
		return
	}
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
}

// Acquire a non-blocking lock for specified UUID, returning true if
// successful.  The op argument is used only for debug logs.
//
// If the lock is not available, uuidLock arranges to wake up the
// scheduler after a short delay, so it can retry whatever operation
// is trying to get the lock (if that operation is still worth doing).
//
// This mechanism helps avoid spamming the controller/database with
// concurrent updates for any single container, even when the
// scheduler loop is running frequently.
func (sch *Scheduler) uuidLock(uuid, op string) bool {
	sch.mtx.Lock()
	defer sch.mtx.Unlock()
	logger := sch.logger.WithFields(logrus.Fields{
		"ContainerUUID": uuid,
		"Op":            op,
	})
	if op, locked := sch.uuidOp[uuid]; locked {
		logger.Debugf("uuidLock not available, Op=%s in progress", op)
		// Make sure the scheduler loop wakes up to retry.
		sch.wakeup.Reset(time.Second / 4)
		return false
	}
	logger.Debug("uuidLock acquired")
	sch.uuidOp[uuid] = op
	return true
}

func (sch *Scheduler) uuidUnlock(uuid string) {
	sch.mtx.Lock()
	defer sch.mtx.Unlock()
	delete(sch.uuidOp, uuid)
}
