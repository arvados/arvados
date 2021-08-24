// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&ServerRequiredSuite{})
var _ = Suite(&StandaloneSuite{})

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

// Standalone tests
type StandaloneSuite struct{}

func (s *StandaloneSuite) SetUpTest(c *C) {
	RefreshServiceDiscovery()
}

func pythonDir() string {
	cwd, _ := os.Getwd()
	return fmt.Sprintf("%s/../../python/tests", cwd)
}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	arvadostest.StartKeep(2, false)
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep(2)
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	RefreshServiceDiscovery()
}

func (s *ServerRequiredSuite) TestMakeKeepClient(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)

	kc, err := MakeKeepClient(arv)

	c.Assert(err, Equals, nil)
	c.Check(len(kc.LocalRoots()), Equals, 2)
	for _, root := range kc.LocalRoots() {
		c.Check(root, Matches, "http://localhost:\\d+")
	}
}

func (s *ServerRequiredSuite) TestDefaultReplications(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)

	kc, err := MakeKeepClient(arv)
	c.Check(err, IsNil)
	c.Assert(kc.Want_replicas, Equals, 2)

	arv.DiscoveryDoc["defaultCollectionReplication"] = 3.0
	kc, err = MakeKeepClient(arv)
	c.Check(err, IsNil)
	c.Assert(kc.Want_replicas, Equals, 3)

	arv.DiscoveryDoc["defaultCollectionReplication"] = 1.0
	kc, err = MakeKeepClient(arv)
	c.Check(err, IsNil)
	c.Assert(kc.Want_replicas, Equals, 1)
}

type StubPutHandler struct {
	c                    *C
	expectPath           string
	expectAPIToken       string
	expectBody           string
	expectStorageClass   string
	returnStorageClasses string
	handled              chan string
	requests             []*http.Request
	mtx                  sync.Mutex
}

func (sph *StubPutHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	sph.mtx.Lock()
	sph.requests = append(sph.requests, req)
	sph.mtx.Unlock()
	sph.c.Check(req.URL.Path, Equals, "/"+sph.expectPath)
	sph.c.Check(req.Header.Get("Authorization"), Equals, fmt.Sprintf("OAuth2 %s", sph.expectAPIToken))
	if sph.expectStorageClass != "*" {
		sph.c.Check(req.Header.Get("X-Keep-Storage-Classes"), Equals, sph.expectStorageClass)
	}
	body, err := ioutil.ReadAll(req.Body)
	sph.c.Check(err, Equals, nil)
	sph.c.Check(body, DeepEquals, []byte(sph.expectBody))
	resp.Header().Set("X-Keep-Replicas-Stored", "1")
	if sph.returnStorageClasses != "" {
		resp.Header().Set("X-Keep-Storage-Classes-Confirmed", sph.returnStorageClasses)
	}
	resp.WriteHeader(200)
	sph.handled <- fmt.Sprintf("http://%s", req.Host)
}

func RunFakeKeepServer(st http.Handler) (ks KeepServer) {
	var err error
	// If we don't explicitly bind it to localhost, ks.listener.Addr() will
	// bind to 0.0.0.0 or [::] which is not a valid address for Dial()
	ks.listener, err = net.ListenTCP("tcp", &net.TCPAddr{IP: []byte{127, 0, 0, 1}, Port: 0})
	if err != nil {
		panic(fmt.Sprintf("Could not listen on any port"))
	}
	ks.url = fmt.Sprintf("http://%s", ks.listener.Addr().String())
	go http.Serve(ks.listener, st)
	return
}

func UploadToStubHelper(c *C, st http.Handler, f func(*KeepClient, string,
	io.ReadCloser, io.WriteCloser, chan uploadStatus)) {

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, _ := arvadosclient.MakeArvadosClient()
	arv.ApiToken = "abc123"

	kc, _ := MakeKeepClient(arv)

	reader, writer := io.Pipe()
	uploadStatusChan := make(chan uploadStatus)

	f(kc, ks.url, reader, writer, uploadStatusChan)
}

func (s *StandaloneSuite) TestUploadToStubKeepServer(c *C) {
	log.Printf("TestUploadToStubKeepServer")

	st := &StubPutHandler{
		c:                    c,
		expectPath:           "acbd18db4cc2f85cedef654fccc4a4d8",
		expectAPIToken:       "abc123",
		expectBody:           "foo",
		expectStorageClass:   "",
		returnStorageClasses: "default=1",
		handled:              make(chan string),
	}

	UploadToStubHelper(c, st,
		func(kc *KeepClient, url string, reader io.ReadCloser, writer io.WriteCloser, uploadStatusChan chan uploadStatus) {
			go kc.uploadToKeepServer(url, st.expectPath, nil, reader, uploadStatusChan, len("foo"), kc.getRequestID())

			writer.Write([]byte("foo"))
			writer.Close()

			<-st.handled
			status := <-uploadStatusChan
			c.Check(status, DeepEquals, uploadStatus{nil, fmt.Sprintf("%s/%s", url, st.expectPath), 200, 1, map[string]int{"default": 1}, ""})
		})
}

