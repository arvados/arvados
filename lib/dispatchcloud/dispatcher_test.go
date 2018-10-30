// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/lib/dispatchcloud/test"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/Sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&DispatcherSuite{})

// fakeCloud provides an exec method that can be used as a
// test.StubExecFunc. It calls the provided makeVM func when called
// with a previously unseen instance ID. Calls to exec are passed on
// to the *fakeVM for the appropriate instance ID.
type fakeCloud struct {
	queue      *test.Queue
	makeVM     func(cloud.Instance) *fakeVM
	onComplete func(string)
	onCancel   func(string)
	vms        map[cloud.InstanceID]*fakeVM
	sync.Mutex
}

func (fc *fakeCloud) exec(inst cloud.Instance, command string, stdin io.Reader, stdout, stderr io.Writer) uint32 {
	fc.Lock()
	fvm, ok := fc.vms[inst.ID()]
	if !ok {
		if fc.vms == nil {
			fc.vms = make(map[cloud.InstanceID]*fakeVM)
		}
		fvm = fc.makeVM(inst)
		fc.vms[inst.ID()] = fvm
	}
	fc.Unlock()
	return fvm.exec(fc.queue, fc.onComplete, fc.onCancel, command, stdin, stdout, stderr)
}

// fakeVM is a fake VM with configurable delays and failure modes.
type fakeVM struct {
	boot                 time.Time
	broken               time.Time
	crunchRunMissing     bool
	crunchRunCrashRate   float64
	crunchRunDetachDelay time.Duration
	ctrExit              int
	running              map[string]bool
	completed            []string
	sync.Mutex
}

func (fvm *fakeVM) exec(queue *test.Queue, onComplete, onCancel func(uuid string), command string, stdin io.Reader, stdout, stderr io.Writer) uint32 {
	uuid := regexp.MustCompile(`.{5}-dz642-.{15}`).FindString(command)
	if eta := fvm.boot.Sub(time.Now()); eta > 0 {
		fmt.Fprintf(stderr, "stub is booting, ETA %s\n", eta)
		return 1
	}
	if !fvm.broken.IsZero() && fvm.broken.Before(time.Now()) {
		fmt.Fprintf(stderr, "cannot fork\n")
		return 2
	}
	if fvm.crunchRunMissing && strings.Contains(command, "crunch-run") {
		fmt.Fprint(stderr, "crunch-run: command not found\n")
		return 1
	}
	if strings.HasPrefix(command, "crunch-run --detach ") {
		fvm.Lock()
		if fvm.running == nil {
			fvm.running = map[string]bool{}
		}
		fvm.running[uuid] = true
		fvm.Unlock()
		time.Sleep(fvm.crunchRunDetachDelay)
		fmt.Fprintf(stderr, "starting %s\n", uuid)
		logger := logrus.WithField("ContainerUUID", uuid)
		logger.Printf("[test] starting crunch-run stub")
		go func() {
			crashluck := rand.Float64()
			ctr, ok := queue.Get(uuid)
			if !ok {
				logger.Print("[test] container not in queue")
				return
			}
			if crashluck > fvm.crunchRunCrashRate/2 {
				time.Sleep(time.Duration(rand.Float64()*20) * time.Millisecond)
				ctr.State = arvados.ContainerStateRunning
				queue.Notify(ctr)
			}

			time.Sleep(time.Duration(rand.Float64()*20) * time.Millisecond)
			fvm.Lock()
			_, running := fvm.running[uuid]
			fvm.Unlock()
			if !running {
				logger.Print("[test] container was killed")
				return
			}
			if crashluck < fvm.crunchRunCrashRate {
				logger.Print("[test] crashing crunch-run stub")
				if onCancel != nil && ctr.State == arvados.ContainerStateRunning {
					onCancel(uuid)
				}
			} else {
				ctr.State = arvados.ContainerStateComplete
				ctr.ExitCode = fvm.ctrExit
				queue.Notify(ctr)
				if onComplete != nil {
					onComplete(uuid)
				}
			}
			logger.Print("[test] exiting crunch-run stub")
			fvm.Lock()
			defer fvm.Unlock()
			delete(fvm.running, uuid)
		}()
		return 0
	}
	if command == "crunch-run --list" {
		fvm.Lock()
		defer fvm.Unlock()
		for uuid := range fvm.running {
			fmt.Fprintf(stdout, "%s\n", uuid)
		}
		return 0
	}
	if strings.HasPrefix(command, "crunch-run --kill ") {
		fvm.Lock()
		defer fvm.Unlock()
		if fvm.running[uuid] {
			delete(fvm.running, uuid)
		} else {
			fmt.Fprintf(stderr, "%s: container is not running\n", uuid)
		}
		return 0
	}
	if command == "true" {
		return 0
	}
	fmt.Fprintf(stderr, "%q: command not found", command)
	return 1
}

