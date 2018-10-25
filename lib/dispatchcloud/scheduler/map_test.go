// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"errors"
	"fmt"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/container"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/test"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/worker"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Sirupsen/logrus"
	check "gopkg.in/check.v1"
)

var (
	logger = logrus.StandardLogger()

	// arbitrary example instance types
	types = func() (r []arvados.InstanceType) {
		for i := 0; i < 16; i++ {
			r = append(r, test.InstanceType(i))
		}
		return
	}()

	// arbitrary example container UUIDs
	uuids = func() (r []string) {
		for i := 0; i < 16; i++ {
			r = append(r, test.ContainerUUID(i))
		}
		return
	}()
)

type stubQueue struct {
	ents map[string]container.QueueEnt
}

func (q *stubQueue) Entries() map[string]container.QueueEnt {
	return q.ents
}
func (q *stubQueue) Lock(uuid string) error {
	return q.setState(uuid, arvados.ContainerStateLocked)
}
func (q *stubQueue) Unlock(uuid string) error {
	return q.setState(uuid, arvados.ContainerStateQueued)
}
func (q *stubQueue) Get(uuid string) (arvados.Container, bool) {
	ent, ok := q.ents[uuid]
	return ent.Container, ok
}
func (q *stubQueue) setState(uuid string, state arvados.ContainerState) error {
	ent, ok := q.ents[uuid]
	if !ok {
		return fmt.Errorf("no such ent: %q", uuid)
	}
	ent.Container.State = state
	q.ents[uuid] = ent
	return nil
}

type stubQuotaError struct {
	error
}

func (stubQuotaError) IsQuotaError() bool { return true }

type stubPool struct {
	notify    <-chan struct{}
	unalloc   map[arvados.InstanceType]int // idle+booting+unknown
	idle      map[arvados.InstanceType]int
	running   map[string]bool
	atQuota   bool
	canCreate int
	creates   []arvados.InstanceType
	starts    []string
	shutdowns int
}

func (p *stubPool) AtQuota() bool               { return p.atQuota }
func (p *stubPool) Subscribe() <-chan struct{}  { return p.notify }
func (p *stubPool) Unsubscribe(<-chan struct{}) {}
func (p *stubPool) Running() map[string]bool    { return p.running }
func (p *stubPool) Unallocated() map[arvados.InstanceType]int {
	r := map[arvados.InstanceType]int{}
	for it, n := range p.unalloc {
		r[it] = n
	}
	return r
}
func (p *stubPool) Create(it arvados.InstanceType) error {
	p.creates = append(p.creates, it)
	if p.canCreate < 1 {
		return stubQuotaError{errors.New("quota")}
	}
	p.canCreate--
	p.unalloc[it]++
	return nil
}
func (p *stubPool) Shutdown(arvados.InstanceType) bool {
	p.shutdowns++
	return false
}
func (p *stubPool) Workers() map[worker.State]int {
	return map[worker.State]int{
		worker.StateBooting: len(p.unalloc) - len(p.idle),
		worker.StateRunning: len(p.idle) - len(p.running),
	}
}
func (p *stubPool) StartContainer(it arvados.InstanceType, ctr arvados.Container) bool {
	p.starts = append(p.starts, ctr.UUID)
	if p.idle[it] == 0 {
		return false
	}
	p.idle[it]--
	p.unalloc[it]--
	p.running[ctr.UUID] = true
	return true
}

var _ = check.Suite(&SchedulerSuite{})

type SchedulerSuite struct{}

