// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/dispatch"
	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&IntegrationSuite{})
var _ = Suite(&StubbedSuite{})

type IntegrationSuite struct {
	disp  Dispatcher
	slurm slurmFake
}

func (s *IntegrationSuite) SetUpTest(c *C) {
	arvadostest.StartAPI()
	os.Setenv("ARVADOS_API_TOKEN", arvadostest.Dispatch1Token)
	s.disp = Dispatcher{}
	s.disp.cluster = &arvados.Cluster{}
	s.disp.setup()
	s.slurm = slurmFake{}
}

func (s *IntegrationSuite) TearDownTest(c *C) {
	arvadostest.ResetEnv()
	arvadostest.StopAPI()
}

type slurmFake struct {
	didBatch      [][]string
	didCancel     []string
	didRelease    []string
	didRenice     [][]string
	queue         string
	rejectNice10K bool
	// If non-nil, run this func during the 2nd+ call to Cancel()
	onCancel func()
	// Error returned by Batch()
	errBatch error
}

func (sf *slurmFake) Batch(script io.Reader, args []string) error {
	sf.didBatch = append(sf.didBatch, args)
	return sf.errBatch
}

func (sf *slurmFake) QueueCommand(args []string) *exec.Cmd {
	return exec.Command("echo", sf.queue)
}

func (sf *slurmFake) Release(name string) error {
	sf.didRelease = append(sf.didRelease, name)
	return nil
}

func (sf *slurmFake) Renice(name string, nice int64) error {
	sf.didRenice = append(sf.didRenice, []string{name, fmt.Sprintf("%d", nice)})
	if sf.rejectNice10K && nice > 10000 {
		return errors.New("scontrol: error: Invalid nice value, must be between -10000 and 10000")
	}
	return nil
}

func (sf *slurmFake) Cancel(name string) error {
	sf.didCancel = append(sf.didCancel, name)
	if len(sf.didCancel) == 1 {
		// simulate error on first attempt
		return errors.New("something terrible happened")
	}
	if sf.onCancel != nil {
		sf.onCancel()
	}
	return nil
}

func (s *IntegrationSuite) integrationTest(c *C,
	expectBatch [][]string,
	runContainer func(*dispatch.Dispatcher, arvados.Container)) (arvados.Container, error) {
	arvadostest.ResetEnv()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	// There should be one queued container
	params := arvadosclient.Dict{
		"filters": [][]string{{"state", "=", "Queued"}},
	}
	var containers arvados.ContainerList
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Assert(len(containers.Items), Equals, 1)

	s.disp.cluster.Containers.CrunchRunCommand = "echo"

	ctx, cancel := context.WithCancel(context.Background())
	doneRun := make(chan struct{})
	doneDispatch := make(chan error)

	s.disp.Dispatcher = &dispatch.Dispatcher{
		Arv:        arv,
		PollPeriod: time.Second,
		RunContainer: func(disp *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) error {
			go func() {
				runContainer(disp, ctr)
				s.slurm.queue = ""
				doneRun <- struct{}{}
			}()
			err := s.disp.runContainer(disp, ctr, status)
			cancel()
			doneDispatch <- err
			return nil
		},
	}

	s.disp.slurm = &s.slurm
	s.disp.sqCheck = &SqueueChecker{
		Logger: logrus.StandardLogger(),
		Period: 500 * time.Millisecond,
		Slurm:  s.disp.slurm,
	}

	err = s.disp.Dispatcher.Run(ctx)
	<-doneRun
	c.Assert(err, Equals, context.Canceled)
	errDispatch := <-doneDispatch

	s.disp.sqCheck.Stop()

	c.Check(s.slurm.didBatch, DeepEquals, expectBatch)

	// There should be no queued containers now
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Check(len(containers.Items), Equals, 0)

	// Previously "Queued" container should now be in "Complete" state
	var container arvados.Container
	err = arv.Get("containers", "zzzzz-dz642-queuedcontainer", nil, &container)
	c.Check(err, IsNil)
	return container, errDispatch
}

func (s *IntegrationSuite) TestNormal(c *C) {
	s.slurm = slurmFake{queue: "zzzzz-dz642-queuedcontainer 10000 100 PENDING Resources\n"}
	container, _ := s.integrationTest(c,
		nil,
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(3 * time.Second)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateComplete)
}

func (s *IntegrationSuite) TestCancel(c *C) {
	s.slurm = slurmFake{queue: "zzzzz-dz642-queuedcontainer 10000 100 PENDING Resources\n"}
	readyToCancel := make(chan bool)
	s.slurm.onCancel = func() { <-readyToCancel }
	container, _ := s.integrationTest(c,
		nil,
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(time.Second)
			dispatcher.Arv.Update("containers", container.UUID,
				arvadosclient.Dict{
					"container": arvadosclient.Dict{"priority": 0}},
				nil)
			readyToCancel <- true
			close(readyToCancel)
		})
	c.Check(container.State, Equals, arvados.ContainerStateCancelled)
	c.Check(len(s.slurm.didCancel) > 1, Equals, true)
	c.Check(s.slurm.didCancel[:2], DeepEquals, []string{"zzzzz-dz642-queuedcontainer", "zzzzz-dz642-queuedcontainer"})
}