func (s *StandaloneSuite) TestUploadToStubKeepServerBufferReader(c *C) {
	st := &StubPutHandler{
		c:                    c,
		expectPath:           "acbd18db4cc2f85cedef654fccc4a4d8",
		expectAPIToken:       "abc123",
		expectBody:           "foo",
		expectStorageClass:   "",
		returnStorageClasses: "default=1",
		handled:              make(chan string),
	}

	UploadToStubHelper(c, st,
		func(kc *KeepClient, url string, _ io.ReadCloser, _ io.WriteCloser, uploadStatusChan chan uploadStatus) {
			go kc.uploadToKeepServer(url, st.expectPath, nil, bytes.NewBuffer([]byte("foo")), uploadStatusChan, 3, kc.getRequestID())

			<-st.handled

			status := <-uploadStatusChan
			c.Check(status, DeepEquals, uploadStatus{nil, fmt.Sprintf("%s/%s", url, st.expectPath), 200, 1, map[string]int{"default": 1}, ""})
		})
}

func (s *StandaloneSuite) TestUploadWithStorageClasses(c *C) {
	for _, trial := range []struct {
		respHeader string
		expectMap  map[string]int
	}{
		{"", nil},
		{"foo=1", map[string]int{"foo": 1}},
		{" foo=1 , bar=2 ", map[string]int{"foo": 1, "bar": 2}},
		{" =foo=1 ", nil},
		{"foo", nil},
	} {
		st := &StubPutHandler{
			c:                    c,
			expectPath:           "acbd18db4cc2f85cedef654fccc4a4d8",
			expectAPIToken:       "abc123",
			expectBody:           "foo",
			expectStorageClass:   "",
			returnStorageClasses: trial.respHeader,
			handled:              make(chan string),
		}

		UploadToStubHelper(c, st,
			func(kc *KeepClient, url string, reader io.ReadCloser, writer io.WriteCloser, uploadStatusChan chan uploadStatus) {
				go kc.uploadToKeepServer(url, st.expectPath, nil, reader, uploadStatusChan, len("foo"), kc.getRequestID())

				writer.Write([]byte("foo"))
				writer.Close()

				<-st.handled
				status := <-uploadStatusChan
				c.Check(status, DeepEquals, uploadStatus{nil, fmt.Sprintf("%s/%s", url, st.expectPath), 200, 1, trial.expectMap, ""})
			})
	}
}

func (s *StandaloneSuite) TestPutWithStorageClasses(c *C) {
	nServers := 5
	for _, trial := range []struct {
		replicas      int
		clientClasses []string
		putClasses    []string // putClasses takes precedence over clientClasses
		minRequests   int
		maxRequests   int
		success       bool
	}{
		{1, []string{"class1"}, nil, 1, 1, true},
		{2, []string{"class1"}, nil, 1, 2, true},
		{3, []string{"class1"}, nil, 2, 3, true},
		{1, []string{"class1", "class2"}, nil, 1, 1, true},
		{3, nil, []string{"class1"}, 2, 3, true},
		{1, nil, []string{"class1", "class2"}, 1, 1, true},
		{1, []string{"class404"}, []string{"class1", "class2"}, 1, 1, true},
		{1, []string{"class1"}, []string{"class404", "class2"}, nServers, nServers, false},
		{nServers*2 + 1, []string{"class1"}, nil, nServers, nServers, false},
		{1, []string{"class404"}, nil, nServers, nServers, false},
		{1, []string{"class1", "class404"}, nil, nServers, nServers, false},
		{1, nil, []string{"class1", "class404"}, nServers, nServers, false},
	} {
		c.Logf("%+v", trial)
		st := &StubPutHandler{
			c:                    c,
			expectPath:           "acbd18db4cc2f85cedef654fccc4a4d8",
			expectAPIToken:       "abc123",
			expectBody:           "foo",
			expectStorageClass:   "*",
			returnStorageClasses: "class1=2, class2=2",
			handled:              make(chan string, 100),
		}
		ks := RunSomeFakeKeepServers(st, nServers)
		arv, _ := arvadosclient.MakeArvadosClient()
		kc, _ := MakeKeepClient(arv)
		kc.Want_replicas = trial.replicas
		kc.StorageClasses = trial.clientClasses
		arv.ApiToken = "abc123"
		localRoots := make(map[string]string)
		writableLocalRoots := make(map[string]string)
		for i, k := range ks {
			localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
			writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
			defer k.listener.Close()
		}
		kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

		_, err := kc.BlockWrite(context.Background(), arvados.BlockWriteOptions{
			Data:           []byte("foo"),
			StorageClasses: trial.putClasses,
		})
		if trial.success {
			c.Check(err, check.IsNil)
		} else {
			c.Check(err, check.NotNil)
		}
		c.Check(len(st.handled) >= trial.minRequests, check.Equals, true, check.Commentf("len(st.handled)==%d, trial.minRequests==%d", len(st.handled), trial.minRequests))
		c.Check(len(st.handled) <= trial.maxRequests, check.Equals, true, check.Commentf("len(st.handled)==%d, trial.maxRequests==%d", len(st.handled), trial.maxRequests))
		if !trial.success && trial.replicas == 1 && c.Check(len(st.requests) >= 2, check.Equals, true) {
			// Max concurrency should be 1. First request
			// should have succeeded for class1. Second
			// request should only ask for class404.
			c.Check(st.requests[1].Header.Get("X-Keep-Storage-Classes"), check.Equals, "class404")
		}
	}
}

