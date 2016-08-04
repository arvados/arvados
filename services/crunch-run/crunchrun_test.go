package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"
	"github.com/curoverse/dockerclient"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
	Content []arvadosclient.Dict
	arvados.Container
	Logs          map[string]*bytes.Buffer
	WasSetRunning bool
	sync.Mutex
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

var fakeAuthUUID = "zzzzz-gj3su-55pqoyepgi2glem"
var fakeAuthToken = "a3ltuwzqcu2u4sc0q7yhpc2w7s00fdcqecg5d6e0u3pfohmbjt"

type TestDockerClient struct {
	imageLoaded string
	logReader   io.ReadCloser
	logWriter   io.WriteCloser
	fn          func(t *TestDockerClient)
	finish      chan dockerclient.WaitResult
	stop        chan bool
	cwd         string
	env         []string
}

func NewTestDockerClient() *TestDockerClient {
	t := &TestDockerClient{}
	t.logReader, t.logWriter = io.Pipe()
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

func (t *TestDockerClient) AttachContainer(id string, options *dockerclient.AttachOptions) (io.ReadCloser, error) {
	return t.logReader, nil
}

func (t *TestDockerClient) Wait(id string) <-chan dockerclient.WaitResult {
	return t.finish
}

func (*TestDockerClient) RemoveImage(name string, force bool) ([]*dockerclient.ImageDelete, error) {
	return nil, nil
}

func (client *ArvTestClient) Create(resourceType string,
	parameters arvadosclient.Dict,
	output interface{}) error {

	client.Mutex.Lock()
	defer client.Mutex.Unlock()

	client.Calls++
	client.Content = append(client.Content, parameters)

	if resourceType == "logs" {
		et := parameters["log"].(arvadosclient.Dict)["event_type"].(string)
		if client.Logs == nil {
			client.Logs = make(map[string]*bytes.Buffer)
		}
		if client.Logs[et] == nil {
			client.Logs[et] = &bytes.Buffer{}
		}
		client.Logs[et].Write([]byte(parameters["log"].(arvadosclient.Dict)["properties"].(map[string]string)["text"]))
	}

	if resourceType == "collections" && output != nil {
		mt := parameters["collection"].(arvadosclient.Dict)["manifest_text"].(string)
		outmap := output.(*arvados.Collection)
		outmap.PortableDataHash = fmt.Sprintf("%x+%d", md5.Sum([]byte(mt)), len(mt))
	}

	return nil
}

func (client *ArvTestClient) Call(method, resourceType, uuid, action string, parameters arvadosclient.Dict, output interface{}) error {
	switch {
	case method == "GET" && resourceType == "containers" && action == "auth":
		return json.Unmarshal([]byte(`{
			"kind": "arvados#api_client_authorization",
			"uuid": "`+fakeAuthUUID+`",
			"api_token": "`+fakeAuthToken+`"
			}`), output)
	default:
		return fmt.Errorf("Not found")
	}
}

func (client *ArvTestClient) Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error {
	if resourceType == "collections" {
		if uuid == hwPDH {
			output.(*arvados.Collection).ManifestText = hwManifest
		} else if uuid == otherPDH {
			output.(*arvados.Collection).ManifestText = otherManifest
		}
	}
	if resourceType == "containers" {
		(*output.(*arvados.Container)) = client.Container
	}
	return nil
}

func (client *ArvTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	client.Mutex.Lock()
	defer client.Mutex.Unlock()
	client.Calls++
	client.Content = append(client.Content, parameters)
	if resourceType == "containers" {
		if parameters["container"].(arvadosclient.Dict)["state"] == "Running" {
			client.WasSetRunning = true
		}
	}
	return nil
}

// CalledWith returns the parameters from the first API call whose
// parameters match jpath/string. E.g., CalledWith(c, "foo.bar",
// "baz") returns parameters with parameters["foo"]["bar"]=="baz". If
// no call matches, it returns nil.
func (client *ArvTestClient) CalledWith(jpath string, expect interface{}) arvadosclient.Dict {
call:
	for _, content := range client.Content {
		var v interface{} = content
		for _, k := range strings.Split(jpath, ".") {
			if dict, ok := v.(arvadosclient.Dict); !ok {
				continue call
			} else {
				v = dict[k]
			}
		}
		if v == expect {
			return content
		}
	}
	return nil
}

func (client *KeepTestClient) PutHB(hash string, buf []byte) (string, int, error) {
	client.Content = buf
	return fmt.Sprintf("%s+%d", hash, len(buf)), len(buf), nil
}

type FileWrapper struct {
	io.ReadCloser
	len uint64
}

func (fw FileWrapper) Len() uint64 {
	return fw.len
}

func (client *KeepTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error) {
	if filename == hwImageId+".tar" {
		rdr := ioutil.NopCloser(&bytes.Buffer{})
		client.Called = true
		return FileWrapper{rdr, 1321984}, nil
	}
	return nil, nil
}

func (s *TestSuite) TestLoadImage(c *C) {
	kc := &KeepTestClient{}
	docker := NewTestDockerClient()
	cr := NewContainerRunner(&ArvTestClient{}, kc, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")

	_, err := cr.Docker.RemoveImage(hwImageId, true)

	_, err = cr.Docker.InspectImage(hwImageId)
	c.Check(err, NotNil)

	cr.Container.ContainerImage = hwPDH

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

func (ArvErrorTestClient) Create(resourceType string,
	parameters arvadosclient.Dict,
	output interface{}) error {
	return nil
}

func (ArvErrorTestClient) Call(method, resourceType, uuid, action string, parameters arvadosclient.Dict, output interface{}) error {
	return errors.New("ArvError")
}

func (ArvErrorTestClient) Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error {
	return errors.New("ArvError")
}

func (ArvErrorTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

type KeepErrorTestClient struct{}

func (KeepErrorTestClient) PutHB(hash string, buf []byte) (string, int, error) {
	return "", 0, errors.New("KeepError")
}

func (KeepErrorTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error) {
	return nil, errors.New("KeepError")
}

type KeepReadErrorTestClient struct{}

func (KeepReadErrorTestClient) PutHB(hash string, buf []byte) (string, int, error) {
	return "", 0, nil
}

type ErrorReader struct{}

func (ErrorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("ErrorReader")
}

func (ErrorReader) Close() error {
	return nil
}

func (ErrorReader) Len() uint64 {
	return 0
}

func (KeepReadErrorTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.ReadCloserWithLen, error) {
	return ErrorReader{}, nil
}

func (s *TestSuite) TestLoadImageArvError(c *C) {
	// (1) Arvados error
	cr := NewContainerRunner(ArvErrorTestClient{}, &KeepTestClient{}, nil, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.Container.ContainerImage = hwPDH

	err := cr.LoadImage()
	c.Check(err.Error(), Equals, "While getting container image collection: ArvError")
}

func (s *TestSuite) TestLoadImageKeepError(c *C) {
	// (2) Keep error
	docker := NewTestDockerClient()
	cr := NewContainerRunner(&ArvTestClient{}, KeepErrorTestClient{}, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.Container.ContainerImage = hwPDH

	err := cr.LoadImage()
	c.Check(err.Error(), Equals, "While creating ManifestFileReader for container image: KeepError")
}

func (s *TestSuite) TestLoadImageCollectionError(c *C) {
	// (3) Collection doesn't contain image
	cr := NewContainerRunner(&ArvTestClient{}, KeepErrorTestClient{}, nil, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.Container.ContainerImage = otherPDH

	err := cr.LoadImage()
	c.Check(err.Error(), Equals, "First file in the container image collection does not end in .tar")
}

func (s *TestSuite) TestLoadImageKeepReadError(c *C) {
	// (4) Collection doesn't contain image
	docker := NewTestDockerClient()
	cr := NewContainerRunner(&ArvTestClient{}, KeepReadErrorTestClient{}, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.Container.ContainerImage = hwPDH

	err := cr.LoadImage()
	c.Check(err, NotNil)
}

type ClosableBuffer struct {
	bytes.Buffer
}

func (*ClosableBuffer) Close() error {
	return nil
}

type TestLogs struct {
	Stdout ClosableBuffer
	Stderr ClosableBuffer
}

func (tl *TestLogs) NewTestLoggingWriter(logstr string) io.WriteCloser {
	if logstr == "stdout" {
		return &tl.Stdout
	}
	if logstr == "stderr" {
		return &tl.Stderr
	}
	return nil
}

func dockerLog(fd byte, msg string) []byte {
	by := []byte(msg)
	header := make([]byte, 8+len(by))
	header[0] = fd
	header[7] = byte(len(by))
	copy(header[8:], by)
	return header
}

func (s *TestSuite) TestRunContainer(c *C) {
	docker := NewTestDockerClient()
	docker.fn = func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, "Hello world\n"))
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{}
	}
	cr := NewContainerRunner(&ArvTestClient{}, &KeepTestClient{}, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")

	var logs TestLogs
	cr.NewLogWriter = logs.NewTestLoggingWriter
	cr.Container.ContainerImage = hwPDH
	cr.Container.Command = []string{"./hw"}
	err := cr.LoadImage()
	c.Check(err, IsNil)

	err = cr.CreateContainer()
	c.Check(err, IsNil)

	err = cr.StartContainer()
	c.Check(err, IsNil)

	err = cr.WaitFinish()
	c.Check(err, IsNil)

	c.Check(strings.HasSuffix(logs.Stdout.String(), "Hello world\n"), Equals, true)
	c.Check(logs.Stderr.String(), Equals, "")
}

func (s *TestSuite) TestCommitLogs(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.CrunchLog.Timestamper = (&TestTimestamper{}).Timestamp

	cr.CrunchLog.Print("Hello world!")
	cr.CrunchLog.Print("Goodbye")
	cr.finalState = "Complete"

	err := cr.CommitLogs()
	c.Check(err, IsNil)

	c.Check(api.Calls, Equals, 2)
	c.Check(api.Content[1]["collection"].(arvadosclient.Dict)["name"], Equals, "logs for zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Check(api.Content[1]["collection"].(arvadosclient.Dict)["manifest_text"], Equals, ". 744b2e4553123b02fa7b452ec5c18993+123 0:123:crunch-run.txt\n")
	c.Check(*cr.LogsPDH, Equals, "63da7bdacf08c40f604daad80c261e9a+60")
}

func (s *TestSuite) TestUpdateContainerRunning(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")

	err := cr.UpdateContainerRunning()
	c.Check(err, IsNil)

	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["state"], Equals, "Running")
}

func (s *TestSuite) TestUpdateContainerComplete(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")

	cr.LogsPDH = new(string)
	*cr.LogsPDH = "d3a229d2fe3690c2c3e75a71a153c6a3+60"

	cr.ExitCode = new(int)
	*cr.ExitCode = 42
	cr.finalState = "Complete"

	err := cr.UpdateContainerFinal()
	c.Check(err, IsNil)

	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["log"], Equals, *cr.LogsPDH)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["exit_code"], Equals, *cr.ExitCode)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["state"], Equals, "Complete")
}

func (s *TestSuite) TestUpdateContainerCancelled(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.Cancelled = true
	cr.finalState = "Cancelled"

	err := cr.UpdateContainerFinal()
	c.Check(err, IsNil)

	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["log"], IsNil)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["exit_code"], IsNil)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["state"], Equals, "Cancelled")
}

