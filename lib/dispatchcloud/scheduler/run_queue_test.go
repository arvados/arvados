// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/test"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/worker"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var (
	// arbitrary example container UUIDs
	uuids = func() (r []string) {
		for i := 0; i < 16; i++ {
			r = append(r, test.ContainerUUID(i))
		}
		return
	}()
)

type stubQuotaError struct {
	error
}

func (stubQuotaError) IsQuotaError() bool { return true }

type stubPool struct {
	notify    <-chan struct{}
	unalloc   map[arvados.InstanceType]int // idle+booting+unknown
	idle      map[arvados.InstanceType]int
	running   map[string]time.Time
	atQuota   bool
	canCreate int
	creates   []arvados.InstanceType
	starts    []string
	shutdowns int
	sync.Mutex
}

func (p *stubPool) AtQuota() bool               { return p.atQuota }
func (p *stubPool) Subscribe() <-chan struct{}  { return p.notify }
func (p *stubPool) Unsubscribe(<-chan struct{}) {}
func (p *stubPool) Running() map[string]time.Time {
	p.Lock()
	defer p.Unlock()
	r := map[string]time.Time{}
	for k, v := range p.running {
		r[k] = v
	}
	return r
}
func (p *stubPool) Unallocated() map[arvados.InstanceType]int {
	p.Lock()
	defer p.Unlock()
	r := map[arvados.InstanceType]int{}
	for it, n := range p.unalloc {
		r[it] = n
	}
	return r
}
func (p *stubPool) Create(it arvados.InstanceType) bool {
	p.Lock()
	defer p.Unlock()
	p.creates = append(p.creates, it)
	if p.canCreate < 1 {
		return false
	}
	p.canCreate--
	p.unalloc[it]++
	return true
}
func (p *stubPool) KillContainer(uuid string) {
	p.Lock()
	defer p.Unlock()
	delete(p.running, uuid)
}
func (p *stubPool) Shutdown(arvados.InstanceType) bool {
	p.shutdowns++
	return false
}
func (p *stubPool) CountWorkers() map[worker.State]int {
	p.Lock()
	defer p.Unlock()
	return map[worker.State]int{
		worker.StateBooting: len(p.unalloc) - len(p.idle),
		worker.StateIdle:    len(p.idle),
		worker.StateRunning: len(p.running),
	}
}
func (p *stubPool) StartContainer(it arvados.InstanceType, ctr arvados.Container) bool {
	p.Lock()
	defer p.Unlock()
	p.starts = append(p.starts, ctr.UUID)
	if p.idle[it] == 0 {
		return false
	}
	p.idle[it]--
	p.unalloc[it]--
	p.running[ctr.UUID] = time.Time{}
	return true
}

func chooseType(ctr *arvados.Container) (arvados.InstanceType, error) {
	return test.InstanceType(ctr.RuntimeConstraints.VCPUs), nil
}

var _ = check.Suite(&SchedulerSuite{})

type SchedulerSuite struct{}

