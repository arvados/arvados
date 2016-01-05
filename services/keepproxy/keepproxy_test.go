package main

import (
	"crypto/md5"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&ServerRequiredSuite{})

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

// Gocheck boilerplate
var _ = Suite(&NoKeepServerSuite{})

// Test with no keepserver to simulate errors
type NoKeepServerSuite struct{}

var TestProxyUUID = "zzzzz-bi6l4-lrixqc4fxofbmzz"

// Wait (up to 1 second) for keepproxy to listen on a port. This
// avoids a race condition where we hit a "connection refused" error
// because we start testing the proxy too soon.
func waitForListener() {
	const (
		ms = 5
	)
	for i := 0; listener == nil && i < 10000; i += ms {
		time.Sleep(ms * time.Millisecond)
	}
	if listener == nil {
		log.Fatalf("Timed out waiting for listener to start")
	}
}

func closeListener() {
	if listener != nil {
		listener.Close()
	}
}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	arvadostest.StartAPI()
	arvadostest.StartKeep(2, false)
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep(2)
	arvadostest.StopAPI()
}

func (s *NoKeepServerSuite) SetUpSuite(c *C) {
	arvadostest.StartAPI()
	// We need API to have some keep services listed, but the
	// services themselves should be unresponsive.
	arvadostest.StartKeep(2, false)
	arvadostest.StopKeep(2)
}

func (s *NoKeepServerSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *NoKeepServerSuite) TearDownSuite(c *C) {
	arvadostest.StopAPI()
}

func runProxy(c *C, args []string, bogusClientToken bool) *keepclient.KeepClient {
	args = append([]string{"keepproxy"}, args...)
	os.Args = append(args, "-listen=:0")
	listener = nil
	go main()
	waitForListener()

	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)
	if bogusClientToken {
		arv.ApiToken = "bogus-token"
	}
	kc := keepclient.New(&arv)
	sr := map[string]string{
		TestProxyUUID: "http://" + listener.Addr().String(),
	}
	kc.SetServiceRoots(sr, sr, sr)
	kc.Arvados.External = true

	return kc
}