// Map priority=4 container to idle node. Create a new instance for
// the priority=3 container. Don't try to start any priority<3
// containers because priority=3 container didn't start
// immediately. Don't try to create any other nodes after the failed
// create.
func (*SchedulerSuite) TestMapIdle(c *check.C) {
	queue := stubQueue{
		ents: map[string]container.QueueEnt{
			uuids[1]: {
				Container:    arvados.Container{UUID: uuids[1], Priority: 1, State: arvados.ContainerStateQueued},
				InstanceType: types[1],
			},
			uuids[2]: {
				Container:    arvados.Container{UUID: uuids[2], Priority: 2, State: arvados.ContainerStateQueued},
				InstanceType: types[1],
			},
			uuids[3]: {
				Container:    arvados.Container{UUID: uuids[3], Priority: 3, State: arvados.ContainerStateQueued},
				InstanceType: types[1],
			},
			uuids[4]: {
				Container:    arvados.Container{UUID: uuids[4], Priority: 4, State: arvados.ContainerStateQueued},
				InstanceType: types[1],
			},
		},
	}
	pool := stubPool{
		unalloc: map[arvados.InstanceType]int{
			types[1]: 1,
			types[2]: 2,
		},
		idle: map[arvados.InstanceType]int{
			types[1]: 1,
			types[2]: 2,
		},
		running:   map[string]bool{},
		canCreate: 1,
	}
	Map(logger, &queue, &pool)
	c.Check(pool.creates, check.DeepEquals, []arvados.InstanceType{types[1]})
	c.Check(pool.starts, check.DeepEquals, []string{uuids[4], uuids[3]})
	c.Check(pool.running, check.DeepEquals, map[string]bool{uuids[4]: true})
}

// Shutdown some nodes if Create() fails -- and without even calling
// Create(), if AtQuota() is true.
func (*SchedulerSuite) TestMapShutdownAtQuota(c *check.C) {
	for quota := 0; quota < 2; quota++ {
		shouldCreate := types[1 : 1+quota]
		queue := stubQueue{
			ents: map[string]container.QueueEnt{
				uuids[1]: {
					Container:    arvados.Container{UUID: uuids[1], Priority: 1, State: arvados.ContainerStateQueued},
					InstanceType: types[1],
				},
			},
		}
		pool := stubPool{
			atQuota: quota == 0,
			unalloc: map[arvados.InstanceType]int{
				types[2]: 2,
			},
			idle: map[arvados.InstanceType]int{
				types[2]: 2,
			},
			running:   map[string]bool{},
			creates:   []arvados.InstanceType{},
			starts:    []string{},
			canCreate: 0,
		}
		Map(logger, &queue, &pool)
		c.Check(pool.creates, check.DeepEquals, shouldCreate)
		c.Check(pool.starts, check.DeepEquals, []string{})
		c.Check(pool.shutdowns, check.Not(check.Equals), 0)
	}
}

// Start lower-priority containers while waiting for new/existing
// workers to come up for higher-priority containers.
func (*SchedulerSuite) TestMapStartWhileCreating(c *check.C) {
	pool := stubPool{
		unalloc: map[arvados.InstanceType]int{
			types[1]: 1,
			types[2]: 1,
		},
		idle: map[arvados.InstanceType]int{
			types[1]: 1,
			types[2]: 1,
		},
		running:   map[string]bool{},
		canCreate: 2,
	}
	queue := stubQueue{
		ents: map[string]container.QueueEnt{
			uuids[1]: {
				// create a new worker
				Container:    arvados.Container{UUID: uuids[1], Priority: 1, State: arvados.ContainerStateQueued},
				InstanceType: types[1],
			},
			uuids[2]: {
				// tentatively map to unalloc worker
				Container:    arvados.Container{UUID: uuids[2], Priority: 2, State: arvados.ContainerStateQueued},
				InstanceType: types[1],
			},
			uuids[3]: {
				// start now on idle worker
				Container:    arvados.Container{UUID: uuids[3], Priority: 3, State: arvados.ContainerStateQueued},
				InstanceType: types[1],
			},
			uuids[4]: {
				// create a new worker
				Container:    arvados.Container{UUID: uuids[4], Priority: 4, State: arvados.ContainerStateQueued},
				InstanceType: types[2],
			},
			uuids[5]: {
				// tentatively map to unalloc worker
				Container:    arvados.Container{UUID: uuids[5], Priority: 5, State: arvados.ContainerStateQueued},
				InstanceType: types[2],
			},
			uuids[6]: {
				// start now on idle worker
				Container:    arvados.Container{UUID: uuids[6], Priority: 6, State: arvados.ContainerStateQueued},
				InstanceType: types[2],
			},
		},
	}
	Map(logger, &queue, &pool)
	c.Check(pool.creates, check.DeepEquals, []arvados.InstanceType{types[2], types[1]})
	c.Check(pool.starts, check.DeepEquals, []string{uuids[6], uuids[5], uuids[3], uuids[2]})
	c.Check(pool.running, check.DeepEquals, map[string]bool{uuids[3]: true, uuids[6]: true})
}