type FailHandler struct {
	handled chan string
}

func (fh FailHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(500)
	fh.handled <- fmt.Sprintf("http://%s", req.Host)
}

type FailThenSucceedHandler struct {
	handled        chan string
	count          int
	successhandler http.Handler
	reqIDs         []string
}

func (fh *FailThenSucceedHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	fh.reqIDs = append(fh.reqIDs, req.Header.Get("X-Request-Id"))
	if fh.count == 0 {
		resp.WriteHeader(500)
		fh.count++
		fh.handled <- fmt.Sprintf("http://%s", req.Host)
	} else {
		fh.successhandler.ServeHTTP(resp, req)
	}
}

type Error404Handler struct {
	handled chan string
}

func (fh Error404Handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(404)
	fh.handled <- fmt.Sprintf("http://%s", req.Host)
}

func (s *StandaloneSuite) TestFailedUploadToStubKeepServer(c *C) {
	st := FailHandler{
		make(chan string)}

	hash := "acbd18db4cc2f85cedef654fccc4a4d8"

	UploadToStubHelper(c, st,
		func(kc *KeepClient, url string, reader io.ReadCloser,
			writer io.WriteCloser, uploadStatusChan chan uploadStatus) {

			go kc.uploadToKeepServer(url, hash, nil, reader, uploadStatusChan, 3, kc.getRequestID())

			writer.Write([]byte("foo"))
			writer.Close()

			<-st.handled

			status := <-uploadStatusChan
			c.Check(status.url, Equals, fmt.Sprintf("%s/%s", url, hash))
			c.Check(status.statusCode, Equals, 500)
		})
}

type KeepServer struct {
	listener net.Listener
	url      string
}

func RunSomeFakeKeepServers(st http.Handler, n int) (ks []KeepServer) {
	ks = make([]KeepServer, n)

	for i := 0; i < n; i++ {
		ks[i] = RunFakeKeepServer(st)
	}

	return ks
}

