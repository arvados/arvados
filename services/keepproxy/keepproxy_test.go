// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepproxy

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	log "github.com/sirupsen/logrus"

	"gopkg.in/check.v1"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&ServerRequiredSuite{})

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

// Gocheck boilerplate
var _ = Suite(&ServerRequiredConfigYmlSuite{})

// Tests that require the Keep servers running as defined in config.yml
type ServerRequiredConfigYmlSuite struct{}

// Gocheck boilerplate
var _ = Suite(&NoKeepServerSuite{})

// Test with no keepserver to simulate errors
type NoKeepServerSuite struct{}

var TestProxyUUID = "zzzzz-bi6l4-lrixqc4fxofbmzz"

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	arvadostest.StartKeep(2, false)
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep(2)
}

func (s *ServerRequiredConfigYmlSuite) SetUpSuite(c *C) {
	// config.yml defines 4 keepstores
	arvadostest.StartKeep(4, false)
}

func (s *ServerRequiredConfigYmlSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *ServerRequiredConfigYmlSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep(4)
}

func (s *NoKeepServerSuite) SetUpSuite(c *C) {
	// We need API to have some keep services listed, but the
	// services themselves should be unresponsive.
	arvadostest.StartKeep(2, false)
	arvadostest.StopKeep(2)
}

func (s *NoKeepServerSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

type testServer struct {
	*httpserver.Server
	proxyHandler *proxyHandler
}

func runProxy(c *C, bogusClientToken bool, loadKeepstoresFromConfig bool, kp *arvados.UploadDownloadRolePermissions) (*testServer, *keepclient.KeepClient, *bytes.Buffer) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, Equals, nil)
	cluster, err := cfg.GetCluster("")
	c.Assert(err, Equals, nil)

	if !loadKeepstoresFromConfig {
		// Do not load Keepstore InternalURLs from the config file
		cluster.Services.Keepstore.InternalURLs = make(map[arvados.URL]arvados.ServiceInstance)
	}

	cluster.Services.Keepproxy.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Host: ":0"}: {}}

	if kp != nil {
		cluster.Collections.KeepproxyPermission = *kp
	}

	logbuf := &bytes.Buffer{}
	logger := log.New()
	logger.Out = logbuf
	ctx := ctxlog.Context(context.Background(), logger)

	handler := newHandlerOrErrorHandler(ctx, cluster, cluster.SystemRootToken, nil).(*proxyHandler)
	srv := &testServer{
		Server: &httpserver.Server{
			Server: http.Server{
				BaseContext: func(net.Listener) context.Context { return ctx },
				Handler: httpserver.AddRequestIDs(
					httpserver.LogRequests(handler)),
			},
			Addr: ":",
		},
		proxyHandler: handler,
	}
	err = srv.Start()
	c.Assert(err, IsNil)

	client := arvados.NewClientFromEnv()
	arv, err := arvadosclient.New(client)
	c.Assert(err, IsNil)
	if bogusClientToken {
		arv.ApiToken = "bogus-token"
	}
	kc := keepclient.New(arv)
	sr := map[string]string{
		TestProxyUUID: "http://" + srv.Addr,
	}
	kc.SetServiceRoots(sr, sr, sr)
	kc.Arvados.External = true
	return srv, kc, logbuf
}

func (s *ServerRequiredSuite) TestResponseViaHeader(c *C) {
	srv, _, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	req, err := http.NewRequest("POST",
		"http://"+srv.Addr+"/",
		strings.NewReader("TestViaHeader"))
	c.Assert(err, Equals, nil)
	req.Header.Add("Authorization", "OAuth2 "+arvadostest.ActiveToken)
	resp, err := (&http.Client{}).Do(req)
	c.Assert(err, Equals, nil)
	c.Check(resp.Header.Get("Via"), Equals, "HTTP/1.1 keepproxy")
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	locator, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, Equals, nil)
	resp.Body.Close()

	req, err = http.NewRequest("GET",
		"http://"+srv.Addr+"/"+string(locator),
		nil)
	c.Assert(err, Equals, nil)
	resp, err = (&http.Client{}).Do(req)
	c.Assert(err, Equals, nil)
	c.Check(resp.Header.Get("Via"), Equals, "HTTP/1.1 keepproxy")
	resp.Body.Close()
}

