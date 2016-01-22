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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// A RequestTester represents the parameters for an HTTP request to
// be issued on behalf of a unit test.
type RequestTester struct {
	uri         string
	apiToken    string
	method      string
	requestBody []byte
}

// Test GetBlockHandler on the following situations:
//   - permissions off, unauthenticated request, unsigned locator
//   - permissions on, authenticated request, signed locator
//   - permissions on, authenticated request, unsigned locator
//   - permissions on, unauthenticated request, signed locator
//   - permissions on, authenticated request, expired locator
//
func TestGetHandler(t *testing.T) {
	defer teardown()

	// Prepare two test Keep volumes. Our block is stored on the second volume.
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	vols := KeepVM.AllWritable()
	if err := vols[0].Put(TestHash, TestBlock); err != nil {
		t.Error(err)
	}

	// Create locators for testing.
	// Turn on permission settings so we can generate signed locators.
	enforcePermissions = true
	PermissionSecret = []byte(knownKey)
	blobSignatureTTL = 300 * time.Second

	var (
		unsignedLocator  = "/" + TestHash
		validTimestamp   = time.Now().Add(blobSignatureTTL)
		expiredTimestamp = time.Now().Add(-time.Hour)
		signedLocator    = "/" + SignLocator(TestHash, knownToken, validTimestamp)
		expiredLocator   = "/" + SignLocator(TestHash, knownToken, expiredTimestamp)
	)

	// -----------------
	// Test unauthenticated request with permissions off.
	enforcePermissions = false

	// Unauthenticated request, unsigned locator
	// => OK
	response := IssueRequest(
		&RequestTester{
			method: "GET",
			uri:    unsignedLocator,
		})
	ExpectStatusCode(t,
		"Unauthenticated request, unsigned locator", http.StatusOK, response)
	ExpectBody(t,
		"Unauthenticated request, unsigned locator",
		string(TestBlock),
		response)

	receivedLen := response.Header().Get("Content-Length")
	expectedLen := fmt.Sprintf("%d", len(TestBlock))
	if receivedLen != expectedLen {
		t.Errorf("expected Content-Length %s, got %s", expectedLen, receivedLen)
	}

	// ----------------
	// Permissions: on.
	enforcePermissions = true

	// Authenticated request, signed locator
	// => OK
	response = IssueRequest(&RequestTester{
		method:   "GET",
		uri:      signedLocator,
		apiToken: knownToken,
	})
	ExpectStatusCode(t,
		"Authenticated request, signed locator", http.StatusOK, response)
	ExpectBody(t,
		"Authenticated request, signed locator", string(TestBlock), response)

	receivedLen = response.Header().Get("Content-Length")
	expectedLen = fmt.Sprintf("%d", len(TestBlock))
	if receivedLen != expectedLen {
		t.Errorf("expected Content-Length %s, got %s", expectedLen, receivedLen)
	}

	// Authenticated request, unsigned locator
	// => PermissionError
	response = IssueRequest(&RequestTester{
		method:   "GET",
		uri:      unsignedLocator,
		apiToken: knownToken,
	})
	ExpectStatusCode(t, "unsigned locator", PermissionError.HTTPCode, response)

	// Unauthenticated request, signed locator
	// => PermissionError
	response = IssueRequest(&RequestTester{
		method: "GET",
		uri:    signedLocator,
	})
	ExpectStatusCode(t,
		"Unauthenticated request, signed locator",
		PermissionError.HTTPCode, response)

	// Authenticated request, expired locator
	// => ExpiredError
	response = IssueRequest(&RequestTester{
		method:   "GET",
		uri:      expiredLocator,
		apiToken: knownToken,
	})
	ExpectStatusCode(t,
		"Authenticated request, expired locator",
		ExpiredError.HTTPCode, response)
}