func (s *ServerRequiredSuite) TestPutAskGet(c *C) {
	kc := runProxy(c, nil, false)
	defer closeListener()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	var hash2 string

	{
		_, _, err := kc.Ask(hash)
		c.Check(err, Equals, keepclient.BlockNotFound)
		log.Print("Finished Ask (expected BlockNotFound)")
	}

	{
		reader, _, _, err := kc.Get(hash)
		c.Check(reader, Equals, nil)
		c.Check(err, Equals, keepclient.BlockNotFound)
		log.Print("Finished Get (expected BlockNotFound)")
	}

	// Note in bug #5309 among other errors keepproxy would set
	// Content-Length incorrectly on the 404 BlockNotFound response, this
	// would result in a protocol violation that would prevent reuse of the
	// connection, which would manifest by the next attempt to use the
	// connection (in this case the PutB below) failing.  So to test for
	// that bug it's necessary to trigger an error response (such as
	// BlockNotFound) and then do something else with the same httpClient
	// connection.

	{
		var rep int
		var err error
		hash2, rep, err = kc.PutB([]byte("foo"))
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+3(\+.+)?$`, hash))
		c.Check(rep, Equals, 2)
		c.Check(err, Equals, nil)
		log.Print("Finished PutB (expected success)")
	}

	{
		blocklen, _, err := kc.Ask(hash2)
		c.Assert(err, Equals, nil)
		c.Check(blocklen, Equals, int64(3))
		log.Print("Finished Ask (expected success)")
	}

	{
		reader, blocklen, _, err := kc.Get(hash2)
		c.Assert(err, Equals, nil)
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, []byte("foo"))
		c.Check(blocklen, Equals, int64(3))
		log.Print("Finished Get (expected success)")
	}

	{
		var rep int
		var err error
		hash2, rep, err = kc.PutB([]byte(""))
		c.Check(hash2, Matches, `^d41d8cd98f00b204e9800998ecf8427e\+0(\+.+)?$`)
		c.Check(rep, Equals, 2)
		c.Check(err, Equals, nil)
		log.Print("Finished PutB zero block")
	}

	{
		reader, blocklen, _, err := kc.Get("d41d8cd98f00b204e9800998ecf8427e")
		c.Assert(err, Equals, nil)
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, []byte(""))
		c.Check(blocklen, Equals, int64(0))
		log.Print("Finished Get zero block")
	}
}

func (s *ServerRequiredSuite) TestPutAskGetForbidden(c *C) {
	kc := runProxy(c, nil, true)
	defer closeListener()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("bar")))

	{
		_, _, err := kc.Ask(hash)
		errNotFound, _ := err.(keepclient.ErrNotFound)
		c.Check(errNotFound, NotNil)
		c.Assert(strings.Contains(err.Error(), "HTTP 403"), Equals, true)
		log.Print("Ask 1")
	}

	{
		hash2, rep, err := kc.PutB([]byte("bar"))
		c.Check(hash2, Equals, "")
		c.Check(rep, Equals, 0)
		c.Check(err, Equals, keepclient.InsufficientReplicasError)
		log.Print("PutB")
	}

	{
		blocklen, _, err := kc.Ask(hash)
		errNotFound, _ := err.(keepclient.ErrNotFound)
		c.Check(errNotFound, NotNil)
		c.Assert(strings.Contains(err.Error(), "HTTP 403"), Equals, true)
		c.Check(blocklen, Equals, int64(0))
		log.Print("Ask 2")
	}

	{
		_, blocklen, _, err := kc.Get(hash)
		errNotFound, _ := err.(keepclient.ErrNotFound)
		c.Check(errNotFound, NotNil)
		c.Assert(strings.Contains(err.Error(), "HTTP 403"), Equals, true)
		c.Check(blocklen, Equals, int64(0))
		log.Print("Get")
	}
}

func (s *ServerRequiredSuite) TestGetDisabled(c *C) {
	kc := runProxy(c, []string{"-no-get"}, false)
	defer closeListener()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("baz")))

	{
		_, _, err := kc.Ask(hash)
		errNotFound, _ := err.(keepclient.ErrNotFound)
		c.Check(errNotFound, NotNil)
		c.Assert(strings.Contains(err.Error(), "HTTP 400"), Equals, true)
		log.Print("Ask 1")
	}

	{
		hash2, rep, err := kc.PutB([]byte("baz"))
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+3(\+.+)?$`, hash))
		c.Check(rep, Equals, 2)
		c.Check(err, Equals, nil)
		log.Print("PutB")
	}

	{
		blocklen, _, err := kc.Ask(hash)
		errNotFound, _ := err.(keepclient.ErrNotFound)
		c.Check(errNotFound, NotNil)
		c.Assert(strings.Contains(err.Error(), "HTTP 400"), Equals, true)
		c.Check(blocklen, Equals, int64(0))
		log.Print("Ask 2")
	}

	{
		_, blocklen, _, err := kc.Get(hash)
		errNotFound, _ := err.(keepclient.ErrNotFound)
		c.Check(errNotFound, NotNil)
		c.Assert(strings.Contains(err.Error(), "HTTP 400"), Equals, true)
		c.Check(blocklen, Equals, int64(0))
		log.Print("Get")
	}
}

func (s *ServerRequiredSuite) TestPutDisabled(c *C) {
	kc := runProxy(c, []string{"-no-put"}, false)
	defer closeListener()

	hash2, rep, err := kc.PutB([]byte("quux"))
	c.Check(hash2, Equals, "")
	c.Check(rep, Equals, 0)
	c.Check(err, Equals, keepclient.InsufficientReplicasError)
}

