// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&UnitSuite{})

func init() {
	arvados.DebugLocksPanicMode = true
}

type UnitSuite struct {
	cluster *arvados.Cluster
	handler *handler
}

func (s *UnitSuite) SetUpTest(c *check.C) {
	logger := ctxlog.TestLogger(c)
	ldr := config.NewLoader(bytes.NewBufferString("Clusters: {zzzzz: {}}"), logger)
	ldr.Path = "-"
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	cc, err := cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.cluster = cc
	s.handler = &handler{
		Cluster: cc,
		Cache: cache{
			cluster:  cc,
			logger:   logger,
			registry: prometheus.NewRegistry(),
		},
	}
}

func (s *UnitSuite) TestCORSPreflight(c *check.C) {
	h := s.handler
	u := mustParseURL("http://keep-web.example/c=" + arvadostest.FooCollection + "/foo")
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
	c.Check(resp.Header().Get("Access-Control-Allow-Methods"), check.Equals, "COPY, DELETE, GET, LOCK, MKCOL, MOVE, OPTIONS, POST, PROPFIND, PROPPATCH, PUT, RMCOL, UNLOCK")
	c.Check(resp.Header().Get("Access-Control-Allow-Headers"), check.Equals, "Authorization, Content-Type, Range, Depth, Destination, If, Lock-Token, Overwrite, Timeout")

	// Check preflight for a disallowed request
	resp = httptest.NewRecorder()
	req.Header.Set("Access-Control-Request-Method", "MAKE-COFFEE")
	h.ServeHTTP(resp, req)
	c.Check(resp.Body.String(), check.Equals, "")
	c.Check(resp.Code, check.Equals, http.StatusMethodNotAllowed)
}

func (s *UnitSuite) TestEmptyResponse(c *check.C) {
	for _, trial := range []struct {
		dataExists    bool
		sendIMSHeader bool
		expectStatus  int
		logRegexp     string
	}{
		// If we return no content due to a Keep read error,
		// we should emit a log message.
		{false, false, http.StatusOK, `(?ms).*only wrote 0 bytes.*`},

		// If we return no content because the client sent an
		// If-Modified-Since header, our response should be
		// 304.  We still expect a "File download" log since it
		// counts as a file access for auditing.
		{true, true, http.StatusNotModified, `(?ms).*msg="File download".*`},
	} {
		c.Logf("trial: %+v", trial)
		arvadostest.StartKeep(2, true)
		if trial.dataExists {
			arv, err := arvadosclient.MakeArvadosClient()
			c.Assert(err, check.IsNil)
			arv.ApiToken = arvadostest.ActiveToken
			kc, err := keepclient.MakeKeepClient(arv)
			c.Assert(err, check.IsNil)
			_, _, err = kc.PutB([]byte("foo"))
			c.Assert(err, check.IsNil)
		}

		u := mustParseURL("http://" + arvadostest.FooCollection + ".keep-web.example/foo")
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + arvadostest.ActiveToken},
			},
		}
		if trial.sendIMSHeader {
			req.Header.Set("If-Modified-Since", strings.Replace(time.Now().UTC().Format(time.RFC1123), "UTC", "GMT", -1))
		}

		var logbuf bytes.Buffer
		logger := logrus.New()
		logger.Out = &logbuf
		req = req.WithContext(ctxlog.Context(context.Background(), logger))

		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, trial.expectStatus)
		c.Check(resp.Body.String(), check.Equals, "")

		c.Log(logbuf.String())
		c.Check(logbuf.String(), check.Matches, trial.logRegexp)
	}
}

func (s *UnitSuite) TestInvalidUUID(c *check.C) {
	bogusID := strings.Replace(arvadostest.FooCollectionPDH, "+", "-", 1) + "-"
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
		u := mustParseURL(trial)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
		}
		resp := httptest.NewRecorder()
		s.cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
		s.handler.ServeHTTP(resp, req)
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
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusNotFound)
		c.Check(resp.Body.String(), check.Equals, notFoundMessage+"\n")
	}
}

// An authorizer modifies an HTTP request to make use of the given
// token -- by adding it to a header, cookie, query param, or whatever
// -- and returns the HTTP status code we should expect from keep-web if
// the token is invalid.
type authorizer func(*http.Request, string) int