// Used by the TestFullRun*() test below to DRY up boilerplate setup to do full
// dress rehearsal of the Run() function, starting from a JSON container record.
func FullRunHelper(c *C, record string, fn func(t *TestDockerClient)) (api *ArvTestClient, cr *ContainerRunner) {
	rec := arvados.Container{}
	err := json.Unmarshal([]byte(record), &rec)
	c.Check(err, IsNil)

	docker := NewTestDockerClient()
	docker.fn = fn
	docker.RemoveImage(hwImageId, true)

	api = &ArvTestClient{Container: rec}
	cr = NewContainerRunner(api, &KeepTestClient{}, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.statInterval = 100 * time.Millisecond
	am := &ArvMountCmdLine{}
	cr.RunArvMount = am.ArvMountTest

	err = cr.Run()
	c.Check(err, IsNil)
	c.Check(api.WasSetRunning, Equals, true)

	c.Check(api.Content[api.Calls-1]["container"].(arvadosclient.Dict)["log"], NotNil)

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
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, "hello world\n"))
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{}
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "hello world\n"), Equals, true)

}

func (s *TestSuite) TestCrunchstat(c *C) {
	api, _ := FullRunHelper(c, `{
		"command": ["sleep", "1"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`, func(t *TestDockerClient) {
		time.Sleep(time.Second)
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{}
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)

	// We didn't actually start a container, so crunchstat didn't
	// find accounting files and therefore didn't log any stats.
	// It should have logged a "can't find accounting files"
	// message after one poll interval, though, so we can confirm
	// it's alive:
	c.Assert(api.Logs["crunchstat"], NotNil)
	c.Check(api.Logs["crunchstat"].String(), Matches, `(?ms).*cgroup stats files have not appeared after 100ms.*`)

	// The "files never appeared" log assures us that we called
	// (*crunchstat.Reporter)Stop(), and that we set it up with
	// the correct container ID "abcde":
	c.Check(api.Logs["crunchstat"].String(), Matches, `(?ms).*cgroup stats files never appeared for abcde\n`)
}

func (s *TestSuite) TestFullRunStderr(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo hello ; echo world 1>&2 ; exit 1"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, "hello\n"))
		t.logWriter.Write(dockerLog(2, "world\n"))
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 1}
	})

	final := api.CalledWith("container.state", "Complete")
	c.Assert(final, NotNil)
	c.Check(final["container"].(arvadosclient.Dict)["exit_code"], Equals, 1)
	c.Check(final["container"].(arvadosclient.Dict)["log"], NotNil)

	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "hello\n"), Equals, true)
	c.Check(strings.HasSuffix(api.Logs["stderr"].String(), "world\n"), Equals, true)
}

