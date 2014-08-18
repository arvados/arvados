package main

// Test methods defined in api_client.go.
//
// These tests launch a fake API server in a goroutine. The fake API
// server only knows how to return a few predefined responses.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Define API tokens for testing:
//   * an administrator token with scope "all"
//   * an administrator token without scope "all"
//   * an unprivileged token with scope "all"
//   * an unprivileged token without scope "all"
//
var api_token = map[string]string{
	"admin_allscope": "admin_with_all_scopes",
	"admin_badscope": "admin_without_all_scope",
	"admin_noscope":  "admin_with_no_scope",
	"user_allscope":  "unprivileged_user_with_all_scope",
	"user_badscope":  "unprivileged_user_with_bad_scope",
	"user_noscope":   "unprivileged_user_with_no_scope",
}

// Canned responses for the fake API server.  If the
// token and path match a request's Authorization header
// and URI path, then the response from the server will
// use the corresponding HTTP status and response body.
//
type APIRoute struct {
	token    string
	path     string
	status   int
	response string
}

var DefaultFakeAPIRoutes = []APIRoute{
	// /users/current requests
	// admin_* tokens return {"is_admin":true}
	// user_* tokens return {"is_admin":false}
	{
		token:    api_token["admin_allscope"],
		path:     "/arvados/v1/users/current",
		status:   http.StatusOK,
		response: `{"is_admin":true}`,
	},
	{
		token:    api_token["admin_badscope"],
		path:     "/arvados/v1/users/current",
		status:   http.StatusOK,
		response: `{"is_admin":true}`,
	},
	{
		token:    api_token["admin_noscope"],
		path:     "/arvados/v1/users/current",
		status:   http.StatusOK,
		response: `{"is_admin":true}`,
	},
	{
		token:    api_token["user_allscope"],
		path:     "/arvados/v1/users/current",
		status:   http.StatusOK,
		response: `{"is_admin":false}`,
	},
	{
		token:    api_token["user_badscope"],
		path:     "/arvados/v1/users/current",
		status:   http.StatusOK,
		response: `{"is_admin":false}`,
	},
	{
		token:    api_token["user_noscope"],
		path:     "/arvados/v1/users/current",
		status:   http.StatusOK,
		response: `{"is_admin":false}`,
	},
	// api_client_authorizations
	// *_allscope tokens get a response with "scopes":["all"].
	// *_badscope tokens get a response with "scopes" including something other than "all".
	// *_noscope tokens have no "scopes" field in the response at all.
	{
		token:    api_token["admin_allscope"],
		path:     "/arvados/v1/api_client_authorizations/" + api_token["admin_allscope"],
		status:   http.StatusOK,
		response: `{"uuid":"admin_allscope","scopes":["all"]}`,
	},
	{
		token:    api_token["admin_badscope"],
		path:     "/arvados/v1/api_client_authorizations/" + api_token["admin_badscope"],
		status:   http.StatusOK,
		response: `{"uuid":"admin_badscope","scopes":["GET /arvados/v1/collections/"]}`,
	},
	{
		token:    api_token["admin_noscope"],
		path:     "/arvados/v1/api_client_authorizations/" + api_token["admin_noscope"],
		status:   http.StatusOK,
		response: `{"uuid":"admin_noscope"}`,
	},
	{
		token:    api_token["user_allscope"],
		path:     "/arvados/v1/api_client_authorizations/" + api_token["user_allscope"],
		status:   http.StatusOK,
		response: `{"uuid":"user_allscope","scopes":["all"]}`,
	},
	{
		token:    api_token["user_badscope"],
		path:     "/arvados/v1/api_client_authorizations/" + api_token["user_badscope"],
		status:   http.StatusOK,
		response: `{"uuid":"user_badscope","scopes":["GET /arvados/v1/collections/"]}`,
	},
	{
		token:    api_token["user_noscope"],
		path:     "/arvados/v1/api_client_authorizations/" + api_token["user_noscope"],
		status:   http.StatusOK,
		response: `{"uuid":"user_noscope"}`,
	},
}

