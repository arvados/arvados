package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/manifest"

	dockertypes "github.com/docker/docker/api/types"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockernetwork "github.com/docker/docker/api/types/network"
	. "gopkg.in/check.v1"
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
	Logs map[string]*bytes.Buffer
	sync.Mutex
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

var normalizedManifestWithSubdirs = ". 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0abcdefgh11234567890@569fa8c3 0:9:file1_in_main.txt 9:18:file2_in_main.txt 0:27:zzzzz-8i9sb-bcdefghijkdhvnk.log.txt\n./subdir1 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396cabcdefghij6419876543234@569fa8c4 0:9:file1_in_subdir1.txt 9:18:file2_in_subdir1.txt\n./subdir1/subdir2 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0bcdefghijk544332211@569fa8c5 0:9:file1_in_subdir2.txt 9:18:file2_in_subdir2.txt\n"
var normalizedWithSubdirsPDH = "a0def87f80dd594d4675809e83bd4f15+367"

var denormalizedManifestWithSubdirs = ". 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0abcdefgh11234567890@569fa8c3 0:9:file1_in_main.txt 9:18:file2_in_main.txt 0:27:zzzzz-8i9sb-bcdefghijkdhvnk.log.txt 0:10:subdir1/file1_in_subdir1.txt 10:17:subdir1/file2_in_subdir1.txt\n"
var denormalizedWithSubdirsPDH = "b0def87f80dd594d4675809e83bd4f15+367"

var fakeAuthUUID = "zzzzz-gj3su-55pqoyepgi2glem"
var fakeAuthToken = "a3ltuwzqcu2u4sc0q7yhpc2w7s00fdcqecg5d6e0u3pfohmbjt"

type TestDockerClient struct {
	imageLoaded string
	logReader   io.ReadCloser
	logWriter   io.WriteCloser
	fn          func(t *TestDockerClient)
	finish      int
	stop        chan bool
	cwd         string
	env         []string
	api         *ArvTestClient
}

func NewTestDockerClient(exitCode int) *TestDockerClient {
	t := &TestDockerClient{}
	t.logReader, t.logWriter = io.Pipe()
	t.finish = exitCode
	t.stop = make(chan bool)
	t.cwd = "/"
	return t
}

type MockConn struct {
	net.Conn
}

func (m *MockConn) Write(b []byte) (int, error) {
	return len(b), nil
}

func NewMockConn() *MockConn {
	c := &MockConn{}
	return c
}

func (t *TestDockerClient) ContainerAttach(ctx context.Context, container string, options dockertypes.ContainerAttachOptions) (dockertypes.HijackedResponse, error) {
	return dockertypes.HijackedResponse{Conn: NewMockConn(), Reader: bufio.NewReader(t.logReader)}, nil
}

func (t *TestDockerClient) ContainerCreate(ctx context.Context, config *dockercontainer.Config, hostConfig *dockercontainer.HostConfig, networkingConfig *dockernetwork.NetworkingConfig, containerName string) (dockercontainer.ContainerCreateCreatedBody, error) {
	if config.WorkingDir != "" {
		t.cwd = config.WorkingDir
	}
	t.env = config.Env
	return dockercontainer.ContainerCreateCreatedBody{ID: "abcde"}, nil
}

func (t *TestDockerClient) ContainerStart(ctx context.Context, container string, options dockertypes.ContainerStartOptions) error {
	if container == "abcde" {
		go t.fn(t)
		return nil
	} else {
		return errors.New("Invalid container id")
	}
}

func (t *TestDockerClient) ContainerStop(ctx context.Context, container string, timeout *time.Duration) error {
	t.stop <- true
	return nil
}

func (t *TestDockerClient) ContainerWait(ctx context.Context, container string) (int64, error) {
	return int64(t.finish), nil
}

func (t *TestDockerClient) ImageInspectWithRaw(ctx context.Context, image string) (dockertypes.ImageInspect, []byte, error) {
	if t.imageLoaded == image {
		return dockertypes.ImageInspect{}, nil, nil
	} else {
		return dockertypes.ImageInspect{}, nil, errors.New("")
	}
}