// Test PutBlockHandler on the following situations:
//   - no server key
//   - with server key, authenticated request, unsigned locator
//   - with server key, unauthenticated request, unsigned locator
//
func TestPutHandler(t *testing.T) {
	defer teardown()

	// Prepare two test Keep volumes.
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	// --------------
	// No server key.

	// Unauthenticated request, no server key
	// => OK (unsigned response)
	unsignedLocator := "/" + TestHash
	response := IssueRequest(
		&RequestTester{
			method:      "PUT",
			uri:         unsignedLocator,
			requestBody: TestBlock,
		})

	ExpectStatusCode(t,
		"Unauthenticated request, no server key", http.StatusOK, response)
	ExpectBody(t,
		"Unauthenticated request, no server key",
		TestHashPutResp, response)

	// ------------------
	// With a server key.

	PermissionSecret = []byte(knownKey)
	blobSignatureTTL = 300 * time.Second

	// When a permission key is available, the locator returned
	// from an authenticated PUT request will be signed.

	// Authenticated PUT, signed locator
	// => OK (signed response)
	response = IssueRequest(
		&RequestTester{
			method:      "PUT",
			uri:         unsignedLocator,
			requestBody: TestBlock,
			apiToken:    knownToken,
		})

	ExpectStatusCode(t,
		"Authenticated PUT, signed locator, with server key",
		http.StatusOK, response)
	responseLocator := strings.TrimSpace(response.Body.String())
	if VerifySignature(responseLocator, knownToken) != nil {
		t.Errorf("Authenticated PUT, signed locator, with server key:\n"+
			"response '%s' does not contain a valid signature",
			responseLocator)
	}

	// Unauthenticated PUT, unsigned locator
	// => OK
	response = IssueRequest(
		&RequestTester{
			method:      "PUT",
			uri:         unsignedLocator,
			requestBody: TestBlock,
		})

	ExpectStatusCode(t,
		"Unauthenticated PUT, unsigned locator, with server key",
		http.StatusOK, response)
	ExpectBody(t,
		"Unauthenticated PUT, unsigned locator, with server key",
		TestHashPutResp, response)
}

