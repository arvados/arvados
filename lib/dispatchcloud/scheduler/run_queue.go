// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud/container"
	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
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

const (
	schedStatusPreparingRuntimeEnvironment = "Container is allocated to an instance and preparing to run."
	schedStatusPriorityZero                = "This container will not be scheduled to run because its priority is 0 and state is %v."
	schedStatusSupervisorLimitReached      = "Waiting in workflow queue at position %v.  Cluster is at capacity and cannot start any new workflows right now."
	schedStatusWaitingForPreviousAttempt   = "Waiting for previous container attempt to exit."
	schedStatusWaitingNewInstance          = "Waiting for a %v instance to boot and be ready to accept work."
	schedStatusWaitingInstanceType         = "Waiting in queue at position %v.  Cluster is at capacity for all eligible instance types (%v) and cannot start a new instance right now."
	schedStatusWaitingCloudResources       = "Waiting in queue at position %v.  Cluster is at cloud account limits and cannot start any new instances right now."
	schedStatusWaitingClusterCapacity      = "Waiting in queue at position %v.  Cluster is at capacity and cannot start any new instances right now."
)

func instanceResourcesForInstanceType(it arvados.InstanceType) container.InstanceResources {
	return container.InstanceResources{
		VCPUs:   it.VCPUs,
		RAM:     it.RAM,
		Scratch: it.Scratch,
		GPUs:    it.GPU.DeviceCount,
		GPUVRAM: it.GPU.VRAM,
	}
}

// Queue returns the sorted queue from the last scheduling iteration.
func (sch *Scheduler) Queue() []QueueEnt {
	ents, _ := sch.lastQueue.Load().([]QueueEnt)
	return ents
}

func (sch *Scheduler) instanceSort(a, b worker.InstanceView) bool {
	if c := a.Price - b.Price; c != 0 {
		return c < 0
	}
	ita := sch.cluster.InstanceTypes[a.ArvadosInstanceType]
	itb := sch.cluster.InstanceTypes[b.ArvadosInstanceType]
	if c := ita.VCPUs - itb.VCPUs; c != 0 {
		return c > 0
	}
	if c := ita.RAM - itb.RAM; c != 0 {
		return c > 0
	}
	if c := ita.Scratch - itb.Scratch; c != 0 {
		return c > 0
	}
	if c := strings.Compare(string(a.Instance), string(b.Instance)); c != 0 {
		return c < 0
	}
	return false
}

