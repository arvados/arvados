// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/lib/dispatchcloud/worker"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"

	"github.com/prometheus/client_golang/prometheus/testutil"

	check "gopkg.in/check.v1"
)

type stubPool struct {
	notify    <-chan struct{}
	workers   map[cloud.InstanceID]*worker.InstanceView
	running   map[string]time.Time
	quota     int
	capacity  map[string]int
	canCreate int
	creates   []arvados.InstanceType
	starts    []string
	shutdowns int
	sync.Mutex
}

func (p *stubPool) AtQuota() bool {
	n := 0
	for _, nn := range p.CountWorkers() {
		n += nn
	}
	return n >= p.quota
}
func (p *stubPool) AtCapacity(it arvados.InstanceType) bool {
	p.Lock()
	defer p.Unlock()
	supply, ok := p.capacity[it.ProviderType]
	if !ok {
		return false
	}
	for _, wkr := range p.workers {
		if wkr.ProviderInstanceType == it.ProviderType {
			supply--
		}
	}
	return supply < 1
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
func (p *stubPool) Instances() (r []worker.InstanceView) {
	p.Lock()
	defer p.Unlock()
	for _, wkr := range p.workers {
		r = append(r, *wkr)
	}
	return
}
func (p *stubPool) Create(it arvados.InstanceType) (worker.InstanceView, bool) {
	p.Lock()
	defer p.Unlock()
	p.creates = append(p.creates, it)
	if p.canCreate < 1 {
		return worker.InstanceView{}, false
	}
	p.canCreate--
	id := cloud.InstanceID(fmt.Sprintf("i-%07d", len(p.creates)))
	if p.workers == nil {
		p.workers = map[cloud.InstanceID]*worker.InstanceView{}
	}
	p.workers[id] = &worker.InstanceView{
		Instance:             id,
		Price:                it.Price,
		ArvadosInstanceType:  it.Name,
		ProviderInstanceType: it.ProviderType,
		WorkerState:          worker.StateBooting,
		IdleBehavior:         worker.IdleBehaviorRun,
	}
	// Returned InstanceView should have a blank instance ID, just
	// like a real pool (instances are created asynchronously so a
	// real cloud provider can't have provided an ID yet).
	created := *p.workers[id]
	created.Instance = ""
	return created, true
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
func (p *stubPool) Shutdown(cloud.InstanceID) bool {
	p.shutdowns++
	return false
}
func (p *stubPool) CountWorkers() map[worker.State]int {
	p.Lock()
	defer p.Unlock()
	r := map[worker.State]int{}
	for _, v := range p.workers {
		r[v.WorkerState]++
	}
	return r
}
func (p *stubPool) StartContainer(id cloud.InstanceID, ctr arvados.Container) bool {
	p.Lock()
	defer p.Unlock()
	p.starts = append(p.starts, ctr.UUID)
	wkr := p.workers[id]
	if wkr == nil {
		return false
	}
	if (wkr.WorkerState == worker.StateIdle || wkr.WorkerState == worker.StateRunning) &&
		wkr.IdleBehavior == worker.IdleBehaviorRun {
		if p.running == nil {
			p.running = map[string]time.Time{}
		}
		p.running[ctr.UUID] = time.Time{}
		wkr.WorkerState = worker.StateRunning
		wkr.LastContainerUUID = ctr.UUID
		wkr.RunningContainerUUIDs = append(wkr.RunningContainerUUIDs, ctr.UUID)
		return true
	}
	return false
}
func (p *stubPool) bootAllInstances() {
	p.Lock()
	defer p.Unlock()
	for _, wkr := range p.workers {
		if wkr.WorkerState == worker.StateBooting {
			wkr.WorkerState = worker.StateIdle
		}
	}
}

func chooseType(ctr *arvados.Container) ([]arvados.InstanceType, error) {
	return []arvados.InstanceType{test.InstanceType(ctr.RuntimeConstraints.VCPUs)}, nil
}

var _ = check.Suite(&SchedulerSuite{})

type SchedulerSuite struct {
	testCluster arvados.Cluster
}

func (s *SchedulerSuite) SetUpTest(c *check.C) {
	s.testCluster = arvados.Cluster{}
	s.testCluster.Containers.StaleLockTimeout = arvados.Duration(time.Millisecond)
	s.testCluster.Containers.CloudVMs.PollInterval = arvados.Duration(time.Millisecond)
	s.testCluster.Containers.CloudVMs.MaxInstances = 10
	s.testCluster.Containers.CloudVMs.SupervisorFraction = 0.2
	s.testCluster.InstanceTypes = make(arvados.InstanceTypeMap)
	for i := 1; i < 16; i++ {
		it := test.InstanceType(i)
		s.testCluster.InstanceTypes[it.Name] = it
	}
}

// Assign priority=4 container to idle node. Create new instances for
// the priority=3, 2, 1 containers.
func (s *SchedulerSuite) TestUseIdleWorkers(c *check.C) {
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
		quota:     1000,
		canCreate: 6,
	}
	pool.Create(test.InstanceType(1))
	pool.Create(test.InstanceType(2))
	pool.Create(test.InstanceType(2))
	pool.bootAllInstances()
	New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster).runQueue()
	c.Check(pool.creates, check.DeepEquals, []arvados.InstanceType{
		test.InstanceType(1), test.InstanceType(2), test.InstanceType(2),
		test.InstanceType(1), test.InstanceType(1), test.InstanceType(1),
	})
	c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(4)})
	c.Check(pool.running, check.HasLen, 1)
	for uuid := range pool.running {
		c.Check(uuid, check.Equals, test.ContainerUUID(4))
	}
}

