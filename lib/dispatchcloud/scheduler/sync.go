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

// sync resolves discrepancies between the queue and the pool:
//
// Lingering crunch-run processes for finalized and unlocked/requeued
// containers are killed.
//
// Locked containers whose crunch-run processes have exited are
// requeued.
//
// Running containers whose crunch-run processes have exited are
// cancelled.
func (sch *Scheduler) sync() {
	running := sch.pool.Running()
	cancel := func(ent container.QueueEnt, reason string) {
		uuid := ent.Container.UUID
		logger := sch.logger.WithField("ContainerUUID", uuid)
		logger.Infof("cancelling container because %s", reason)
		err := sch.queue.Cancel(uuid)
		if err != nil {
			logger.WithError(err).Print("error cancelling container")
		}
	}
	kill := func(ent container.QueueEnt) {
		uuid := ent.Container.UUID
		logger := sch.logger.WithField("ContainerUUID", uuid)
		logger.Debugf("killing crunch-run process because state=%q", ent.Container.State)
		sch.pool.KillContainer(uuid)
	}
	qEntries, qUpdated := sch.queue.Entries()
	for uuid, ent := range qEntries {
		exited, running := running[uuid]
		switch ent.Container.State {
		case arvados.ContainerStateRunning:
			if !running {
				go cancel(ent, "not running on any worker")
			} else if !exited.IsZero() && qUpdated.After(exited) {
				go cancel(ent, "state=\"Running\" after crunch-run exited")
			}
		case arvados.ContainerStateComplete, arvados.ContainerStateCancelled:
			if running {
				// Kill crunch-run in case it's stuck;
				// nothing it does now will matter
				// anyway. If crunch-run has already
				// exited and we just haven't found
				// out about it yet, the only effect
				// of kill() will be to make the
				// worker available for the next
				// container.
				go kill(ent)
			} else {
				sch.logger.WithFields(logrus.Fields{
					"ContainerUUID": uuid,
					"State":         ent.Container.State,
				}).Info("container finished")
				sch.queue.Forget(uuid)
			}
		case arvados.ContainerStateQueued:
			if running {
				// Can happen if a worker returns from
				// a network outage and is still
				// preparing to run a container that
				// has already been unlocked/requeued.
				go kill(ent)
			}
		case arvados.ContainerStateLocked:
			if running && !exited.IsZero() && qUpdated.After(exited) {
				logger := sch.logger.WithFields(logrus.Fields{
					"ContainerUUID": uuid,
					"Exited":        time.Since(exited).Seconds(),
				})
				logger.Infof("requeueing container because state=%q after crunch-run exited", ent.Container.State)
				err := sch.queue.Unlock(uuid)
				if err != nil {
					logger.WithError(err).Info("error requeueing container")
				}
			}
		default:
			sch.logger.WithField("ContainerUUID", uuid).Errorf("BUG: unexpected state %q", ent.Container.State)
		}
	}
}