func TestIsAdmin(t *testing.T) {
	ts := MakeFakeAPIServer(DefaultFakeAPIRoutes)
	ts.StartTLS()
	defer ts.Close()

	os.Setenv("ARVADOS_API_HOST", ts.Listener.Addr().String())
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	expected_results := map[string]bool{
		"admin_allscope": true,
		"admin_badscope": true,
		"admin_noscope":  true,
		"user_allscope":  false,
		"user_badscope":  false,
		"user_noscope":   false,
	}

	for test, token := range api_token {
		result := IsAdmin(token)
		if result != expected_results[test] {
			t.Errorf("%s: expected %v, got %v\n",
				token, expected_results[test], result)
		}
	}
}

func TestHasUnlimitedScope(t *testing.T) {
	ts := MakeFakeAPIServer(DefaultFakeAPIRoutes)
	ts.StartTLS()
	defer ts.Close()

	os.Setenv("ARVADOS_API_HOST", ts.Listener.Addr().String())
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	expected_results := map[string]bool{
		"admin_allscope": true,
		"admin_badscope": false,
		"admin_noscope":  false,
		"user_allscope":  true,
		"user_badscope":  false,
		"user_noscope":   false,
	}

	for test, token := range api_token {
		result := HasUnlimitedScope(token)
		if result != expected_results[test] {
			t.Errorf("%s: expected %v, got %v\n",
				token, expected_results[test], result)
		}
	}
}

// TestTokenCache exercises the API token cache.  It issues a token
// lookup to fill the cache, then tests that a second lookup retrieves
// the data from cache.
//
func TestTokenCache(t *testing.T) {
	cache_test_token := "cache_test_token"
	// Default token_cache_ttl = 1 hour (plenty long for unit test)
	token_cache_ttl = 3600
	testroutes := append(DefaultFakeAPIRoutes,
		APIRoute{
			token:    cache_test_token,
			path:     "/arvados/v1/users/current",
			status:   http.StatusOK,
			response: `{"is_admin":true}`,
		},
		APIRoute{
			token:    cache_test_token,
			path:     "/arvados/v1/api_client_authorizations/" + cache_test_token,
			status:   http.StatusOK,
			response: `{"uuid":"cache_test_token","scopes":["all"]}`,
		})

	ts := MakeFakeAPIServer(testroutes)
	ts.StartTLS()
	defer ts.Close()

	os.Setenv("ARVADOS_API_HOST", ts.Listener.Addr().String())
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	// Fill the cache.
	// Use a token belonging to an admin to ensure that
	// IsAdmin(cache_test_token) will return true.
	//
	uncached_result := IsAdmin(cache_test_token)
	if uncached_result != true {
		t.Errorf("%s: expected %v\nreceived %v",
			cache_test_token, true, uncached_result)
	}

	// Restart the server, but without the cache_test_token.  When
	// IsAdmin is called, it should return the cached value even
	// though the API server no longer knows about cache_test_token.
	ts.Close()

	ts = MakeFakeAPIServer(DefaultFakeAPIRoutes)
	ts.StartTLS()

	os.Setenv("ARVADOS_API_HOST", ts.Listener.Addr().String())
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	cached_result := IsAdmin(cache_test_token)
	if cached_result != true {
		t.Errorf("%s: expected %v\nreceived %v",
			cache_test_token, true, cached_result)
	}
}

func MakeFakeAPIServer(routes []APIRoute) *httptest.Server {
	var apihandler = http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		tok := GetApiToken(req)
		for _, route := range routes {
			if route.token == tok && route.path == req.URL.Path {
				resp.WriteHeader(route.status)
				resp.Write([]byte(route.response))
				return
			}
		}
		http.Error(resp, "Internal server error", http.StatusInternalServerError)
	})

	return httptest.NewUnstartedServer(apihandler)
}