func (s *IntegrationSuite) TestMissingFromSqueue(c *C) {
	container, _ := s.integrationTest(c,
		[][]string{{
			fmt.Sprintf("--job-name=%s", "zzzzz-dz642-queuedcontainer"),
			fmt.Sprintf("--nice=%d", 10000),
			"--no-requeue",
			fmt.Sprintf("--mem=%d", 11445),
			fmt.Sprintf("--cpus-per-task=%d", 4),
			fmt.Sprintf("--tmp=%d", 45777),
		}},
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(3 * time.Second)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateCancelled)
}

func (s *IntegrationSuite) TestSbatchFail(c *C) {
	s.slurm = slurmFake{errBatch: errors.New("something terrible happened")}
	container, err := s.integrationTest(c,
		[][]string{{"--job-name=zzzzz-dz642-queuedcontainer", "--nice=10000", "--no-requeue", "--mem=11445", "--cpus-per-task=4", "--tmp=45777"}},
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateComplete)
	c.Check(err, ErrorMatches, `something terrible happened`)
}

type StubbedSuite struct {
	disp Dispatcher
}

func (s *StubbedSuite) SetUpTest(c *C) {
	s.disp = Dispatcher{}
	s.disp.cluster = &arvados.Cluster{}
	s.disp.setup()
}

func (s *StubbedSuite) TestAPIErrorGettingContainers(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/api_client_authorizations/current"] = arvadostest.StubResponse{200, `{"uuid":"` + arvadostest.Dispatch1AuthUUID + `"}`}
	apiStubResponses["/arvados/v1/containers"] = arvadostest.StubResponse{500, string(`{}`)}

	s.testWithServerStub(c, apiStubResponses, "echo", "error getting count of containers")
}

func (s *StubbedSuite) testWithServerStub(c *C, apiStubResponses map[string]arvadostest.StubResponse, crunchCmd string, expected string) {
	apiStub := arvadostest.ServerStub{apiStubResponses}

	api := httptest.NewServer(&apiStub)
	defer api.Close()

	arv := &arvadosclient.ArvadosClient{
		Scheme:    "http",
		ApiServer: api.URL[7:],
		ApiToken:  "abc123",
		Client:    &http.Client{Transport: &http.Transport{}},
		Retries:   0,
	}

	buf := bytes.NewBuffer(nil)
	logrus.SetOutput(io.MultiWriter(buf, os.Stderr))
	defer logrus.SetOutput(os.Stderr)

	s.disp.cluster.Containers.CrunchRunCommand = "crunchCmd"

	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := dispatch.Dispatcher{
		Arv:        arv,
		PollPeriod: time.Second,
		RunContainer: func(disp *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) error {
			go func() {
				time.Sleep(time.Second)
				disp.UpdateState(ctr.UUID, dispatch.Running)
				disp.UpdateState(ctr.UUID, dispatch.Complete)
			}()
			s.disp.runContainer(disp, ctr, status)
			cancel()
			return nil
		},
	}

	go func() {
		for i := 0; i < 80 && !strings.Contains(buf.String(), expected); i++ {
			time.Sleep(100 * time.Millisecond)
		}
		cancel()
	}()

	err := dispatcher.Run(ctx)
	c.Assert(err, Equals, context.Canceled)

	c.Check(buf.String(), Matches, `(?ms).*`+expected+`.*`)
}

func (s *StubbedSuite) TestSbatchArgs(c *C) {
	container := arvados.Container{
		UUID:               "123",
		RuntimeConstraints: arvados.RuntimeConstraints{RAM: 250000000, VCPUs: 2},
		Priority:           1,
	}

	for _, defaults := range [][]string{
		nil,
		{},
		{"--arg1=v1", "--arg2"},
	} {
		c.Logf("%#v", defaults)
		s.disp.cluster.Containers.SLURM.SbatchArgumentsList = defaults

		args, err := s.disp.sbatchArgs(container)
		c.Check(args, DeepEquals, append(defaults, "--job-name=123", "--nice=10000", "--no-requeue", "--mem=239", "--cpus-per-task=2", "--tmp=0"))
		c.Check(err, IsNil)
	}
}

