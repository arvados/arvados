// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Tests for Keep HTTP handlers:
//
//     GetBlockHandler
//     PutBlockHandler
//     IndexHandler
//
// The HTTP handlers are responsible for enforcing permission policy,
// so these tests must exercise all possible permission permutations.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

var testServiceURL = func() arvados.URL {
	return arvados.URL{Host: "localhost:12345", Scheme: "http"}
}()

func testCluster(t TB) *arvados.Cluster {
	cfg, err := config.NewLoader(bytes.NewBufferString("Clusters: {zzzzz: {}}"), ctxlog.TestLogger(t)).Load()
	if err != nil {
		t.Fatal(err)
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		t.Fatal(err)
	}
	cluster.SystemRootToken = arvadostest.SystemRootToken
	cluster.ManagementToken = arvadostest.ManagementToken
	cluster.Collections.BlobSigning = false
	return cluster
}

var _ = check.Suite(&HandlerSuite{})

type HandlerSuite struct {
	cluster *arvados.Cluster
	handler *handler
}

func (s *HandlerSuite) SetUpTest(c *check.C) {
	s.cluster = testCluster(c)
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Replication: 1, Driver: "mock"},
		"zzzzz-nyw5e-111111111111111": {Replication: 1, Driver: "mock"},
	}
	s.handler = &handler{}
}

// A RequestTester represents the parameters for an HTTP request to
// be issued on behalf of a unit test.
type RequestTester struct {
	uri            string
	apiToken       string
	method         string
	requestBody    []byte
	storageClasses string
}

// Test GetBlockHandler on the following situations:
//   - permissions off, unauthenticated request, unsigned locator
//   - permissions on, authenticated request, signed locator
//   - permissions on, authenticated request, unsigned locator
//   - permissions on, unauthenticated request, signed locator
//   - permissions on, authenticated request, expired locator
//   - permissions on, authenticated request, signed locator, transient error from backend
//
func (s *HandlerSuite) TestGetHandler(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	vols := s.handler.volmgr.AllWritable()
	err := vols[0].Put(context.Background(), TestHash, TestBlock)
	c.Check(err, check.IsNil)

	// Create locators for testing.
	// Turn on permission settings so we can generate signed locators.
	s.cluster.Collections.BlobSigning = true
	s.cluster.Collections.BlobSigningKey = knownKey
	s.cluster.Collections.BlobSigningTTL.Set("5m")

	var (
		unsignedLocator  = "/" + TestHash
		validTimestamp   = time.Now().Add(s.cluster.Collections.BlobSigningTTL.Duration())
		expiredTimestamp = time.Now().Add(-time.Hour)
		signedLocator    = "/" + SignLocator(s.cluster, TestHash, knownToken, validTimestamp)
		expiredLocator   = "/" + SignLocator(s.cluster, TestHash, knownToken, expiredTimestamp)
	)

	// -----------------
	// Test unauthenticated request with permissions off.
	s.cluster.Collections.BlobSigning = false

	// Unauthenticated request, unsigned locator
	// => OK
	response := IssueRequest(s.handler,
		&RequestTester{
			method: "GET",
			uri:    unsignedLocator,
		})
	ExpectStatusCode(c,
		"Unauthenticated request, unsigned locator", http.StatusOK, response)
	ExpectBody(c,
		"Unauthenticated request, unsigned locator",
		string(TestBlock),
		response)

	receivedLen := response.Header().Get("Content-Length")
	expectedLen := fmt.Sprintf("%d", len(TestBlock))
	if receivedLen != expectedLen {
		c.Errorf("expected Content-Length %s, got %s", expectedLen, receivedLen)
	}

	// ----------------
	// Permissions: on.
	s.cluster.Collections.BlobSigning = true

	// Authenticated request, signed locator
	// => OK
	response = IssueRequest(s.handler, &RequestTester{
		method:   "GET",
		uri:      signedLocator,
		apiToken: knownToken,
	})
	ExpectStatusCode(c,
		"Authenticated request, signed locator", http.StatusOK, response)
	ExpectBody(c,
		"Authenticated request, signed locator", string(TestBlock), response)

	receivedLen = response.Header().Get("Content-Length")
	expectedLen = fmt.Sprintf("%d", len(TestBlock))
	if receivedLen != expectedLen {
		c.Errorf("expected Content-Length %s, got %s", expectedLen, receivedLen)
	}

	// Authenticated request, unsigned locator
	// => PermissionError
	response = IssueRequest(s.handler, &RequestTester{
		method:   "GET",
		uri:      unsignedLocator,
		apiToken: knownToken,
	})
	ExpectStatusCode(c, "unsigned locator", PermissionError.HTTPCode, response)

	// Unauthenticated request, signed locator
	// => PermissionError
	response = IssueRequest(s.handler, &RequestTester{
		method: "GET",
		uri:    signedLocator,
	})
	ExpectStatusCode(c,
		"Unauthenticated request, signed locator",
		PermissionError.HTTPCode, response)

	// Authenticated request, expired locator
	// => ExpiredError
	response = IssueRequest(s.handler, &RequestTester{
		method:   "GET",
		uri:      expiredLocator,
		apiToken: knownToken,
	})
	ExpectStatusCode(c,
		"Authenticated request, expired locator",
		ExpiredError.HTTPCode, response)

	// Authenticated request, signed locator
	// => 503 Server busy (transient error)

	// Set up the block owning volume to respond with errors
	vols[0].Volume.(*MockVolume).Bad = true
	vols[0].Volume.(*MockVolume).BadVolumeError = VolumeBusyError
	response = IssueRequest(s.handler, &RequestTester{
		method:   "GET",
		uri:      signedLocator,
		apiToken: knownToken,
	})
	// A transient error from one volume while the other doesn't find the block
	// should make the service return a 503 so that clients can retry.
	ExpectStatusCode(c,
		"Volume backend busy",
		503, response)
}

