package main

import (
	"bytes"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/dispatch"
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
	container := s.integrationTest(c, func() *exec.Cmd { return exec.Command("echo", "zzzzz-dz642-queuedcontainer") },
		[]string(nil),
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(3 * time.Second)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateComplete)
}

func (s *TestSuite) TestIntegrationCancel(c *C) {

	// Override sbatchCmd
	var scancelCmdLine []string
	defer func(orig func(arvados.Container) *exec.Cmd) {
		scancelCmd = orig
	}(scancelCmd)
	scancelCmd = func(container arvados.Container) *exec.Cmd {
		scancelCmdLine = scancelFunc(container).Args
		return exec.Command("echo")
	}

	container := s.integrationTest(c, func() *exec.Cmd { return exec.Command("echo", "zzzzz-dz642-queuedcontainer") },
		[]string(nil),
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(1 * time.Second)
			dispatcher.Arv.Update("containers", container.UUID,
				arvadosclient.Dict{
					"container": arvadosclient.Dict{"priority": 0}},
				nil)
		})
	c.Check(container.State, Equals, arvados.ContainerStateCancelled)
	c.Check(scancelCmdLine, DeepEquals, []string{"scancel", "--name=zzzzz-dz642-queuedcontainer"})
}

func (s *TestSuite) TestIntegrationMissingFromSqueue(c *C) {
	container := s.integrationTest(c, func() *exec.Cmd { return exec.Command("echo") }, []string{"sbatch", "--share",
		fmt.Sprintf("--job-name=%s", "zzzzz-dz642-queuedcontainer"),
		fmt.Sprintf("--mem-per-cpu=%d", 2862),
		fmt.Sprintf("--cpus-per-task=%d", 4)},
		func(dispatcher *dispatch.Dispatcher, container arvados.Container) {
			dispatcher.UpdateState(container.UUID, dispatch.Running)
			time.Sleep(3 * time.Second)
			dispatcher.UpdateState(container.UUID, dispatch.Complete)
		})
	c.Check(container.State, Equals, arvados.ContainerStateCancelled)
}

func (s *TestSuite) integrationTest(c *C,
	newSqueueCmd func() *exec.Cmd,
	sbatchCmdComps []string,
	runContainer func(*dispatch.Dispatcher, arvados.Container)) arvados.Container {
	arvadostest.ResetEnv()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	var sbatchCmdLine []string

	// Override sbatchCmd
	defer func(orig func(arvados.Container) *exec.Cmd) {
		sbatchCmd = orig
	}(sbatchCmd)
	sbatchCmd = func(container arvados.Container) *exec.Cmd {
		sbatchCmdLine = sbatchFunc(container).Args
		return exec.Command("sh")
	}

	// Override squeueCmd
	defer func(orig func() *exec.Cmd) {
		squeueCmd = orig
	}(squeueCmd)
	squeueCmd = newSqueueCmd

	// There should be no queued containers now
	params := arvadosclient.Dict{
		"filters": [][]string{{"state", "=", "Queued"}},
	}
	var containers arvados.ContainerList
	err = arv.List("containers", params, &containers)
	c.Check(err, IsNil)
	c.Check(len(containers.Items), Equals, 1)

	config.CrunchRunCommand = []string{"echo"}

	doneProcessing := make(chan struct{})
	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		PollInterval: time.Duration(1) * time.Second,
		RunContainer: func(dispatcher *dispatch.Dispatcher,
			container arvados.Container,
			status chan arvados.Container) {
			go runContainer(dispatcher, container)
			run(dispatcher, container, status)
			doneProcessing <- struct{}{}
		},
		DoneProcessing: doneProcessing}

	squeueUpdater.StartMonitor(time.Duration(500) * time.Millisecond)

	err = dispatcher.RunDispatcher()
	c.Assert(err, IsNil)

	squeueUpdater.Done()

	c.Check(sbatchCmdLine, DeepEquals, sbatchCmdComps)

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

func (s *MockArvadosServerSuite) TestAPIErrorGettingContainers(c *C) {
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

	config.CrunchRunCommand = []string{crunchCmd}

	doneProcessing := make(chan struct{})
	dispatcher := dispatch.Dispatcher{
		Arv:          arv,
		PollInterval: time.Duration(1) * time.Second,
		RunContainer: func(dispatcher *dispatch.Dispatcher,
			container arvados.Container,
			status chan arvados.Container) {
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

func (s *MockArvadosServerSuite) TestNoSuchConfigFile(c *C) {
	var config Config
	err := readConfig(&config, "/nosuchdir89j7879/8hjwr7ojgyy7")
	c.Assert(err, NotNil)
}

func (s *MockArvadosServerSuite) TestBadSbatchArgsConfig(c *C) {
	var config Config

	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"SbatchArguments": "oops this is not a string array"}`))
	c.Check(err, IsNil)

	err = readConfig(&config, tmpfile.Name())
	c.Assert(err, NotNil)
}

func (s *MockArvadosServerSuite) TestNoSuchArgInConfigIgnored(c *C) {
	var config Config

	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"NoSuchArg": "Nobody loves me, not one tiny hunk."}`))
	c.Check(err, IsNil)

	err = readConfig(&config, tmpfile.Name())
	c.Assert(err, IsNil)
	c.Check(0, Equals, len(config.SbatchArguments))
}

func (s *MockArvadosServerSuite) TestReadConfig(c *C) {
	var config Config

	tmpfile, err := ioutil.TempFile(os.TempDir(), "config")
	c.Check(err, IsNil)
	defer os.Remove(tmpfile.Name())

	args := []string{"--arg1=v1", "--arg2", "--arg3=v3"}
	argsS := `{"SbatchArguments": ["--arg1=v1",  "--arg2", "--arg3=v3"]}`
	_, err = tmpfile.Write([]byte(argsS))
	c.Check(err, IsNil)

	err = readConfig(&config, tmpfile.Name())
	c.Assert(err, IsNil)
	c.Check(3, Equals, len(config.SbatchArguments))
	c.Check(args, DeepEquals, config.SbatchArguments)
}

func (s *MockArvadosServerSuite) TestSbatchFuncWithNoConfigArgs(c *C) {
	testSbatchFuncWithArgs(c, nil)
}

func (s *MockArvadosServerSuite) TestSbatchFuncWithEmptyConfigArgs(c *C) {
	testSbatchFuncWithArgs(c, []string{})
}

func (s *MockArvadosServerSuite) TestSbatchFuncWithConfigArgs(c *C) {
	testSbatchFuncWithArgs(c, []string{"--arg1=v1", "--arg2"})
}

func testSbatchFuncWithArgs(c *C, args []string) {
	config.SbatchArguments = append(config.SbatchArguments, args...)

	container := arvados.Container{UUID: "123", RuntimeConstraints: arvados.RuntimeConstraints{RAM: 1000000, VCPUs: 2}}
	sbatchCmd := sbatchFunc(container)

	var expected []string
	expected = append(expected, "sbatch", "--share")
	expected = append(expected, config.SbatchArguments...)
	expected = append(expected, "--job-name=123", "--mem-per-cpu=1", "--cpus-per-task=2")

	c.Check(sbatchCmd.Args, DeepEquals, expected)
}
