// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/manifest"

	. "gopkg.in/check.v1"
	git_client "gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	git_http "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

// Gocheck boilerplate
func TestCrunchExec(t *testing.T) {
	TestingT(t)
}

const logLineStart = `(?m)(.*\n)*\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d+Z `

var _ = Suite(&TestSuite{})

type TestSuite struct {
	client                   *arvados.Client
	api                      *ArvTestClient
	runner                   *ContainerRunner
	executor                 *stubExecutor
	keepmount                string
	keepmountTmp             []string
	testDispatcherKeepClient KeepTestClient
	testContainerKeepClient  KeepTestClient
	debian12MemoryCurrent    int64
	debian12SwapCurrent      int64
}

func (s *TestSuite) SetUpSuite(c *C) {
	buf, err := os.ReadFile("../crunchstat/testdata/debian12/sys/fs/cgroup/user.slice/user-1000.slice/session-4.scope/memory.current")
	c.Assert(err, IsNil)
	_, err = fmt.Sscanf(string(buf), "%d", &s.debian12MemoryCurrent)
	c.Assert(err, IsNil)

	buf, err = os.ReadFile("../crunchstat/testdata/debian12/sys/fs/cgroup/user.slice/user-1000.slice/session-4.scope/memory.swap.current")
	c.Assert(err, IsNil)
	_, err = fmt.Sscanf(string(buf), "%d", &s.debian12SwapCurrent)
	c.Assert(err, IsNil)
}

func (s *TestSuite) SetUpTest(c *C) {
	s.client = arvados.NewClientFromEnv()
	s.executor = &stubExecutor{}
	var err error
	s.api = &ArvTestClient{}
	s.runner, err = NewContainerRunner(s.client, s.api, &s.testDispatcherKeepClient, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)
	s.runner.executor = s.executor
	s.runner.MkArvClient = func(token string) (IArvadosClient, IKeepClient, *arvados.Client, error) {
		return s.api, &s.testContainerKeepClient, s.client, nil
	}
	s.runner.RunArvMount = func(cmd []string, tok string) (*exec.Cmd, error) {
		s.runner.ArvMountPoint = s.keepmount
		for i, opt := range cmd {
			if opt == "--mount-tmp" {
				err := os.Mkdir(s.keepmount+"/"+cmd[i+1], 0700)
				if err != nil {
					return nil, err
				}
				s.keepmountTmp = append(s.keepmountTmp, cmd[i+1])
			}
		}
		return nil, nil
	}
	s.keepmount = c.MkDir()
	err = os.Mkdir(s.keepmount+"/by_id", 0755)
	s.keepmountTmp = nil
	c.Assert(err, IsNil)
	err = os.Mkdir(s.keepmount+"/by_id/"+arvadostest.DockerImage112PDH, 0755)
	c.Assert(err, IsNil)
	err = ioutil.WriteFile(s.keepmount+"/by_id/"+arvadostest.DockerImage112PDH+"/"+arvadostest.DockerImage112Filename, []byte("#notarealtarball"), 0644)
	err = os.Mkdir(s.keepmount+"/by_id/"+fakeInputCollectionPDH, 0755)
	c.Assert(err, IsNil)
	err = ioutil.WriteFile(s.keepmount+"/by_id/"+fakeInputCollectionPDH+"/input.json", []byte(`{"input":true}`), 0644)
	c.Assert(err, IsNil)
	s.runner.ArvMountPoint = s.keepmount
}

type ArvTestClient struct {
	Total   int64
	Calls   int
	Content []arvadosclient.Dict
	arvados.Container
	secretMounts []byte
	Logs         map[string]*bytes.Buffer
	sync.Mutex
	WasSetRunning bool
	callraw       bool
}

type KeepTestClient struct {
	Called         bool
	Content        []byte
	StorageClasses []string
}

type stubExecutor struct {
	imageLoaded bool
	loaded      string
	loadErr     error
	exitCode    int
	createErr   error
	created     containerSpec
	startErr    error
	waitSleep   time.Duration
	waitErr     error
	stopErr     error
	stopped     bool
	closed      bool
	runFunc     func() int
	exit        chan int
}

func (e *stubExecutor) LoadImage(imageId string, tarball string, container arvados.Container, keepMount string,
	containerClient *arvados.Client) error {
	e.loaded = tarball
	return e.loadErr
}
func (e *stubExecutor) Runtime() string                 { return "stub" }
func (e *stubExecutor) Version() string                 { return "stub " + cmd.Version.String() }
func (e *stubExecutor) Create(spec containerSpec) error { e.created = spec; return e.createErr }
func (e *stubExecutor) Start() error {
	e.exit = make(chan int, 1)
	go func() { e.exit <- e.runFunc() }()
	return e.startErr
}
func (e *stubExecutor) Pid() int    { return 1115883 } // matches pid in ../crunchstat/testdata/debian12/proc/
func (e *stubExecutor) Stop() error { e.stopped = true; go func() { e.exit <- -1 }(); return e.stopErr }
func (e *stubExecutor) Close()      { e.closed = true }
func (e *stubExecutor) Wait(context.Context) (int, error) {
	return <-e.exit, e.waitErr
}
func (e *stubExecutor) InjectCommand(ctx context.Context, _, _ string, _ bool, _ []string) (*exec.Cmd, error) {
	return nil, errors.New("unimplemented")
}
func (e *stubExecutor) IPAddress() (string, error) { return "", errors.New("unimplemented") }

const fakeInputCollectionPDH = "ffffffffaaaaaaaa88888888eeeeeeee+1234"

var hwManifest = ". 82ab40c24fc8df01798e57ba66795bb1+841216+Aa124ac75e5168396c73c0a18eda641a4f41791c0@569fa8c3 0:841216:9c31ee32b3d15268a0754e8edc74d4f815ee014b693bc5109058e431dd5caea7.tar\n"
var hwPDH = "a45557269dcb65a6b78f9ac061c0850b+120"
var hwImageID = "9c31ee32b3d15268a0754e8edc74d4f815ee014b693bc5109058e431dd5caea7"

var otherManifest = ". 68a84f561b1d1708c6baff5e019a9ab3+46+Ae5d0af96944a3690becb1decdf60cc1c937f556d@5693216f 0:46:md5sum.txt\n"
var otherPDH = "a3e8f74c6f101eae01fa08bfb4e49b3a+54"

var normalizedManifestWithSubdirs = `. 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0abcdefgh11234567890@569fa8c3 0:9:file1_in_main.txt 9:18:file2_in_main.txt 0:27:zzzzz-8i9sb-bcdefghijkdhvnk.log.txt
./subdir1 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396cabcdefghij6419876543234@569fa8c4 0:9:file1_in_subdir1.txt 9:18:file2_in_subdir1.txt
./subdir1/subdir2 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0bcdefghijk544332211@569fa8c5 0:9:file1_in_subdir2.txt 9:18:file2_in_subdir2.txt
`

var normalizedWithSubdirsPDH = "a0def87f80dd594d4675809e83bd4f15+367"

var denormalizedManifestWithSubdirs = ". 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0abcdefgh11234567890@569fa8c3 0:9:file1_in_main.txt 9:18:file2_in_main.txt 0:27:zzzzz-8i9sb-bcdefghijkdhvnk.log.txt 0:10:subdir1/file1_in_subdir1.txt 10:17:subdir1/file2_in_subdir1.txt\n"
var denormalizedWithSubdirsPDH = "b0def87f80dd594d4675809e83bd4f15+367"

var fakeAuthUUID = "zzzzz-gj3su-55pqoyepgi2glem"
var fakeAuthToken = "a3ltuwzqcu2u4sc0q7yhpc2w7s00fdcqecg5d6e0u3pfohmbjt"

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
		md5sum := md5.Sum([]byte(mt))
		outmap := output.(*arvados.Collection)
		outmap.PortableDataHash = fmt.Sprintf("%x+%d", md5sum, len(mt))
		outmap.UUID = fmt.Sprintf("zzzzz-4zz18-%015x", md5sum[:7])
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
	case method == "GET" && resourceType == "containers" && action == "secret_mounts":
		if client.secretMounts != nil {
			return json.Unmarshal(client.secretMounts, output)
		}
		return json.Unmarshal([]byte(`{"secret_mounts":{}}`), output)
	default:
		return fmt.Errorf("Not found")
	}
}

func (client *ArvTestClient) CallRaw(method, resourceType, uuid, action string,
	parameters arvadosclient.Dict) (reader io.ReadCloser, err error) {
	var j []byte
	if method == "GET" && resourceType == "nodes" && uuid == "" && action == "" {
		j = []byte(`{
			"kind": "arvados#nodeList",
			"items": [{
				"uuid": "zzzzz-7ekkf-2z3mc76g2q73aio",
				"hostname": "compute2",
				"properties": {"total_cpu_cores": 16}
			}]}`)
	} else if method == "GET" && resourceType == "containers" && action == "" && !client.callraw {
		if uuid == "" {
			j, err = json.Marshal(map[string]interface{}{
				"items": []interface{}{client.Container},
				"kind":  "arvados#nodeList",
			})
		} else {
			j, err = json.Marshal(client.Container)
		}
	} else {
		j = []byte(`{
			"command": ["sleep", "1"],
			"container_image": "` + arvadostest.DockerImage112PDH + `",
			"cwd": ".",
			"environment": {},
			"mounts": {"/tmp": {"kind": "tmp"}, "/json": {"kind": "json", "content": {"number": 123456789123456789}}},
			"output_path": "/tmp",
			"priority": 1,
			"runtime_constraints": {}
		}`)
	}
	return ioutil.NopCloser(bytes.NewReader(j)), err
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
	} else if resourceType == "collections" && output != nil {
		mt := parameters["collection"].(arvadosclient.Dict)["manifest_text"].(string)
		output.(*arvados.Collection).UUID = uuid
		output.(*arvados.Collection).PortableDataHash = fmt.Sprintf("%x", md5.Sum([]byte(mt)))
	}
	return nil
}