func (s *ServerRequiredSuite) TestCorsHeaders(c *C) {
	runProxy(c, nil, false)
	defer closeListener()

	{
		client := http.Client{}
		req, err := http.NewRequest("OPTIONS",
			fmt.Sprintf("http://%s/%x+3", listener.Addr().String(), md5.Sum([]byte("foo"))),
			nil)
		req.Header.Add("Access-Control-Request-Method", "PUT")
		req.Header.Add("Access-Control-Request-Headers", "Authorization, X-Keep-Desired-Replicas")
		resp, err := client.Do(req)
		c.Check(err, Equals, nil)
		c.Check(resp.StatusCode, Equals, 200)
		body, err := ioutil.ReadAll(resp.Body)
		c.Check(string(body), Equals, "")
		c.Check(resp.Header.Get("Access-Control-Allow-Methods"), Equals, "GET, HEAD, POST, PUT, OPTIONS")
		c.Check(resp.Header.Get("Access-Control-Allow-Origin"), Equals, "*")
	}

	{
		resp, err := http.Get(
			fmt.Sprintf("http://%s/%x+3", listener.Addr().String(), md5.Sum([]byte("foo"))))
		c.Check(err, Equals, nil)
		c.Check(resp.Header.Get("Access-Control-Allow-Headers"), Equals, "Authorization, Content-Length, Content-Type, X-Keep-Desired-Replicas")
		c.Check(resp.Header.Get("Access-Control-Allow-Origin"), Equals, "*")
	}
}