// Test PutBlockHandler on the following situations:
//   - no server key
//   - with server key, authenticated request, unsigned locator
//   - with server key, unauthenticated request, unsigned locator
//
func (s *HandlerSuite) TestPutHandler(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	// --------------
	// No server key.

	s.cluster.Collections.BlobSigningKey = ""

	// Unauthenticated request, no server key
	// => OK (unsigned response)
	unsignedLocator := "/" + TestHash
	response := IssueRequest(s.handler,
		&RequestTester{
			method:      "PUT",
			uri:         unsignedLocator,
			requestBody: TestBlock,
		})

	ExpectStatusCode(c,
		"Unauthenticated request, no server key", http.StatusOK, response)
	ExpectBody(c,
		"Unauthenticated request, no server key",
		TestHashPutResp, response)

	// ------------------
	// With a server key.

	s.cluster.Collections.BlobSigningKey = knownKey
	s.cluster.Collections.BlobSigningTTL.Set("5m")

	// When a permission key is available, the locator returned
	// from an authenticated PUT request will be signed.

	// Authenticated PUT, signed locator
	// => OK (signed response)
	response = IssueRequest(s.handler,
		&RequestTester{
			method:      "PUT",
			uri:         unsignedLocator,
			requestBody: TestBlock,
			apiToken:    knownToken,
		})

	ExpectStatusCode(c,
		"Authenticated PUT, signed locator, with server key",
		http.StatusOK, response)
	responseLocator := strings.TrimSpace(response.Body.String())
	if VerifySignature(s.cluster, responseLocator, knownToken) != nil {
		c.Errorf("Authenticated PUT, signed locator, with server key:\n"+
			"response '%s' does not contain a valid signature",
			responseLocator)
	}

	// Unauthenticated PUT, unsigned locator
	// => OK
	response = IssueRequest(s.handler,
		&RequestTester{
			method:      "PUT",
			uri:         unsignedLocator,
			requestBody: TestBlock,
		})

	ExpectStatusCode(c,
		"Unauthenticated PUT, unsigned locator, with server key",
		http.StatusOK, response)
	ExpectBody(c,
		"Unauthenticated PUT, unsigned locator, with server key",
		TestHashPutResp, response)
}

func (s *HandlerSuite) TestPutAndDeleteSkipReadonlyVolumes(c *check.C) {
	s.cluster.Volumes["zzzzz-nyw5e-000000000000000"] = arvados.Volume{Driver: "mock", ReadOnly: true}
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	s.cluster.SystemRootToken = "fake-data-manager-token"
	IssueRequest(s.handler,
		&RequestTester{
			method:      "PUT",
			uri:         "/" + TestHash,
			requestBody: TestBlock,
		})

	s.cluster.Collections.BlobTrash = true
	IssueRequest(s.handler,
		&RequestTester{
			method:      "DELETE",
			uri:         "/" + TestHash,
			requestBody: TestBlock,
			apiToken:    s.cluster.SystemRootToken,
		})
	type expect struct {
		volid     string
		method    string
		callcount int
	}
	for _, e := range []expect{
		{"zzzzz-nyw5e-000000000000000", "Get", 0},
		{"zzzzz-nyw5e-000000000000000", "Compare", 0},
		{"zzzzz-nyw5e-000000000000000", "Touch", 0},
		{"zzzzz-nyw5e-000000000000000", "Put", 0},
		{"zzzzz-nyw5e-000000000000000", "Delete", 0},
		{"zzzzz-nyw5e-111111111111111", "Get", 0},
		{"zzzzz-nyw5e-111111111111111", "Compare", 1},
		{"zzzzz-nyw5e-111111111111111", "Touch", 1},
		{"zzzzz-nyw5e-111111111111111", "Put", 1},
		{"zzzzz-nyw5e-111111111111111", "Delete", 1},
	} {
		if calls := s.handler.volmgr.mountMap[e.volid].Volume.(*MockVolume).CallCount(e.method); calls != e.callcount {
			c.Errorf("Got %d %s() on vol %s, expect %d", calls, e.method, e.volid, e.callcount)
		}
	}
}

func (s *HandlerSuite) TestReadsOrderedByStorageClassPriority(c *check.C) {
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-111111111111111": {
			Driver:         "mock",
			Replication:    1,
			StorageClasses: map[string]bool{"class1": true}},
		"zzzzz-nyw5e-222222222222222": {
			Driver:         "mock",
			Replication:    1,
			StorageClasses: map[string]bool{"class2": true, "class3": true}},
	}

	for _, trial := range []struct {
		priority1 int // priority of class1, thus vol1
		priority2 int // priority of class2
		priority3 int // priority of class3 (vol2 priority will be max(priority2, priority3))
		get1      int // expected number of "get" ops on vol1
		get2      int // expected number of "get" ops on vol2
	}{
		{100, 50, 50, 1, 0},   // class1 has higher priority => try vol1 first, no need to try vol2
		{100, 100, 100, 1, 0}, // same priority, vol1 is first lexicographically => try vol1 first and succeed
		{66, 99, 33, 1, 1},    // class2 has higher priority => try vol2 first, then try vol1
		{66, 33, 99, 1, 1},    // class3 has highest priority => vol2 has highest => try vol2 first, then try vol1
	} {
		c.Logf("%+v", trial)
		s.cluster.StorageClasses = map[string]arvados.StorageClassConfig{
			"class1": {Priority: trial.priority1},
			"class2": {Priority: trial.priority2},
			"class3": {Priority: trial.priority3},
		}
		c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
		IssueRequest(s.handler,
			&RequestTester{
				method:         "PUT",
				uri:            "/" + TestHash,
				requestBody:    TestBlock,
				storageClasses: "class1",
			})
		IssueRequest(s.handler,
			&RequestTester{
				method: "GET",
				uri:    "/" + TestHash,
			})
		c.Check(s.handler.volmgr.mountMap["zzzzz-nyw5e-111111111111111"].Volume.(*MockVolume).CallCount("Get"), check.Equals, trial.get1)
		c.Check(s.handler.volmgr.mountMap["zzzzz-nyw5e-222222222222222"].Volume.(*MockVolume).CallCount("Get"), check.Equals, trial.get2)
	}
}

