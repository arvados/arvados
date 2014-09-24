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
	"github.com/gorilla/mux"
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
	uri          string
	api_token    string
	method       string
	request_body []byte
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
	defer KeepVM.Quit()

	vols := KeepVM.Volumes()
	if err := vols[0].Put(TEST_HASH, TEST_BLOCK); err != nil {
		t.Error(err)
	}

	// Set up a REST router for testing the handlers.
	rest := MakeRESTRouter()

	// Create locators for testing.
	// Turn on permission settings so we can generate signed locators.
	enforce_permissions = true
	PermissionSecret = []byte(known_key)
	permission_ttl = time.Duration(300) * time.Second

	var (
		unsigned_locator  = "/" + TEST_HASH
		valid_timestamp   = time.Now().Add(permission_ttl)
		expired_timestamp = time.Now().Add(-time.Hour)
		signed_locator    = "/" + SignLocator(TEST_HASH, known_token, valid_timestamp)
		expired_locator   = "/" + SignLocator(TEST_HASH, known_token, expired_timestamp)
	)

	// -----------------
	// Test unauthenticated request with permissions off.
	enforce_permissions = false

	// Unauthenticated request, unsigned locator
	// => OK
	response := IssueRequest(rest,
		&RequestTester{
			method: "GET",
			uri:    unsigned_locator,
		})
	ExpectStatusCode(t,
		"Unauthenticated request, unsigned locator", http.StatusOK, response)
	ExpectBody(t,
		"Unauthenticated request, unsigned locator",
		string(TEST_BLOCK),
		response)
	received_xbs := response.Header().Get("X-Block-Size")
	expected_xbs := fmt.Sprintf("%d", len(TEST_BLOCK))
	if received_xbs != expected_xbs {
		t.Errorf("expected X-Block-Size %s, got %s", expected_xbs, received_xbs)
	}

	// ----------------
	// Permissions: on.
	enforce_permissions = true

	// Authenticated request, signed locator
	// => OK
	response = IssueRequest(rest, &RequestTester{
		method:    "GET",
		uri:       signed_locator,
		api_token: known_token,
	})
	ExpectStatusCode(t,
		"Authenticated request, signed locator", http.StatusOK, response)
	ExpectBody(t,
		"Authenticated request, signed locator", string(TEST_BLOCK), response)
	received_xbs = response.Header().Get("X-Block-Size")
	expected_xbs = fmt.Sprintf("%d", len(TEST_BLOCK))
	if received_xbs != expected_xbs {
		t.Errorf("expected X-Block-Size %s, got %s", expected_xbs, received_xbs)
	}

	// Authenticated request, unsigned locator
	// => PermissionError
	response = IssueRequest(rest, &RequestTester{
		method:    "GET",
		uri:       unsigned_locator,
		api_token: known_token,
	})
	ExpectStatusCode(t, "unsigned locator", PermissionError.HTTPCode, response)

	// Unauthenticated request, signed locator
	// => PermissionError
	response = IssueRequest(rest, &RequestTester{
		method: "GET",
		uri:    signed_locator,
	})
	ExpectStatusCode(t,
		"Unauthenticated request, signed locator",
		PermissionError.HTTPCode, response)

	// Authenticated request, expired locator
	// => ExpiredError
	response = IssueRequest(rest, &RequestTester{
		method:    "GET",
		uri:       expired_locator,
		api_token: known_token,
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
	defer KeepVM.Quit()

	// Set up a REST router for testing the handlers.
	rest := MakeRESTRouter()

	// --------------
	// No server key.

	// Unauthenticated request, no server key
	// => OK (unsigned response)
	unsigned_locator := "/" + TEST_HASH
	response := IssueRequest(rest,
		&RequestTester{
			method:       "PUT",
			uri:          unsigned_locator,
			request_body: TEST_BLOCK,
		})

	ExpectStatusCode(t,
		"Unauthenticated request, no server key", http.StatusOK, response)
	ExpectBody(t,
		"Unauthenticated request, no server key",
		TEST_HASH_PUT_RESPONSE, response)

	// ------------------
	// With a server key.

	PermissionSecret = []byte(known_key)
	permission_ttl = time.Duration(300) * time.Second

	// When a permission key is available, the locator returned
	// from an authenticated PUT request will be signed.

	// Authenticated PUT, signed locator
	// => OK (signed response)
	response = IssueRequest(rest,
		&RequestTester{
			method:       "PUT",
			uri:          unsigned_locator,
			request_body: TEST_BLOCK,
			api_token:    known_token,
		})

	ExpectStatusCode(t,
		"Authenticated PUT, signed locator, with server key",
		http.StatusOK, response)
	response_locator := strings.TrimSpace(response.Body.String())
	if !VerifySignature(response_locator, known_token) {
		t.Errorf("Authenticated PUT, signed locator, with server key:\n"+
			"response '%s' does not contain a valid signature",
			response_locator)
	}

	// Unauthenticated PUT, unsigned locator
	// => OK
	response = IssueRequest(rest,
		&RequestTester{
			method:       "PUT",
			uri:          unsigned_locator,
			request_body: TEST_BLOCK,
		})

	ExpectStatusCode(t,
		"Unauthenticated PUT, unsigned locator, with server key",
		http.StatusOK, response)
	ExpectBody(t,
		"Unauthenticated PUT, unsigned locator, with server key",
		TEST_HASH_PUT_RESPONSE, response)
}

// Test /index requests:
//   - enforce_permissions off | unauthenticated /index request
//   - enforce_permissions off | unauthenticated /index/prefix request
//   - enforce_permissions off | authenticated /index request        | non-superuser
//   - enforce_permissions off | authenticated /index/prefix request | non-superuser
//   - enforce_permissions off | authenticated /index request        | superuser
//   - enforce_permissions off | authenticated /index/prefix request | superuser
//   - enforce_permissions on  | unauthenticated /index request
//   - enforce_permissions on  | unauthenticated /index/prefix request
//   - enforce_permissions on  | authenticated /index request        | non-superuser
//   - enforce_permissions on  | authenticated /index/prefix request | non-superuser
//   - enforce_permissions on  | authenticated /index request        | superuser
//   - enforce_permissions on  | authenticated /index/prefix request | superuser
//
// The only /index requests that should succeed are those issued by the
// superuser when enforce_permissions = true.
//
func TestIndexHandler(t *testing.T) {
	defer teardown()

	// Set up Keep volumes and populate them.
	// Include multiple blocks on different volumes, and
	// some metadata files (which should be omitted from index listings)
	KeepVM = MakeTestVolumeManager(2)
	defer KeepVM.Quit()

	vols := KeepVM.Volumes()
	vols[0].Put(TEST_HASH, TEST_BLOCK)
	vols[1].Put(TEST_HASH_2, TEST_BLOCK_2)
	vols[0].Put(TEST_HASH+".meta", []byte("metadata"))
	vols[1].Put(TEST_HASH_2+".meta", []byte("metadata"))

	// Set up a REST router for testing the handlers.
	rest := MakeRESTRouter()

	data_manager_token = "DATA MANAGER TOKEN"

	unauthenticated_req := &RequestTester{
		method: "GET",
		uri:    "/index",
	}
	authenticated_req := &RequestTester{
		method:    "GET",
		uri:       "/index",
		api_token: known_token,
	}
	superuser_req := &RequestTester{
		method:    "GET",
		uri:       "/index",
		api_token: data_manager_token,
	}
	unauth_prefix_req := &RequestTester{
		method: "GET",
		uri:    "/index/" + TEST_HASH[0:3],
	}
	auth_prefix_req := &RequestTester{
		method:    "GET",
		uri:       "/index/" + TEST_HASH[0:3],
		api_token: known_token,
	}
	superuser_prefix_req := &RequestTester{
		method:    "GET",
		uri:       "/index/" + TEST_HASH[0:3],
		api_token: data_manager_token,
	}

	// ----------------------------
	// enforce_permissions disabled
	// All /index requests should fail.
	enforce_permissions = false

	// unauthenticated /index request
	// => PermissionError
	response := IssueRequest(rest, unauthenticated_req)
	ExpectStatusCode(t,
		"enforce_permissions off, unauthenticated request",
		PermissionError.HTTPCode,
		response)

	// unauthenticated /index/prefix request
	// => PermissionError
	response = IssueRequest(rest, unauth_prefix_req)
	ExpectStatusCode(t,
		"enforce_permissions off, unauthenticated /index/prefix request",
		PermissionError.HTTPCode,
		response)

	// authenticated /index request, non-superuser
	// => PermissionError
	response = IssueRequest(rest, authenticated_req)
	ExpectStatusCode(t,
		"enforce_permissions off, authenticated request, non-superuser",
		PermissionError.HTTPCode,
		response)

	// authenticated /index/prefix request, non-superuser
	// => PermissionError
	response = IssueRequest(rest, auth_prefix_req)
	ExpectStatusCode(t,
		"enforce_permissions off, authenticated /index/prefix request, non-superuser",
		PermissionError.HTTPCode,
		response)

	// authenticated /index request, superuser
	// => PermissionError
	response = IssueRequest(rest, superuser_req)
	ExpectStatusCode(t,
		"enforce_permissions off, superuser request",
		PermissionError.HTTPCode,
		response)

	// superuser /index/prefix request
	// => PermissionError
	response = IssueRequest(rest, superuser_prefix_req)
	ExpectStatusCode(t,
		"enforce_permissions off, superuser /index/prefix request",
		PermissionError.HTTPCode,
		response)

	// ---------------------------
	// enforce_permissions enabled
	// Only the superuser should be allowed to issue /index requests.
	enforce_permissions = true

	// unauthenticated /index request
	// => PermissionError
	response = IssueRequest(rest, unauthenticated_req)
	ExpectStatusCode(t,
		"enforce_permissions on, unauthenticated request",
		PermissionError.HTTPCode,
		response)

	// unauthenticated /index/prefix request
	// => PermissionError
	response = IssueRequest(rest, unauth_prefix_req)
	ExpectStatusCode(t,
		"permissions on, unauthenticated /index/prefix request",
		PermissionError.HTTPCode,
		response)

	// authenticated /index request, non-superuser
	// => PermissionError
	response = IssueRequest(rest, authenticated_req)
	ExpectStatusCode(t,
		"permissions on, authenticated request, non-superuser",
		PermissionError.HTTPCode,
		response)

	// authenticated /index/prefix request, non-superuser
	// => PermissionError
	response = IssueRequest(rest, auth_prefix_req)
	ExpectStatusCode(t,
		"permissions on, authenticated /index/prefix request, non-superuser",
		PermissionError.HTTPCode,
		response)

	// superuser /index request
	// => OK
	response = IssueRequest(rest, superuser_req)
	ExpectStatusCode(t,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	expected := `^` + TEST_HASH + `\+\d+ \d+\n` +
		TEST_HASH_2 + `\+\d+ \d+\n$`
	match, _ := regexp.MatchString(expected, response.Body.String())
	if !match {
		t.Errorf(
			"permissions on, superuser request: expected %s, got:\n%s",
			expected, response.Body.String())
	}

	// superuser /index/prefix request
	// => OK
	response = IssueRequest(rest, superuser_prefix_req)
	ExpectStatusCode(t,
		"permissions on, superuser request",
		http.StatusOK,
		response)

	expected = `^` + TEST_HASH + `\+\d+ \d+\n$`
	match, _ = regexp.MatchString(expected, response.Body.String())
	if !match {
		t.Errorf(
			"permissions on, superuser /index/prefix request: expected %s, got:\n%s",
			expected, response.Body.String())
	}
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
	defer KeepVM.Quit()

	vols := KeepVM.Volumes()
	vols[0].Put(TEST_HASH, TEST_BLOCK)

	// Explicitly set the permission_ttl to 0 for these
	// tests, to ensure the MockVolume deletes the blocks
	// even though they have just been created.
	permission_ttl = time.Duration(0)

	// Set up a REST router for testing the handlers.
	rest := MakeRESTRouter()

	var user_token = "NOT DATA MANAGER TOKEN"
	data_manager_token = "DATA MANAGER TOKEN"

	unauth_req := &RequestTester{
		method: "DELETE",
		uri:    "/" + TEST_HASH,
	}

	user_req := &RequestTester{
		method:    "DELETE",
		uri:       "/" + TEST_HASH,
		api_token: user_token,
	}

	superuser_existing_block_req := &RequestTester{
		method:    "DELETE",
		uri:       "/" + TEST_HASH,
		api_token: data_manager_token,
	}

	superuser_nonexistent_block_req := &RequestTester{
		method:    "DELETE",
		uri:       "/" + TEST_HASH_2,
		api_token: data_manager_token,
	}

	// Unauthenticated request returns PermissionError.
	var response *httptest.ResponseRecorder
	response = IssueRequest(rest, unauth_req)
	ExpectStatusCode(t,
		"unauthenticated request",
		PermissionError.HTTPCode,
		response)

	// Authenticated non-admin request returns PermissionError.
	response = IssueRequest(rest, user_req)
	ExpectStatusCode(t,
		"authenticated non-admin request",
		PermissionError.HTTPCode,
		response)

	// Authenticated admin request for nonexistent block.
	type deletecounter struct {
		Deleted int `json:"copies_deleted"`
		Failed  int `json:"copies_failed"`
	}
	var response_dc, expected_dc deletecounter

	response = IssueRequest(rest, superuser_nonexistent_block_req)
	ExpectStatusCode(t,
		"data manager request, nonexistent block",
		http.StatusNotFound,
		response)

	// Authenticated admin request for existing block while never_delete is set.
	never_delete = true
	response = IssueRequest(rest, superuser_existing_block_req)
	ExpectStatusCode(t,
		"authenticated request, existing block, method disabled",
		MethodDisabledError.HTTPCode,
		response)
	never_delete = false

	// Authenticated admin request for existing block.
	response = IssueRequest(rest, superuser_existing_block_req)
	ExpectStatusCode(t,
		"data manager request, existing block",
		http.StatusOK,
		response)
	// Expect response {"copies_deleted":1,"copies_failed":0}
	expected_dc = deletecounter{1, 0}
	json.NewDecoder(response.Body).Decode(&response_dc)
	if response_dc != expected_dc {
		t.Errorf("superuser_existing_block_req\nexpected: %+v\nreceived: %+v",
			expected_dc, response_dc)
	}
	// Confirm the block has been deleted
	_, err := vols[0].Get(TEST_HASH)
	var block_deleted = os.IsNotExist(err)
	if !block_deleted {
		t.Error("superuser_existing_block_req: block not deleted")
	}

	// A DELETE request on a block newer than permission_ttl should return
	// success but leave the block on the volume.
	vols[0].Put(TEST_HASH, TEST_BLOCK)
	permission_ttl = time.Duration(1) * time.Hour

	response = IssueRequest(rest, superuser_existing_block_req)
	ExpectStatusCode(t,
		"data manager request, existing block",
		http.StatusOK,
		response)
	// Expect response {"copies_deleted":1,"copies_failed":0}
	expected_dc = deletecounter{1, 0}
	json.NewDecoder(response.Body).Decode(&response_dc)
	if response_dc != expected_dc {
		t.Errorf("superuser_existing_block_req\nexpected: %+v\nreceived: %+v",
			expected_dc, response_dc)
	}
	// Confirm the block has NOT been deleted.
	_, err = vols[0].Get(TEST_HASH)
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

	// Set up a REST router for testing the handlers.
	rest := MakeRESTRouter()

	var user_token = "USER TOKEN"
	data_manager_token = "DATA MANAGER TOKEN"

	good_json := []byte(`[
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

	bad_json := []byte(`{ "key":"I'm a little teapot" }`)

	type pullTest struct {
		name          string
		req           RequestTester
		response_code int
		response_body string
	}
	var testcases = []pullTest{
		{
			"Valid pull list from an ordinary user",
			RequestTester{"/pull", user_token, "PUT", good_json},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Invalid pull request from an ordinary user",
			RequestTester{"/pull", user_token, "PUT", bad_json},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Valid pull request from the data manager",
			RequestTester{"/pull", data_manager_token, "PUT", good_json},
			http.StatusOK,
			"Received 3 pull requests\n",
		},
		{
			"Invalid pull request from the data manager",
			RequestTester{"/pull", data_manager_token, "PUT", bad_json},
			http.StatusBadRequest,
			"Bad Request\n",
		},
	}

	for _, tst := range testcases {
		response := IssueRequest(rest, &tst.req)
		ExpectStatusCode(t, tst.name, tst.response_code, response)
		ExpectBody(t, tst.name, tst.response_body, response)
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

	// Set up a REST router for testing the handlers.
	rest := MakeRESTRouter()

	var user_token = "USER TOKEN"
	data_manager_token = "DATA MANAGER TOKEN"

	good_json := []byte(`[
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

	bad_json := []byte(`I am not a valid JSON string`)

	type trashTest struct {
		name          string
		req           RequestTester
		response_code int
		response_body string
	}

	var testcases = []trashTest{
		{
			"Valid trash list from an ordinary user",
			RequestTester{"/trash", user_token, "PUT", good_json},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Invalid trash list from an ordinary user",
			RequestTester{"/trash", user_token, "PUT", bad_json},
			http.StatusUnauthorized,
			"Unauthorized\n",
		},
		{
			"Valid trash list from the data manager",
			RequestTester{"/trash", data_manager_token, "PUT", good_json},
			http.StatusOK,
			"Received 3 trash requests\n",
		},
		{
			"Invalid trash list from the data manager",
			RequestTester{"/trash", data_manager_token, "PUT", bad_json},
			http.StatusBadRequest,
			"Bad Request\n",
		},
	}

	for _, tst := range testcases {
		response := IssueRequest(rest, &tst.req)
		ExpectStatusCode(t, tst.name, tst.response_code, response)
		ExpectBody(t, tst.name, tst.response_body, response)
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
// specified REST router.  It returns the HTTP response to the request.
func IssueRequest(router *mux.Router, rt *RequestTester) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	body := bytes.NewReader(rt.request_body)
	req, _ := http.NewRequest(rt.method, rt.uri, body)
	if rt.api_token != "" {
		req.Header.Set("Authorization", "OAuth2 "+rt.api_token)
	}
	router.ServeHTTP(response, req)
	return response
}

// ExpectStatusCode checks whether a response has the specified status code,
// and reports a test failure if not.
func ExpectStatusCode(
	t *testing.T,
	testname string,
	expected_status int,
	response *httptest.ResponseRecorder) {
	if response.Code != expected_status {
		t.Errorf("%s: expected status %s, got %+v",
			testname, expected_status, response)
	}
}

func ExpectBody(
	t *testing.T,
	testname string,
	expected_body string,
	response *httptest.ResponseRecorder) {
	if response.Body.String() != expected_body {
		t.Errorf("%s: expected response body '%s', got %+v",
			testname, expected_body, response)
	}
}