var discoveryMap = map[string]interface{}{
	"defaultTrashLifetime":               float64(1209600),
	"crunchLimitLogBytesPerJob":          float64(67108864),
	"crunchLogThrottleBytes":             float64(65536),
	"crunchLogThrottlePeriod":            float64(60),
	"crunchLogThrottleLines":             float64(1024),
	"crunchLogPartialLineThrottlePeriod": float64(5),
	"crunchLogBytesPerEvent":             float64(4096),
	"crunchLogSecondsBetweenEvents":      float64(1),
}

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

func (client *KeepTestClient) LocalLocator(locator string) (string, error) {
	return locator, nil
}

func (client *KeepTestClient) BlockWrite(_ context.Context, opts arvados.BlockWriteOptions) (arvados.BlockWriteResponse, error) {
	client.Content = opts.Data
	return arvados.BlockWriteResponse{
		Locator: fmt.Sprintf("%x+%d", md5.Sum(opts.Data), len(opts.Data)),
	}, nil
}

func (client *KeepTestClient) ReadAt(string, []byte, int) (int, error) {
	return 0, errors.New("not implemented")
}

func (client *KeepTestClient) ClearBlockCache() {
}

func (client *KeepTestClient) Close() {
	client.Content = nil
}

func (client *KeepTestClient) SetStorageClasses(sc []string) {
	client.StorageClasses = sc
}

type FileWrapper struct {
	io.ReadCloser
	len int64
}

func (fw FileWrapper) Readdir(n int) ([]os.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func (fw FileWrapper) Seek(int64, int) (int64, error) {
	return 0, errors.New("not implemented")
}

func (fw FileWrapper) Size() int64 {
	return fw.len
}

func (fw FileWrapper) Stat() (os.FileInfo, error) {
	return nil, errors.New("not implemented")
}

func (fw FileWrapper) Truncate(int64) error {
	return errors.New("not implemented")
}

func (fw FileWrapper) Write([]byte) (int, error) {
	return 0, errors.New("not implemented")
}

func (fw FileWrapper) Sync() error {
	return errors.New("not implemented")
}

func (fw FileWrapper) Snapshot() (*arvados.Subtree, error) {
	return nil, errors.New("not implemented")
}

func (fw FileWrapper) Splice(*arvados.Subtree) error {
	return errors.New("not implemented")
}

func (client *KeepTestClient) ManifestFileReader(m manifest.Manifest, filename string) (arvados.File, error) {
	if filename == hwImageID+".tar" {
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

type apiStubServer struct {
	server    *httptest.Server
	proxy     *httputil.ReverseProxy
	intercept func(http.ResponseWriter, *http.Request) bool

	container arvados.Container
	logs      map[string]string
}

func apiStub() (*arvados.Client, *apiStubServer) {
	client := arvados.NewClientFromEnv()
	apistub := &apiStubServer{}
	apistub.server = httptest.NewTLSServer(apistub)
	apistub.proxy = httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "https", Host: client.APIHost})
	if client.Insecure {
		apistub.proxy.Transport = arvados.InsecureHTTPClient.Transport
	}
	client.APIHost = apistub.server.Listener.Addr().String()
	return client, apistub
}

func (apistub *apiStubServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if apistub.intercept != nil && apistub.intercept(w, r) {
		return
	}
	if r.Method == "POST" && r.URL.Path == "/arvados/v1/logs" {
		var body struct {
			Log struct {
				EventType  string `json:"event_type"`
				Properties struct {
					Text string
				}
			}
		}
		json.NewDecoder(r.Body).Decode(&body)
		apistub.logs[body.Log.EventType] += body.Log.Properties.Text
		return
	}
	if r.Method == "GET" && r.URL.Path == "/arvados/v1/collections/"+hwPDH {
		json.NewEncoder(w).Encode(arvados.Collection{ManifestText: hwManifest})
		return
	}
	if r.Method == "GET" && r.URL.Path == "/arvados/v1/collections/"+otherPDH {
		json.NewEncoder(w).Encode(arvados.Collection{ManifestText: otherManifest})
		return
	}
	if r.Method == "GET" && r.URL.Path == "/arvados/v1/collections/"+normalizedWithSubdirsPDH {
		json.NewEncoder(w).Encode(arvados.Collection{ManifestText: normalizedManifestWithSubdirs})
		return
	}
	if r.Method == "GET" && r.URL.Path == "/arvados/v1/collections/"+denormalizedWithSubdirsPDH {
		json.NewEncoder(w).Encode(arvados.Collection{ManifestText: denormalizedManifestWithSubdirs})
		return
	}
	if r.Method == "GET" && r.URL.Path == "/arvados/v1/containers/"+apistub.container.UUID {
		json.NewEncoder(w).Encode(apistub.container)
		return
	}
	apistub.proxy.ServeHTTP(w, r)
}

func (s *TestSuite) TestLoadImage(c *C) {
	s.runner.Container.ContainerImage = arvadostest.DockerImage112PDH
	s.runner.Container.Mounts = map[string]arvados.Mount{
		"/out": {Kind: "tmp", Writable: true},
	}
	s.runner.Container.OutputPath = "/out"

	_, err := s.runner.SetupMounts()
	c.Assert(err, IsNil)

	imageID, err := s.runner.LoadImage()
	c.Check(err, IsNil)
	c.Check(s.executor.loaded, Matches, ".*"+regexp.QuoteMeta(arvadostest.DockerImage112Filename))
	c.Check(imageID, Equals, strings.TrimSuffix(arvadostest.DockerImage112Filename, ".tar"))

	s.runner.Container.ContainerImage = arvadostest.DockerImage112PDH
	s.executor.imageLoaded = false
	s.executor.loaded = ""
	s.executor.loadErr = errors.New("bork")
	imageID, err = s.runner.LoadImage()
	c.Check(err, ErrorMatches, ".*bork")
	c.Check(s.executor.loaded, Matches, ".*"+regexp.QuoteMeta(arvadostest.DockerImage112Filename))

	s.runner.Container.ContainerImage = fakeInputCollectionPDH
	s.executor.imageLoaded = false
	s.executor.loaded = ""
	s.executor.loadErr = nil
	imageID, err = s.runner.LoadImage()
	c.Check(err, ErrorMatches, "image collection does not include a \\.tar image file")
	c.Check(s.executor.loaded, Equals, "")
}

type ArvErrorTestClient struct{}

func (ArvErrorTestClient) Create(resourceType string,
	parameters arvadosclient.Dict,
	output interface{}) error {
	return nil
}

func (ArvErrorTestClient) Call(method, resourceType, uuid, action string, parameters arvadosclient.Dict, output interface{}) error {
	if method == "GET" && resourceType == "containers" && action == "auth" {
		return nil
	}
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

type KeepErrorTestClient struct {
	KeepTestClient
}

func (*KeepErrorTestClient) ManifestFileReader(manifest.Manifest, string) (arvados.File, error) {
	return nil, errors.New("KeepError")
}

func (*KeepErrorTestClient) BlockWrite(context.Context, arvados.BlockWriteOptions) (arvados.BlockWriteResponse, error) {
	return arvados.BlockWriteResponse{}, errors.New("KeepError")
}

func (*KeepErrorTestClient) LocalLocator(string) (string, error) {
	return "", errors.New("KeepError")
}

type KeepReadErrorTestClient struct {
	KeepTestClient
}

func (*KeepReadErrorTestClient) ReadAt(string, []byte, int) (int, error) {
	return 0, errors.New("KeepError")
}

type ErrorReader struct {
	FileWrapper
}

func (ErrorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("ErrorReader")
}

func (ErrorReader) Seek(int64, int) (int64, error) {
	return 0, errors.New("ErrorReader")
}

func (KeepReadErrorTestClient) ManifestFileReader(m manifest.Manifest, filename string) (arvados.File, error) {
	return ErrorReader{}, nil
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

func (tl *TestLogs) NewTestLoggingWriter(logstr string) (io.WriteCloser, error) {
	if logstr == "stdout" {
		return &tl.Stdout, nil
	}
	if logstr == "stderr" {
		return &tl.Stderr, nil
	}
	return nil, errors.New("???")
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
	s.executor.runFunc = func() int {
		fmt.Fprintf(s.executor.created.Stdout, "Hello world\n")
		return 0
	}

	var logs TestLogs
	s.runner.NewLogWriter = logs.NewTestLoggingWriter
	s.runner.Container.ContainerImage = arvadostest.DockerImage112PDH
	s.runner.Container.Command = []string{"./hw"}
	s.runner.Container.OutputStorageClasses = []string{"default"}

	imageID, err := s.runner.LoadImage()
	c.Assert(err, IsNil)

	err = s.runner.CreateContainer(imageID, nil)
	c.Assert(err, IsNil)

	err = s.runner.StartContainer()
	c.Assert(err, IsNil)

	err = s.runner.WaitFinish()
	c.Assert(err, IsNil)

	c.Check(logs.Stdout.String(), Matches, ".*Hello world\n")
	c.Check(logs.Stderr.String(), Equals, "")
}

func (s *TestSuite) TestCommitLogs(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	defer kc.Close()
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)
	cr.CrunchLog.Timestamper = (&TestTimestamper{}).Timestamp

	cr.CrunchLog.Print("Hello world!")
	cr.CrunchLog.Print("Goodbye")
	cr.finalState = "Complete"

	err = cr.CommitLogs()
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
	defer kc.Close()
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)

	err = cr.UpdateContainerRunning("")
	c.Check(err, IsNil)

	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["state"], Equals, "Running")
}

func (s *TestSuite) TestUpdateContainerComplete(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	defer kc.Close()
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)

	cr.LogsPDH = new(string)
	*cr.LogsPDH = "d3a229d2fe3690c2c3e75a71a153c6a3+60"

	cr.ExitCode = new(int)
	*cr.ExitCode = 42
	cr.finalState = "Complete"

	err = cr.UpdateContainerFinal()
	c.Check(err, IsNil)

	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["log"], Equals, *cr.LogsPDH)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["exit_code"], Equals, *cr.ExitCode)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["state"], Equals, "Complete")
}

