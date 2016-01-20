package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"github.com/curoverse/dockerclient"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Gocheck boilerplate
func TestCrunchExec(t *testing.T) {
	TestingT(t)
}

type TestSuite struct{}

// Gocheck boilerplate
var _ = Suite(&TestSuite{})

type ArvTestClient struct {
	Total   int64
	Calls   int
	Content arvadosclient.Dict
	ContainerRecord
	Logs          map[string]*bytes.Buffer
	WasSetRunning bool
}

type KeepTestClient struct {
	Called  bool
	Content []byte
}

var hwManifest = ". 82ab40c24fc8df01798e57ba66795bb1+841216+Aa124ac75e5168396c73c0a18eda641a4f41791c0@569fa8c3 0:841216:9c31ee32b3d15268a0754e8edc74d4f815ee014b693bc5109058e431dd5caea7.tar\n"
var hwPDH = "a45557269dcb65a6b78f9ac061c0850b+120"
var hwImageId = "9c31ee32b3d15268a0754e8edc74d4f815ee014b693bc5109058e431dd5caea7"

var otherManifest = ". 68a84f561b1d1708c6baff5e019a9ab3+46+Ae5d0af96944a3690becb1decdf60cc1c937f556d@5693216f 0:46:md5sum.txt\n"
var otherPDH = "a3e8f74c6f101eae01fa08bfb4e49b3a+54"

type TestDockerClient struct {
	imageLoaded  string
	stdoutReader io.ReadCloser
	stderrReader io.ReadCloser
	stdoutWriter io.WriteCloser
	stderrWriter io.WriteCloser
	fn           func(t *TestDockerClient)
	finish       chan dockerclient.WaitResult
	stop         chan bool
	cwd          string
	env          []string
}

func NewTestDockerClient() *TestDockerClient {
	t := &TestDockerClient{}
	t.stdoutReader, t.stdoutWriter = io.Pipe()
	t.stderrReader, t.stderrWriter = io.Pipe()
	t.finish = make(chan dockerclient.WaitResult)
	t.stop = make(chan bool)
	t.cwd = "/"
	return t
}

func (t *TestDockerClient) StopContainer(id string, timeout int) error {
	t.stop <- true
	return nil
}

func (t *TestDockerClient) InspectImage(id string) (*dockerclient.ImageInfo, error) {
	if t.imageLoaded == id {
		return &dockerclient.ImageInfo{}, nil
	} else {
		return nil, errors.New("")
	}
}

func (t *TestDockerClient) LoadImage(reader io.Reader) error {
	_, err := io.Copy(ioutil.Discard, reader)
	if err != nil {
		return err
	} else {
		t.imageLoaded = hwImageId
		return nil
	}
}

func (t *TestDockerClient) CreateContainer(config *dockerclient.ContainerConfig, name string, authConfig *dockerclient.AuthConfig) (string, error) {
	if config.WorkingDir != "" {
		t.cwd = config.WorkingDir
	}
	t.env = config.Env
	return "abcde", nil
}

func (t *TestDockerClient) StartContainer(id string, config *dockerclient.HostConfig) error {
	if id == "abcde" {
		go t.fn(t)
		return nil
	} else {
		return errors.New("Invalid container id")
	}
}

func (t *TestDockerClient) ContainerLogs(id string, options *dockerclient.LogOptions) (io.ReadCloser, error) {
	if options.Stdout {
		return t.stdoutReader, nil
	}
	if options.Stderr {
		return t.stderrReader, nil
	}
	return nil, nil
}

func (t *TestDockerClient) Wait(id string) <-chan dockerclient.WaitResult {
	return t.finish
}

func (*TestDockerClient) RemoveImage(name string, force bool) ([]*dockerclient.ImageDelete, error) {
	return nil, nil
}

func (this *ArvTestClient) Create(resourceType string,
	parameters arvadosclient.Dict,
	output interface{}) error {

	this.Calls += 1
	this.Content = parameters

	if resourceType == "logs" {
		et := parameters["event_type"].(string)
		if this.Logs == nil {
			this.Logs = make(map[string]*bytes.Buffer)
		}
		if this.Logs[et] == nil {
			this.Logs[et] = &bytes.Buffer{}
		}
		this.Logs[et].Write([]byte(parameters["properties"].(map[string]string)["text"]))
	}

	if resourceType == "collections" && output != nil {
		mt := parameters["manifest_text"].(string)
		outmap := output.(map[string]string)
		outmap["portable_data_hash"] = fmt.Sprintf("%x+%d", md5.Sum([]byte(mt)), len(mt))
	}

	return nil
}