func (s *IntegrationSuite) TestVhostViaAuthzHeaderOAuth2(c *check.C) {
	s.doVhostRequests(c, authzViaAuthzHeaderOAuth2)
}
func authzViaAuthzHeaderOAuth2(r *http.Request, tok string) int {
	r.Header.Add("Authorization", "Bearer "+tok)
	return http.StatusUnauthorized
}
func (s *IntegrationSuite) TestVhostViaAuthzHeaderBearer(c *check.C) {
	s.doVhostRequests(c, authzViaAuthzHeaderBearer)
}
func authzViaAuthzHeaderBearer(r *http.Request, tok string) int {
	r.Header.Add("Authorization", "Bearer "+tok)
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
		arvadostest.FooCollectionPDH + ".example.com/foo",
		strings.Replace(arvadostest.FooCollectionPDH, "+", "-", -1) + "--collections.example.com/foo",
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
			if code == 404 {
				c.Check(body, check.Equals, notFoundMessage+"\n")
			} else {
				c.Check(body, check.Equals, unauthorizedMessage+"\n")
			}
		}
	}
}

func (s *IntegrationSuite) TestVhostPortMatch(c *check.C) {
	for _, host := range []string{"download.example.com", "DOWNLOAD.EXAMPLE.COM"} {
		for _, port := range []string{"80", "443", "8000"} {
			s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = fmt.Sprintf("download.example.com:%v", port)
			u := mustParseURL(fmt.Sprintf("http://%v/by_id/%v/foo", host, arvadostest.FooCollection))
			req := &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header:     http.Header{"Authorization": []string{"Bearer " + arvadostest.ActiveToken}},
			}
			req, resp := s.doReq(req)
			code, _ := resp.Code, resp.Body.String()

			if port == "8000" {
				c.Check(code, check.Equals, 401)
			} else {
				c.Check(code, check.Equals, 200)
			}
		}
	}
}

func (s *IntegrationSuite) do(method string, urlstring string, token string, hdr http.Header) (*http.Request, *httptest.ResponseRecorder) {
	u := mustParseURL(urlstring)
	if hdr == nil && token != "" {
		hdr = http.Header{"Authorization": {"Bearer " + token}}
	} else if hdr == nil {
		hdr = http.Header{}
	} else if token != "" {
		panic("must not pass both token and hdr")
	}
	return s.doReq(&http.Request{
		Method:     method,
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     hdr,
	})
}

func (s *IntegrationSuite) doReq(req *http.Request) (*http.Request, *httptest.ResponseRecorder) {
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
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
		nil,
		"",
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestSingleOriginSecretLink(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/t="+arvadostest.ActiveToken+"/foo",
		"",
		nil,
		"",
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestCollectionSharingToken(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooFileCollectionUUID+"/t="+arvadostest.FooFileCollectionSharingToken+"/foo",
		"",
		nil,
		"",
		http.StatusOK,
		"foo",
	)
	// Same valid sharing token, but requesting a different collection
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/t="+arvadostest.FooFileCollectionSharingToken+"/foo",
		"",
		nil,
		"",
		http.StatusNotFound,
		notFoundMessage+"\n",
	)
}

// Bad token in URL is 404 Not Found because it doesn't make sense to
// retry the same URL with different authorization.
func (s *IntegrationSuite) TestSingleOriginSecretLinkBadToken(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/t=bogus/foo",
		"",
		nil,
		"",
		http.StatusNotFound,
		notFoundMessage+"\n",
	)
}