func (s *HandlerSuite) TestConcurrentWritesToMultipleStorageClasses(c *check.C) {
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-111111111111111": {
			Driver:         "mock",
			Replication:    1,
			StorageClasses: map[string]bool{"class1": true}},
		"zzzzz-nyw5e-121212121212121": {
			Driver:         "mock",
			Replication:    1,
			StorageClasses: map[string]bool{"class1": true, "class2": true}},
		"zzzzz-nyw5e-222222222222222": {
			Driver:         "mock",
			Replication:    1,
			StorageClasses: map[string]bool{"class2": true}},
	}

	for _, trial := range []struct {
		setCounter uint32 // value to stuff vm.counter, to control offset
		classes    string // desired classes
		put111     int    // expected number of "put" ops on 11111... after 2x put reqs
		put121     int    // expected number of "put" ops on 12121...
		put222     int    // expected number of "put" ops on 22222...
		cmp111     int    // expected number of "compare" ops on 11111... after 2x put reqs
		cmp121     int    // expected number of "compare" ops on 12121...
		cmp222     int    // expected number of "compare" ops on 22222...
	}{
		{0, "class1",
			1, 0, 0,
			2, 1, 0}, // first put compares on all vols with class2; second put succeeds after checking 121
		{0, "class2",
			0, 1, 0,
			0, 2, 1}, // first put compares on all vols with class2; second put succeeds after checking 121
		{0, "class1,class2",
			1, 1, 0,
			2, 2, 1}, // first put compares on all vols; second put succeeds after checking 111 and 121
		{1, "class1,class2",
			0, 1, 0, // vm.counter offset is 1 so the first volume attempted is 121
			2, 2, 1}, // first put compares on all vols; second put succeeds after checking 111 and 121
		{0, "class1,class2,class404",
			1, 1, 0,
			2, 2, 1}, // first put compares on all vols; second put doesn't compare on 222 because it already satisfied class2 on 121
	} {
		c.Logf("%+v", trial)
		s.cluster.StorageClasses = map[string]arvados.StorageClassConfig{
			"class1": {},
			"class2": {},
			"class3": {},
		}
		c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
		atomic.StoreUint32(&s.handler.volmgr.counter, trial.setCounter)
		for i := 0; i < 2; i++ {
			IssueRequest(s.handler,
				&RequestTester{
					method:         "PUT",
					uri:            "/" + TestHash,
					requestBody:    TestBlock,
					storageClasses: trial.classes,
				})
		}
		c.Check(s.handler.volmgr.mountMap["zzzzz-nyw5e-111111111111111"].Volume.(*MockVolume).CallCount("Put"), check.Equals, trial.put111)
		c.Check(s.handler.volmgr.mountMap["zzzzz-nyw5e-121212121212121"].Volume.(*MockVolume).CallCount("Put"), check.Equals, trial.put121)
		c.Check(s.handler.volmgr.mountMap["zzzzz-nyw5e-222222222222222"].Volume.(*MockVolume).CallCount("Put"), check.Equals, trial.put222)
		c.Check(s.handler.volmgr.mountMap["zzzzz-nyw5e-111111111111111"].Volume.(*MockVolume).CallCount("Compare"), check.Equals, trial.cmp111)
		c.Check(s.handler.volmgr.mountMap["zzzzz-nyw5e-121212121212121"].Volume.(*MockVolume).CallCount("Compare"), check.Equals, trial.cmp121)
		c.Check(s.handler.volmgr.mountMap["zzzzz-nyw5e-222222222222222"].Volume.(*MockVolume).CallCount("Compare"), check.Equals, trial.cmp222)
	}
}

// Test TOUCH requests.
func (s *HandlerSuite) TestTouchHandler(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
	vols := s.handler.volmgr.AllWritable()
	vols[0].Put(context.Background(), TestHash, TestBlock)
	vols[0].Volume.(*MockVolume).TouchWithDate(TestHash, time.Now().Add(-time.Hour))
	afterPut := time.Now()
	t, err := vols[0].Mtime(TestHash)
	c.Assert(err, check.IsNil)
	c.Assert(t.Before(afterPut), check.Equals, true)

	ExpectStatusCode(c,
		"touch with no credentials",
		http.StatusUnauthorized,
		IssueRequest(s.handler, &RequestTester{
			method: "TOUCH",
			uri:    "/" + TestHash,
		}))

	ExpectStatusCode(c,
		"touch with non-root credentials",
		http.StatusUnauthorized,
		IssueRequest(s.handler, &RequestTester{
			method:   "TOUCH",
			uri:      "/" + TestHash,
			apiToken: arvadostest.ActiveTokenV2,
		}))

	ExpectStatusCode(c,
		"touch non-existent block",
		http.StatusNotFound,
		IssueRequest(s.handler, &RequestTester{
			method:   "TOUCH",
			uri:      "/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			apiToken: s.cluster.SystemRootToken,
		}))

	beforeTouch := time.Now()
	ExpectStatusCode(c,
		"touch block",
		http.StatusOK,
		IssueRequest(s.handler, &RequestTester{
			method:   "TOUCH",
			uri:      "/" + TestHash,
			apiToken: s.cluster.SystemRootToken,
		}))
	t, err = vols[0].Mtime(TestHash)
	c.Assert(err, check.IsNil)
	c.Assert(t.After(beforeTouch), check.Equals, true)
}