// If pool.AtQuota() is true, shutdown some unalloc nodes, and don't
// call Create().
func (s *SchedulerSuite) TestShutdownAtQuota(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	for quota := 1; quota <= 3; quota++ {
		c.Logf("quota=%d", quota)
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
			quota:     quota,
			canCreate: 3,
		}
		pool.Create(test.InstanceType(2))
		pool.Create(test.InstanceType(2))
		pool.bootAllInstances()
		sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
		sch.sync()
		sch.runQueue()
		sch.sync()
		switch quota {
		case 1, 2:
			// Can't create a type3 node for ctr3, so we
			// shutdown the idle type2 nodes, and unlock
			// the 2nd-in-line container, but not the
			// 1st-in-line container.
			c.Check(pool.starts, check.HasLen, 0)
			c.Check(pool.shutdowns, check.Equals, 2)
			c.Check(pool.creates, check.HasLen, 2)
			c.Check(queue.StateChanges(), check.DeepEquals, []test.QueueStateChange{
				{UUID: test.ContainerUUID(2), From: "Locked", To: "Queued"},
			})
		case 3:
			// Creating a type3 instance works, so we
			// start ctr2 on a type2 instance, and leave
			// ctr3 locked while we wait for the new
			// instance to come up.
			c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(2)})
			c.Check(pool.shutdowns, check.Equals, 0)
			c.Check(pool.creates, check.DeepEquals, []arvados.InstanceType{
				test.InstanceType(2),
				test.InstanceType(2),
				test.InstanceType(3),
			})
			c.Check(queue.StateChanges(), check.HasLen, 0)
		default:
			panic("test not written for quota>3")
		}
	}
}

// If pool.AtCapacity(it) is true for one instance type, try running a
// lower-priority container that uses a different node type.  Don't
// lock/unlock/start any container that requires the affected instance
// type.
func (s *SchedulerSuite) TestInstanceCapacity(c *check.C) {
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
				State:    arvados.ContainerStateQueued,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 4,
					RAM:   4 << 30,
				},
			},
			{
				UUID:     test.ContainerUUID(3),
				Priority: 3,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 4,
					RAM:   4 << 30,
				},
			},
			{
				UUID:     test.ContainerUUID(4),
				Priority: 4,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 4,
					RAM:   4 << 30,
				},
			},
		},
	}
	queue.Update()
	pool := stubPool{
		quota:     99,
		capacity:  map[string]int{test.InstanceType(4).ProviderType: 1},
		canCreate: 2,
	}
	pool.Create(test.InstanceType(4))
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	sch.sync()
	sch.runQueue()
	sch.sync()

	// Start container4, but then pool reports AtCapacity for
	// type4, so we skip trying to create an instance for
	// container3, skip locking container2, but do try to create a
	// type1 instance for container1.
	c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(4)})
	c.Check(pool.shutdowns, check.Equals, 0)
	c.Check(pool.creates, check.DeepEquals, []arvados.InstanceType{test.InstanceType(4), test.InstanceType(1)})
	c.Check(queue.StateChanges(), check.HasLen, 0)
}