type DispatcherSuite struct {
	cluster     *arvados.Cluster
	instanceSet *test.LameInstanceSet
	stubDriver  *test.StubDriver
	disp        *dispatcher
}

func (s *DispatcherSuite) SetUpSuite(c *check.C) {
	if os.Getenv("ARVADOS_DEBUG") != "" {
		logrus.StandardLogger().SetLevel(logrus.DebugLevel)
	}
}

func (s *DispatcherSuite) SetUpTest(c *check.C) {
	dispatchpub, _ := test.LoadTestKey(c, "test/sshkey_dispatch")
	dispatchprivraw, err := ioutil.ReadFile("test/sshkey_dispatch")
	c.Assert(err, check.IsNil)

	_, hostpriv := test.LoadTestKey(c, "test/sshkey_vm")
	s.stubDriver = &test.StubDriver{
		Exec: func(inst cloud.Instance, command string, _ io.Reader, _, _ io.Writer) uint32 {
			c.Logf("stubDriver SSHExecFunc(%s, %q, ...)", inst, command)
			return 1
		},
		HostKey:        hostpriv,
		AuthorizedKeys: []ssh.PublicKey{dispatchpub},
	}

	s.cluster = &arvados.Cluster{
		CloudVMs: arvados.CloudVMs{
			Driver:          "test",
			SyncInterval:    arvados.Duration(10 * time.Millisecond),
			TimeoutIdle:     arvados.Duration(30 * time.Millisecond),
			TimeoutBooting:  arvados.Duration(30 * time.Millisecond),
			TimeoutProbe:    arvados.Duration(15 * time.Millisecond),
			TimeoutShutdown: arvados.Duration(5 * time.Millisecond),
		},
		Dispatch: arvados.Dispatch{
			PrivateKey:         dispatchprivraw,
			PollInterval:       arvados.Duration(5 * time.Millisecond),
			ProbeInterval:      arvados.Duration(5 * time.Millisecond),
			StaleLockTimeout:   arvados.Duration(5 * time.Millisecond),
			MaxProbesPerSecond: 1000,
		},
		InstanceTypes: arvados.InstanceTypeMap{
			test.InstanceType(1).Name:  test.InstanceType(1),
			test.InstanceType(2).Name:  test.InstanceType(2),
			test.InstanceType(3).Name:  test.InstanceType(3),
			test.InstanceType(4).Name:  test.InstanceType(4),
			test.InstanceType(6).Name:  test.InstanceType(6),
			test.InstanceType(8).Name:  test.InstanceType(8),
			test.InstanceType(16).Name: test.InstanceType(16),
		},
		NodeProfiles: map[string]arvados.NodeProfile{
			"*": {
				Controller:    arvados.SystemServiceInstance{Listen: os.Getenv("ARVADOS_API_HOST")},
				DispatchCloud: arvados.SystemServiceInstance{Listen: ":"},
			},
		},
	}
	s.disp = &dispatcher{Cluster: s.cluster}
	// Test cases can modify s.cluster before calling
	// initialize(), and then modify private state before calling
	// go run().
}

func (s *DispatcherSuite) TearDownTest(c *check.C) {
	s.disp.Close()
}

