// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"sort"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
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

type PoolSuite struct {
	logger      logrus.FieldLogger
	testCluster *arvados.Cluster
}

func (suite *PoolSuite) SetUpTest(c *check.C) {
	suite.logger = ctxlog.TestLogger(c)
	cfg, err := config.NewLoader(nil, suite.logger).Load()
	c.Assert(err, check.IsNil)
	suite.testCluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
}

func (suite *PoolSuite) TestResumeAfterRestart(c *check.C) {
	type1 := test.InstanceType(1)
	type2 := test.InstanceType(2)
	type3 := test.InstanceType(3)
	type4 := test.InstanceType(4)
	waitForIdle := func(pool *Pool, notify <-chan struct{}) {
		timeout := time.NewTimer(time.Second)
		for {
			instances := pool.Instances()
			sort.Slice(instances, func(i, j int) bool {
				return strings.Compare(instances[i].ArvadosInstanceType, instances[j].ArvadosInstanceType) < 0
			})
			if len(instances) == 4 &&
				instances[0].ArvadosInstanceType == type1.Name &&
				instances[0].WorkerState == StateIdle &&
				instances[1].ArvadosInstanceType == type1.Name &&
				instances[1].WorkerState == StateIdle &&
				instances[2].ArvadosInstanceType == type2.Name &&
				instances[2].WorkerState == StateIdle &&
				instances[3].ArvadosInstanceType == type4.Name &&
				instances[3].WorkerState == StateIdle {
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

	driver := &test.StubDriver{}
	instanceSetID := cloud.InstanceSetID("test-instance-set-id")
	is, err := driver.InstanceSet(nil, instanceSetID, nil, suite.logger, nil)
	c.Assert(err, check.IsNil)
	defer is.Stop()

	newExecutor := func(cloud.Instance) Executor {
		return &stubExecutor{
			response: map[string]stubResp{
				"crunch-run-custom --list": {},
				"true":                     {},
			},
		}
	}

	suite.testCluster.Containers.CloudVMs = arvados.CloudVMsConfig{
		BootProbeCommand:   "true",
		MaxProbesPerSecond: 1000,
		ProbeInterval:      arvados.Duration(time.Millisecond * 10),
		SyncInterval:       arvados.Duration(time.Millisecond * 10),
		TagKeyPrefix:       "testprefix:",
	}
	suite.testCluster.Containers.CrunchRunCommand = "crunch-run-custom"
	suite.testCluster.InstanceTypes = arvados.InstanceTypeMap{
		type1.Name: type1,
		type2.Name: type2,
		type3.Name: type3,
		type4.Name: type4,
	}

	pool := NewPool(suite.logger, arvados.NewClientFromEnv(), prometheus.NewRegistry(), instanceSetID, is, newExecutor, nil, suite.testCluster)
	notify := pool.Subscribe()
	defer pool.Unsubscribe(notify)
	pool.Create(type1)
	pool.Create(type1)
	pool.Create(type2)
	pool.Create(type4)
	waitForIdle(pool, notify)
	var heldInstanceID cloud.InstanceID
	for _, inst := range pool.Instances() {
		if inst.ArvadosInstanceType == type2.Name {
			heldInstanceID = cloud.InstanceID(inst.Instance)
			pool.SetIdleBehavior(heldInstanceID, IdleBehaviorHold)
		}
	}
	// Wait for the tags to save to the cloud provider
	tagKey := suite.testCluster.Containers.CloudVMs.TagKeyPrefix + tagKeyIdleBehavior
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

	delete(suite.testCluster.InstanceTypes, type4.Name)
	pool2 := NewPool(suite.logger, arvados.NewClientFromEnv(), prometheus.NewRegistry(), instanceSetID, is, newExecutor, nil, suite.testCluster)
	notify2 := pool2.Subscribe()
	defer pool2.Unsubscribe(notify2)
	waitForIdle(pool2, notify2)
	for _, inst := range pool2.Instances() {
		if inst.ArvadosInstanceType == type2.Name {
			c.Check(inst.Instance, check.Equals, heldInstanceID)
			c.Check(inst.IdleBehavior, check.Equals, IdleBehaviorHold)
		} else if inst.ArvadosInstanceType == type4.Name {
			// type4 instance is tagged IdleBehaviorRun,
			// but type4 was removed from config, so the
			// worker should be added with
			// IdleBehaviorDrain.
			c.Check(inst.IdleBehavior, check.Equals, IdleBehaviorDrain)
		} else {
			c.Check(inst.IdleBehavior, check.Equals, IdleBehaviorRun)
		}
	}
	pool2.Stop()
}

func (suite *PoolSuite) TestDrain(c *check.C) {
	driver := test.StubDriver{}
	instanceSet, err := driver.InstanceSet(nil, "test-instance-set-id", nil, suite.logger, nil)
	c.Assert(err, check.IsNil)
	defer instanceSet.Stop()

	ac := arvados.NewClientFromEnv()

	type1 := test.InstanceType(1)
	pool := &Pool{
		arvClient:   ac,
		logger:      suite.logger,
		newExecutor: func(cloud.Instance) Executor { return &stubExecutor{} },
		cluster:     suite.testCluster,
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
		started := pool.StartContainer(pool.Instances()[0].Instance, arvados.Container{UUID: "testcontainer"})
		c.Check(started, check.Equals, test.result)
	}
}

func (suite *PoolSuite) TestNodeCreateThrottle(c *check.C) {
	driver := test.StubDriver{HoldCloudOps: true}
	instanceSet, err := driver.InstanceSet(nil, "test-instance-set-id", nil, suite.logger, nil)
	c.Assert(err, check.IsNil)
	defer instanceSet.Stop()

	type1 := test.InstanceType(1)
	pool := &Pool{
		logger:                         suite.logger,
		instanceSet:                    &throttledInstanceSet{InstanceSet: instanceSet},
		cluster:                        suite.testCluster,
		maxConcurrentInstanceCreateOps: 1,
		instanceTypes: arvados.InstanceTypeMap{
			type1.Name: type1,
		},
	}

	c.Check(suite.instancesByType(pool, type1), check.HasLen, 0)
	_, res := pool.Create(type1)
	c.Check(suite.instancesByType(pool, type1), check.HasLen, 1)
	c.Check(res, check.Equals, true)

	_, res = pool.Create(type1)
	c.Check(suite.instancesByType(pool, type1), check.HasLen, 1)
	c.Check(res, check.Equals, false)

	pool.instanceSet.throttleCreate.err = nil
	pool.maxConcurrentInstanceCreateOps = 2

	_, res = pool.Create(type1)
	c.Check(suite.instancesByType(pool, type1), check.HasLen, 2)
	c.Check(res, check.Equals, true)

	pool.instanceSet.throttleCreate.err = nil
	pool.maxConcurrentInstanceCreateOps = 0

	_, res = pool.Create(type1)
	c.Check(suite.instancesByType(pool, type1), check.HasLen, 3)
	c.Check(res, check.Equals, true)
}

func (suite *PoolSuite) TestCreateUnallocShutdown(c *check.C) {
	driver := test.StubDriver{HoldCloudOps: true}
	instanceSet, err := driver.InstanceSet(nil, "test-instance-set-id", nil, suite.logger, nil)
	c.Assert(err, check.IsNil)
	defer instanceSet.Stop()

	type1 := arvados.InstanceType{Name: "a1s", ProviderType: "a1.small", VCPUs: 1, RAM: 1 * GiB, Price: .01}
	type2 := arvados.InstanceType{Name: "a2m", ProviderType: "a2.medium", VCPUs: 2, RAM: 2 * GiB, Price: .02}
	type3 := arvados.InstanceType{Name: "a2l", ProviderType: "a2.large", VCPUs: 4, RAM: 4 * GiB, Price: .04}
	pool := &Pool{
		logger:      suite.logger,
		newExecutor: func(cloud.Instance) Executor { return &stubExecutor{} },
		cluster:     suite.testCluster,
		instanceSet: &throttledInstanceSet{InstanceSet: instanceSet},
		instanceTypes: arvados.InstanceTypeMap{
			type1.Name: type1,
			type2.Name: type2,
			type3.Name: type3,
		},
		instanceInitCommand: "echo 'instance init command goes here'",
	}
	notify := pool.Subscribe()
	defer pool.Unsubscribe(notify)
	notify2 := pool.Subscribe()
	defer pool.Unsubscribe(notify2)

	c.Check(pool.Instances(), check.HasLen, 0)
	pool.Create(type2)
	pool.Create(type1)
	pool.Create(type2)
	pool.Create(type3)

	// Check the pending instances already appear in
	// pool.Instances() even though the cloud driver has not yet
	// responded to CreateInstance.
	c.Check(suite.instancesByType(pool, type1), check.HasLen, 1)
	c.Check(suite.instancesByType(pool, type2), check.HasLen, 2)
	c.Check(suite.instancesByType(pool, type3), check.HasLen, 1)

	// Unblock driver operations for the duration of the test.
	go driver.ReleaseCloudOps(4444)

	// Wait for each instance to either return from its Create
	// call, or show up in a poll.
	suite.wait(c, pool, notify, func() bool {
		pool.mtx.RLock()
		defer pool.mtx.RUnlock()
		return len(pool.workers) == 4
	})

	vms := instanceSet.(*test.StubInstanceSet).StubVMs()
	c.Check(string(vms[0].InitCommand), check.Matches, `umask 0177 && echo -n "[0-9a-f]+" >/var/run/arvados-instance-secret\necho 'instance init command goes here'`)

	// Place type3 node on admin-hold
	ivs := suite.instancesByType(pool, type3)
	c.Assert(ivs, check.HasLen, 1)
	type3instanceID := ivs[0].Instance
	err = pool.SetIdleBehavior(type3instanceID, IdleBehaviorHold)
	c.Check(err, check.IsNil)

	// Check admin-hold behavior: refuse to shutdown, and
	// Instances() reports IdleBehaviorHold.
	c.Check(pool.Shutdown(type3instanceID), check.Equals, false)
	suite.wait(c, pool, notify, func() bool {
		return suite.instancesByType(pool, type3)[0].IdleBehavior == IdleBehaviorHold
	})
	c.Check(suite.instancesByType(pool, type3), check.HasLen, 1)

	// Shutdown both type2 nodes
	for n, iv := range suite.instancesByType(pool, type2) {
		c.Check(pool.Shutdown(iv.Instance), check.Equals, true)
		suite.wait(c, pool, notify, func() bool {
			pool.getInstancesAndSync()
			return len(suite.instancesByType(pool, type1)) == 1 && len(suite.instancesByType(pool, type2)) == 1-n
		})
	}
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
	c.Check(pool.Shutdown(suite.instancesByType(pool, type1)[0].Instance), check.Equals, true)
	suite.wait(c, pool, notify, func() bool {
		pool.getInstancesAndSync()
		return len(suite.instancesByType(pool, type1)) == 0
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
		return suite.instancesByType(pool, type3)[0].IdleBehavior == IdleBehaviorRun
	})

	// Check admin-drain behavior: shut down right away.
	err = pool.SetIdleBehavior(type3instanceID, IdleBehaviorDrain)
	c.Check(err, check.IsNil)
	suite.wait(c, pool, notify, func() bool {
		ivs := suite.instancesByType(pool, type3)
		return len(ivs) == 1 && ivs[0].WorkerState == StateShutdown
	})

	// Sync until all instances disappear from the provider list.
	suite.wait(c, pool, notify, func() bool {
		pool.getInstancesAndSync()
		return len(pool.Instances()) == 0
	})
}

func (suite *PoolSuite) TestInstanceQuotaGroup(c *check.C) {
	driver := test.StubDriver{}
	instanceSet, err := driver.InstanceSet(nil, "test-instance-set-id", nil, suite.logger, nil)
	c.Assert(err, check.IsNil)
	defer instanceSet.Stop()

	// Note the stub driver uses the first character of
	// ProviderType as the instance family, so we have two
	// instance families here, "a" and "b".
	typeA1 := test.InstanceType(1)
	typeA1.ProviderType = "a1"
	typeA1p := test.InstanceType(1)
	typeA1p.Name += "-p"
	typeA1p.Preemptible = true
	typeA1p.ProviderType = "a1"
	typeA2 := test.InstanceType(2)
	typeA2.ProviderType = "a2"
	typeB3 := test.InstanceType(3)
	typeB3.ProviderType = "b3"
	typeB3p := test.InstanceType(3)
	typeB3p.Name += "-p"
	typeB3p.Preemptible = true
	typeB3p.ProviderType = "b3"
	typeB4 := test.InstanceType(4)
	typeB4.ProviderType = "b4"
	typeB4p := test.InstanceType(4)
	typeB4p.Name += "-p"
	typeB4p.Preemptible = true
	typeB4p.ProviderType = "b4"

	pool := &Pool{
		logger:      suite.logger,
		newExecutor: func(cloud.Instance) Executor { return &stubExecutor{} },
		cluster:     suite.testCluster,
		instanceSet: &throttledInstanceSet{InstanceSet: instanceSet},
		instanceTypes: arvados.InstanceTypeMap{
			typeA1.Name:  typeA1,
			typeA1p.Name: typeA1p,
			typeA2.Name:  typeA2,
			typeB3.Name:  typeB3,
			typeB4.Name:  typeB4,
		},
	}

	// Arrange for a quota-group-specific error on next
	// instanceSet.Create().
	driver.SetupVM = func(*test.StubVM) error { return test.CapacityError{InstanceQuotaGroupSpecific: true} }
	// pool.Create() returns true when it starts a goroutine to
	// call instanceSet.Create() in the background.
	_, created := pool.Create(typeA1)
	c.Check(created, check.Equals, true)
	// Wait for the pool to start reporting that the provider is
	// at capacity for instance type A1.
	for deadline := time.Now().Add(time.Second); !pool.AtCapacity(typeA1); time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			c.Fatal("timed out waiting for pool to report quota")
		}
	}

	// Arrange for a type-specific error on next
	// instanceSet.Create().
	driver.SetupVM = func(*test.StubVM) error { return test.CapacityError{InstanceTypeSpecific: true} }
	_, created = pool.Create(typeB4p)
	c.Check(created, check.Equals, true)
	for deadline := time.Now().Add(time.Second); !pool.AtCapacity(typeB4p); time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			c.Fatal("timed out waiting for pool to report quota")
		}
	}

	// The pool should now report AtCapacity for the affected
	// instance family (A1, A2) and specific instance type B4p,
	// and refuse to call instanceSet.Create() for those types --
	// but types A1p, B3, B4, and B3p are still usable.
	driver.SetupVM = func(*test.StubVM) error { return nil }
	c.Check(pool.AtCapacity(typeA1), check.Equals, true)
	c.Check(pool.AtCapacity(typeA1p), check.Equals, false)
	c.Check(pool.AtCapacity(typeA2), check.Equals, true)
	c.Check(pool.AtCapacity(typeB3), check.Equals, false)
	c.Check(pool.AtCapacity(typeB3p), check.Equals, false)
	c.Check(pool.AtCapacity(typeB4), check.Equals, false)
	c.Check(pool.AtCapacity(typeB4p), check.Equals, true)
	_, created = pool.Create(typeA2)
	c.Check(created, check.Equals, false)
	_, created = pool.Create(typeB3)
	c.Check(created, check.Equals, true)
	_, created = pool.Create(typeB3p)
	c.Check(created, check.Equals, true)
	_, created = pool.Create(typeB4)
	c.Check(created, check.Equals, true)
	_, created = pool.Create(typeB4p)
	c.Check(created, check.Equals, false)
	_, created = pool.Create(typeA2)
	c.Check(created, check.Equals, false)
	_, created = pool.Create(typeA1)
	c.Check(created, check.Equals, false)
	_, created = pool.Create(typeA1p)
	c.Check(created, check.Equals, true)
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
