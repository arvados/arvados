package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"net"
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
		hdr, body, _ := s.runCurl(c, token, "dl.example.com", "/collections/"+arvadostest.FooCollection+"/foo")
		c.Check(hdr, check.Matches, `(?s)HTTP/1.1 401 Unauthorized\r\n.*`)
		c.Check(body, check.Equals, "")

		if token != "" {
			hdr, body, _ = s.runCurl(c, token, "dl.example.com", "/collections/download/"+arvadostest.FooCollection+"/"+token+"/foo")
			c.Check(hdr, check.Matches, `(?s)HTTP/1.1 404 Not Found\r\n.*`)
			c.Check(body, check.Equals, "")
		}

		hdr, body, _ = s.runCurl(c, token, "dl.example.com", "/bad-route")
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
		hdr, body, _ := s.runCurl(c, arvadostest.ActiveToken, "dl.example.com", uri)
		c.Check(hdr, check.Matches, "(?s)HTTP/1.1 404 Not Found\r\n.*")
		c.Check(body, check.Equals, "")
	}
}

func (s *IntegrationSuite) Test1GBFile(c *check.C) {
	if testing.Short() {
		c.Skip("skipping 1GB integration test in short mode")
	}
	s.test100BlockFile(c, 10000000)
}

func (s *IntegrationSuite) Test300MBFile(c *check.C) {
	s.test100BlockFile(c, 3000000)
}

func (s *IntegrationSuite) test100BlockFile(c *check.C, blocksize int) {
	testdata := make([]byte, blocksize)
	for i := 0; i < blocksize; i++ {
		testdata[i] = byte(' ')
	}
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)
	arv.ApiToken = arvadostest.ActiveToken
	kc, err := keepclient.MakeKeepClient(&arv)
	c.Assert(err, check.Equals, nil)
	loc, _, err := kc.PutB(testdata[:])
	c.Assert(err, check.Equals, nil)
	mtext := "."
	for i := 0; i < 100; i++ {
		mtext = mtext + " " + loc
	}
	mtext = mtext + fmt.Sprintf(" 0:%d00:testdata.bin\n", blocksize)
	coll := map[string]interface{}{}
	err = arv.Create("collections",
		map[string]interface{}{
			"collection": map[string]interface{}{
				"name": fmt.Sprintf("testdata blocksize=%d", blocksize),
				"manifest_text": mtext,
			},
		}, &coll)
	c.Assert(err, check.Equals, nil)
	uuid := coll["uuid"].(string)

	hdr, body, size := s.runCurl(c, arv.ApiToken, uuid + ".dl.example.com", "/testdata.bin")
	c.Check(hdr, check.Matches, `(?s)HTTP/1.1 200 OK\r\n.*`)
	c.Check(hdr, check.Matches, `(?si).*Content-length: `+fmt.Sprintf("%d00", blocksize)+`\r\n.*`)
	c.Check([]byte(body)[:1234], check.DeepEquals, testdata[:1234])
	c.Check(size, check.Equals, int64(blocksize)*100)
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
		// Anonymously accessible user agreement.
		{"", "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt", "f0ef7081e1539ac00ef5b761b4fb01b3"},
		{arvadostest.ActiveToken, "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt", "f0ef7081e1539ac00ef5b761b4fb01b3"},
		{arvadostest.SpectatorToken, "/collections/" + arvadostest.HelloWorldCollection + "/Hello%20world.txt", "f0ef7081e1539ac00ef5b761b4fb01b3"},
		{arvadostest.SpectatorToken, "/collections/download/" + arvadostest.HelloWorldCollection + "/" + arvadostest.SpectatorToken + "/Hello%20world.txt", "f0ef7081e1539ac00ef5b761b4fb01b3"},
	} {
		hdr, body, _ := s.runCurl(c, spec[0], "dl.example.com", spec[1])
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
func (s *IntegrationSuite) runCurl(c *check.C, token, host, uri string, args ...string) (hdr, bodyPart string, bodySize int64) {
	curlArgs := []string{"--silent", "--show-error", "--include"}
	testHost, testPort, _ := net.SplitHostPort(s.testServer.Addr)
	curlArgs = append(curlArgs, "--resolve", host + ":" + testPort + ":" + testHost)
	if token != "" {
		curlArgs = append(curlArgs, "-H", "Authorization: OAuth2 "+token)
	}
	curlArgs = append(curlArgs, args...)
	curlArgs = append(curlArgs, "http://"+host+":"+testPort+uri)
	c.Log(fmt.Sprintf("curlArgs == %#v", curlArgs))
	cmd := exec.Command("curl", curlArgs...)
	stdout, err := cmd.StdoutPipe()
	c.Assert(err, check.Equals, nil)
	cmd.Stderr = cmd.Stdout
	go cmd.Start()
	buf := make([]byte, 2<<27)
	n, err := io.ReadFull(stdout, buf)
	// Discard (but measure size of) anything past 128 MiB.
	var discarded int64
	if err == io.ErrUnexpectedEOF {
		err = nil
		buf = buf[:n]
	} else {
		c.Assert(err, check.Equals, nil)
		discarded, err = io.Copy(ioutil.Discard, stdout)
		c.Assert(err, check.Equals, nil)
	}
	err = cmd.Wait()
	// Without "-f", curl exits 0 as long as it gets a valid HTTP
	// response from the server, even if the response status
	// indicates that the request failed. In our test suite, we
	// always expect a valid HTTP response, and we parse the
	// headers ourselves. If curl exits non-zero, our testing
	// environment is broken.
	c.Assert(err, check.Equals, nil)
	hdrsAndBody := strings.SplitN(string(buf), "\r\n\r\n", 2)
	c.Assert(len(hdrsAndBody), check.Equals, 2)
	hdr = hdrsAndBody[0]
	bodyPart = hdrsAndBody[1]
	bodySize = int64(len(bodyPart)) + discarded
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
