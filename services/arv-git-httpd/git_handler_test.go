package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&GitHandlerSuite{})

type GitHandlerSuite struct {}

func (s *GitHandlerSuite) TestEnvVars(c *check.C) {
	u, err := url.Parse("git.zzzzz.arvadosapi.com/test")
	c.Check(err, check.Equals, nil)
	resp := httptest.NewRecorder()
	req := &http.Request{
		Method: "GET",
		URL: u,
		RemoteAddr: "[::1]:12345",
	}
	h := newGitHandler()
	h.(*gitHandler).Path = "/bin/sh"
	h.(*gitHandler).Args = []string{"-c", "echo HTTP/1.1 200 OK; echo Content-Type: text/plain; echo; env"}
	os.Setenv("GITOLITE_HTTP_HOME", "/test/ghh")
	os.Setenv("GL_BYPASS_ACCESS_CHECKS", "yesplease")

	h.ServeHTTP(resp, req)

	c.Check(resp.Code, check.Equals, http.StatusOK)
	body := resp.Body.String()
	c.Check(body, check.Matches, `(?ms).*^GITOLITE_HTTP_HOME=/test/ghh$.*`)
	c.Check(body, check.Matches, `(?ms).*^GL_BYPASS_ACCESS_CHECKS=yesplease$.*`)
	c.Check(body, check.Matches, `(?ms).*^REMOTE_HOST=::1$.*`)
	c.Check(body, check.Matches, `(?ms).*^REMOTE_PORT=12345$.*`)
	c.Check(body, check.Matches, `(?ms).*^SERVER_ADDR=` + regexp.QuoteMeta(theConfig.Addr) + `$.*`)
}

func (s *GitHandlerSuite) TestCGIError(c *check.C) {
	u, err := url.Parse("git.zzzzz.arvadosapi.com/test")
	c.Check(err, check.Equals, nil)
	resp := httptest.NewRecorder()
	req := &http.Request{
		Method: "GET",
		URL: u,
		RemoteAddr: "bogus",
	}
	h := newGitHandler()
	h.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusInternalServerError)
	c.Check(resp.Body.String(), check.Equals, "")
}