func (s *TestSuite) TestUpdateContainerCancelled(c *C) {
	api := &ArvTestClient{}
	kc := &KeepTestClient{}
	defer kc.Close()
	cr, err := NewContainerRunner(s.client, api, kc, "zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Assert(err, IsNil)
	cr.cCancelled = true
	cr.finalState = "Cancelled"

	err = cr.UpdateContainerFinal()
	c.Check(err, IsNil)

	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["log"], IsNil)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["exit_code"], IsNil)
	c.Check(api.Content[0]["container"].(arvadosclient.Dict)["state"], Equals, "Cancelled")
}

// Used by the TestFullRun*() test below to DRY up boilerplate setup to do full
// dress rehearsal of the Run() function, starting from a JSON container record.
func (s *TestSuite) fullRunHelper(c *C, record string, extraMounts []string, fn func() int) (*ArvTestClient, *ContainerRunner, string) {
	err := json.Unmarshal([]byte(record), &s.api.Container)
	c.Assert(err, IsNil)
	initialState := s.api.Container.State

	var sm struct {
		SecretMounts map[string]arvados.Mount `json:"secret_mounts"`
	}
	err = json.Unmarshal([]byte(record), &sm)
	c.Check(err, IsNil)
	secretMounts, err := json.Marshal(sm)
	c.Assert(err, IsNil)
	c.Logf("SecretMounts decoded %v json %q", sm, secretMounts)

	s.executor.runFunc = fn

	s.runner.statInterval = 100 * time.Millisecond
	s.runner.containerWatchdogInterval = time.Second

	realTemp := c.MkDir()
	tempcount := 0
	s.runner.MkTempDir = func(_, prefix string) (string, error) {
		tempcount++
		d := fmt.Sprintf("%s/%s%d", realTemp, prefix, tempcount)
		err := os.Mkdir(d, os.ModePerm)
		if err != nil && strings.Contains(err.Error(), ": file exists") {
			// Test case must have pre-populated the tempdir
			err = nil
		}
		return d, err
	}
	client, _ := apiStub()
	s.runner.MkArvClient = func(token string) (IArvadosClient, IKeepClient, *arvados.Client, error) {
		return &ArvTestClient{secretMounts: secretMounts}, &s.testContainerKeepClient, client, nil
	}

	if extraMounts != nil && len(extraMounts) > 0 {
		err := s.runner.SetupArvMountPoint("keep")
		c.Check(err, IsNil)

		for _, m := range extraMounts {
			os.MkdirAll(s.runner.ArvMountPoint+"/by_id/"+m, os.ModePerm)
		}
	}

	err = s.runner.Run()
	if s.api.CalledWith("container.state", "Complete") != nil {
		c.Check(err, IsNil)
	}
	if s.executor.loadErr == nil && s.executor.createErr == nil && initialState != "Running" {
		c.Check(s.api.WasSetRunning, Equals, true)
		var lastupdate arvadosclient.Dict
		for _, content := range s.api.Content {
			if content["container"] != nil {
				lastupdate = content["container"].(arvadosclient.Dict)
			}
		}
		if lastupdate["log"] == nil {
			c.Errorf("no container update with non-nil log -- updates were: %v", s.api.Content)
		}
	}

	if err != nil {
		for k, v := range s.api.Logs {
			c.Log(k)
			c.Log(v.String())
		}
	}

	return s.api, s.runner, realTemp
}

func (s *TestSuite) TestFullRunHello(c *C) {
	s.runner.enableMemoryLimit = true
	s.runner.networkMode = "default"
	s.fullRunHelper(c, `{
    "command": ["echo", "hello world"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {"foo":"bar","baz":"waz"},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {"vcpus":1,"ram":1000000},
    "state": "Locked",
    "output_storage_classes": ["default"]
}`, nil, func() int {
		c.Check(s.executor.created.Command, DeepEquals, []string{"echo", "hello world"})
		c.Check(s.executor.created.Image, Equals, "sha256:d8309758b8fe2c81034ffc8a10c36460b77db7bc5e7b448c4e5b684f9d95a678")
		c.Check(s.executor.created.Env, DeepEquals, map[string]string{"foo": "bar", "baz": "waz"})
		c.Check(s.executor.created.VCPUs, Equals, 1)
		c.Check(s.executor.created.RAM, Equals, int64(1000000))
		c.Check(s.executor.created.NetworkMode, Equals, "default")
		c.Check(s.executor.created.EnableNetwork, Equals, false)
		c.Check(s.executor.created.CUDADeviceCount, Equals, 0)
		fmt.Fprintln(s.executor.created.Stdout, "hello world")
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.api.Logs["stdout"].String(), Matches, ".*hello world\n")
	c.Check(s.testDispatcherKeepClient.StorageClasses, DeepEquals, []string{"default"})
	c.Check(s.testContainerKeepClient.StorageClasses, DeepEquals, []string{"default"})
}

func (s *TestSuite) TestRunAlreadyRunning(c *C) {
	var ran bool
	s.fullRunHelper(c, `{
    "command": ["sleep", "3"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "scheduling_parameters":{"max_run_time": 1},
    "state": "Running"
}`, nil, func() int {
		ran = true
		return 2
	})
	c.Check(s.api.CalledWith("container.state", "Cancelled"), IsNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), IsNil)
	c.Check(ran, Equals, false)
}

func ec2MetadataServerStub(c *C, token *string, failureRate float64, stoptime *atomic.Value) *httptest.Server {
	failedOnce := false
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !failedOnce || rand.Float64() < failureRate {
			w.WriteHeader(http.StatusServiceUnavailable)
			failedOnce = true
			return
		}
		switch r.URL.Path {
		case "/latest/api/token":
			fmt.Fprintln(w, *token)
		case "/latest/meta-data/spot/instance-action":
			if r.Header.Get("X-aws-ec2-metadata-token") != *token {
				w.WriteHeader(http.StatusUnauthorized)
			} else if t, _ := stoptime.Load().(time.Time); t.IsZero() {
				w.WriteHeader(http.StatusNotFound)
			} else {
				fmt.Fprintf(w, `{"action":"stop","time":"%s"}`, t.Format(time.RFC3339))
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func (s *TestSuite) TestSpotInterruptionNotice(c *C) {
	s.testSpotInterruptionNotice(c, 0.1)
}

func (s *TestSuite) TestSpotInterruptionNoticeNotAvailable(c *C) {
	s.testSpotInterruptionNotice(c, 1)
}

func (s *TestSuite) testSpotInterruptionNotice(c *C, failureRate float64) {
	var stoptime atomic.Value
	token := "fake-ec2-metadata-token"
	stub := ec2MetadataServerStub(c, &token, failureRate, &stoptime)
	defer stub.Close()

	defer func(i time.Duration, u string) {
		spotInterruptionCheckInterval = i
		ec2MetadataBaseURL = u
	}(spotInterruptionCheckInterval, ec2MetadataBaseURL)
	spotInterruptionCheckInterval = time.Second / 8
	ec2MetadataBaseURL = stub.URL

	go s.runner.checkSpotInterruptionNotices()
	s.fullRunHelper(c, `{
    "command": ["sleep", "3"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`, nil, func() int {
		time.Sleep(time.Second)
		stoptime.Store(time.Now().Add(time.Minute).UTC())
		token = "different-fake-ec2-metadata-token"
		time.Sleep(time.Second)
		return 0
	})
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*Checking for spot interruptions every 125ms using instance metadata at http://.*`)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*Error checking spot interruptions: 503 Service Unavailable.*`)
	if failureRate == 1 {
		c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*Giving up on checking spot interruptions after too many consecutive failures.*`)
	} else {
		text := `Cloud provider scheduled instance stop at ` + stoptime.Load().(time.Time).Format(time.RFC3339)
		c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*`+text+`.*`)
		c.Check(s.api.CalledWith("container.runtime_status.warning", "preemption notice"), NotNil)
		c.Check(s.api.CalledWith("container.runtime_status.warningDetail", text), NotNil)
		c.Check(s.api.CalledWith("container.runtime_status.preemptionNotice", text), NotNil)
	}
}

func (s *TestSuite) TestRunTimeExceeded(c *C) {
	s.fullRunHelper(c, `{
    "command": ["sleep", "3"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "scheduling_parameters":{"max_run_time": 1},
    "state": "Locked"
}`, nil, func() int {
		time.Sleep(3 * time.Second)
		return 0
	})

	c.Check(s.api.CalledWith("container.state", "Cancelled"), NotNil)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, "(?ms).*maximum run time exceeded.*")
}

func (s *TestSuite) TestContainerWaitFails(c *C) {
	s.fullRunHelper(c, `{
    "command": ["sleep", "3"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "state": "Locked"
}`, nil, func() int {
		s.executor.waitErr = errors.New("Container is not running")
		return 0
	})

	c.Check(s.api.CalledWith("container.state", "Cancelled"), NotNil)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, "(?ms).*Container is not running.*")
}

