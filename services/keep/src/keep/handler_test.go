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
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
)

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

	// Test an unsigned GET request.
	test_url := "http://localhost:25107/" + TEST_HASH
	req, _ := http.NewRequest("GET", test_url, nil)
	resp := httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "unsigned GET", resp, http.StatusOK)
	ExpectBody(t, "unsigned GET", resp, string(TEST_BLOCK))

	// Enable permissions.
	enforce_permissions = true
	PermissionSecret = []byte(known_key)
	permission_ttl = 300
	expiry := time.Now().Add(time.Duration(permission_ttl) * time.Second)

	// Test GET with a signed locator.
	test_url = "http://localhost:25107/" + SignLocator(TEST_HASH, known_token, expiry)
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", test_url, nil)
	req.Header.Set("Authorization", "OAuth "+known_token)
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "signed GET", resp, http.StatusOK)
	ExpectBody(t, "signed GET", resp, string(TEST_BLOCK))

	// Test GET with an unsigned locator.
	test_url = "http://localhost:25107/" + TEST_HASH
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", test_url, nil)
	req.Header.Set("Authorization", "OAuth "+known_token)
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "unsigned locator", resp, PermissionError.HTTPCode)

	// Test GET with a signed locator and an unauthenticated request.
	test_url = "http://localhost:25107/" + SignLocator(TEST_HASH, known_token, expiry)
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", test_url, nil)
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "signed locator", resp, PermissionError.HTTPCode)

	// Test GET with an expired, signed locator.
	expired_ts := time.Now().Add(-time.Hour)
	test_url = "http://localhost:25107/" + SignLocator(TEST_HASH, known_token, expired_ts)
	resp = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", test_url, nil)
	req.Header.Set("Authorization", "OAuth "+known_token)
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "expired signature", resp, ExpiredError.HTTPCode)
}

func TestPutHandler(t *testing.T) {
	defer teardown()

	// Prepare two test Keep volumes.
	KeepVM = MakeTestVolumeManager(2)
	defer func() { KeepVM.Quit() }()

	// Set up a REST router for testing the handlers.
	rest := NewRESTRouter()

	// Execute a PUT request.
	test_url := "http://localhost:25107/" + TEST_HASH
	test_body := bytes.NewReader(TEST_BLOCK)
	req, _ := http.NewRequest("PUT", test_url, test_body)
	resp := httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "permissions off", resp, http.StatusOK)
	ExpectBody(t, "permissions off", resp, TEST_HASH)

	// Add a permission key.
	// When a permission key is available, the locator returned
	// from a PUT request will be signed.
	PermissionSecret = []byte(known_key)
	permission_ttl = time.Duration(300) * time.Second

	// An authenticated PUT request returns a signed locator.
	test_url = "http://localhost:25107/" + TEST_HASH
	test_body = bytes.NewReader(TEST_BLOCK)
	req, _ = http.NewRequest("PUT", test_url, test_body)
	req.Header.Set("Authorization", "OAuth "+known_token)
	resp = httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "authenticated PUT", resp, http.StatusOK)
	if !VerifySignature(resp.Body.String(), known_token) {
		t.Errorf("authenticated PUT: response '%s' failed signature check",
			resp.Body.String())
	}

	// An unauthenticated PUT request returns an unsigned locator
	// even when a permission key is available.
	test_url = "http://localhost:25107/" + TEST_HASH
	test_body = bytes.NewReader(TEST_BLOCK)
	req, _ = http.NewRequest("PUT", test_url, test_body)
	resp = httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "anon PUT with server key", resp, http.StatusOK)
	ExpectBody(t, "anon PUT with server key", resp, TEST_HASH)
}

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

	// Requests for /index with a prefix are okay even if unauthenticated.
	test_url := "http://localhost:25107/index/" + TEST_HASH[0:5]
	req, _ := http.NewRequest("GET", test_url, nil)
	resp := httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	expected := `^` + TEST_HASH + `\+\d+ \d+\n$`
	match, _ := regexp.MatchString(expected, resp.Body.String())
	if !match {
		t.Errorf("IndexHandler expected: %s, returned:\n%s",
			expected, resp.Body.String())
	}

	// Unauthenticated /index requests: fail.
	test_url = "http://localhost:25107/index"
	req, _ = http.NewRequest("GET", test_url, nil)
	resp = httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "unauthenticated /index", resp, PermissionError.HTTPCode)

	// Authenticated /index requests by a non-superuser: also fail.
	test_url = "http://localhost:25107/index"
	req, _ = http.NewRequest("GET", test_url, nil)
	req.Header.Set("Authorization", "OAuth "+known_token)
	resp = httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "authenticated /index", resp, PermissionError.HTTPCode)

	// Even superuser /index requests fail if enforce_permissions is off!
	enforce_permissions = false
	data_manager_token = "DATA MANAGER TOKEN"
	test_url = "http://localhost:25107/index"
	req, _ = http.NewRequest("GET", test_url, nil)
	req.Header.Set("Authorization", "OAuth "+data_manager_token)
	resp = httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "superuser /index (permissions off)", resp, PermissionError.HTTPCode)

	// Superuser /index requests with enforce_permissions set: succeed!
	enforce_permissions = true
	data_manager_token = "DATA MANAGER TOKEN"
	test_url = "http://localhost:25107/index"
	req, _ = http.NewRequest("GET", test_url, nil)
	req.Header.Set("Authorization", "OAuth "+data_manager_token)
	resp = httptest.NewRecorder()
	rest.ServeHTTP(resp, req)

	ExpectStatusCode(t, "superuser /index (permissions on)", resp, http.StatusOK)
	expected = `^` + TEST_HASH + `\+\d+ \d+\n` +
		TEST_HASH_2 + `\+\d+ \d+\n$`
	match, _ = regexp.MatchString(expected, resp.Body.String())
	if !match {
		t.Errorf("superuser /index: expected %s, got:\n%s",
			expected, resp.Body.String())
	}
}

// ExpectStatusCode checks whether a response has the specified status code,
// and reports a test failure if not.
func ExpectStatusCode(
	t *testing.T,
	testname string,
	response *httptest.ResponseRecorder,
	expected_status int) {
	if response.Code != expected_status {
		t.Errorf("%s: expected status %s, got %+v",
			testname, expected_status, response)
	}
}

func ExpectBody(
	t *testing.T,
	testname string,
	response *httptest.ResponseRecorder,
	expected_body string) {
	if response.Body.String() != expected_body {
		t.Errorf("%s: expected response body '%s', got %+v",
			testname, expected_body, response)
	}
}