// Don't unlock containers or shutdown unalloc (booting/idle) nodes
// just because some 503 errors caused us to reduce maxContainers
// below the current load level.
//
// We expect to raise maxContainers soon when we stop seeing 503s. If
// that doesn't happen soon, the idle timeout will take care of the
// excess nodes.
func (s *SchedulerSuite) TestIdleIn503QuietPeriod(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	queue := test.Queue{
		ChooseType: chooseType,
		Containers: []arvados.Container{
			// scheduled on an instance (but not Running yet)
			{
				UUID:     test.ContainerUUID(1),
				Priority: 1000,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 2,
					RAM:   2 << 30,
				},
			},
			// not yet scheduled
			{
				UUID:     test.ContainerUUID(2),
				Priority: 1000,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 2,
					RAM:   2 << 30,
				},
			},
			// scheduled on an instance (but not Running yet)
			{
				UUID:     test.ContainerUUID(3),
				Priority: 1000,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 3,
					RAM:   3 << 30,
				},
			},
			// not yet scheduled
			{
				UUID:     test.ContainerUUID(4),
				Priority: 1000,
				State:    arvados.ContainerStateLocked,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 3,
					RAM:   3 << 30,
				},
			},
			// not yet locked
			{
				UUID:     test.ContainerUUID(5),
				Priority: 1000,
				State:    arvados.ContainerStateQueued,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 3,
					RAM:   3 << 30,
				},
			},
		},
	}
	queue.Update()
	pool := stubPool{
		quota:     16,
		creates:   []arvados.InstanceType{},
		starts:    []string{},
		canCreate: 4,
	}
	pool.Create(test.InstanceType(2))
	pool.Create(test.InstanceType(2))
	pool.Create(test.InstanceType(3))
	pool.Create(test.InstanceType(3))
	pool.bootAllInstances()
	instances := pool.Instances()
	sort.Slice(instances, func(i, j int) bool { return instances[i].Price < instances[j].Price })
	pool.StartContainer(instances[0].Instance, queue.Containers[0])
	pool.StartContainer(instances[2].Instance, queue.Containers[2])
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	sch.last503time = time.Now()
	sch.maxContainers = 3
	sch.sync()
	sch.runQueue()
	sch.sync()

	c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(1), test.ContainerUUID(3), test.ContainerUUID(2)})
	c.Check(pool.shutdowns, check.Equals, 0)
	c.Check(pool.creates, check.HasLen, 4)
	c.Check(queue.StateChanges(), check.HasLen, 0)
}