func (s *ServerRequiredSuite) TestLoopDetection(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	sr := map[string]string{
		TestProxyUUID: "http://" + srv.Addr,
	}
	srv.proxyHandler.KeepClient.SetServiceRoots(sr, sr, sr)

	content := []byte("TestLoopDetection")
	_, _, err := kc.PutB(content)
	c.Check(err, ErrorMatches, `.*loop detected.*`)

	hash := fmt.Sprintf("%x", md5.Sum(content))
	_, _, _, err = kc.Get(hash)
	c.Check(err, ErrorMatches, `.*loop detected.*`)
}

func (s *ServerRequiredSuite) TestStorageClassesHeader(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	// Set up fake keepstore to record request headers
	var hdr http.Header
	ts := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			hdr = r.Header
			http.Error(w, "Error", http.StatusInternalServerError)
		}))
	defer ts.Close()

	// Point keepproxy router's keepclient to the fake keepstore
	sr := map[string]string{
		TestProxyUUID: ts.URL,
	}
	srv.proxyHandler.KeepClient.SetServiceRoots(sr, sr, sr)

	// Set up client to ask for storage classes to keepproxy
	kc.StorageClasses = []string{"secure"}
	content := []byte("Very important data")
	_, _, err := kc.PutB(content)
	c.Check(err, NotNil)
	c.Check(hdr.Get("X-Keep-Storage-Classes"), Equals, "secure")
}

func (s *ServerRequiredSuite) TestStorageClassesConfirmedHeader(c *C) {
	srv, _, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	content := []byte("foo")
	hash := fmt.Sprintf("%x", md5.Sum(content))
	client := &http.Client{}

	req, err := http.NewRequest("PUT",
		fmt.Sprintf("http://%s/%s", srv.Addr, hash),
		bytes.NewReader(content))
	c.Assert(err, IsNil)
	req.Header.Set("X-Keep-Storage-Classes", "default")
	req.Header.Set("Authorization", "OAuth2 "+arvadostest.ActiveToken)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)
	c.Assert(resp.Header.Get("X-Keep-Storage-Classes-Confirmed"), Equals, "default=2")
}

func (s *ServerRequiredSuite) TestDesiredReplicas(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	content := []byte("TestDesiredReplicas")
	hash := fmt.Sprintf("%x", md5.Sum(content))

	for _, kc.Want_replicas = range []int{0, 1, 2, 3} {
		locator, rep, err := kc.PutB(content)
		if kc.Want_replicas < 3 {
			c.Check(err, Equals, nil)
			c.Check(rep, Equals, kc.Want_replicas)
			if rep > 0 {
				c.Check(locator, Matches, fmt.Sprintf(`^%s\+%d(\+.+)?$`, hash, len(content)))
			}
		} else {
			c.Check(err, ErrorMatches, ".*503.*")
		}
	}
}

func (s *ServerRequiredSuite) TestPutWrongContentLength(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	content := []byte("TestPutWrongContentLength")
	hash := fmt.Sprintf("%x", md5.Sum(content))

	// If we use http.Client to send these requests to the network
	// server we just started, the Go http library automatically
	// fixes the invalid Content-Length header. In order to test
	// our server behavior, we have to call the handler directly
	// using an httptest.ResponseRecorder.
	rtr, err := newHandler(context.Background(), kc, 10*time.Second, &arvados.Cluster{})
	c.Assert(err, check.IsNil)

	type testcase struct {
		sendLength   string
		expectStatus int
	}

	for _, t := range []testcase{
		{"1", http.StatusBadRequest},
		{"", http.StatusLengthRequired},
		{"-1", http.StatusLengthRequired},
		{"abcdef", http.StatusLengthRequired},
	} {
		req, err := http.NewRequest("PUT",
			fmt.Sprintf("http://%s/%s+%d", srv.Addr, hash, len(content)),
			bytes.NewReader(content))
		c.Assert(err, IsNil)
		req.Header.Set("Content-Length", t.sendLength)
		req.Header.Set("Authorization", "OAuth2 "+arvadostest.ActiveToken)
		req.Header.Set("Content-Type", "application/octet-stream")

		resp := httptest.NewRecorder()
		rtr.ServeHTTP(resp, req)
		c.Check(resp.Code, Equals, t.expectStatus)
	}
}