func (this *ArvTestClient) Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error {
	if resourceType == "collections" {
		if uuid == hwPDH {
			output.(*Collection).ManifestText = hwManifest
		} else if uuid == otherPDH {
			output.(*Collection).ManifestText = otherManifest
		}
	}
	if resourceType == "containers" {
		(*output.(*ContainerRecord)) = this.ContainerRecord
	}
	return nil
}

func (this *ArvTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {

	this.Content = parameters
	if resourceType == "containers" {
		if parameters["state"] == "Running" {
			this.WasSetRunning = true
		}

	}
	return nil
}

func (this *KeepTestClient) PutHB(hash string, buf []byte) (string, int, error) {
	this.Content = buf
	return fmt.Sprintf("%s+%d", hash, len(buf)), len(buf), nil
}

type FileWrapper struct {
	io.ReadCloser
	len uint64
}

func (this FileWrapper) Len() uint64 {
	return this.len
}

func (this *KeepTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error) {
	if filename == hwImageId+".tar" {
		rdr := ioutil.NopCloser(&bytes.Buffer{})
		this.Called = true
		return FileWrapper{rdr, 1321984}, nil
	}
	return nil, nil
}

func (s *TestSuite) TestLoadImage(c *C) {
	kc := &KeepTestClient{}
	docker := NewTestDockerClient()
	cr := NewContainerRunner(&ArvTestClient{}, kc, docker)

	_, err := cr.Docker.RemoveImage(hwImageId, true)

	_, err = cr.Docker.InspectImage(hwImageId)
	c.Check(err, NotNil)

	cr.ContainerRecord.ContainerImage = hwPDH

	// (1) Test loading image from keep
	c.Check(kc.Called, Equals, false)
	c.Check(cr.ContainerConfig.Image, Equals, "")

	err = cr.LoadImage()

	c.Check(err, IsNil)
	defer func() {
		cr.Docker.RemoveImage(hwImageId, true)
	}()

	c.Check(kc.Called, Equals, true)
	c.Check(cr.ContainerConfig.Image, Equals, hwImageId)

	_, err = cr.Docker.InspectImage(hwImageId)
	c.Check(err, IsNil)

	// (2) Test using image that's already loaded
	kc.Called = false
	cr.ContainerConfig.Image = ""

	err = cr.LoadImage()
	c.Check(err, IsNil)
	c.Check(kc.Called, Equals, false)
	c.Check(cr.ContainerConfig.Image, Equals, hwImageId)

}

type ArvErrorTestClient struct{}
type KeepErrorTestClient struct{}
type KeepReadErrorTestClient struct{}

func (this ArvErrorTestClient) Create(resourceType string,
	parameters arvadosclient.Dict,
	output interface{}) error {
	return nil
}

func (this ArvErrorTestClient) Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error {
	return errors.New("ArvError")
}

func (this ArvErrorTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (this KeepErrorTestClient) PutHB(hash string, buf []byte) (string, int, error) {
	return "", 0, nil
}

func (this KeepErrorTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error) {
	return nil, errors.New("KeepError")
}

func (this KeepReadErrorTestClient) PutHB(hash string, buf []byte) (string, int, error) {
	return "", 0, nil
}

type ErrorReader struct{}

func (this ErrorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("ErrorReader")
}

func (this ErrorReader) Close() error {
	return nil
}

func (this ErrorReader) Len() uint64 {
	return 0
}

func (this KeepReadErrorTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error) {
	return ErrorReader{}, nil
}

func (s *TestSuite) TestLoadImageArvError(c *C) {
	// (1) Arvados error
	cr := NewContainerRunner(ArvErrorTestClient{}, &KeepTestClient{}, nil)
	cr.ContainerRecord.ContainerImage = hwPDH

	err := cr.LoadImage()
	c.Check(err.Error(), Equals, "ArvError")
}

func (s *TestSuite) TestLoadImageKeepError(c *C) {
	// (2) Keep error
	docker := NewTestDockerClient()
	cr := NewContainerRunner(&ArvTestClient{}, KeepErrorTestClient{}, docker)
	cr.ContainerRecord.ContainerImage = hwPDH

	err := cr.LoadImage()
	c.Check(err.Error(), Equals, "KeepError")
}

func (s *TestSuite) TestLoadImageCollectionError(c *C) {
	// (3) Collection doesn't contain image
	cr := NewContainerRunner(&ArvTestClient{}, KeepErrorTestClient{}, nil)
	cr.ContainerRecord.ContainerImage = otherPDH

	err := cr.LoadImage()
	c.Check(err.Error(), Equals, "First file in the collection does not end in .tar")
}

func (s *TestSuite) TestLoadImageKeepReadError(c *C) {
	// (4) Collection doesn't contain image
	docker := NewTestDockerClient()
	cr := NewContainerRunner(&ArvTestClient{}, KeepReadErrorTestClient{}, docker)
	cr.ContainerRecord.ContainerImage = hwPDH

	err := cr.LoadImage()
	c.Check(err, NotNil)
}

type ClosableBuffer struct {
	bytes.Buffer
}

type TestLogs struct {
	Stdout ClosableBuffer
	Stderr ClosableBuffer
}

func (this *ClosableBuffer) Close() error {
	return nil
}

func (this *TestLogs) NewTestLoggingWriter(logstr string) io.WriteCloser {
	if logstr == "stdout" {
		return &this.Stdout
	}
	if logstr == "stderr" {
		return &this.Stderr
	}
	return nil
}

func (s *TestSuite) TestRunContainer(c *C) {
	docker := NewTestDockerClient()
	docker.fn = func(t *TestDockerClient) {
		t.stdoutWriter.Write([]byte("Hello world\n"))
		t.stdoutWriter.Close()
		t.stderrWriter.Close()
		t.finish <- dockerclient.WaitResult{}
	}
	cr := NewContainerRunner(&ArvTestClient{}, &KeepTestClient{}, docker)

	var logs TestLogs
	cr.NewLogWriter = logs.NewTestLoggingWriter
	cr.ContainerRecord.ContainerImage = hwPDH
	cr.ContainerRecord.Command = []string{"./hw"}
	err := cr.LoadImage()
	c.Check(err, IsNil)

	err = cr.StartContainer()
	c.Check(err, IsNil)

	err = cr.AttachLogs()
	c.Check(err, IsNil)

	err = cr.WaitFinish()
	c.Check(err, IsNil)

	c.Check(strings.HasSuffix(logs.Stdout.String(), "Hello world\n"), Equals, true)
	c.Check(logs.Stderr.String(), Equals, "")
}

func (s *TestSuite) TestCommitLogs(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil)
	cr.ContainerRecord.UUID = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"
	cr.CrunchLog.Timestamper = (&TestTimestamper{}).Timestamp

	cr.CrunchLog.Print("Hello world!")
	cr.CrunchLog.Print("Goodbye")
	cr.finalState = "Complete"

	err := cr.CommitLogs()
	c.Check(err, IsNil)

	c.Check(api.Content["name"], Equals, "logs for zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Check(api.Content["manifest_text"], Equals, ". 744b2e4553123b02fa7b452ec5c18993+123 0:123:crunch-run.txt\n")
	c.Check(*cr.LogsPDH, Equals, "63da7bdacf08c40f604daad80c261e9a+60")
}

func (s *TestSuite) TestUpdateContainerRecordRunning(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil)
	cr.ContainerRecord.UUID = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"

	err := cr.UpdateContainerRecordRunning()
	c.Check(err, IsNil)

	c.Check(api.Content["state"], Equals, "Running")
}

func (s *TestSuite) TestUpdateContainerRecordComplete(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil)
	cr.ContainerRecord.UUID = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"

	cr.LogsPDH = new(string)
	*cr.LogsPDH = "d3a229d2fe3690c2c3e75a71a153c6a3+60"

	cr.ExitCode = new(int)
	*cr.ExitCode = 42
	cr.finalState = "Complete"

	err := cr.UpdateContainerRecordComplete()
	c.Check(err, IsNil)

	c.Check(api.Content["log"], Equals, *cr.LogsPDH)
	c.Check(api.Content["exit_code"], Equals, *cr.ExitCode)
	c.Check(api.Content["state"], Equals, "Complete")
}

