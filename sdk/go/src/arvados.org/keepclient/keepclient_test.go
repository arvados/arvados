package keepclient

import (
	. "gopkg.in/check.v1"
	"os"
	"os/exec"
	"testing"
)

// Gocheck boilerplate
func Test(t *testing.T) { TestingT(t) }

// Gocheck boilerplate
var _ = Suite(&MySuite{})

// Our test fixture
type MySuite struct{}

func (s *MySuite) SetUpSuite(c *C) {
	os.Chdir(os.ExpandEnv("$GOPATH../python"))
	exec.Command("python", "run_test_server.py", "start").Run()
}

func (s *MySuite) TearDownSuite(c *C) {
	os.Chdir(os.ExpandEnv("$GOPATH../python"))
	exec.Command("python", "run_test_server.py", "stop").Run()
}

func (s *MySuite) TestInit(c *C) {
	os.Setenv("ARVADOS_API_HOST", "localhost:3001")
	os.Setenv("ARVADOS_API_TOKEN", "12345")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "")
	kc := InitKeepClient()
	c.Assert(kc.apiServer, Equals, "localhost:3001")
	c.Assert(kc.apiToken, Equals, "12345")
	c.Assert(kc.apiInsecure, Equals, false)

	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")
	kc = InitKeepClient()
	c.Assert(kc.apiServer, Equals, "localhost:3001")
	c.Assert(kc.apiToken, Equals, "12345")
	c.Assert(kc.apiInsecure, Equals, true)
}
