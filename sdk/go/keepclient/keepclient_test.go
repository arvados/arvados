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
	"sync/atomic"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	DefaultRetryDelay = 50 * time.Millisecond
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&ServerRequiredSuite{})
var _ = Suite(&StandaloneSuite{})

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

// Standalone tests
type StandaloneSuite struct {
	origDefaultRetryDelay time.Duration
	origMinimumRetryDelay time.Duration
}

var origHOME = os.Getenv("HOME")

func (s *StandaloneSuite) SetUpTest(c *C) {
	RefreshServiceDiscovery()
	// Prevent cache state from leaking between test cases
	os.Setenv("HOME", c.MkDir())
	s.origDefaultRetryDelay = DefaultRetryDelay
	s.origMinimumRetryDelay = MinimumRetryDelay
}

func (s *StandaloneSuite) TearDownTest(c *C) {
	os.Setenv("HOME", origHOME)
	DefaultRetryDelay = s.origDefaultRetryDelay
	MinimumRetryDelay = s.origMinimumRetryDelay
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
	os.Setenv("HOME", origHOME)
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	RefreshServiceDiscovery()
	// Prevent cache state from leaking between test cases
	os.Setenv("HOME", c.MkDir())
}

func (s *ServerRequiredSuite) TestMakeKeepClient(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	kc, err := MakeKeepClient(arv)

	c.Assert(err, IsNil)
	c.Check(len(kc.LocalRoots()), Equals, 2)
	for _, root := range kc.LocalRoots() {
		c.Check(root, Matches, "http://localhost:\\d+")
	}
}

func (s *ServerRequiredSuite) TestDefaultStorageClasses(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	cc, err := arv.ClusterConfig("StorageClasses")
	c.Assert(err, IsNil)
	c.Assert(cc, NotNil)
	c.Assert(cc.(map[string]interface{})["default"], NotNil)

	kc := New(arv)
	c.Assert(kc.DefaultStorageClasses, DeepEquals, []string{"default"})
}

func (s *ServerRequiredSuite) TestDefaultReplications(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

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
	sph.c.Check(err, IsNil)
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
		panic("Could not listen on any port")
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

func (s *StandaloneSuite) TestPutWithoutStorageClassesClusterSupport(c *C) {
	nServers := 5
	for _, trial := range []struct {
		replicas      int
		clientClasses []string
		putClasses    []string
		minRequests   int
		maxRequests   int
		success       bool
	}{
		// Talking to an older cluster (no default storage classes exported
		// config) and no other additional storage classes requirements.
		{1, nil, nil, 1, 1, true},
		{2, nil, nil, 2, 2, true},
		{3, nil, nil, 3, 3, true},
		{nServers*2 + 1, nil, nil, nServers, nServers, false},

		{1, []string{"class1"}, nil, 1, 1, true},
		{2, []string{"class1"}, nil, 2, 2, true},
		{3, []string{"class1"}, nil, 3, 3, true},
		{1, []string{"class1", "class2"}, nil, 1, 1, true},
		{nServers*2 + 1, []string{"class1"}, nil, nServers, nServers, false},

		{1, nil, []string{"class1"}, 1, 1, true},
		{2, nil, []string{"class1"}, 2, 2, true},
		{3, nil, []string{"class1"}, 3, 3, true},
		{1, nil, []string{"class1", "class2"}, 1, 1, true},
		{nServers*2 + 1, nil, []string{"class1"}, nServers, nServers, false},
	} {
		c.Logf("%+v", trial)
		st := &StubPutHandler{
			c:                    c,
			expectPath:           "acbd18db4cc2f85cedef654fccc4a4d8",
			expectAPIToken:       "abc123",
			expectBody:           "foo",
			expectStorageClass:   "*",
			returnStorageClasses: "", // Simulate old cluster without SC keep support
			handled:              make(chan string, 100),
		}
		ks := RunSomeFakeKeepServers(st, nServers)
		arv, _ := arvadosclient.MakeArvadosClient()
		kc, _ := MakeKeepClient(arv)
		kc.Want_replicas = trial.replicas
		kc.StorageClasses = trial.clientClasses
		kc.DefaultStorageClasses = nil // Simulate an old cluster without SC defaults
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
			c.Check(err, IsNil)
		} else {
			c.Check(err, NotNil)
		}
		c.Check(len(st.handled) >= trial.minRequests, Equals, true, Commentf("len(st.handled)==%d, trial.minRequests==%d", len(st.handled), trial.minRequests))
		c.Check(len(st.handled) <= trial.maxRequests, Equals, true, Commentf("len(st.handled)==%d, trial.maxRequests==%d", len(st.handled), trial.maxRequests))
		if trial.clientClasses == nil && trial.putClasses == nil {
			c.Check(st.requests[0].Header.Get("X-Keep-Storage-Classes"), Equals, "")
		}
	}
}

