// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"context"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"

	"github.com/prometheus/client_golang/prometheus/testutil"

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
	unknown   map[arvados.InstanceType]int
	running   map[string]time.Time
	quota     int
	canCreate int
	creates   []arvados.InstanceType
	starts    []string
	shutdowns int
	sync.Mutex
}

func (p *stubPool) AtQuota() bool {
	p.Lock()
	defer p.Unlock()
	return len(p.unalloc)+len(p.running)+len(p.unknown) >= p.quota
}
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
		r[it] = n - p.unknown[it]
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
func (p *stubPool) ForgetContainer(uuid string) {
}
func (p *stubPool) KillContainer(uuid, reason string) bool {
	p.Lock()
	defer p.Unlock()
	defer delete(p.running, uuid)
	t, ok := p.running[uuid]
	return ok && t.IsZero()
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
		worker.StateUnknown: len(p.unknown),
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

// Assign priority=4 container to idle node. Create new instances for
// the priority=3, 2, 1 containers.
func (*SchedulerSuite) TestUseIdleWorkers(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
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
		quota: 1000,
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
	New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond).runQueue()
	c.Check(pool.creates, check.DeepEquals, []arvados.InstanceType{test.InstanceType(1), test.InstanceType(1), test.InstanceType(1)})
	c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(4)})
	c.Check(pool.running, check.HasLen, 1)
	for uuid := range pool.running {
		c.Check(uuid, check.Equals, uuids[4])
	}
}

// If pool.AtQuota() is true, shutdown some unalloc nodes, and don't
// call Create().
func (*SchedulerSuite) TestShutdownAtQuota(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	for quota := 1; quota < 3; quota++ {
		c.Logf("quota=%d", quota)
		shouldCreate := []arvados.InstanceType{}
		for i := 1; i < quota; i++ {
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
			quota: quota,
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
		sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond)
		sch.runQueue()
		sch.sync()
		sch.runQueue()
		sch.sync()
		c.Check(pool.creates, check.DeepEquals, shouldCreate)
		if len(shouldCreate) == 0 {
			c.Check(pool.starts, check.DeepEquals, []string{})
		} else {
			c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(2)})
		}
		c.Check(pool.shutdowns, check.Equals, 3-quota)
		c.Check(queue.StateChanges(), check.DeepEquals, []test.QueueStateChange{
			{UUID: "zzzzz-dz642-000000000000003", From: "Locked", To: "Queued"},
			{UUID: "zzzzz-dz642-000000000000002", From: "Locked", To: "Queued"},
		})
	}
}

// Don't flap lock/unlock when equal-priority containers compete for
// limited workers.
//
// (Unless we use FirstSeenAt as a secondary sort key, each runQueue()
// tends to choose a different one of the equal-priority containers as
// the "first" one that should be locked, and unlock the one it chose
// last time. This generates logging noise, and fails containers by
// reaching MaxDispatchAttempts quickly.)
func (*SchedulerSuite) TestEqualPriorityContainers(c *check.C) {
	logger := ctxlog.TestLogger(c)
	ctx := ctxlog.Context(context.Background(), logger)
	queue := test.Queue{
		ChooseType: chooseType,
		Logger:     logger,
	}
	for i := 0; i < 8; i++ {
		queue.Containers = append(queue.Containers, arvados.Container{
			UUID:     test.ContainerUUID(i),
			Priority: 333,
			State:    arvados.ContainerStateQueued,
			RuntimeConstraints: arvados.RuntimeConstraints{
				VCPUs: 3,
				RAM:   3 << 30,
			},
		})
	}
	queue.Update()
	pool := stubPool{
		quota: 2,
		unalloc: map[arvados.InstanceType]int{
			test.InstanceType(3): 1,
		},
		idle: map[arvados.InstanceType]int{
			test.InstanceType(3): 1,
		},
		running:   map[string]time.Time{},
		creates:   []arvados.InstanceType{},
		starts:    []string{},
		canCreate: 1,
	}
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond)
	for i := 0; i < 30; i++ {
		sch.runQueue()
		sch.sync()
		time.Sleep(time.Millisecond)
	}
	c.Check(pool.shutdowns, check.Equals, 0)
	c.Check(pool.starts, check.HasLen, 1)
	unlocked := map[string]int{}
	for _, chg := range queue.StateChanges() {
		if chg.To == arvados.ContainerStateQueued {
			unlocked[chg.UUID]++
		}
	}
	for uuid, count := range unlocked {
		c.Check(count, check.Equals, 1, check.Commentf("%s", uuid))
	}
}