func (s *TestSuite) TestFullRunDefaultCwd(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.cwd+"\n"))
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Log(api.Logs["stdout"])
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "/\n"), Equals, true)
}

func (s *TestSuite) TestFullRunSetCwd(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.cwd+"\n"))
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "/bin\n"), Equals, true)
}

func (s *TestSuite) TestCancel(c *C) {
	record := `{
    "command": ["/bin/sh", "-c", "echo foo && sleep 30 && echo bar"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`

	rec := arvados.Container{}
	err := json.Unmarshal([]byte(record), &rec)
	c.Check(err, IsNil)

	docker := NewTestDockerClient()
	docker.fn = func(t *TestDockerClient) {
		<-t.stop
		t.logWriter.Write(dockerLog(1, "foo\n"))
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	}
	docker.RemoveImage(hwImageId, true)

	api := &ArvTestClient{Container: rec}
	cr := NewContainerRunner(api, &KeepTestClient{}, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	am := &ArvMountCmdLine{}
	cr.RunArvMount = am.ArvMountTest

	go func() {
		for cr.ContainerID == "" {
			time.Sleep(time.Millisecond)
		}
		cr.SigChan <- syscall.SIGINT
	}()

	err = cr.Run()

	c.Check(err, IsNil)
	if err != nil {
		for k, v := range api.Logs {
			c.Log(k)
			c.Log(v.String())
		}
	}

	c.Check(api.CalledWith("container.log", nil), NotNil)
	c.Check(api.CalledWith("container.state", "Cancelled"), NotNil)
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "foo\n"), Equals, true)

}

