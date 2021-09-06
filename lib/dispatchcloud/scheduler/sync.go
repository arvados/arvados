// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"fmt"

	"git.arvados.org/arvados.git/lib/dispatchcloud/container"
	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

var reportedUnexpectedState = false

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
	anyUnknownWorkers := sch.pool.CountWorkers()[worker.StateUnknown] > 0
	running := sch.pool.Running()
	qEntries, qUpdated := sch.queue.Entries()
	for uuid, ent := range qEntries {
		exited, running := running[uuid]
		switch ent.Container.State {
		case arvados.ContainerStateRunning:
			if !running {
				if !anyUnknownWorkers {
					go sch.cancel(uuid, "not running on any worker")
				}
			} else if !exited.IsZero() && qUpdated.After(exited) {
				go sch.cancel(uuid, "state=Running after crunch-run exited")
			} else if ent.Container.Priority == 0 {
				go sch.kill(uuid, "priority=0")
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
				go sch.kill(uuid, fmt.Sprintf("state=%s", ent.Container.State))
			} else {
				sch.logger.WithFields(logrus.Fields{
					"ContainerUUID": uuid,
					"State":         ent.Container.State,
				}).Info("container finished -- dropping from queue")
				sch.queue.Forget(uuid)
			}
		case arvados.ContainerStateQueued:
			if running {
				// Can happen if a worker returns from
				// a network outage and is still
				// preparing to run a container that
				// has already been unlocked/requeued.
				go sch.kill(uuid, fmt.Sprintf("state=%s", ent.Container.State))
			} else if ent.Container.Priority == 0 {
				sch.logger.WithFields(logrus.Fields{
					"ContainerUUID": uuid,
					"State":         ent.Container.State,
					"Priority":      ent.Container.Priority,
				}).Info("container on hold -- dropping from queue")
				sch.queue.Forget(uuid)
			}
		case arvados.ContainerStateLocked:
			if running && !exited.IsZero() && qUpdated.After(exited) {
				go sch.requeue(ent, "crunch-run exited")
			} else if running && exited.IsZero() && ent.Container.Priority == 0 {
				go sch.kill(uuid, "priority=0")
			} else if !running && ent.Container.Priority == 0 {
				go sch.requeue(ent, "priority=0")
			}
		default:
			if !reportedUnexpectedState {
				sch.logger.WithFields(logrus.Fields{
					"ContainerUUID": uuid,
					"State":         ent.Container.State,
				}).Error("BUG: unexpected state")
				reportedUnexpectedState = true
			}
		}
	}
	for uuid := range running {
		if _, known := qEntries[uuid]; !known {
			go sch.kill(uuid, "not in queue")
		}
	}
}

func (sch *Scheduler) cancel(uuid string, reason string) {
	if !sch.uuidLock(uuid, "cancel") {
		return
	}
	defer sch.uuidUnlock(uuid)
	logger := sch.logger.WithField("ContainerUUID", uuid)
	logger.Infof("cancelling container because %s", reason)
	err := sch.queue.Cancel(uuid)
	if err != nil {
		logger.WithError(err).Print("error cancelling container")
	}
}

func (sch *Scheduler) kill(uuid string, reason string) {
	if !sch.uuidLock(uuid, "kill") {
		return
	}
	defer sch.uuidUnlock(uuid)
	sch.pool.KillContainer(uuid, reason)
	sch.pool.ForgetContainer(uuid)
}

func (sch *Scheduler) requeue(ent container.QueueEnt, reason string) {
	uuid := ent.Container.UUID
	if !sch.uuidLock(uuid, "requeue") {
		return
	}
	defer sch.uuidUnlock(uuid)
	logger := sch.logger.WithFields(logrus.Fields{
		"ContainerUUID": uuid,
		"State":         ent.Container.State,
		"Priority":      ent.Container.Priority,
	})
	logger.Infof("requeueing locked container because %s", reason)
	err := sch.queue.Unlock(uuid)
	if err != nil {
		logger.WithError(err).Error("error requeueing container")
	}
}