func (s *StandaloneSuite) TestPutB(c *C) {
	hash := Md5String("foo")

	st := &StubPutHandler{
		c:                    c,
		expectPath:           hash,
		expectAPIToken:       "abc123",
		expectBody:           "foo",
		expectStorageClass:   "",
		returnStorageClasses: "",
		handled:              make(chan string, 5),
	}

	arv, _ := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks := RunSomeFakeKeepServers(st, 5)

	for i, k := range ks {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	kc.PutB([]byte("foo"))

	shuff := NewRootSorter(
		kc.LocalRoots(), Md5String("foo")).GetSortedRoots()

	s1 := <-st.handled
	s2 := <-st.handled
	c.Check((s1 == shuff[0] && s2 == shuff[1]) ||
		(s1 == shuff[1] && s2 == shuff[0]),
		Equals,
		true)
}

func (s *StandaloneSuite) TestPutHR(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := &StubPutHandler{
		c:                    c,
		expectPath:           hash,
		expectAPIToken:       "abc123",
		expectBody:           "foo",
		expectStorageClass:   "",
		returnStorageClasses: "",
		handled:              make(chan string, 5),
	}

	arv, _ := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks := RunSomeFakeKeepServers(st, 5)

	for i, k := range ks {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	reader, writer := io.Pipe()

	go func() {
		writer.Write([]byte("foo"))
		writer.Close()
	}()

	kc.PutHR(hash, reader, 3)

	shuff := NewRootSorter(kc.LocalRoots(), hash).GetSortedRoots()

	s1 := <-st.handled
	s2 := <-st.handled

	c.Check((s1 == shuff[0] && s2 == shuff[1]) ||
		(s1 == shuff[1] && s2 == shuff[0]),
		Equals,
		true)
}

func (s *StandaloneSuite) TestPutWithFail(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := &StubPutHandler{
		c:                    c,
		expectPath:           hash,
		expectAPIToken:       "abc123",
		expectBody:           "foo",
		expectStorageClass:   "",
		returnStorageClasses: "",
		handled:              make(chan string, 4),
	}

	fh := FailHandler{
		make(chan string, 1)}

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 4)
	ks2 := RunSomeFakeKeepServers(fh, 1)

	for i, k := range ks1 {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	shuff := NewRootSorter(
		kc.LocalRoots(), Md5String("foo")).GetSortedRoots()
	c.Logf("%+v", shuff)

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
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := &StubPutHandler{
		c:                    c,
		expectPath:           hash,
		expectAPIToken:       "abc123",
		expectBody:           "foo",
		expectStorageClass:   "",
		returnStorageClasses: "",
		handled:              make(chan string, 1),
	}

	fh := FailHandler{
		make(chan string, 4)}

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 2
	kc.Retries = 0
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 1)
	ks2 := RunSomeFakeKeepServers(fh, 4)

	for i, k := range ks1 {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	_, replicas, err := kc.PutB([]byte("foo"))

	c.Check(err, FitsTypeOf, InsufficientReplicasError{})
	c.Check(replicas, Equals, 1)
	c.Check(<-st.handled, Equals, ks1[0].url)
}

type StubGetHandler struct {
	c              *C
	expectPath     string
	expectAPIToken string
	httpStatus     int
	body           []byte
}

func (sgh StubGetHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	sgh.c.Check(req.URL.Path, Equals, "/"+sgh.expectPath)
	sgh.c.Check(req.Header.Get("Authorization"), Equals, fmt.Sprintf("OAuth2 %s", sgh.expectAPIToken))
	resp.WriteHeader(sgh.httpStatus)
	resp.Header().Set("Content-Length", fmt.Sprintf("%d", len(sgh.body)))
	resp.Write(sgh.body)
}

func (s *StandaloneSuite) TestGet(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubGetHandler{
		c,
		hash,
		"abc123",
		http.StatusOK,
		[]byte("foo")}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, n, url2, err := kc.Get(hash)
	defer r.Close()
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(url2, Equals, fmt.Sprintf("%s/%s", ks.url, hash))

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(content, DeepEquals, []byte("foo"))
}

func (s *StandaloneSuite) TestGet404(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := Error404Handler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, n, url2, err := kc.Get(hash)
	c.Check(err, Equals, BlockNotFound)
	c.Check(n, Equals, int64(0))
	c.Check(url2, Equals, "")
	c.Check(r, Equals, nil)
}

func (s *StandaloneSuite) TestGetEmptyBlock(c *C) {
	st := Error404Handler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, n, url2, err := kc.Get("d41d8cd98f00b204e9800998ecf8427e+0")
	c.Check(err, IsNil)
	c.Check(n, Equals, int64(0))
	c.Check(url2, Equals, "")
	c.Assert(r, NotNil)
	buf, err := ioutil.ReadAll(r)
	c.Check(err, IsNil)
	c.Check(buf, DeepEquals, []byte{})
}

func (s *StandaloneSuite) TestGetFail(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := FailHandler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)
	kc.Retries = 0

	r, n, url2, err := kc.Get(hash)
	errNotFound, _ := err.(*ErrNotFound)
	c.Check(errNotFound, NotNil)
	c.Check(strings.Contains(errNotFound.Error(), "HTTP 500"), Equals, true)
	c.Check(errNotFound.Temporary(), Equals, true)
	c.Check(n, Equals, int64(0))
	c.Check(url2, Equals, "")
	c.Check(r, Equals, nil)
}

func (s *StandaloneSuite) TestGetFailRetry(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := &FailThenSucceedHandler{
		handled: make(chan string, 1),
		successhandler: StubGetHandler{
			c,
			hash,
			"abc123",
			http.StatusOK,
			[]byte("foo")}}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, n, url2, err := kc.Get(hash)
	defer r.Close()
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(url2, Equals, fmt.Sprintf("%s/%s", ks.url, hash))

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(content, DeepEquals, []byte("foo"))

	c.Logf("%q", st.reqIDs)
	c.Assert(len(st.reqIDs) > 1, Equals, true)
	for _, reqid := range st.reqIDs {
		c.Check(reqid, Not(Equals), "")
		c.Check(reqid, Equals, st.reqIDs[0])
	}
}

func (s *StandaloneSuite) TestGetNetError(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": "http://localhost:62222"}, nil, nil)

	r, n, url2, err := kc.Get(hash)
	errNotFound, _ := err.(*ErrNotFound)
	c.Check(errNotFound, NotNil)
	c.Check(strings.Contains(errNotFound.Error(), "connection refused"), Equals, true)
	c.Check(errNotFound.Temporary(), Equals, true)
	c.Check(n, Equals, int64(0))
	c.Check(url2, Equals, "")
	c.Check(r, Equals, nil)
}

func (s *StandaloneSuite) TestGetWithServiceHint(c *C) {
	uuid := "zzzzz-bi6l4-123451234512345"
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	// This one shouldn't be used:
	ks0 := RunFakeKeepServer(StubGetHandler{
		c,
		"error if used",
		"abc123",
		http.StatusOK,
		[]byte("foo")})
	defer ks0.listener.Close()
	// This one should be used:
	ks := RunFakeKeepServer(StubGetHandler{
		c,
		hash + "+K@" + uuid,
		"abc123",
		http.StatusOK,
		[]byte("foo")})
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(
		map[string]string{"x": ks0.url},
		nil,
		map[string]string{uuid: ks.url})

	r, n, uri, err := kc.Get(hash + "+K@" + uuid)
	defer r.Close()
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(uri, Equals, fmt.Sprintf("%s/%s", ks.url, hash+"+K@"+uuid))

	content, err := ioutil.ReadAll(r)
	c.Check(err, Equals, nil)
	c.Check(content, DeepEquals, []byte("foo"))
}

// Use a service hint to fetch from a local disk service, overriding
// rendezvous probe order.
func (s *StandaloneSuite) TestGetWithLocalServiceHint(c *C) {
	uuid := "zzzzz-bi6l4-zzzzzzzzzzzzzzz"
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	// This one shouldn't be used, although it appears first in
	// rendezvous probe order:
	ks0 := RunFakeKeepServer(StubGetHandler{
		c,
		"error if used",
		"abc123",
		http.StatusOK,
		[]byte("foo")})
	defer ks0.listener.Close()
	// This one should be used:
	ks := RunFakeKeepServer(StubGetHandler{
		c,
		hash + "+K@" + uuid,
		"abc123",
		http.StatusOK,
		[]byte("foo")})
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(
		map[string]string{
			"zzzzz-bi6l4-yyyyyyyyyyyyyyy": ks0.url,
			"zzzzz-bi6l4-xxxxxxxxxxxxxxx": ks0.url,
			"zzzzz-bi6l4-wwwwwwwwwwwwwww": ks0.url,
			uuid:                          ks.url},
		nil,
		map[string]string{
			"zzzzz-bi6l4-yyyyyyyyyyyyyyy": ks0.url,
			"zzzzz-bi6l4-xxxxxxxxxxxxxxx": ks0.url,
			"zzzzz-bi6l4-wwwwwwwwwwwwwww": ks0.url,
			uuid:                          ks.url},
	)

	r, n, uri, err := kc.Get(hash + "+K@" + uuid)
	defer r.Close()
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(uri, Equals, fmt.Sprintf("%s/%s", ks.url, hash+"+K@"+uuid))

	content, err := ioutil.ReadAll(r)
	c.Check(err, Equals, nil)
	c.Check(content, DeepEquals, []byte("foo"))
}

func (s *StandaloneSuite) TestGetWithServiceHintFailoverToLocals(c *C) {
	uuid := "zzzzz-bi6l4-123451234512345"
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	ksLocal := RunFakeKeepServer(StubGetHandler{
		c,
		hash + "+K@" + uuid,
		"abc123",
		http.StatusOK,
		[]byte("foo")})
	defer ksLocal.listener.Close()
	ksGateway := RunFakeKeepServer(StubGetHandler{
		c,
		hash + "+K@" + uuid,
		"abc123",
		http.StatusInternalServerError,
		[]byte("Error")})
	defer ksGateway.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(
		map[string]string{"zzzzz-bi6l4-keepdisk0000000": ksLocal.url},
		nil,
		map[string]string{uuid: ksGateway.url})

	r, n, uri, err := kc.Get(hash + "+K@" + uuid)
	c.Assert(err, Equals, nil)
	defer r.Close()
	c.Check(n, Equals, int64(3))
	c.Check(uri, Equals, fmt.Sprintf("%s/%s", ksLocal.url, hash+"+K@"+uuid))

	content, err := ioutil.ReadAll(r)
	c.Check(err, Equals, nil)
	c.Check(content, DeepEquals, []byte("foo"))
}

type BarHandler struct {
	handled chan string
}

func (h BarHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte("bar"))
	h.handled <- fmt.Sprintf("http://%s", req.Host)
}