func (s *StandaloneSuite) TestPutWithStorageClasses(c *C) {
	nServers := 5
	for _, trial := range []struct {
		replicas       int
		defaultClasses []string
		clientClasses  []string // clientClasses takes precedence over defaultClasses
		putClasses     []string // putClasses takes precedence over clientClasses
		minRequests    int
		maxRequests    int
		success        bool
	}{
		{1, []string{"class1"}, nil, nil, 1, 1, true},
		{2, []string{"class1"}, nil, nil, 1, 2, true},
		{3, []string{"class1"}, nil, nil, 2, 3, true},
		{1, []string{"class1", "class2"}, nil, nil, 1, 1, true},

		// defaultClasses doesn't matter when any of the others is specified.
		{1, []string{"class1"}, []string{"class1"}, nil, 1, 1, true},
		{2, []string{"class1"}, []string{"class1"}, nil, 1, 2, true},
		{3, []string{"class1"}, []string{"class1"}, nil, 2, 3, true},
		{1, []string{"class1"}, []string{"class1", "class2"}, nil, 1, 1, true},
		{3, []string{"class1"}, nil, []string{"class1"}, 2, 3, true},
		{1, []string{"class1"}, nil, []string{"class1", "class2"}, 1, 1, true},
		{1, []string{"class1"}, []string{"class404"}, []string{"class1", "class2"}, 1, 1, true},
		{1, []string{"class1"}, []string{"class1"}, []string{"class404", "class2"}, nServers, nServers, false},
		{nServers*2 + 1, []string{}, []string{"class1"}, nil, nServers, nServers, false},
		{1, []string{"class1"}, []string{"class404"}, nil, nServers, nServers, false},
		{1, []string{"class1"}, []string{"class1", "class404"}, nil, nServers, nServers, false},
		{1, []string{"class1"}, nil, []string{"class1", "class404"}, nServers, nServers, false},
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
		kc.DefaultStorageClasses = trial.defaultClasses
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
			c.Check(err, IsNil)
		} else {
			c.Check(err, NotNil)
		}
		c.Check(len(st.handled) >= trial.minRequests, Equals, true, Commentf("len(st.handled)==%d, trial.minRequests==%d", len(st.handled), trial.minRequests))
		c.Check(len(st.handled) <= trial.maxRequests, Equals, true, Commentf("len(st.handled)==%d, trial.maxRequests==%d", len(st.handled), trial.maxRequests))
		if !trial.success && trial.replicas == 1 && c.Check(len(st.requests) >= 2, Equals, true) {
			// Max concurrency should be 1. First request
			// should have succeeded for class1. Second
			// request should only ask for class404.
			c.Check(st.requests[1].Header.Get("X-Keep-Storage-Classes"), Equals, "class404")
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
	morefails      int // fail 1 + this many times before succeeding
	handled        chan string
	count          atomic.Int64
	successhandler http.Handler
	reqIDs         []string
}

func (fh *FailThenSucceedHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	fh.reqIDs = append(fh.reqIDs, req.Header.Get("X-Request-Id"))
	if int(fh.count.Add(1)) <= fh.morefails+1 {
		resp.WriteHeader(500)
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
		expectStorageClass:   "default",
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
		expectStorageClass:   "default",
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

	kc.PutHR(hash, bytes.NewBuffer([]byte("foo")), 3)

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
		expectStorageClass:   "default",
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

	c.Check(err, IsNil)
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
		expectStorageClass:   "default",
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
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

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

	r, n, _, err := kc.Get(hash)
	c.Assert(err, IsNil)
	c.Check(n, Equals, int64(3))

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, IsNil)
	c.Check(content, DeepEquals, []byte("foo"))
	c.Check(r.Close(), IsNil)
}

func (s *StandaloneSuite) TestGet404(c *C) {
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

	st := Error404Handler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, n, _, err := kc.Get(hash)
	c.Check(err, Equals, BlockNotFound)
	c.Check(n, Equals, int64(0))
	c.Check(r, IsNil)
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

	r, n, _, err := kc.Get("d41d8cd98f00b204e9800998ecf8427e+0")
	c.Check(err, IsNil)
	c.Check(n, Equals, int64(0))
	c.Assert(r, NotNil)
	buf, err := ioutil.ReadAll(r)
	c.Check(err, IsNil)
	c.Check(buf, DeepEquals, []byte{})
	c.Check(r.Close(), IsNil)
}

func (s *StandaloneSuite) TestGetFail(c *C) {
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

	st := FailHandler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)
	kc.Retries = 0

	r, n, _, err := kc.Get(hash)
	errNotFound, _ := err.(*ErrNotFound)
	if c.Check(errNotFound, NotNil) {
		c.Check(strings.Contains(errNotFound.Error(), "HTTP 500"), Equals, true)
		c.Check(errNotFound.Temporary(), Equals, true)
	}
	c.Check(n, Equals, int64(0))
	c.Check(r, IsNil)
}