// Test /index requests:
//   - unauthenticated /index request
//   - unauthenticated /index/prefix request
//   - authenticated   /index request        | non-superuser
//   - authenticated   /index/prefix request | non-superuser
//   - authenticated   /index request        | superuser
//   - authenticated   /index/prefix request | superuser
//
// The only /index requests that should succeed are those issued by the
// superuser. They should pass regardless of the value of BlobSigning.
//
func (s *HandlerSuite) TestIndexHandler(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	// Include multiple blocks on different volumes, and
	// some metadata files (which should be omitted from index listings)
	vols := s.handler.volmgr.AllWritable()
	vols[0].Put(context.Background(), TestHash, TestBlock)
	vols[1].Put(context.Background(), TestHash2, TestBlock2)
	vols[0].Put(context.Background(), TestHash+".meta", []byte("metadata"))
	vols[1].Put(context.Background(), TestHash2+".meta", []byte("metadata"))

	s.cluster.SystemRootToken = "DATA MANAGER TOKEN"

	unauthenticatedReq := &RequestTester{
		method: "GET",
		uri:    "/index",
	}
	authenticatedReq := &RequestTester{
		method:   "GET",
		uri:      "/index",
		apiToken: knownToken,
	}
	superuserReq := &RequestTester{
		method:   "GET",
		uri:      "/index",
		apiToken: s.cluster.SystemRootToken,
	}
	unauthPrefixReq := &RequestTester{
		method: "GET",
		uri:    "/index/" + TestHash[0:3],
	}
	authPrefixReq := &RequestTester{
		method:   "GET",
		uri:      "/index/" + TestHash[0:3],
		apiToken: knownToken,
	}
	superuserPrefixReq := &RequestTester{
		method:   "GET",
		uri:      "/index/" + TestHash[0:3],
		apiToken: s.cluster.SystemRootToken,
	}
	superuserNoSuchPrefixReq := &RequestTester{
		method:   "GET",
		uri:      "/index/abcd",
		apiToken: s.cluster.SystemRootToken,
	}
	superuserInvalidPrefixReq := &RequestTester{
		method:   "GET",
		uri:      "/index/xyz",
		apiToken: s.cluster.SystemRootToken,
	}

	// -------------------------------------------------------------
	// Only the superuser should be allowed to issue /index requests.

	// ---------------------------
	// BlobSigning enabled
	// This setting should not affect tests passing.
	s.cluster.Collections.BlobSigning = true

	// unauthenticated /index request
	// => UnauthorizedError
	response := IssueRequest(s.handler, unauthenticatedReq)
	ExpectStatusCode(c,
		"permissions on, unauthenticated request",
		UnauthorizedError.HTTPCode,
		response)

	// unauthenticated /index/prefix request
	// => UnauthorizedError
	response = IssueRequest(s.handler, unauthPrefixReq)
	ExpectStatusCode(c,
		"permissions on, unauthenticated /index/prefix request",
		UnauthorizedError.HTTPCode,
		response)

	// authenticated /index request, non-superuser
	// => UnauthorizedError
	response = IssueRequest(s.handler, authenticatedReq)
	ExpectStatusCode(c,
		"permissions on, authenticated request, non-superuser",
		UnauthorizedError.HTTPCode,
		response)

	// authenticated /index/prefix request, non-superuser
	// => UnauthorizedError
	response = IssueRequest(s.handler, authPrefixReq)
	ExpectStatusCode(c,
		"permissions on, authenticated /index/prefix request, non-superuser",
		UnauthorizedError.HTTPCode,
		response)

	// superuser /index request
	// => OK
	response = IssueRequest(s.handler, superuserReq)
	ExpectStatusCode(c,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	// ----------------------------
	// BlobSigning disabled
	// Valid Request should still pass.
	s.cluster.Collections.BlobSigning = false

	// superuser /index request
	// => OK
	response = IssueRequest(s.handler, superuserReq)
	ExpectStatusCode(c,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	expected := `^` + TestHash + `\+\d+ \d+\n` +
		TestHash2 + `\+\d+ \d+\n\n$`
	c.Check(response.Body.String(), check.Matches, expected, check.Commentf(
		"permissions on, superuser request"))

	// superuser /index/prefix request
	// => OK
	response = IssueRequest(s.handler, superuserPrefixReq)
	ExpectStatusCode(c,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	expected = `^` + TestHash + `\+\d+ \d+\n\n$`
	c.Check(response.Body.String(), check.Matches, expected, check.Commentf(
		"permissions on, superuser /index/prefix request"))

	// superuser /index/{no-such-prefix} request
	// => OK
	response = IssueRequest(s.handler, superuserNoSuchPrefixReq)
	ExpectStatusCode(c,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	if "\n" != response.Body.String() {
		c.Errorf("Expected empty response for %s. Found %s", superuserNoSuchPrefixReq.uri, response.Body.String())
	}

	// superuser /index/{invalid-prefix} request
	// => StatusBadRequest
	response = IssueRequest(s.handler, superuserInvalidPrefixReq)
	ExpectStatusCode(c,
		"permissions on, superuser request",
		http.StatusBadRequest,
		response)
}

// TestDeleteHandler
//
// Cases tested:
//
//   With no token and with a non-data-manager token:
//   * Delete existing block
//     (test for 403 Forbidden, confirm block not deleted)
//
//   With data manager token:
//
//   * Delete existing block
//     (test for 200 OK, response counts, confirm block deleted)
//
//   * Delete nonexistent block
//     (test for 200 OK, response counts)
//
//   TODO(twp):
//
//   * Delete block on read-only and read-write volume
//     (test for 200 OK, response with copies_deleted=1,
//     copies_failed=1, confirm block deleted only on r/w volume)
//
//   * Delete block on read-only volume only
//     (test for 200 OK, response with copies_deleted=0, copies_failed=1,
//     confirm block not deleted)
//
func (s *HandlerSuite) TestDeleteHandler(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	vols := s.handler.volmgr.AllWritable()
	vols[0].Put(context.Background(), TestHash, TestBlock)

	// Explicitly set the BlobSigningTTL to 0 for these
	// tests, to ensure the MockVolume deletes the blocks
	// even though they have just been created.
	s.cluster.Collections.BlobSigningTTL = arvados.Duration(0)

	var userToken = "NOT DATA MANAGER TOKEN"
	s.cluster.SystemRootToken = "DATA MANAGER TOKEN"

	s.cluster.Collections.BlobTrash = true

	unauthReq := &RequestTester{
		method: "DELETE",
		uri:    "/" + TestHash,
	}

	userReq := &RequestTester{
		method:   "DELETE",
		uri:      "/" + TestHash,
		apiToken: userToken,
	}

	superuserExistingBlockReq := &RequestTester{
		method:   "DELETE",
		uri:      "/" + TestHash,
		apiToken: s.cluster.SystemRootToken,
	}

	superuserNonexistentBlockReq := &RequestTester{
		method:   "DELETE",
		uri:      "/" + TestHash2,
		apiToken: s.cluster.SystemRootToken,
	}

	// Unauthenticated request returns PermissionError.
	var response *httptest.ResponseRecorder
	response = IssueRequest(s.handler, unauthReq)
	ExpectStatusCode(c,
		"unauthenticated request",
		PermissionError.HTTPCode,
		response)

	// Authenticated non-admin request returns PermissionError.
	response = IssueRequest(s.handler, userReq)
	ExpectStatusCode(c,
		"authenticated non-admin request",
		PermissionError.HTTPCode,
		response)

	// Authenticated admin request for nonexistent block.
	type deletecounter struct {
		Deleted int `json:"copies_deleted"`
		Failed  int `json:"copies_failed"`
	}
	var responseDc, expectedDc deletecounter

	response = IssueRequest(s.handler, superuserNonexistentBlockReq)
	ExpectStatusCode(c,
		"data manager request, nonexistent block",
		http.StatusNotFound,
		response)

	// Authenticated admin request for existing block while BlobTrash is false.
	s.cluster.Collections.BlobTrash = false
	response = IssueRequest(s.handler, superuserExistingBlockReq)
	ExpectStatusCode(c,
		"authenticated request, existing block, method disabled",
		MethodDisabledError.HTTPCode,
		response)
	s.cluster.Collections.BlobTrash = true

	// Authenticated admin request for existing block.
	response = IssueRequest(s.handler, superuserExistingBlockReq)
	ExpectStatusCode(c,
		"data manager request, existing block",
		http.StatusOK,
		response)
	// Expect response {"copies_deleted":1,"copies_failed":0}
	expectedDc = deletecounter{1, 0}
	json.NewDecoder(response.Body).Decode(&responseDc)
	if responseDc != expectedDc {
		c.Errorf("superuserExistingBlockReq\nexpected: %+v\nreceived: %+v",
			expectedDc, responseDc)
	}
	// Confirm the block has been deleted
	buf := make([]byte, BlockSize)
	_, err := vols[0].Get(context.Background(), TestHash, buf)
	var blockDeleted = os.IsNotExist(err)
	if !blockDeleted {
		c.Error("superuserExistingBlockReq: block not deleted")
	}

	// A DELETE request on a block newer than BlobSigningTTL
	// should return success but leave the block on the volume.
	vols[0].Put(context.Background(), TestHash, TestBlock)
	s.cluster.Collections.BlobSigningTTL = arvados.Duration(time.Hour)

	response = IssueRequest(s.handler, superuserExistingBlockReq)
	ExpectStatusCode(c,
		"data manager request, existing block",
		http.StatusOK,
		response)
	// Expect response {"copies_deleted":1,"copies_failed":0}
	expectedDc = deletecounter{1, 0}
	json.NewDecoder(response.Body).Decode(&responseDc)
	if responseDc != expectedDc {
		c.Errorf("superuserExistingBlockReq\nexpected: %+v\nreceived: %+v",
			expectedDc, responseDc)
	}
	// Confirm the block has NOT been deleted.
	_, err = vols[0].Get(context.Background(), TestHash, buf)
	if err != nil {
		c.Errorf("testing delete on new block: %s\n", err)
	}
}

// TestPullHandler
//
// Test handling of the PUT /pull statement.
//
// Cases tested: syntactically valid and invalid pull lists, from the
// data manager and from unprivileged users:
//
//   1. Valid pull list from an ordinary user
//      (expected result: 401 Unauthorized)
//
//   2. Invalid pull request from an ordinary user
//      (expected result: 401 Unauthorized)
//
//   3. Valid pull request from the data manager
//      (expected result: 200 OK with request body "Received 3 pull
//      requests"
//
//   4. Invalid pull request from the data manager
//      (expected result: 400 Bad Request)
//
// Test that in the end, the pull manager received a good pull list with
// the expected number of requests.
//
// TODO(twp): test concurrency: launch 100 goroutines to update the
// pull list simultaneously.  Make sure that none of them return 400
// Bad Request and that pullq.GetList() returns a valid list.
//
func (s *HandlerSuite) TestPullHandler(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	// Replace the router's pullq -- which the worker goroutines
	// started by setup() are now receiving from -- with a new
	// one, so we can see what the handler sends to it.
	pullq := NewWorkQueue()
	s.handler.Handler.(*router).pullq = pullq

	var userToken = "USER TOKEN"
	s.cluster.SystemRootToken = "DATA MANAGER TOKEN"

	goodJSON := []byte(`[
		{
			"locator":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+12345",
			"servers":[
				"http://server1",
				"http://server2"
		 	]
		},
		{
			"locator":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+12345",
			"servers":[]
		},
		{
			"locator":"cccccccccccccccccccccccccccccccc+12345",
			"servers":["http://server1"]
		}
	]`)

	badJSON := []byte(`{ "key":"I'm a little teapot" }`)

	type pullTest struct {
		name         string
		req          RequestTester
		responseCode int
		responseBody string
	}
	var testcases = []pullTest{
		{
			"Valid pull list from an ordinary user",
			RequestTester{"/pull", userToken, "PUT", goodJSON, ""},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Invalid pull request from an ordinary user",
			RequestTester{"/pull", userToken, "PUT", badJSON, ""},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Valid pull request from the data manager",
			RequestTester{"/pull", s.cluster.SystemRootToken, "PUT", goodJSON, ""},
			http.StatusOK,
			"Received 3 pull requests\n",
		},
		{
			"Invalid pull request from the data manager",
			RequestTester{"/pull", s.cluster.SystemRootToken, "PUT", badJSON, ""},
			http.StatusBadRequest,
			"",
		},
	}

	for _, tst := range testcases {
		response := IssueRequest(s.handler, &tst.req)
		ExpectStatusCode(c, tst.name, tst.responseCode, response)
		ExpectBody(c, tst.name, tst.responseBody, response)
	}

	// The Keep pull manager should have received one good list with 3
	// requests on it.
	for i := 0; i < 3; i++ {
		var item interface{}
		select {
		case item = <-pullq.NextItem:
		case <-time.After(time.Second):
			c.Error("timed out")
		}
		if _, ok := item.(PullRequest); !ok {
			c.Errorf("item %v could not be parsed as a PullRequest", item)
		}
	}

	expectChannelEmpty(c, pullq.NextItem)
}

// TestTrashHandler
//
// Test cases:
//
// Cases tested: syntactically valid and invalid trash lists, from the
// data manager and from unprivileged users:
//
//   1. Valid trash list from an ordinary user
//      (expected result: 401 Unauthorized)
//
//   2. Invalid trash list from an ordinary user
//      (expected result: 401 Unauthorized)
//
//   3. Valid trash list from the data manager
//      (expected result: 200 OK with request body "Received 3 trash
//      requests"
//
//   4. Invalid trash list from the data manager
//      (expected result: 400 Bad Request)
//
// Test that in the end, the trash collector received a good list
// trash list with the expected number of requests.
//
// TODO(twp): test concurrency: launch 100 goroutines to update the
// pull list simultaneously.  Make sure that none of them return 400
// Bad Request and that replica.Dump() returns a valid list.
//
func (s *HandlerSuite) TestTrashHandler(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
	// Replace the router's trashq -- which the worker goroutines
	// started by setup() are now receiving from -- with a new
	// one, so we can see what the handler sends to it.
	trashq := NewWorkQueue()
	s.handler.Handler.(*router).trashq = trashq

	var userToken = "USER TOKEN"
	s.cluster.SystemRootToken = "DATA MANAGER TOKEN"

	goodJSON := []byte(`[
		{
			"locator":"block1",
			"block_mtime":1409082153
		},
		{
			"locator":"block2",
			"block_mtime":1409082153
		},
		{
			"locator":"block3",
			"block_mtime":1409082153
		}
	]`)

	badJSON := []byte(`I am not a valid JSON string`)

	type trashTest struct {
		name         string
		req          RequestTester
		responseCode int
		responseBody string
	}

	var testcases = []trashTest{
		{
			"Valid trash list from an ordinary user",
			RequestTester{"/trash", userToken, "PUT", goodJSON, ""},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Invalid trash list from an ordinary user",
			RequestTester{"/trash", userToken, "PUT", badJSON, ""},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Valid trash list from the data manager",
			RequestTester{"/trash", s.cluster.SystemRootToken, "PUT", goodJSON, ""},
			http.StatusOK,
			"Received 3 trash requests\n",
		},
		{
			"Invalid trash list from the data manager",
			RequestTester{"/trash", s.cluster.SystemRootToken, "PUT", badJSON, ""},
			http.StatusBadRequest,
			"",
		},
	}

	for _, tst := range testcases {
		response := IssueRequest(s.handler, &tst.req)
		ExpectStatusCode(c, tst.name, tst.responseCode, response)
		ExpectBody(c, tst.name, tst.responseBody, response)
	}

	// The trash collector should have received one good list with 3
	// requests on it.
	for i := 0; i < 3; i++ {
		item := <-trashq.NextItem
		if _, ok := item.(TrashRequest); !ok {
			c.Errorf("item %v could not be parsed as a TrashRequest", item)
		}
	}

	expectChannelEmpty(c, trashq.NextItem)
}

// ====================
// Helper functions
// ====================

// IssueTestRequest executes an HTTP request described by rt, to a
// REST router.  It returns the HTTP response to the request.
func IssueRequest(handler http.Handler, rt *RequestTester) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	body := bytes.NewReader(rt.requestBody)
	req, _ := http.NewRequest(rt.method, rt.uri, body)
	if rt.apiToken != "" {
		req.Header.Set("Authorization", "OAuth2 "+rt.apiToken)
	}
	if rt.storageClasses != "" {
		req.Header.Set("X-Keep-Storage-Classes", rt.storageClasses)
	}
	handler.ServeHTTP(response, req)
	return response
}

func IssueHealthCheckRequest(handler http.Handler, rt *RequestTester) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	body := bytes.NewReader(rt.requestBody)
	req, _ := http.NewRequest(rt.method, rt.uri, body)
	if rt.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+rt.apiToken)
	}
	handler.ServeHTTP(response, req)
	return response
}