func (t *TestDockerClient) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (dockertypes.ImageLoadResponse, error) {
	_, err := io.Copy(ioutil.Discard, input)
	if err != nil {
		return dockertypes.ImageLoadResponse{}, err
	} else {
		t.imageLoaded = hwImageId
		return dockertypes.ImageLoadResponse{Body: ioutil.NopCloser(input)}, nil
	}
}

func (*TestDockerClient) ImageRemove(ctx context.Context, image string, options dockertypes.ImageRemoveOptions) ([]dockertypes.ImageDeleteResponseItem, error) {
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

func (client *ArvTestClient) CallRaw(method, resourceType, uuid, action string,
	parameters arvadosclient.Dict) (reader io.ReadCloser, err error) {
	j := []byte(`{
		"command": ["sleep", "1"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`)
	return ioutil.NopCloser(bytes.NewReader(j)), nil
}

func (client *ArvTestClient) Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error {
	if resourceType == "collections" {
		if uuid == hwPDH {
			output.(*arvados.Collection).ManifestText = hwManifest
		} else if uuid == otherPDH {
			output.(*arvados.Collection).ManifestText = otherManifest
		} else if uuid == normalizedWithSubdirsPDH {
			output.(*arvados.Collection).ManifestText = normalizedManifestWithSubdirs
		} else if uuid == denormalizedWithSubdirsPDH {
			output.(*arvados.Collection).ManifestText = denormalizedManifestWithSubdirs
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

var discoveryMap = map[string]interface{}{"defaultTrashLifetime": float64(1209600)}

func (client *ArvTestClient) Discovery(key string) (interface{}, error) {
	return discoveryMap[key], nil
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

func (fw FileWrapper) Seek(int64, int) (int64, error) {
	return 0, errors.New("not implemented")
}

func (client *KeepTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.Reader, error) {
	if filename == hwImageId+".tar" {
		rdr := ioutil.NopCloser(&bytes.Buffer{})
		client.Called = true
		return FileWrapper{rdr, 1321984}, nil
	} else if filename == "/file1_in_main.txt" {
		rdr := ioutil.NopCloser(strings.NewReader("foo"))
		client.Called = true
		return FileWrapper{rdr, 3}, nil
	}
	return nil, nil
}

func (s *TestSuite) TestLoadImage(c *C) {
	kc := &KeepTestClient{}
	docker := NewTestDockerClient(0)
	cr := NewContainerRunner(&ArvTestClient{}, kc, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")

	_, err := cr.Docker.ImageRemove(nil, hwImageId, dockertypes.ImageRemoveOptions{})

	_, _, err = cr.Docker.ImageInspectWithRaw(nil, hwImageId)
	c.Check(err, NotNil)

	cr.Container.ContainerImage = hwPDH

	// (1) Test loading image from keep
	c.Check(kc.Called, Equals, false)
	c.Check(cr.ContainerConfig.Image, Equals, "")

	err = cr.LoadImage()

	c.Check(err, IsNil)
	defer func() {
		cr.Docker.ImageRemove(nil, hwImageId, dockertypes.ImageRemoveOptions{})
	}()

	c.Check(kc.Called, Equals, true)
	c.Check(cr.ContainerConfig.Image, Equals, hwImageId)

	_, _, err = cr.Docker.ImageInspectWithRaw(nil, hwImageId)
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

func (ArvErrorTestClient) CallRaw(method, resourceType, uuid, action string,
	parameters arvadosclient.Dict) (reader io.ReadCloser, err error) {
	return nil, errors.New("ArvError")
}

func (ArvErrorTestClient) Get(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) error {
	return errors.New("ArvError")
}

func (ArvErrorTestClient) Update(resourceType string, uuid string, parameters arvadosclient.Dict, output interface{}) (err error) {
	return nil
}

func (ArvErrorTestClient) Discovery(key string) (interface{}, error) {
	return discoveryMap[key], nil
}

type KeepErrorTestClient struct{}

func (KeepErrorTestClient) PutHB(hash string, buf []byte) (string, int, error) {
	return "", 0, errors.New("KeepError")
}

func (KeepErrorTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.Reader, error) {
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

func (ErrorReader) Seek(int64, int) (int64, error) {
	return 0, errors.New("ErrorReader")
}

func (KeepReadErrorTestClient) ManifestFileReader(m manifest.Manifest, filename string) (keepclient.Reader, error) {
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
	docker := NewTestDockerClient(0)
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
	docker := NewTestDockerClient(0)
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
	docker := NewTestDockerClient(0)
	docker.fn = func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, "Hello world\n"))
		t.logWriter.Close()
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
	c.Check(api.Content[1]["ensure_unique_name"], Equals, true)
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
	cr.cCancelled = true
	cr.finalState = "Cancelled"

	err := cr.UpdateContainerFinal()
	c.Check(err, IsNil)

	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["log"], IsNil)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["exit_code"], IsNil)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["state"], Equals, "Cancelled")
}

// Used by the TestFullRun*() test below to DRY up boilerplate setup to do full
// dress rehearsal of the Run() function, starting from a JSON container record.
func FullRunHelper(c *C, record string, extraMounts []string, exitCode int, fn func(t *TestDockerClient)) (api *ArvTestClient, cr *ContainerRunner, realTemp string) {
	rec := arvados.Container{}
	err := json.Unmarshal([]byte(record), &rec)
	c.Check(err, IsNil)

	docker := NewTestDockerClient(exitCode)
	docker.fn = fn
	docker.ImageRemove(nil, hwImageId, dockertypes.ImageRemoveOptions{})

	api = &ArvTestClient{Container: rec}
	docker.api = api
	cr = NewContainerRunner(api, &KeepTestClient{}, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.statInterval = 100 * time.Millisecond
	am := &ArvMountCmdLine{}
	cr.RunArvMount = am.ArvMountTest

	realTemp, err = ioutil.TempDir("", "crunchrun_test1-")
	c.Assert(err, IsNil)
	defer os.RemoveAll(realTemp)

	tempcount := 0
	cr.MkTempDir = func(_ string, prefix string) (string, error) {
		tempcount++
		d := fmt.Sprintf("%s/%s%d", realTemp, prefix, tempcount)
		err := os.Mkdir(d, os.ModePerm)
		if err != nil && strings.Contains(err.Error(), ": file exists") {
			// Test case must have pre-populated the tempdir
			err = nil
		}
		return d, err
	}

	if extraMounts != nil && len(extraMounts) > 0 {
		err := cr.SetupArvMountPoint("keep")
		c.Check(err, IsNil)

		for _, m := range extraMounts {
			os.MkdirAll(cr.ArvMountPoint+"/by_id/"+m, os.ModePerm)
		}
	}

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
	api, _, _ := FullRunHelper(c, `{
    "command": ["echo", "hello world"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, nil, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, "hello world\n"))
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "hello world\n"), Equals, true)

}

func (s *TestSuite) TestCrunchstat(c *C) {
	api, _, _ := FullRunHelper(c, `{
		"command": ["sleep", "1"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`, nil, 0, func(t *TestDockerClient) {
		time.Sleep(time.Second)
		t.logWriter.Close()
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

func (s *TestSuite) TestNodeInfoLog(c *C) {
	api, _, _ := FullRunHelper(c, `{
		"command": ["sleep", "1"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`, nil, 0,
		func(t *TestDockerClient) {
			time.Sleep(time.Second)
			t.logWriter.Close()
		})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)

	c.Assert(api.Logs["node-info"], NotNil)
	c.Check(api.Logs["node-info"].String(), Matches, `(?ms).*Host Information.*`)
	c.Check(api.Logs["node-info"].String(), Matches, `(?ms).*CPU Information.*`)
	c.Check(api.Logs["node-info"].String(), Matches, `(?ms).*Memory Information.*`)
	c.Check(api.Logs["node-info"].String(), Matches, `(?ms).*Disk Space.*`)
	c.Check(api.Logs["node-info"].String(), Matches, `(?ms).*Disk INodes.*`)
}

func (s *TestSuite) TestContainerRecordLog(c *C) {
	api, _, _ := FullRunHelper(c, `{
		"command": ["sleep", "1"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`, nil, 0,
		func(t *TestDockerClient) {
			time.Sleep(time.Second)
			t.logWriter.Close()
		})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)

	c.Assert(api.Logs["container"], NotNil)
	c.Check(api.Logs["container"].String(), Matches, `(?ms).*container_image.*`)
}

func (s *TestSuite) TestFullRunStderr(c *C) {
	api, _, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo hello ; echo world 1>&2 ; exit 1"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, nil, 1, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, "hello\n"))
		t.logWriter.Write(dockerLog(2, "world\n"))
		t.logWriter.Close()
	})

	final := api.CalledWith("container.state", "Complete")
	c.Assert(final, NotNil)
	c.Check(final["container"].(arvadosclient.Dict)["exit_code"], Equals, 1)
	c.Check(final["container"].(arvadosclient.Dict)["log"], NotNil)

	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "hello\n"), Equals, true)
	c.Check(strings.HasSuffix(api.Logs["stderr"].String(), "world\n"), Equals, true)
}

func (s *TestSuite) TestFullRunDefaultCwd(c *C) {
	api, _, _ := FullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, nil, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.cwd+"\n"))
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Log(api.Logs["stdout"])
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "/\n"), Equals, true)
}

func (s *TestSuite) TestFullRunSetCwd(c *C) {
	api, _, _ := FullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, nil, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.cwd+"\n"))
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "/bin\n"), Equals, true)
}

func (s *TestSuite) TestStopOnSignal(c *C) {
	s.testStopContainer(c, func(cr *ContainerRunner) {
		go func() {
			for !cr.cStarted {
				time.Sleep(time.Millisecond)
			}
			cr.SigChan <- syscall.SIGINT
		}()
	})
}

func (s *TestSuite) TestStopOnArvMountDeath(c *C) {
	s.testStopContainer(c, func(cr *ContainerRunner) {
		cr.ArvMountExit = make(chan error)
		go func() {
			cr.ArvMountExit <- exec.Command("true").Run()
			close(cr.ArvMountExit)
		}()
	})
}

func (s *TestSuite) testStopContainer(c *C, setup func(cr *ContainerRunner)) {
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

	docker := NewTestDockerClient(0)
	docker.fn = func(t *TestDockerClient) {
		<-t.stop
		t.logWriter.Write(dockerLog(1, "foo\n"))
		t.logWriter.Close()
	}
	docker.ImageRemove(nil, hwImageId, dockertypes.ImageRemoveOptions{})

	api := &ArvTestClient{Container: rec}
	cr := NewContainerRunner(api, &KeepTestClient{}, docker, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	cr.RunArvMount = func([]string, string) (*exec.Cmd, error) { return nil, nil }
	setup(cr)

	done := make(chan error)
	go func() {
		done <- cr.Run()
	}()
	select {
	case <-time.After(20 * time.Second):
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
		c.Fatal("timed out")
	case err = <-done:
		c.Check(err, IsNil)
	}
	for k, v := range api.Logs {
		c.Log(k)
		c.Log(v.String())
	}

	c.Check(api.CalledWith("container.log", nil), NotNil)
	c.Check(api.CalledWith("container.state", "Cancelled"), NotNil)
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "foo\n"), Equals, true)
}

func (s *TestSuite) TestFullRunSetEnv(c *C) {
	api, _, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo $FROBIZ"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": "/bin",
    "environment": {"FROBIZ": "bilbo"},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, nil, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
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

func stubCert(temp string) string {
	path := temp + "/ca-certificates.crt"
	crt, _ := os.Create(path)
	crt.Close()
	arvadosclient.CertFiles = []string{path}
	return path
}

func (s *TestSuite) TestSetupMounts(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	cr := NewContainerRunner(api, kc, nil, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	am := &ArvMountCmdLine{}
	cr.RunArvMount = am.ArvMountTest

	realTemp, err := ioutil.TempDir("", "crunchrun_test1-")
	c.Assert(err, IsNil)
	certTemp, err := ioutil.TempDir("", "crunchrun_test2-")
	c.Assert(err, IsNil)
	stubCertPath := stubCert(certTemp)

	defer os.RemoveAll(realTemp)
	defer os.RemoveAll(certTemp)

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
		cr.ArvMountPoint = ""
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
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts["/tmp"] = arvados.Mount{Kind: "tmp"}
		cr.OutputPath = "/tmp"

		apiflag := true
		cr.Container.RuntimeConstraints.API = &apiflag

		err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"--foreground", "--allow-other", "--read-write", "--mount-by-pdh", "by_id", realTemp + "/keep1"})
		c.Check(cr.Binds, DeepEquals, []string{realTemp + "/2:/tmp", stubCertPath + ":/etc/arvados/ca-certificates.crt:ro"})
		cr.CleanupDirs()
		checkEmpty()

		apiflag = false
	}

	{
		i = 0
		cr.ArvMountPoint = ""
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
		cr.ArvMountPoint = ""
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

	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.RuntimeConstraints.KeepCacheRAM = 512
		cr.Container.Mounts = map[string]arvados.Mount{
			"/keepinp": {Kind: "collection", PortableDataHash: "59389a8f9ee9d399be35462a0f92541c+53"},
			"/keepout": {Kind: "collection", Writable: true},
		}
		cr.OutputPath = "/keepout"

		os.MkdirAll(realTemp+"/keep1/by_id/59389a8f9ee9d399be35462a0f92541c+53", os.ModePerm)
		os.MkdirAll(realTemp+"/keep1/tmp0", os.ModePerm)

		err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"--foreground", "--allow-other", "--read-write", "--file-cache", "512", "--mount-tmp", "tmp0", "--mount-by-pdh", "by_id", realTemp + "/keep1"})
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
		cr.ArvMountPoint = ""
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

	// Read-only mount points are allowed underneath output_dir mount point
	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts = map[string]arvados.Mount{
			"/tmp":     {Kind: "tmp"},
			"/tmp/foo": {Kind: "collection"},
		}
		cr.OutputPath = "/tmp"

		os.MkdirAll(realTemp+"/keep1/tmp0", os.ModePerm)

		err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"--foreground", "--allow-other", "--read-write", "--file-cache", "512", "--mount-tmp", "tmp0", "--mount-by-pdh", "by_id", realTemp + "/keep1"})
		c.Check(cr.Binds, DeepEquals, []string{realTemp + "/2:/tmp", realTemp + "/keep1/tmp0:/tmp/foo:ro"})
		cr.CleanupDirs()
		checkEmpty()
	}

	// Writable mount points are not allowed underneath output_dir mount point
	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts = map[string]arvados.Mount{
			"/tmp":     {Kind: "tmp"},
			"/tmp/foo": {Kind: "collection", Writable: true},
		}
		cr.OutputPath = "/tmp"

		err := cr.SetupMounts()
		c.Check(err, NotNil)
		c.Check(err, ErrorMatches, `Writable mount points are not permitted underneath the output_path.*`)
		cr.CleanupDirs()
		checkEmpty()
	}

	// Only mount points of kind 'collection' are allowed underneath output_dir mount point
	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts = map[string]arvados.Mount{
			"/tmp":     {Kind: "tmp"},
			"/tmp/foo": {Kind: "json"},
		}
		cr.OutputPath = "/tmp"

		err := cr.SetupMounts()
		c.Check(err, NotNil)
		c.Check(err, ErrorMatches, `Only mount points of kind 'collection' are supported underneath the output_path.*`)
		cr.CleanupDirs()
		checkEmpty()
	}

	// Only mount point of kind 'collection' is allowed for stdin
	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts = map[string]arvados.Mount{
			"stdin": {Kind: "tmp"},
		}

		err := cr.SetupMounts()
		c.Check(err, NotNil)
		c.Check(err, ErrorMatches, `Unsupported mount kind 'tmp' for stdin.*`)
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

	api, _, _ := FullRunHelper(c, helperRecord, nil, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
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

	docker := NewTestDockerClient(0)
	docker.fn = fn
	docker.ImageRemove(nil, hwImageId, dockertypes.ImageRemoveOptions{})

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

func (s *TestSuite) TestFullRunWithAPI(c *C) {
	os.Setenv("ARVADOS_API_HOST", "test.arvados.org")
	defer os.Unsetenv("ARVADOS_API_HOST")
	api, _, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo $ARVADOS_API_HOST"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {"API": true}
}`, nil, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[1][17:]+"\n"))
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(strings.HasSuffix(api.Logs["stdout"].String(), "test.arvados.org\n"), Equals, true)
	c.Check(api.CalledWith("container.output", "d41d8cd98f00b204e9800998ecf8427e+0"), NotNil)
}

func (s *TestSuite) TestFullRunSetOutput(c *C) {
	os.Setenv("ARVADOS_API_HOST", "test.arvados.org")
	defer os.Unsetenv("ARVADOS_API_HOST")
	api, _, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo $ARVADOS_API_HOST"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {"API": true}
}`, nil, 0, func(t *TestDockerClient) {
		t.api.Container.Output = "d4ab34d3d4f8a72f5c4973051ae69fab+122"
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(api.CalledWith("container.output", "d4ab34d3d4f8a72f5c4973051ae69fab+122"), NotNil)
}

func (s *TestSuite) TestStdoutWithExcludeFromOutputMountPointUnderOutputDir(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
        "/tmp": {"kind": "tmp"},
        "/tmp/foo": {"kind": "collection",
                     "portable_data_hash": "a3e8f74c6f101eae01fa08bfb4e49b3a+54",
                     "exclude_from_output": true
        },
        "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"}
    },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`

	extraMounts := []string{"a3e8f74c6f101eae01fa08bfb4e49b3a+54"}

	api, _, _ := FullRunHelper(c, helperRecord, extraMounts, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(api.CalledWith("collection.manifest_text", "./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out\n"), NotNil)
}

func (s *TestSuite) TestStdoutWithMultipleMountPointsUnderOutputDir(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
        "/tmp": {"kind": "tmp"},
        "/tmp/foo/bar": {"kind": "collection", "portable_data_hash": "a0def87f80dd594d4675809e83bd4f15+367", "path":"/file2_in_main.txt"},
        "/tmp/foo/sub1": {"kind": "collection", "portable_data_hash": "a0def87f80dd594d4675809e83bd4f15+367", "path":"/subdir1"},
        "/tmp/foo/sub1file2": {"kind": "collection", "portable_data_hash": "a0def87f80dd594d4675809e83bd4f15+367", "path":"/subdir1/file2_in_subdir1.txt"},
        "/tmp/foo/baz/sub2file2": {"kind": "collection", "portable_data_hash": "a0def87f80dd594d4675809e83bd4f15+367", "path":"/subdir1/subdir2/file2_in_subdir2.txt"},
        "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"}
    },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`

	extraMounts := []string{
		"a0def87f80dd594d4675809e83bd4f15+367/file2_in_main.txt",
		"a0def87f80dd594d4675809e83bd4f15+367/subdir1/file2_in_subdir1.txt",
		"a0def87f80dd594d4675809e83bd4f15+367/subdir1/subdir2/file2_in_subdir2.txt",
	}

	api, runner, realtemp := FullRunHelper(c, helperRecord, extraMounts, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
	})

	c.Check(runner.Binds, DeepEquals, []string{realtemp + "/2:/tmp",
		realtemp + "/keep1/by_id/a0def87f80dd594d4675809e83bd4f15+367/file2_in_main.txt:/tmp/foo/bar:ro",
		realtemp + "/keep1/by_id/a0def87f80dd594d4675809e83bd4f15+367/subdir1/subdir2/file2_in_subdir2.txt:/tmp/foo/baz/sub2file2:ro",
		realtemp + "/keep1/by_id/a0def87f80dd594d4675809e83bd4f15+367/subdir1:/tmp/foo/sub1:ro",
		realtemp + "/keep1/by_id/a0def87f80dd594d4675809e83bd4f15+367/subdir1/file2_in_subdir1.txt:/tmp/foo/sub1file2:ro",
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	for _, v := range api.Content {
		if v["collection"] != nil {
			c.Check(v["ensure_unique_name"], Equals, true)
			collection := v["collection"].(arvadosclient.Dict)
			if strings.Index(collection["name"].(string), "output") == 0 {
				manifest := collection["manifest_text"].(string)

				c.Check(manifest, Equals, `./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out
./foo 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0abcdefgh11234567890@569fa8c3 9:18:bar 9:18:sub1file2
./foo/baz 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0bcdefghijk544332211@569fa8c5 9:18:sub2file2
./foo/sub1 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396cabcdefghij6419876543234@569fa8c4 0:9:file1_in_subdir1.txt 9:18:file2_in_subdir1.txt
./foo/sub1/subdir2 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0bcdefghijk544332211@569fa8c5 0:9:file1_in_subdir2.txt 9:18:file2_in_subdir2.txt
`)
			}
		}
	}
}

func (s *TestSuite) TestStdoutWithMountPointsUnderOutputDirDenormalizedManifest(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
        "/tmp": {"kind": "tmp"},
        "/tmp/foo/bar": {"kind": "collection", "portable_data_hash": "b0def87f80dd594d4675809e83bd4f15+367/subdir1/file2_in_subdir1.txt"},
        "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"}
    },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`

	extraMounts := []string{
		"b0def87f80dd594d4675809e83bd4f15+367/subdir1/file2_in_subdir1.txt",
	}

	api, _, _ := FullRunHelper(c, helperRecord, extraMounts, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	for _, v := range api.Content {
		if v["collection"] != nil {
			collection := v["collection"].(arvadosclient.Dict)
			if strings.Index(collection["name"].(string), "output") == 0 {
				manifest := collection["manifest_text"].(string)

				c.Check(manifest, Equals, `./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out
./foo 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0abcdefgh11234567890@569fa8c3 10:17:bar
`)
			}
		}
	}
}

func (s *TestSuite) TestStdinCollectionMountPoint(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
        "/tmp": {"kind": "tmp"},
        "stdin": {"kind": "collection", "portable_data_hash": "b0def87f80dd594d4675809e83bd4f15+367", "path": "/file1_in_main.txt"},
        "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"}
    },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`

	extraMounts := []string{
		"b0def87f80dd594d4675809e83bd4f15+367/file1_in_main.txt",
	}

	api, _, _ := FullRunHelper(c, helperRecord, extraMounts, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	for _, v := range api.Content {
		if v["collection"] != nil {
			collection := v["collection"].(arvadosclient.Dict)
			if strings.Index(collection["name"].(string), "output") == 0 {
				manifest := collection["manifest_text"].(string)
				c.Check(manifest, Equals, `./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out
`)
			}
		}
	}
}

func (s *TestSuite) TestStdinJsonMountPoint(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
        "/tmp": {"kind": "tmp"},
        "stdin": {"kind": "json", "content": "foo"},
        "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"}
    },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {}
	}`

	api, _, _ := FullRunHelper(c, helperRecord, nil, 0, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, t.env[0][7:]+"\n"))
		t.logWriter.Close()
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	for _, v := range api.Content {
		if v["collection"] != nil {
			collection := v["collection"].(arvadosclient.Dict)
			if strings.Index(collection["name"].(string), "output") == 0 {
				manifest := collection["manifest_text"].(string)
				c.Check(manifest, Equals, `./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out
`)
			}
		}
	}
}

func (s *TestSuite) TestStderrMount(c *C) {
	api, _, _ := FullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo hello;exit 1"],
    "container_image": "d4ab34d3d4f8a72f5c4973051ae69fab+122",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"},
               "stdout": {"kind": "file", "path": "/tmp/a/out.txt"},
               "stderr": {"kind": "file", "path": "/tmp/b/err.txt"}},
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {}
}`, nil, 1, func(t *TestDockerClient) {
		t.logWriter.Write(dockerLog(1, "hello\n"))
		t.logWriter.Write(dockerLog(2, "oops\n"))
		t.logWriter.Close()
	})

	final := api.CalledWith("container.state", "Complete")
	c.Assert(final, NotNil)
	c.Check(final["container"].(arvadosclient.Dict)["exit_code"], Equals, 1)
	c.Check(final["container"].(arvadosclient.Dict)["log"], NotNil)

	c.Check(api.CalledWith("collection.manifest_text", "./a b1946ac92492d2347c6235b4d2611184+6 0:6:out.txt\n./b 38af5c54926b620264ab1501150cf189+5 0:5:err.txt\n"), NotNil)
}
