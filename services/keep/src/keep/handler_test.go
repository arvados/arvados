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
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"regexp"
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
	defer func() { KeepVM.Quit() }()

	vols := KeepVM.Volumes()
	if err := vols[0].Put(TEST_HASH, TEST_BLOCK); err != nil {
		t.Error(err)
	}

	// Set up a REST router for testing the handlers.
	rest := NewRESTRouter()

	// Create locators for testing.
	// Turn on permission settings so we can generate signed locators.
	enforce_permissions = true
	PermissionSecret = []byte(known_key)
	permission_ttl = time.Duration(300) * time.Second

	var (
		unsigned_locator  = "http://localhost:25107/" + TEST_HASH
		valid_timestamp   = time.Now().Add(permission_ttl)
		expired_timestamp = time.Now().Add(-time.Hour)
		signed_locator    = "http://localhost:25107/" + SignLocator(TEST_HASH, known_token, valid_timestamp)
		expired_locator   = "http://localhost:25107/" + SignLocator(TEST_HASH, known_token, expired_timestamp)
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
	defer func() { KeepVM.Quit() }()

	// Set up a REST router for testing the handlers.
	rest := NewRESTRouter()

	// --------------
	// No server key.

	// Unauthenticated request, no server key
	// => OK (unsigned response)
	unsigned_locator := "http://localhost:25107/" + TEST_HASH
	response := IssueRequest(rest,
		&RequestTester{
			method:       "PUT",
			uri:          unsigned_locator,
			request_body: TEST_BLOCK,
		})

	ExpectStatusCode(t,
		"Unauthenticated request, no server key", http.StatusOK, response)
	ExpectBody(t, "Unauthenticated request, no server key", TEST_HASH, response)

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
	if !VerifySignature(response.Body.String(), known_token) {
		t.Errorf("Authenticated PUT, signed locator, with server key:\n"+
			"response '%s' does not contain a valid signature",
			response.Body.String())
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
		TEST_HASH, response)
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
	defer func() { KeepVM.Quit() }()

	vols := KeepVM.Volumes()
	vols[0].Put(TEST_HASH, TEST_BLOCK)
	vols[1].Put(TEST_HASH_2, TEST_BLOCK_2)
	vols[0].Put(TEST_HASH+".meta", []byte("metadata"))
	vols[1].Put(TEST_HASH_2+".meta", []byte("metadata"))

	// Set up a REST router for testing the handlers.
	rest := NewRESTRouter()

	data_manager_token = "DATA MANAGER TOKEN"

	unauthenticated_req := &RequestTester{
		method: "GET",
		uri:    "http://localhost:25107/index",
	}
	authenticated_req := &RequestTester{
		method:    "GET",
		uri:       "http://localhost:25107/index",
		api_token: known_token,
	}
	superuser_req := &RequestTester{
		method:    "GET",
		uri:       "http://localhost:25107/index",
		api_token: data_manager_token,
	}
	unauth_prefix_req := &RequestTester{
		method: "GET",
		uri:    "http://localhost:25107/index/" + TEST_HASH[0:3],
	}
	auth_prefix_req := &RequestTester{
		method:    "GET",
		uri:       "http://localhost:25107/index/" + TEST_HASH[0:3],
		api_token: known_token,
	}
	superuser_prefix_req := &RequestTester{
		method:    "GET",
		uri:       "http://localhost:25107/index/" + TEST_HASH[0:3],
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
		req.Header.Set("Authorization", "OAuth "+rt.api_token)
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