// ExpectStatusCode checks whether a response has the specified status code,
// and reports a test failure if not.
func ExpectStatusCode(
	c *check.C,
	testname string,
	expectedStatus int,
	response *httptest.ResponseRecorder) {
	c.Check(response.Code, check.Equals, expectedStatus, check.Commentf("%s", testname))
}

func ExpectBody(
	c *check.C,
	testname string,
	expectedBody string,
	response *httptest.ResponseRecorder) {
	if expectedBody != "" && response.Body.String() != expectedBody {
		c.Errorf("%s: expected response body '%s', got %+v",
			testname, expectedBody, response)
	}
}

// See #7121
func (s *HandlerSuite) TestPutNeedsOnlyOneBuffer(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	defer func(orig *bufferPool) {
		bufs = orig
	}(bufs)
	bufs = newBufferPool(ctxlog.TestLogger(c), 1, BlockSize)

	ok := make(chan struct{})
	go func() {
		for i := 0; i < 2; i++ {
			response := IssueRequest(s.handler,
				&RequestTester{
					method:      "PUT",
					uri:         "/" + TestHash,
					requestBody: TestBlock,
				})
			ExpectStatusCode(c,
				"TestPutNeedsOnlyOneBuffer", http.StatusOK, response)
		}
		ok <- struct{}{}
	}()

	select {
	case <-ok:
	case <-time.After(time.Second):
		c.Fatal("PUT deadlocks with MaxKeepBlobBuffers==1")
	}
}

