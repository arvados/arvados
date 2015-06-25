package main

import (
	"crypto/md5"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&IntegrationSuite{})

// IntegrationSuite tests need an API server and a keep-web server
type IntegrationSuite struct {
	testServer *server
}

func (s *IntegrationSuite) TestNoToken(c *check.C) {
	for _, token := range []string{
		"",
		"bogustoken",
	} {
		hdr, body := s.runCurl(c, token, "/collections/"+arvadostest.FooCollection+"/foo")
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 401 Unauthorized\r\n.*`)
		c.Check(body, check.Equals, "")

		if token != "" {
			hdr, body = s.runCurl(c, token, "/collections/download/"+arvadostest.FooCollection+"/"+token+"/foo")
			c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
			c.Check(body, check.Equals, "")
		}

		hdr, body = s.runCurl(c, token, "/bad-route")
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
		c.Check(body, check.Equals, "")
	}
}

// TODO: Move most cases to functional tests -- at least use Go's own
// http client instead of forking curl. Just leave enough of an
// integration test to assure that the documented way of invoking curl
// really works against the server.
func (s *IntegrationSuite) Test404(c *check.C) {
	for _, uri := range []string{
		// Routing errors
		"/",
		"/foo",
		"/download",
		"/collections",
		"/collections/",
		"/collections/" + arvadostest.FooCollection,
		"/collections/" + arvadostest.FooCollection + "/",
		// Non-existent file in collection
		"/collections/" + arvadostest.FooCollection + "/theperthcountyconspiracy",
		"/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/theperthcountyconspiracy",
		// Non-existent collection
		"/collections/" + arvadostest.NonexistentCollection,
		"/collections/" + arvadostest.NonexistentCollection + "/",
		"/collections/" + arvadostest.NonexistentCollection + "/theperthcountyconspiracy",
		"/collections/download/" + arvadostest.NonexistentCollection + "/" + arvadostest.ActiveToken + "/theperthcountyconspiracy",
	} {
		hdr, body := s.runCurl(c, arvadostest.ActiveToken, uri)
		c.Check(hdr, check.Matches, "(?s)HTTP/1.1 404 Not Found\r\n.*")
		c.Check(body, check.Equals, "")
	}
}

func (s *IntegrationSuite) Test200(c *check.C) {
	anonymousTokens = []string{arvadostest.AnonymousToken}
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)
	arv.ApiToken = arvadostest.ActiveToken
	kc, err := keepclient.MakeKeepClient(&arv)
	c.Assert(err, check.Equals, nil)
	kc.PutB([]byte("Hello world\n"))
	kc.PutB([]byte("foo"))
	for _, spec := range [][]string{
		// My collection
		{arvadostest.ActiveToken, "/collections/" + arvadostest.FooCollection + "/foo", "acbd18db4cc2f85cedef654fccc4a4d8"},
		{"", "/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/foo", "acbd18db4cc2f85cedef654fccc4a4d8"},
		{"tokensobogus", "/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/foo", "acbd18db4cc2f85cedef654fccc4a4d8"},
		{arvadostest.ActiveToken, "/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/foo", "acbd18db4cc2f85cedef654fccc4a4d8"},
		{arvadostest.AnonymousToken, "/collections/download/" + arvadostest.FooCollection + "/" + arvadostest.ActiveToken + "/foo", "acbd18db4cc2f85cedef654fccc4a4d8"},
		// Anonymously accessible user agreement. These should
		// start working when CollectionFileReader provides
		// real data instead of fake/stub data.
		{"", "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt", "f0ef7081e1539ac00ef5b761b4fb01b3"},
		{arvadostest.ActiveToken, "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt", "f0ef7081e1539ac00ef5b761b4fb01b3"},
		{arvadostest.SpectatorToken, "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt", "f0ef7081e1539ac00ef5b761b4fb01b3"},
		{arvadostest.SpectatorToken, "/collections/download/" + arvadostest.HelloWorldCollection + "/" + arvadostest.SpectatorToken + "/Hello%20world.txt", "f0ef7081e1539ac00ef5b761b4fb01b3"},
	} {
		hdr, body := s.runCurl(c, spec[0], spec[1])
		if strings.HasPrefix(hdr, "HTTP/1.1 501 Not Implemented\r\n") && body == "" {
			c.Log("Not implemented!")
			continue
		}
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 200 OK\r\n.*`)
		if strings.HasSuffix(spec[1], ".txt") {
			c.Check(hdr, check.Matches, `(?s).*\r\nContent-Type: text/plain.*`)
			// TODO: Check some types that aren't
			// automatically detected by Go's http server
			// by sniffing the content.
		}
		c.Check(fmt.Sprintf("%x", md5.Sum([]byte(body))), check.Equals, spec[2])
	}
}

// Return header block and body.
func (s *IntegrationSuite) runCurl(c *check.C, token, uri string, args ...string) (hdr, body string) {
	curlArgs := []string{"--silent", "--show-error", "--include"}
	if token != "" {
		curlArgs = append(curlArgs, "-H", "Authorization: OAuth2 "+token)
	}
	curlArgs = append(curlArgs, args...)
	curlArgs = append(curlArgs, "http://"+s.testServer.Addr+uri)
	c.Log(fmt.Sprintf("curlArgs == %#v", curlArgs))
	output, err := exec.Command("curl", curlArgs...).CombinedOutput()
	// Without "-f", curl exits 0 as long as it gets a valid HTTP
	// response from the server, even if the response status
	// indicates that the request failed. In our test suite, we
	// always expect a valid HTTP response, and we parse the
	// headers ourselves. If curl exits non-zero, our testing
	// environment is broken.
	c.Assert(err, check.Equals, nil)
	hdrsAndBody := strings.SplitN(string(output), "\r\n\r\n", 2)
	c.Assert(len(hdrsAndBody), check.Equals, 2)
	hdr = hdrsAndBody[0]
	body = hdrsAndBody[1]
	return
}

func (s *IntegrationSuite) SetUpSuite(c *check.C) {
	arvadostest.StartAPI()
	arvadostest.StartKeep()
}

func (s *IntegrationSuite) TearDownSuite(c *check.C) {
	arvadostest.StopKeep()
	arvadostest.StopAPI()
}

func (s *IntegrationSuite) SetUpTest(c *check.C) {
	arvadostest.ResetEnv()
	s.testServer = &server{}
	var err error
	address = "127.0.0.1:0"
	err = s.testServer.Start()
	c.Assert(err, check.Equals, nil)
}

func (s *IntegrationSuite) TearDownTest(c *check.C) {
	var err error
	if s.testServer != nil {
		err = s.testServer.Close()
	}
	c.Check(err, check.Equals, nil)
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