// If we somehow have more supervisor containers in Locked state than
// we should (e.g., config changed since they started), and some
// appropriate-sized instances booting up, unlock the excess
// supervisor containers, but let the instances keep booting.
func (s *SchedulerSuite) TestUnlockExcessSupervisors(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	queue := test.Queue{
		ChooseType: chooseType,
	}
	for i := 1; i <= 6; i++ {
		queue.Containers = append(queue.Containers, arvados.Container{
			UUID:     test.ContainerUUID(i),
			Priority: int64(1000 - i),
			State:    arvados.ContainerStateLocked,
			RuntimeConstraints: arvados.RuntimeConstraints{
				VCPUs: 2,
				RAM:   2 << 30,
			},
			SchedulingParameters: arvados.SchedulingParameters{
				Supervisor: true,
			},
		})
	}
	queue.Update()
	pool := stubPool{
		quota:     16,
		creates:   []arvados.InstanceType{},
		starts:    []string{},
		canCreate: 6,
	}
	for i := 0; i < 6; i++ {
		pool.Create(test.InstanceType(2))
	}
	pool.bootAllInstances()
	for i := 0; i < 4; i++ {
		pool.StartContainer(pool.Instances()[i].Instance, queue.Containers[i])
	}
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	sch.sync()
	sch.runQueue()
	sch.sync()

	c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(1), test.ContainerUUID(2), test.ContainerUUID(3), test.ContainerUUID(4)})
	c.Check(pool.shutdowns, check.Equals, 0)
	c.Check(pool.creates, check.HasLen, 6)
	c.Check(queue.StateChanges(), check.DeepEquals, []test.QueueStateChange{
		{UUID: test.ContainerUUID(5), From: "Locked", To: "Queued"},
		{UUID: test.ContainerUUID(6), From: "Locked", To: "Queued"},
	})
}

// Assuming we're not at quota, don't try to shutdown idle nodes
// merely because we have more queued/locked supervisor containers
// than MaxSupervisors -- it won't help.
func (s *SchedulerSuite) TestExcessSupervisors(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	queue := test.Queue{
		ChooseType: chooseType,
	}
	for i := 1; i <= 8; i++ {
		queue.Containers = append(queue.Containers, arvados.Container{
			UUID:     test.ContainerUUID(i),
			Priority: int64(1000 + i),
			State:    arvados.ContainerStateQueued,
			RuntimeConstraints: arvados.RuntimeConstraints{
				VCPUs: 2,
				RAM:   2 << 30,
			},
			SchedulingParameters: arvados.SchedulingParameters{
				Supervisor: true,
			},
		})
	}
	for i := 2; i < 4; i++ {
		queue.Containers[i].State = arvados.ContainerStateLocked
	}
	queue.Update()
	pool := stubPool{
		quota:     16,
		creates:   []arvados.InstanceType{},
		starts:    []string{},
		canCreate: 3,
	}
	pool.Create(test.InstanceType(2))
	pool.Create(test.InstanceType(2))
	pool.Create(test.InstanceType(2))
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	sch.sync()
	sch.runQueue()
	sch.sync()

	c.Check(pool.starts, check.HasLen, 2)
	c.Check(pool.shutdowns, check.Equals, 0)
	c.Check(pool.creates, check.HasLen, 3)
	c.Check(queue.StateChanges(), check.HasLen, 0)
}

// Don't flap lock/unlock when equal-priority containers compete for
// limited workers.
//
// (Unless we use FirstSeenAt as a secondary sort key, each runQueue()
// tends to choose a different one of the equal-priority containers as
// the "first" one that should be locked, and unlock the one it chose
// last time. This generates logging noise, and fails containers by
// reaching MaxDispatchAttempts quickly.)
func (s *SchedulerSuite) TestEqualPriorityContainers(c *check.C) {
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
		quota:     2,
		creates:   []arvados.InstanceType{},
		starts:    []string{},
		canCreate: 2,
	}
	pool.Create(test.InstanceType(3))
	pool.Create(test.InstanceType(3))
	pool.bootAllInstances()
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	for i := 0; i < 30; i++ {
		sch.runQueue()
		sch.sync()
		time.Sleep(time.Millisecond)
	}
	c.Check(pool.shutdowns, check.Equals, 0)
	c.Check(pool.starts, check.HasLen, 2)
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
func (s *SchedulerSuite) TestStartWhileCreating(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
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

	pool := stubPool{
		quota:     1000,
		canCreate: 4,
	}
	pool.Create(test.InstanceType(1))
	pool.Create(test.InstanceType(1))
	pool.workers["i-0000002"].WorkerState = worker.StateIdle
	pool.Create(test.InstanceType(2))
	pool.Create(test.InstanceType(2))
	pool.workers["i-0000004"].WorkerState = worker.StateIdle

	New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster).runQueue()
	c.Check(pool.creates, check.HasLen, 6)
	c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(6), test.ContainerUUID(5), test.ContainerUUID(3), test.ContainerUUID(2)})
	running := map[string]bool{}
	for uuid, t := range pool.running {
		if t.IsZero() {
			running[uuid] = false
		} else {
			running[uuid] = true
		}
	}
	c.Check(running, check.DeepEquals, map[string]bool{test.ContainerUUID(3): false, test.ContainerUUID(6): false})
}

