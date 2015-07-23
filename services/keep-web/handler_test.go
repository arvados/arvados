package main

import (
	"html"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&UnitSuite{})

type UnitSuite struct {}

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
		req := &http.Request{
			Method: "GET",
			URL: mustParseURL(testURL),
		}
		(&handler{}).ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusNotFound)
		c.Check(resp.Body.String(), check.Equals, "")
	}
}

type authorizer func(*http.Request, string) int

func (s *IntegrationSuite) TestVhostViaAuthzHeader(c *check.C) {
	doVhostRequests(c, authzViaAuthzHeader)
}
func authzViaAuthzHeader(r *http.Request, tok string) int {
	r.Header.Add("Authorization", "OAuth2 " + tok)
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaCookieValue(c *check.C) {
	doVhostRequests(c, authzViaCookieValue)
}
func authzViaCookieValue(r *http.Request, tok string) int {
	r.AddCookie(&http.Cookie{
		Name: "api_token",
		Value: auth.EncodeTokenCookie([]byte(tok)),
	})
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaPath(c *check.C) {
	doVhostRequests(c, authzViaPath)
}
func authzViaPath(r *http.Request, tok string) int {
	r.URL.Path = "/t=" + tok + r.URL.Path
	return http.StatusNotFound
}

func (s *IntegrationSuite) TestVhostViaQueryString(c *check.C) {
	doVhostRequests(c, authzViaQueryString)
}
func authzViaQueryString(r *http.Request, tok string) int {
	r.URL.RawQuery = "api_token=" + tok
	return http.StatusUnauthorized
}

func (s *IntegrationSuite) TestVhostViaPOST(c *check.C) {
	doVhostRequests(c, authzViaPOST)
}
func authzViaPOST(r *http.Request, tok string) int {
	r.Method = "POST"
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	r.Body = ioutil.NopCloser(strings.NewReader(
		url.Values{"api_token": {tok}}.Encode()))
	return http.StatusUnauthorized
}

// Try some combinations of {url, token} using the given authorization
// mechanism, and verify the result is correct.
func doVhostRequests(c *check.C, authz authorizer) {
	hostPath := arvadostest.FooCollection + ".example.com/foo"
	for _, tok := range []string{
		arvadostest.ActiveToken,
		arvadostest.ActiveToken[:15],
		arvadostest.SpectatorToken,
		"bogus",
		"",
	} {
		u := mustParseURL("http://" + hostPath)
		req := &http.Request{
			Method: "GET",
			Host: u.Host,
			URL: u,
			Header: http.Header{},
		}
		failCode := authz(req, tok)
		resp := doReq(req)
		code, body := resp.Code, resp.Body.String()
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

func doReq(req *http.Request) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	(&handler{}).ServeHTTP(resp, req)
	if resp.Code != http.StatusSeeOther {
		return resp
	}
	cookies := (&http.Response{Header: resp.Header()}).Cookies()
	u, _ := req.URL.Parse(resp.Header().Get("Location"))
	req = &http.Request{
		Method: "GET",
		Host: u.Host,
		URL: u,
		Header: http.Header{},
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return doReq(req)
}

func (s *IntegrationSuite) TestVhostRedirectQueryTokenToCookie(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "GET",
		arvadostest.FooCollection + ".example.com/foo",
		"?api_token=" + arvadostest.ActiveToken,
		"text/plain",
		"",
		http.StatusOK,
	)
}

func (s *IntegrationSuite) TestVhostRedirectPOSTFormTokenToCookie(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "POST",
		arvadostest.FooCollection + ".example.com/foo",
		"",
		"application/x-www-form-urlencoded",
		url.Values{"api_token": {arvadostest.ActiveToken}}.Encode(),
		http.StatusOK,
	)
}

func (s *IntegrationSuite) TestVhostRedirectPOSTFormTokenToCookie404(c *check.C) {
	s.testVhostRedirectTokenToCookie(c, "POST",
		arvadostest.FooCollection + ".example.com/foo",
		"",
		"application/x-www-form-urlencoded",
		url.Values{"api_token": {arvadostest.SpectatorToken}}.Encode(),
		http.StatusNotFound,
	)
}

func (s *IntegrationSuite) testVhostRedirectTokenToCookie(c *check.C, method, hostPath, queryString, contentType, body string, expectStatus int) {
	u, _ := url.Parse(`http://` + hostPath + queryString)
	req := &http.Request{
		Method: method,
		Host: u.Host,
		URL: u,
		Header: http.Header{"Content-Type": {contentType}},
		Body: ioutil.NopCloser(strings.NewReader(body)),
	}

	resp := httptest.NewRecorder()
	(&handler{}).ServeHTTP(resp, req)
	c.Assert(resp.Code, check.Equals, http.StatusSeeOther)
	c.Check(resp.Body.String(), check.Matches, `.*href="//` + regexp.QuoteMeta(html.EscapeString(hostPath)) + `".*`)
	cookies := (&http.Response{Header: resp.Header()}).Cookies()

	u, _ = u.Parse(resp.Header().Get("Location"))
	req = &http.Request{
		Method: "GET",
		Host: u.Host,
		URL: u,
		Header: http.Header{},
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}

	resp = httptest.NewRecorder()
	(&handler{}).ServeHTTP(resp, req)
	c.Check(resp.Header().Get("Location"), check.Equals, "")
	c.Check(resp.Code, check.Equals, expectStatus)
	if expectStatus == http.StatusOK {
		c.Check(resp.Body.String(), check.Equals, "foo")
	}
}