func (s *StandaloneSuite) TestGetFailRetry(c *C) {
	defer func(origDefault, origMinimum time.Duration) {
		DefaultRetryDelay = origDefault
		MinimumRetryDelay = origMinimum
	}(DefaultRetryDelay, MinimumRetryDelay)
	DefaultRetryDelay = time.Second / 8
	MinimumRetryDelay = time.Millisecond

	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

	for _, delay := range []time.Duration{0, time.Nanosecond, time.Second / 8, time.Second / 16} {
		c.Logf("=== initial delay %v", delay)

		st := &FailThenSucceedHandler{
			morefails: 2,
			handled:   make(chan string, 4),
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
		kc.Retries = 3
		kc.RetryDelay = delay
		kc.DiskCacheSize = DiskCacheDisabled

		t0 := time.Now()
		r, n, _, err := kc.Get(hash)
		c.Assert(err, IsNil)
		c.Check(n, Equals, int64(3))
		elapsed := time.Since(t0)

		nonsleeptime := time.Second / 10
		expect := kc.RetryDelay
		if expect == 0 {
			expect = DefaultRetryDelay
		}
		min := MinimumRetryDelay * 3
		max := expect + expect*2 + expect*2*2 + nonsleeptime
		c.Check(elapsed >= min, Equals, true, Commentf("elapsed %v / expect min %v", elapsed, min))
		c.Check(elapsed <= max, Equals, true, Commentf("elapsed %v / expect max %v", elapsed, max))

		content, err := ioutil.ReadAll(r)
		c.Check(err, IsNil)
		c.Check(content, DeepEquals, []byte("foo"))
		c.Check(r.Close(), IsNil)

		c.Logf("%q", st.reqIDs)
		if c.Check(st.reqIDs, Not(HasLen), 0) {
			for _, reqid := range st.reqIDs {
				c.Check(reqid, Not(Equals), "")
				c.Check(reqid, Equals, st.reqIDs[0])
			}
		}
	}
}

func (s *StandaloneSuite) TestGetNetError(c *C) {
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": "http://localhost:62222"}, nil, nil)

	r, n, _, err := kc.Get(hash)
	errNotFound, _ := err.(*ErrNotFound)
	if c.Check(errNotFound, NotNil) {
		c.Check(strings.Contains(errNotFound.Error(), "connection refused"), Equals, true)
		c.Check(errNotFound.Temporary(), Equals, true)
	}
	c.Check(n, Equals, int64(0))
	c.Check(r, IsNil)
}

func (s *StandaloneSuite) TestGetWithServiceHint(c *C) {
	uuid := "zzzzz-bi6l4-123451234512345"
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

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

	r, n, _, err := kc.Get(hash + "+K@" + uuid)
	c.Assert(err, IsNil)
	c.Check(n, Equals, int64(3))

	content, err := ioutil.ReadAll(r)
	c.Check(err, IsNil)
	c.Check(content, DeepEquals, []byte("foo"))
	c.Check(r.Close(), IsNil)
}

// Use a service hint to fetch from a local disk service, overriding
// rendezvous probe order.
func (s *StandaloneSuite) TestGetWithLocalServiceHint(c *C) {
	uuid := "zzzzz-bi6l4-zzzzzzzzzzzzzzz"
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

	// This one shouldn't be used, although it appears first in
	// rendezvous probe order:
	ks0 := RunFakeKeepServer(StubGetHandler{
		c,
		"error if used",
		"abc123",
		http.StatusBadGateway,
		nil})
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

	r, n, _, err := kc.Get(hash + "+K@" + uuid)
	c.Assert(err, IsNil)
	c.Check(n, Equals, int64(3))

	content, err := ioutil.ReadAll(r)
	c.Check(err, IsNil)
	c.Check(content, DeepEquals, []byte("foo"))
	c.Check(r.Close(), IsNil)
}

func (s *StandaloneSuite) TestGetWithServiceHintFailoverToLocals(c *C) {
	uuid := "zzzzz-bi6l4-123451234512345"
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

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

	r, n, _, err := kc.Get(hash + "+K@" + uuid)
	c.Assert(err, IsNil)
	c.Check(n, Equals, int64(3))

	content, err := ioutil.ReadAll(r)
	c.Check(err, IsNil)
	c.Check(content, DeepEquals, []byte("foo"))
	c.Check(r.Close(), IsNil)
}

type BarHandler struct {
	handled chan string
}

func (h BarHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.Write([]byte("bar"))
	h.handled <- fmt.Sprintf("http://%s", req.Host)
}

func (s *StandaloneSuite) TestChecksum(c *C) {
	foohash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))
	barhash := fmt.Sprintf("%x+3", md5.Sum([]byte("bar")))

	st := BarHandler{make(chan string, 1)}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, n, _, err := kc.Get(barhash)
	if c.Check(err, IsNil) {
		_, err = ioutil.ReadAll(r)
		c.Check(n, Equals, int64(3))
		c.Check(err, IsNil)
	}

	select {
	case <-st.handled:
	case <-time.After(time.Second):
		c.Fatal("timed out")
	}

	r, n, _, err = kc.Get(foohash)
	if err == nil {
		buf, readerr := ioutil.ReadAll(r)
		c.Logf("%q", buf)
		err = readerr
	}
	c.Check(err, Equals, BadChecksum)

	select {
	case <-st.handled:
	case <-time.After(time.Second):
		c.Fatal("timed out")
	}
}