func (s *TestSuite) TestUpdateContainerRecordCancelled(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil)
	cr.ContainerRecord.UUID = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"
	cr.Cancelled = true
	cr.finalState = "Cancelled"

	err := cr.UpdateContainerRecordComplete()
	c.Check(err, IsNil)

	c.Check(api.Content["log"], IsNil)
	c.Check(api.Content["exit_code"], IsNil)
	c.Check(api.Content["state"], Equals, "Cancelled")
}

// Used by the TestFullRun*() test below to DRY up boilerplate setup to do full
// dress rehersal of the Run() function, starting from a JSON container record.
func FullRunHelper(c *C, record string, fn func(t *TestDockerClient)) (api *ArvTestClient, cr *ContainerRunner) {
	rec := ContainerRecord{}
	err := json.NewDecoder(strings.NewReader(record)).Decode(&rec)
	c.Check(err, IsNil)

	docker := NewTestDockerClient()
	docker.fn = fn
	docker.RemoveImage(hwImageId, true)

	api = &ArvTestClient{ContainerRecord: rec}
	cr = NewContainerRunner(api, &KeepTestClient{}, docker)

	err = cr.Run("zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Check(err, IsNil)
	c.Check(api.WasSetRunning, Equals, true)

	c.Check(api.Content["log"], NotNil)

	if err != nil {
		for k, v := range api.Logs {
			c.Log(k)
			c.Log(v.String())
		}
	}

	return
}

func (s *TestSuite) TestFullRunHello(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["echo", "hello world"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {},
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.stdoutWriter.Write([]byte("hello world\n"))
		t.stdoutWriter.Close()
		t.stderrWriter.Close()
		t.finish <- dockerclient.WaitResult{}
	})

	c.Check(api.Content["exit_code"], Equals, 0)
	c.Check(api.Content["state"], Equals, "Complete")

	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "hello world\n"), Equals, true)

}