func (s *SchedulerSuite) TestKillNonexistentContainer(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
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
	pool := stubPool{
		quota:     1000,
		canCreate: 1,
	}
	queue.Update()
	pool.Create(test.InstanceType(2))
	pool.bootAllInstances()
	pool.StartContainer("i-0000001", arvados.Container{UUID: test.ContainerUUID(2)})
	c.Check(pool.Running(), check.HasLen, 1)
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	c.Check(pool.running, check.HasLen, 1)
	sch.sync()
	for deadline := time.Now().Add(time.Second); len(pool.Running()) > 0 && time.Now().Before(deadline); time.Sleep(time.Millisecond) {
	}
	c.Check(pool.Running(), check.HasLen, 0)
}

func (s *SchedulerSuite) TestContainersMetrics(c *check.C) {
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

	// Create a pool with one worker in booting state.  The
	// container will be allocated but not started yet.
	pool := stubPool{
		canCreate: 1,
	}
	_, ok := pool.Create(test.InstanceType(1))
	c.Assert(ok, check.Equals, true)
	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	sch.runQueue()
	sch.updateMetrics()

	c.Check(int(testutil.ToFloat64(sch.mContainersAllocatedNotStarted)), check.Equals, 1)
	c.Check(int(testutil.ToFloat64(sch.mContainersNotAllocatedOverQuota)), check.Equals, 0)
	c.Check(int(testutil.ToFloat64(sch.mLongestWaitTimeSinceQueue)), check.Equals, 10)

	// Create a pool without workers. The queued container will
	// not be started, and the 'over quota' metric will be 1
	// because no workers are available and canCreate defaults to
	// zero.
	pool = stubPool{}
	sch = New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	sch.runQueue()
	sch.updateMetrics()

	c.Check(int(testutil.ToFloat64(sch.mContainersAllocatedNotStarted)), check.Equals, 0)
	c.Check(int(testutil.ToFloat64(sch.mContainersNotAllocatedOverQuota)), check.Equals, 1)
	c.Check(int(testutil.ToFloat64(sch.mLongestWaitTimeSinceQueue)), check.Equals, 10)

	// Reset the queue, and create a pool with an idle worker. The
	// queued container will be started immediately and
	// mLongestWaitTimeSinceQueue should be zero.
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
		canCreate: 1,
	}
	pool.Create(test.InstanceType(1))
	pool.bootAllInstances()
	sch = New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster)
	sch.runQueue()
	sch.updateMetrics()

	c.Check(int(testutil.ToFloat64(sch.mLongestWaitTimeSinceQueue)), check.Equals, 0)
}

// Assign priority=4, 3 and 1 containers to idle nodes. Ignore the
// supervisor at priority 2.
func (s *SchedulerSuite) TestSkipSupervisors(c *check.C) {
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
				SchedulingParameters: arvados.SchedulingParameters{
					Supervisor: true,
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
				SchedulingParameters: arvados.SchedulingParameters{
					Supervisor: true,
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
				SchedulingParameters: arvados.SchedulingParameters{
					Supervisor: true,
				},
			},
		},
	}
	queue.Update()
	pool := stubPool{
		quota:     1000,
		running:   map[string]time.Time{},
		canCreate: 8,
	}
	for i := 0; i < 8; i++ {
		pool.Create(test.InstanceType(i/4 + 1))
	}
	New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, &s.testCluster).runQueue()
	c.Check(pool.creates, check.HasLen, 8)
	c.Check(pool.starts, check.DeepEquals, []string{test.ContainerUUID(4), test.ContainerUUID(3), test.ContainerUUID(1)})
}