func (s *ServerRequiredSuite) TestManyFailedPuts(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()
	srv.proxyHandler.timeout = time.Nanosecond

	buf := make([]byte, 1<<20)
	rand.Read(buf)
	var wg sync.WaitGroup
	for i := 0; i < 128; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			kc.PutB(buf)
		}()
	}
	done := make(chan bool)
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		c.Error("timeout")
	}
}

func (s *ServerRequiredSuite) TestPutAskGet(c *C) {
	srv, kc, logbuf := runProxy(c, false, false, nil)
	defer srv.Close()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	var hash2 string

	{
		_, _, err := kc.Ask(hash)
		c.Check(err, Equals, keepclient.BlockNotFound)
		c.Log("Finished Ask (expected BlockNotFound)")
	}

	{
		reader, _, _, err := kc.Get(hash)
		c.Check(reader, Equals, nil)
		c.Check(err, Equals, keepclient.BlockNotFound)
		c.Log("Finished Get (expected BlockNotFound)")
	}

	// Note in bug #5309 among other errors keepproxy would set
	// Content-Length incorrectly on the 404 BlockNotFound response, this
	// would result in a protocol violation that would prevent reuse of the
	// connection, which would manifest by the next attempt to use the
	// connection (in this case the PutB below) failing.  So to test for
	// that bug it's necessary to trigger an error response (such as
	// BlockNotFound) and then do something else with the same httpClient
	// connection.

	{
		var rep int
		var err error
		hash2, rep, err = kc.PutB([]byte("foo"))
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+3(\+.+)?$`, hash))
		c.Check(rep, Equals, 2)
		c.Check(err, Equals, nil)
		c.Log("Finished PutB (expected success)")

		c.Check(logbuf.String(), Matches, `(?ms).* locator=acbd18db4cc2f85cedef654fccc4a4d8\+3.* userFullName="TestCase Administrator".* userUUID=zzzzz-tpzed-d9tiejq69daie8f.*`)
		logbuf.Reset()
	}

	{
		blocklen, _, err := kc.Ask(hash2)
		c.Assert(err, Equals, nil)
		c.Check(blocklen, Equals, int64(3))
		c.Log("Finished Ask (expected success)")
		c.Check(logbuf.String(), Matches, `(?ms).* locator=acbd18db4cc2f85cedef654fccc4a4d8\+3.* userFullName="TestCase Administrator".* userUUID=zzzzz-tpzed-d9tiejq69daie8f.*`)
		logbuf.Reset()
	}

	{
		reader, blocklen, _, err := kc.Get(hash2)
		c.Assert(err, Equals, nil)
		all, err := ioutil.ReadAll(reader)
		c.Check(err, IsNil)
		c.Check(all, DeepEquals, []byte("foo"))
		c.Check(blocklen, Equals, int64(3))
		c.Log("Finished Get (expected success)")
		c.Check(logbuf.String(), Matches, `(?ms).* locator=acbd18db4cc2f85cedef654fccc4a4d8\+3.* userFullName="TestCase Administrator".* userUUID=zzzzz-tpzed-d9tiejq69daie8f.*`)
		logbuf.Reset()
	}

	{
		var rep int
		var err error
		hash2, rep, err = kc.PutB([]byte(""))
		c.Check(hash2, Matches, `^d41d8cd98f00b204e9800998ecf8427e\+0(\+.+)?$`)
		c.Check(rep, Equals, 2)
		c.Check(err, Equals, nil)
		c.Log("Finished PutB zero block")
	}

	{
		reader, blocklen, _, err := kc.Get("d41d8cd98f00b204e9800998ecf8427e")
		c.Assert(err, Equals, nil)
		all, err := ioutil.ReadAll(reader)
		c.Check(err, IsNil)
		c.Check(all, DeepEquals, []byte(""))
		c.Check(blocklen, Equals, int64(0))
		c.Log("Finished Get zero block")
	}
}

func (s *ServerRequiredSuite) TestPutAskGetForbidden(c *C) {
	srv, kc, _ := runProxy(c, true, false, nil)
	defer srv.Close()

	hash := fmt.Sprintf("%x+3", md5.Sum([]byte("bar")))

	_, _, err := kc.Ask(hash)
	c.Check(err, FitsTypeOf, &keepclient.ErrNotFound{})

	hash2, rep, err := kc.PutB([]byte("bar"))
	c.Check(hash2, Equals, "")
	c.Check(rep, Equals, 0)
	c.Check(err, FitsTypeOf, keepclient.InsufficientReplicasError{})

	blocklen, _, err := kc.Ask(hash)
	c.Check(err, FitsTypeOf, &keepclient.ErrNotFound{})
	c.Check(err, ErrorMatches, ".*HTTP 403.*")
	c.Check(blocklen, Equals, int64(0))

	_, blocklen, _, err = kc.Get(hash)
	c.Check(err, FitsTypeOf, &keepclient.ErrNotFound{})
	c.Check(err, ErrorMatches, ".*HTTP 403.*")
	c.Check(blocklen, Equals, int64(0))
}

func testPermission(c *C, admin bool, perm arvados.UploadDownloadPermission) {
	kp := arvados.UploadDownloadRolePermissions{}
	if admin {
		kp.Admin = perm
		kp.User = arvados.UploadDownloadPermission{Upload: true, Download: true}
	} else {
		kp.Admin = arvados.UploadDownloadPermission{Upload: true, Download: true}
		kp.User = perm
	}

	srv, kc, logbuf := runProxy(c, false, false, &kp)
	defer srv.Close()
	if admin {
		kc.Arvados.ApiToken = arvadostest.AdminToken
	} else {
		kc.Arvados.ApiToken = arvadostest.ActiveToken
	}

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	var hash2 string

	{
		var rep int
		var err error
		hash2, rep, err = kc.PutB([]byte("foo"))

		if perm.Upload {
			c.Check(hash2, Matches, fmt.Sprintf(`^%s\+3(\+.+)?$`, hash))
			c.Check(rep, Equals, 2)
			c.Check(err, Equals, nil)
			c.Log("Finished PutB (expected success)")
			if admin {
				c.Check(logbuf.String(), Matches, `(?ms).* locator=acbd18db4cc2f85cedef654fccc4a4d8\+3.* userFullName="TestCase Administrator".* userUUID=zzzzz-tpzed-d9tiejq69daie8f.*`)
			} else {

				c.Check(logbuf.String(), Matches, `(?ms).* locator=acbd18db4cc2f85cedef654fccc4a4d8\+3.* userFullName="Active User".* userUUID=zzzzz-tpzed-xurymjxw79nv3jz.*`)
			}
		} else {
			c.Check(hash2, Equals, "")
			c.Check(rep, Equals, 0)
			c.Check(err, FitsTypeOf, keepclient.InsufficientReplicasError{})
		}
		logbuf.Reset()
	}
	if perm.Upload {
		// can't test download without upload.

		reader, blocklen, _, err := kc.Get(hash2)
		if perm.Download {
			c.Assert(err, Equals, nil)
			all, err := ioutil.ReadAll(reader)
			c.Check(err, IsNil)
			c.Check(all, DeepEquals, []byte("foo"))
			c.Check(blocklen, Equals, int64(3))
			c.Log("Finished Get (expected success)")
			if admin {
				c.Check(logbuf.String(), Matches, `(?ms).* locator=acbd18db4cc2f85cedef654fccc4a4d8\+3.* userFullName="TestCase Administrator".* userUUID=zzzzz-tpzed-d9tiejq69daie8f.*`)
			} else {
				c.Check(logbuf.String(), Matches, `(?ms).* locator=acbd18db4cc2f85cedef654fccc4a4d8\+3.* userFullName="Active User".* userUUID=zzzzz-tpzed-xurymjxw79nv3jz.*`)
			}
		} else {
			c.Check(err, FitsTypeOf, &keepclient.ErrNotFound{})
			c.Check(err, ErrorMatches, ".*Missing or invalid Authorization header, or method not allowed.*")
			c.Check(blocklen, Equals, int64(0))
		}
		logbuf.Reset()
	}

}

func (s *ServerRequiredSuite) TestPutGetPermission(c *C) {

	for _, adminperm := range []bool{true, false} {
		for _, userperm := range []bool{true, false} {

			testPermission(c, true,
				arvados.UploadDownloadPermission{
					Upload:   adminperm,
					Download: true,
				})
			testPermission(c, true,
				arvados.UploadDownloadPermission{
					Upload:   true,
					Download: adminperm,
				})
			testPermission(c, false,
				arvados.UploadDownloadPermission{
					Upload:   true,
					Download: userperm,
				})
			testPermission(c, false,
				arvados.UploadDownloadPermission{
					Upload:   true,
					Download: userperm,
				})
		}
	}
}

func (s *ServerRequiredSuite) TestCorsHeaders(c *C) {
	srv, _, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	{
		client := http.Client{}
		req, err := http.NewRequest("OPTIONS",
			fmt.Sprintf("http://%s/%x+3", srv.Addr, md5.Sum([]byte("foo"))),
			nil)
		c.Assert(err, IsNil)
		req.Header.Add("Access-Control-Request-Method", "PUT")
		req.Header.Add("Access-Control-Request-Headers", "Authorization, X-Keep-Desired-Replicas")
		resp, err := client.Do(req)
		c.Check(err, Equals, nil)
		c.Check(resp.StatusCode, Equals, 200)
		body, err := ioutil.ReadAll(resp.Body)
		c.Check(err, IsNil)
		c.Check(string(body), Equals, "")
		c.Check(resp.Header.Get("Access-Control-Allow-Methods"), Equals, "GET, HEAD, POST, PUT, OPTIONS")
		c.Check(resp.Header.Get("Access-Control-Allow-Origin"), Equals, "*")
	}

	{
		resp, err := http.Get(fmt.Sprintf("http://%s/%x+3", srv.Addr, md5.Sum([]byte("foo"))))
		c.Check(err, Equals, nil)
		c.Check(resp.Header.Get("Access-Control-Allow-Headers"), Equals, "Authorization, Content-Length, Content-Type, X-Keep-Desired-Replicas")
		c.Check(resp.Header.Get("Access-Control-Allow-Origin"), Equals, "*")
	}
}

func (s *ServerRequiredSuite) TestPostWithoutHash(c *C) {
	srv, _, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	{
		client := http.Client{}
		req, err := http.NewRequest("POST",
			"http://"+srv.Addr+"/",
			strings.NewReader("qux"))
		c.Check(err, IsNil)
		req.Header.Add("Authorization", "OAuth2 "+arvadostest.ActiveToken)
		req.Header.Add("Content-Type", "application/octet-stream")
		resp, err := client.Do(req)
		c.Check(err, Equals, nil)
		body, err := ioutil.ReadAll(resp.Body)
		c.Check(err, Equals, nil)
		c.Check(string(body), Matches,
			fmt.Sprintf(`^%x\+3(\+.+)?$`, md5.Sum([]byte("qux"))))
	}
}

func (s *ServerRequiredSuite) TestStripHint(c *C) {
	c.Check(removeHint.ReplaceAllString("http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73+K@zzzzz", "$1"),
		Equals,
		"http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73")
	c.Check(removeHint.ReplaceAllString("http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+K@zzzzz+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73", "$1"),
		Equals,
		"http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73")
	c.Check(removeHint.ReplaceAllString("http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73+K@zzzzz-zzzzz-zzzzzzzzzzzzzzz", "$1"),
		Equals,
		"http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73+K@zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Check(removeHint.ReplaceAllString("http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+K@zzzzz-zzzzz-zzzzzzzzzzzzzzz+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73", "$1"),
		Equals,
		"http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+K@zzzzz-zzzzz-zzzzzzzzzzzzzzz+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73")

}

// Test GetIndex
//   Put one block, with 2 replicas
//   With no prefix (expect the block locator, twice)
//   With an existing prefix (expect the block locator, twice)
//   With a valid but non-existing prefix (expect "\n")
//   With an invalid prefix (expect error)
func (s *ServerRequiredSuite) TestGetIndex(c *C) {
	getIndexWorker(c, false)
}

// Test GetIndex
//   Uses config.yml
//   Put one block, with 2 replicas
//   With no prefix (expect the block locator, twice)
//   With an existing prefix (expect the block locator, twice)
//   With a valid but non-existing prefix (expect "\n")
//   With an invalid prefix (expect error)
func (s *ServerRequiredConfigYmlSuite) TestGetIndex(c *C) {
	getIndexWorker(c, true)
}

func getIndexWorker(c *C, useConfig bool) {
	srv, kc, _ := runProxy(c, false, useConfig, nil)
	defer srv.Close()

	// Put "index-data" blocks
	data := []byte("index-data")
	hash := fmt.Sprintf("%x", md5.Sum(data))

	hash2, rep, err := kc.PutB(data)
	c.Check(hash2, Matches, fmt.Sprintf(`^%s\+10(\+.+)?$`, hash))
	c.Check(rep, Equals, 2)
	c.Check(err, Equals, nil)

	reader, blocklen, _, err := kc.Get(hash)
	c.Assert(err, IsNil)
	c.Check(blocklen, Equals, int64(10))
	all, err := ioutil.ReadAll(reader)
	c.Assert(err, IsNil)
	c.Check(all, DeepEquals, data)

	// Put some more blocks
	_, _, err = kc.PutB([]byte("some-more-index-data"))
	c.Check(err, IsNil)

	kc.Arvados.ApiToken = arvadostest.SystemRootToken

	// Invoke GetIndex
	for _, spec := range []struct {
		prefix         string
		expectTestHash bool
		expectOther    bool
	}{
		{"", true, true},         // with no prefix
		{hash[:3], true, false},  // with matching prefix
		{"abcdef", false, false}, // with no such prefix
	} {
		indexReader, err := kc.GetIndex(TestProxyUUID, spec.prefix)
		c.Assert(err, Equals, nil)
		indexResp, err := ioutil.ReadAll(indexReader)
		c.Assert(err, Equals, nil)
		locators := strings.Split(string(indexResp), "\n")
		gotTestHash := 0
		gotOther := 0
		for _, locator := range locators {
			if locator == "" {
				continue
			}
			c.Check(locator[:len(spec.prefix)], Equals, spec.prefix)
			if locator[:32] == hash {
				gotTestHash++
			} else {
				gotOther++
			}
		}
		c.Check(gotTestHash == 2, Equals, spec.expectTestHash)
		c.Check(gotOther > 0, Equals, spec.expectOther)
	}

	// GetIndex with invalid prefix
	_, err = kc.GetIndex(TestProxyUUID, "xyz")
	c.Assert((err != nil), Equals, true)
}

func (s *ServerRequiredSuite) TestCollectionSharingToken(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()
	hash, _, err := kc.PutB([]byte("shareddata"))
	c.Check(err, IsNil)
	kc.Arvados.ApiToken = arvadostest.FooCollectionSharingToken
	rdr, _, _, err := kc.Get(hash)
	c.Assert(err, IsNil)
	data, err := ioutil.ReadAll(rdr)
	c.Check(err, IsNil)
	c.Check(data, DeepEquals, []byte("shareddata"))
}

func (s *ServerRequiredSuite) TestPutAskGetInvalidToken(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	// Put a test block
	hash, rep, err := kc.PutB([]byte("foo"))
	c.Check(err, IsNil)
	c.Check(rep, Equals, 2)

	for _, badToken := range []string{
		"nosuchtoken",
		"2ym314ysp27sk7h943q6vtc378srb06se3pq6ghurylyf3pdmx", // expired
	} {
		kc.Arvados.ApiToken = badToken

		// Ask and Get will fail only if the upstream
		// keepstore server checks for valid signatures.
		// Without knowing the blob signing key, there is no
		// way for keepproxy to know whether a given token is
		// permitted to read a block.  So these tests fail:
		if false {
			_, _, err = kc.Ask(hash)
			c.Assert(err, FitsTypeOf, &keepclient.ErrNotFound{})
			c.Check(err.(*keepclient.ErrNotFound).Temporary(), Equals, false)
			c.Check(err, ErrorMatches, ".*HTTP 403.*")

			_, _, _, err = kc.Get(hash)
			c.Assert(err, FitsTypeOf, &keepclient.ErrNotFound{})
			c.Check(err.(*keepclient.ErrNotFound).Temporary(), Equals, false)
			c.Check(err, ErrorMatches, ".*HTTP 403 \"Missing or invalid Authorization header, or method not allowed\".*")
		}

		_, _, err = kc.PutB([]byte("foo"))
		c.Check(err, ErrorMatches, ".*403.*Missing or invalid Authorization header, or method not allowed")
	}
}

func (s *ServerRequiredSuite) TestAskGetKeepProxyConnectionError(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	// Point keepproxy at a non-existent keepstore
	locals := map[string]string{
		TestProxyUUID: "http://localhost:12345",
	}
	srv.proxyHandler.KeepClient.SetServiceRoots(locals, nil, nil)

	// Ask should result in temporary bad gateway error
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	_, _, err := kc.Ask(hash)
	c.Check(err, NotNil)
	errNotFound, _ := err.(*keepclient.ErrNotFound)
	c.Check(errNotFound.Temporary(), Equals, true)
	c.Assert(err, ErrorMatches, ".*HTTP 502.*")

	// Get should result in temporary bad gateway error
	_, _, _, err = kc.Get(hash)
	c.Check(err, NotNil)
	errNotFound, _ = err.(*keepclient.ErrNotFound)
	c.Check(errNotFound.Temporary(), Equals, true)
	c.Assert(err, ErrorMatches, ".*HTTP 502.*")
}

func (s *NoKeepServerSuite) TestAskGetNoKeepServerError(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	for _, f := range []func() error{
		func() error {
			_, _, err := kc.Ask(hash)
			return err
		},
		func() error {
			_, _, _, err := kc.Get(hash)
			return err
		},
	} {
		err := f()
		c.Assert(err, NotNil)
		errNotFound, _ := err.(*keepclient.ErrNotFound)
		c.Check(errNotFound.Temporary(), Equals, true)
		c.Check(err, ErrorMatches, `.*HTTP 502.*`)
	}
}

func (s *ServerRequiredSuite) TestPing(c *C) {
	srv, kc, _ := runProxy(c, false, false, nil)
	defer srv.Close()

	rtr, err := newHandler(context.Background(), kc, 10*time.Second, &arvados.Cluster{ManagementToken: arvadostest.ManagementToken})
	c.Assert(err, check.IsNil)

	req, err := http.NewRequest("GET",
		"http://"+srv.Addr+"/_health/ping",
		nil)
	c.Assert(err, IsNil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ManagementToken)

	resp := httptest.NewRecorder()
	rtr.ServeHTTP(resp, req)
	c.Check(resp.Code, Equals, 200)
	c.Assert(resp.Body.String(), Matches, `{"health":"OK"}\n?`)
}