// Invoke the PutBlockHandler a bunch of times to test for bufferpool resource
// leak.
func (s *HandlerSuite) TestPutHandlerNoBufferleak(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	ok := make(chan bool)
	go func() {
		for i := 0; i < s.cluster.API.MaxKeepBlobBuffers+1; i++ {
			// Unauthenticated request, no server key
			// => OK (unsigned response)
			unsignedLocator := "/" + TestHash
			response := IssueRequest(s.handler,
				&RequestTester{
					method:      "PUT",
					uri:         unsignedLocator,
					requestBody: TestBlock,
				})
			ExpectStatusCode(c,
				"TestPutHandlerBufferleak", http.StatusOK, response)
			ExpectBody(c,
				"TestPutHandlerBufferleak",
				TestHashPutResp, response)
		}
		ok <- true
	}()
	select {
	case <-time.After(20 * time.Second):
		// If the buffer pool leaks, the test goroutine hangs.
		c.Fatal("test did not finish, assuming pool leaked")
	case <-ok:
	}
}

type notifyingResponseRecorder struct {
	*httptest.ResponseRecorder
	closer chan bool
}

func (r *notifyingResponseRecorder) CloseNotify() <-chan bool {
	return r.closer
}

func (s *HandlerSuite) TestGetHandlerClientDisconnect(c *check.C) {
	s.cluster.Collections.BlobSigning = false
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	defer func(orig *bufferPool) {
		bufs = orig
	}(bufs)
	bufs = newBufferPool(ctxlog.TestLogger(c), 1, BlockSize)
	defer bufs.Put(bufs.Get(BlockSize))

	if err := s.handler.volmgr.AllWritable()[0].Put(context.Background(), TestHash, TestBlock); err != nil {
		c.Error(err)
	}

	resp := &notifyingResponseRecorder{
		ResponseRecorder: httptest.NewRecorder(),
		closer:           make(chan bool, 1),
	}
	if _, ok := http.ResponseWriter(resp).(http.CloseNotifier); !ok {
		c.Fatal("notifyingResponseRecorder is broken")
	}
	// If anyone asks, the client has disconnected.
	resp.closer <- true

	ok := make(chan struct{})
	go func() {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/%s+%d", TestHash, len(TestBlock)), nil)
		s.handler.ServeHTTP(resp, req)
		ok <- struct{}{}
	}()

	select {
	case <-time.After(20 * time.Second):
		c.Fatal("request took >20s, close notifier must be broken")
	case <-ok:
	}

	ExpectStatusCode(c, "client disconnect", http.StatusServiceUnavailable, resp.ResponseRecorder)
	for i, v := range s.handler.volmgr.AllWritable() {
		if calls := v.Volume.(*MockVolume).called["GET"]; calls != 0 {
			c.Errorf("volume %d got %d calls, expected 0", i, calls)
		}
	}
}