// Start lower-priority containers while waiting for new/existing
// workers to come up for higher-priority containers.
func (*SchedulerSuite) TestStartWhileCreating(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	pool := stubPool{
		quota: 1000,
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
	New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond).runQueue()
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

func (*SchedulerSuite) TestKillNonexistentContainer(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	pool := stubPool{
		quota: 1000,
		unalloc: map[arvados.InstanceType]int{
			test.InstanceType(2): 0,
		},
		idle: map[arvados.InstanceType]int{
			test.InstanceType(2): 0,
		},
		running: map[string]time.Time{
			test.ContainerUUID(2): {},
		},
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
		},
	}
	queue.Update()
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond)
	c.Check(pool.running, check.HasLen, 1)
	sch.sync()
	for deadline := time.Now().Add(time.Second); len(pool.Running()) > 0 && time.Now().Before(deadline); time.Sleep(time.Millisecond) {
	}
	c.Check(pool.Running(), check.HasLen, 0)
}

func (*SchedulerSuite) TestContainersMetrics(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	queue := test.Queue{
		ChooseType: chooseType,
		Containers: []arvados.Container{
			{
				UUID:      test.ContainerUUID(1),
				Priority:  1,
				State:     arvados.ContainerStateLocked,
				CreatedAt: time.Now().Add(-10 * time.Second),
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
		},
	}
	queue.Update()

	// Create a pool with one unallocated (idle/booting/unknown) worker,
	// and `idle` and `unknown` not set (empty). Iow this worker is in the booting
	// state, and the container will be allocated but not started yet.
	pool := stubPool{
		unalloc: map[arvados.InstanceType]int{test.InstanceType(1): 1},
	}
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond)
	sch.runQueue()
	sch.updateMetrics()

	c.Check(int(testutil.ToFloat64(sch.mContainersAllocatedNotStarted)), check.Equals, 1)
	c.Check(int(testutil.ToFloat64(sch.mContainersNotAllocatedOverQuota)), check.Equals, 0)
	c.Check(int(testutil.ToFloat64(sch.mLongestWaitTimeSinceQueue)), check.Equals, 10)

	// Create a pool without workers. The queued container will not be started, and the
	// 'over quota' metric will be 1 because no workers are available and canCreate defaults
	// to zero.
	pool = stubPool{}
	sch = New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond)
	sch.runQueue()
	sch.updateMetrics()

	c.Check(int(testutil.ToFloat64(sch.mContainersAllocatedNotStarted)), check.Equals, 0)
	c.Check(int(testutil.ToFloat64(sch.mContainersNotAllocatedOverQuota)), check.Equals, 1)
	c.Check(int(testutil.ToFloat64(sch.mLongestWaitTimeSinceQueue)), check.Equals, 10)

	// Reset the queue, and create a pool with an idle worker. The queued
	// container will be started immediately and mLongestWaitTimeSinceQueue
	// should be zero.
	queue = test.Queue{
		ChooseType: chooseType,
		Containers: []arvados.Container{
			{
				UUID:      test.ContainerUUID(1),
				Priority:  1,
				State:     arvados.ContainerStateLocked,
				CreatedAt: time.Now().Add(-10 * time.Second),
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
		},
	}
	queue.Update()

	pool = stubPool{
		idle:    map[arvados.InstanceType]int{test.InstanceType(1): 1},
		unalloc: map[arvados.InstanceType]int{test.InstanceType(1): 1},
		running: map[string]time.Time{},
	}
	sch = New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond)
	sch.runQueue()
	sch.updateMetrics()

	c.Check(int(testutil.ToFloat64(sch.mLongestWaitTimeSinceQueue)), check.Equals, 0)
}
