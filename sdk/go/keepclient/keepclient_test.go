package keepclient

import (
	"crypto/md5"
	"flag"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/streamer"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&ServerRequiredSuite{})
var _ = Suite(&StandaloneSuite{})

var no_server = flag.Bool("no-server", false, "Skip 'ServerRequireSuite'")

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

// Standalone tests
type StandaloneSuite struct{}

func pythonDir() string {
	cwd, _ := os.Getwd()
	return fmt.Sprintf("%s/../../python/tests", cwd)
}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	if *no_server {
		c.Skip("Skipping tests that require server")
		return
	}
	os.Chdir(pythonDir())
	{
		cmd := exec.Command("python", "run_test_server.py", "start")
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalf("Setting up stderr pipe: %s", err)
		}
		go io.Copy(os.Stderr, stderr)
		if err := cmd.Run(); err != nil {
			panic(fmt.Sprintf("'python run_test_server.py start' returned error %s", err))
		}
	}
	{
		cmd := exec.Command("python", "run_test_server.py", "start_keep")
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalf("Setting up stderr pipe: %s", err)
		}
		go io.Copy(os.Stderr, stderr)
		if err := cmd.Run(); err != nil {
			panic(fmt.Sprintf("'python run_test_server.py start_keep' returned error %s", err))
		}
	}
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	os.Chdir(pythonDir())
	exec.Command("python", "run_test_server.py", "stop_keep").Run()
	exec.Command("python", "run_test_server.py", "stop").Run()
}

func (s *ServerRequiredSuite) TestMakeKeepClient(c *C) {
	os.Setenv("ARVADOS_API_HOST", "localhost:3000")
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)

	kc, err := MakeKeepClient(&arv)

	c.Assert(err, Equals, nil)
	c.Check(len(kc.ServiceRoots()), Equals, 2)
	for _, root := range kc.ServiceRoots() {
		c.Check(root, Matches, "http://localhost:2510[\\d]")
	}
}

type StubPutHandler struct {
	c              *C
	expectPath     string
	expectApiToken string
	expectBody     string
	handled        chan string
}

func (this StubPutHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	this.c.Check(req.URL.Path, Equals, "/"+this.expectPath)
	this.c.Check(req.Header.Get("Authorization"), Equals, fmt.Sprintf("OAuth2 %s", this.expectApiToken))
	body, err := ioutil.ReadAll(req.Body)
	this.c.Check(err, Equals, nil)
	this.c.Check(body, DeepEquals, []byte(this.expectBody))
	resp.WriteHeader(200)
	this.handled <- fmt.Sprintf("http://%s", req.Host)
}

func RunFakeKeepServer(st http.Handler) (ks KeepServer) {
	var err error
	ks.listener, err = net.ListenTCP("tcp", &net.TCPAddr{Port: 0})
	if err != nil {
		panic(fmt.Sprintf("Could not listen on any port"))
	}
	ks.url = fmt.Sprintf("http://%s", ks.listener.Addr().String())
	go http.Serve(ks.listener, st)
	return
}

func UploadToStubHelper(c *C, st http.Handler, f func(KeepClient, string,
	io.ReadCloser, io.WriteCloser, chan uploadStatus)) {

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, _ := arvadosclient.MakeArvadosClient()
	arv.ApiToken = "abc123"

	kc, _ := MakeKeepClient(&arv)

	reader, writer := io.Pipe()
	upload_status := make(chan uploadStatus)

	f(kc, ks.url, reader, writer, upload_status)
}

func (s *StandaloneSuite) TestUploadToStubKeepServer(c *C) {
	log.Printf("TestUploadToStubKeepServer")

	st := StubPutHandler{
		c,
		"acbd18db4cc2f85cedef654fccc4a4d8",
		"abc123",
		"foo",
		make(chan string)}

	UploadToStubHelper(c, st,
		func(kc KeepClient, url string, reader io.ReadCloser,
			writer io.WriteCloser, upload_status chan uploadStatus) {

			go kc.uploadToKeepServer(url, st.expectPath, reader, upload_status, int64(len("foo")), "TestUploadToStubKeepServer")

			writer.Write([]byte("foo"))
			writer.Close()

			<-st.handled
			status := <-upload_status
			c.Check(status, DeepEquals, uploadStatus{nil, fmt.Sprintf("%s/%s", url, st.expectPath), 200, 1, ""})
		})

	log.Printf("TestUploadToStubKeepServer done")
}