// Bad token in a cookie (even if it got there via our own
// query-string-to-cookie redirect) is, in principle, retryable via
// wb2-login-and-redirect flow.
func (s *IntegrationSuite) TestVhostRedirectQueryTokenToBogusCookie(c *check.C) {
	// Inline
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?api_token=thisisabogustoken",
		http.Header{"Sec-Fetch-Mode": {"navigate"}},
		"",
		http.StatusSeeOther,
		"",
	)
	u, err := url.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Logf("redirected to %s", u)
	c.Check(u.Host, check.Equals, s.handler.Cluster.Services.Workbench2.ExternalURL.Host)
	c.Check(u.Query().Get("redirectToPreview"), check.Equals, "/c="+arvadostest.FooCollection+"/foo")
	c.Check(u.Query().Get("redirectToDownload"), check.Equals, "")

	// Download/attachment indicated by ?disposition=attachment
	resp = s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?api_token=thisisabogustoken&disposition=attachment",
		http.Header{"Sec-Fetch-Mode": {"navigate"}},
		"",
		http.StatusSeeOther,
		"",
	)
	u, err = url.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Logf("redirected to %s", u)
	c.Check(u.Host, check.Equals, s.handler.Cluster.Services.Workbench2.ExternalURL.Host)
	c.Check(u.Query().Get("redirectToPreview"), check.Equals, "")
	c.Check(u.Query().Get("redirectToDownload"), check.Equals, "/c="+arvadostest.FooCollection+"/foo")

	// Download/attachment indicated by vhost
	resp = s.testVhostRedirectTokenToCookie(c, "GET",
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host+"/c="+arvadostest.FooCollection+"/foo",
		"?api_token=thisisabogustoken",
		http.Header{"Sec-Fetch-Mode": {"navigate"}},
		"",
		http.StatusSeeOther,
		"",
	)
	u, err = url.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Logf("redirected to %s", u)
	c.Check(u.Host, check.Equals, s.handler.Cluster.Services.Workbench2.ExternalURL.Host)
	c.Check(u.Query().Get("redirectToPreview"), check.Equals, "")
	c.Check(u.Query().Get("redirectToDownload"), check.Equals, "/c="+arvadostest.FooCollection+"/foo")

	// Without "Sec-Fetch-Mode: navigate" header, just 401.
	s.testVhostRedirectTokenToCookie(c, "GET",
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host+"/c="+arvadostest.FooCollection+"/foo",
		"?api_token=thisisabogustoken",
		http.Header{"Sec-Fetch-Mode": {"cors"}},
		"",
		http.StatusUnauthorized,
		unauthorizedMessage+"\n",
	)
	s.testVhostRedirectTokenToCookie(c, "GET",
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host+"/c="+arvadostest.FooCollection+"/foo",
		"?api_token=thisisabogustoken",
		nil,
		"",
		http.StatusUnauthorized,
		unauthorizedMessage+"\n",
	)
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenSingleOriginError(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusBadRequest,
		"cannot serve inline content at this URL (possible configuration error; see https://doc.arvados.org/install/install-keep-web.html#dns)\n",
	)
}

// If client requests an attachment by putting ?disposition=attachment
// in the query string, and gets redirected, the redirect target
// should respond with an attachment.
func (s *IntegrationSuite) TestVhostRedirectQueryTokenRequestAttachment(c *check.C) {
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection+".example.com/foo",
		"?disposition=attachment&api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"foo",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenSiteFS(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		"download.example.com/by_id/"+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"foo",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
}

func (s *IntegrationSuite) TestPastCollectionVersionFileAccess(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"
	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		"download.example.com/c="+arvadostest.WazVersion1Collection+"/waz",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"waz",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
	resp = s.testVhostRedirectTokenToCookie(c, "GET",
		"download.example.com/by_id/"+arvadostest.WazVersion1Collection+"/waz",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"waz",
	)
	c.Check(resp.Header().Get("Content-Disposition"), check.Matches, "attachment(;.*)?")
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenTrustAllContent(c *check.C) {
	s.handler.Cluster.Collections.TrustAllContent = true
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenAttachmentOnlyHost(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "example.com:1234"

	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
		"",
		http.StatusBadRequest,
		"cannot serve inline content at this URL (possible configuration error; see https://doc.arvados.org/install/install-keep-web.html#dns)\n",
	)

	resp := s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com:1234/c="+arvadostest.FooCollection+"/foo",
		"?api_token="+arvadostest.ActiveToken,
		nil,
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
		http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		url.Values{"api_token": {arvadostest.ActiveToken}}.Encode(),
		http.StatusOK,
		"foo",
	)
}

func (s *IntegrationSuite) TestVhostRedirectPOSTFormTokenToCookie404(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "POST",
		arvadostest.FooCollection+".example.com/foo",
		"",
		http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		url.Values{"api_token": {arvadostest.SpectatorToken}}.Encode(),
		http.StatusNotFound,
		notFoundMessage+"\n",
	)
}

func (s *IntegrationSuite) TestAnonymousTokenOK(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.HelloWorldCollection+"/Hello%20world.txt",
		"",
		nil,
		"",
		http.StatusOK,
		"Hello world\n",
	)
}

