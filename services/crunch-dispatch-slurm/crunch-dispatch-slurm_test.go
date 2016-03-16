package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"

	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
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
	args := []string{"crunch-dispatch-slurm"}
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

	var sbatchCmdLine []string
	var striggerCmdLine []string

	// Override sbatchCmd
	defer func(orig func(string) *exec.Cmd) {
		sbatchCmd = orig
	}(sbatchCmd)
	sbatchCmd = func(uuid string) *exec.Cmd {
		sbatchCmdLine = sbatchFunc(uuid).Args
		return exec.Command("echo", uuid)
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
					"container": arvadosclient.Dict{"state": "Complete"}},
				nil)
		}()
		return exec.Command("echo", "strigger")
	}

	go func() {
		time.Sleep(8 * time.Second)
		sigChan <- syscall.SIGINT
	}()

	// There should be no queued containers now
	params := arvadosclient.Dict{
		"filters": [][]string{[]string{"state", "=", "Queued"}},
	}
	var containers ContainerList
	err := arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Check(len(containers.Items), Equals, 1)

	err = doMain()
	c.Check(err, IsNil)

	c.Check(sbatchCmdLine, DeepEquals, []string{"sbatch", "--job-name=zzzzz-dz642-queuedcontainer", "--share", "--parsable"})
	c.Check(striggerCmdLine, DeepEquals, []string{"strigger", "--set", "--jobid=zzzzz-dz642-queuedcontainer\n", "--fini",
		"--program=/usr/bin/crunch-finish-slurm.sh " + os.Getenv("ARVADOS_API_HOST") + " 4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h 1 zzzzz-dz642-queuedcontainer"})

	// There should be no queued containers now
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Check(len(containers.Items), Equals, 0)

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

	tempfile, err := ioutil.TempFile(os.TempDir(), "temp-log-file")
	c.Check(err, IsNil)
	defer os.Remove(tempfile.Name())
	log.SetOutput(tempfile)

	go func() {
		time.Sleep(2 * time.Second)
		sigChan <- syscall.SIGTERM
	}()

	runQueuedContainers(2, 1, crunchCmd, crunchCmd)

	buf, _ := ioutil.ReadFile(tempfile.Name())
	c.Check(strings.Contains(string(buf), expected), Equals, true)
}