func (s *TestSuite) TestFullRunSetEnv(c *C) {
	api, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo $FROBIZ"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": "/bin",
    "environment": {"FROBIZ": "bilbo"},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "bilbo\n"), Equals, true)
}

type ArvMountCmdLine struct {
	Cmd   []string
	token string
}

func (am *ArvMountCmdLine) ArvMountTest(c []string, token string) (*exec.Cmd, error) {
	am.Cmd = c
	am.token = token
	return nil, nil
}

func (s *TestSuite) TestSetupMounts(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	am := &ArvMountCmdLine{}
	cr.RunArvMount = am.ArvMountTest

	realTemp, err := ioutil.TempDir("", "crunchrun_test-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(realTemp)

	i := 0
	cr.MkTempDir = func(_ string, prefix string) (string, error) {
		i++
		d := fmt.Sprintf("%s/%s%d", realTemp, prefix, i)
		err := os.Mkdir(d, os.ModePerm)
		if err != nil && strings.Contains(err.Error(), ": file exists") {
			// Test case must have pre-populated the tempdir
			err = nil
		}
		return d, err
	}

	checkEmpty := func() {
		filepath.Walk(realTemp, func(path string, _ os.FileInfo, err error) error {
			c.Check(path, Equals, realTemp)
			c.Check(err, IsNil)
			return nil
		})
	}

	{
		i = 0
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts["/tmp"] = arvados.Mount{Kind: "tmp"}
		cr.OutputPath = "/tmp"

		err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"--foreground", "--allow-other", "--read-write", "--mount-by-pdh", "by_id", realTemp + "/keep1"})
		c.Check(cr.Binds, DeepEquals, []string{realTemp + "/2:/tmp"})
		cr.CleanupDirs()
		checkEmpty()
	}

	{
		i = 0
		cr.Container.Mounts = map[string]arvados.Mount{
			"/keeptmp": {Kind: "collection", Writable: true},
		}
		cr.OutputPath = "/keeptmp"

		os.MkdirAll(realTemp+"/keep1/tmp0", os.ModePerm)

		err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"--foreground", "--allow-other", "--read-write", "--mount-tmp", "tmp0", "--mount-by-pdh", "by_id", realTemp + "/keep1"})
		c.Check(cr.Binds, DeepEquals, []string{realTemp + "/keep1/tmp0:/keeptmp"})
		cr.CleanupDirs()
		checkEmpty()
	}

	{
		i = 0
		cr.Container.Mounts = map[string]arvados.Mount{
			"/keepinp": {Kind: "collection", PortableDataHash: "59389a8f9ee9d399be35462a0f92541c+53"},
			"/keepout": {Kind: "collection", Writable: true},
		}
		cr.OutputPath = "/keepout"

		os.MkdirAll(realTemp+"/keep1/by_id/59389a8f9ee9d399be35462a0f92541c+53", os.ModePerm)
		os.MkdirAll(realTemp+"/keep1/tmp0", os.ModePerm)

		err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"--foreground", "--allow-other", "--read-write", "--mount-tmp", "tmp0", "--mount-by-pdh", "by_id", realTemp + "/keep1"})
		sort.StringSlice(cr.Binds).Sort()
		c.Check(cr.Binds, DeepEquals, []string{realTemp + "/keep1/by_id/59389a8f9ee9d399be35462a0f92541c+53:/keepinp:ro",
			realTemp + "/keep1/tmp0:/keepout"})
		cr.CleanupDirs()
		checkEmpty()
	}

	for _, test := range []struct {
		in  interface{}
		out string
	}{
		{in: "foo", out: `"foo"`},
		{in: nil, out: `null`},
		{in: map[string]int{"foo": 123}, out: `{"foo":123}`},
	} {
		i = 0
		cr.Container.Mounts = map[string]arvados.Mount{
			"/mnt/test.json": {Kind: "json", Content: test.in},
		}
		err := cr.SetupMounts()
		c.Check(err, IsNil)
		sort.StringSlice(cr.Binds).Sort()
		c.Check(cr.Binds, DeepEquals, []string{realTemp + "/2/mountdata.json:/mnt/test.json:ro"})
		content, err := ioutil.ReadFile(realTemp + "/2/mountdata.json")
		c.Check(err, IsNil)
		c.Check(content, DeepEquals, []byte(test.out))
		cr.CleanupDirs()
		checkEmpty()
	}
}

