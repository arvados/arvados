package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"

	"bytes"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&TestSuite{})
var _ = Suite(&MockArvadosServerSuite{})

type TestSuite struct{}
type MockArvadosServerSuite struct{}

var initialArgs []string

func (s *TestSuite) SetUpSuite(c *C) {
	initialArgs = os.Args
	arvadostest.StartAPI()
}

func (s *TestSuite) TearDownSuite(c *C) {
	arvadostest.StopAPI()
}

func (s *TestSuite) SetUpTest(c *C) {
	args := []string{"crunch-dispatch-slurm"}
	os.Args = args

	os.Setenv("ARVADOS_API_TOKEN", arvadostest.Dispatch1Token)
}

func (s *TestSuite) TearDownTest(c *C) {
	arvadostest.ResetEnv()
	os.Args = initialArgs
}

func (s *MockArvadosServerSuite) TearDownTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *TestSuite) TestIntegration(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	var sbatchCmdLine []string
	var striggerCmdLine []string

	// Override sbatchCmd
	defer func(orig func(dispatch.Container) *exec.Cmd) {
		sbatchCmd = orig
	}(sbatchCmd)
	sbatchCmd = func(container dispatch.Container) *exec.Cmd {
		sbatchCmdLine = sbatchFunc(container).Args
		return exec.Command("sh")
	}

	// Override striggerCmd
	defer func(orig func(jobid, containerUUID, finishCommand,
		apiHost, apiToken, apiInsecure string) *exec.Cmd) {
		striggerCmd = orig
	}(striggerCmd)
	striggerCmd = func(jobid, containerUUID, finishCommand, apiHost, apiToken, apiInsecure string) *exec.Cmd {
		striggerCmdLine = striggerFunc(jobid, containerUUID, finishCommand,
			apiHost, apiToken, apiInsecure).Args
		go func() {
			time.Sleep(5 * time.Second)
			arv.Update("containers", containerUUID,
				arvadosclient.Dict{
					"container": arvadosclient.Dict{"state": dispatch.Complete}},
				nil)
		}()
		return exec.Command("echo", striggerCmdLine...)
	}

	// Override squeueCmd
	defer func(orig func() *exec.Cmd) {
		squeueCmd = orig
	}(squeueCmd)
	squeueCmd = func() *exec.Cmd {
		return exec.Command("echo")
	}

	// There should be no queued containers now
	params := arvadosclient.Dict{
		"filters": [][]string{[]string{"state", "=", "Queued"}},
	}
	var containers dispatch.ContainerList
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Check(len(containers.Items), Equals, 1)

	echo := "echo"
	crunchRunCommand = &echo
	finishCmd := "/usr/bin/crunch-finish-slurm.sh"
	finishCommand = &finishCmd

	doneProcessing := make(chan struct{})
	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		PollInterval: time.Duration(1) * time.Second,
		RunContainer: func(dispatcher *dispatch.Dispatcher,
			container dispatch.Container,
			status chan dispatch.Container) {
			go func() {
				time.Sleep(1)
				dispatcher.UpdateState(container.UUID, dispatch.Running)
				dispatcher.UpdateState(container.UUID, dispatch.Complete)
			}()
			run(dispatcher, container, status)
			doneProcessing <- struct{}{}
		},
		DoneProcessing: doneProcessing}

	err = dispatcher.RunDispatcher()
	c.Assert(err, IsNil)

	item := containers.Items[0]
	sbatchCmdComps := []string{"sbatch", "--share", "--parsable",
		fmt.Sprintf("--job-name=%s", item.UUID),
		fmt.Sprintf("--mem-per-cpu=%d", int(math.Ceil(float64(item.RuntimeConstraints["ram"])/float64(item.RuntimeConstraints["vcpus"]*1048576)))),
		fmt.Sprintf("--cpus-per-task=%d", int(item.RuntimeConstraints["vcpus"])),
		fmt.Sprintf("--priority=%d", item.Priority)}
	c.Check(sbatchCmdLine, DeepEquals, sbatchCmdComps)

	c.Check(striggerCmdLine, DeepEquals, []string{"strigger", "--set", "--jobid=zzzzz-dz642-queuedcontainer", "--fini",
		"--program=/usr/bin/crunch-finish-slurm.sh " + os.Getenv("ARVADOS_API_HOST") + " " + arvadostest.Dispatch1Token + " 1 zzzzz-dz642-queuedcontainer"})

	// There should be no queued containers now
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Check(len(containers.Items), Equals, 0)

	// Previously "Queued" container should now be in "Complete" state
	var container dispatch.Container
	err = arv.Get("containers", "zzzzz-dz642-queuedcontainer", nil, &container)
	c.Check(err, IsNil)
	c.Check(container.State, Equals, "Complete")
}

func (s *MockArvadosServerSuite) Test_APIErrorGettingContainers(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/api_client_authorizations/current"] = arvadostest.StubResponse{200, `{"uuid":"` + arvadostest.Dispatch1AuthUUID + `"}`}
	apiStubResponses["/arvados/v1/containers"] = arvadostest.StubResponse{500, string(`{}`)}

	testWithServerStub(c, apiStubResponses, "echo", "Error getting list of containers")
}

func testWithServerStub(c *C, apiStubResponses map[string]arvadostest.StubResponse, crunchCmd string, expected string) {
	apiStub := arvadostest.ServerStub{apiStubResponses}

	api := httptest.NewServer(&apiStub)
	defer api.Close()

	arv := arvadosclient.ArvadosClient{
		Scheme:    "http",
		ApiServer: api.URL[7:],
		ApiToken:  "abc123",
		Client:    &http.Client{Transport: &http.Transport{}},
		Retries:   0,
	}

	buf := bytes.NewBuffer(nil)
	log.SetOutput(buf)
	defer log.SetOutput(os.Stderr)

	crunchRunCommand = &crunchCmd
	finishCmd := "/usr/bin/crunch-finish-slurm.sh"
	finishCommand = &finishCmd

	doneProcessing := make(chan struct{})
	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		PollInterval: time.Duration(1) * time.Second,
		RunContainer: func(dispatcher *dispatch.Dispatcher,
			container dispatch.Container,
			status chan dispatch.Container) {
			go func() {
				time.Sleep(1)
				dispatcher.UpdateState(container.UUID, dispatch.Running)
				dispatcher.UpdateState(container.UUID, dispatch.Complete)
			}()
			run(dispatcher, container, status)
			doneProcessing <- struct{}{}
		},
		DoneProcessing: doneProcessing}

	go func() {
		for i := 0; i < 80 && !strings.Contains(buf.String(), expected); i++ {
			time.Sleep(100 * time.Millisecond)
		}
		dispatcher.DoneProcessing <- struct{}{}
	}()

	err := dispatcher.RunDispatcher()
	c.Assert(err, IsNil)

	c.Check(buf.String(), Matches, `(?ms).*`+expected+`.*`)
}