func (s *IntegrationSuite) TestAnonymousTokenError(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = "anonymousTokenConfiguredButInvalid"
	s.testVhostRedirectTokenToCookie(c, "GET",
		"example.com/c="+arvadostest.HelloWorldCollection+"/Hello%20world.txt",
		"",
		nil,
		"",
		http.StatusNotFound,
		notFoundMessage+"\n",
	)
}

func (s *IntegrationSuite) TestSpecialCharsInPath(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"

	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveToken
	fs, err := (&arvados.Collection{}).FileSystem(client, nil)
	c.Assert(err, check.IsNil)
	f, err := fs.OpenFile("https:\\\"odd' path chars", os.O_CREATE, 0777)
	c.Assert(err, check.IsNil)
	f.Close()
	mtxt, err := fs.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	var coll arvados.Collection
	err = client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": mtxt,
		},
	})
	c.Assert(err, check.IsNil)

	u, _ := url.Parse("http://download.example.com/c=" + coll.UUID + "/")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Authorization": {"Bearer " + client.AuthToken},
		},
	}
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, `(?ms).*href="./https:%5c%22odd%27%20path%20chars"\S+https:\\&#34;odd&#39; path chars.*`)
}

func (s *IntegrationSuite) TestForwardSlashSubstitution(c *check.C) {
	arv := arvados.NewClientFromEnv()
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"
	s.handler.Cluster.Collections.ForwardSlashNameSubstitution = "{SOLIDUS}"
	name := "foo/bar/baz"
	nameShown := strings.Replace(name, "/", "{SOLIDUS}", -1)
	nameShownEscaped := strings.Replace(name, "/", "%7bSOLIDUS%7d", -1)

	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveToken
	fs, err := (&arvados.Collection{}).FileSystem(client, nil)
	c.Assert(err, check.IsNil)
	f, err := fs.OpenFile("filename", os.O_CREATE, 0777)
	c.Assert(err, check.IsNil)
	f.Close()
	mtxt, err := fs.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	var coll arvados.Collection
	err = client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": mtxt,
			"name":          name,
			"owner_uuid":    arvadostest.AProjectUUID,
		},
	})
	c.Assert(err, check.IsNil)
	defer arv.RequestAndDecode(&coll, "DELETE", "arvados/v1/collections/"+coll.UUID, nil, nil)

	base := "http://download.example.com/by_id/" + coll.OwnerUUID + "/"
	for tryURL, expectRegexp := range map[string]string{
		base:                          `(?ms).*href="./` + nameShownEscaped + `/"\S+` + nameShown + `.*`,
		base + nameShownEscaped + "/": `(?ms).*href="./filename"\S+filename.*`,
	} {
		u, _ := url.Parse(tryURL)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + client.AuthToken},
			},
		}
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusOK)
		c.Check(resp.Body.String(), check.Matches, expectRegexp)
	}
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
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "foo")
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")

	// GET + Origin header is representative of both AJAX GET
	// requests and inline images via <IMG crossorigin="anonymous"
	// src="...">.
	u.RawQuery = "api_token=" + url.QueryEscape(arvadostest.ActiveTokenV2)
	req = &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Origin": {"https://origin.example"},
		},
	}
	resp = httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "foo")
	c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, "*")
}

func (s *IntegrationSuite) testVhostRedirectTokenToCookie(c *check.C, method, hostPath, queryString string, reqHeader http.Header, reqBody string, expectStatus int, expectRespBody string) *httptest.ResponseRecorder {
	if reqHeader == nil {
		reqHeader = http.Header{}
	}
	u, _ := url.Parse(`http://` + hostPath + queryString)
	c.Logf("requesting %s", u)
	req := &http.Request{
		Method:     method,
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     reqHeader,
		Body:       ioutil.NopCloser(strings.NewReader(reqBody)),
	}

	resp := httptest.NewRecorder()
	defer func() {
		c.Check(resp.Code, check.Equals, expectStatus)
		c.Check(resp.Body.String(), check.Equals, expectRespBody)
	}()

	s.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusSeeOther {
		return resp
	}
	c.Check(resp.Body.String(), check.Matches, `.*href="http://`+regexp.QuoteMeta(html.EscapeString(hostPath))+`(\?[^"]*)?".*`)
	c.Check(strings.Split(resp.Header().Get("Location"), "?")[0], check.Equals, "http://"+hostPath)
	cookies := (&http.Response{Header: resp.Header()}).Cookies()

	u, err := u.Parse(resp.Header().Get("Location"))
	c.Assert(err, check.IsNil)
	c.Logf("following redirect to %s", u)
	req = &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     reqHeader,
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp = httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		c.Check(resp.Header().Get("Location"), check.Equals, "")
	}
	return resp
}

