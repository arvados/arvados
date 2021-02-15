// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"sort"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

const GiB arvados.ByteSize = 1 << 30

var _ = check.Suite(&PoolSuite{})

type lessChecker struct {
	*check.CheckerInfo
}

func (*lessChecker) Check(params []interface{}, names []string) (result bool, error string) {
	return params[0].(int) < params[1].(int), ""
}

var less = &lessChecker{&check.CheckerInfo{Name: "less", Params: []string{"obtained", "expected"}}}

type PoolSuite struct{}

func (suite *PoolSuite) TestResumeAfterRestart(c *check.C) {
	type1 := test.InstanceType(1)
	type2 := test.InstanceType(2)
	type3 := test.InstanceType(3)
	waitForIdle := func(pool *Pool, notify <-chan struct{}) {
		timeout := time.NewTimer(time.Second)
		for {
			instances := pool.Instances()
			sort.Slice(instances, func(i, j int) bool {
				return strings.Compare(instances[i].ArvadosInstanceType, instances[j].ArvadosInstanceType) < 0
			})
			if len(instances) == 3 &&
				instances[0].ArvadosInstanceType == type1.Name &&
				instances[0].WorkerState == StateIdle.String() &&
				instances[1].ArvadosInstanceType == type1.Name &&
				instances[1].WorkerState == StateIdle.String() &&
				instances[2].ArvadosInstanceType == type2.Name &&
				instances[2].WorkerState == StateIdle.String() {
				return
			}
			select {
			case <-timeout.C:
				c.Logf("pool.Instances() == %#v", instances)
				c.Error("timed out")
				return
			case <-notify:
			}
		}
	}

	logger := ctxlog.TestLogger(c)
	driver := &test.StubDriver{}
	instanceSetID := cloud.InstanceSetID("test-instance-set-id")
	is, err := driver.InstanceSet(nil, instanceSetID, nil, logger)
	c.Assert(err, check.IsNil)

	newExecutor := func(cloud.Instance) Executor {
		return &stubExecutor{
			response: map[string]stubResp{
				"crunch-run-custom --list": {},
				"true":                     {},
			},
		}
	}

	cluster := &arvados.Cluster{
		Containers: arvados.ContainersConfig{
			CloudVMs: arvados.CloudVMsConfig{
				BootProbeCommand:   "true",
				MaxProbesPerSecond: 1000,
				ProbeInterval:      arvados.Duration(time.Millisecond * 10),
				SyncInterval:       arvados.Duration(time.Millisecond * 10),
				TagKeyPrefix:       "testprefix:",
			},
			CrunchRunCommand: "crunch-run-custom",
		},
		InstanceTypes: arvados.InstanceTypeMap{
			type1.Name: type1,
			type2.Name: type2,
			type3.Name: type3,
		},
	}

	pool := NewPool(logger, arvados.NewClientFromEnv(), prometheus.NewRegistry(), instanceSetID, is, newExecutor, nil, cluster)
	notify := pool.Subscribe()
	defer pool.Unsubscribe(notify)
	pool.Create(type1)
	pool.Create(type1)
	pool.Create(type2)
	waitForIdle(pool, notify)
	var heldInstanceID cloud.InstanceID
	for _, inst := range pool.Instances() {
		if inst.ArvadosInstanceType == type2.Name {
			heldInstanceID = cloud.InstanceID(inst.Instance)
			pool.SetIdleBehavior(heldInstanceID, IdleBehaviorHold)
		}
	}
	// Wait for the tags to save to the cloud provider
	tagKey := cluster.Containers.CloudVMs.TagKeyPrefix + tagKeyIdleBehavior
	deadline := time.Now().Add(time.Second)
	for !func() bool {
		pool.mtx.RLock()
		defer pool.mtx.RUnlock()
		for _, wkr := range pool.workers {
			if wkr.instType == type2 {
				return wkr.instance.Tags()[tagKey] == string(IdleBehaviorHold)
			}
		}
		return false
	}() {
		if time.Now().After(deadline) {
			c.Fatal("timeout")
		}
		time.Sleep(time.Millisecond * 10)
	}
	pool.Stop()

	c.Log("------- starting new pool, waiting to recover state")

	pool2 := NewPool(logger, arvados.NewClientFromEnv(), prometheus.NewRegistry(), instanceSetID, is, newExecutor, nil, cluster)
	notify2 := pool2.Subscribe()
	defer pool2.Unsubscribe(notify2)
	waitForIdle(pool2, notify2)
	for _, inst := range pool2.Instances() {
		if inst.ArvadosInstanceType == type2.Name {
			c.Check(inst.Instance, check.Equals, heldInstanceID)
			c.Check(inst.IdleBehavior, check.Equals, IdleBehaviorHold)
		} else {
			c.Check(inst.IdleBehavior, check.Equals, IdleBehaviorRun)
		}
	}
	pool2.Stop()
}