func (sch *Scheduler) runQueue() {
	running := sch.pool.Running()
	instances := sch.pool.Instances()
	sort.Slice(instances, func(i, j int) bool {
		return sch.instanceSort(instances[i], instances[j])
	})
	// instanceResources[i] tracks the remaining resources on
	// instances[i].  Resources consumed by already-running
	// containers are subtracted below.
	instanceResources := make([]container.InstanceResources, len(instances))
	for i, instance := range instances {
		it := sch.cluster.InstanceTypes[instance.ArvadosInstanceType]
		instanceResources[i] = instanceResourcesForInstanceType(it)
	}

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

	containers := map[string]*arvados.Container{}
	for i := range sorted {
		containers[sorted[i].Container.UUID] = &sorted[i].Container
	}
	for i := range instances {
		for _, uuid := range instances[i].RunningContainerUUIDs {
			if containers[uuid] == nil {
				// Container size unknown.  Assume the
				// instance has no capacity to spare.
				instanceResources[i] = container.InstanceResources{}
				break
			}
			rsc := container.InstanceResourcesNeeded(sch.cluster, containers[uuid])
			instanceResources[i] = instanceResources[i].Minus(rsc)
		}
	}

	if t := sch.client.Last503(); t.After(sch.last503time) {
		// API has sent an HTTP 503 response since last time
		// we checked. Use current #containers - 1 as
		// maxContainers, i.e., try to stay just below the
		// level where we see 503s.
		sch.last503time = t
		if newlimit := len(running) - 1; newlimit < 1 {
			sch.maxContainers = 1
		} else {
			sch.maxContainers = newlimit
		}
	} else if sch.maxContainers > 0 && time.Since(sch.last503time) > quietAfter503 {
		// If we haven't seen any 503 errors lately, raise
		// limit to ~10% beyond the current workload.
		//
		// As we use the added 10% to schedule more
		// containers, len(running) will increase and we'll
		// push the limit up further. Soon enough,
		// maxContainers will get high enough to schedule the
		// entire queue, hit pool quota, or get 503s again.
		max := len(running)*11/10 + 1
		if sch.maxContainers < max {
			sch.maxContainers = max
		}
	}
	if sch.last503time.IsZero() {
		sch.mLast503Time.Set(0)
	} else {
		sch.mLast503Time.Set(float64(sch.last503time.Unix()))
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
		// Now that sch.maxContainers is set, we will only
		// raise it past len(running) by 10%.  This helps
		// avoid running an inappropriate number of
		// supervisors when we reach the cloud-imposed quota
		// (which may be based on # CPUs etc) long before the
		// configured MaxInstances.
		if sch.maxContainers == 0 || sch.maxContainers > totalInstances {
			if totalInstances == 0 {
				sch.maxContainers = 1
			} else {
				sch.maxContainers = totalInstances
			}
		}
		sch.instancesWithinQuota = totalInstances
	} else if sch.instancesWithinQuota > 0 && sch.maxContainers > sch.instancesWithinQuota+1 {
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
		sch.maxContainers = sch.instancesWithinQuota + 1
	}
	sch.mMaxContainerConcurrency.Set(float64(sch.maxContainers))

	maxSupervisors := int(float64(sch.maxContainers) * sch.cluster.Containers.CloudVMs.SupervisorFraction)
	if maxSupervisors < 1 && sch.cluster.Containers.CloudVMs.SupervisorFraction > 0 && sch.maxContainers > 0 {
		maxSupervisors = 1
	}

	sch.logger.WithFields(logrus.Fields{
		"Containers":    len(sorted),
		"Processes":     len(running),
		"maxContainers": sch.maxContainers,
	}).Debug("runQueue")

	var atcapacity = map[string]bool{} // ProviderTypes reported as AtCapacity during this runQueue() invocation
	var overquota []QueueEnt           // entries that are unmappable because of worker pool quota
	var overmaxsuper []QueueEnt        // unmappable because max supervisors (these are not included in overquota)
	var containerAllocatedWorkerBootingCount int

	// trying is #containers running + #containers we're trying to
	// start. We stop trying to start more containers if this
	// reaches the dynamic maxContainers limit.
	trying := len(running)

	qpos := 0
	supervisors := 0

tryrun:
	for i, ent := range sorted {
		ctr, ctrResources, types := ent.Container, ent.InstanceResources, ent.InstanceTypes
		logger := sch.logger.WithFields(logrus.Fields{
			"ContainerUUID": ctr.UUID,
		})
		if ctr.SchedulingParameters.Supervisor {
			supervisors += 1
		}
		if _, running := running[ctr.UUID]; running {
			if ctr.State == arvados.ContainerStateQueued || ctr.State == arvados.ContainerStateLocked {
				sorted[i].SchedulingStatus = schedStatusPreparingRuntimeEnvironment
			}
			continue
		}
		if ctr.Priority < 1 {
			sorted[i].SchedulingStatus = fmt.Sprintf(schedStatusPriorityZero, string(ctr.State))
			continue
		}
		if ctr.SchedulingParameters.Supervisor && maxSupervisors > 0 && supervisors > maxSupervisors {
			overmaxsuper = append(overmaxsuper, sorted[i])
			sorted[i].SchedulingStatus = fmt.Sprintf(schedStatusSupervisorLimitReached, len(overmaxsuper))
			continue
		}
		typesM := map[string]bool{}
		for _, it := range types {
			typesM[it.Name] = true
		}
		// ready>=0 means instances[ready] is where we should
		// try to run ctr (it's one of the eligible instance
		// types, and has enough resources to accommodate
		// ctr).  ready<0 means we can't start ctr right now,
		// all we can do is request a new instance or just
		// wait.
		ready := -1
		for i, instance := range instances {
			switch {
			case instance.WorkerState != worker.StateUnknown &&
				instance.WorkerState != worker.StateRunning &&
				instance.WorkerState != worker.StateBooting &&
				instance.WorkerState != worker.StateIdle:
			case sch.cluster.Containers.CloudVMs.MaxRunningContainersPerInstance > 0 &&
				sch.cluster.Containers.CloudVMs.MaxRunningContainersPerInstance <= len(instance.RunningContainerUUIDs):
				// reached configured limit on #
				// containers per instance
			case !typesM[instance.ArvadosInstanceType]:
				// incompatible or too expensive
			case instanceResources[i].Less(ctrResources):
				// insufficient spare resources
				if instance.WorkerState == worker.StateIdle && len(instance.RunningContainerUUIDs) == 0 {
					// This should be impossible
					// -- it means the selected
					// node type is too small even
					// when idle.
					logger.Infof("BUG? insufficient resources on idle instance %s type %s for container %s: ir %+v ctrr %+v", instance.Instance, instance.ArvadosInstanceType, ctr.UUID, instanceResources[i], ctrResources)
				}
			case ready < 0:
				// first eligible instance found
				ready = i
			case len(instance.RunningContainerUUIDs) > len(instances[ready].RunningContainerUUIDs):
				// already found an eligible instance,
				// but this one has more containers
				// running, which we prefer (if
				// workload decreases we want some
				// busy nodes and some idle nodes so
				// the idle ones can shut down)
				ready = i
			case (instances[ready].WorkerState == worker.StateBooting ||
				instances[ready].WorkerState == worker.StateUnknown) &&
				(instance.WorkerState == worker.StateIdle ||
					instance.WorkerState == worker.StateRunning):
				// prefer an idle/running instance
				// over a (possibly lower-priced)
				// booting/unprobed instance
				ready = i
			}
		}
		if ready >= 0 {
			logger.Tracef("ready instance %s for container %s: instanceResources %v ctrResources %v", instances[ready].Instance, ctr.UUID, instanceResources[ready], ctrResources)
		}
		// If the pool is not reporting AtCapacity for any of
		// the eligible instance types, availableOK is true
		// and availableType is the lowest-cost type.
		var availableOK bool
		var availableType arvados.InstanceType
		for _, it := range types {
			capkey := fmt.Sprintf("%s, preemptible=%v", it.ProviderType, it.Preemptible)
			if atcapacity[capkey] {
				continue
			} else if sch.pool.AtCapacity(it) {
				atcapacity[capkey] = true
				continue
			} else {
				availableOK = true
				availableType = it
				break
			}
		}
		switch ctr.State {
		case arvados.ContainerStateQueued:
			if sch.maxContainers > 0 && trying >= sch.maxContainers {
				logger.Tracef("not locking: already at maxContainers %d", sch.maxContainers)
				continue
			}
			trying++
			if ready < 0 && sch.pool.AtQuota() {
				logger.Trace("not starting: AtQuota and no workers with capacity")
				overquota = sorted[i:]
				break tryrun
			}
			if ready < 0 && !availableOK {
				logger.Trace("not locking: AtCapacity and no workers with capacity")
				continue
			}
			if sch.pool.KillContainer(ctr.UUID, "about to lock") {
				logger.Info("not locking: crunch-run process from previous attempt has not exited")
				continue
			}
			go sch.lockContainer(logger, ctr.UUID)
			if ready >= 0 {
				instanceResources[ready] = instanceResources[ready].Minus(ctrResources)
				instances[ready].RunningContainerUUIDs = append(instances[ready].RunningContainerUUIDs, ctr.UUID)
			}
		case arvados.ContainerStateLocked:
			if sch.maxContainers > 0 && trying >= sch.maxContainers {
				logger.Tracef("not starting: already at maxContainers %d", sch.maxContainers)
				continue
			}
			trying++
			if ready >= 0 {
				// We have a suitable instance type,
				// so mark it as allocated, and try to
				// start the container.
				instanceResources[ready] = instanceResources[ready].Minus(ctrResources)
				instances[ready].RunningContainerUUIDs = append(instances[ready].RunningContainerUUIDs, ctr.UUID)
				logger = logger.WithFields(logrus.Fields{
					"Instance":     instances[ready].Instance,
					"InstanceType": instances[ready].ArvadosInstanceType,
				})
				if sch.pool.KillContainer(ctr.UUID, "about to start") {
					sorted[i].SchedulingStatus = schedStatusWaitingForPreviousAttempt
					logger.Info("not restarting yet: crunch-run process from previous attempt has not exited")
				} else if sch.pool.StartContainer(instances[ready].Instance, ctr) {
					sorted[i].SchedulingStatus = schedStatusPreparingRuntimeEnvironment
					logger.Trace("StartContainer => true")
				} else {
					sorted[i].SchedulingStatus = fmt.Sprintf(schedStatusWaitingNewInstance, instances[ready].ArvadosInstanceType)
					logger.Trace("StartContainer => false")
					containerAllocatedWorkerBootingCount += 1
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
				var typenames []string
				for _, tp := range types {
					typenames = append(typenames, tp.Name)
				}
				sorted[i].SchedulingStatus = fmt.Sprintf(schedStatusWaitingInstanceType, qpos, strings.Join(typenames, ", "))
				logger.Trace("all eligible types at capacity")
				continue
			}
			logger = logger.WithField("InstanceType", availableType.Name)
			newInstance, ok := sch.pool.Create(availableType)
			if !ok {
				// Failed despite not being at quota,
				// e.g., cloud ops throttled.
				logger.Trace("pool declined to create new instance")
				continue
			}
			// Success. (Note pool.Create works
			// asynchronously and does its own logging
			// about the eventual outcome, so we don't
			// need to.)
			sorted[i].SchedulingStatus = fmt.Sprintf(schedStatusWaitingNewInstance, availableType.Name)
			logger.Info("creating new instance")
			// Don't bother trying to start the container
			// yet -- obviously the instance will take
			// some time to boot and become ready.
			containerAllocatedWorkerBootingCount += 1

			// Insert new entry in instances and
			// instanceResources.
			idx := 0
			for ; idx < len(instances) && sch.instanceSort(instances[idx], newInstance); idx++ {
			}
			newInstance.RunningContainerUUIDs = append(newInstance.RunningContainerUUIDs, ctr.UUID)
			instances = slices.Insert(instances, idx, newInstance)
			instanceResources = slices.Insert(instanceResources, idx, instanceResourcesForInstanceType(availableType).Minus(ctrResources))
		}
	}

	sch.mContainersAllocatedNotStarted.Set(float64(containerAllocatedWorkerBootingCount))
	sch.mContainersNotAllocatedOverQuota.Set(float64(len(overquota) + len(overmaxsuper)))

	var qreason string
	if sch.pool.AtQuota() {
		qreason = schedStatusWaitingCloudResources
	} else {
		qreason = schedStatusWaitingClusterCapacity
	}
	for i, ent := range sorted {
		if ent.SchedulingStatus == "" && (ent.Container.State == arvados.ContainerStateQueued || ent.Container.State == arvados.ContainerStateLocked) {
			qpos++
			sorted[i].SchedulingStatus = fmt.Sprintf(qreason, qpos)
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
		for _, instance := range instances {
			if len(instance.RunningContainerUUIDs) == 0 {
				sch.pool.Shutdown(instance.Instance)
			}
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
