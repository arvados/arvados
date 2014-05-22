package main

import (
	"arvados.org/keepclient"
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

func pythonDir() string {
	gopath := os.Getenv("GOPATH")
	return fmt.Sprintf("%s/../../sdk/python", strings.Split(gopath, ":")[0])
}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	os.Chdir(pythonDir())
	exec.Command("python", "run_test_server.py", "start").Run()
	exec.Command("python", "run_test_server.py", "start_keep").Run()

	os.Setenv("ARVADOS_API_HOST", "localhost:3001")
	os.Setenv("ARVADOS_API_TOKEN", "4axaw8zxe0qm22wa6urpp5nskcne8z88cvbupv653y1njyi05h")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")

	SetupProxyService()
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)

	os.Chdir(pythonDir())
	exec.Command("python", "run_test_server.py", "stop_keep").Run()
	exec.Command("python", "run_test_server.py", "stop").Run()
}

func SetupProxyService() {

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

func (s *ServerRequiredSuite) TestPutAskGet(c *C) {
	log.Print("TestPutAndGet start")

	os.Setenv("ARVADOS_EXTERNAL_CLIENT", "true")
	kc, err := keepclient.MakeKeepClient()
	c.Check(kc.External, Equals, true)
	c.Check(kc.Using_proxy, Equals, true)
	c.Check(len(kc.Service_roots), Equals, 1)
	c.Check(kc.Service_roots[0], Equals, "http://localhost:29950")
	c.Check(err, Equals, nil)
	os.Setenv("ARVADOS_EXTERNAL_CLIENT", "")

	log.Print("keepclient created")

	os.Args = []string{"keepproxy", "-listen=:29950"}
	go main()

	time.Sleep(100 * time.Millisecond)

	log.Print("keepproxy main started")

	hash := fmt.Sprintf("%x", md5.Sum([]byte("foo")))

	// Uncomment this when actual keep server supports HEAD
	/*{
		_, _, err := kc.Ask(hash)
		c.Check(err, Equals, keepclient.BlockNotFound)
		log.Print("Ask 1")
	}*/

	{
		hash2, rep, err := kc.PutB([]byte("foo"))
		c.Check(hash2, Equals, hash)
		c.Check(rep, Equals, 2)
		c.Check(err, Equals, nil)
		log.Print("PutB")
	}

	// Uncomment this when actual keep server supports HEAD
	/*{
		blocklen, _, err := kc.Ask(hash)
		c.Check(blocklen, Equals, int64(3))
		c.Check(err, Equals, nil)
		log.Print("Ask 2")
	}*/

	{
		reader, blocklen, _, err := kc.Get(hash)
		all, err := ioutil.ReadAll(reader)
		c.Check(all, DeepEquals, []byte("foo"))
		c.Check(blocklen, Equals, int64(3))
		c.Check(err, Equals, nil)
		log.Print("Get")
	}

	// Close internal listener socket.
	listener.Close()

	log.Print("TestPutAndGet done")
}
