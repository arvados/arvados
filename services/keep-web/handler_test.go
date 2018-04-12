// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&UnitSuite{})

type UnitSuite struct{}

func (s *UnitSuite) TestCORSPreflight(c *check.C) {
	h := handler{Config: DefaultConfig()}
	u, _ := url.Parse("http://keep-web.example/c=" + arvadostest.FooCollection + "/foo")
	req := &http.Request{
		Method:     "OPTIONS",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Origin":                        {"https://workbench.example"},
			"Access-Control-Request-Method": {"POST"},
		},
	}

	// Check preflight for an allowed request
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "")
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")
	c.Check(resp.Header().Get("Access-Control-Allow-Methods"), check.Equals, "COPY, DELETE, GET, MKCOL, MOVE, OPTIONS, POST, PROPFIND, PUT, RMCOL")
	c.Check(resp.Header().Get("Access-Control-Allow-Headers"), check.Equals, "Authorization, Content-Type, Range")

	// Check preflight for a disallowed request
	resp = httptest.NewRecorder()
	req.Header.Set("Access-Control-Request-Method", "MAKE-COFFEE")
	h.ServeHTTP(resp, req)
	c.Check(resp.Body.String(), check.Equals, "")
	c.Check(resp.Code, check.Equals, http.StatusMethodNotAllowed)
}

func (s *UnitSuite) TestInvalidUUID(c *check.C) {
	bogusID := strings.Replace(arvadostest.FooPdh, "+", "-", 1) + "-"
	token := arvadostest.ActiveToken
	for _, trial := range []string{
		"http://keep-web/c=" + bogusID + "/foo",
		"http://keep-web/c=" + bogusID + "/t=" + token + "/foo",
		"http://keep-web/collections/download/" + bogusID + "/" + token + "/foo",
		"http://keep-web/collections/" + bogusID + "/foo",
		"http://" + bogusID + ".keep-web/" + bogusID + "/foo",
		"http://" + bogusID + ".keep-web/t=" + token + "/" + bogusID + "/foo",
	} {
		c.Log(trial)
		u, err := url.Parse(trial)
		c.Assert(err, check.IsNil)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
		}
		resp := httptest.NewRecorder()
		cfg := DefaultConfig()
		cfg.AnonymousTokens = []string{arvadostest.AnonymousToken}
		h := handler{Config: cfg}
		h.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusNotFound)
	}
}

func mustParseURL(s string) *url.URL {
	r, err := url.Parse(s)
	if err != nil {
		panic("parse URL: " + s)
	}
	return r
}

func (s *IntegrationSuite) TestVhost404(c *check.C) {
	for _, testURL := range []string{
		arvadostest.NonexistentCollection + ".example.com/theperthcountyconspiracy",
		arvadostest.NonexistentCollection + ".example.com/t=" + arvadostest.ActiveToken + "/theperthcountyconspiracy",
	} {
		resp := httptest.NewRecorder()
		u := mustParseURL(testURL)
		req := &http.Request{
			Method:     "GET",
			URL:        u,
			RequestURI: u.RequestURI(),
		}
		s.testServer.Handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusNotFound)
		c.Check(resp.Body.String(), check.Equals, "")
	}
}

// An authorizer modifies an HTTP request to make use of the given
// token -- by adding it to a header, cookie, query param, or whatever
// -- and returns the HTTP status code we should expect from keep-web if
// the token is invalid.
type authorizer func(*http.Request, string) int