func (s *StandaloneSuite) TestGetWithFailures(c *C) {
	content := []byte("waz")
	hash := fmt.Sprintf("%x+3", md5.Sum(content))

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

	r, n, _, err := kc.Get(hash)

	select {
	case <-fh.handled:
	case <-time.After(time.Second):
		c.Fatal("timed out")
	}
	c.Assert(err, IsNil)
	c.Check(n, Equals, int64(3))

	readContent, err2 := ioutil.ReadAll(r)
	c.Check(err2, IsNil)
	c.Check(readContent, DeepEquals, content)
	c.Check(r.Close(), IsNil)
}

func (s *ServerRequiredSuite) TestPutGetHead(c *C) {
	content := []byte("TestPutGetHead")

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, err := MakeKeepClient(arv)
	c.Assert(err, IsNil)

	hash := fmt.Sprintf("%x+%d", md5.Sum(content), len(content))

	{
		n, _, err := kc.Ask(hash)
		c.Check(err, Equals, BlockNotFound)
		c.Check(n, Equals, int64(0))
	}
	{
		hash2, replicas, err := kc.PutB(content)
		c.Check(err, IsNil)
		c.Check(hash2, Matches, `\Q`+hash+`\E\b.*`)
		c.Check(replicas, Equals, 2)
	}
	{
		r, n, _, err := kc.Get(hash)
		c.Check(err, IsNil)
		c.Check(n, Equals, int64(len(content)))
		if c.Check(r, NotNil) {
			readContent, err := ioutil.ReadAll(r)
			c.Check(err, IsNil)
			if c.Check(len(readContent), Equals, len(content)) {
				c.Check(readContent, DeepEquals, content)
			}
			c.Check(r.Close(), IsNil)
		}
	}
	{
		n, url2, err := kc.Ask(hash)
		c.Check(err, IsNil)
		c.Check(n, Equals, int64(len(content)))
		c.Check(url2, Matches, "http://localhost:\\d+/\\Q"+hash+"\\E")
	}
	{
		loc, err := kc.LocalLocator(hash)
		c.Check(err, IsNil)
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

	c.Check(err, IsNil)
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
	c.Check(err, IsNil)
	c.Check(l.Hash, Equals, "91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(l.Size, Equals, 3)
	c.Check(l.Hints, DeepEquals, []string{"3", "Aabcde@12345678"})
}

func (s *StandaloneSuite) TestMakeLocatorNoHints(c *C) {
	l, err := MakeLocator("91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(err, IsNil)
	c.Check(l.Hash, Equals, "91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(l.Size, Equals, -1)
	c.Check(l.Hints, DeepEquals, []string{})
}

func (s *StandaloneSuite) TestMakeLocatorNoSizeHint(c *C) {
	l, err := MakeLocator("91f372a266fe2bf2823cb8ec7fda31ce+Aabcde@12345678")
	c.Check(err, IsNil)
	c.Check(l.Hash, Equals, "91f372a266fe2bf2823cb8ec7fda31ce")
	c.Check(l.Size, Equals, -1)
	c.Check(l.Hints, DeepEquals, []string{"Aabcde@12345678"})
}

func (s *StandaloneSuite) TestMakeLocatorPreservesUnrecognizedHints(c *C) {
	str := "91f372a266fe2bf2823cb8ec7fda31ce+3+Unknown+Kzzzzz+Afoobar"
	l, err := MakeLocator(str)
	c.Check(err, IsNil)
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
		expectStorageClass:   "default",
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
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

	st := StubGetIndexHandler{
		c,
		"/index",
		"abc123",
		http.StatusOK,
		[]byte(hash + " 1443559274\n\n")}

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
	c.Check(err2, IsNil)
	c.Check(content, DeepEquals, st.body[0:len(st.body)-1])
}

func (s *StandaloneSuite) TestGetIndexWithPrefix(c *C) {
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

	st := StubGetIndexHandler{
		c,
		"/index/" + hash[0:3],
		"abc123",
		http.StatusOK,
		[]byte(hash + " 1443559274\n\n")}

	ks := RunFakeKeepServer(st)
	defer ks.listener.Close()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Check(err, IsNil)
	kc, _ := MakeKeepClient(arv)
	arv.ApiToken = "abc123"
	kc.SetServiceRoots(map[string]string{"x": ks.url}, nil, nil)

	r, err := kc.GetIndex("x", hash[0:3])
	c.Assert(err, IsNil)

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, IsNil)
	c.Check(content, DeepEquals, st.body[0:len(st.body)-1])
}

func (s *StandaloneSuite) TestGetIndexIncomplete(c *C) {
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

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
	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("foo")))

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
	c.Check(err, IsNil)

	content, err2 := ioutil.ReadAll(r)
	c.Check(err2, IsNil)
	c.Check(content, DeepEquals, st.body[0:len(st.body)-1])
}

func (s *StandaloneSuite) TestPutBRetry(c *C) {
	DefaultRetryDelay = time.Second / 8
	MinimumRetryDelay = time.Millisecond

	for _, delay := range []time.Duration{0, time.Nanosecond, time.Second / 8, time.Second / 16} {
		c.Logf("=== initial delay %v", delay)

		st := &FailThenSucceedHandler{
			morefails: 5, // handler will fail 6x in total, 3 for each server
			handled:   make(chan string, 10),
			successhandler: &StubPutHandler{
				c:                    c,
				expectPath:           Md5String("foo"),
				expectAPIToken:       "abc123",
				expectBody:           "foo",
				expectStorageClass:   "default",
				returnStorageClasses: "",
				handled:              make(chan string, 5),
			},
		}

		arv, _ := arvadosclient.MakeArvadosClient()
		kc, _ := MakeKeepClient(arv)
		kc.Retries = 3
		kc.RetryDelay = delay
		kc.DiskCacheSize = DiskCacheDisabled
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

		t0 := time.Now()
		hash, replicas, err := kc.PutB([]byte("foo"))

		c.Check(err, IsNil)
		c.Check(hash, Equals, "")
		c.Check(replicas, Equals, 2)
		elapsed := time.Since(t0)

		nonsleeptime := time.Second / 10
		expect := kc.RetryDelay
		if expect == 0 {
			expect = DefaultRetryDelay
		}
		min := MinimumRetryDelay * 3
		max := expect + expect*2 + expect*2*2
		max += nonsleeptime
		checkInterval(c, elapsed, min, max)
	}
}

func (s *ServerRequiredSuite) TestMakeKeepClientWithNonDiskTypeService(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, IsNil)

	// Add an additional "testblobstore" keepservice
	blobKeepService := make(arvadosclient.Dict)
	err = arv.Create("keep_services",
		arvadosclient.Dict{"keep_service": arvadosclient.Dict{
			"service_host": "localhost",
			"service_port": "21321",
			"service_type": "testblobstore"}},
		&blobKeepService)
	c.Assert(err, IsNil)
	defer func() { arv.Delete("keep_services", blobKeepService["uuid"].(string), nil, nil) }()
	RefreshServiceDiscovery()

	// Make a keepclient and ensure that the testblobstore is included
	kc, err := MakeKeepClient(arv)
	c.Assert(err, IsNil)

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

func (s *StandaloneSuite) TestDelayCalculator_Default(c *C) {
	MinimumRetryDelay = time.Second / 2
	DefaultRetryDelay = time.Second

	dc := delayCalculator{InitialMaxDelay: 0}
	checkInterval(c, dc.Next(), time.Second/2, time.Second)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*2)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*4)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*8)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*10)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*10)
}

