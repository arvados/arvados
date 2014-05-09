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

	// -----------------
	// Permissions: off.

	// Unauthenticated request, unsigned locator
	// => OK
	unsigned_locator := "http://localhost:25107/" + TEST_HASH
	response := IssueRequest(rest,
		&RequestTester{
			method: "GET",
			uri:    unsigned_locator,
		})
	ExpectStatusCode(t, "unsigned GET (permissions off)", http.StatusOK, response)
	ExpectBody(t, "unsigned GET (permissions off)", string(TEST_BLOCK), response)

	// ----------------
	// Permissions: on.

	// Create signed and expired locators for testing.
	enforce_permissions = true
	PermissionSecret = []byte(known_key)
	permission_ttl = time.Duration(300) * time.Second

	var (
		expiration        = time.Now().Add(permission_ttl)
		expired_timestamp = time.Now().Add(-time.Hour)
		signed_locator    = "http://localhost:25107/" + SignLocator(TEST_HASH, known_token, expiration)
		expired_locator   = "http://localhost:25107/" + SignLocator(TEST_HASH, known_token, expired_timestamp)
	)

	// Authenticated request, signed locator
	// => OK
	response = IssueRequest(rest, &RequestTester{
		method:    "GET",
		uri:       signed_locator,
		api_token: known_token,
	})
	ExpectStatusCode(t, "signed GET (permissions on)", http.StatusOK, response)
	ExpectBody(t, "signed GET (permissions on)", string(TEST_BLOCK), response)

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
	ExpectStatusCode(t, "signed locator", PermissionError.HTTPCode, response)

	// Authenticated request, expired locator
	// => ExpiredError
	response = IssueRequest(rest, &RequestTester{
		method:    "GET",
		uri:       expired_locator,
		api_token: known_token,
	})
	ExpectStatusCode(t, "expired signature", ExpiredError.HTTPCode, response)
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
		"unauthenticated PUT (no server key)", http.StatusOK, response)
	ExpectBody(t, "unauthenticated PUT (no server key)", TEST_HASH, response)

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
		"authenticated PUT (with server key)", http.StatusOK, response)
	if !VerifySignature(response.Body.String(), known_token) {
		t.Errorf("authenticated PUT (with server key): response '%s' does not contain a valid signature",
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
		"unauthenticated PUT (with server key)", http.StatusOK, response)
	ExpectBody(t,
		"unauthenticated PUT (with server key)", TEST_HASH, response)
}

// Test /index requests:
//   - unauthenticated /index/{prefix} request
//   - unauthenticated /index request
//   - authenticated /index request, non-superuser
//   - authenticated /index request by superuser, enforce_permissions = false
//   - authenticated /index request by superuser, enforce_permissions = true
//
func TestIndexHandler(t *testing.T) {
	defer teardown()

	// Set up Keep volumes and populate them.
	// Include multiple blocks on different volumes, and
	// some metadata files.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	vols := KeepVM.Volumes()
	vols[0].Put(TEST_HASH, TEST_BLOCK)
	vols[1].Put(TEST_HASH_2, TEST_BLOCK_2)

	// Set up a REST router for testing the handlers.
	rest := NewRESTRouter()

	// Unauthenticated /index/{prefix}
	// => OK
	response := IssueRequest(rest,
		&RequestTester{
			method: "GET",
			uri:    "http://localhost:25107/index/" + TEST_HASH[0:5],
		})

	expected := `^` + TEST_HASH + `\+\d+ \d+\n$`
	match, _ := regexp.MatchString(expected, response.Body.String())
	if !match {
		t.Errorf("IndexHandler expected: %s, returned:\n%s",
			expected, response.Body.String())
	}

	// Unauthenticated /index
	// => PermissionError
	response = IssueRequest(rest,
		&RequestTester{
			method: "GET",
			uri:    "http://localhost:25107/index",
		})

	ExpectStatusCode(t,
		"unauthenticated /index", PermissionError.HTTPCode, response)

	// Authenticated /index request by non-superuser
	// => PermissionError
	response = IssueRequest(rest,
		&RequestTester{
			method:    "GET",
			uri:       "http://localhost:25107/index",
			api_token: known_token,
		})

	ExpectStatusCode(t,
		"authenticated /index by non-superuser",
		PermissionError.HTTPCode,
		response)

	// Authenticated /index request by superuser, enforce_permissions = false
	// => PermissionError
	enforce_permissions = false
	data_manager_token = "DATA MANAGER TOKEN"

	response = IssueRequest(rest,
		&RequestTester{
			method:    "GET",
			uri:       "http://localhost:25107/index",
			api_token: data_manager_token,
		})

	ExpectStatusCode(t,
		"authenticated /index request by superuser (permissions off)",
		PermissionError.HTTPCode,
		response)

	// Authenticated /index request by superuser, enforce_permissions = true
	// => OK
	enforce_permissions = true
	response = IssueRequest(rest,
		&RequestTester{
			method:    "GET",
			uri:       "http://localhost:25107/index",
			api_token: data_manager_token,
		})

	ExpectStatusCode(t,
		"authenticated /index request by superuser (permissions on)",
		http.StatusOK,
		response)

	expected = `^` + TEST_HASH + `\+\d+ \d+\n` +
		TEST_HASH_2 + `\+\d+ \d+\n$`
	match, _ = regexp.MatchString(expected, response.Body.String())
	if !match {
		t.Errorf("superuser /index: expected %s, got:\n%s",
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