func (s *StandaloneSuite) TestChecksum(c *C) {
	foohash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	barhash := fmt.Sprintf("%x", md5.Sum([]byte("bar")))

	st := BarHandler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, n, _, err := kc.Get(barhash)
	c.Check(err, IsNil)
	_, err = ioutil.ReadAll(r)
	c.Check(n, Equals, int64(3))
	c.Check(err, Equals, nil)

	<-st.handled

	r, n, _, err = kc.Get(foohash)
	c.Check(err, IsNil)
	_, err = ioutil.ReadAll(r)
	c.Check(n, Equals, int64(3))
	c.Check(err, Equals, BadChecksum)

	<-st.handled
}

func (s *StandaloneSuite) TestGetWithFailures(c *C) {
	content := []byte("waz")
	hash := fmt.Sprintf("%x", md5.Sum(content))

	fh := Error404Handler{
		make(chan string, 4)}

	st := StubGetHandler{
		c,
		hash,
		"abc123",
		http.StatusOK,
		content}

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 1)
	ks2 := RunSomeFakeKeepServers(fh, 4)

	for i, k := range ks1 {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}
	for i, k := range ks2 {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i+len(ks1))] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)
	kc.Retries = 0

	// This test works only if one of the failing services is
	// attempted before the succeeding service. Otherwise,
	// <-fh.handled below will just hang! (Probe order depends on
	// the choice of block content "waz" and the UUIDs of the fake
	// servers, so we just tried different strings until we found
	// an example that passes this Assert.)
	c.Assert(NewRootSorter(localRoots, hash).GetSortedRoots()[0], Not(Equals), ks1[0].url)

	r, n, url2, err := kc.Get(hash)

	<-fh.handled
	c.Check(err, Equals, nil)
	c.Check(n, Equals, int64(3))
	c.Check(url2, Equals, fmt.Sprintf("%s/%s", ks1[0].url, hash))

	readContent, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(readContent, DeepEquals, content)
}