// Invoke the GetBlockHandler a bunch of times to test for bufferpool resource
// leak.
func (s *HandlerSuite) TestGetHandlerNoBufferLeak(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	vols := s.handler.volmgr.AllWritable()
	if err := vols[0].Put(context.Background(), TestHash, TestBlock); err != nil {
		c.Error(err)
	}

	ok := make(chan bool)
	go func() {
		for i := 0; i < s.cluster.API.MaxKeepBlobBuffers+1; i++ {
			// Unauthenticated request, unsigned locator
			// => OK
			unsignedLocator := "/" + TestHash
			response := IssueRequest(s.handler,
				&RequestTester{
					method: "GET",
					uri:    unsignedLocator,
				})
			ExpectStatusCode(c,
				"Unauthenticated request, unsigned locator", http.StatusOK, response)
			ExpectBody(c,
				"Unauthenticated request, unsigned locator",
				string(TestBlock),
				response)
		}
		ok <- true
	}()
	select {
	case <-time.After(20 * time.Second):
		// If the buffer pool leaks, the test goroutine hangs.
		c.Fatal("test did not finish, assuming pool leaked")
	case <-ok:
	}
}

func (s *HandlerSuite) TestPutStorageClasses(c *check.C) {
	s.cluster.Volumes = map[string]arvados.Volume{
		"zzzzz-nyw5e-000000000000000": {Replication: 1, Driver: "mock"}, // "default" is implicit
		"zzzzz-nyw5e-111111111111111": {Replication: 1, Driver: "mock", StorageClasses: map[string]bool{"special": true, "extra": true}},
		"zzzzz-nyw5e-222222222222222": {Replication: 1, Driver: "mock", StorageClasses: map[string]bool{"readonly": true}, ReadOnly: true},
	}
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
	rt := RequestTester{
		method:      "PUT",
		uri:         "/" + TestHash,
		requestBody: TestBlock,
	}

	for _, trial := range []struct {
		ask    string
		expect string
	}{
		{"", ""},
		{"default", "default=1"},
		{" , default , default , ", "default=1"},
		{"special", "extra=1, special=1"},
		{"special, readonly", "extra=1, special=1"},
		{"special, nonexistent", "extra=1, special=1"},
		{"extra, special", "extra=1, special=1"},
		{"default, special", "default=1, extra=1, special=1"},
	} {
		c.Logf("success case %#v", trial)
		rt.storageClasses = trial.ask
		resp := IssueRequest(s.handler, &rt)
		if trial.expect == "" {
			// any non-empty value is correct
			c.Check(resp.Header().Get("X-Keep-Storage-Classes-Confirmed"), check.Not(check.Equals), "")
		} else {
			c.Check(sortCommaSeparated(resp.Header().Get("X-Keep-Storage-Classes-Confirmed")), check.Equals, trial.expect)
		}
	}

	for _, trial := range []struct {
		ask string
	}{
		{"doesnotexist"},
		{"doesnotexist, readonly"},
		{"readonly"},
	} {
		c.Logf("failure case %#v", trial)
		rt.storageClasses = trial.ask
		resp := IssueRequest(s.handler, &rt)
		c.Check(resp.Code, check.Equals, http.StatusServiceUnavailable)
	}
}