func (s *IntegrationSuite) TestVhostViaAuthzHeader(c *check.C) {
	s.doVhostRequests(c, authzViaAuthzHeader)
}
func authzViaAuthzHeader(r *http.Request, tok string) int {
	r.Header.Add("Authorization", "OAuth2 "+tok)
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaCookieValue(c *check.C) {
	s.doVhostRequests(c, authzViaCookieValue)
}
func authzViaCookieValue(r *http.Request, tok string) int {
	r.AddCookie(&http.Cookie{
		Name:  "arvados_api_token",
		Value: auth.EncodeTokenCookie([]byte(tok)),
	})
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaPath(c *check.C) {
	s.doVhostRequests(c, authzViaPath)
}
func authzViaPath(r *http.Request, tok string) int {
	r.URL.Path = "/t=" + tok + r.URL.Path
	return http.StatusNotFound
}

func (s *IntegrationSuite) TestVhostViaQueryString(c *check.C) {
	s.doVhostRequests(c, authzViaQueryString)
}
func authzViaQueryString(r *http.Request, tok string) int {
	r.URL.RawQuery = "api_token=" + tok
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaPOST(c *check.C) {
	s.doVhostRequests(c, authzViaPOST)
}
func authzViaPOST(r *http.Request, tok string) int {
	r.Method = "POST"
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Body = ioutil.NopCloser(strings.NewReader(
		url.Values{"api_token": {tok}}.Encode()))
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaXHRPOST(c *check.C) {
	s.doVhostRequests(c, authzViaPOST)
}
func authzViaXHRPOST(r *http.Request, tok string) int {
	r.Method = "POST"
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Add("Origin", "https://origin.example")
	r.Body = ioutil.NopCloser(strings.NewReader(
		url.Values{
			"api_token":   {tok},
			"disposition": {"attachment"},
		}.Encode()))
	return http.StatusUnauthorized
}

// Try some combinations of {url, token} using the given authorization
// mechanism, and verify the result is correct.
func (s *IntegrationSuite) doVhostRequests(c *check.C, authz authorizer) {
	for _, hostPath := range []string{
		arvadostest.FooCollection + ".example.com/foo",
		arvadostest.FooCollection + "--collections.example.com/foo",
		arvadostest.FooCollection + "--collections.example.com/_/foo",
		arvadostest.FooPdh + ".example.com/foo",
		strings.Replace(arvadostest.FooPdh, "+", "-", -1) + "--collections.example.com/foo",
		arvadostest.FooBarDirCollection + ".example.com/dir1/foo",
	} {
		c.Log("doRequests: ", hostPath)
		s.doVhostRequestsWithHostPath(c, authz, hostPath)
	}
}

func (s *IntegrationSuite) doVhostRequestsWithHostPath(c *check.C, authz authorizer, hostPath string) {
	for _, tok := range []string{
		arvadostest.ActiveToken,
		arvadostest.ActiveToken[:15],
		arvadostest.SpectatorToken,
		"bogus",
		"",
	} {
		u := mustParseURL("http://" + hostPath)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     http.Header{},
		}
		failCode := authz(req, tok)
		req, resp := s.doReq(req)
		code, body := resp.Code, resp.Body.String()

		// If the initial request had a (non-empty) token
		// showing in the query string, we should have been
		// redirected in order to hide it in a cookie.
		c.Check(req.URL.String(), check.Not(check.Matches), `.*api_token=.+`)

		if tok == arvadostest.ActiveToken {
			c.Check(code, check.Equals, http.StatusOK)
			c.Check(body, check.Equals, "foo")

		} else {
			c.Check(code >= 400, check.Equals, true)
			c.Check(code < 500, check.Equals, true)
			if tok == arvadostest.SpectatorToken {
				// Valid token never offers to retry
				// with different credentials.
				c.Check(code, check.Equals, http.StatusNotFound)
			} else {
				// Invalid token can ask to retry
				// depending on the authz method.
				c.Check(code, check.Equals, failCode)
			}
			c.Check(body, check.Equals, "")
		}
	}
}

func (s *IntegrationSuite) doReq(req *http.Request) (*http.Request, *httptest.ResponseRecorder) {
	resp := httptest.NewRecorder()
	s.testServer.Handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusSeeOther {
		return req, resp
	}
	cookies := (&http.Response{Header: resp.Header()}).Cookies()
	u, _ := req.URL.Parse(resp.Header().Get("Location"))
	req = &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     http.Header{},
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return s.doReq(req)
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenToCookie(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?api_token="+arvadostest.ActiveToken,
		"",
		"",
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestSingleOriginSecretLink(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/t="+arvadostest.ActiveToken+"/foo",
		"",
		"",
		"",
		http.StatusOK,
		"foo",
	)
}

// Bad token in URL is 404 Not Found because it doesn't make sense to
// retry the same URL with different authorization.
func (s *IntegrationSuite) TestSingleOriginSecretLinkBadToken(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/t=bogus/foo",
		"",
		"",
		"",
		http.StatusNotFound,
		"",
	)
}

// Bad token in a cookie (even if it got there via our own
// query-string-to-cookie redirect) is, in principle, retryable at the
// same URL so it's 401 Unauthorized.
func (s *IntegrationSuite) TestVhostRedirectQueryTokenToBogusCookie(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?api_token=thisisabogustoken",
		"",
		"",
		http.StatusUnauthorized,
		"",
	)
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenSingleOriginError(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		"",
		"",
		http.StatusBadRequest,
		"",
	)
}

// If client requests an attachment by putting ?disposition=attachment
// in the query string, and gets redirected, the redirect target
// should respond with an attachment.
func (s *IntegrationSuite) TestVhostRedirectQueryTokenRequestAttachment(c *check.C) {
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?disposition=attachment&api_token="+arvadostest.ActiveToken,
		"",
		"",
		http.StatusOK,
		"foo",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenSiteFS(c *check.C) {
	s.testServer.Config.AttachmentOnlyHost = "download.example.com"
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		"download.example.com/by_id/"+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		"",
		"",
		http.StatusOK,
		"foo",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenTrustAllContent(c *check.C) {
	s.testServer.Config.TrustAllContent = true
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		"",
		"",
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenAttachmentOnlyHost(c *check.C) {
	s.testServer.Config.AttachmentOnlyHost = "example.com:1234"

	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		"",
		"",
		http.StatusBadRequest,
		"",
	)

	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com:1234/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		"",
		"",
		http.StatusOK,
		"foo",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Equals, "attachment")
}

func (s *IntegrationSuite) TestVhostRedirectPOSTFormTokenToCookie(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "POST",
		arvadostest.FooCollection+".example.com/foo",
		"",
		"application/x-www-form-urlencoded",
		url.Values{"api_token": {arvadostest.ActiveToken}}.Encode(),
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestVhostRedirectPOSTFormTokenToCookie404(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "POST",
		arvadostest.FooCollection+".example.com/foo",
		"",
		"application/x-www-form-urlencoded",
		url.Values{"api_token": {arvadostest.SpectatorToken}}.Encode(),
		http.StatusNotFound,
		"",
	)
}

func (s *IntegrationSuite) TestAnonymousTokenOK(c *check.C) {
	s.testServer.Config.AnonymousTokens = []string{arvadostest.AnonymousToken}
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.HelloWorldCollection+"/Hello%20world.txt",
		"",
		"",
		"",
		http.StatusOK,
		"Hello world\n",
	)
}

func (s *IntegrationSuite) TestAnonymousTokenError(c *check.C) {
	s.testServer.Config.AnonymousTokens = []string{"anonymousTokenConfiguredButInvalid"}
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.HelloWorldCollection+"/Hello%20world.txt",
		"",
		"",
		"",
		http.StatusNotFound,
		"",
	)
}

// XHRs can't follow redirect-with-cookie so they rely on method=POST
// and disposition=attachment (telling us it's acceptable to respond
// with content instead of a redirect) and an Origin header that gets
// added automatically by the browser (telling us it's desirable to do
// so).
func (s *IntegrationSuite) TestXHRNoRedirect(c *check.C) {
	u, _ := url.Parse("http://example.com/c=" + arvadostest.FooCollection + "/foo")
	req := &http.Request{
		Method:     "POST",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Origin":       {"https://origin.example"},
			"Content-Type": {"application/x-www-form-urlencoded"},
		},
		Body: ioutil.NopCloser(strings.NewReader(url.Values{
			"api_token":   {arvadostest.ActiveToken},
			"disposition": {"attachment"},
		}.Encode())),
	}
	resp := httptest.NewRecorder()
	s.testServer.Handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "foo")
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")
}