func (s *ServerRequiredSuite) TestPostWithoutHash(c *C) {
	runProxy(c, nil, false)
	defer closeListener()

	{
		client := http.Client{}
		req, err := http.NewRequest("POST",
			"http://"+listener.Addr().String()+"/",
			strings.NewReader("qux"))
		req.Header.Add("Authorization", "OAuth2 4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
		req.Header.Add("Content-Type", "application/octet-stream")
		resp, err := client.Do(req)
		c.Check(err, Equals, nil)
		body, err := ioutil.ReadAll(resp.Body)
		c.Check(err, Equals, nil)
		c.Check(string(body), Matches,
			fmt.Sprintf(`^%x\+3(\+.+)?$`, md5.Sum([]byte("qux"))))
	}
}

func (s *ServerRequiredSuite) TestStripHint(c *C) {
	c.Check(removeHint.ReplaceAllString("http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73+K@zzzzz", "$1"),
		Equals,
		"http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73")
	c.Check(removeHint.ReplaceAllString("http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+K@zzzzz+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73", "$1"),
		Equals,
		"http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73")
	c.Check(removeHint.ReplaceAllString("http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73+K@zzzzz-zzzzz-zzzzzzzzzzzzzzz", "$1"),
		Equals,
		"http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73+K@zzzzz-zzzzz-zzzzzzzzzzzzzzz")
	c.Check(removeHint.ReplaceAllString("http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+K@zzzzz-zzzzz-zzzzzzzzzzzzzzz+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73", "$1"),
		Equals,
		"http://keep.zzzzz.arvadosapi.com:25107/2228819a18d3727630fa30c81853d23f+67108864+K@zzzzz-zzzzz-zzzzzzzzzzzzzzz+A37b6ab198qqqq28d903b975266b23ee711e1852c@55635f73")

}

// Test GetIndex
//   Put one block, with 2 replicas
//   With no prefix (expect the block locator, twice)
//   With an existing prefix (expect the block locator, twice)
//   With a valid but non-existing prefix (expect "\n")
//   With an invalid prefix (expect error)
func (s *ServerRequiredSuite) TestGetIndex(c *C) {
	kc := runProxy(c, nil, false)
	defer closeListener()

	// Put "index-data" blocks
	data := []byte("index-data")
	hash := fmt.Sprintf("%x", md5.Sum(data))

	hash2, rep, err := kc.PutB(data)
	c.Check(hash2, Matches, fmt.Sprintf(`^%s\+10(\+.+)?$`, hash))
	c.Check(rep, Equals, 2)
	c.Check(err, Equals, nil)

	reader, blocklen, _, err := kc.Get(hash)
	c.Assert(err, Equals, nil)
	c.Check(blocklen, Equals, int64(10))
	all, err := ioutil.ReadAll(reader)
	c.Check(all, DeepEquals, data)

	// Put some more blocks
	_, rep, err = kc.PutB([]byte("some-more-index-data"))
	c.Check(err, Equals, nil)

	kc.Arvados.ApiToken = arvadostest.DataManagerToken

	// Invoke GetIndex
	for _, spec := range []struct {
		prefix         string
		expectTestHash bool
		expectOther    bool
	}{
		{"", true, true},         // with no prefix
		{hash[:3], true, false},  // with matching prefix
		{"abcdef", false, false}, // with no such prefix
	} {
		indexReader, err := kc.GetIndex(TestProxyUUID, spec.prefix)
		c.Assert(err, Equals, nil)
		indexResp, err := ioutil.ReadAll(indexReader)
		c.Assert(err, Equals, nil)
		locators := strings.Split(string(indexResp), "\n")
		gotTestHash := 0
		gotOther := 0
		for _, locator := range locators {
			if locator == "" {
				continue
			}
			c.Check(locator[:len(spec.prefix)], Equals, spec.prefix)
			if locator[:32] == hash {
				gotTestHash++
			} else {
				gotOther++
			}
		}
		c.Check(gotTestHash == 2, Equals, spec.expectTestHash)
		c.Check(gotOther > 0, Equals, spec.expectOther)
	}

	// GetIndex with invalid prefix
	_, err = kc.GetIndex(TestProxyUUID, "xyz")
	c.Assert((err != nil), Equals, true)
}

func (s *ServerRequiredSuite) TestPutAskGetInvalidToken(c *C) {
	kc := runProxy(c, nil, false)
	defer closeListener()

	// Put a test block
	hash, rep, err := kc.PutB([]byte("foo"))
	c.Check(err, Equals, nil)
	c.Check(rep, Equals, 2)

	for _, token := range []string{
		"nosuchtoken",
		"2ym314ysp27sk7h943q6vtc378srb06se3pq6ghurylyf3pdmx", // expired
	} {
		// Change token to given bad token
		kc.Arvados.ApiToken = token

		// Ask should result in error
		_, _, err = kc.Ask(hash)
		c.Check(err, NotNil)
		errNotFound, _ := err.(keepclient.ErrNotFound)
		c.Check(errNotFound.Temporary(), Equals, false)
		c.Assert(strings.Contains(err.Error(), "HTTP 403"), Equals, true)

		// Get should result in error
		_, _, _, err = kc.Get(hash)
		c.Check(err, NotNil)
		errNotFound, _ = err.(keepclient.ErrNotFound)
		c.Check(errNotFound.Temporary(), Equals, false)
		c.Assert(strings.Contains(err.Error(), "HTTP 403 \"Missing or invalid Authorization header\""), Equals, true)
	}
}

func (s *ServerRequiredSuite) TestAskGetKeepProxyConnectionError(c *C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)

	// keepclient with no such keep server
	kc := keepclient.New(&arv)
	locals := map[string]string{
		TestProxyUUID: "http://localhost:12345",
	}
	kc.SetServiceRoots(locals, nil, nil)

	// Ask should result in temporary connection refused error
	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	_, _, err = kc.Ask(hash)
	c.Check(err, NotNil)
	errNotFound, _ := err.(*keepclient.ErrNotFound)
	c.Check(errNotFound.Temporary(), Equals, true)
	c.Assert(strings.Contains(err.Error(), "connection refused"), Equals, true)

	// Get should result in temporary connection refused error
	_, _, _, err = kc.Get(hash)
	c.Check(err, NotNil)
	errNotFound, _ = err.(*keepclient.ErrNotFound)
	c.Check(errNotFound.Temporary(), Equals, true)
	c.Assert(strings.Contains(err.Error(), "connection refused"), Equals, true)
}

func (s *NoKeepServerSuite) TestAskGetNoKeepServerError(c *C) {
	kc := runProxy(c, nil, false)
	defer closeListener()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	for _, f := range []func() error{
		func() error {
			_, _, err := kc.Ask(hash)
			return err
		},
		func() error {
			_, _, _, err := kc.Get(hash)
			return err
		},
	} {
		err := f()
		c.Assert(err, NotNil)
		errNotFound, _ := err.(*keepclient.ErrNotFound)
		c.Check(errNotFound.Temporary(), Equals, true)
		c.Check(err, ErrorMatches, `.*HTTP 502.*`)
	}
}