func (s *TestSuite) TestCrunchstat(c *C) {
	s.runner.crunchstatFakeFS = os.DirFS("../crunchstat/testdata/debian12")
	s.fullRunHelper(c, `{
		"command": ["sleep", "1"],
		"container_image": "`+arvadostest.DockerImage112PDH+`",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`, nil, func() int {
		time.Sleep(time.Second)
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)

	c.Assert(s.api.Logs["crunchstat"], NotNil)
	c.Check(s.api.Logs["crunchstat"].String(), Matches, `(?ms).*mem \d+ swap \d+ pgmajfault \d+ rss.*`)

	// Check that we called (*crunchstat.Reporter)Stop().
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*Maximum crunch-run memory rss usage was \d+ bytes\n.*`)
}

func (s *TestSuite) TestNodeInfoLog(c *C) {
	os.Setenv("SLURMD_NODENAME", "compute2")
	s.fullRunHelper(c, `{
		"command": ["sleep", "1"],
		"container_image": "`+arvadostest.DockerImage112PDH+`",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`, nil, func() int {
		time.Sleep(time.Second)
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)

	c.Assert(s.api.Logs["node"], NotNil)
	json := s.api.Logs["node"].String()
	c.Check(json, Matches, `(?ms).*"uuid": *"zzzzz-7ekkf-2z3mc76g2q73aio".*`)
	c.Check(json, Matches, `(?ms).*"total_cpu_cores": *16.*`)
	c.Check(json, Not(Matches), `(?ms).*"info":.*`)

	c.Assert(s.api.Logs["node-info"], NotNil)
	json = s.api.Logs["node-info"].String()
	c.Check(json, Matches, `(?ms).*Host Information.*`)
	c.Check(json, Matches, `(?ms).*CPU Information.*`)
	c.Check(json, Matches, `(?ms).*Memory Information.*`)
	c.Check(json, Matches, `(?ms).*Disk Space.*`)
	c.Check(json, Matches, `(?ms).*Disk INodes.*`)
}

func (s *TestSuite) TestLogVersionAndRuntime(c *C) {
	s.fullRunHelper(c, `{
		"command": ["sleep", "1"],
		"container_image": "`+arvadostest.DockerImage112PDH+`",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`, nil, func() int {
		return 0
	})

	c.Assert(s.api.Logs["crunch-run"], NotNil)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*crunch-run \S+ \(go\S+\) start.*`)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*crunch-run process has uid=\d+\(.+\) gid=\d+\(.+\) groups=\d+\(.+\)(,\d+\(.+\))*\n.*`)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*Executing container: zzzzz-zzzzz-zzzzzzzzzzzzzzz.*`)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*Using container runtime: stub.*`)
}

func (s *TestSuite) testLogRSSThresholds(c *C, ram int64, expected []int, notExpected int) {
	s.runner.crunchstatFakeFS = os.DirFS("../crunchstat/testdata/debian12")
	s.fullRunHelper(c, `{
		"command": ["true"],
		"container_image": "`+arvadostest.DockerImage112PDH+`",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {"ram": `+fmt.Sprintf("%d", ram)+`},
		"state": "Locked"
	}`, nil, func() int { return 0 })
	c.Logf("=== crunchstat logs\n%s\n", s.api.Logs["crunchstat"].String())
	logs := s.api.Logs["crunch-run"].String()
	pattern := logLineStart + `Container using over %d%% of memory \(rss %d/%d bytes\)`
	var threshold int
	for _, threshold = range expected {
		c.Check(logs, Matches, fmt.Sprintf(pattern, threshold, s.debian12MemoryCurrent, ram))
	}
	if notExpected > threshold {
		c.Check(logs, Not(Matches), fmt.Sprintf(pattern, notExpected, s.debian12MemoryCurrent, ram))
	}
}

func (s *TestSuite) TestLogNoRSSThresholds(c *C) {
	s.testLogRSSThresholds(c, s.debian12MemoryCurrent*10, []int{}, 90)
}

func (s *TestSuite) TestLogSomeRSSThresholds(c *C) {
	onePercentRSS := s.debian12MemoryCurrent / 100
	s.testLogRSSThresholds(c, 102*onePercentRSS, []int{90, 95}, 99)
}

func (s *TestSuite) TestLogAllRSSThresholds(c *C) {
	s.testLogRSSThresholds(c, s.debian12MemoryCurrent, []int{90, 95, 99}, 0)
}

func (s *TestSuite) TestLogMaximaAfterRun(c *C) {
	s.runner.crunchstatFakeFS = os.DirFS("../crunchstat/testdata/debian12")
	s.runner.parentTemp = c.MkDir()
	s.fullRunHelper(c, `{
        "command": ["true"],
        "container_image": "`+arvadostest.DockerImage112PDH+`",
        "cwd": ".",
        "environment": {},
        "mounts": {"/tmp": {"kind": "tmp"} },
        "output_path": "/tmp",
        "priority": 1,
        "runtime_constraints": {"ram": `+fmt.Sprintf("%d", s.debian12MemoryCurrent*10)+`},
        "state": "Locked"
    }`, nil, func() int { return 0 })
	logs := s.api.Logs["crunch-run"].String()
	for _, expected := range []string{
		`Maximum disk usage was \d+%, \d+/\d+ bytes`,
		fmt.Sprintf(`Maximum container memory swap usage was %d bytes`, s.debian12SwapCurrent),
		`Maximum container memory pgmajfault usage was \d+ faults`,
		fmt.Sprintf(`Maximum container memory rss usage was 10%%, %d/%d bytes`, s.debian12MemoryCurrent, s.debian12MemoryCurrent*10),
		`Maximum crunch-run memory rss usage was \d+ bytes`,
	} {
		c.Check(logs, Matches, logLineStart+expected)
	}
}

func (s *TestSuite) TestCommitNodeInfoBeforeStart(c *C) {
	var collection_create, container_update arvadosclient.Dict
	s.fullRunHelper(c, `{
		"command": ["true"],
		"container_image": "`+arvadostest.DockerImage112PDH+`",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked",
		"uuid": "zzzzz-dz642-202301121543210"
	}`, nil, func() int {
		collection_create = s.api.CalledWith("ensure_unique_name", true)
		container_update = s.api.CalledWith("container.state", "Running")
		return 0
	})

	c.Assert(collection_create, NotNil)
	log_collection := collection_create["collection"].(arvadosclient.Dict)
	c.Check(log_collection["name"], Equals, "logs for zzzzz-dz642-202301121543210")
	manifest_text := log_collection["manifest_text"].(string)
	// We check that the file size is at least two digits as an easy way to
	// check the file isn't empty.
	c.Check(manifest_text, Matches, `\. .+ \d+:\d{2,}:node-info\.txt( .+)?\n`)
	c.Check(manifest_text, Matches, `\. .+ \d+:\d{2,}:node\.json( .+)?\n`)

	c.Assert(container_update, NotNil)
	// As of Arvados 2.5.0, the container update must specify its log in PDH
	// format for the API server to propagate it to container requests, which
	// is what we care about for this test.
	expect_pdh := fmt.Sprintf("%x+%d", md5.Sum([]byte(manifest_text)), len(manifest_text))
	c.Check(container_update["container"].(arvadosclient.Dict)["log"], Equals, expect_pdh)
}

func (s *TestSuite) TestContainerRecordLog(c *C) {
	s.fullRunHelper(c, `{
		"command": ["sleep", "1"],
		"container_image": "`+arvadostest.DockerImage112PDH+`",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`, nil,
		func() int {
			time.Sleep(time.Second)
			return 0
		})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)

	c.Assert(s.api.Logs["container"], NotNil)
	c.Check(s.api.Logs["container"].String(), Matches, `(?ms).*container_image.*`)
}

func (s *TestSuite) TestFullRunStderr(c *C) {
	s.fullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo hello ; echo world 1>&2 ; exit 1"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`, nil, func() int {
		fmt.Fprintln(s.executor.created.Stdout, "hello")
		fmt.Fprintln(s.executor.created.Stderr, "world")
		return 1
	})

	final := s.api.CalledWith("container.state", "Complete")
	c.Assert(final, NotNil)
	c.Check(final["container"].(arvadosclient.Dict)["exit_code"], Equals, 1)
	c.Check(final["container"].(arvadosclient.Dict)["log"], NotNil)

	c.Check(s.api.Logs["stdout"].String(), Matches, ".*hello\n")
	c.Check(s.api.Logs["stderr"].String(), Matches, ".*world\n")
}

func (s *TestSuite) TestFullRunDefaultCwd(c *C) {
	s.fullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`, nil, func() int {
		fmt.Fprintf(s.executor.created.Stdout, "workdir=%q", s.executor.created.WorkingDir)
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Log(s.api.Logs["stdout"])
	c.Check(s.api.Logs["stdout"].String(), Matches, `.*workdir=""\n`)
}

func (s *TestSuite) TestFullRunSetCwd(c *C) {
	s.fullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`, nil, func() int {
		fmt.Fprintln(s.executor.created.Stdout, s.executor.created.WorkingDir)
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.api.Logs["stdout"].String(), Matches, ".*/bin\n")
}

func (s *TestSuite) TestFullRunSetOutputStorageClasses(c *C) {
	s.fullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked",
    "output_storage_classes": ["foo", "bar"]
}`, nil, func() int {
		fmt.Fprintln(s.executor.created.Stdout, s.executor.created.WorkingDir)
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.api.Logs["stdout"].String(), Matches, ".*/bin\n")
	c.Check(s.testDispatcherKeepClient.StorageClasses, DeepEquals, []string{"foo", "bar"})
	c.Check(s.testContainerKeepClient.StorageClasses, DeepEquals, []string{"foo", "bar"})
}

func (s *TestSuite) TestEnableCUDADeviceCount(c *C) {
	s.fullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {"cuda": {"device_count": 2}},
    "state": "Locked",
    "output_storage_classes": ["foo", "bar"]
}`, nil, func() int {
		fmt.Fprintln(s.executor.created.Stdout, "ok")
		return 0
	})
	c.Check(s.executor.created.CUDADeviceCount, Equals, 2)
}

func (s *TestSuite) TestEnableCUDAHardwareCapability(c *C) {
	s.fullRunHelper(c, `{
    "command": ["pwd"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {"cuda": {"hardware_capability": "foo"}},
    "state": "Locked",
    "output_storage_classes": ["foo", "bar"]
}`, nil, func() int {
		fmt.Fprintln(s.executor.created.Stdout, "ok")
		return 0
	})
	c.Check(s.executor.created.CUDADeviceCount, Equals, 0)
}

func (s *TestSuite) TestStopOnSignal(c *C) {
	s.executor.runFunc = func() int {
		s.executor.created.Stdout.Write([]byte("foo\n"))
		s.runner.SigChan <- syscall.SIGINT
		time.Sleep(10 * time.Second)
		return 0
	}
	s.testStopContainer(c)
}

func (s *TestSuite) TestStopOnArvMountDeath(c *C) {
	s.executor.runFunc = func() int {
		s.executor.created.Stdout.Write([]byte("foo\n"))
		s.runner.ArvMountExit <- nil
		close(s.runner.ArvMountExit)
		time.Sleep(10 * time.Second)
		return 0
	}
	s.runner.ArvMountExit = make(chan error)
	s.testStopContainer(c)
}

