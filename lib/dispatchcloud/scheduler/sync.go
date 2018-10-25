// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/container"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Sirupsen/logrus"
)

// Sync resolves discrepancies between the queue and the pool:
//
// Lingering crunch-run processes for finalized and unlocked/requeued
// containers are killed.
//
// Locked containers whose crunch-run processes have exited are
// requeued.
//
// Running containers whose crunch-run processes have exited are
// cancelled.
//
// Sync must not be called concurrently with other calls to Map or
// Sync using the same queue or pool.
func Sync(logger logrus.FieldLogger, queue ContainerQueue, pool WorkerPool) {
	running := pool.Running()
	cancel := func(ent container.QueueEnt, reason string) {
		uuid := ent.Container.UUID
		logger := logger.WithField("ContainerUUID", uuid)
		logger.Infof("cancelling container because %s", reason)
		err := queue.Cancel(uuid)
		if err != nil {
			logger.WithError(err).Print("error cancelling container")
		}
	}
	kill := func(ent container.QueueEnt) {
		uuid := ent.Container.UUID
		logger := logger.WithField("ContainerUUID", uuid)
		logger.Debugf("killing crunch-run process because state=%q", ent.Container.State)
		pool.KillContainer(uuid)
	}
	qEntries, qUpdated := queue.Entries()
	for uuid, ent := range qEntries {
		exited, running := running[uuid]
		switch ent.Container.State {
		case arvados.ContainerStateRunning:
			if !running {
				cancel(ent, "not running on any worker")
			} else if !exited.IsZero() && qUpdated.After(exited) {
				cancel(ent, "state=\"Running\" after crunch-run exited")
			}
		case arvados.ContainerStateComplete, arvados.ContainerStateCancelled:
			if running {
				kill(ent)
			} else {
				logger.WithFields(logrus.Fields{
					"ContainerUUID": uuid,
					"State":         ent.Container.State,
				}).Info("container finished")
				queue.Forget(uuid)
			}
		case arvados.ContainerStateQueued:
			if running {
				kill(ent)
			}
		case arvados.ContainerStateLocked:
			if running && !exited.IsZero() && qUpdated.After(exited) {
				logger = logger.WithFields(logrus.Fields{
					"ContainerUUID": uuid,
					"Exited":        time.Since(exited).Seconds(),
				})
				logger.Infof("requeueing container because state=%q after crunch-run exited", ent.Container.State)
				err := queue.Unlock(uuid)
				if err != nil {
					logger.WithError(err).Info("error requeueing container")
				}
			}
		default:
			logger.WithField("ContainerUUID", uuid).Errorf("BUG: unexpected state %q", ent.Container.State)
		}
	}
}
