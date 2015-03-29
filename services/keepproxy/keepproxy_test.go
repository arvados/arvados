package main

import (
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

// Gocheck boilerplate
var _ = Suite(&ServerRequiredSuite{})

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

// Wait (up to 1 second) for keepproxy to listen on a port. This
// avoids a race condition where we hit a "connection refused" error
// because we start testing the proxy too soon.
func waitForListener() {
	const (
		ms = 5
	)
	for i := 0; listener == nil && i < 1000; i += ms {
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
	arvadostest.StartKeep()
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep()
	arvadostest.StopAPI()
}

func setupProxyService() {

	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

	var req *http.Request
	var err error
	if req, err = http.NewRequest("POST", fmt.Sprintf("https://%s/arvados/v1/keep_services", os.Getenv("ARVADOS_API_HOST")), nil); err != nil {
		panic(err.Error())
	}
	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", os.Getenv("ARVADOS_API_TOKEN")))

	reader, writer := io.Pipe()

	req.Body = reader

	go func() {
		data := url.Values{}
		data.Set("keep_service", `{
  "service_host": "localhost",
  "service_port": 29950,
  "service_ssl_flag": false,
  "service_type": "proxy"
}`)

		writer.Write([]byte(data.Encode()))
		writer.Close()
	}()

	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		panic(err.Error())
	}
	if resp.StatusCode != 200 {
		panic(resp.Status)
	}
}

func runProxy(c *C, args []string, port int, bogusClientToken bool) keepclient.KeepClient {
	if bogusClientToken {
		os.Setenv("ARVADOS_API_TOKEN", "bogus-token")
	}
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)
	kc := keepclient.KeepClient{
		Arvados:       &arv,
		Want_replicas: 2,
		Using_proxy:   true,
		Client:        &http.Client{},
	}
	kc.SetServiceRoots(map[string]string{
		"proxy": fmt.Sprintf("http://localhost:%v", port),
	}, nil)
	c.Check(kc.Using_proxy, Equals, true)
	c.Check(len(kc.LocalRoots()), Equals, 1)
	for _, root := range kc.LocalRoots() {
		c.Check(root, Equals, fmt.Sprintf("http://localhost:%v", port))
	}
	log.Print("keepclient created")
	if bogusClientToken {
		arvadostest.ResetEnv()
	}

	{
		os.Args = append(args, fmt.Sprintf("-listen=:%v", port))
		listener = nil
		go main()
	}

	return kc
}

func (s *ServerRequiredSuite) TestPutAskGet(c *C) {
	log.Print("TestPutAndGet start")

	os.Args = []string{"keepproxy", "-listen=:29950"}
	listener = nil
	go main()
	time.Sleep(100 * time.Millisecond)

	setupProxyService()

	os.Setenv("ARVADOS_EXTERNAL_CLIENT", "true")
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)
	kc, err := keepclient.MakeKeepClient(&arv)
	c.Assert(err, Equals, nil)
	c.Check(kc.Arvados.External, Equals, true)
	c.Check(kc.Using_proxy, Equals, true)
	c.Check(len(kc.LocalRoots()), Equals, 1)
	for _, root := range kc.LocalRoots() {
		c.Check(root, Equals, "http://localhost:29950")
	}
	os.Setenv("ARVADOS_EXTERNAL_CLIENT", "")

	waitForListener()
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

	log.Print("TestPutAndGet done")
}

func (s *ServerRequiredSuite) TestPutAskGetForbidden(c *C) {
	log.Print("TestPutAskGetForbidden start")

	kc := runProxy(c, []string{"keepproxy"}, 29951, true)
	waitForListener()
	defer closeListener()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("bar")))

	{
		_, _, err := kc.Ask(hash)
		c.Check(err, Equals, keepclient.BlockNotFound)
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
		c.Assert(err, Equals, keepclient.BlockNotFound)
		c.Check(blocklen, Equals, int64(0))
		log.Print("Ask 2")
	}

	{
		_, blocklen, _, err := kc.Get(hash)
		c.Assert(err, Equals, keepclient.BlockNotFound)
		c.Check(blocklen, Equals, int64(0))
		log.Print("Get")
	}

	log.Print("TestPutAskGetForbidden done")
}

func (s *ServerRequiredSuite) TestGetDisabled(c *C) {
	log.Print("TestGetDisabled start")

	kc := runProxy(c, []string{"keepproxy", "-no-get"}, 29952, false)
	waitForListener()
	defer closeListener()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("baz")))

	{
		_, _, err := kc.Ask(hash)
		c.Check(err, Equals, keepclient.BlockNotFound)
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
		c.Assert(err, Equals, keepclient.BlockNotFound)
		c.Check(blocklen, Equals, int64(0))
		log.Print("Ask 2")
	}

	{
		_, blocklen, _, err := kc.Get(hash)
		c.Assert(err, Equals, keepclient.BlockNotFound)
		c.Check(blocklen, Equals, int64(0))
		log.Print("Get")
	}

	log.Print("TestGetDisabled done")
}

func (s *ServerRequiredSuite) TestPutDisabled(c *C) {
	log.Print("TestPutDisabled start")

	kc := runProxy(c, []string{"keepproxy", "-no-put"}, 29953, false)
	waitForListener()
	defer closeListener()

	{
		hash2, rep, err := kc.PutB([]byte("quux"))
		c.Check(hash2, Equals, "")
		c.Check(rep, Equals, 0)
		c.Check(err, Equals, keepclient.InsufficientReplicasError)
		log.Print("PutB")
	}

	log.Print("TestPutDisabled done")
}

func (s *ServerRequiredSuite) TestCorsHeaders(c *C) {
	runProxy(c, []string{"keepproxy"}, 29954, false)
	waitForListener()
	defer closeListener()

	{
		client := http.Client{}
		req, err := http.NewRequest("OPTIONS",
			fmt.Sprintf("http://localhost:29954/%x+3",
				md5.Sum([]byte("foo"))),
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
			fmt.Sprintf("http://localhost:29954/%x+3",
				md5.Sum([]byte("foo"))))
		c.Check(err, Equals, nil)
		c.Check(resp.Header.Get("Access-Control-Allow-Headers"), Equals, "Authorization, Content-Length, Content-Type, X-Keep-Desired-Replicas")
		c.Check(resp.Header.Get("Access-Control-Allow-Origin"), Equals, "*")
	}
}

func (s *ServerRequiredSuite) TestPostWithoutHash(c *C) {
	runProxy(c, []string{"keepproxy"}, 29955, false)
	waitForListener()
	defer closeListener()

	{
		client := http.Client{}
		req, err := http.NewRequest("POST",
			"http://localhost:29955/",
			strings.NewReader("qux"))
		req.Header.Add("Authorization", "OAuth2 4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
		req.Header.Add("Content-Type", "application/octet-stream")
		resp, err := client.Do(req)
		c.Check(err, Equals, nil)
		body, err := ioutil.ReadAll(resp.Body)
		c.Check(err, Equals, nil)
		c.Check(string(body), Equals,
			fmt.Sprintf("%x+%d", md5.Sum([]byte("qux")), 3))
	}
}