func (s *IntegrationSuite) TestDirectoryListingWithAnonymousToken(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = arvadostest.AnonymousToken
	s.testDirectoryListing(c)
}

func (s *IntegrationSuite) TestDirectoryListingWithNoAnonymousToken(c *check.C) {
	s.handler.Cluster.Users.AnonymousUserToken = ""
	s.testDirectoryListing(c)
}

func (s *IntegrationSuite) testDirectoryListing(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"
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
			// URLs of this form ignore authHeader, and
			// FooAndBarFilesInDirUUID isn't public, so
			// this returns 401.
			uri:    "download.example.com/collections/" + arvadostest.FooAndBarFilesInDirUUID + "/",
			header: authHeader,
			expect: nil,
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
		{
			uri:     "download.example.com/c=" + arvadostest.WazVersion1Collection,
			header:  authHeader,
			expect:  []string{"waz"},
			cutDirs: 1,
		},
		{
			uri:     "download.example.com/by_id/" + arvadostest.WazVersion1Collection,
			header:  authHeader,
			expect:  []string{"waz"},
			cutDirs: 2,
		},
	} {
		comment := check.Commentf("HTML: %q => %q", trial.uri, trial.expect)
		resp := httptest.NewRecorder()
		u := mustParseURL("//" + trial.uri)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     copyHeader(trial.header),
		}
		s.handler.ServeHTTP(resp, req)
		var cookies []*http.Cookie
		for resp.Code == http.StatusSeeOther {
			u, _ := req.URL.Parse(resp.Header().Get("Location"))
			req = &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header:     copyHeader(trial.header),
			}
			cookies = append(cookies, (&http.Response{Header: resp.Header()}).Cookies()...)
			for _, c := range cookies {
				req.AddCookie(c)
			}
			resp = httptest.NewRecorder()
			s.handler.ServeHTTP(resp, req)
		}
		if trial.redirect != "" {
			c.Check(req.URL.Path, check.Equals, trial.redirect, comment)
		}
		if trial.expect == nil {
			if s.handler.Cluster.Users.AnonymousUserToken == "" {
				c.Check(resp.Code, check.Equals, http.StatusUnauthorized, comment)
			} else {
				c.Check(resp.Code, check.Equals, http.StatusNotFound, comment)
			}
		} else {
			c.Check(resp.Code, check.Equals, http.StatusOK, comment)
			for _, e := range trial.expect {
				c.Check(resp.Body.String(), check.Matches, `(?ms).*href="./`+e+`".*`, comment)
			}
			c.Check(resp.Body.String(), check.Matches, `(?ms).*--cut-dirs=`+fmt.Sprintf("%d", trial.cutDirs)+` .*`, comment)
		}

		comment = check.Commentf("WebDAV: %q => %q", trial.uri, trial.expect)
		req = &http.Request{
			Method:     "OPTIONS",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     copyHeader(trial.header),
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
		}
		resp = httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		if trial.expect == nil {
			if s.handler.Cluster.Users.AnonymousUserToken == "" {
				c.Check(resp.Code, check.Equals, http.StatusUnauthorized, comment)
			} else {
				c.Check(resp.Code, check.Equals, http.StatusNotFound, comment)
			}
		} else {
			c.Check(resp.Code, check.Equals, http.StatusOK, comment)
		}

		req = &http.Request{
			Method:     "PROPFIND",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header:     copyHeader(trial.header),
			Body:       ioutil.NopCloser(&bytes.Buffer{}),
		}
		resp = httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		if trial.expect == nil {
			if s.handler.Cluster.Users.AnonymousUserToken == "" {
				c.Check(resp.Code, check.Equals, http.StatusUnauthorized, comment)
			} else {
				c.Check(resp.Code, check.Equals, http.StatusNotFound, comment)
			}
		} else {
			c.Check(resp.Code, check.Equals, http.StatusMultiStatus, comment)
			for _, e := range trial.expect {
				if strings.HasSuffix(e, "/") {
					e = filepath.Join(u.Path, e) + "/"
				} else {
					e = filepath.Join(u.Path, e)
				}
				c.Check(resp.Body.String(), check.Matches, `(?ms).*<D:href>`+e+`</D:href>.*`, comment)
			}
		}
	}
}