func (s *StandaloneSuite) TestDelayCalculator_SetInitial(c *C) {
	MinimumRetryDelay = time.Second / 2
	DefaultRetryDelay = time.Second

	dc := delayCalculator{InitialMaxDelay: time.Second * 2}
	checkInterval(c, dc.Next(), time.Second/2, time.Second*2)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*4)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*8)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*16)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*20)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*20)
	checkInterval(c, dc.Next(), time.Second/2, time.Second*20)
}

func (s *StandaloneSuite) TestDelayCalculator_EnsureSomeLongDelays(c *C) {
	dc := delayCalculator{InitialMaxDelay: time.Second * 5}
	var d time.Duration
	n := 4000
	for i := 0; i < n; i++ {
		if i < 20 || i%10 == 0 {
			c.Logf("i=%d, delay=%v", i, d)
		}
		if d = dc.Next(); d > dc.InitialMaxDelay*9 {
			return
		}
	}
	c.Errorf("after %d trials, never got a delay more than 90%% of expected max %d; last was %v", n, dc.InitialMaxDelay*10, d)
}

// If InitialMaxDelay is less than MinimumRetryDelay/10, then delay is
// always MinimumRetryDelay.
func (s *StandaloneSuite) TestDelayCalculator_InitialLessThanMinimum(c *C) {
	MinimumRetryDelay = time.Second / 2
	dc := delayCalculator{InitialMaxDelay: time.Millisecond}
	for i := 0; i < 20; i++ {
		c.Check(dc.Next(), Equals, time.Second/2)
	}
}

func checkInterval(c *C, t, min, max time.Duration) {
	c.Check(t >= min, Equals, true, Commentf("got %v which is below expected min %v", t, min))
	c.Check(t <= max, Equals, true, Commentf("got %v which is above expected max %v", t, max))
}