func (suite *PoolSuite) TestDrain(c *check.C) {
	logger := ctxlog.TestLogger(c)
	driver := test.StubDriver{}
	instanceSet, err := driver.InstanceSet(nil, "test-instance-set-id", nil, logger)
	c.Assert(err, check.IsNil)

	ac := arvados.NewClientFromEnv()

	type1 := test.InstanceType(1)
	pool := &Pool{
		arvClient:   ac,
		logger:      logger,
		newExecutor: func(cloud.Instance) Executor { return &stubExecutor{} },
		instanceSet: &throttledInstanceSet{InstanceSet: instanceSet},
		instanceTypes: arvados.InstanceTypeMap{
			type1.Name: type1,
		},
	}
	notify := pool.Subscribe()
	defer pool.Unsubscribe(notify)

	pool.Create(type1)

	// Wait for the instance to either return from its Create
	// call, or show up in a poll.
	suite.wait(c, pool, notify, func() bool {
		pool.mtx.RLock()
		defer pool.mtx.RUnlock()
		return len(pool.workers) == 1
	})

	tests := []struct {
		state        State
		idleBehavior IdleBehavior
		result       bool
	}{
		{StateIdle, IdleBehaviorHold, false},
		{StateIdle, IdleBehaviorDrain, false},
		{StateIdle, IdleBehaviorRun, true},
	}

	for _, test := range tests {
		for _, wkr := range pool.workers {
			wkr.state = test.state
			wkr.idleBehavior = test.idleBehavior
		}

		// Try to start a container
		started := pool.StartContainer(type1, arvados.Container{UUID: "testcontainer"})
		c.Check(started, check.Equals, test.result)
	}
}

func (suite *PoolSuite) TestNodeCreateThrottle(c *check.C) {
	logger := ctxlog.TestLogger(c)
	driver := test.StubDriver{HoldCloudOps: true}
	instanceSet, err := driver.InstanceSet(nil, "test-instance-set-id", nil, logger)
	c.Assert(err, check.IsNil)

	type1 := test.InstanceType(1)
	pool := &Pool{
		logger:                         logger,
		instanceSet:                    &throttledInstanceSet{InstanceSet: instanceSet},
		maxConcurrentInstanceCreateOps: 1,
		instanceTypes: arvados.InstanceTypeMap{
			type1.Name: type1,
		},
	}

	c.Check(pool.Unallocated()[type1], check.Equals, 0)
	res := pool.Create(type1)
	c.Check(pool.Unallocated()[type1], check.Equals, 1)
	c.Check(res, check.Equals, true)

	res = pool.Create(type1)
	c.Check(pool.Unallocated()[type1], check.Equals, 1)
	c.Check(res, check.Equals, false)

	pool.instanceSet.throttleCreate.err = nil
	pool.maxConcurrentInstanceCreateOps = 2

	res = pool.Create(type1)
	c.Check(pool.Unallocated()[type1], check.Equals, 2)
	c.Check(res, check.Equals, true)

	pool.instanceSet.throttleCreate.err = nil
	pool.maxConcurrentInstanceCreateOps = 0

	res = pool.Create(type1)
	c.Check(pool.Unallocated()[type1], check.Equals, 3)
	c.Check(res, check.Equals, true)
}