func (s *TestSuite) TestFullRunStderr(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo hello ; echo world 1>&2 ; exit 1"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {},
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.stdoutWriter.Write([]byte("hello\n"))
		t.stderrWriter.Write([]byte("world\n"))
		t.stdoutWriter.Close()
		t.stderrWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 1}
	})

	c.Check(api.Content["log"], NotNil)
	c.Check(api.Content["exit_code"], Equals, 1)
	c.Check(api.Content["state"], Equals, "Complete")

	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "hello\n"), Equals, true)
	c.Check(strings.HasSuffix(api.Logs["stderr"].String(), "world\n"), Equals, true)
}

func (s *TestSuite) TestFullRunDefaultCwd(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {},
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.stdoutWriter.Write([]byte(t.cwd + "\n"))
		t.stdoutWriter.Close()
		t.stderrWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	})

	c.Check(api.Content["exit_code"], Equals, 0)
	c.Check(api.Content["state"], Equals, "Complete")

	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "/\n"), Equals, true)
}

func (s *TestSuite) TestFullRunSetCwd(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": "/bin",
    "environment": {},
    "mounts": {},
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.stdoutWriter.Write([]byte(t.cwd + "\n"))
		t.stdoutWriter.Close()
		t.stderrWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	})

	c.Check(api.Content["exit_code"], Equals, 0)
	c.Check(api.Content["state"], Equals, "Complete")

	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "/bin\n"), Equals, true)
}

func (s *TestSuite) TestCancel(c *C) {
	record := `{
    "command": ["/bin/sh", "-c", "echo foo && sleep 30 && echo bar"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {},
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`

	rec := ContainerRecord{}
	err := json.NewDecoder(strings.NewReader(record)).Decode(&rec)
	c.Check(err, IsNil)

	docker := NewTestDockerClient()
	docker.fn = func(t *TestDockerClient) {
		<-t.stop
		t.stdoutWriter.Write([]byte("foo\n"))
		t.stdoutWriter.Close()
		t.stderrWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	}
	docker.RemoveImage(hwImageId, true)

	api := &ArvTestClient{ContainerRecord: rec}
	cr := NewContainerRunner(api, &KeepTestClient{}, docker)

	go func() {
		for cr.ContainerID == "" {
			time.Sleep(1 * time.Second)
		}
		cr.SigChan <- syscall.SIGINT
	}()

	err = cr.Run("zzzzz-zzzzz-zzzzzzzzzzzzzzz")

	c.Check(err, IsNil)

	c.Check(api.Content["log"], NotNil)

	if err != nil {
		for k, v := range api.Logs {
			c.Log(k)
			c.Log(v.String())
		}
	}

	c.Check(api.Content["state"], Equals, "Cancelled")

	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "foo\n"), Equals, true)

}

func (s *TestSuite) TestFullRunSetEnv(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo $FROBIZ"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": "/bin",
    "environment": {"FROBIZ": "bilbo"},
    "mounts": {},
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.stdoutWriter.Write([]byte(t.env[0][7:] + "\n"))
		t.stdoutWriter.Close()
		t.stderrWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	})

	c.Check(api.Content["exit_code"], Equals, 0)
	c.Check(api.Content["state"], Equals, "Complete")

	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "bilbo\n"), Equals, true)
}