// Assign priority=4 container to idle node. Create a new instance for
// the priority=3 container. Don't try to start any priority<3
// containers because priority=3 container didn't start
// immediately. Don't try to create any other nodes after the failed
// create.
func (*SchedulerSuite) TestUseIdleWorkers(c *check.C) {
	queue := test.Queue{
		ChooseType: chooseType,
		Containers: []arvados.Container{
			{
				UUID:     test.ContainerUUID(1),
				Priority: 1,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
			{
				UUID:     test.ContainerUUID(2),
				Priority: 2,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
			{
				UUID:     test.ContainerUUID(3),
				Priority: 3,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
			{
				UUID:     test.ContainerUUID(4),
				Priority: 4,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
		},
	}
	queue.Update()
	pool := stubPool{
		unalloc: map[arvados.InstanceType]int{
			test.InstanceType(1): 1,
			test.InstanceType(2): 2,
		},
		idle: map[arvados.InstanceType]int{
			test.InstanceType(1): 1,
			test.InstanceType(2): 2,
		},
		running:   map[string]time.Time{},
		canCreate: 0,
	}
	New(test.Logger(), &queue, &pool, time.Millisecond, time.Millisecond).runQueue()
	c.Check(pool.creates, check.DeepEquals, []arvados.InstanceType{test.InstanceType(1)})
	c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(4)})
	c.Check(pool.running, check.HasLen, 1)
	for uuid := range pool.running {
		c.Check(uuid, check.Equals, uuids[4])
	}
}

// If Create() fails, shutdown some nodes, and don't call Create()
// again.  Don't call Create() at all if AtQuota() is true.
func (*SchedulerSuite) TestShutdownAtQuota(c *check.C) {
	for quota := 0; quota < 2; quota++ {
		c.Logf("quota=%d", quota)
		shouldCreate := []arvados.InstanceType{}
		for i := 0; i < quota; i++ {
			shouldCreate = append(shouldCreate, test.InstanceType(3))
		}
		queue := test.Queue{
			ChooseType: chooseType,
			Containers: []arvados.Container{
				{
					UUID:     test.ContainerUUID(2),
					Priority: 2,
					State:    arvados.ContainerStateLocked,
					RuntimeConstraints: arvados.RuntimeConstraints{
						VCPUs: 2,
						RAM:   2 << 30,
					},
				},
				{
					UUID:     test.ContainerUUID(3),
					Priority: 3,
					State:    arvados.ContainerStateLocked,
					RuntimeConstraints: arvados.RuntimeConstraints{
						VCPUs: 3,
						RAM:   3 << 30,
					},
				},
			},
		}
		queue.Update()
		pool := stubPool{
			atQuota: quota == 0,
			unalloc: map[arvados.InstanceType]int{
				test.InstanceType(2): 2,
			},
			idle: map[arvados.InstanceType]int{
				test.InstanceType(2): 2,
			},
			running:   map[string]time.Time{},
			creates:   []arvados.InstanceType{},
			starts:    []string{},
			canCreate: 0,
		}
		New(test.Logger(), &queue, &pool, time.Millisecond, time.Millisecond).runQueue()
		c.Check(pool.creates, check.DeepEquals, shouldCreate)
		c.Check(pool.starts, check.DeepEquals, []string{})
		c.Check(pool.shutdowns, check.Not(check.Equals), 0)
	}
}

// Start lower-priority containers while waiting for new/existing
// workers to come up for higher-priority containers.
func (*SchedulerSuite) TestStartWhileCreating(c *check.C) {
	pool := stubPool{
		unalloc: map[arvados.InstanceType]int{
			test.InstanceType(1): 2,
			test.InstanceType(2): 2,
		},
		idle: map[arvados.InstanceType]int{
			test.InstanceType(1): 1,
			test.InstanceType(2): 1,
		},
		running:   map[string]time.Time{},
		canCreate: 4,
	}
	queue := test.Queue{
		ChooseType: chooseType,
		Containers: []arvados.Container{
			{
				// create a new worker
				UUID:     test.ContainerUUID(1),
				Priority: 1,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
			{
				// tentatively map to unalloc worker
				UUID:     test.ContainerUUID(2),
				Priority: 2,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
			{
				// start now on idle worker
				UUID:     test.ContainerUUID(3),
				Priority: 3,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
			{
				// create a new worker
				UUID:     test.ContainerUUID(4),
				Priority: 4,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 2,
					RAM:   2 << 30,
				},
			},
			{
				// tentatively map to unalloc worker
				UUID:     test.ContainerUUID(5),
				Priority: 5,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 2,
					RAM:   2 << 30,
				},
			},
			{
				// start now on idle worker
				UUID:     test.ContainerUUID(6),
				Priority: 6,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 2,
					RAM:   2 << 30,
				},
			},
		},
	}
	queue.Update()
	New(test.Logger(), &queue, &pool, time.Millisecond, time.Millisecond).runQueue()
	c.Check(pool.creates, check.DeepEquals, []arvados.InstanceType{test.InstanceType(2), test.InstanceType(1)})
	c.Check(pool.starts, check.DeepEquals, []string{uuids[6], uuids[5], uuids[3], uuids[2]})
	running := map[string]bool{}
	for uuid, t := range pool.running {
		if t.IsZero() {
			running[uuid] = false
		} else {
			running[uuid] = true
		}
	}
	c.Check(running, check.DeepEquals, map[string]bool{uuids[3]: false, uuids[6]: false})
}