func TestPutAndDeleteSkipReadonlyVolumes(t *testing.T) {
	defer teardown()
	dataManagerToken = "fake-data-manager-token"
	vols := []*MockVolume{CreateMockVolume(), CreateMockVolume()}
	vols[0].Readonly = true
	KeepVM = MakeRRVolumeManager([]Volume{vols[0], vols[1]})
	defer KeepVM.Close()
	IssueRequest(
		&RequestTester{
			method:      "PUT",
			uri:         "/" + TestHash,
			requestBody: TestBlock,
		})
	defer func(orig bool) {
		neverDelete = orig
	}(neverDelete)
	neverDelete = false
	IssueRequest(
		&RequestTester{
			method:      "DELETE",
			uri:         "/" + TestHash,
			requestBody: TestBlock,
			apiToken:    dataManagerToken,
		})
	type expect struct {
		volnum    int
		method    string
		callcount int
	}
	for _, e := range []expect{
		{0, "Get", 0},
		{0, "Compare", 0},
		{0, "Touch", 0},
		{0, "Put", 0},
		{0, "Delete", 0},
		{1, "Get", 0},
		{1, "Compare", 1},
		{1, "Touch", 1},
		{1, "Put", 1},
		{1, "Delete", 1},
	} {
		if calls := vols[e.volnum].CallCount(e.method); calls != e.callcount {
			t.Errorf("Got %d %s() on vol %d, expect %d", calls, e.method, e.volnum, e.callcount)
		}
	}
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
// superuser. They should pass regardless of the value of enforcePermissions.
//
func TestIndexHandler(t *testing.T) {
	defer teardown()

	// Set up Keep volumes and populate them.
	// Include multiple blocks on different volumes, and
	// some metadata files (which should be omitted from index listings)
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	vols := KeepVM.AllWritable()
	vols[0].Put(TestHash, TestBlock)
	vols[1].Put(TestHash2, TestBlock2)
	vols[0].Put(TestHash+".meta", []byte("metadata"))
	vols[1].Put(TestHash2+".meta", []byte("metadata"))

	dataManagerToken = "DATA MANAGER TOKEN"

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
		apiToken: dataManagerToken,
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
		apiToken: dataManagerToken,
	}
	superuserNoSuchPrefixReq := &RequestTester{
		method:   "GET",
		uri:      "/index/abcd",
		apiToken: dataManagerToken,
	}
	superuserInvalidPrefixReq := &RequestTester{
		method:   "GET",
		uri:      "/index/xyz",
		apiToken: dataManagerToken,
	}

	// -------------------------------------------------------------
	// Only the superuser should be allowed to issue /index requests.

	// ---------------------------
	// enforcePermissions enabled
	// This setting should not affect tests passing.
	enforcePermissions = true

	// unauthenticated /index request
	// => UnauthorizedError
	response := IssueRequest(unauthenticatedReq)
	ExpectStatusCode(t,
		"enforcePermissions on, unauthenticated request",
		UnauthorizedError.HTTPCode,
		response)

	// unauthenticated /index/prefix request
	// => UnauthorizedError
	response = IssueRequest(unauthPrefixReq)
	ExpectStatusCode(t,
		"permissions on, unauthenticated /index/prefix request",
		UnauthorizedError.HTTPCode,
		response)

	// authenticated /index request, non-superuser
	// => UnauthorizedError
	response = IssueRequest(authenticatedReq)
	ExpectStatusCode(t,
		"permissions on, authenticated request, non-superuser",
		UnauthorizedError.HTTPCode,
		response)

	// authenticated /index/prefix request, non-superuser
	// => UnauthorizedError
	response = IssueRequest(authPrefixReq)
	ExpectStatusCode(t,
		"permissions on, authenticated /index/prefix request, non-superuser",
		UnauthorizedError.HTTPCode,
		response)

	// superuser /index request
	// => OK
	response = IssueRequest(superuserReq)
	ExpectStatusCode(t,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	// ----------------------------
	// enforcePermissions disabled
	// Valid Request should still pass.
	enforcePermissions = false

	// superuser /index request
	// => OK
	response = IssueRequest(superuserReq)
	ExpectStatusCode(t,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	expected := `^` + TestHash + `\+\d+ \d+\n` +
		TestHash2 + `\+\d+ \d+\n\n$`
	match, _ := regexp.MatchString(expected, response.Body.String())
	if !match {
		t.Errorf(
			"permissions on, superuser request: expected %s, got:\n%s",
			expected, response.Body.String())
	}

	// superuser /index/prefix request
	// => OK
	response = IssueRequest(superuserPrefixReq)
	ExpectStatusCode(t,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	expected = `^` + TestHash + `\+\d+ \d+\n\n$`
	match, _ = regexp.MatchString(expected, response.Body.String())
	if !match {
		t.Errorf(
			"permissions on, superuser /index/prefix request: expected %s, got:\n%s",
			expected, response.Body.String())
	}

	// superuser /index/{no-such-prefix} request
	// => OK
	response = IssueRequest(superuserNoSuchPrefixReq)
	ExpectStatusCode(t,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	if "\n" != response.Body.String() {
		t.Errorf("Expected empty response for %s. Found %s", superuserNoSuchPrefixReq.uri, response.Body.String())
	}

	// superuser /index/{invalid-prefix} request
	// => StatusBadRequest
	response = IssueRequest(superuserInvalidPrefixReq)
	ExpectStatusCode(t,
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
func TestDeleteHandler(t *testing.T) {
	defer teardown()

	// Set up Keep volumes and populate them.
	// Include multiple blocks on different volumes, and
	// some metadata files (which should be omitted from index listings)
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	vols := KeepVM.AllWritable()
	vols[0].Put(TestHash, TestBlock)

	// Explicitly set the blobSignatureTTL to 0 for these
	// tests, to ensure the MockVolume deletes the blocks
	// even though they have just been created.
	blobSignatureTTL = time.Duration(0)

	var userToken = "NOT DATA MANAGER TOKEN"
	dataManagerToken = "DATA MANAGER TOKEN"

	neverDelete = false

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
		apiToken: dataManagerToken,
	}

	superuserNonexistentBlockReq := &RequestTester{
		method:   "DELETE",
		uri:      "/" + TestHash2,
		apiToken: dataManagerToken,
	}

	// Unauthenticated request returns PermissionError.
	var response *httptest.ResponseRecorder
	response = IssueRequest(unauthReq)
	ExpectStatusCode(t,
		"unauthenticated request",
		PermissionError.HTTPCode,
		response)

	// Authenticated non-admin request returns PermissionError.
	response = IssueRequest(userReq)
	ExpectStatusCode(t,
		"authenticated non-admin request",
		PermissionError.HTTPCode,
		response)

	// Authenticated admin request for nonexistent block.
	type deletecounter struct {
		Deleted int `json:"copies_deleted"`
		Failed  int `json:"copies_failed"`
	}
	var responseDc, expectedDc deletecounter

	response = IssueRequest(superuserNonexistentBlockReq)
	ExpectStatusCode(t,
		"data manager request, nonexistent block",
		http.StatusNotFound,
		response)

	// Authenticated admin request for existing block while neverDelete is set.
	neverDelete = true
	response = IssueRequest(superuserExistingBlockReq)
	ExpectStatusCode(t,
		"authenticated request, existing block, method disabled",
		MethodDisabledError.HTTPCode,
		response)
	neverDelete = false

	// Authenticated admin request for existing block.
	response = IssueRequest(superuserExistingBlockReq)
	ExpectStatusCode(t,
		"data manager request, existing block",
		http.StatusOK,
		response)
	// Expect response {"copies_deleted":1,"copies_failed":0}
	expectedDc = deletecounter{1, 0}
	json.NewDecoder(response.Body).Decode(&responseDc)
	if responseDc != expectedDc {
		t.Errorf("superuserExistingBlockReq\nexpected: %+v\nreceived: %+v",
			expectedDc, responseDc)
	}
	// Confirm the block has been deleted
	_, err := vols[0].Get(TestHash)
	var blockDeleted = os.IsNotExist(err)
	if !blockDeleted {
		t.Error("superuserExistingBlockReq: block not deleted")
	}

	// A DELETE request on a block newer than blobSignatureTTL
	// should return success but leave the block on the volume.
	vols[0].Put(TestHash, TestBlock)
	blobSignatureTTL = time.Hour

	response = IssueRequest(superuserExistingBlockReq)
	ExpectStatusCode(t,
		"data manager request, existing block",
		http.StatusOK,
		response)
	// Expect response {"copies_deleted":1,"copies_failed":0}
	expectedDc = deletecounter{1, 0}
	json.NewDecoder(response.Body).Decode(&responseDc)
	if responseDc != expectedDc {
		t.Errorf("superuserExistingBlockReq\nexpected: %+v\nreceived: %+v",
			expectedDc, responseDc)
	}
	// Confirm the block has NOT been deleted.
	_, err = vols[0].Get(TestHash)
	if err != nil {
		t.Errorf("testing delete on new block: %s\n", err)
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
func TestPullHandler(t *testing.T) {
	defer teardown()

	var userToken = "USER TOKEN"
	dataManagerToken = "DATA MANAGER TOKEN"

	pullq = NewWorkQueue()

	goodJSON := []byte(`[
		{
			"locator":"locator_with_two_servers",
			"servers":[
				"server1",
				"server2"
		 	]
		},
		{
			"locator":"locator_with_no_servers",
			"servers":[]
		},
		{
			"locator":"",
			"servers":["empty_locator"]
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
			RequestTester{"/pull", userToken, "PUT", goodJSON},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Invalid pull request from an ordinary user",
			RequestTester{"/pull", userToken, "PUT", badJSON},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Valid pull request from the data manager",
			RequestTester{"/pull", dataManagerToken, "PUT", goodJSON},
			http.StatusOK,
			"Received 3 pull requests\n",
		},
		{
			"Invalid pull request from the data manager",
			RequestTester{"/pull", dataManagerToken, "PUT", badJSON},
			http.StatusBadRequest,
			"",
		},
	}

	for _, tst := range testcases {
		response := IssueRequest(&tst.req)
		ExpectStatusCode(t, tst.name, tst.responseCode, response)
		ExpectBody(t, tst.name, tst.responseBody, response)
	}

	// The Keep pull manager should have received one good list with 3
	// requests on it.
	for i := 0; i < 3; i++ {
		item := <-pullq.NextItem
		if _, ok := item.(PullRequest); !ok {
			t.Errorf("item %v could not be parsed as a PullRequest", item)
		}
	}

	expectChannelEmpty(t, pullq.NextItem)
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
func TestTrashHandler(t *testing.T) {
	defer teardown()

	var userToken = "USER TOKEN"
	dataManagerToken = "DATA MANAGER TOKEN"

	trashq = NewWorkQueue()

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
			RequestTester{"/trash", userToken, "PUT", goodJSON},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Invalid trash list from an ordinary user",
			RequestTester{"/trash", userToken, "PUT", badJSON},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Valid trash list from the data manager",
			RequestTester{"/trash", dataManagerToken, "PUT", goodJSON},
			http.StatusOK,
			"Received 3 trash requests\n",
		},
		{
			"Invalid trash list from the data manager",
			RequestTester{"/trash", dataManagerToken, "PUT", badJSON},
			http.StatusBadRequest,
			"",
		},
	}

	for _, tst := range testcases {
		response := IssueRequest(&tst.req)
		ExpectStatusCode(t, tst.name, tst.responseCode, response)
		ExpectBody(t, tst.name, tst.responseBody, response)
	}

	// The trash collector should have received one good list with 3
	// requests on it.
	for i := 0; i < 3; i++ {
		item := <-trashq.NextItem
		if _, ok := item.(TrashRequest); !ok {
			t.Errorf("item %v could not be parsed as a TrashRequest", item)
		}
	}

	expectChannelEmpty(t, trashq.NextItem)
}

// ====================
// Helper functions
// ====================

// IssueTestRequest executes an HTTP request described by rt, to a
// REST router.  It returns the HTTP response to the request.
func IssueRequest(rt *RequestTester) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	body := bytes.NewReader(rt.requestBody)
	req, _ := http.NewRequest(rt.method, rt.uri, body)
	if rt.apiToken != "" {
		req.Header.Set("Authorization", "OAuth2 "+rt.apiToken)
	}
	loggingRouter := MakeLoggingRESTRouter()
	loggingRouter.ServeHTTP(response, req)
	return response
}

// ExpectStatusCode checks whether a response has the specified status code,
// and reports a test failure if not.
func ExpectStatusCode(
	t *testing.T,
	testname string,
	expectedStatus int,
	response *httptest.ResponseRecorder) {
	if response.Code != expectedStatus {
		t.Errorf("%s: expected status %d, got %+v",
			testname, expectedStatus, response)
	}
}

func ExpectBody(
	t *testing.T,
	testname string,
	expectedBody string,
	response *httptest.ResponseRecorder) {
	if expectedBody != "" && response.Body.String() != expectedBody {
		t.Errorf("%s: expected response body '%s', got %+v",
			testname, expectedBody, response)
	}
}

// See #7121
func TestPutNeedsOnlyOneBuffer(t *testing.T) {
	defer teardown()
	KeepVM = MakeTestVolumeManager(1)
	defer KeepVM.Close()

	defer func(orig *bufferPool) {
		bufs = orig
	}(bufs)
	bufs = newBufferPool(1, BlockSize)

	ok := make(chan struct{})
	go func() {
		for i := 0; i < 2; i++ {
			response := IssueRequest(
				&RequestTester{
					method:      "PUT",
					uri:         "/" + TestHash,
					requestBody: TestBlock,
				})
			ExpectStatusCode(t,
				"TestPutNeedsOnlyOneBuffer", http.StatusOK, response)
		}
		ok <- struct{}{}
	}()

	select {
	case <-ok:
	case <-time.After(time.Second):
		t.Fatal("PUT deadlocks with maxBuffers==1")
	}
}

// Invoke the PutBlockHandler a bunch of times to test for bufferpool resource
// leak.
func TestPutHandlerNoBufferleak(t *testing.T) {
	defer teardown()

	// Prepare two test Keep volumes.
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	ok := make(chan bool)
	go func() {
		for i := 0; i < maxBuffers+1; i++ {
			// Unauthenticated request, no server key
			// => OK (unsigned response)
			unsignedLocator := "/" + TestHash
			response := IssueRequest(
				&RequestTester{
					method:      "PUT",
					uri:         unsignedLocator,
					requestBody: TestBlock,
				})
			ExpectStatusCode(t,
				"TestPutHandlerBufferleak", http.StatusOK, response)
			ExpectBody(t,
				"TestPutHandlerBufferleak",
				TestHashPutResp, response)
		}
		ok <- true
	}()
	select {
	case <-time.After(20 * time.Second):
		// If the buffer pool leaks, the test goroutine hangs.
		t.Fatal("test did not finish, assuming pool leaked")
	case <-ok:
	}
}

// Invoke the GetBlockHandler a bunch of times to test for bufferpool resource
// leak.
func TestGetHandlerNoBufferleak(t *testing.T) {
	defer teardown()

	// Prepare two test Keep volumes. Our block is stored on the second volume.
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	vols := KeepVM.AllWritable()
	if err := vols[0].Put(TestHash, TestBlock); err != nil {
		t.Error(err)
	}

	ok := make(chan bool)
	go func() {
		for i := 0; i < maxBuffers+1; i++ {
			// Unauthenticated request, unsigned locator
			// => OK
			unsignedLocator := "/" + TestHash
			response := IssueRequest(
				&RequestTester{
					method: "GET",
					uri:    unsignedLocator,
				})
			ExpectStatusCode(t,
				"Unauthenticated request, unsigned locator", http.StatusOK, response)
			ExpectBody(t,
				"Unauthenticated request, unsigned locator",
				string(TestBlock),
				response)
		}
		ok <- true
	}()
	select {
	case <-time.After(20 * time.Second):
		// If the buffer pool leaks, the test goroutine hangs.
		t.Fatal("test did not finish, assuming pool leaked")
	case <-ok:
	}
}

func TestPutReplicationHeader(t *testing.T) {
	defer teardown()

	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()

	resp := IssueRequest(&RequestTester{
		method:      "PUT",
		uri:         "/" + TestHash,
		requestBody: TestBlock,
	})
	if r := resp.Header().Get("X-Keep-Replicas-Stored"); r != "1" {
		t.Errorf("Got X-Keep-Replicas-Stored: %q, expected %q", r, "1")
	}
}

func TestUntrashHandler(t *testing.T) {
	defer teardown()

	// Set up Keep volumes
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Close()
	vols := KeepVM.AllWritable()
	vols[0].Put(TestHash, TestBlock)

	dataManagerToken = "DATA MANAGER TOKEN"

	// unauthenticatedReq => UnauthorizedError
	unauthenticatedReq := &RequestTester{
		method: "PUT",
		uri:    "/untrash/" + TestHash,
	}
	response := IssueRequest(unauthenticatedReq)
	ExpectStatusCode(t,
		"Unauthenticated request",
		UnauthorizedError.HTTPCode,
		response)

	// notDataManagerReq => UnauthorizedError
	notDataManagerReq := &RequestTester{
		method:   "PUT",
		uri:      "/untrash/" + TestHash,
		apiToken: knownToken,
	}

	response = IssueRequest(notDataManagerReq)
	ExpectStatusCode(t,
		"Non-datamanager token",
		UnauthorizedError.HTTPCode,
		response)

	// datamanagerWithBadHashReq => StatusBadRequest
	datamanagerWithBadHashReq := &RequestTester{
		method:   "PUT",
		uri:      "/untrash/thisisnotalocator",
		apiToken: dataManagerToken,
	}
	response = IssueRequest(datamanagerWithBadHashReq)
	ExpectStatusCode(t,
		"Bad locator in untrash request",
		http.StatusBadRequest,
		response)

	// datamanagerWrongMethodReq => StatusBadRequest
	datamanagerWrongMethodReq := &RequestTester{
		method:   "GET",
		uri:      "/untrash/" + TestHash,
		apiToken: dataManagerToken,
	}
	response = IssueRequest(datamanagerWrongMethodReq)
	ExpectStatusCode(t,
		"Only PUT method is supported for untrash",
		http.StatusBadRequest,
		response)

	// datamanagerReq => StatusOK
	datamanagerReq := &RequestTester{
		method:   "PUT",
		uri:      "/untrash/" + TestHash,
		apiToken: dataManagerToken,
	}
	response = IssueRequest(datamanagerReq)
	ExpectStatusCode(t,
		"",
		http.StatusOK,
		response)
	expected := "Successfully untrashed on: [MockVolume],[MockVolume]"
	if response.Body.String() != expected {
		t.Errorf(
			"Untrash response mismatched: expected %s, got:\n%s",
			expected, response.Body.String())
	}
}

func TestUntrashHandlerWithNoWritableVolumes(t *testing.T) {
	defer teardown()

	// Set up readonly Keep volumes
	vols := []*MockVolume{CreateMockVolume(), CreateMockVolume()}
	vols[0].Readonly = true
	vols[1].Readonly = true
	KeepVM = MakeRRVolumeManager([]Volume{vols[0], vols[1]})
	defer KeepVM.Close()

	dataManagerToken = "DATA MANAGER TOKEN"

	// datamanagerReq => StatusOK
	datamanagerReq := &RequestTester{
		method:   "PUT",
		uri:      "/untrash/" + TestHash,
		apiToken: dataManagerToken,
	}
	response := IssueRequest(datamanagerReq)
	ExpectStatusCode(t,
		"No writable volumes",
		http.StatusNotFound,
		response)
}