func (s *TestSuite) TestStdout(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {"/tmp": {"kind": "tmp"}, "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`

	api, _ := FullRunHelper(c, helperRecord, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
		t.finish <- dockerclient.WaitResult{ExitCode: 0}
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(api.CalledWith("collection.manifest_text", "./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out\n"), NotNil)
}

// Used by the TestStdoutWithWrongPath*()
func StdoutErrorRunHelper(c *C, record string, fn func(t *TestDockerClient)) (api *ArvTestClient, cr *ContainerRunner, err error) {
	rec := arvados.Container{}
	err = json.Unmarshal([]byte(record), &rec)
	c.Check(err, IsNil)

	docker := NewTestDockerClient()
	docker.fn = fn
	docker.RemoveImage(hwImageId, true)

	api = &ArvTestClient{Container: rec}
	cr = NewContainerRunner(api, &KeepTestClient{}, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	am := &ArvMountCmdLine{}
	cr.RunArvMount = am.ArvMountTest

	err = cr.Run()
	return
}

func (s *TestSuite) TestStdoutWithWrongPath(c *C) {
	_, _, err := StdoutErrorRunHelper(c, `{
    "mounts": {"/tmp": {"kind": "tmp"}, "stdout": {"kind": "file", "path":"/tmpa.out"} },
    "output_path": "/tmp"
}`, func(t *TestDockerClient) {})

	c.Check(err, NotNil)
	c.Check(strings.Contains(err.Error(), "Stdout path does not start with OutputPath"), Equals, true)
}

func (s *TestSuite) TestStdoutWithWrongKindTmp(c *C) {
	_, _, err := StdoutErrorRunHelper(c, `{
    "mounts": {"/tmp": {"kind": "tmp"}, "stdout": {"kind": "tmp", "path":"/tmp/a.out"} },
    "output_path": "/tmp"
}`, func(t *TestDockerClient) {})

	c.Check(err, NotNil)
	c.Check(strings.Contains(err.Error(), "Unsupported mount kind 'tmp' for stdout"), Equals, true)
}

func (s *TestSuite) TestStdoutWithWrongKindCollection(c *C) {
	_, _, err := StdoutErrorRunHelper(c, `{
    "mounts": {"/tmp": {"kind": "tmp"}, "stdout": {"kind": "collection", "path":"/tmp/a.out"} },
    "output_path": "/tmp"
}`, func(t *TestDockerClient) {})

	c.Check(err, NotNil)
	c.Check(strings.Contains(err.Error(), "Unsupported mount kind 'collection' for stdout"), Equals, true)
}