func (s *ServerRequiredSuite) TestPutGetHead(c *C) {
	content := []byte("TestPutGetHead")

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, err := MakeKeepClient(arv)
	c.Assert(err, Equals, nil)

	hash := fmt.Sprintf("%x", md5.Sum(content))

	{
		n, _, err := kc.Ask(hash)
		c.Check(err, Equals, BlockNotFound)
		c.Check(n, Equals, int64(0))
	}
	{
		hash2, replicas, err := kc.PutB(content)
		c.Check(hash2, Matches, fmt.Sprintf(`%s\+%d\b.*`, hash, len(content)))
		c.Check(replicas, Equals, 2)
		c.Check(err, Equals, nil)
	}
	{
		r, n, url2, err := kc.Get(hash)
		c.Check(err, Equals, nil)
		c.Check(n, Equals, int64(len(content)))
		c.Check(url2, Matches, fmt.Sprintf("http://localhost:\\d+/%s", hash))

		readContent, err2 := ioutil.ReadAll(r)
		c.Check(err2, Equals, nil)
		c.Check(readContent, DeepEquals, content)
	}
	{
		n, url2, err := kc.Ask(hash)
		c.Check(err, Equals, nil)
		c.Check(n, Equals, int64(len(content)))
		c.Check(url2, Matches, fmt.Sprintf("http://localhost:\\d+/%s", hash))
	}
	{
		loc, err := kc.LocalLocator(hash)
		c.Check(err, Equals, nil)
		c.Assert(len(loc) >= 32, Equals, true)
		c.Check(loc[:32], Equals, hash[:32])
	}
	{
		content := []byte("the perth county conspiracy")
		loc, err := kc.LocalLocator(fmt.Sprintf("%x+%d+Rzaaaa-abcde@12345", md5.Sum(content), len(content)))
		c.Check(loc, Equals, "")
		c.Check(err, ErrorMatches, `.*HEAD .*\+R.*`)
		c.Check(err, ErrorMatches, `.*HTTP 400.*`)
	}
}

type StubProxyHandler struct {
	handled chan string
}

func (h StubProxyHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("X-Keep-Replicas-Stored", "2")
	h.handled <- fmt.Sprintf("http://%s", req.Host)
}

func (s *StandaloneSuite) TestPutProxy(c *C) {
	st := StubProxyHandler{make(chan string, 1)}

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 1)

	for i, k := range ks1 {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	_, replicas, err := kc.PutB([]byte("foo"))
	<-st.handled

	c.Check(err, Equals, nil)
	c.Check(replicas, Equals, 2)
}

func (s *StandaloneSuite) TestPutProxyInsufficientReplicas(c *C) {
	st := StubProxyHandler{make(chan string, 1)}

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 3
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks1 := RunSomeFakeKeepServers(st, 1)

	for i, k := range ks1 {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}
	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	_, replicas, err := kc.PutB([]byte("foo"))
	<-st.handled

	c.Check(err, FitsTypeOf, InsufficientReplicasError{})
	c.Check(replicas, Equals, 2)
}