func (suite *PoolSuite) TestCreateUnallocShutdown(c *check.C) {
	logger := ctxlog.TestLogger(c)
	driver := test.StubDriver{HoldCloudOps: true}
	instanceSet, err := driver.InstanceSet(nil, "test-instance-set-id", nil, logger)
	c.Assert(err, check.IsNil)

	type1 := arvados.InstanceType{Name: "a1s", ProviderType: "a1.small", VCPUs: 1, RAM: 1 * GiB, Price: .01}
	type2 := arvados.InstanceType{Name: "a2m", ProviderType: "a2.medium", VCPUs: 2, RAM: 2 * GiB, Price: .02}
	type3 := arvados.InstanceType{Name: "a2l", ProviderType: "a2.large", VCPUs: 4, RAM: 4 * GiB, Price: .04}
	pool := &Pool{
		logger:      logger,
		newExecutor: func(cloud.Instance) Executor { return &stubExecutor{} },
		instanceSet: &throttledInstanceSet{InstanceSet: instanceSet},
		instanceTypes: arvados.InstanceTypeMap{
			type1.Name: type1,
			type2.Name: type2,
			type3.Name: type3,
		},
	}
	notify := pool.Subscribe()
	defer pool.Unsubscribe(notify)
	notify2 := pool.Subscribe()
	defer pool.Unsubscribe(notify2)

	c.Check(pool.Unallocated()[type1], check.Equals, 0)
	c.Check(pool.Unallocated()[type2], check.Equals, 0)
	c.Check(pool.Unallocated()[type3], check.Equals, 0)
	pool.Create(type2)
	pool.Create(type1)
	pool.Create(type2)
	pool.Create(type3)
	c.Check(pool.Unallocated()[type1], check.Equals, 1)
	c.Check(pool.Unallocated()[type2], check.Equals, 2)
	c.Check(pool.Unallocated()[type3], check.Equals, 1)

	// Unblock the pending Create calls.
	go driver.ReleaseCloudOps(4)

	// Wait for each instance to either return from its Create
	// call, or show up in a poll.
	suite.wait(c, pool, notify, func() bool {
		pool.mtx.RLock()
		defer pool.mtx.RUnlock()
		return len(pool.workers) == 4
	})

	// Place type3 node on admin-hold
	ivs := suite.instancesByType(pool, type3)
	c.Assert(ivs, check.HasLen, 1)
	type3instanceID := ivs[0].Instance
	err = pool.SetIdleBehavior(type3instanceID, IdleBehaviorHold)
	c.Check(err, check.IsNil)

	// Check admin-hold behavior: refuse to shutdown, and don't
	// report as Unallocated ("available now or soon").
	c.Check(pool.Shutdown(type3), check.Equals, false)
	suite.wait(c, pool, notify, func() bool {
		return pool.Unallocated()[type3] == 0
	})
	c.Check(suite.instancesByType(pool, type3), check.HasLen, 1)

	// Shutdown both type2 nodes
	c.Check(pool.Shutdown(type2), check.Equals, true)
	suite.wait(c, pool, notify, func() bool {
		return pool.Unallocated()[type1] == 1 && pool.Unallocated()[type2] == 1
	})
	c.Check(pool.Shutdown(type2), check.Equals, true)
	suite.wait(c, pool, notify, func() bool {
		return pool.Unallocated()[type1] == 1 && pool.Unallocated()[type2] == 0
	})
	c.Check(pool.Shutdown(type2), check.Equals, false)
	for {
		// Consume any waiting notifications to ensure the
		// next one we get is from Shutdown.
		select {
		case <-notify:
			continue
		default:
		}
		break
	}

	// Shutdown type1 node
	c.Check(pool.Shutdown(type1), check.Equals, true)
	suite.wait(c, pool, notify, func() bool {
		return pool.Unallocated()[type1] == 0 && pool.Unallocated()[type2] == 0 && pool.Unallocated()[type3] == 0
	})
	select {
	case <-notify2:
	case <-time.After(time.Second):
		c.Error("notify did not receive")
	}

	// Put type3 node back in service.
	err = pool.SetIdleBehavior(type3instanceID, IdleBehaviorRun)
	c.Check(err, check.IsNil)
	suite.wait(c, pool, notify, func() bool {
		return pool.Unallocated()[type3] == 1
	})

	// Check admin-drain behavior: shut down right away, and don't
	// report as Unallocated.
	err = pool.SetIdleBehavior(type3instanceID, IdleBehaviorDrain)
	c.Check(err, check.IsNil)
	suite.wait(c, pool, notify, func() bool {
		return pool.Unallocated()[type3] == 0
	})
	suite.wait(c, pool, notify, func() bool {
		ivs := suite.instancesByType(pool, type3)
		return len(ivs) == 1 && ivs[0].WorkerState == StateShutdown.String()
	})

	// Unblock all pending Destroy calls. Pool calls Destroy again
	// if a node still appears in the provider list after a
	// previous attempt, so there might be more than 4 Destroy
	// calls to unblock.
	go driver.ReleaseCloudOps(4444)

	// Sync until all instances disappear from the provider list.
	suite.wait(c, pool, notify, func() bool {
		pool.getInstancesAndSync()
		return len(pool.Instances()) == 0
	})
}

func (suite *PoolSuite) instancesByType(pool *Pool, it arvados.InstanceType) []InstanceView {
	var ivs []InstanceView
	for _, iv := range pool.Instances() {
		if iv.ArvadosInstanceType == it.Name {
			ivs = append(ivs, iv)
		}
	}
	return ivs
}

func (suite *PoolSuite) wait(c *check.C, pool *Pool, notify <-chan struct{}, ready func() bool) {
	timeout := time.NewTimer(time.Second).C
	for !ready() {
		select {
		case <-notify:
			continue
		case <-timeout:
		}
		break
	}
	c.Check(ready(), check.Equals, true)
}
