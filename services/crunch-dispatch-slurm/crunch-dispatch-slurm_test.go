package main

import (
	"bytes"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
	"io"
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
	arvadostest.StopAPI()
}

func (s *MockArvadosServerSuite) TearDownTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *TestSuite) TestIntegrationNormal(c *C) {
	s.integrationTest(c, false)
}

func (s *TestSuite) TestIntegrationMissingFromSqueue(c *C) {
	s.integrationTest(c, true)
}

func (s *TestSuite) integrationTest(c *C, missingFromSqueue bool) {
	arvadostest.ResetEnv()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	var sbatchCmdLine []string

	// Override sbatchCmd
	defer func(orig func(dispatch.Container) *exec.Cmd) {
		sbatchCmd = orig
	}(sbatchCmd)
	sbatchCmd = func(container dispatch.Container) *exec.Cmd {
		sbatchCmdLine = sbatchFunc(container).Args
		return exec.Command("sh")
	}

	// Override squeueCmd
	defer func(orig func() *exec.Cmd) {
		squeueCmd = orig
	}(squeueCmd)
	squeueCmd = func() *exec.Cmd {
		if missingFromSqueue {
			return exec.Command("echo")
		} else {
			return exec.Command("echo", "zzzzz-dz642-queuedcontainer")
		}
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

	doneProcessing := make(chan struct{})
	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		PollInterval: time.Duration(1) * time.Second,
		RunContainer: func(dispatcher *dispatch.Dispatcher,
			container dispatch.Container,
			status chan dispatch.Container) {
			go func() {
				dispatcher.UpdateState(container.UUID, dispatch.Running)
				time.Sleep(3 * time.Second)
				dispatcher.UpdateState(container.UUID, dispatch.Complete)
			}()
			run(dispatcher, container, status)
			doneProcessing <- struct{}{}
		},
		DoneProcessing: doneProcessing}

	squeueUpdater.SqueueDone = make(chan struct{})
	go squeueUpdater.SyncSqueue(time.Duration(500) * time.Millisecond)

	err = dispatcher.RunDispatcher()
	c.Assert(err, IsNil)

	squeueUpdater.SqueueDone <- struct{}{}
	close(squeueUpdater.SqueueDone)

	item := containers.Items[0]
	sbatchCmdComps := []string{"sbatch", "--share", "--parsable",
		fmt.Sprintf("--job-name=%s", item.UUID),
		fmt.Sprintf("--mem-per-cpu=%d", int(math.Ceil(float64(item.RuntimeConstraints["ram"])/float64(item.RuntimeConstraints["vcpus"]*1048576)))),
		fmt.Sprintf("--cpus-per-task=%d", int(item.RuntimeConstraints["vcpus"])),
		fmt.Sprintf("--priority=%d", item.Priority)}

	if missingFromSqueue {
		// not in squeue when run() started, so it will have called sbatch
		c.Check(sbatchCmdLine, DeepEquals, sbatchCmdComps)
	} else {
		// already in squeue when run() started, will have just monitored it instead
		c.Check(sbatchCmdLine, DeepEquals, []string(nil))
	}

	// There should be no queued containers now
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Check(len(containers.Items), Equals, 0)

	// Previously "Queued" container should now be in "Complete" state
	var container dispatch.Container
	err = arv.Get("containers", "zzzzz-dz642-queuedcontainer", nil, &container)
	c.Check(err, IsNil)
	if missingFromSqueue {
		c.Check(container.State, Equals, "Cancelled")
	} else {
		c.Check(container.State, Equals, "Complete")
	}
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
	log.SetOutput(io.MultiWriter(buf, os.Stderr))
	defer log.SetOutput(os.Stderr)

	crunchRunCommand = &crunchCmd

	doneProcessing := make(chan struct{})
	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		PollInterval: time.Duration(1) * time.Second,
		RunContainer: func(dispatcher *dispatch.Dispatcher,
			container dispatch.Container,
			status chan dispatch.Container) {
			go func() {
				time.Sleep(1 * time.Second)
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
