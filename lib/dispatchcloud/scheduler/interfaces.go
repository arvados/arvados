// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/container"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/worker"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

// A ContainerQueue is a set of containers that need to be started or
// stopped. Implemented by container.Queue and test stubs.
type ContainerQueue interface {
	Entries() (entries map[string]container.QueueEnt, updated time.Time)
	Lock(uuid string) error
	Unlock(uuid string) error
	Cancel(uuid string) error
	Forget(uuid string)
	Get(uuid string) (arvados.Container, bool)
	Subscribe() <-chan struct{}
	Unsubscribe(<-chan struct{})
	Update() error
}

// A WorkerPool asynchronously starts and stops worker VMs, and starts
// and stops containers on them. Implemented by worker.Pool and test
// stubs.
type WorkerPool interface {
	Running() map[string]time.Time
	Unallocated() map[arvados.InstanceType]int
	Workers() map[worker.State]int
	AtQuota() bool
	Create(arvados.InstanceType) error
	Shutdown(arvados.InstanceType) bool
	StartContainer(arvados.InstanceType, arvados.Container) bool
	KillContainer(uuid string)
	Subscribe() <-chan struct{}
	Unsubscribe(<-chan struct{})
}
