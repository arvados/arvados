// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Package scheduler uses a resizable worker pool to execute
// containers in priority order.
//
// Scheduler functions must not be called concurrently using the same
// queue or pool.
package scheduler

import (
	"sort"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/container"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Sirupsen/logrus"
)

// Map maps queued containers onto unallocated workers in priority
// order, creating new workers if needed. It locks containers that can
// be mapped onto existing/pending workers, and starts them if
// possible.
//
// Map unlocks any containers that are locked but can't be
// mapped. (For example, this happens when the cloud provider reaches
// quota/capacity and a previously mappable container's priority is
// surpassed by a newer container.)
//
// If it encounters errors while creating new workers, Map shuts down
// idle workers, in case they are consuming quota.
//
// Map should not be called without first calling FixStaleLocks.
func Map(logger logrus.FieldLogger, queue ContainerQueue, pool WorkerPool) {
	unsorted, _ := queue.Entries()
	sorted := make([]container.QueueEnt, 0, len(unsorted))
	for _, ent := range unsorted {
		sorted = append(sorted, ent)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Container.Priority > sorted[j].Container.Priority
	})

	running := pool.Running()
	unalloc := pool.Unallocated()

	logger.WithFields(logrus.Fields{
		"Containers": len(sorted),
		"Processes":  len(running),
	}).Debug("mapping")

	dontstart := map[arvados.InstanceType]bool{}
	var overquota []container.QueueEnt // entries that are unmappable because of worker pool quota

	for i, ctr := range sorted {
		ctr, it := ctr.Container, ctr.InstanceType
		logger := logger.WithFields(logrus.Fields{
			"ContainerUUID": ctr.UUID,
			"InstanceType":  it.Name,
		})
		if _, running := running[ctr.UUID]; running || ctr.Priority < 1 {
			continue
		}
		if ctr.State == arvados.ContainerStateQueued {
			logger.Debugf("locking")
			if unalloc[it] < 1 && pool.AtQuota() {
				overquota = sorted[i:]
				break
			}
			err := queue.Lock(ctr.UUID)
			if err != nil {
				logger.WithError(err).Warnf("lock error")
				unalloc[it]++
				continue
			}
			var ok bool
			ctr, ok = queue.Get(ctr.UUID)
			if !ok {
				logger.Error("(BUG?) container disappeared from queue after Lock succeeded")
				continue
			}
			if ctr.State != arvados.ContainerStateLocked {
				logger.Debugf("(race?) container has state=%q after Lock succeeded", ctr.State)
			}
		}
		if ctr.State != arvados.ContainerStateLocked {
			continue
		}
		if unalloc[it] < 1 {
			logger.Info("creating new instance")
			err := pool.Create(it)
			if err != nil {
				if _, ok := err.(cloud.QuotaError); !ok {
					logger.WithError(err).Warn("error creating worker")
				}
				queue.Unlock(ctr.UUID)
				// Don't let lower-priority containers
				// starve this one by using keeping
				// idle workers alive on different
				// instance types.  TODO: avoid
				// getting starved here if instances
				// of a specific type always fail.
				overquota = sorted[i:]
				break
			}
			unalloc[it]++
		}
		if dontstart[it] {
			// We already tried & failed to start a
			// higher-priority container on the same
			// instance type. Don't let this one sneak in
			// ahead of it.
		} else if pool.StartContainer(it, ctr) {
			unalloc[it]--
		} else {
			dontstart[it] = true
		}
	}

	if len(overquota) > 0 {
		// Unlock any containers that are unmappable while
		// we're at quota.
		for _, ctr := range overquota {
			ctr := ctr.Container
			if ctr.State == arvados.ContainerStateLocked {
				logger := logger.WithField("ContainerUUID", ctr.UUID)
				logger.Debug("unlock because pool capacity is used by higher priority containers")
				err := queue.Unlock(ctr.UUID)
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
			pool.Shutdown(it)
		}
	}
}