func (s *StandaloneSuite) TestMakeLocator(c *C) {
	l, err := MakeLocator("91f372a266fe2bf2823cb8ec7fda31ce+3+Aabcde@12345678")
	c.Check(err, Equals, nil)
	c.Check(l.Hash, Equals, "91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(l.Size, Equals, 3)
	c.Check(l.Hints, DeepEquals, []string{"3", "Aabcde@12345678"})
}

func (s *StandaloneSuite) TestMakeLocatorNoHints(c *C) {
	l, err := MakeLocator("91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(err, Equals, nil)
	c.Check(l.Hash, Equals, "91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(l.Size, Equals, -1)
	c.Check(l.Hints, DeepEquals, []string{})
}

func (s *StandaloneSuite) TestMakeLocatorNoSizeHint(c *C) {
	l, err := MakeLocator("91f372a266fe2bf2823cb8ec7fda31ce+Aabcde@12345678")
	c.Check(err, Equals, nil)
	c.Check(l.Hash, Equals, "91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(l.Size, Equals, -1)
	c.Check(l.Hints, DeepEquals, []string{"Aabcde@12345678"})
}

func (s *StandaloneSuite) TestMakeLocatorPreservesUnrecognizedHints(c *C) {
	str := "91f372a266fe2bf2823cb8ec7fda31ce+3+Unknown+Kzzzzz+Afoobar"
	l, err := MakeLocator(str)
	c.Check(err, Equals, nil)
	c.Check(l.Hash, Equals, "91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(l.Size, Equals, 3)
	c.Check(l.Hints, DeepEquals, []string{"3", "Unknown", "Kzzzzz", "Afoobar"})
	c.Check(l.String(), Equals, str)
}

func (s *StandaloneSuite) TestMakeLocatorInvalidInput(c *C) {
	_, err := MakeLocator("91f372a266fe2bf2823cb8ec7fda31c")
	c.Check(err, Equals, InvalidLocatorError)
}

func (s *StandaloneSuite) TestPutBWant2ReplicasWithOnlyOneWritableLocalRoot(c *C) {
	hash := Md5String("foo")

	st := &StubPutHandler{
		c:                    c,
		expectPath:           hash,
		expectAPIToken:       "abc123",
		expectBody:           "foo",
		expectStorageClass:   "",
		returnStorageClasses: "",
		handled:              make(chan string, 5),
	}

	arv, _ := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks := RunSomeFakeKeepServers(st, 5)

	for i, k := range ks {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		if i == 0 {
			writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		}
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	_, replicas, err := kc.PutB([]byte("foo"))

	c.Check(err, FitsTypeOf, InsufficientReplicasError{})
	c.Check(replicas, Equals, 1)

	c.Check(<-st.handled, Equals, localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", 0)])
}

func (s *StandaloneSuite) TestPutBWithNoWritableLocalRoots(c *C) {
	hash := Md5String("foo")

	st := &StubPutHandler{
		c:                    c,
		expectPath:           hash,
		expectAPIToken:       "abc123",
		expectBody:           "foo",
		expectStorageClass:   "",
		returnStorageClasses: "",
		handled:              make(chan string, 5),
	}

	arv, _ := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks := RunSomeFakeKeepServers(st, 5)

	for i, k := range ks {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	_, replicas, err := kc.PutB([]byte("foo"))

	c.Check(err, FitsTypeOf, InsufficientReplicasError{})
	c.Check(replicas, Equals, 0)
}

type StubGetIndexHandler struct {
	c              *C
	expectPath     string
	expectAPIToken string
	httpStatus     int
	body           []byte
}

func (h StubGetIndexHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	h.c.Check(req.URL.Path, Equals, h.expectPath)
	h.c.Check(req.Header.Get("Authorization"), Equals, fmt.Sprintf("OAuth2 %s", h.expectAPIToken))
	resp.WriteHeader(h.httpStatus)
	resp.Header().Set("Content-Length", fmt.Sprintf("%d", len(h.body)))
	resp.Write(h.body)
}

func (s *StandaloneSuite) TestGetIndexWithNoPrefix(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubGetIndexHandler{
		c,
		"/index",
		"abc123",
		http.StatusOK,
		[]byte(hash + "+3 1443559274\n\n")}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)
	kc, err := MakeKeepClient(arv)
	c.Assert(err, IsNil)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, err := kc.GetIndex("x", "")
	c.Check(err, IsNil)

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(content, DeepEquals, st.body[0:len(st.body)-1])
}

func (s *StandaloneSuite) TestGetIndexWithPrefix(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubGetIndexHandler{
		c,
		"/index/" + hash[0:3],
		"abc123",
		http.StatusOK,
		[]byte(hash + "+3 1443559274\n\n")}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, err := kc.GetIndex("x", hash[0:3])
	c.Assert(err, Equals, nil)

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(content, DeepEquals, st.body[0:len(st.body)-1])
}

func (s *StandaloneSuite) TestGetIndexIncomplete(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubGetIndexHandler{
		c,
		"/index/" + hash[0:3],
		"abc123",
		http.StatusOK,
		[]byte(hash)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	_, err = kc.GetIndex("x", hash[0:3])
	c.Check(err, Equals, ErrIncompleteIndex)
}

func (s *StandaloneSuite) TestGetIndexWithNoSuchServer(c *C) {
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	st := StubGetIndexHandler{
		c,
		"/index/" + hash[0:3],
		"abc123",
		http.StatusOK,
		[]byte(hash)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	_, err = kc.GetIndex("y", hash[0:3])
	c.Check(err, Equals, ErrNoSuchKeepServer)
}

func (s *StandaloneSuite) TestGetIndexWithNoSuchPrefix(c *C) {
	st := StubGetIndexHandler{
		c,
		"/index/abcd",
		"abc123",
		http.StatusOK,
		[]byte("\n")}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, err := kc.GetIndex("x", "abcd")
	c.Check(err, Equals, nil)

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, Equals, nil)
	c.Check(content, DeepEquals, st.body[0:len(st.body)-1])
}

func (s *StandaloneSuite) TestPutBRetry(c *C) {
	st := &FailThenSucceedHandler{
		handled: make(chan string, 1),
		successhandler: &StubPutHandler{
			c:                    c,
			expectPath:           Md5String("foo"),
			expectAPIToken:       "abc123",
			expectBody:           "foo",
			expectStorageClass:   "",
			returnStorageClasses: "",
			handled:              make(chan string, 5),
		},
	}

	arv, _ := arvadosclient.MakeArvadosClient()
	kc, _ := MakeKeepClient(arv)

	kc.Want_replicas = 2
	arv.ApiToken = "abc123"
	localRoots := make(map[string]string)
	writableLocalRoots := make(map[string]string)

	ks := RunSomeFakeKeepServers(st, 2)

	for i, k := range ks {
		localRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		writableLocalRoots[fmt.Sprintf("zzzzz-bi6l4-fakefakefake%03d", i)] = k.url
		defer k.listener.Close()
	}

	kc.SetServiceRoots(localRoots, writableLocalRoots, nil)

	hash, replicas, err := kc.PutB([]byte("foo"))

	c.Check(err, Equals, nil)
	c.Check(hash, Equals, "")
	c.Check(replicas, Equals, 2)
}

func (s *ServerRequiredSuite) TestMakeKeepClientWithNonDiskTypeService(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)

	// Add an additional "testblobstore" keepservice
	blobKeepService := make(arvadosclient.Dict)
	err = arv.Create("keep_services",
		arvadosclient.Dict{"keep_service": arvadosclient.Dict{
			"service_host": "localhost",
			"service_port": "21321",
			"service_type": "testblobstore"}},
		&blobKeepService)
	c.Assert(err, Equals, nil)
	defer func() { arv.Delete("keep_services", blobKeepService["uuid"].(string), nil, nil) }()
	RefreshServiceDiscovery()

	// Make a keepclient and ensure that the testblobstore is included
	kc, err := MakeKeepClient(arv)
	c.Assert(err, Equals, nil)

	// verify kc.LocalRoots
	c.Check(len(kc.LocalRoots()), Equals, 3)
	for _, root := range kc.LocalRoots() {
		c.Check(root, Matches, "http://localhost:\\d+")
	}
	c.Assert(kc.LocalRoots()[blobKeepService["uuid"].(string)], Not(Equals), "")

	// verify kc.GatewayRoots
	c.Check(len(kc.GatewayRoots()), Equals, 3)
	for _, root := range kc.GatewayRoots() {
		c.Check(root, Matches, "http://localhost:\\d+")
	}
	c.Assert(kc.GatewayRoots()[blobKeepService["uuid"].(string)], Not(Equals), "")

	// verify kc.WritableLocalRoots
	c.Check(len(kc.WritableLocalRoots()), Equals, 3)
	for _, root := range kc.WritableLocalRoots() {
		c.Check(root, Matches, "http://localhost:\\d+")
	}
	c.Assert(kc.WritableLocalRoots()[blobKeepService["uuid"].(string)], Not(Equals), "")

	c.Assert(kc.replicasPerService, Equals, 0)
	c.Assert(kc.foundNonDiskSvc, Equals, true)
	c.Assert(kc.httpClient().(*http.Client).Timeout, Equals, 300*time.Second)
}