func (s *DispatcherSuite) TestDispatchToStubDriver(c *check.C) {
	drivers["test"] = s.stubDriver
	s.disp.setupOnce.Do(s.disp.initialize)
	queue := &test.Queue{
		ChooseType: func(ctr *arvados.Container) (arvados.InstanceType, error) {
			return ChooseInstanceType(s.cluster, ctr)
		},
	}
	for i := 0; i < 200; i++ {
		queue.Containers = append(queue.Containers, arvados.Container{
			UUID:     test.ContainerUUID(i + 1),
			State:    arvados.ContainerStateQueued,
			Priority: int64(i%20 + 1),
			RuntimeConstraints: arvados.RuntimeConstraints{
				RAM:   int64(i%3+1) << 30,
				VCPUs: i%8 + 1,
			},
		})
	}
	s.disp.queue = queue

	var mtx sync.Mutex
	done := make(chan struct{})
	waiting := map[string]struct{}{}
	for _, ctr := range queue.Containers {
		waiting[ctr.UUID] = struct{}{}
	}
	onComplete := func(uuid string) {
		mtx.Lock()
		defer mtx.Unlock()
		if _, ok := waiting[uuid]; !ok {
			c.Errorf("container completed twice: %s", uuid)
		}
		delete(waiting, uuid)
		if len(waiting) == 0 {
			close(done)
		}
	}
	n := 0
	fc := &fakeCloud{
		queue: queue,
		makeVM: func(inst cloud.Instance) *fakeVM {
			n++
			fvm := &fakeVM{
				boot:                 time.Now().Add(time.Duration(rand.Int63n(int64(5 * time.Millisecond)))),
				crunchRunDetachDelay: time.Duration(rand.Int63n(int64(10 * time.Millisecond))),
				ctrExit:              int(rand.Uint32() & 0x3),
			}
			switch n % 7 {
			case 0:
				fvm.broken = time.Now().Add(time.Duration(rand.Int63n(90)) * time.Millisecond)
			case 1:
				fvm.crunchRunMissing = true
			default:
				fvm.crunchRunCrashRate = 0.1
			}
			return fvm
		},
		onComplete: onComplete,
		onCancel:   onComplete,
	}
	s.stubDriver.Exec = fc.exec

	start := time.Now()
	go s.disp.run()
	err := s.disp.CheckHealth()
	c.Check(err, check.IsNil)

	select {
	case <-done:
		c.Logf("containers finished (%s), waiting for instances to shutdown and queue to clear", time.Since(start))
	case <-time.After(10 * time.Second):
		c.Fatalf("timed out; still waiting for %d containers: %q", len(waiting), waiting)
	}

	deadline := time.Now().Add(time.Second)
	for range time.NewTicker(10 * time.Millisecond).C {
		insts, err := s.stubDriver.InstanceSets()[0].Instances(nil)
		c.Check(err, check.IsNil)
		queue.Update()
		ents, _ := queue.Entries()
		if len(ents) == 0 && len(insts) == 0 {
			break
		}
		if time.Now().After(deadline) {
			c.Fatalf("timed out with %d containers (%v), %d instances (%+v)", len(ents), ents, len(insts), insts)
		}
	}
}

func (s *DispatcherSuite) TestInstancesAPI(c *check.C) {
	s.cluster.CloudVMs.TimeoutBooting = arvados.Duration(time.Second)
	drivers["test"] = s.stubDriver

	type instance struct {
		Instance             string
		WorkerState          string
		Price                float64
		LastContainerUUID    string
		ArvadosInstanceType  string
		ProviderInstanceType string
	}
	type instancesResponse struct {
		Items []instance
	}
	getInstances := func() instancesResponse {
		req := httptest.NewRequest("GET", "/arvados/v1/dispatch/instances", nil)
		resp := httptest.NewRecorder()
		s.disp.ServeHTTP(resp, req)
		var sr instancesResponse
		err := json.Unmarshal(resp.Body.Bytes(), &sr)
		c.Check(err, check.IsNil)
		return sr
	}

	sr := getInstances()
	c.Check(len(sr.Items), check.Equals, 0)

	ch := s.disp.pool.Subscribe()
	defer s.disp.pool.Unsubscribe(ch)
	err := s.disp.pool.Create(test.InstanceType(1))
	c.Check(err, check.IsNil)
	<-ch

	sr = getInstances()
	c.Assert(len(sr.Items), check.Equals, 1)
	c.Check(sr.Items[0].Instance, check.Matches, "stub.*")
	c.Check(sr.Items[0].WorkerState, check.Equals, "booting")
	c.Check(sr.Items[0].Price, check.Equals, 0.123)
	c.Check(sr.Items[0].LastContainerUUID, check.Equals, "")
	c.Check(sr.Items[0].ProviderInstanceType, check.Equals, test.InstanceType(1).ProviderType)
	c.Check(sr.Items[0].ArvadosInstanceType, check.Equals, test.InstanceType(1).Name)
}