func (s *StandaloneSuite) TestUploadToStubKeepServerBufferReader(c *C) {
	log.Printf("TestUploadToStubKeepServerBufferReader")

	st := StubPutHandler{
		c,
		"acbd18db4cc2f85cedef654fccc4a4d8",
		"abc123",
		"foo",
		make(chan string)}

	UploadToStubHelper(c, st,
		func(kc KeepClient, url string, reader io.ReadCloser,
			writer io.WriteCloser, upload_status chan uploadStatus) {

			tr := streamer.AsyncStreamFromReader(512, reader)
			defer tr.Close()

			br1 := tr.MakeStreamReader()

			go kc.uploadToKeepServer(url, st.expectPath, br1, upload_status, 3, "TestUploadToStubKeepServerBufferReader")

			writer.Write([]byte("foo"))
			writer.Close()

			<-st.handled

			status := <-upload_status
			c.Check(status, DeepEquals, uploadStatus{nil, fmt.Sprintf("%s/%s", url, st.expectPath), 200, 1, ""})
		})

	log.Printf("TestUploadToStubKeepServerBufferReader done")
}

type FailHandler struct {
	handled chan string
}

func (this FailHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(500)
	this.handled <- fmt.Sprintf("http://%s", req.Host)
}

func (s *StandaloneSuite) TestFailedUploadToStubKeepServer(c *C) {
	log.Printf("TestFailedUploadToStubKeepServer")

	st := FailHandler{
		make(chan string)}

	hash := "acbd18db4cc2f85cedef654fccc4a4d8"

	UploadToStubHelper(c, st,
		func(kc KeepClient, url string, reader io.ReadCloser,
			writer io.WriteCloser, upload_status chan uploadStatus) {

			go kc.uploadToKeepServer(url, hash, reader, upload_status, 3, "TestFailedUploadToStubKeepServer")

			writer.Write([]byte("foo"))
			writer.Close()

			<-st.handled

			status := <-upload_status
			c.Check(status.url, Equals, fmt.Sprintf("%s/%s", url, hash))
			c.Check(status.statusCode, Equals, 500)
		})
	log.Printf("TestFailedUploadToStubKeepServer done")
}

type KeepServer struct {
	listener net.Listener
	url      string
}

func RunSomeFakeKeepServers(st http.Handler, n int) (ks []KeepServer) {
	ks = make([]KeepServer, n)

	for i := 0; i < n; i += 1 {
		ks[i] = RunFakeKeepServer(st)
	}

	return ks
}