func sortCommaSeparated(s string) string {
	slice := strings.Split(s, ", ")
	sort.Strings(slice)
	return strings.Join(slice, ", ")
}

func (s *HandlerSuite) TestPutResponseHeader(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	resp := IssueRequest(s.handler, &RequestTester{
		method:      "PUT",
		uri:         "/" + TestHash,
		requestBody: TestBlock,
	})
	c.Logf("%#v", resp)
	c.Check(resp.Header().Get("X-Keep-Replicas-Stored"), check.Equals, "1")
	c.Check(resp.Header().Get("X-Keep-Storage-Classes-Confirmed"), check.Equals, "default=1")
}

func (s *HandlerSuite) TestUntrashHandler(c *check.C) {
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	// Set up Keep volumes
	vols := s.handler.volmgr.AllWritable()
	vols[0].Put(context.Background(), TestHash, TestBlock)

	s.cluster.SystemRootToken = "DATA MANAGER TOKEN"

	// unauthenticatedReq => UnauthorizedError
	unauthenticatedReq := &RequestTester{
		method: "PUT",
		uri:    "/untrash/" + TestHash,
	}
	response := IssueRequest(s.handler, unauthenticatedReq)
	ExpectStatusCode(c,
		"Unauthenticated request",
		UnauthorizedError.HTTPCode,
		response)

	// notDataManagerReq => UnauthorizedError
	notDataManagerReq := &RequestTester{
		method:   "PUT",
		uri:      "/untrash/" + TestHash,
		apiToken: knownToken,
	}

	response = IssueRequest(s.handler, notDataManagerReq)
	ExpectStatusCode(c,
		"Non-datamanager token",
		UnauthorizedError.HTTPCode,
		response)

	// datamanagerWithBadHashReq => StatusBadRequest
	datamanagerWithBadHashReq := &RequestTester{
		method:   "PUT",
		uri:      "/untrash/thisisnotalocator",
		apiToken: s.cluster.SystemRootToken,
	}
	response = IssueRequest(s.handler, datamanagerWithBadHashReq)
	ExpectStatusCode(c,
		"Bad locator in untrash request",
		http.StatusBadRequest,
		response)

	// datamanagerWrongMethodReq => StatusBadRequest
	datamanagerWrongMethodReq := &RequestTester{
		method:   "GET",
		uri:      "/untrash/" + TestHash,
		apiToken: s.cluster.SystemRootToken,
	}
	response = IssueRequest(s.handler, datamanagerWrongMethodReq)
	ExpectStatusCode(c,
		"Only PUT method is supported for untrash",
		http.StatusMethodNotAllowed,
		response)

	// datamanagerReq => StatusOK
	datamanagerReq := &RequestTester{
		method:   "PUT",
		uri:      "/untrash/" + TestHash,
		apiToken: s.cluster.SystemRootToken,
	}
	response = IssueRequest(s.handler, datamanagerReq)
	ExpectStatusCode(c,
		"",
		http.StatusOK,
		response)
	c.Check(response.Body.String(), check.Equals, "Successfully untrashed on: [MockVolume], [MockVolume]\n")
}

func (s *HandlerSuite) TestUntrashHandlerWithNoWritableVolumes(c *check.C) {
	// Change all volumes to read-only
	for uuid, v := range s.cluster.Volumes {
		v.ReadOnly = true
		s.cluster.Volumes[uuid] = v
	}
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)

	// datamanagerReq => StatusOK
	datamanagerReq := &RequestTester{
		method:   "PUT",
		uri:      "/untrash/" + TestHash,
		apiToken: s.cluster.SystemRootToken,
	}
	response := IssueRequest(s.handler, datamanagerReq)
	ExpectStatusCode(c,
		"No writable volumes",
		http.StatusNotFound,
		response)
}

func (s *HandlerSuite) TestHealthCheckPing(c *check.C) {
	s.cluster.ManagementToken = arvadostest.ManagementToken
	c.Assert(s.handler.setup(context.Background(), s.cluster, "", prometheus.NewRegistry(), testServiceURL), check.IsNil)
	pingReq := &RequestTester{
		method:   "GET",
		uri:      "/_health/ping",
		apiToken: arvadostest.ManagementToken,
	}
	response := IssueHealthCheckRequest(s.handler, pingReq)
	ExpectStatusCode(c,
		"",
		http.StatusOK,
		response)
	want := `{"health":"OK"}`
	if !strings.Contains(response.Body.String(), want) {
		c.Errorf("expected response to include %s: got %s", want, response.Body.String())
	}
}