func (s *IntegrationSuite) TestDeleteLastFile(c *check.C) {
	arv := arvados.NewClientFromEnv()
	var newCollection arvados.Collection
	err := arv.RequestAndDecode(&newCollection, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt 0:3:bar.txt\n",
			"name":          "keep-web test collection",
		},
		"ensure_unique_name": true,
	})
	c.Assert(err, check.IsNil)
	defer arv.RequestAndDecode(&newCollection, "DELETE", "arvados/v1/collections/"+newCollection.UUID, nil, nil)

	var updated arvados.Collection
	for _, fnm := range []string{"foo.txt", "bar.txt"} {
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "example.com"
		u, _ := url.Parse("http://example.com/c=" + newCollection.UUID + "/" + fnm)
		req := &http.Request{
			Method:     "DELETE",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + arvadostest.ActiveToken},
			},
		}
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusNoContent)

		updated = arvados.Collection{}
		err = arv.RequestAndDecode(&updated, "GET", "arvados/v1/collections/"+newCollection.UUID, nil, nil)
		c.Check(err, check.IsNil)
		c.Check(updated.ManifestText, check.Not(check.Matches), `(?ms).*\Q`+fnm+`\E.*`)
		c.Logf("updated manifest_text %q", updated.ManifestText)
	}
	c.Check(updated.ManifestText, check.Equals, "")
}

func (s *IntegrationSuite) TestFileContentType(c *check.C) {
	s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = "download.example.com"

	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveToken
	arv, err := arvadosclient.New(client)
	c.Assert(err, check.Equals, nil)
	kc, err := keepclient.MakeKeepClient(arv)
	c.Assert(err, check.Equals, nil)

	fs, err := (&arvados.Collection{}).FileSystem(client, kc)
	c.Assert(err, check.IsNil)

	trials := []struct {
		filename    string
		content     string
		contentType string
	}{
		{"picture.txt", "BMX bikes are small this year\n", "text/plain; charset=utf-8"},
		{"picture.bmp", "BMX bikes are small this year\n", "image/(x-ms-)?bmp"},
		{"picture.jpg", "BMX bikes are small this year\n", "image/jpeg"},
		{"picture1", "BMX bikes are small this year\n", "image/bmp"},            // content sniff; "BM" is the magic signature for .bmp
		{"picture2", "Cars are small this year\n", "text/plain; charset=utf-8"}, // content sniff
	}
	for _, trial := range trials {
		f, err := fs.OpenFile(trial.filename, os.O_CREATE|os.O_WRONLY, 0777)
		c.Assert(err, check.IsNil)
		_, err = f.Write([]byte(trial.content))
		c.Assert(err, check.IsNil)
		c.Assert(f.Close(), check.IsNil)
	}
	mtxt, err := fs.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	var coll arvados.Collection
	err = client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": mtxt,
		},
	})
	c.Assert(err, check.IsNil)

	for _, trial := range trials {
		u, _ := url.Parse("http://download.example.com/by_id/" + coll.UUID + "/" + trial.filename)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + client.AuthToken},
			},
		}
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusOK)
		c.Check(resp.Header().Get("Content-Type"), check.Matches, trial.contentType)
		c.Check(resp.Body.String(), check.Equals, trial.content)
	}
}

func (s *IntegrationSuite) TestKeepClientBlockCache(c *check.C) {
	s.handler.Cluster.Collections.WebDAVCache.MaxBlockEntries = 42
	c.Check(keepclient.DefaultBlockCache.MaxBlocks, check.Not(check.Equals), 42)
	u := mustParseURL("http://keep-web.example/c=" + arvadostest.FooCollection + "/t=" + arvadostest.ActiveToken + "/foo")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
	}
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(keepclient.DefaultBlockCache.MaxBlocks, check.Equals, 42)
}