func (s *StandaloneSuite) TestPutB(c *C) {
	log.Printf("TestPutB")

	hash := Md5String("foo")

	st := StubPutHandler{
		c,
		hash,
		"abc123",
		"foo",
		make(chan string, 5)}

	arv, _ := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	service_roots := make(map[string]string)

	ks := RunSomeFakeKeepServers(st, 5)

	for i, k := range ks {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(service_roots)

	kc.PutB([]byte("foo"))

	shuff := NewRootSorter(
		kc.ServiceRoots(), Md5String("foo")).GetSortedRoots()

	s1 := <-st.handled
	s2 := <-st.handled
	c.Check((s1 == shuff[0] && s2 == shuff[1]) ||
		(s1 == shuff[1] && s2 == shuff[0]),
		Equals,
		true)

	log.Printf("TestPutB done")
}

func (s *StandaloneSuite) TestPutHR(c *C) {
	log.Printf("TestPutHR")

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubPutHandler{
		c,
		hash,
		"abc123",
		"foo",
		make(chan string, 5)}

	arv, _ := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	service_roots := make(map[string]string)

	ks := RunSomeFakeKeepServers(st, 5)

	for i, k := range ks {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(service_roots)

	reader, writer := io.Pipe()

	go func() {
		writer.Write([]byte("foo"))
		writer.Close()
	}()

	kc.PutHR(hash, reader, 3)

	shuff := NewRootSorter(kc.ServiceRoots(), hash).GetSortedRoots()
	log.Print(shuff)

	s1 := <-st.handled
	s2 := <-st.handled

	c.Check((s1 == shuff[0] && s2 == shuff[1]) ||
		(s1 == shuff[1] && s2 == shuff[0]),
		Equals,
		true)

	log.Printf("TestPutHR done")
}

func (s *StandaloneSuite) TestPutWithFail(c *C) {
	log.Printf("TestPutWithFail")

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubPutHandler{
		c,
		hash,
		"abc123",
		"foo",
		make(chan string, 4)}

	fh := FailHandler{
		make(chan string, 1)}

	arv, err := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	service_roots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 4)
	ks2 := RunSomeFakeKeepServers(fh, 1)

	for i, k := range ks1 {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(service_roots)

	shuff := NewRootSorter(
		kc.ServiceRoots(), Md5String("foo")).GetSortedRoots()

	phash, replicas, err := kc.PutB([]byte("foo"))

	<-fh.handled

	c.Check(err, Equals, nil)
	c.Check(phash, Equals, "")
	c.Check(replicas, Equals, 2)

	s1 := <-st.handled
	s2 := <-st.handled

	c.Check((s1 == shuff[1] && s2 == shuff[2]) ||
		(s1 == shuff[2] && s2 == shuff[1]),
		Equals,
		true)
}

func (s *StandaloneSuite) TestPutWithTooManyFail(c *C) {
	log.Printf("TestPutWithTooManyFail")

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubPutHandler{
		c,
		hash,
		"abc123",
		"foo",
		make(chan string, 1)}

	fh := FailHandler{
		make(chan string, 4)}

	arv, err := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	service_roots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 1)
	ks2 := RunSomeFakeKeepServers(fh, 4)

	for i, k := range ks1 {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(service_roots)

	_, replicas, err := kc.PutB([]byte("foo"))

	c.Check(err, Equals, InsufficientReplicasError)
	c.Check(replicas, Equals, 1)
	c.Check(<-st.handled, Equals, ks1[0].url)

	log.Printf("TestPutWithTooManyFail done")
}

type StubGetHandler struct {
	c              *C
	expectPath     string
	expectApiToken string
	returnBody     []byte
}

func (this StubGetHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	this.c.Check(req.URL.Path, Equals, "/"+this.expectPath)
	this.c.Check(req.Header.Get("Authorization"), Equals, fmt.Sprintf("OAuth2 %s", this.expectApiToken))
	resp.Header().Set("Content-Length", fmt.Sprintf("%d", len(this.returnBody)))
	resp.Write(this.returnBody)
}

func (s *StandaloneSuite) TestGet(c *C) {
	log.Printf("TestGet")

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubGetHandler{
		c,
		hash,
		"abc123",
		[]byte("foo")}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url})

	r, n, url2, err := kc.Get(hash)
	defer r.Close()
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(url2, Equals, fmt.Sprintf("%s/%s", ks.url, hash))

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(content, DeepEquals, []byte("foo"))

	log.Printf("TestGet done")
}

func (s *StandaloneSuite) TestGetFail(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := FailHandler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url})

	r, n, url2, err := kc.Get(hash)
	c.Check(err, Equals, BlockNotFound)
	c.Check(n, Equals, int64(0))
	c.Check(url2, Equals, "")
	c.Check(r, Equals, nil)
}

type BarHandler struct {
	handled chan string
}

func (this BarHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte("bar"))
	this.handled <- fmt.Sprintf("http://%s", req.Host)
}

func (s *StandaloneSuite) TestChecksum(c *C) {
	foohash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	barhash := fmt.Sprintf("%x", md5.Sum([]byte("bar")))

	st := BarHandler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url})

	r, n, _, err := kc.Get(barhash)
	_, err = ioutil.ReadAll(r)
	c.Check(n, Equals, int64(3))
	c.Check(err, Equals, nil)

	<-st.handled

	r, n, _, err = kc.Get(foohash)
	_, err = ioutil.ReadAll(r)
	c.Check(n, Equals, int64(3))
	c.Check(err, Equals, BadChecksum)

	<-st.handled
}

