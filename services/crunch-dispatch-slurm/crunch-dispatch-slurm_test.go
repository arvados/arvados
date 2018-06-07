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
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
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
	s.disp.setup()
	s.slurm = slurmFake{}
}

func (s *IntegrationSuite) TearDownTest(c *C) {
	arvadostest.ResetEnv()
	arvadostest.StopAPI()
}

type slurmFake struct {
	didBatch   [][]string
	didCancel  []string
	didRelease []string
	didRenice  [][]string
	queue      string
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
	runContainer func(*dispatch.Dispatcher, arvados.Container)) arvados.Container {
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
	c.Check(len(containers.Items), Equals, 1)

	s.disp.CrunchRunCommand = []string{"echo"}

	ctx, cancel := context.WithCancel(context.Background())
	doneRun := make(chan struct{})

	s.disp.Dispatcher = &dispatch.Dispatcher{
		Arv:        arv,
		PollPeriod: time.Second,
		RunContainer: func(disp *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
			go func() {
				runContainer(disp, ctr)
				s.slurm.queue = ""
				doneRun <- struct{}{}
			}()
			s.disp.runContainer(disp, ctr, status)
			cancel()
		},
	}

	s.disp.slurm = &s.slurm
	s.disp.sqCheck = &SqueueChecker{Period: 500 * time.Millisecond, Slurm: s.disp.slurm}

	err = s.disp.Dispatcher.Run(ctx)
	<-doneRun
	c.Assert(err, Equals, context.Canceled)

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
	return container
}

func (s *IntegrationSuite) TestNormal(c *C) {
	s.slurm = slurmFake{queue: "zzzzz-dz642-queuedcontainer 10000 100 PENDING Resources\n"}
	container := s.integrationTest(c,
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
	container := s.integrationTest(c,
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
	container := s.integrationTest(c,
		[][]string{{
			fmt.Sprintf("--job-name=%s", "zzzzz-dz642-queuedcontainer"),
			fmt.Sprintf("--nice=%d", 10000),
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
	container := s.integrationTest(c,
		[][]string{{"--job-name=zzzzz-dz642-queuedcontainer", "--nice=10000", "--mem=11445", "--cpus-per-task=4", "--tmp=45777"}},
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateComplete)

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	var ll arvados.LogList
	err = arv.List("logs", arvadosclient.Dict{"filters": [][]string{
		{"object_uuid", "=", container.UUID},
		{"event_type", "=", "dispatch"},
	}}, &ll)
	c.Assert(err, IsNil)
	c.Assert(len(ll.Items), Equals, 1)
}

type StubbedSuite struct {
	disp Dispatcher
}

func (s *StubbedSuite) SetUpTest(c *C) {
	s.disp = Dispatcher{}
	s.disp.setup()
}

func (s *StubbedSuite) TestAPIErrorGettingContainers(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/api_client_authorizations/current"] = arvadostest.StubResponse{200, `{"uuid":"` + arvadostest.Dispatch1AuthUUID + `"}`}
	apiStubResponses["/arvados/v1/containers"] = arvadostest.StubResponse{500, string(`{}`)}

	s.testWithServerStub(c, apiStubResponses, "echo", "Error getting list of containers")
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
	log.SetOutput(io.MultiWriter(buf, os.Stderr))
	defer log.SetOutput(os.Stderr)

	s.disp.CrunchRunCommand = []string{crunchCmd}

	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := dispatch.Dispatcher{
		Arv:        arv,
		PollPeriod: time.Second,
		RunContainer: func(disp *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
			go func() {
				time.Sleep(time.Second)
				disp.UpdateState(ctr.UUID, dispatch.Running)
				disp.UpdateState(ctr.UUID, dispatch.Complete)
			}()
			s.disp.runContainer(disp, ctr, status)
			cancel()
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

func (s *StubbedSuite) TestNoSuchConfigFile(c *C) {
	err := s.disp.readConfig("/nosuchdir89j7879/8hjwr7ojgyy7")
	c.Assert(err, NotNil)
}

func (s *StubbedSuite) TestBadSbatchArgsConfig(c *C) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"SbatchArguments": "oops this is not a string array"}`))
	c.Check(err, IsNil)

	err = s.disp.readConfig(tmpfile.Name())
	c.Assert(err, NotNil)
}

func (s *StubbedSuite) TestNoSuchArgInConfigIgnored(c *C) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"NoSuchArg": "Nobody loves me, not one tiny hunk."}`))
	c.Check(err, IsNil)

	err = s.disp.readConfig(tmpfile.Name())
	c.Assert(err, IsNil)
	c.Check(0, Equals, len(s.disp.SbatchArguments))
}

func (s *StubbedSuite) TestReadConfig(c *C) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	args := []string{"--arg1=v1", "--arg2", "--arg3=v3"}
	argsS := `{"SbatchArguments": ["--arg1=v1",  "--arg2", "--arg3=v3"]}`
	_, err = tmpfile.Write([]byte(argsS))
	c.Check(err, IsNil)

	err = s.disp.readConfig(tmpfile.Name())
	c.Assert(err, IsNil)
	c.Check(args, DeepEquals, s.disp.SbatchArguments)
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
		s.disp.SbatchArguments = defaults

		args, err := s.disp.sbatchArgs(container)
		c.Check(args, DeepEquals, append(defaults, "--job-name=123", "--nice=10000", "--mem=239", "--cpus-per-task=2", "--tmp=0"))
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
		types      []arvados.InstanceType
		sbatchArgs []string
		err        error
	}{
		// Choose node type => use --constraint arg
		{
			types: []arvados.InstanceType{
				{Name: "a1.tiny", Price: 0.02, RAM: 128000000, VCPUs: 1},
				{Name: "a1.small", Price: 0.04, RAM: 256000000, VCPUs: 2},
				{Name: "a1.medium", Price: 0.08, RAM: 512000000, VCPUs: 4},
				{Name: "a1.large", Price: 0.16, RAM: 1024000000, VCPUs: 8},
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
			types: []arvados.InstanceType{
				{Name: "a1.tiny", Price: 0.02, RAM: 128000000, VCPUs: 1},
			},
			err: dispatchcloud.ConstraintsNotSatisfiableError{},
		},
	} {
		c.Logf("%#v", trial)
		s.disp.cluster = &arvados.Cluster{InstanceTypes: trial.types}

		args, err := s.disp.sbatchArgs(container)
		c.Check(err == nil, Equals, trial.err == nil)
		if trial.err == nil {
			c.Check(args, DeepEquals, append([]string{"--job-name=123", "--nice=10000"}, trial.sbatchArgs...))
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
		"--job-name=123", "--nice=10000",
		"--mem=239", "--cpus-per-task=1", "--tmp=0",
		"--partition=blurb,b2",
	})
	c.Check(err, IsNil)
}
