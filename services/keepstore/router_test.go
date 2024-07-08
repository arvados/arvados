// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/prometheus/client_golang/prometheus"
	. "gopkg.in/check.v1"
)

// routerSuite tests that the router correctly translates HTTP
// requests to the appropriate keepstore functionality, and translates
// the results to HTTP responses.
type routerSuite struct {
	cluster *arvados.Cluster
}

var _ = Suite(&routerSuite{})

func testRouter(t TB, cluster *arvados.Cluster, reg *prometheus.Registry) (*router, context.CancelFunc) {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}
	ctx, cancel := context.WithCancel(context.Background())
	ks, kcancel := testKeepstore(t, cluster, reg)
	go func() {
		<-ctx.Done()
		kcancel()
	}()
	puller := newPuller(ctx, ks, reg)
	trasher := newTrasher(ctx, ks, reg)
	return newRouter(ks, puller, trasher).(*router), cancel
}

func (s *routerSuite) SetUpTest(c *C) {
	s.cluster = testCluster(c)
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Replication: 1, Driver: "stub", StorageClasses: map[string]bool{"testclass1": true}},
		"zzzzz-nyw5e-111111111111111": {Replication: 1, Driver: "stub", StorageClasses: map[string]bool{"testclass2": true}},
	}
	s.cluster.StorageClasses = map[string]arvados.StorageClassConfig{
		"testclass1": arvados.StorageClassConfig{
			Default: true,
		},
		"testclass2": arvados.StorageClassConfig{
			Default: true,
		},
	}
}

func (s *routerSuite) TestBlockRead_Token(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	err := router.keepstore.mountsW[0].BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Assert(err, IsNil)
	locSigned := router.keepstore.signLocator(arvadostest.ActiveTokenV2, fooHash+"+3")
	c.Assert(locSigned, Not(Equals), fooHash+"+3")

	// No token provided
	resp := call(router, "GET", "http://example/"+locSigned, "", nil, nil)
	c.Check(resp.Code, Equals, http.StatusUnauthorized)
	c.Check(resp.Body.String(), Matches, "no token provided in Authorization header\n")
	checkCORSHeaders(c, resp.Header())

	// Different token => invalid signature
	resp = call(router, "GET", "http://example/"+locSigned, "badtoken", nil, nil)
	c.Check(resp.Code, Equals, http.StatusBadRequest)
	c.Check(resp.Body.String(), Equals, "invalid signature\n")
	checkCORSHeaders(c, resp.Header())

	// Correct token
	resp = call(router, "GET", "http://example/"+locSigned, arvadostest.ActiveTokenV2, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Equals, "foo")
	checkCORSHeaders(c, resp.Header())

	// HEAD
	resp = call(router, "HEAD", "http://example/"+locSigned, arvadostest.ActiveTokenV2, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Result().ContentLength, Equals, int64(3))
	c.Check(resp.Body.String(), Equals, "")
	checkCORSHeaders(c, resp.Header())
}

// As a special case we allow HEAD requests that only provide a hash
// without a size hint. This accommodates uses of keep-block-check
// where it's inconvenient to attach size hints to known hashes.
//
// GET requests must provide a size hint -- otherwise we can't
// propagate a checksum mismatch error.
func (s *routerSuite) TestBlockRead_NoSizeHint(c *C) {
	s.cluster.Collections.BlobSigning = true
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()
	err := router.keepstore.mountsW[0].BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Assert(err, IsNil)

	// hash+signature
	hashSigned := router.keepstore.signLocator(arvadostest.ActiveTokenV2, fooHash)
	resp := call(router, "GET", "http://example/"+hashSigned, arvadostest.ActiveTokenV2, nil, nil)
	c.Check(resp.Code, Equals, http.StatusMethodNotAllowed)

	resp = call(router, "HEAD", "http://example/"+fooHash, "", nil, nil)
	c.Check(resp.Code, Equals, http.StatusUnauthorized)
	resp = call(router, "HEAD", "http://example/"+fooHash+"+3", "", nil, nil)
	c.Check(resp.Code, Equals, http.StatusUnauthorized)

	s.cluster.Collections.BlobSigning = false
	router, cancel = testRouter(c, s.cluster, nil)
	defer cancel()
	err = router.keepstore.mountsW[0].BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Assert(err, IsNil)

	resp = call(router, "GET", "http://example/"+fooHash, "", nil, nil)
	c.Check(resp.Code, Equals, http.StatusMethodNotAllowed)

	resp = call(router, "HEAD", "http://example/"+fooHash, "", nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Equals, "")
	c.Check(resp.Result().ContentLength, Equals, int64(3))
	c.Check(resp.Header().Get("Content-Length"), Equals, "3")
}