func (s *TestSuite) testStopContainer(c *C) {
	record := `{
    "command": ["/bin/sh", "-c", "echo foo && sleep 30 && echo bar"],
    "container_image": "` + arvadostest.DockerImage112PDH + `",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`

	err := json.Unmarshal([]byte(record), &s.api.Container)
	c.Assert(err, IsNil)

	s.runner.RunArvMount = func([]string, string) (*exec.Cmd, error) { return nil, nil }
	s.runner.MkArvClient = func(token string) (IArvadosClient, IKeepClient, *arvados.Client, error) {
		return &ArvTestClient{}, &KeepTestClient{}, nil, nil
	}

	done := make(chan error)
	go func() {
		done <- s.runner.Run()
	}()
	select {
	case <-time.After(20 * time.Second):
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
		c.Fatal("timed out")
	case err = <-done:
		c.Check(err, IsNil)
	}
	for k, v := range s.api.Logs {
		c.Log(k)
		c.Log(v.String(), "\n")
	}

	c.Check(s.api.CalledWith("container.log", nil), NotNil)
	c.Check(s.api.CalledWith("container.state", "Cancelled"), NotNil)
	c.Check(s.api.Logs["stdout"].String(), Matches, "(?ms).*foo\n$")
}

func (s *TestSuite) TestFullRunSetEnv(c *C) {
	s.fullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo $FROBIZ"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": "/bin",
    "environment": {"FROBIZ": "bilbo"},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`, nil, func() int {
		fmt.Fprintf(s.executor.created.Stdout, "%v", s.executor.created.Env)
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.api.Logs["stdout"].String(), Matches, `.*map\[FROBIZ:bilbo\]\n`)
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

func stubCert(c *C, temp string) string {
	path := temp + "/ca-certificates.crt"
	err := os.WriteFile(path, []byte{}, 0666)
	c.Assert(err, IsNil)
	os.Setenv("SSL_CERT_FILE", path)
	return path
}

func (s *TestSuite) TestSetupMounts(c *C) {
	cr := s.runner
	am := &ArvMountCmdLine{}
	cr.RunArvMount = am.ArvMountTest
	cr.containerClient, _ = apiStub()
	cr.ContainerArvClient = &ArvTestClient{}
	cr.ContainerKeepClient = &KeepTestClient{}
	cr.Container.OutputStorageClasses = []string{"default"}

	realTemp := c.MkDir()
	certTemp := c.MkDir()
	stubCertPath := stubCert(c, certTemp)
	cr.parentTemp = realTemp

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
		// Should be deleted.
		_, err := os.Stat(realTemp)
		c.Assert(os.IsNotExist(err), Equals, true)

		// Now recreate it for the next test.
		c.Assert(os.Mkdir(realTemp, 0777), IsNil)
	}

	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts["/tmp"] = arvados.Mount{Kind: "tmp"}
		cr.Container.OutputPath = "/tmp"
		cr.statInterval = 5 * time.Second
		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"arv-mount", "--foreground",
			"--read-write", "--storage-classes", "default", "--crunchstat-interval=5",
			"--mount-by-pdh", "by_id", "--disable-event-listening", "--mount-by-id", "by_uuid", realTemp + "/keep1"})
		c.Check(bindmounts, DeepEquals, map[string]bindmount{"/tmp": {realTemp + "/tmp2", false}})
		os.RemoveAll(cr.ArvMountPoint)
		cr.CleanupDirs()
		checkEmpty()
	}

	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts["/out"] = arvados.Mount{Kind: "tmp"}
		cr.Container.Mounts["/tmp"] = arvados.Mount{Kind: "tmp"}
		cr.Container.OutputPath = "/out"
		cr.Container.OutputStorageClasses = []string{"foo", "bar"}

		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"arv-mount", "--foreground",
			"--read-write", "--storage-classes", "foo,bar", "--crunchstat-interval=5",
			"--mount-by-pdh", "by_id", "--disable-event-listening", "--mount-by-id", "by_uuid", realTemp + "/keep1"})
		c.Check(bindmounts, DeepEquals, map[string]bindmount{"/out": {realTemp + "/tmp2", false}, "/tmp": {realTemp + "/tmp3", false}})
		os.RemoveAll(cr.ArvMountPoint)
		cr.CleanupDirs()
		checkEmpty()
	}

	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts["/tmp"] = arvados.Mount{Kind: "tmp"}
		cr.Container.OutputPath = "/tmp"
		cr.Container.RuntimeConstraints.API = true
		cr.Container.OutputStorageClasses = []string{"default"}

		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"arv-mount", "--foreground",
			"--read-write", "--storage-classes", "default", "--crunchstat-interval=5",
			"--mount-by-pdh", "by_id", "--disable-event-listening", "--mount-by-id", "by_uuid", realTemp + "/keep1"})
		c.Check(bindmounts, DeepEquals, map[string]bindmount{"/tmp": {realTemp + "/tmp2", false}, "/etc/arvados/ca-certificates.crt": {stubCertPath, true}})
		os.RemoveAll(cr.ArvMountPoint)
		cr.CleanupDirs()
		checkEmpty()

		cr.Container.RuntimeConstraints.API = false
	}

	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = map[string]arvados.Mount{
			"/keeptmp": {Kind: "collection", Writable: true},
		}
		cr.Container.OutputPath = "/keeptmp"

		os.MkdirAll(realTemp+"/keep1/tmp0", os.ModePerm)

		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"arv-mount", "--foreground",
			"--read-write", "--storage-classes", "default", "--crunchstat-interval=5",
			"--mount-tmp", "tmp0", "--mount-by-pdh", "by_id", "--disable-event-listening", "--mount-by-id", "by_uuid", realTemp + "/keep1"})
		c.Check(bindmounts, DeepEquals, map[string]bindmount{"/keeptmp": {realTemp + "/keep1/tmp0", false}})
		os.RemoveAll(cr.ArvMountPoint)
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
		cr.Container.OutputPath = "/keepout"

		os.MkdirAll(realTemp+"/keep1/by_id/59389a8f9ee9d399be35462a0f92541c+53", os.ModePerm)
		os.MkdirAll(realTemp+"/keep1/tmp0", os.ModePerm)

		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"arv-mount", "--foreground",
			"--read-write", "--storage-classes", "default", "--crunchstat-interval=5",
			"--mount-tmp", "tmp0", "--mount-by-pdh", "by_id", "--disable-event-listening", "--mount-by-id", "by_uuid", realTemp + "/keep1"})
		c.Check(bindmounts, DeepEquals, map[string]bindmount{
			"/keepinp": {realTemp + "/keep1/by_id/59389a8f9ee9d399be35462a0f92541c+53", true},
			"/keepout": {realTemp + "/keep1/tmp0", false},
		})
		os.RemoveAll(cr.ArvMountPoint)
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
		cr.Container.OutputPath = "/keepout"

		os.MkdirAll(realTemp+"/keep1/by_id/59389a8f9ee9d399be35462a0f92541c+53", os.ModePerm)
		os.MkdirAll(realTemp+"/keep1/tmp0", os.ModePerm)

		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"arv-mount", "--foreground",
			"--read-write", "--storage-classes", "default", "--crunchstat-interval=5", "--ram-cache",
			"--file-cache", "512", "--mount-tmp", "tmp0", "--mount-by-pdh", "by_id", "--disable-event-listening", "--mount-by-id", "by_uuid", realTemp + "/keep1"})
		c.Check(bindmounts, DeepEquals, map[string]bindmount{
			"/keepinp": {realTemp + "/keep1/by_id/59389a8f9ee9d399be35462a0f92541c+53", true},
			"/keepout": {realTemp + "/keep1/tmp0", false},
		})
		os.RemoveAll(cr.ArvMountPoint)
		cr.CleanupDirs()
		checkEmpty()
	}

	for _, test := range []struct {
		in  interface{}
		out string
	}{
		{in: "foo", out: `"foo"`},
		{in: nil, out: `null`},
		{in: map[string]int64{"foo": 123456789123456789}, out: `{"foo":123456789123456789}`},
	} {
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = map[string]arvados.Mount{
			"/mnt/test.json": {Kind: "json", Content: test.in},
		}
		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(bindmounts, DeepEquals, map[string]bindmount{
			"/mnt/test.json": {realTemp + "/json2/mountdata.json", true},
		})
		content, err := ioutil.ReadFile(realTemp + "/json2/mountdata.json")
		c.Check(err, IsNil)
		c.Check(content, DeepEquals, []byte(test.out))
		os.RemoveAll(cr.ArvMountPoint)
		cr.CleanupDirs()
		checkEmpty()
	}

	for _, test := range []struct {
		in  interface{}
		out string
	}{
		{in: "foo", out: `foo`},
		{in: nil, out: "error"},
		{in: map[string]int64{"foo": 123456789123456789}, out: "error"},
	} {
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = map[string]arvados.Mount{
			"/mnt/test.txt": {Kind: "text", Content: test.in},
		}
		bindmounts, err := cr.SetupMounts()
		if test.out == "error" {
			c.Check(err.Error(), Equals, "content for mount \"/mnt/test.txt\" must be a string")
		} else {
			c.Check(err, IsNil)
			c.Check(bindmounts, DeepEquals, map[string]bindmount{
				"/mnt/test.txt": {realTemp + "/text2/mountdata.text", true},
			})
			content, err := ioutil.ReadFile(realTemp + "/text2/mountdata.text")
			c.Check(err, IsNil)
			c.Check(content, DeepEquals, []byte(test.out))
		}
		os.RemoveAll(cr.ArvMountPoint)
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
		cr.Container.OutputPath = "/tmp"

		os.MkdirAll(realTemp+"/keep1/tmp0", os.ModePerm)

		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)
		c.Check(am.Cmd, DeepEquals, []string{"arv-mount", "--foreground",
			"--read-write", "--storage-classes", "default", "--crunchstat-interval=5", "--ram-cache",
			"--file-cache", "512", "--mount-tmp", "tmp0", "--mount-by-pdh", "by_id", "--disable-event-listening", "--mount-by-id", "by_uuid", realTemp + "/keep1"})
		c.Check(bindmounts, DeepEquals, map[string]bindmount{
			"/tmp":     {realTemp + "/tmp2", false},
			"/tmp/foo": {realTemp + "/keep1/tmp0", true},
		})
		os.RemoveAll(cr.ArvMountPoint)
		cr.CleanupDirs()
		checkEmpty()
	}

	// Writable mount points copied to output_dir mount point
	{
		i = 0
		cr.ArvMountPoint = ""
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts = map[string]arvados.Mount{
			"/tmp": {Kind: "tmp"},
			"/tmp/foo": {Kind: "collection",
				PortableDataHash: "59389a8f9ee9d399be35462a0f92541c+53",
				Writable:         true},
			"/tmp/bar": {Kind: "collection",
				PortableDataHash: "59389a8f9ee9d399be35462a0f92541d+53",
				Path:             "baz",
				Writable:         true},
		}
		cr.Container.OutputPath = "/tmp"

		os.MkdirAll(realTemp+"/keep1/by_id/59389a8f9ee9d399be35462a0f92541c+53", os.ModePerm)
		os.MkdirAll(realTemp+"/keep1/by_id/59389a8f9ee9d399be35462a0f92541d+53/baz", os.ModePerm)

		rf, _ := os.Create(realTemp + "/keep1/by_id/59389a8f9ee9d399be35462a0f92541d+53/baz/quux")
		rf.Write([]byte("bar"))
		rf.Close()

		_, err := cr.SetupMounts()
		c.Check(err, IsNil)
		_, err = os.Stat(cr.HostOutputDir + "/foo")
		c.Check(err, IsNil)
		_, err = os.Stat(cr.HostOutputDir + "/bar/quux")
		c.Check(err, IsNil)
		os.RemoveAll(cr.ArvMountPoint)
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
			"/tmp/foo": {Kind: "tmp"},
		}
		cr.Container.OutputPath = "/tmp"

		_, err := cr.SetupMounts()
		c.Check(err, NotNil)
		c.Check(err, ErrorMatches, `only mount points of kind 'collection', 'text' or 'json' are supported underneath the output_path.*`)
		os.RemoveAll(cr.ArvMountPoint)
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

		_, err := cr.SetupMounts()
		c.Check(err, NotNil)
		c.Check(err, ErrorMatches, `unsupported mount kind 'tmp' for stdin.*`)
		os.RemoveAll(cr.ArvMountPoint)
		cr.CleanupDirs()
		checkEmpty()
	}

	// git_tree mounts
	{
		i = 0
		cr.ArvMountPoint = ""
		git_client.InstallProtocol("https", git_http.NewClient(arvados.InsecureHTTPClient))
		cr.token = arvadostest.ActiveToken
		cr.Container.Mounts = make(map[string]arvados.Mount)
		cr.Container.Mounts = map[string]arvados.Mount{
			"/tip": {
				Kind:   "git_tree",
				UUID:   arvadostest.Repository2UUID,
				Commit: "fd3531f42995344f36c30b79f55f27b502f3d344",
				Path:   "/",
			},
			"/non-tip": {
				Kind:   "git_tree",
				UUID:   arvadostest.Repository2UUID,
				Commit: "5ebfab0522851df01fec11ec55a6d0f4877b542e",
				Path:   "/",
			},
		}
		cr.Container.OutputPath = "/tmp"

		bindmounts, err := cr.SetupMounts()
		c.Check(err, IsNil)

		for path, mount := range bindmounts {
			c.Check(mount.ReadOnly, Equals, !cr.Container.Mounts[path].Writable, Commentf("%s %#v", path, mount))
		}

		data, err := ioutil.ReadFile(bindmounts["/tip"].HostPath + "/dir1/dir2/file with mode 0644")
		c.Check(err, IsNil)
		c.Check(string(data), Equals, "\000\001\002\003")
		_, err = ioutil.ReadFile(bindmounts["/tip"].HostPath + "/file only on testbranch")
		c.Check(err, FitsTypeOf, &os.PathError{})
		c.Check(os.IsNotExist(err), Equals, true)

		data, err = ioutil.ReadFile(bindmounts["/non-tip"].HostPath + "/dir1/dir2/file with mode 0644")
		c.Check(err, IsNil)
		c.Check(string(data), Equals, "\000\001\002\003")
		data, err = ioutil.ReadFile(bindmounts["/non-tip"].HostPath + "/file only on testbranch")
		c.Check(err, IsNil)
		c.Check(string(data), Equals, "testfile\n")

		cr.CleanupDirs()
		checkEmpty()
	}
}

func (s *TestSuite) TestStdout(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {"/tmp": {"kind": "tmp"}, "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`

	s.fullRunHelper(c, helperRecord, nil, func() int {
		fmt.Fprintln(s.executor.created.Stdout, s.executor.created.Env["FROBIZ"])
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.runner.ContainerArvClient.(*ArvTestClient).CalledWith("collection.manifest_text", "./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out\n"), NotNil)
}

// Used by the TestStdoutWithWrongPath*()
func (s *TestSuite) stdoutErrorRunHelper(c *C, record string, fn func() int) (*ArvTestClient, *ContainerRunner, error) {
	err := json.Unmarshal([]byte(record), &s.api.Container)
	c.Assert(err, IsNil)
	s.executor.runFunc = fn
	s.runner.RunArvMount = (&ArvMountCmdLine{}).ArvMountTest
	s.runner.MkArvClient = func(token string) (IArvadosClient, IKeepClient, *arvados.Client, error) {
		return s.api, &KeepTestClient{}, nil, nil
	}
	return s.api, s.runner, s.runner.Run()
}

func (s *TestSuite) TestStdoutWithWrongPath(c *C) {
	_, _, err := s.stdoutErrorRunHelper(c, `{
    "mounts": {"/tmp": {"kind": "tmp"}, "stdout": {"kind": "file", "path":"/tmpa.out"} },
    "output_path": "/tmp",
    "state": "Locked"
}`, func() int { return 0 })
	c.Check(err, ErrorMatches, ".*Stdout path does not start with OutputPath.*")
}

func (s *TestSuite) TestStdoutWithWrongKindTmp(c *C) {
	_, _, err := s.stdoutErrorRunHelper(c, `{
    "mounts": {"/tmp": {"kind": "tmp"}, "stdout": {"kind": "tmp", "path":"/tmp/a.out"} },
    "output_path": "/tmp",
    "state": "Locked"
}`, func() int { return 0 })
	c.Check(err, ErrorMatches, ".*unsupported mount kind 'tmp' for stdout.*")
}

func (s *TestSuite) TestStdoutWithWrongKindCollection(c *C) {
	_, _, err := s.stdoutErrorRunHelper(c, `{
    "mounts": {"/tmp": {"kind": "tmp"}, "stdout": {"kind": "collection", "path":"/tmp/a.out"} },
    "output_path": "/tmp",
    "state": "Locked"
}`, func() int { return 0 })
	c.Check(err, ErrorMatches, ".*unsupported mount kind 'collection' for stdout.*")
}

func (s *TestSuite) TestFullRunWithAPI(c *C) {
	s.fullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "true $ARVADOS_API_HOST"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {"API": true},
    "state": "Locked"
}`, nil, func() int {
		c.Check(s.executor.created.Env["ARVADOS_API_HOST"], Equals, os.Getenv("ARVADOS_API_HOST"))
		return 3
	})
	c.Check(s.api.CalledWith("container.exit_code", 3), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*status code 3\n.*`)
}