func (s *IntegrationSuite) testVhostRedirectTokenToCookie(c *check.C, method, hostPath, queryString, contentType, reqBody string, expectStatus int, expectRespBody string) *httptest.ResponseRecorder {
	u, _ := url.Parse(`http://` + hostPath + queryString)
	req := &http.Request{
		Method:     method,
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     http.Header{"Content-Type": {contentType}},
		Body:       ioutil.NopCloser(strings.NewReader(reqBody)),
	}

	resp := httptest.NewRecorder()
	defer func() {
		c.Check(resp.Code, check.Equals, expectStatus)
		c.Check(resp.Body.String(), check.Equals, expectRespBody)
	}()

	s.testServer.Handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusSeeOther {
		return resp
	}
	c.Check(resp.Body.String(), check.Matches, `.*href="//`+regexp.QuoteMeta(html.EscapeString(hostPath))+`(\?[^"]*)?".*`)
	cookies := (&http.Response{Header: resp.Header()}).Cookies()

	u, _ = u.Parse(resp.Header().Get("Location"))
	req = &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     http.Header{},
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp = httptest.NewRecorder()
	s.testServer.Handler.ServeHTTP(resp, req)
	c.Check(resp.Header().Get("Location"), check.Equals, "")
	return resp
}

func (s *IntegrationSuite) TestDirectoryListing(c *check.C) {
	s.testServer.Config.AttachmentOnlyHost = "download.example.com"
	authHeader := http.Header{
		"Authorization": {"OAuth2 " + arvadostest.ActiveToken},
	}
	for _, trial := range []struct {
		uri      string
		header   http.Header
		expect   []string
		redirect string
		cutDirs  int
	}{
		{
			uri:     strings.Replace(arvadostest.FooAndBarFilesInDirPDH, "+", "-", -1) + ".example.com/",
			header:  authHeader,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 0,
		},
		{
			uri:     strings.Replace(arvadostest.FooAndBarFilesInDirPDH, "+", "-", -1) + ".example.com/dir1/",
			header:  authHeader,
			expect:  []string{"foo", "bar"},
			cutDirs: 1,
		},
		{
			uri:     "download.example.com/collections/" + arvadostest.FooAndBarFilesInDirUUID + "/",
			header:  authHeader,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 2,
		},
		{
			uri:     "download.example.com/users/active/foo_file_in_dir/",
			header:  authHeader,
			expect:  []string{"dir1/"},
			cutDirs: 3,
		},
		{
			uri:     "download.example.com/users/active/foo_file_in_dir/dir1/",
			header:  authHeader,
			expect:  []string{"bar"},
			cutDirs: 4,
		},
		{
			uri:     "download.example.com/",
			header:  authHeader,
			expect:  []string{"users/"},
			cutDirs: 0,
		},
		{
			uri:      "download.example.com/users",
			header:   authHeader,
			redirect: "/users/",
			expect:   []string{"active/"},
			cutDirs:  1,
		},
		{
			uri:     "download.example.com/users/",
			header:  authHeader,
			expect:  []string{"active/"},
			cutDirs: 1,
		},
		{
			uri:      "download.example.com/users/active",
			header:   authHeader,
			redirect: "/users/active/",
			expect:   []string{"foo_file_in_dir/"},
			cutDirs:  2,
		},
		{
			uri:     "download.example.com/users/active/",
			header:  authHeader,
			expect:  []string{"foo_file_in_dir/"},
			cutDirs: 2,
		},
		{
			uri:     "collections.example.com/collections/download/" + arvadostest.FooAndBarFilesInDirUUID + "/" + arvadostest.ActiveToken + "/",
			header:  nil,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 4,
		},
		{
			uri:     "collections.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/t=" + arvadostest.ActiveToken + "/",
			header:  nil,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 2,
		},
		{
			uri:     "collections.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/t=" + arvadostest.ActiveToken,
			header:  nil,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 2,
		},
		{
			uri:     "download.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID,
			header:  authHeader,
			expect:  []string{"dir1/foo", "dir1/bar"},
			cutDirs: 1,
		},
		{
			uri:      "download.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/dir1",
			header:   authHeader,
			redirect: "/c=" + arvadostest.FooAndBarFilesInDirUUID + "/dir1/",
			expect:   []string{"foo", "bar"},
			cutDirs:  2,
		},
		{
			uri:     "download.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/_/dir1/",
			header:  authHeader,
			expect:  []string{"foo", "bar"},
			cutDirs: 3,
		},
		{
			uri:      arvadostest.FooAndBarFilesInDirUUID + ".example.com/dir1?api_token=" + arvadostest.ActiveToken,
			header:   authHeader,
			redirect: "/dir1/",
			expect:   []string{"foo", "bar"},
			cutDirs:  1,
		},
		{
			uri:    "collections.example.com/c=" + arvadostest.FooAndBarFilesInDirUUID + "/theperthcountyconspiracydoesnotexist/",
			header: authHeader,
			expect: nil,
		},
	} {
		c.Logf("HTML: %q => %q", trial.uri, trial.expect)
		resp := httptest.NewRecorder()
		u := mustParseURL("//" + trial.uri)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     trial.header,
		}
		s.testServer.Handler.ServeHTTP(resp, req)
		var cookies []*http.Cookie
		for resp.Code == http.StatusSeeOther {
			u, _ := req.URL.Parse(resp.Header().Get("Location"))
			req = &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header:     trial.header,
			}
			cookies = append(cookies, (&http.Response{Header: resp.Header()}).Cookies()...)
			for _, c := range cookies {
				req.AddCookie(c)
			}
			resp = httptest.NewRecorder()
			s.testServer.Handler.ServeHTTP(resp, req)
		}
		if trial.redirect != "" {
			c.Check(req.URL.Path, check.Equals, trial.redirect)
		}
		if trial.expect == nil {
			c.Check(resp.Code, check.Equals, http.StatusNotFound)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusOK)
			for _, e := range trial.expect {
				c.Check(resp.Body.String(), check.Matches, `(?ms).*href="`+e+`".*`)
			}
			c.Check(resp.Body.String(), check.Matches, `(?ms).*--cut-dirs=`+fmt.Sprintf("%d", trial.cutDirs)+` .*`)
		}

		c.Logf("WebDAV: %q => %q", trial.uri, trial.expect)
		req = &http.Request{
			Method:     "OPTIONS",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     trial.header,
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
		}
		resp = httptest.NewRecorder()
		s.testServer.Handler.ServeHTTP(resp, req)
		if trial.expect == nil {
			c.Check(resp.Code, check.Equals, http.StatusNotFound)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusOK)
		}

		req = &http.Request{
			Method:     "PROPFIND",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     trial.header,
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
		}
		resp = httptest.NewRecorder()
		s.testServer.Handler.ServeHTTP(resp, req)
		if trial.expect == nil {
			c.Check(resp.Code, check.Equals, http.StatusNotFound)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusMultiStatus)
			for _, e := range trial.expect {
				c.Check(resp.Body.String(), check.Matches, `(?ms).*<D:href>`+filepath.Join(u.Path, e)+`</D:href>.*`)
			}
		}
	}
}

func (s *IntegrationSuite) TestHealthCheckPing(c *check.C) {
	s.testServer.Config.ManagementToken = arvadostest.ManagementToken
	authHeader := http.Header{
		"Authorization": {"Bearer " + arvadostest.ManagementToken},
	}

	resp := httptest.NewRecorder()
	u := mustParseURL("http://download.example.com/_health/ping")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     authHeader,
	}
	s.testServer.Handler.ServeHTTP(resp, req)

	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, `{"health":"OK"}\n`)
}