func (s *StubbedSuite) TestSbatchInstanceTypeConstraint(c *C) {
	container := arvados.Container{
		UUID:               "123",
		RuntimeConstraints: arvados.RuntimeConstraints{RAM: 250000000, VCPUs: 2},
		Priority:           1,
	}

	for _, trial := range []struct {
		types      map[string]arvados.InstanceType
		sbatchArgs []string
		err        error
	}{
		// Choose node type => use --constraint arg
		{
			types: map[string]arvados.InstanceType{
				"a1.tiny":   {Name: "a1.tiny", Price: 0.02, RAM: 128000000, VCPUs: 1},
				"a1.small":  {Name: "a1.small", Price: 0.04, RAM: 256000000, VCPUs: 2},
				"a1.medium": {Name: "a1.medium", Price: 0.08, RAM: 512000000, VCPUs: 4},
				"a1.large":  {Name: "a1.large", Price: 0.16, RAM: 1024000000, VCPUs: 8},
			},
			sbatchArgs: []string{"--constraint=instancetype=a1.medium"},
		},
		// No node types configured => no slurm constraint
		{
			types:      nil,
			sbatchArgs: []string{"--mem=239", "--cpus-per-task=2", "--tmp=0"},
		},
		// No node type is big enough => error
		{
			types: map[string]arvados.InstanceType{
				"a1.tiny": {Name: "a1.tiny", Price: 0.02, RAM: 128000000, VCPUs: 1},
			},
			err: dispatchcloud.ConstraintsNotSatisfiableError{},
		},
	} {
		c.Logf("%#v", trial)
		s.disp.cluster = &arvados.Cluster{InstanceTypes: trial.types}

		args, err := s.disp.sbatchArgs(container)
		c.Check(err == nil, Equals, trial.err == nil)
		if trial.err == nil {
			c.Check(args, DeepEquals, append([]string{"--job-name=123", "--nice=10000", "--no-requeue"}, trial.sbatchArgs...))
		} else {
			c.Check(len(err.(dispatchcloud.ConstraintsNotSatisfiableError).AvailableTypes), Equals, len(trial.types))
		}
	}
}

func (s *StubbedSuite) TestSbatchPartition(c *C) {
	container := arvados.Container{
		UUID:                 "123",
		RuntimeConstraints:   arvados.RuntimeConstraints{RAM: 250000000, VCPUs: 1},
		SchedulingParameters: arvados.SchedulingParameters{Partitions: []string{"blurb", "b2"}},
		Priority:             1,
	}

	args, err := s.disp.sbatchArgs(container)
	c.Check(args, DeepEquals, []string{
		"--job-name=123", "--nice=10000", "--no-requeue",
		"--mem=239", "--cpus-per-task=1", "--tmp=0",
		"--partition=blurb,b2",
	})
	c.Check(err, IsNil)
}

func (s *StubbedSuite) TestLoadLegacyConfig(c *C) {
	content := []byte(`
Client:
  APIHost: example.com
  AuthToken: abcdefg
  KeepServiceURIs:
    - https://example.com/keep1
    - https://example.com/keep2
SbatchArguments: ["--foo", "bar"]
PollPeriod: 12s
PrioritySpread: 42
CrunchRunCommand: ["x-crunch-run", "--cgroup-parent-subsystem=memory"]
ReserveExtraRAM: 12345
MinRetryPeriod: 13s
BatchSize: 99
`)
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		c.Error(err)
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write(content); err != nil {
		c.Error(err)
	}
	if err := tmpfile.Close(); err != nil {
		c.Error(err)

	}
	os.Setenv("ARVADOS_KEEP_SERVICES", "")
	err = s.disp.configure("crunch-dispatch-slurm", []string{"-config", tmpfile.Name()})
	c.Check(err, IsNil)

	c.Check(s.disp.cluster.Services.Controller.ExternalURL, Equals, arvados.URL{Scheme: "https", Host: "example.com", Path: "/"})
	c.Check(s.disp.cluster.SystemRootToken, Equals, "abcdefg")
	c.Check(s.disp.cluster.Containers.SLURM.SbatchArgumentsList, DeepEquals, []string{"--foo", "bar"})
	c.Check(s.disp.cluster.Containers.CloudVMs.PollInterval, Equals, arvados.Duration(12*time.Second))
	c.Check(s.disp.cluster.Containers.SLURM.PrioritySpread, Equals, int64(42))
	c.Check(s.disp.cluster.Containers.CrunchRunCommand, Equals, "x-crunch-run")
	c.Check(s.disp.cluster.Containers.CrunchRunArgumentsList, DeepEquals, []string{"--cgroup-parent-subsystem=memory"})
	c.Check(s.disp.cluster.Containers.ReserveExtraRAM, Equals, arvados.ByteSize(12345))
	c.Check(s.disp.cluster.Containers.MinRetryPeriod, Equals, arvados.Duration(13*time.Second))
	c.Check(s.disp.cluster.API.MaxItemsPerResponse, Equals, 99)
	c.Check(s.disp.cluster.Containers.SLURM.SbatchEnvironmentVariables, DeepEquals, map[string]string{
		"ARVADOS_KEEP_SERVICES": "https://example.com/keep1 https://example.com/keep2",
	})
	c.Check(os.Getenv("ARVADOS_KEEP_SERVICES"), Equals, "https://example.com/keep1 https://example.com/keep2")
}