func (s *TestSuite) TestFullRunSetOutput(c *C) {
	defer os.Setenv("ARVADOS_API_HOST", os.Getenv("ARVADOS_API_HOST"))
	os.Setenv("ARVADOS_API_HOST", "test.arvados.org")
	s.fullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo $ARVADOS_API_HOST"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {"API": true},
    "state": "Locked"
}`, nil, func() int {
		s.api.Container.Output = arvadostest.DockerImage112PDH
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.api.CalledWith("container.output", arvadostest.DockerImage112PDH), NotNil)
}

func (s *TestSuite) TestArvMountRuntimeStatusWarning(c *C) {
	s.runner.RunArvMount = func([]string, string) (*exec.Cmd, error) {
		os.Mkdir(s.runner.ArvMountPoint+"/by_id", 0666)
		ioutil.WriteFile(s.runner.ArvMountPoint+"/by_id/README", nil, 0666)
		return s.runner.ArvMountCmd([]string{"bash", "-c", "echo >&2 Test: Keep write error: I am a teapot; sleep 3"}, "")
	}
	s.executor.runFunc = func() int {
		time.Sleep(time.Second)
		return 137
	}
	record := `{
    "command": ["sleep", "1"],
    "container_image": "` + arvadostest.DockerImage112PDH + `",
    "cwd": "/bin",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {"API": true},
    "state": "Locked"
}`
	err := json.Unmarshal([]byte(record), &s.api.Container)
	c.Assert(err, IsNil)
	err = s.runner.Run()
	c.Assert(err, IsNil)
	c.Check(s.api.CalledWith("container.exit_code", 137), NotNil)
	c.Check(s.api.CalledWith("container.runtime_status.warning", "arv-mount: Keep write error"), NotNil)
	c.Check(s.api.CalledWith("container.runtime_status.warningDetail", "Test: Keep write error: I am a teapot"), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.api.Logs["crunch-run"].String(), Matches, `(?ms).*Container exited with status code 137 \(signal 9, SIGKILL\).*`)
}

func (s *TestSuite) TestStdoutWithExcludeFromOutputMountPointUnderOutputDir(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
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
		"runtime_constraints": {},
		"state": "Locked"
	}`

	extraMounts := []string{"a3e8f74c6f101eae01fa08bfb4e49b3a+54"}

	s.fullRunHelper(c, helperRecord, extraMounts, func() int {
		fmt.Fprintln(s.executor.created.Stdout, s.executor.created.Env["FROBIZ"])
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.runner.ContainerArvClient.(*ArvTestClient).CalledWith("collection.manifest_text", "./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out\n"), NotNil)
}