// Writing to a collection shouldn't affect its entry in the
// PDH-to-manifest cache.
func (s *IntegrationSuite) TestCacheWriteCollectionSamePDH(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)
	arv.ApiToken = arvadostest.ActiveToken

	u := mustParseURL("http://x.example/testfile")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header:     http.Header{"Authorization": {"Bearer " + arv.ApiToken}},
	}

	checkWithID := func(id string, status int) {
		req.URL.Host = strings.Replace(id, "+", "-", -1) + ".example"
		req.Host = req.URL.Host
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, status)
	}

	var colls [2]arvados.Collection
	for i := range colls {
		err := arv.Create("collections",
			map[string]interface{}{
				"ensure_unique_name": true,
				"collection": map[string]interface{}{
					"name": "test collection",
				},
			}, &colls[i])
		c.Assert(err, check.Equals, nil)
	}

	// Populate cache with empty collection
	checkWithID(colls[0].PortableDataHash, http.StatusNotFound)

	// write a file to colls[0]
	reqPut := *req
	reqPut.Method = "PUT"
	reqPut.URL.Host = colls[0].UUID + ".example"
	reqPut.Host = req.URL.Host
	reqPut.Body = ioutil.NopCloser(bytes.NewBufferString("testdata"))
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, &reqPut)
	c.Check(resp.Code, check.Equals, http.StatusCreated)

	// new file should not appear in colls[1]
	checkWithID(colls[1].PortableDataHash, http.StatusNotFound)
	checkWithID(colls[1].UUID, http.StatusNotFound)

	checkWithID(colls[0].UUID, http.StatusOK)
}

func copyHeader(h http.Header) http.Header {
	hc := http.Header{}
	for k, v := range h {
		hc[k] = append([]string(nil), v...)
	}
	return hc
}

func (s *IntegrationSuite) checkUploadDownloadRequest(c *check.C, req *http.Request,
	successCode int, direction string, perm bool, userUuid, collectionUuid, collectionPDH, filepath string) {

	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.AdminToken
	var logentries arvados.LogList
	limit1 := 1
	err := client.RequestAndDecode(&logentries, "GET", "arvados/v1/logs", nil,
		arvados.ResourceListParams{
			Limit: &limit1,
			Order: "created_at desc"})
	c.Check(err, check.IsNil)
	c.Check(logentries.Items, check.HasLen, 1)
	lastLogId := logentries.Items[0].ID
	c.Logf("lastLogId: %d", lastLogId)

	var logbuf bytes.Buffer
	logger := logrus.New()
	logger.Out = &logbuf
	resp := httptest.NewRecorder()
	req = req.WithContext(ctxlog.Context(context.Background(), logger))
	s.handler.ServeHTTP(resp, req)

	if perm {
		c.Check(resp.Result().StatusCode, check.Equals, successCode)
		c.Check(logbuf.String(), check.Matches, `(?ms).*msg="File `+direction+`".*`)
		c.Check(logbuf.String(), check.Not(check.Matches), `(?ms).*level=error.*`)

		deadline := time.Now().Add(time.Second)
		for {
			c.Assert(time.Now().After(deadline), check.Equals, false, check.Commentf("timed out waiting for log entry"))
			logentries = arvados.LogList{}
			err = client.RequestAndDecode(&logentries, "GET", "arvados/v1/logs", nil,
				arvados.ResourceListParams{
					Filters: []arvados.Filter{
						{Attr: "event_type", Operator: "=", Operand: "file_" + direction},
						{Attr: "object_uuid", Operator: "=", Operand: userUuid},
					},
					Limit: &limit1,
					Order: "created_at desc",
				})
			c.Assert(err, check.IsNil)
			if len(logentries.Items) > 0 &&
				logentries.Items[0].ID > lastLogId &&
				logentries.Items[0].ObjectUUID == userUuid &&
				logentries.Items[0].Properties["collection_uuid"] == collectionUuid &&
				(collectionPDH == "" || logentries.Items[0].Properties["portable_data_hash"] == collectionPDH) &&
				logentries.Items[0].Properties["collection_file_path"] == filepath {
				break
			}
			c.Logf("logentries.Items: %+v", logentries.Items)
			time.Sleep(50 * time.Millisecond)
		}
	} else {
		c.Check(resp.Result().StatusCode, check.Equals, http.StatusForbidden)
		c.Check(logbuf.String(), check.Equals, "")
	}
}

