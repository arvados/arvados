package keepclient

import (
	"fmt"
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

func (s *MySuite) TestGetKeepDisks(c *C) {
	sr, err := KeepDisks()
	c.Assert(err, Equals, nil)
	c.Assert(len(sr), Equals, 2)
	c.Assert(sr[0], Equals, "http://localhost:25107")
	c.Assert(sr[1], Equals, "http://localhost:25108")

	service_roots := []string{"http://localhost:25107", "http://localhost:25108", "http://localhost:25109", "http://localhost:25110", "http://localhost:25111", "http://localhost:25112", "http://localhost:25113", "http://localhost:25114", "http://localhost:25115", "http://localhost:25116", "http://localhost:25117", "http://localhost:25118", "http://localhost:25119", "http://localhost:25120", "http://localhost:25121", "http://localhost:25122", "http://localhost:25123"}

	// "foo" acbd18db4cc2f85cedef654fccc4a4d8
	foo_shuffle := []string{"http://localhost:25116", "http://localhost:25120", "http://localhost:25119", "http://localhost:25122", "http://localhost:25108", "http://localhost:25114", "http://localhost:25112", "http://localhost:25107", "http://localhost:25118", "http://localhost:25111", "http://localhost:25113", "http://localhost:25121", "http://localhost:25110", "http://localhost:25117", "http://localhost:25109", "http://localhost:25115", "http://localhost:25123"}
	c.Check(ShuffledServiceRoots(service_roots, "acbd18db4cc2f85cedef654fccc4a4d8"), DeepEquals, foo_shuffle)

	// "bar" 37b51d194a7513e45b56f6524f2d51f2
	bar_shuffle := []string{"http://localhost:25108", "http://localhost:25112", "http://localhost:25119", "http://localhost:25107", "http://localhost:25110", "http://localhost:25116", "http://localhost:25122", "http://localhost:25120", "http://localhost:25121", "http://localhost:25117", "http://localhost:25111", "http://localhost:25123", "http://localhost:25118", "http://localhost:25113", "http://localhost:25114", "http://localhost:25115", "http://localhost:25109"}
	c.Check(ShuffledServiceRoots(service_roots, "37b51d194a7513e45b56f6524f2d51f2"), DeepEquals, bar_shuffle)
}