func (s *TestSuite) TestStdoutWithMultipleMountPointsUnderOutputDir(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
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
		"runtime_constraints": {},
		"state": "Locked",
		"uuid": "zzzzz-dz642-202301130848001"
	}`

	extraMounts := []string{
		"a0def87f80dd594d4675809e83bd4f15+367/file2_in_main.txt",
		"a0def87f80dd594d4675809e83bd4f15+367/subdir1/file2_in_subdir1.txt",
		"a0def87f80dd594d4675809e83bd4f15+367/subdir1/subdir2/file2_in_subdir2.txt",
	}

	api, _, realtemp := s.fullRunHelper(c, helperRecord, extraMounts, func() int {
		fmt.Fprintln(s.executor.created.Stdout, s.executor.created.Env["FROBIZ"])
		return 0
	})

	c.Check(s.executor.created.BindMounts, DeepEquals, map[string]bindmount{
		"/tmp":                   {realtemp + "/tmp1", false},
		"/tmp/foo/bar":           {s.keepmount + "/by_id/a0def87f80dd594d4675809e83bd4f15+367/file2_in_main.txt", true},
		"/tmp/foo/baz/sub2file2": {s.keepmount + "/by_id/a0def87f80dd594d4675809e83bd4f15+367/subdir1/subdir2/file2_in_subdir2.txt", true},
		"/tmp/foo/sub1":          {s.keepmount + "/by_id/a0def87f80dd594d4675809e83bd4f15+367/subdir1", true},
		"/tmp/foo/sub1file2":     {s.keepmount + "/by_id/a0def87f80dd594d4675809e83bd4f15+367/subdir1/file2_in_subdir1.txt", true},
	})

	c.Check(api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(api.CalledWith("container.state", "Complete"), NotNil)
	output_count := uint(0)
	for _, v := range s.runner.ContainerArvClient.(*ArvTestClient).Content {
		if v["collection"] == nil {
			continue
		}
		collection := v["collection"].(arvadosclient.Dict)
		if collection["name"].(string) != "output for zzzzz-dz642-202301130848001" {
			continue
		}
		c.Check(v["ensure_unique_name"], Equals, true)
		c.Check(collection["manifest_text"].(string), Equals, `./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out
