package main

import (
	"arvados.org/keepclient"
	"fmt"
	. "gopkg.in/check.v1"
	"os"
	"os/exec"
	"strings"
	"testing"
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
	return fmt.Sprintf("%s/../python", strings.Split(gopath, ":")[0])
}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	os.Chdir(pythonDir())
	exec.Command("python", "run_test_server.py", "start").Run()
	exec.Command("python", "run_test_server.py", "start_keep").Run()
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	os.Chdir(pythonDir())
	exec.Command("python", "run_test_server.py", "stop_keep").Run()
	exec.Command("python", "run_test_server.py", "stop").Run()
}

func (s *ServerRequiredSuite) TestPutAndGet(c *C) {
	kc := keepclient.KeepClient{"localhost", "", true, 29950, nil, 2, nil, true}
}
