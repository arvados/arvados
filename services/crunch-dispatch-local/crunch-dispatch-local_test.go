package main

import (
	"bytes"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
	. "gopkg.in/check.v1"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
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
	runningCmds = make(map[string]*exec.Cmd)
}

func (s *TestSuite) TearDownSuite(c *C) {
	arvadostest.StopAPI()
}

func (s *TestSuite) SetUpTest(c *C) {
	args := []string{"crunch-dispatch-local"}
	os.Args = args
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

	echo := "echo"
	crunchRunCommand = &echo

	doneProcessing := make(chan struct{})
	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		PollInterval: time.Second,
		RunContainer: func(dispatcher *dispatch.Dispatcher,
			container arvados.Container,
			status chan arvados.Container) {
			run(dispatcher, container, status)
			doneProcessing <- struct{}{}
		},
		DoneProcessing: doneProcessing}

	startCmd = func(container arvados.Container, cmd *exec.Cmd) error {
		dispatcher.UpdateState(container.UUID, "Running")
		dispatcher.UpdateState(container.UUID, "Complete")
		return cmd.Start()
	}

	err = dispatcher.RunDispatcher()
	c.Assert(err, IsNil)

	// Wait for all running crunch jobs to complete / terminate
	waitGroup.Wait()

	// There should be no queued containers now
	params := arvadosclient.Dict{
		"filters": [][]string{{"state", "=", "Queued"}},
	}
	var containers arvados.ContainerList
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Assert(len(containers.Items), Equals, 0)

	// Previously "Queued" container should now be in "Complete" state
	var container arvados.Container
	err = arv.Get("containers", "zzzzz-dz642-queuedcontainer", nil, &container)
	c.Check(err, IsNil)
	c.Check(string(container.State), Equals, "Complete")
}

func (s *MockArvadosServerSuite) Test_APIErrorGettingContainers(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/containers"] = arvadostest.StubResponse{500, string(`{}`)}

	testWithServerStub(c, apiStubResponses, "echo", "Error getting list of containers")
}

func (s *MockArvadosServerSuite) Test_APIErrorUpdatingContainerState(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/containers"] =
		arvadostest.StubResponse{200, string(`{"items_available":1, "items":[{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx1","State":"Queued","Priority":1}]}`)}
	apiStubResponses["/arvados/v1/containers/zzzzz-dz642-xxxxxxxxxxxxxx1"] =
		arvadostest.StubResponse{500, string(`{}`)}

	testWithServerStub(c, apiStubResponses, "echo", "Error updating container zzzzz-dz642-xxxxxxxxxxxxxx1 to state \"Locked\"")
}

func (s *MockArvadosServerSuite) Test_ContainerStillInRunningAfterRun(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/containers"] =
		arvadostest.StubResponse{200, string(`{"items_available":1, "items":[{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx2","State":"Queued","Priority":1}]}`)}
	apiStubResponses["/arvados/v1/containers/zzzzz-dz642-xxxxxxxxxxxxxx2"] =
		arvadostest.StubResponse{200, string(`{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx2", "state":"Running", "priority":1, "locked_by_uuid": "` + arvadostest.Dispatch1AuthUUID + `"}`)}

	testWithServerStub(c, apiStubResponses, "echo",
		`After echo process termination, container state for Running is "zzzzz-dz642-xxxxxxxxxxxxxx2".  Updating it to "Cancelled"`)
}

func (s *MockArvadosServerSuite) Test_ErrorRunningContainer(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/containers"] =
		arvadostest.StubResponse{200, string(`{"items_available":1, "items":[{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx3","State":"Queued","Priority":1}]}`)}

	apiStubResponses["/arvados/v1/containers/zzzzz-dz642-xxxxxxxxxxxxxx3"] =
		arvadostest.StubResponse{200, string(`{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx3", "state":"Running", "priority":1}`)}

	testWithServerStub(c, apiStubResponses, "nosuchcommand", "Error starting nosuchcommand for zzzzz-dz642-xxxxxxxxxxxxxx3")
}

func testWithServerStub(c *C, apiStubResponses map[string]arvadostest.StubResponse, crunchCmd string, expected string) {
	apiStubResponses["/arvados/v1/api_client_authorizations/current"] =
		arvadostest.StubResponse{200, string(`{"uuid": "` + arvadostest.Dispatch1AuthUUID + `", "api_token": "xyz"}`)}

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

	*crunchRunCommand = crunchCmd

	doneProcessing := make(chan struct{})
	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		PollInterval: time.Duration(1) * time.Second,
		RunContainer: func(dispatcher *dispatch.Dispatcher,
			container arvados.Container,
			status chan arvados.Container) {
			run(dispatcher, container, status)
			doneProcessing <- struct{}{}
		},
		DoneProcessing: doneProcessing}

	startCmd = func(container arvados.Container, cmd *exec.Cmd) error {
		dispatcher.UpdateState(container.UUID, "Running")
		dispatcher.UpdateState(container.UUID, "Complete")
		return cmd.Start()
	}

	go func() {
		for i := 0; i < 80 && !strings.Contains(buf.String(), expected); i++ {
			time.Sleep(100 * time.Millisecond)
		}
		dispatcher.DoneProcessing <- struct{}{}
	}()

	err := dispatcher.RunDispatcher()
	c.Assert(err, IsNil)

	// Wait for all running crunch jobs to complete / terminate
	waitGroup.Wait()

	c.Check(buf.String(), Matches, `(?ms).*`+expected+`.*`)
}
