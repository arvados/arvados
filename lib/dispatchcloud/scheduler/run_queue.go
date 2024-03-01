// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"fmt"
	"sort"
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud/container"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

var quietAfter503 = time.Minute

type QueueEnt struct {
	container.QueueEnt

	// Human-readable scheduling status as of the last scheduling
	// iteration.
	SchedulingStatus string `json:"scheduling_status"`
}

// Queue returns the sorted queue from the last scheduling iteration.
func (sch *Scheduler) Queue() []QueueEnt {
	ents, _ := sch.lastQueue.Load().([]QueueEnt)
	return ents
}

func (sch *Scheduler) runQueue() {
	running := sch.pool.Running()
	unalloc := sch.pool.Unallocated()

	totalInstances := 0
	for _, n := range sch.pool.CountWorkers() {
		totalInstances += n
	}

	unsorted, _ := sch.queue.Entries()
	sorted := make([]QueueEnt, 0, len(unsorted))
	for _, ent := range unsorted {
		sorted = append(sorted, QueueEnt{QueueEnt: ent})
	}
	sort.Slice(sorted, func(i, j int) bool {
		_, irunning := running[sorted[i].Container.UUID]
		_, jrunning := running[sorted[j].Container.UUID]
		if irunning != jrunning {
			// Ensure the "tryrun" loop (see below) sees
			// already-scheduled containers first, to
			// ensure existing supervisor containers are
			// properly counted before we decide whether
			// we have room for new ones.
			return irunning
		}
		ilocked := sorted[i].Container.State == arvados.ContainerStateLocked
		jlocked := sorted[j].Container.State == arvados.ContainerStateLocked
		if ilocked != jlocked {
			// Give precedence to containers that we have
			// already locked, even if higher-priority
			// containers have since arrived in the
			// queue. This avoids undesirable queue churn
			// effects including extra lock/unlock cycles
			// and bringing up new instances and quickly
			// shutting them down to make room for
			// different instance sizes.
			return ilocked
		} else if pi, pj := sorted[i].Container.Priority, sorted[j].Container.Priority; pi != pj {
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
	if sch.maxInstances > 0 && sch.maxConcurrency > sch.maxInstances {
		sch.maxConcurrency = sch.maxInstances
	}
	if sch.instancesWithinQuota > 0 && sch.instancesWithinQuota < totalInstances {
		// Evidently it is possible to run this many
		// instances, so raise our estimate.
		sch.instancesWithinQuota = totalInstances
	}
	if sch.pool.AtQuota() {
		// Consider current workload to be the maximum
		// allowed, for the sake of reporting metrics and
		// calculating max supervisors.
		//
		// Now that sch.maxConcurrency is set, we will only
		// raise it past len(running) by 10%.  This helps
		// avoid running an inappropriate number of
		// supervisors when we reach the cloud-imposed quota
		// (which may be based on # CPUs etc) long before the
		// configured MaxInstances.
		if sch.maxConcurrency == 0 || sch.maxConcurrency > totalInstances {
			if totalInstances == 0 {
				sch.maxConcurrency = 1
			} else {
				sch.maxConcurrency = totalInstances
			}
		}
		sch.instancesWithinQuota = totalInstances
	} else if sch.instancesWithinQuota > 0 && sch.maxConcurrency > sch.instancesWithinQuota+1 {
		// Once we've hit a quota error and started tracking
		// instancesWithinQuota (i.e., it's not zero), we
		// avoid exceeding that known-working level by more
		// than 1.
		//
		// If we don't do this, we risk entering a pattern of
		// repeatedly locking several containers, hitting
		// quota again, and unlocking them again each time the
		// driver stops reporting AtQuota, which tends to use
		// up the max lock/unlock cycles on the next few
		// containers in the queue, and cause them to fail.
		sch.maxConcurrency = sch.instancesWithinQuota + 1
	}
	sch.mMaxContainerConcurrency.Set(float64(sch.maxConcurrency))

	maxSupervisors := int(float64(sch.maxConcurrency) * sch.supervisorFraction)
	if maxSupervisors < 1 && sch.supervisorFraction > 0 && sch.maxConcurrency > 0 {
		maxSupervisors = 1
	}

	sch.logger.WithFields(logrus.Fields{
		"Containers":     len(sorted),
		"Processes":      len(running),
		"maxConcurrency": sch.maxConcurrency,
	}).Debug("runQueue")

	dontstart := map[arvados.InstanceType]bool{}
	var atcapacity = map[string]bool{} // ProviderTypes reported as AtCapacity during this runQueue() invocation
	var overquota []QueueEnt           // entries that are unmappable because of worker pool quota
	var overmaxsuper []QueueEnt        // unmappable because max supervisors (these are not included in overquota)
	var containerAllocatedWorkerBootingCount int

	// trying is #containers running + #containers we're trying to
	// start. We stop trying to start more containers if this
	// reaches the dynamic maxConcurrency limit.
	trying := len(running)

	qpos := 0
	supervisors := 0

tryrun:
	for i, ent := range sorted {
		ctr, types := ent.Container, ent.InstanceTypes
		logger := sch.logger.WithFields(logrus.Fields{
			"ContainerUUID": ctr.UUID,
		})
		if ctr.SchedulingParameters.Supervisor {
			supervisors += 1
		}
		if _, running := running[ctr.UUID]; running {
			if ctr.State == arvados.ContainerStateQueued || ctr.State == arvados.ContainerStateLocked {
				sorted[i].SchedulingStatus = "preparing runtime environment"
			}
			continue
		}
		if ctr.Priority < 1 {
			sorted[i].SchedulingStatus = "not scheduling: priority 0, state " + string(ctr.State)
			continue
		}
		if ctr.SchedulingParameters.Supervisor && maxSupervisors > 0 && supervisors > maxSupervisors {
			overmaxsuper = append(overmaxsuper, sorted[i])
			sorted[i].SchedulingStatus = "not starting: supervisor container limit has been reached"
			continue
		}
		// If we have unalloc instances of any of the eligible
		// instance types, unallocOK is true and unallocType
		// is the lowest-cost type.
		var unallocOK bool
		var unallocType arvados.InstanceType
		for _, it := range types {
			if unalloc[it] > 0 {
				unallocOK = true
				unallocType = it
				break
			}
		}
		// If the pool is not reporting AtCapacity for any of
		// the eligible instance types, availableOK is true
		// and availableType is the lowest-cost type.
		var availableOK bool
		var availableType arvados.InstanceType
		for _, it := range types {
			if atcapacity[it.ProviderType] {
				continue
			} else if sch.pool.AtCapacity(it) {
				atcapacity[it.ProviderType] = true
				continue
			} else {
				availableOK = true
				availableType = it
				break
			}
		}
		switch ctr.State {
		case arvados.ContainerStateQueued:
			if sch.maxConcurrency > 0 && trying >= sch.maxConcurrency {
				logger.Tracef("not locking: already at maxConcurrency %d", sch.maxConcurrency)
				continue
			}
			trying++
			if !unallocOK && sch.pool.AtQuota() {
				logger.Trace("not starting: AtQuota and no unalloc workers")
				overquota = sorted[i:]
				break tryrun
			}
			if !unallocOK && !availableOK {
				logger.Trace("not locking: AtCapacity and no unalloc workers")
				continue
			}
			if sch.pool.KillContainer(ctr.UUID, "about to lock") {
				logger.Info("not locking: crunch-run process from previous attempt has not exited")
				continue
			}
			go sch.lockContainer(logger, ctr.UUID)
			unalloc[unallocType]--
		case arvados.ContainerStateLocked:
			if sch.maxConcurrency > 0 && trying >= sch.maxConcurrency {
				logger.Tracef("not starting: already at maxConcurrency %d", sch.maxConcurrency)
				continue
			}
			trying++
			if unallocOK {
				// We have a suitable instance type,
				// so mark it as allocated, and try to
				// start the container.
				unalloc[unallocType]--
				logger = logger.WithField("InstanceType", unallocType.Name)
				if dontstart[unallocType] {
					// We already tried & failed to start
					// a higher-priority container on the
					// same instance type. Don't let this
					// one sneak in ahead of it.
				} else if sch.pool.KillContainer(ctr.UUID, "about to start") {
					sorted[i].SchedulingStatus = "waiting for previous attempt to exit"
					logger.Info("not restarting yet: crunch-run process from previous attempt has not exited")
				} else if sch.pool.StartContainer(unallocType, ctr) {
					sorted[i].SchedulingStatus = "preparing runtime environment"
					logger.Trace("StartContainer => true")
				} else {
					sorted[i].SchedulingStatus = "waiting for new instance to be ready"
					logger.Trace("StartContainer => false")
					containerAllocatedWorkerBootingCount += 1
					dontstart[unallocType] = true
				}
				continue
			}
			if sch.pool.AtQuota() {
				// Don't let lower-priority containers
				// starve this one by using keeping
				// idle workers alive on different
				// instance types.
				logger.Trace("overquota")
				overquota = sorted[i:]
				break tryrun
			}
			if !availableOK {
				// Continue trying lower-priority
				// containers in case they can run on
				// different instance types that are
				// available.
				//
				// The local "atcapacity" cache helps
				// when the pool's flag resets after
				// we look at container A but before
				// we look at lower-priority container
				// B. In that case we want to run
				// container A on the next call to
				// runQueue(), rather than run
				// container B now.
				qpos++
				sorted[i].SchedulingStatus = fmt.Sprintf("waiting for suitable instance type to become available: queue position %d", qpos)
				logger.Trace("all eligible types at capacity")
				continue
			}
			logger = logger.WithField("InstanceType", availableType.Name)
			if !sch.pool.Create(availableType) {
				// Failed despite not being at quota,
				// e.g., cloud ops throttled.
				logger.Trace("pool declined to create new instance")
				continue
			}
			// Success. (Note pool.Create works
			// asynchronously and does its own logging
			// about the eventual outcome, so we don't
			// need to.)
			sorted[i].SchedulingStatus = "waiting for new instance to be ready"
			logger.Info("creating new instance")
			// Don't bother trying to start the container
			// yet -- obviously the instance will take
			// some time to boot and become ready.
			containerAllocatedWorkerBootingCount += 1
			dontstart[availableType] = true
		}
	}

	sch.mContainersAllocatedNotStarted.Set(float64(containerAllocatedWorkerBootingCount))
	sch.mContainersNotAllocatedOverQuota.Set(float64(len(overquota) + len(overmaxsuper)))

	var qreason string
	if sch.pool.AtQuota() {
		qreason = "waiting for cloud resources"
	} else {
		qreason = "waiting while cluster is running at capacity"
	}
	for i, ent := range sorted {
		if ent.SchedulingStatus == "" && (ent.Container.State == arvados.ContainerStateQueued || ent.Container.State == arvados.ContainerStateLocked) {
			qpos++
			sorted[i].SchedulingStatus = fmt.Sprintf("%s: queue position %d", qreason, qpos)
		}
	}
	sch.lastQueue.Store(sorted)

	if len(overquota)+len(overmaxsuper) > 0 {
		// Unlock any containers that are unmappable while
		// we're at quota (but if they have already been
		// scheduled and they're loading docker images etc.,
		// let them run).
		var unlock []QueueEnt
		unlock = append(unlock, overmaxsuper...)
		if totalInstances > 0 && len(overquota) > 1 {
			// We don't unlock the next-in-line container
			// when at quota.  This avoids a situation
			// where our "at quota" state expires, we lock
			// the next container and try to create an
			// instance, the cloud provider still returns
			// a quota error, we unlock the container, and
			// we repeat this until the container reaches
			// its limit of lock/unlock cycles.
			unlock = append(unlock, overquota[1:]...)
		} else {
			// However, if totalInstances is 0 and we're
			// still getting quota errors, then the
			// next-in-line container is evidently not
			// possible to run, so we should let it
			// exhaust its lock/unlock cycles and
			// eventually cancel, to avoid starvation.
			unlock = append(unlock, overquota...)
		}
		for _, ctr := range unlock {
			ctr := ctr.Container
			_, toolate := running[ctr.UUID]
			if ctr.State == arvados.ContainerStateLocked && !toolate {
				logger := sch.logger.WithField("ContainerUUID", ctr.UUID)
				logger.Info("unlock because pool capacity is used by higher priority containers")
				err := sch.queue.Unlock(ctr.UUID)
				if err != nil {
					logger.WithError(err).Warn("error unlocking")
				}
			}
		}
	}
	if len(overquota) > 0 {
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