// By the time we discover the checksum mismatch, it's too late to
// change the response code, but the expected block size is given in
// the Content-Length response header, so a generic http client can
// detect the problem.
func (s *routerSuite) TestBlockRead_ChecksumMismatch(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	gooddata := make([]byte, 10_000_000)
	gooddata[0] = 'a'
	hash := fmt.Sprintf("%x", md5.Sum(gooddata))
	locSigned := router.keepstore.signLocator(arvadostest.ActiveTokenV2, fmt.Sprintf("%s+%d", hash, len(gooddata)))

	for _, baddata := range [][]byte{
		make([]byte, 3),
		make([]byte, len(gooddata)),
		make([]byte, len(gooddata)-1),
		make([]byte, len(gooddata)+1),
		make([]byte, len(gooddata)*2),
	} {
		c.Logf("=== baddata len %d", len(baddata))
		err := router.keepstore.mountsW[0].BlockWrite(context.Background(), hash, baddata)
		c.Assert(err, IsNil)

		resp := call(router, "GET", "http://example/"+locSigned, arvadostest.ActiveTokenV2, nil, nil)
		if !c.Check(resp.Code, Equals, http.StatusOK) {
			c.Logf("resp.Body: %s", resp.Body.String())
		}
		c.Check(resp.Body.Len(), Not(Equals), len(gooddata))
		c.Check(resp.Result().ContentLength, Equals, int64(len(gooddata)))
		checkCORSHeaders(c, resp.Header())

		resp = call(router, "HEAD", "http://example/"+locSigned, arvadostest.ActiveTokenV2, nil, nil)
		c.Check(resp.Code, Equals, http.StatusBadGateway)
		checkCORSHeaders(c, resp.Header())

		hashSigned := router.keepstore.signLocator(arvadostest.ActiveTokenV2, hash)
		resp = call(router, "HEAD", "http://example/"+hashSigned, arvadostest.ActiveTokenV2, nil, nil)
		c.Check(resp.Code, Equals, http.StatusBadGateway)
		checkCORSHeaders(c, resp.Header())
	}
}

func (s *routerSuite) TestBlockWrite(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	resp := call(router, "PUT", "http://example/"+fooHash, arvadostest.ActiveTokenV2, []byte("foo"), nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	checkCORSHeaders(c, resp.Header())
	locator := strings.TrimSpace(resp.Body.String())

	resp = call(router, "GET", "http://example/"+locator, arvadostest.ActiveTokenV2, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Equals, "foo")
}

func (s *routerSuite) TestBlockWrite_Headers(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	resp := call(router, "PUT", "http://example/"+fooHash, arvadostest.ActiveTokenV2, []byte("foo"), http.Header{"X-Keep-Desired-Replicas": []string{"2"}})
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Header().Get("X-Keep-Replicas-Stored"), Equals, "1")
	c.Check(sortCommaSeparated(resp.Header().Get("X-Keep-Storage-Classes-Confirmed")), Equals, "testclass1=1")

	resp = call(router, "PUT", "http://example/"+fooHash, arvadostest.ActiveTokenV2, []byte("foo"), http.Header{"X-Keep-Storage-Classes": []string{"testclass1"}})
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Header().Get("X-Keep-Replicas-Stored"), Equals, "1")
	c.Check(resp.Header().Get("X-Keep-Storage-Classes-Confirmed"), Equals, "testclass1=1")

	resp = call(router, "PUT", "http://example/"+fooHash, arvadostest.ActiveTokenV2, []byte("foo"), http.Header{"X-Keep-Storage-Classes": []string{" , testclass2 , "}})
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Header().Get("X-Keep-Replicas-Stored"), Equals, "1")
	c.Check(resp.Header().Get("X-Keep-Storage-Classes-Confirmed"), Equals, "testclass2=1")

	resp = call(router, "PUT", "http://example/"+fooHash, arvadostest.ActiveTokenV2, []byte("foo"), http.Header{"X-Keep-Storage-Classes": []string{"testclass1, testclass2"}})
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Header().Get("X-Keep-Replicas-Stored"), Equals, "2")
	c.Check(resp.Header().Get("X-Keep-Storage-Classes-Confirmed"), Equals, "testclass1=1, testclass2=1")
}