func (s *StandaloneSuite) TestGetWithFailures(c *C) {
	content := []byte("waz")
	hash := fmt.Sprintf("%x", md5.Sum(content))

	fh := FailHandler{
		make(chan string, 4)}

	st := StubGetHandler{
		c,
		hash,
		"abc123",
		content}

	arv, err := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)
	arv.ApiToken = "abc123"
	service_roots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 1)
	ks2 := RunSomeFakeKeepServers(fh, 4)

	for i, k := range ks1 {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(service_roots)

	// This test works only if one of the failing services is
	// attempted before the succeeding service. Otherwise,
	// <-fh.handled below will just hang! (Probe order depends on
	// the choice of block content "waz" and the UUIDs of the fake
	// servers, so we just tried different strings until we found
	// an example that passes this Assert.)
	c.Assert(NewRootSorter(service_roots, hash).GetSortedRoots()[0], Not(Equals), ks1[0].url)

	r, n, url2, err := kc.Get(hash)

	<-fh.handled
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(url2, Equals, fmt.Sprintf("%s/%s", ks1[0].url, hash))

	read_content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(read_content, DeepEquals, content)
}

func (s *ServerRequiredSuite) TestPutGetHead(c *C) {
	os.Setenv("ARVADOS_API_HOST", "localhost:3000")
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")
	content := []byte("TestPutGetHead")

	arv, err := arvadosclient.MakeArvadosClient()
	kc, err := MakeKeepClient(&arv)
	c.Assert(err, Equals, nil)

	hash := fmt.Sprintf("%x", md5.Sum(content))

	{
		n, _, err := kc.Ask(hash)
		c.Check(err, Equals, BlockNotFound)
		c.Check(n, Equals, int64(0))
	}
	{
		hash2, replicas, err := kc.PutB(content)
		c.Check(hash2, Equals, fmt.Sprintf("%s+%d", hash, len(content)))
		c.Check(replicas, Equals, 2)
		c.Check(err, Equals, nil)
	}
	{
		r, n, url2, err := kc.Get(hash)
		c.Check(err, Equals, nil)
		c.Check(n, Equals, int64(len(content)))
		c.Check(url2, Equals, fmt.Sprintf("http://localhost:25108/%s", hash))

		read_content, err2 := ioutil.ReadAll(r)
		c.Check(err2, Equals, nil)
		c.Check(read_content, DeepEquals, content)
	}
	{
		n, url2, err := kc.Ask(hash)
		c.Check(err, Equals, nil)
		c.Check(n, Equals, int64(len(content)))
		c.Check(url2, Equals, fmt.Sprintf("http://localhost:25108/%s", hash))
	}
}

type StubProxyHandler struct {
	handled chan string
}

func (this StubProxyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("X-Keep-Replicas-Stored", "2")
	this.handled <- fmt.Sprintf("http://%s", req.Host)
}

func (s *StandaloneSuite) TestPutProxy(c *C) {
	log.Printf("TestPutProxy")

	st := StubProxyHandler{make(chan string, 1)}

	arv, err := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)

	kc.Want_replicas = 2
	kc.Using_proxy = true
	arv.ApiToken = "abc123"
	service_roots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 1)

	for i, k := range ks1 {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(service_roots)

	_, replicas, err := kc.PutB([]byte("foo"))
	<-st.handled

	c.Check(err, Equals, nil)
	c.Check(replicas, Equals, 2)

	log.Printf("TestPutProxy done")
}

func (s *StandaloneSuite) TestPutProxyInsufficientReplicas(c *C) {
	log.Printf("TestPutProxy")

	st := StubProxyHandler{make(chan string, 1)}

	arv, err := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(&arv)

	kc.Want_replicas = 3
	kc.Using_proxy = true
	arv.ApiToken = "abc123"
	service_roots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 1)

	for i, k := range ks1 {
		service_roots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}
	kc.SetServiceRoots(service_roots)

	_, replicas, err := kc.PutB([]byte("foo"))
	<-st.handled

	c.Check(err, Equals, InsufficientReplicasError)
	c.Check(replicas, Equals, 2)

	log.Printf("TestPutProxy done")
}

func (s *StandaloneSuite) TestMakeLocator(c *C) {
	l := MakeLocator("91f372a266fe2bf2823cb8ec7fda31ce+3+Aabcde@12345678")

	c.Check(l.Hash, Equals, "91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(l.Size, Equals, 3)
	c.Check(l.Signature, Equals, "abcde")
	c.Check(l.Timestamp, Equals, "12345678")
}