./foo 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0abcdefgh11234567890@569fa8c3 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396cabcdefghij6419876543234@569fa8c4 9:18:bar 36:18:sub1file2
./foo/baz 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0bcdefghijk544332211@569fa8c5 9:18:sub2file2
./foo/sub1 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396cabcdefghij6419876543234@569fa8c4 0:9:file1_in_subdir1.txt 9:18:file2_in_subdir1.txt
./foo/sub1/subdir2 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0bcdefghijk544332211@569fa8c5 0:9:file1_in_subdir2.txt 9:18:file2_in_subdir2.txt
`)
		output_count++
	}
	c.Check(output_count, Not(Equals), uint(0))
}

func (s *TestSuite) TestStdoutWithMountPointsUnderOutputDirDenormalizedManifest(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
        "/tmp": {"kind": "tmp"},
        "/tmp/foo/bar": {"kind": "collection", "portable_data_hash": "b0def87f80dd594d4675809e83bd4f15+367", "path": "/subdir1/file2_in_subdir1.txt"},
        "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"}
    },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked",
		"uuid": "zzzzz-dz642-202301130848002"
	}`

	extraMounts := []string{
		"b0def87f80dd594d4675809e83bd4f15+367/subdir1/file2_in_subdir1.txt",
	}

	s.fullRunHelper(c, helperRecord, extraMounts, func() int {
		fmt.Fprintln(s.executor.created.Stdout, s.executor.created.Env["FROBIZ"])
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	output_count := uint(0)
	for _, v := range s.runner.ContainerArvClient.(*ArvTestClient).Content {
		if v["collection"] == nil {
			continue
		}
		collection := v["collection"].(arvadosclient.Dict)
		if collection["name"].(string) != "output for zzzzz-dz642-202301130848002" {
			continue
		}
		c.Check(collection["manifest_text"].(string), Equals, `./a/b 307372fa8fd5c146b22ae7a45b49bc31+6 0:6:c.out
./foo 3e426d509afffb85e06c4c96a7c15e91+27+Aa124ac75e5168396c73c0abcdefgh11234567890@569fa8c3 10:17:bar
`)
		output_count++
	}
	c.Check(output_count, Not(Equals), uint(0))
}

func (s *TestSuite) TestOutputError(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
			"/tmp": {"kind": "tmp"}
		},
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`
	s.fullRunHelper(c, helperRecord, nil, func() int {
		os.Symlink("/etc/hosts", s.runner.HostOutputDir+"/baz")
		return 0
	})

	c.Check(s.api.CalledWith("container.state", "Cancelled"), NotNil)
}

func (s *TestSuite) TestStdinCollectionMountPoint(c *C) {
	helperRecord := `{
		"command": ["/bin/sh", "-c", "echo $FROBIZ"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
        "/tmp": {"kind": "tmp"},
        "stdin": {"kind": "collection", "portable_data_hash": "b0def87f80dd594d4675809e83bd4f15+367", "path": "/file1_in_main.txt"},
        "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"}
    },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`

	extraMounts := []string{
		"b0def87f80dd594d4675809e83bd4f15+367/file1_in_main.txt",
	}

	api, _, _ := s.fullRunHelper(c, helperRecord, extraMounts, func() int {
		fmt.Fprintln(s.executor.created.Stdout, s.executor.created.Env["FROBIZ"])
		return 0
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
		"container_image": "` + arvadostest.DockerImage112PDH + `",
		"cwd": "/bin",
		"environment": {"FROBIZ": "bilbo"},
		"mounts": {
        "/tmp": {"kind": "tmp"},
        "stdin": {"kind": "json", "content": "foo"},
        "stdout": {"kind": "file", "path": "/tmp/a/b/c.out"}
    },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`

	api, _, _ := s.fullRunHelper(c, helperRecord, nil, func() int {
		fmt.Fprintln(s.executor.created.Stdout, s.executor.created.Env["FROBIZ"])
		return 0
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
	api, cr, _ := s.fullRunHelper(c, `{
    "command": ["/bin/sh", "-c", "echo hello;exit 1"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"},
               "stdout": {"kind": "file", "path": "/tmp/a/out.txt"},
               "stderr": {"kind": "file", "path": "/tmp/b/err.txt"}},
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`, nil, func() int {
		fmt.Fprintln(s.executor.created.Stdout, "hello")
		fmt.Fprintln(s.executor.created.Stderr, "oops")
		return 1
	})

	final := api.CalledWith("container.state", "Complete")
	c.Assert(final, NotNil)
	c.Check(final["container"].(arvadosclient.Dict)["exit_code"], Equals, 1)
	c.Check(final["container"].(arvadosclient.Dict)["log"], NotNil)

	c.Check(cr.ContainerArvClient.(*ArvTestClient).CalledWith("collection.manifest_text", "./a b1946ac92492d2347c6235b4d2611184+6 0:6:out.txt\n./b 38af5c54926b620264ab1501150cf189+5 0:5:err.txt\n"), NotNil)
}

func (s *TestSuite) TestNumberRoundTrip(c *C) {
	s.api.callraw = true
	err := s.runner.fetchContainerRecord()
	c.Assert(err, IsNil)
	jsondata, err := json.Marshal(s.runner.Container.Mounts["/json"].Content)
	c.Logf("%#v", s.runner.Container)
	c.Check(err, IsNil)
	c.Check(string(jsondata), Equals, `{"number":123456789123456789}`)
}

func (s *TestSuite) TestFullBrokenDocker(c *C) {
	nextState := ""
	for _, setup := range []func(){
		func() {
			c.Log("// waitErr = ocl runtime error")
			s.executor.waitErr = errors.New(`Error response from daemon: oci runtime error: container_linux.go:247: starting container process caused "process_linux.go:359: container init caused \"rootfs_linux.go:54: mounting \\\"/tmp/keep453790790/by_id/99999999999999999999999999999999+99999/myGenome\\\" to rootfs \\\"/tmp/docker/overlay2/9999999999999999999999999999999999999999999999999999999999999999/merged\\\" at \\\"/tmp/docker/overlay2/9999999999999999999999999999999999999999999999999999999999999999/merged/keep/99999999999999999999999999999999+99999/myGenome\\\" caused \\\"no such file or directory\\\"\""`)
			nextState = "Cancelled"
		},
		func() {
			c.Log("// loadErr = cannot connect")
			s.executor.loadErr = errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?")
			s.runner.brokenNodeHook = c.MkDir() + "/broken-node-hook"
			err := ioutil.WriteFile(s.runner.brokenNodeHook, []byte("#!/bin/sh\nexec echo killme\n"), 0700)
			c.Assert(err, IsNil)
			nextState = "Queued"
		},
	} {
		s.SetUpTest(c)
		setup()
		s.fullRunHelper(c, `{
    "command": ["echo", "hello world"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`, nil, func() int { return 0 })
		c.Check(s.api.CalledWith("container.state", nextState), NotNil)
		c.Check(s.api.Logs["crunch-run"].String(), Matches, "(?ms).*unable to run containers.*")
		if s.runner.brokenNodeHook != "" {
			c.Check(s.api.Logs["crunch-run"].String(), Matches, "(?ms).*Running broken node hook.*")
			c.Check(s.api.Logs["crunch-run"].String(), Matches, "(?ms).*killme.*")
			c.Check(s.api.Logs["crunch-run"].String(), Not(Matches), "(?ms).*Writing /var/lock/crunch-run-broken to mark node as broken.*")
		} else {
			c.Check(s.api.Logs["crunch-run"].String(), Matches, "(?ms).*Writing /var/lock/crunch-run-broken to mark node as broken.*")
		}
	}
}

func (s *TestSuite) TestBadCommand(c *C) {
	for _, startError := range []string{
		`panic: standard_init_linux.go:175: exec user process caused "no such file or directory"`,
		`Error response from daemon: Cannot start container 41f26cbc43bcc1280f4323efb1830a394ba8660c9d1c2b564ba42bf7f7694845: [8] System error: no such file or directory`,
		`Error response from daemon: Cannot start container 58099cd76c834f3dc2a4fb76c8028f049ae6d4fdf0ec373e1f2cfea030670c2d: [8] System error: exec: "foobar": executable file not found in $PATH`,
	} {
		s.SetUpTest(c)
		s.executor.startErr = errors.New(startError)
		s.fullRunHelper(c, `{
    "command": ["echo", "hello world"],
    "container_image": "`+arvadostest.DockerImage112PDH+`",
    "cwd": ".",
    "environment": {},
    "mounts": {"/tmp": {"kind": "tmp"} },
    "output_path": "/tmp",
    "priority": 1,
    "runtime_constraints": {},
    "state": "Locked"
}`, nil, func() int { return 0 })
		c.Check(s.api.CalledWith("container.state", "Cancelled"), NotNil)
		c.Check(s.api.Logs["crunch-run"].String(), Matches, "(?ms).*Possible causes:.*is missing.*")
	}
}

func (s *TestSuite) TestSecretTextMountPoint(c *C) {
	helperRecord := `{
		"command": ["true"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
		"cwd": "/bin",
		"mounts": {
                    "/tmp": {"kind": "tmp"},
                    "/tmp/secret.conf": {"kind": "text", "content": "mypassword"}
                },
                "secret_mounts": {
                },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`

	s.fullRunHelper(c, helperRecord, nil, func() int {
		content, err := ioutil.ReadFile(s.runner.HostOutputDir + "/secret.conf")
		c.Check(err, IsNil)
		c.Check(string(content), Equals, "mypassword")
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.runner.ContainerArvClient.(*ArvTestClient).CalledWith("collection.manifest_text", ". 34819d7beeabb9260a5c854bc85b3e44+10 0:10:secret.conf\n"), NotNil)
	c.Check(s.runner.ContainerArvClient.(*ArvTestClient).CalledWith("collection.manifest_text", ""), IsNil)

	// under secret mounts, not captured in output
	helperRecord = `{
		"command": ["true"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
		"cwd": "/bin",
		"mounts": {
                    "/tmp": {"kind": "tmp"}
                },
                "secret_mounts": {
                    "/tmp/secret.conf": {"kind": "text", "content": "mypassword"}
                },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`

	s.SetUpTest(c)
	s.fullRunHelper(c, helperRecord, nil, func() int {
		content, err := ioutil.ReadFile(s.runner.HostOutputDir + "/secret.conf")
		c.Check(err, IsNil)
		c.Check(string(content), Equals, "mypassword")
		return 0
	})

	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.runner.ContainerArvClient.(*ArvTestClient).CalledWith("collection.manifest_text", ". 34819d7beeabb9260a5c854bc85b3e44+10 0:10:secret.conf\n"), IsNil)
	c.Check(s.runner.ContainerArvClient.(*ArvTestClient).CalledWith("collection.manifest_text", ""), NotNil)

	// under secret mounts, output dir is a collection, not captured in output
	helperRecord = `{
		"command": ["true"],
		"container_image": "` + arvadostest.DockerImage112PDH + `",
		"cwd": "/bin",
		"mounts": {
                    "/tmp": {"kind": "collection", "writable": true}
                },
                "secret_mounts": {
                    "/tmp/secret.conf": {"kind": "text", "content": "mypassword"}
                },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked"
	}`

	s.SetUpTest(c)
	_, _, realtemp := s.fullRunHelper(c, helperRecord, nil, func() int {
		// secret.conf should be provisioned as a separate
		// bind mount, i.e., it should not appear in the
		// (fake) fuse filesystem as viewed from the host.
		content, err := ioutil.ReadFile(s.runner.HostOutputDir + "/secret.conf")
		if !c.Check(errors.Is(err, os.ErrNotExist), Equals, true) {
			c.Logf("secret.conf: content %q, err %#v", content, err)
		}
		err = ioutil.WriteFile(s.runner.HostOutputDir+"/.arvados#collection", []byte(`{"manifest_text":". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt\n"}`), 0700)
		c.Check(err, IsNil)
		return 0
	})

	content, err := ioutil.ReadFile(realtemp + "/text1/mountdata.text")
	c.Check(err, IsNil)
	c.Check(string(content), Equals, "mypassword")
	c.Check(s.executor.created.BindMounts["/tmp/secret.conf"], DeepEquals, bindmount{realtemp + "/text1/mountdata.text", true})
	c.Check(s.api.CalledWith("container.exit_code", 0), NotNil)
	c.Check(s.api.CalledWith("container.state", "Complete"), NotNil)
	c.Check(s.runner.ContainerArvClient.(*ArvTestClient).CalledWith("collection.manifest_text", ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt\n"), NotNil)
}

func (s *TestSuite) TestCalculateCost(c *C) {
	defer func(s string) { lockdir = s }(lockdir)
	lockdir = c.MkDir()
	now := time.Now()
	cr := s.runner
	cr.costStartTime = now.Add(-time.Hour)
	var logbuf bytes.Buffer
	cr.CrunchLog.Immediate = log.New(&logbuf, "", 0)

	// if there's no InstanceType env var, cost is calculated as 0
	os.Unsetenv("InstanceType")
	cost := cr.calculateCost(now)
	c.Check(cost, Equals, 0.0)

	// with InstanceType env var and loadPrices() hasn't run (or
	// hasn't found any data), cost is calculated based on
	// InstanceType env var
	os.Setenv("InstanceType", `{"Price":1.2}`)
	defer os.Unsetenv("InstanceType")
	cost = cr.calculateCost(now)
	c.Check(cost, Equals, 1.2)

	// first update tells us the spot price was $1/h until 30
	// minutes ago when it increased to $2/h
	j, err := json.Marshal([]cloud.InstancePrice{
		{StartTime: now.Add(-4 * time.Hour), Price: 1.0},
		{StartTime: now.Add(-time.Hour / 2), Price: 2.0},
	})
	c.Assert(err, IsNil)
	os.WriteFile(lockdir+"/"+pricesfile, j, 0777)
	cr.loadPrices()
	cost = cr.calculateCost(now)
	c.Check(cost, Equals, 1.5)

	// next update (via --list + SIGUSR2) tells us the spot price
	// increased to $3/h 15 minutes ago
	j, err = json.Marshal([]cloud.InstancePrice{
		{StartTime: now.Add(-time.Hour / 3), Price: 2.0}, // dup of -time.Hour/2 price
		{StartTime: now.Add(-time.Hour / 4), Price: 3.0},
	})
	c.Assert(err, IsNil)
	os.WriteFile(lockdir+"/"+pricesfile, j, 0777)
	cr.loadPrices()
	cost = cr.calculateCost(now)
	c.Check(cost, Equals, 1.0/2+2.0/4+3.0/4)

	cost = cr.calculateCost(now.Add(-time.Hour / 2))
	c.Check(cost, Equals, 0.5)

	c.Logf("%s", logbuf.String())
	c.Check(logbuf.String(), Matches, `(?ms).*Instance price changed to 1\.00 at 20.* changed to 2\.00 .* changed to 3\.00 .*`)
	c.Check(logbuf.String(), Not(Matches), `(?ms).*changed to 2\.00 .* changed to 2\.00 .*`)
}

func (s *TestSuite) TestSIGUSR2CostUpdate(c *C) {
	pid := os.Getpid()
	now := time.Now()
	pricesJSON, err := json.Marshal([]cloud.InstancePrice{
		{StartTime: now.Add(-4 * time.Hour), Price: 2.4},
		{StartTime: now.Add(-2 * time.Hour), Price: 2.6},
	})
	c.Assert(err, IsNil)

	os.Setenv("InstanceType", `{"Price":2.2}`)
	defer os.Unsetenv("InstanceType")
	defer func(s string) { lockdir = s }(lockdir)
	lockdir = c.MkDir()

	// We can't use s.api.CalledWith because timing differences will yield
	// different cost values across runs. getCostUpdate iterates over API
	// calls until it finds one that sets the cost, then writes that value
	// to the next index of costUpdates.
	deadline := now.Add(time.Second)
	costUpdates := make([]float64, 2)
	costIndex := 0
	apiIndex := 0
	getCostUpdate := func() {
		for ; time.Now().Before(deadline); time.Sleep(time.Second / 10) {
			for apiIndex < len(s.api.Content) {
				update := s.api.Content[apiIndex]
				apiIndex++
				var ok bool
				var cost float64
				if update, ok = update["container"].(arvadosclient.Dict); !ok {
					continue
				}
				if cost, ok = update["cost"].(float64); !ok {
					continue
				}
				c.Logf("API call #%d updates cost to %v", apiIndex-1, cost)
				costUpdates[costIndex] = cost
				costIndex++
				return
			}
		}
	}

	s.fullRunHelper(c, `{
		"command": ["true"],
		"container_image": "`+arvadostest.DockerImage112PDH+`",
		"cwd": ".",
		"environment": {},
		"mounts": {"/tmp": {"kind": "tmp"} },
		"output_path": "/tmp",
		"priority": 1,
		"runtime_constraints": {},
		"state": "Locked",
		"uuid": "zzzzz-dz642-20230320101530a"
	}`, nil, func() int {
		s.runner.costStartTime = now.Add(-3 * time.Hour)
		err := syscall.Kill(pid, syscall.SIGUSR2)
		c.Check(err, IsNil, Commentf("error sending first SIGUSR2 to runner"))
		getCostUpdate()

		err = os.WriteFile(path.Join(lockdir, pricesfile), pricesJSON, 0o700)
		c.Check(err, IsNil, Commentf("error writing JSON prices file"))
		err = syscall.Kill(pid, syscall.SIGUSR2)
		c.Check(err, IsNil, Commentf("error sending second SIGUSR2 to runner"))
		getCostUpdate()

		return 0
	})
	// Comparing with format strings makes it easy to ignore minor variations
	// in cost across runs while keeping diagnostics pretty.
	c.Check(fmt.Sprintf("%.3f", costUpdates[0]), Equals, "6.600")
	c.Check(fmt.Sprintf("%.3f", costUpdates[1]), Equals, "7.600")
}

type FakeProcess struct {
	cmdLine []string
}

func (fp FakeProcess) CmdlineSlice() ([]string, error) {
	return fp.cmdLine, nil
}