func (s *IntegrationSuite) TestDownloadLoggingPermission(c *check.C) {
	u := mustParseURL("http://" + arvadostest.FooCollection + ".keep-web.example/foo")

	s.handler.Cluster.Collections.TrustAllContent = true

	for _, adminperm := range []bool{true, false} {
		for _, userperm := range []bool{true, false} {
			s.handler.Cluster.Collections.WebDAVPermission.Admin.Download = adminperm
			s.handler.Cluster.Collections.WebDAVPermission.User.Download = userperm

			// Test admin permission
			req := &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header: http.Header{
					"Authorization": {"Bearer " + arvadostest.AdminToken},
				},
			}
			s.checkUploadDownloadRequest(c, req, http.StatusOK, "download", adminperm,
				arvadostest.AdminUserUUID, arvadostest.FooCollection, arvadostest.FooCollectionPDH, "foo")

			// Test user permission
			req = &http.Request{
				Method:     "GET",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header: http.Header{
					"Authorization": {"Bearer " + arvadostest.ActiveToken},
				},
			}
			s.checkUploadDownloadRequest(c, req, http.StatusOK, "download", userperm,
				arvadostest.ActiveUserUUID, arvadostest.FooCollection, arvadostest.FooCollectionPDH, "foo")
		}
	}

	s.handler.Cluster.Collections.WebDAVPermission.User.Download = true

	for _, tryurl := range []string{"http://" + arvadostest.MultilevelCollection1 + ".keep-web.example/dir1/subdir/file1",
		"http://keep-web/users/active/multilevel_collection_1/dir1/subdir/file1"} {

		u = mustParseURL(tryurl)
		req := &http.Request{
			Method:     "GET",
			Host:       u.Host,
			URL:        u,
			RequestURI: u.RequestURI(),
			Header: http.Header{
				"Authorization": {"Bearer " + arvadostest.ActiveToken},
			},
		}
		s.checkUploadDownloadRequest(c, req, http.StatusOK, "download", true,
			arvadostest.ActiveUserUUID, arvadostest.MultilevelCollection1, arvadostest.MultilevelCollection1PDH, "dir1/subdir/file1")
	}

	u = mustParseURL("http://" + strings.Replace(arvadostest.FooCollectionPDH, "+", "-", 1) + ".keep-web.example/foo")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Authorization": {"Bearer " + arvadostest.ActiveToken},
		},
	}
	s.checkUploadDownloadRequest(c, req, http.StatusOK, "download", true,
		arvadostest.ActiveUserUUID, "", arvadostest.FooCollectionPDH, "foo")
}

func (s *IntegrationSuite) TestUploadLoggingPermission(c *check.C) {
	for _, adminperm := range []bool{true, false} {
		for _, userperm := range []bool{true, false} {

			arv := arvados.NewClientFromEnv()
			arv.AuthToken = arvadostest.ActiveToken

			var coll arvados.Collection
			err := arv.RequestAndDecode(&coll,
				"POST",
				"/arvados/v1/collections",
				nil,
				map[string]interface{}{
					"ensure_unique_name": true,
					"collection": map[string]interface{}{
						"name": "test collection",
					},
				})
			c.Assert(err, check.Equals, nil)

			u := mustParseURL("http://" + coll.UUID + ".keep-web.example/bar")

			s.handler.Cluster.Collections.WebDAVPermission.Admin.Upload = adminperm
			s.handler.Cluster.Collections.WebDAVPermission.User.Upload = userperm

			// Test admin permission
			req := &http.Request{
				Method:     "PUT",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header: http.Header{
					"Authorization": {"Bearer " + arvadostest.AdminToken},
				},
				Body: io.NopCloser(bytes.NewReader([]byte("bar"))),
			}
			s.checkUploadDownloadRequest(c, req, http.StatusCreated, "upload", adminperm,
				arvadostest.AdminUserUUID, coll.UUID, "", "bar")

			// Test user permission
			req = &http.Request{
				Method:     "PUT",
				Host:       u.Host,
				URL:        u,
				RequestURI: u.RequestURI(),
				Header: http.Header{
					"Authorization": {"Bearer " + arvadostest.ActiveToken},
				},
				Body: io.NopCloser(bytes.NewReader([]byte("bar"))),
			}
			s.checkUploadDownloadRequest(c, req, http.StatusCreated, "upload", userperm,
				arvadostest.ActiveUserUUID, coll.UUID, "", "bar")
		}
	}
}
