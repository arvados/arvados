package main

import (
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"crypto/md5"
	"crypto/tls"
	"fmt"
	. "gopkg.in/check.v1"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

func pythonDir() string {
	cwd, _ := os.Getwd()
	return fmt.Sprintf("%s/../../sdk/python/tests", cwd)
}

// Wait (up to 1 second) for keepproxy to listen on a port. This
// avoids a race condition where we hit a "connection refused" error
// because we start testing the proxy too soon.
func waitForListener() {
	const (ms = 5)
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
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	os.Chdir(pythonDir())
	{
		cmd := exec.Command("python", "run_test_server.py", "start")
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalf("Setting up stderr pipe: %s", err)
		}
		go io.Copy(os.Stderr, stderr)
		if err := cmd.Run(); err != nil {
			panic(fmt.Sprintf("'python run_test_server.py start' returned error %s", err))
		}
	}
	{
		cmd := exec.Command("python", "run_test_server.py", "start_keep")
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatalf("Setting up stderr pipe: %s", err)
		}
		go io.Copy(os.Stderr, stderr)
		if err := cmd.Run(); err != nil {
			panic(fmt.Sprintf("'python run_test_server.py start_keep' returned error %s", err))
		}
	}

	os.Setenv("ARVADOS_API_HOST", "localhost:3000")
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	os.Chdir(pythonDir())
	exec.Command("python", "run_test_server.py", "stop_keep").Run()
	exec.Command("python", "run_test_server.py", "stop").Run()
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

func runProxy(c *C, args []string, token string, port int) keepclient.KeepClient {
	os.Args = append(args, fmt.Sprintf("-listen=:%v", port))
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")

	go main()
	time.Sleep(100 * time.Millisecond)

	os.Setenv("ARVADOS_KEEP_PROXY", fmt.Sprintf("http://localhost:%v", port))
	os.Setenv("ARVADOS_API_TOKEN", token)
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, Equals, nil)
	kc, err := keepclient.MakeKeepClient(&arv)
	c.Assert(err, Equals, nil)
	c.Check(kc.Using_proxy, Equals, true)
	c.Check(len(kc.ServiceRoots()), Equals, 1)
	for _, root := range(kc.ServiceRoots()) {
		c.Check(root, Equals, fmt.Sprintf("http://localhost:%v", port))
	}
	os.Setenv("ARVADOS_KEEP_PROXY", "")
	log.Print("keepclient created")
	return kc
}

func (s *ServerRequiredSuite) TestPutAskGet(c *C) {
	log.Print("TestPutAndGet start")

	os.Args = []string{"keepproxy", "-listen=:29950"}
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
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
	c.Check(len(kc.ServiceRoots()), Equals, 1)
	for _, root := range kc.ServiceRoots() {
		c.Check(root, Equals, "http://localhost:29950")
	}
	os.Setenv("ARVADOS_EXTERNAL_CLIENT", "")
	log.Print("keepclient created")

	waitForListener()
	defer closeListener()

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))
	var hash2 string

	{
		_, _, err := kc.Ask(hash)
		c.Check(err, Equals, keepclient.BlockNotFound)
		log.Print("Ask 1")
	}

	{
		var rep int
		var err error
		hash2, rep, err = kc.PutB([]byte("foo"))
		c.Check(hash2, Matches, fmt.Sprintf(`^%s\+3(\+.+)?$`, hash))
		c.Check(rep, Equals, 2)
		c.Check(err, Equals, nil)
		log.Print("PutB")
	}

	{
		blocklen, _, err := kc.Ask(hash2)
		c.Assert(err, Equals, nil)
		c.Check(blocklen, Equals, int64(3))
		log.Print("Ask 2")
	}

	{
		reader, blocklen, _, err := kc.Get(hash2)
		c.Assert(err, Equals, nil)
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, []byte("foo"))
		c.Check(blocklen, Equals, int64(3))
		log.Print("Get")
	}

	log.Print("TestPutAndGet done")
}

func (s *ServerRequiredSuite) TestPutAskGetForbidden(c *C) {
	log.Print("TestPutAndGet start")

	kc := runProxy(c, []string{"keepproxy"}, "123abc", 29951)
	waitForListener()
	defer closeListener()

	log.Print("keepclient created")

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

	log.Print("TestPutAndGetForbidden done")
}

func (s *ServerRequiredSuite) TestGetDisabled(c *C) {
	log.Print("TestGetDisabled start")

	kc := runProxy(c, []string{"keepproxy", "-no-get"}, "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h", 29952)
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

	kc := runProxy(c, []string{"keepproxy", "-no-put"}, "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h", 29953)
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
