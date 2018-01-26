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

var _ = Suite(&TestSuite{})
var _ = Suite(&MockArvadosServerSuite{})

type TestSuite struct {
	cmd command
}

var initialArgs []string

func (s *TestSuite) SetUpSuite(c *C) {
	initialArgs = os.Args
}

func (s *TestSuite) TearDownSuite(c *C) {
}

func (s *TestSuite) SetUpTest(c *C) {
	args := []string{"crunch-dispatch-slurm"}
	os.Args = args

	arvadostest.StartAPI()
	os.Setenv("ARVADOS_API_TOKEN", arvadostest.Dispatch1Token)
}

func (s *TestSuite) TearDownTest(c *C) {
	os.Args = initialArgs
	arvadostest.ResetEnv()
	arvadostest.StopAPI()
}

type slurmFake struct {
	didBatch  [][]string
	didCancel []string
	didRenice [][]string
	queue     string
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

func (sf *slurmFake) Renice(name string, nice int) error {
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

func (s *TestSuite) integrationTest(c *C, slurm *slurmFake,
	expectBatch [][]string,
	runContainer func(*dispatch.Dispatcher, arvados.Container)) arvados.Container {
	arvadostest.ResetEnv()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	s.cmd.slurm = slurm

	// There should be one queued container
	params := arvadosclient.Dict{
		"filters": [][]string{{"state", "=", "Queued"}},
	}
	var containers arvados.ContainerList
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Check(len(containers.Items), Equals, 1)

	s.cmd.CrunchRunCommand = []string{"echo"}

	ctx, cancel := context.WithCancel(context.Background())
	doneRun := make(chan struct{})

	s.cmd.dispatcher = &dispatch.Dispatcher{
		Arv:        arv,
		PollPeriod: time.Duration(1) * time.Second,
		RunContainer: func(disp *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
			go func() {
				runContainer(disp, ctr)
				slurm.queue = ""
				doneRun <- struct{}{}
			}()
			s.cmd.run(disp, ctr, status)
			cancel()
		},
	}

	s.cmd.sqCheck = &SqueueChecker{Period: 500 * time.Millisecond, Slurm: slurm}

	err = s.cmd.dispatcher.Run(ctx)
	<-doneRun
	c.Assert(err, Equals, context.Canceled)

	s.cmd.sqCheck.Stop()

	c.Check(slurm.didBatch, DeepEquals, expectBatch)

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

func (s *TestSuite) TestIntegrationNormal(c *C) {
	container := s.integrationTest(c,
		&slurmFake{queue: "zzzzz-dz642-queuedcontainer 9990 100\n"},
		nil,
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(3 * time.Second)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateComplete)
}

func (s *TestSuite) TestIntegrationCancel(c *C) {
	slurm := &slurmFake{queue: "zzzzz-dz642-queuedcontainer 9990 100\n"}
	readyToCancel := make(chan bool)
	slurm.onCancel = func() { <-readyToCancel }
	container := s.integrationTest(c,
		slurm,
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
	c.Check(len(slurm.didCancel) > 1, Equals, true)
	c.Check(slurm.didCancel[:2], DeepEquals, []string{"zzzzz-dz642-queuedcontainer", "zzzzz-dz642-queuedcontainer"})
}

func (s *TestSuite) TestIntegrationMissingFromSqueue(c *C) {
	container := s.integrationTest(c, &slurmFake{},
		[][]string{{
			fmt.Sprintf("--job-name=%s", "zzzzz-dz642-queuedcontainer"),
			fmt.Sprintf("--mem=%d", 11445),
			fmt.Sprintf("--cpus-per-task=%d", 4),
			fmt.Sprintf("--tmp=%d", 45777),
			fmt.Sprintf("--nice=%d", 9990)}},
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(3 * time.Second)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateCancelled)
}

func (s *TestSuite) TestSbatchFail(c *C) {
	container := s.integrationTest(c,
		&slurmFake{errBatch: errors.New("something terrible happened")},
		[][]string{{"--job-name=zzzzz-dz642-queuedcontainer", "--mem=11445", "--cpus-per-task=4", "--tmp=45777", "--nice=9990"}},
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
	c.Assert(len(ll.Items), Equals, 1)
}

func (s *TestSuite) TestIntegrationChangePriority(c *C) {
	slurm := &slurmFake{queue: "zzzzz-dz642-queuedcontainer 9990 100\n"}
	container := s.integrationTest(c, slurm, nil,
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(time.Second)
			dispatcher.Arv.Update("containers", container.UUID,
				arvadosclient.Dict{
					"container": arvadosclient.Dict{"priority": 600}},
				nil)
			time.Sleep(time.Second)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateComplete)
	c.Assert(len(slurm.didRenice), Not(Equals), 0)
	c.Check(slurm.didRenice[len(slurm.didRenice)-1], DeepEquals, []string{"zzzzz-dz642-queuedcontainer", "4000"})
}

type MockArvadosServerSuite struct {
	cmd command
}

func (s *MockArvadosServerSuite) TearDownTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *MockArvadosServerSuite) TestAPIErrorGettingContainers(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/api_client_authorizations/current"] = arvadostest.StubResponse{200, `{"uuid":"` + arvadostest.Dispatch1AuthUUID + `"}`}
	apiStubResponses["/arvados/v1/containers"] = arvadostest.StubResponse{500, string(`{}`)}

	s.testWithServerStub(c, apiStubResponses, "echo", "Error getting list of containers")
}

func (s *MockArvadosServerSuite) testWithServerStub(c *C, apiStubResponses map[string]arvadostest.StubResponse, crunchCmd string, expected string) {
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

	s.cmd.CrunchRunCommand = []string{crunchCmd}

	ctx, cancel := context.WithCancel(context.Background())
	dispatcher := dispatch.Dispatcher{
		Arv:        arv,
		PollPeriod: time.Duration(1) * time.Second,
		RunContainer: func(disp *dispatch.Dispatcher, ctr arvados.Container, status <-chan arvados.Container) {
			go func() {
				time.Sleep(1 * time.Second)
				disp.UpdateState(ctr.UUID, dispatch.Running)
				disp.UpdateState(ctr.UUID, dispatch.Complete)
			}()
			s.cmd.run(disp, ctr, status)
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

func (s *MockArvadosServerSuite) TestNoSuchConfigFile(c *C) {
	err := s.cmd.readConfig("/nosuchdir89j7879/8hjwr7ojgyy7")
	c.Assert(err, NotNil)
}

func (s *MockArvadosServerSuite) TestBadSbatchArgsConfig(c *C) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"SbatchArguments": "oops this is not a string array"}`))
	c.Check(err, IsNil)

	err = s.cmd.readConfig(tmpfile.Name())
	c.Assert(err, NotNil)
}

func (s *MockArvadosServerSuite) TestNoSuchArgInConfigIgnored(c *C) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"NoSuchArg": "Nobody loves me, not one tiny hunk."}`))
	c.Check(err, IsNil)

	err = s.cmd.readConfig(tmpfile.Name())
	c.Assert(err, IsNil)
	c.Check(0, Equals, len(s.cmd.SbatchArguments))
}

func (s *MockArvadosServerSuite) TestReadConfig(c *C) {
	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	args := []string{"--arg1=v1", "--arg2", "--arg3=v3"}
	argsS := `{"SbatchArguments": ["--arg1=v1",  "--arg2", "--arg3=v3"]}`
	_, err = tmpfile.Write([]byte(argsS))
	c.Check(err, IsNil)

	err = s.cmd.readConfig(tmpfile.Name())
	c.Assert(err, IsNil)
	c.Check(args, DeepEquals, s.cmd.SbatchArguments)
}

func (s *MockArvadosServerSuite) TestSbatchFuncWithNoConfigArgs(c *C) {
	s.testSbatchFuncWithArgs(c, nil)
}

func (s *MockArvadosServerSuite) TestSbatchFuncWithEmptyConfigArgs(c *C) {
	s.testSbatchFuncWithArgs(c, []string{})
}

func (s *MockArvadosServerSuite) TestSbatchFuncWithConfigArgs(c *C) {
	s.testSbatchFuncWithArgs(c, []string{"--arg1=v1", "--arg2"})
}

func (s *MockArvadosServerSuite) testSbatchFuncWithArgs(c *C, args []string) {
	s.cmd.SbatchArguments = append([]string(nil), args...)

	container := arvados.Container{
		UUID:               "123",
		RuntimeConstraints: arvados.RuntimeConstraints{RAM: 250000000, VCPUs: 2},
		Priority:           1}

	var expected []string
	expected = append(expected, s.cmd.SbatchArguments...)
	expected = append(expected, "--job-name=123", "--mem=239", "--cpus-per-task=2", "--tmp=0", "--nice=9990")
	args, err := s.cmd.sbatchArgs(container)
	c.Check(args, DeepEquals, expected)
	c.Check(err, IsNil)
}

func (s *MockArvadosServerSuite) TestSbatchPartition(c *C) {
	container := arvados.Container{
		UUID:                 "123",
		RuntimeConstraints:   arvados.RuntimeConstraints{RAM: 250000000, VCPUs: 1},
		SchedulingParameters: arvados.SchedulingParameters{Partitions: []string{"blurb", "b2"}},
		Priority:             1}

	args, err := s.cmd.sbatchArgs(container)
	c.Check(args, DeepEquals, []string{
		"--job-name=123", "--mem=239", "--cpus-per-task=1", "--tmp=0", "--nice=9990",
		"--partition=blurb,b2",
	})
	c.Check(err, IsNil)
}
