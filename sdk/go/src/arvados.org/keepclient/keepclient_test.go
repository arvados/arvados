package keepclient

import (
	"arvados.org/buffer"
	"crypto/md5"
	"flag"
	"fmt"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
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
	gopath := os.Getenv("GOPATH")
	return fmt.Sprintf("%s/../python", strings.Split(gopath, ":")[0])
}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	if *no_server {
		c.Skip("Skipping tests that require server")
	} else {
		os.Chdir(pythonDir())
		exec.Command("python", "run_test_server.py", "start").Run()
		exec.Command("python", "run_test_server.py", "start_keep").Run()
	}
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	os.Chdir(pythonDir())
	exec.Command("python", "run_test_server.py", "stop_keep").Run()
	exec.Command("python", "run_test_server.py", "stop").Run()
}

func (s *ServerRequiredSuite) TestMakeKeepClient(c *C) {
	os.Setenv("ARVADOS_API_HOST", "localhost:3001")
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "")

	kc, err := MakeKeepClient()
	c.Check(kc.ApiServer, Equals, "localhost:3001")
	c.Check(kc.ApiToken, Equals, "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	c.Check(kc.ApiInsecure, Equals, false)

	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	kc, err = MakeKeepClient()
	c.Check(kc.ApiServer, Equals, "localhost:3001")
	c.Check(kc.ApiToken, Equals, "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	c.Check(kc.ApiInsecure, Equals, true)
	c.Check(kc.Client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify, Equals, true)

	c.Assert(err, Equals, nil)
	c.Check(len(kc.Service_roots), Equals, 2)
	c.Check(kc.Service_roots[0], Equals, "http://localhost:25107")
	c.Check(kc.Service_roots[1], Equals, "http://localhost:25108")
}

func (s *StandaloneSuite) TestShuffleServiceRoots(c *C) {
	kc := KeepClient{Service_roots: []string{"http://localhost:25107", "http://localhost:25108", "http://localhost:25109", "http://localhost:25110", "http://localhost:25111", "http://localhost:25112", "http://localhost:25113", "http://localhost:25114", "http://localhost:25115", "http://localhost:25116", "http://localhost:25117", "http://localhost:25118", "http://localhost:25119", "http://localhost:25120", "http://localhost:25121", "http://localhost:25122", "http://localhost:25123"}}

	// "foo" acbd18db4cc2f85cedef654fccc4a4d8
	foo_shuffle := []string{"http://localhost:25116", "http://localhost:25120", "http://localhost:25119", "http://localhost:25122", "http://localhost:25108", "http://localhost:25114", "http://localhost:25112", "http://localhost:25107", "http://localhost:25118", "http://localhost:25111", "http://localhost:25113", "http://localhost:25121", "http://localhost:25110", "http://localhost:25117", "http://localhost:25109", "http://localhost:25115", "http://localhost:25123"}
	c.Check(kc.shuffledServiceRoots("acbd18db4cc2f85cedef654fccc4a4d8"), DeepEquals, foo_shuffle)

	// "bar" 37b51d194a7513e45b56f6524f2d51f2
	bar_shuffle := []string{"http://localhost:25108", "http://localhost:25112", "http://localhost:25119", "http://localhost:25107", "http://localhost:25110", "http://localhost:25116", "http://localhost:25122", "http://localhost:25120", "http://localhost:25121", "http://localhost:25117", "http://localhost:25111", "http://localhost:25123", "http://localhost:25118", "http://localhost:25113", "http://localhost:25114", "http://localhost:25115", "http://localhost:25109"}
	c.Check(kc.shuffledServiceRoots("37b51d194a7513e45b56f6524f2d51f2"), DeepEquals, bar_shuffle)
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

func RunBogusKeepServer(st http.Handler, port int) (listener net.Listener, url string) {
	server := http.Server{Handler: st}

	var err error
	listener, err = net.ListenTCP("tcp", &net.TCPAddr{Port: port})
	if err != nil {
		panic(fmt.Sprintf("Could not listen on tcp port %v", port))
	}

	url = fmt.Sprintf("http://localhost:%d", listener.Addr().(*net.TCPAddr).Port)

	go server.Serve(listener)
	return listener, url
}

func UploadToStubHelper(c *C, st http.Handler, f func(KeepClient, string,
	io.ReadCloser, io.WriteCloser, chan uploadStatus)) {

	listener, url := RunBogusKeepServer(st, 2990)
	defer listener.Close()

	kc, _ := MakeKeepClient()
	kc.ApiToken = "abc123"

	reader, writer := io.Pipe()
	upload_status := make(chan uploadStatus)

	f(kc, url, reader, writer, upload_status)
}

func (s *StandaloneSuite) TestUploadToStubKeepServer(c *C) {
	st := StubPutHandler{
		c,
		"acbd18db4cc2f85cedef654fccc4a4d8",
		"abc123",
		"foo",
		make(chan string)}

	UploadToStubHelper(c, st,
		func(kc KeepClient, url string, reader io.ReadCloser,
			writer io.WriteCloser, upload_status chan uploadStatus) {

			go kc.uploadToKeepServer(url, st.expectPath, reader, upload_status, int64(len("foo")))

			writer.Write([]byte("foo"))
			writer.Close()

			<-st.handled
			status := <-upload_status
			c.Check(status, DeepEquals, uploadStatus{nil, fmt.Sprintf("%s/%s", url, st.expectPath), 200})
		})
}

func (s *StandaloneSuite) TestUploadToStubKeepServerBufferReader(c *C) {
	st := StubPutHandler{
		c,
		"acbd18db4cc2f85cedef654fccc4a4d8",
		"abc123",
		"foo",
		make(chan string)}

	UploadToStubHelper(c, st,
		func(kc KeepClient, url string, reader io.ReadCloser,
			writer io.WriteCloser, upload_status chan uploadStatus) {

			tr := buffer.StartTransferFromReader(512, reader)
			defer tr.Close()

			br1 := tr.MakeBufferReader()

			go kc.uploadToKeepServer(url, st.expectPath, br1, upload_status, 3)

			writer.Write([]byte("foo"))
			writer.Close()

			<-tr.Reader_status
			<-st.handled

			status := <-upload_status
			c.Check(status, DeepEquals, uploadStatus{nil, fmt.Sprintf("%s/%s", url, st.expectPath), 200})
		})
}

type FailHandler struct {
	handled chan string
}

func (this FailHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(500)
	this.handled <- fmt.Sprintf("http://%s", req.Host)
}

func (s *StandaloneSuite) TestFailedUploadToStubKeepServer(c *C) {
	st := FailHandler{
		make(chan string)}

	hash := "acbd18db4cc2f85cedef654fccc4a4d8"

	UploadToStubHelper(c, st,
		func(kc KeepClient, url string, reader io.ReadCloser,
			writer io.WriteCloser, upload_status chan uploadStatus) {

			go kc.uploadToKeepServer(url, hash, reader, upload_status, 3)

			writer.Write([]byte("foo"))
			writer.Close()

			<-st.handled

			status := <-upload_status
			c.Check(status.Url, Equals, fmt.Sprintf("%s/%s", url, hash))
			c.Check(status.StatusCode, Equals, 500)
		})

}

type KeepServer struct {
	listener net.Listener
	url      string
}

func RunSomeFakeKeepServers(st http.Handler, n int, port int) (ks []KeepServer) {
	ks = make([]KeepServer, n)

	for i := 0; i < n; i += 1 {
		boguslistener, bogusurl := RunBogusKeepServer(st, port+i)
		ks[i] = KeepServer{boguslistener, bogusurl}
	}

	return ks
}

func (s *StandaloneSuite) TestPutB(c *C) {
	log.Printf("TestPutB")

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubPutHandler{
		c,
		hash,
		"abc123",
		"foo",
		make(chan string, 2)}

	kc, _ := MakeKeepClient()

	kc.Want_replicas = 2
	kc.ApiToken = "abc123"
	kc.Service_roots = make([]string, 5)

	ks := RunSomeFakeKeepServers(st, 5, 2990)

	for i := 0; i < len(ks); i += 1 {
		kc.Service_roots[i] = ks[i].url
		defer ks[i].listener.Close()
	}

	sort.Strings(kc.Service_roots)

	kc.PutB([]byte("foo"))

	shuff := kc.shuffledServiceRoots(fmt.Sprintf("%x", md5.Sum([]byte("foo"))))

	s1 := <-st.handled
	s2 := <-st.handled
	c.Check((s1 == shuff[0] && s2 == shuff[1]) ||
		(s1 == shuff[1] && s2 == shuff[0]),
		Equals,
		true)
}

func (s *StandaloneSuite) TestPutHR(c *C) {
	log.Printf("TestPutHR")

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubPutHandler{
		c,
		hash,
		"abc123",
		"foo",
		make(chan string, 2)}

	kc, _ := MakeKeepClient()

	kc.Want_replicas = 2
	kc.ApiToken = "abc123"
	kc.Service_roots = make([]string, 5)

	ks := RunSomeFakeKeepServers(st, 5, 2990)

	for i := 0; i < len(ks); i += 1 {
		kc.Service_roots[i] = ks[i].url
		defer ks[i].listener.Close()
	}

	sort.Strings(kc.Service_roots)

	reader, writer := io.Pipe()

	go func() {
		writer.Write([]byte("foo"))
		writer.Close()
	}()

	kc.PutHR(hash, reader, 3)

	shuff := kc.shuffledServiceRoots(hash)
	log.Print(shuff)

	s1 := <-st.handled
	s2 := <-st.handled

	c.Check((s1 == shuff[0] && s2 == shuff[1]) ||
		(s1 == shuff[1] && s2 == shuff[0]),
		Equals,
		true)
}

func (s *StandaloneSuite) TestPutWithFail(c *C) {
	log.Printf("TestPutWithFail")

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubPutHandler{
		c,
		hash,
		"abc123",
		"foo",
		make(chan string, 2)}

	fh := FailHandler{
		make(chan string, 1)}

	kc, _ := MakeKeepClient()

	kc.Want_replicas = 2
	kc.ApiToken = "abc123"
	kc.Service_roots = make([]string, 5)

	ks1 := RunSomeFakeKeepServers(st, 4, 2990)
	ks2 := RunSomeFakeKeepServers(fh, 1, 2995)

	for i, k := range ks1 {
		kc.Service_roots[i] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		kc.Service_roots[len(ks1)+i] = k.url
		defer k.listener.Close()
	}

	sort.Strings(kc.Service_roots)

	shuff := kc.shuffledServiceRoots(fmt.Sprintf("%x", md5.Sum([]byte("foo"))))

	phash, replicas, err := kc.PutB([]byte("foo"))

	<-fh.handled

	c.Check(err, Equals, nil)
	c.Check(phash, Equals, hash)
	c.Check(replicas, Equals, 2)
	c.Check(<-st.handled, Equals, shuff[1])
	c.Check(<-st.handled, Equals, shuff[2])
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

	kc, _ := MakeKeepClient()

	kc.Want_replicas = 2
	kc.ApiToken = "abc123"
	kc.Service_roots = make([]string, 5)

	ks1 := RunSomeFakeKeepServers(st, 1, 2990)
	ks2 := RunSomeFakeKeepServers(fh, 4, 2991)

	for i, k := range ks1 {
		kc.Service_roots[i] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		kc.Service_roots[len(ks1)+i] = k.url
		defer k.listener.Close()
	}

	sort.Strings(kc.Service_roots)

	shuff := kc.shuffledServiceRoots(fmt.Sprintf("%x", md5.Sum([]byte("foo"))))

	_, replicas, err := kc.PutB([]byte("foo"))

	c.Check(err, Equals, InsufficientReplicasError)
	c.Check(replicas, Equals, 1)
	c.Check(<-st.handled, Equals, shuff[1])
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

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubGetHandler{
		c,
		hash,
		"abc123",
		[]byte("foo")}

	listener, url := RunBogusKeepServer(st, 2990)
	defer listener.Close()

	kc, _ := MakeKeepClient()
	kc.ApiToken = "abc123"
	kc.Service_roots = []string{url}

	r, n, url2, err := kc.Get(hash)
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(url2, Equals, fmt.Sprintf("%s/%s", url, hash))

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(content, DeepEquals, []byte("foo"))
}

func (s *StandaloneSuite) TestGetFail(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := FailHandler{make(chan string, 1)}

	listener, url := RunBogusKeepServer(st, 2990)
	defer listener.Close()

	kc, _ := MakeKeepClient()
	kc.ApiToken = "abc123"
	kc.Service_roots = []string{url}

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

	listener, url := RunBogusKeepServer(st, 2990)
	defer listener.Close()

	kc, _ := MakeKeepClient()
	kc.ApiToken = "abc123"
	kc.Service_roots = []string{url}

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

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	fh := FailHandler{
		make(chan string, 1)}

	st := StubGetHandler{
		c,
		hash,
		"abc123",
		[]byte("foo")}

	kc, _ := MakeKeepClient()
	kc.ApiToken = "abc123"
	kc.Service_roots = make([]string, 5)

	ks1 := RunSomeFakeKeepServers(st, 1, 2990)
	ks2 := RunSomeFakeKeepServers(fh, 4, 2991)

	for i, k := range ks1 {
		kc.Service_roots[i] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		kc.Service_roots[len(ks1)+i] = k.url
		defer k.listener.Close()
	}

	sort.Strings(kc.Service_roots)

	r, n, url2, err := kc.Get(hash)
	<-fh.handled
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(url2, Equals, fmt.Sprintf("%s/%s", ks1[0].url, hash))

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(content, DeepEquals, []byte("foo"))
}

func (s *ServerRequiredSuite) TestPutGetHead(c *C) {
	os.Setenv("ARVADOS_API_HOST", "localhost:3001")
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	kc, err := MakeKeepClient()
	c.Assert(err, Equals, nil)

	hash, replicas, err := kc.PutB([]byte("foo"))
	c.Check(hash, Equals, fmt.Sprintf("%x", md5.Sum([]byte("foo"))))
	c.Check(replicas, Equals, 2)
	c.Check(err, Equals, nil)

	{
		r, n, url2, err := kc.Get(hash)
		c.Check(err, Equals, nil)
		c.Check(n, Equals, int64(3))
		c.Check(url2, Equals, fmt.Sprintf("http://localhost:25108/%s", hash))

		content, err2 := ioutil.ReadAll(r)
		c.Check(err2, Equals, nil)
		c.Check(content, DeepEquals, []byte("foo"))
	}

	{
		n, url2, err := kc.Ask(hash)
		c.Check(err, Equals, nil)
		c.Check(n, Equals, int64(3))
		c.Check(url2, Equals, fmt.Sprintf("http://localhost:25108/%s", hash))
	}
}
