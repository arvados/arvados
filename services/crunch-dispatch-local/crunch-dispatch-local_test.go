package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"

	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
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
	args := []string{"crunch-dispatch-local"}
	os.Args = args

	var err error
	arv, err = arvadosclient.MakeArvadosClient()
	if err != nil {
		c.Fatalf("Error making arvados client: %s", err)
	}
}

func (s *TestSuite) TearDownTest(c *C) {
	arvadostest.ResetEnv()
	os.Args = initialArgs
}

func (s *MockArvadosServerSuite) TearDownTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *TestSuite) Test_doMain(c *C) {
	args := []string{"-poll-interval", "2", "-container-priority-poll-interval", "1", "-crunch-run-command", "echo"}
	os.Args = append(os.Args, args...)

	go func() {
		time.Sleep(5 * time.Second)
		sigChan <- syscall.SIGINT
	}()

	err := doMain()
	c.Check(err, IsNil)

	// There should be no queued containers now
	params := arvadosclient.Dict{
		"filters": [][]string{[]string{"state", "=", "Queued"}},
	}
	var containers ContainerList
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Assert(len(containers.Items), Equals, 0)

	// Previously "Queued" container should now be in "Complete" state
	var container Container
	err = arv.Get("containers", "zzzzz-dz642-queuedcontainer", nil, &container)
	c.Check(err, IsNil)
	c.Check(container.State, Equals, "Complete")
}

func (s *MockArvadosServerSuite) Test_APIErrorGettingContainers(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/containers"] = arvadostest.StubResponse{500, string(`{}`)}

	testWithServerStub(c, apiStubResponses, "echo", "Error getting list of queued containers")
}

func (s *MockArvadosServerSuite) Test_APIErrorUpdatingContainerState(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/containers"] =
		arvadostest.StubResponse{200, string(`{"items_available":1, "items":[{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx1"}]}`)}
	apiStubResponses["/arvados/v1/containers/zzzzz-dz642-xxxxxxxxxxxxxx1"] =
		arvadostest.StubResponse{500, string(`{}`)}

	testWithServerStub(c, apiStubResponses, "echo", "Error updating container zzzzz-dz642-xxxxxxxxxxxxxx1 to 'Locked' state")
}

func (s *MockArvadosServerSuite) Test_ContainerStillInRunningAfterRun(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/containers"] =
		arvadostest.StubResponse{200, string(`{"items_available":1, "items":[{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx2"}]}`)}
	apiStubResponses["/arvados/v1/containers/zzzzz-dz642-xxxxxxxxxxxxxx2"] =
		arvadostest.StubResponse{200, string(`{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx2", "state":"Running", "priority":1}`)}

	testWithServerStub(c, apiStubResponses, "echo",
		"After crunch-run process termination, the state is still 'Running' for zzzzz-dz642-xxxxxxxxxxxxxx2")
}

func (s *MockArvadosServerSuite) Test_ErrorRunningContainer(c *C) {
	apiStubResponses := make(map[string]arvadostest.StubResponse)
	apiStubResponses["/arvados/v1/containers"] =
		arvadostest.StubResponse{200, string(`{"items_available":1, "items":[{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx3"}]}`)}
	apiStubResponses["/arvados/v1/containers/zzzzz-dz642-xxxxxxxxxxxxxx3"] =
		arvadostest.StubResponse{200, string(`{"uuid":"zzzzz-dz642-xxxxxxxxxxxxxx3", "state":"Running", "priority":1}`)}

	testWithServerStub(c, apiStubResponses, "nosuchcommand", "Error starting crunch-run for zzzzz-dz642-xxxxxxxxxxxxxx3")
}

func testWithServerStub(c *C, apiStubResponses map[string]arvadostest.StubResponse, crunchCmd string, expected string) {
	apiStub := arvadostest.ServerStub{apiStubResponses}

	api := httptest.NewServer(&apiStub)
	defer api.Close()

	arv = arvadosclient.ArvadosClient{
		Scheme:    "http",
		ApiServer: api.URL[7:],
		ApiToken:  "abc123",
		Client:    &http.Client{Transport: &http.Transport{}},
		Retries:   0,
	}

	buf := bytes.NewBuffer(nil)
	log.SetOutput(buf)
	defer log.SetOutput(os.Stderr)

	go func() {
		time.Sleep(2 * time.Second)
		sigChan <- syscall.SIGTERM
	}()

	runQueuedContainers(time.Second, time.Second, crunchCmd)

	// Wait for all running crunch jobs to complete / terminate
	waitGroup.Wait()

	c.Check(buf.String(), Matches, `(?ms).*`+expected+`.*`)
}