func sortCommaSeparated(s string) string {
	slice := strings.Split(s, ", ")
	sort.Strings(slice)
	return strings.Join(slice, ", ")
}

func (s *routerSuite) TestBlockTouch(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	resp := call(router, "TOUCH", "http://example/"+fooHash+"+3", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusNotFound)

	vol0 := router.keepstore.mountsW[0].volume.(*stubVolume)
	err := vol0.BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Assert(err, IsNil)
	vol1 := router.keepstore.mountsW[1].volume.(*stubVolume)
	err = vol1.BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Assert(err, IsNil)

	t1 := time.Now()
	resp = call(router, "TOUCH", "http://example/"+fooHash+"+3", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	t2 := time.Now()

	// Unauthorized request is a no-op
	resp = call(router, "TOUCH", "http://example/"+fooHash+"+3", arvadostest.ActiveTokenV2, nil, nil)
	c.Check(resp.Code, Equals, http.StatusForbidden)

	// Volume 0 mtime should be updated
	t, err := vol0.Mtime(fooHash)
	c.Check(err, IsNil)
	c.Check(t.After(t1), Equals, true)
	c.Check(t.Before(t2), Equals, true)

	// Volume 1 mtime should not be updated
	t, err = vol1.Mtime(fooHash)
	c.Check(err, IsNil)
	c.Check(t.Before(t1), Equals, true)

	err = vol0.BlockTrash(fooHash)
	c.Assert(err, IsNil)
	err = vol1.BlockTrash(fooHash)
	c.Assert(err, IsNil)
	resp = call(router, "TOUCH", "http://example/"+fooHash+"+3", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusNotFound)
}

func (s *routerSuite) TestBlockTrash(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	vol0 := router.keepstore.mountsW[0].volume.(*stubVolume)
	err := vol0.BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Assert(err, IsNil)
	err = vol0.blockTouchWithTime(fooHash, time.Now().Add(-s.cluster.Collections.BlobSigningTTL.Duration()))
	c.Assert(err, IsNil)
	resp := call(router, "DELETE", "http://example/"+fooHash+"+3", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(vol0.stubLog.String(), Matches, `(?ms).* trash .*`)
	err = vol0.BlockRead(context.Background(), fooHash, brdiscard)
	c.Assert(err, Equals, os.ErrNotExist)
}

func (s *routerSuite) TestBlockUntrash(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	vol0 := router.keepstore.mountsW[0].volume.(*stubVolume)
	err := vol0.BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Assert(err, IsNil)
	err = vol0.BlockTrash(fooHash)
	c.Assert(err, IsNil)
	err = vol0.BlockRead(context.Background(), fooHash, brdiscard)
	c.Assert(err, Equals, os.ErrNotExist)
	resp := call(router, "PUT", "http://example/untrash/"+fooHash+"+3", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(vol0.stubLog.String(), Matches, `(?ms).* untrash .*`)
	err = vol0.BlockRead(context.Background(), fooHash, brdiscard)
	c.Check(err, IsNil)
}

func (s *routerSuite) TestBadRequest(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	for _, trial := range []string{
		"GET /",
		"GET /xyz",
		"GET /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabcdefg",
		"GET /untrash",
		"GET /mounts/blocks/123",
		"GET /trash",
		"GET /pull",
		"GET /debug.json",  // old endpoint, no longer exists
		"GET /status.json", // old endpoint, no longer exists
		"POST /",
		"POST /aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"POST /trash",
		"PROPFIND /",
		"MAKE-COFFEE /",
	} {
		c.Logf("=== %s", trial)
		methodpath := strings.Split(trial, " ")
		req := httptest.NewRequest(methodpath[0], "http://example"+methodpath[1], nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		c.Check(resp.Code, Equals, http.StatusBadRequest)
	}
}

func (s *routerSuite) TestRequireAdminMgtToken(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	for _, token := range []string{"badtoken", ""} {
		for _, trial := range []string{
			"PUT /pull",
			"PUT /trash",
			"GET /index",
			"GET /index/",
			"GET /index/1234",
			"PUT /untrash/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		} {
			c.Logf("=== %s", trial)
			methodpath := strings.Split(trial, " ")
			req := httptest.NewRequest(methodpath[0], "http://example"+methodpath[1], nil)
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			if token == "" {
				c.Check(resp.Code, Equals, http.StatusUnauthorized)
			} else {
				c.Check(resp.Code, Equals, http.StatusForbidden)
			}
		}
	}
	req := httptest.NewRequest("TOUCH", "http://example/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	c.Check(resp.Code, Equals, http.StatusUnauthorized)
}

func (s *routerSuite) TestVolumeErrorStatusCode(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()
	router.keepstore.mountsW[0].volume.(*stubVolume).blockRead = func(_ context.Context, hash string, w io.WriterAt) error {
		return httpserver.ErrorWithStatus(errors.New("test error"), http.StatusBadGateway)
	}

	// To test whether we fall back to volume 1 after volume 0
	// returns an error, we need to use a block whose rendezvous
	// order has volume 0 first. Luckily "bar" is such a block.
	c.Assert(router.keepstore.rendezvous(barHash, router.keepstore.mountsR)[0].UUID, DeepEquals, router.keepstore.mountsR[0].UUID)

	locSigned := router.keepstore.signLocator(arvadostest.ActiveTokenV2, barHash+"+3")

	// Volume 0 fails with an error that specifies an HTTP status
	// code, so that code should be propagated to caller.
	resp := call(router, "GET", "http://example/"+locSigned, arvadostest.ActiveTokenV2, nil, nil)
	c.Check(resp.Code, Equals, http.StatusBadGateway)
	c.Check(resp.Body.String(), Equals, "test error\n")

	router.keepstore.mountsW[0].volume.(*stubVolume).blockRead = func(_ context.Context, hash string, w io.WriterAt) error {
		return errors.New("no http status provided")
	}
	resp = call(router, "GET", "http://example/"+locSigned, arvadostest.ActiveTokenV2, nil, nil)
	c.Check(resp.Code, Equals, http.StatusInternalServerError)
	c.Check(resp.Body.String(), Equals, "no http status provided\n")

	c.Assert(router.keepstore.mountsW[1].volume.BlockWrite(context.Background(), barHash, []byte("bar")), IsNil)

	// If the requested block is available on the second volume,
	// it doesn't matter that the first volume failed.
	resp = call(router, "GET", "http://example/"+locSigned, arvadostest.ActiveTokenV2, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Equals, "bar")
}

func (s *routerSuite) TestIndex(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	resp := call(router, "GET", "http://example/index", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Equals, "\n")

	resp = call(router, "GET", "http://example/index?prefix=fff", s.cluster.SystemRootToken, nil, nil)
	c.Check(resp.Code, Equals, http.StatusOK)
	c.Check(resp.Body.String(), Equals, "\n")

	t0 := time.Now().Add(-time.Hour)
	vol0 := router.keepstore.mounts["zzzzz-nyw5e-000000000000000"].volume.(*stubVolume)
	err := vol0.BlockWrite(context.Background(), fooHash, []byte("foo"))
	c.Assert(err, IsNil)
	err = vol0.blockTouchWithTime(fooHash, t0)
	c.Assert(err, IsNil)
	err = vol0.BlockWrite(context.Background(), barHash, []byte("bar"))
	c.Assert(err, IsNil)
	err = vol0.blockTouchWithTime(barHash, t0)
	c.Assert(err, IsNil)
	t1 := time.Now().Add(-time.Minute)
	vol1 := router.keepstore.mounts["zzzzz-nyw5e-111111111111111"].volume.(*stubVolume)
	err = vol1.BlockWrite(context.Background(), barHash, []byte("bar"))
	c.Assert(err, IsNil)
	err = vol1.blockTouchWithTime(barHash, t1)
	c.Assert(err, IsNil)

	for _, path := range []string{
		"/index?prefix=acb",
		"/index/acb",
		"/index/?prefix=acb",
		"/mounts/zzzzz-nyw5e-000000000000000/blocks?prefix=acb",
		"/mounts/zzzzz-nyw5e-000000000000000/blocks/?prefix=acb",
		"/mounts/zzzzz-nyw5e-000000000000000/blocks/acb",
	} {
		c.Logf("=== %s", path)
		resp = call(router, "GET", "http://example"+path, s.cluster.SystemRootToken, nil, nil)
		c.Check(resp.Code, Equals, http.StatusOK)
		c.Check(resp.Body.String(), Equals, fooHash+"+3 "+fmt.Sprintf("%d", t0.UnixNano())+"\n\n")
	}

	for _, path := range []string{
		"/index?prefix=37",
		"/index/37",
		"/index/?prefix=37",
	} {
		c.Logf("=== %s", path)
		resp = call(router, "GET", "http://example"+path, s.cluster.SystemRootToken, nil, nil)
		c.Check(resp.Code, Equals, http.StatusOK)
		c.Check(resp.Body.String(), Equals, ""+
			barHash+"+3 "+fmt.Sprintf("%d", t0.UnixNano())+"\n"+
			barHash+"+3 "+fmt.Sprintf("%d", t1.UnixNano())+"\n\n")
	}

	for _, path := range []string{
		"/mounts/zzzzz-nyw5e-111111111111111/blocks",
		"/mounts/zzzzz-nyw5e-111111111111111/blocks/",
		"/mounts/zzzzz-nyw5e-111111111111111/blocks?prefix=37",
		"/mounts/zzzzz-nyw5e-111111111111111/blocks/?prefix=37",
		"/mounts/zzzzz-nyw5e-111111111111111/blocks/37",
	} {
		c.Logf("=== %s", path)
		resp = call(router, "GET", "http://example"+path, s.cluster.SystemRootToken, nil, nil)
		c.Check(resp.Code, Equals, http.StatusOK)
		c.Check(resp.Body.String(), Equals, barHash+"+3 "+fmt.Sprintf("%d", t1.UnixNano())+"\n\n")
	}

	for _, path := range []string{
		"/index",
		"/index?prefix=",
		"/index/",
		"/index/?prefix=",
	} {
		c.Logf("=== %s", path)
		resp = call(router, "GET", "http://example"+path, s.cluster.SystemRootToken, nil, nil)
		c.Check(resp.Code, Equals, http.StatusOK)
		c.Check(strings.Split(resp.Body.String(), "\n"), HasLen, 5)
	}
}

// Check that the context passed to a volume method gets cancelled
// when the http client hangs up.
func (s *routerSuite) TestCancelOnDisconnect(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	unblock := make(chan struct{})
	router.keepstore.mountsW[0].volume.(*stubVolume).blockRead = func(ctx context.Context, hash string, w io.WriterAt) error {
		<-unblock
		c.Check(ctx.Err(), NotNil)
		return ctx.Err()
	}
	go func() {
		time.Sleep(time.Second / 10)
		cancel()
		close(unblock)
	}()
	locSigned := router.keepstore.signLocator(arvadostest.ActiveTokenV2, fooHash+"+3")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "http://example/"+locSigned, nil)
	c.Assert(err, IsNil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveTokenV2)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	c.Check(resp.Code, Equals, 499)
}

func (s *routerSuite) TestCORSPreflight(c *C) {
	router, cancel := testRouter(c, s.cluster, nil)
	defer cancel()

	for _, path := range []string{"/", "/whatever", "/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+123"} {
		c.Logf("=== %s", path)
		resp := call(router, http.MethodOptions, "http://example"+path, arvadostest.ActiveTokenV2, nil, nil)
		c.Check(resp.Code, Equals, http.StatusOK)
		c.Check(resp.Body.String(), Equals, "")
		checkCORSHeaders(c, resp.Header())
	}
}

func call(handler http.Handler, method, path, tok string, body []byte, hdr http.Header) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
	for k := range hdr {
		req.Header.Set(k, hdr.Get(k))
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	handler.ServeHTTP(resp, req)
	return resp
}

func checkCORSHeaders(c *C, h http.Header) {
	c.Check(h.Get("Access-Control-Allow-Methods"), Equals, "GET, HEAD, PUT, OPTIONS")
	c.Check(h.Get("Access-Control-Allow-Origin"), Equals, "*")
	c.Check(h.Get("Access-Control-Allow-Headers"), Equals, "Authorization, Content-Length, Content-Type, X-Keep-Desired-Replicas, X-Keep-Signature, X-Keep-Storage-Classes")
	c.Check(h.Get("Access-Control-Expose-Headers"), Equals, "X-Keep-Locator, X-Keep-Replicas-Stored, X-Keep-Storage-Classes-Confirmed")
}
