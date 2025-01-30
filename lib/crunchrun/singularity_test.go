// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"os"
	"os/exec"

	. "gopkg.in/check.v1"
)

var _ = Suite(&singularitySuite{})

type singularitySuite struct {
	executorSuite
}

func (s *singularitySuite) SetUpSuite(c *C) {
	_, err := exec.LookPath("singularity")
	if err != nil {
		c.Skip("looks like singularity is not installed")
	}
	s.newExecutor = func(c *C) {
		var err error
		s.executor, err = newSingularityExecutor(c.Logf)
		c.Assert(err, IsNil)
	}
}

func (s *singularitySuite) TearDownSuite(c *C) {
	if s.executor != nil {
		s.executor.Close()
	}
}

func (s *singularitySuite) TestEnableNetwork_Listen(c *C) {
	// With modern iptables, singularity (as of 4.2.1) cannot
	// enable networking when invoked by a regular user. Under
	// arvados-dispatch-cloud, crunch-run runs as root, so it's
	// OK. For testing, assuming tests are not running as root, we
	// use sudo -- but only if requested via environment variable.
	if os.Getuid() == 0 {
		// already root
	} else if os.Getenv("ARVADOS_TEST_PRIVESC") == "sudo" {
		c.Logf("ARVADOS_TEST_PRIVESC is 'sudo', invoking 'sudo singularity ...'")
		s.executor.(*singularityExecutor).sudo = true
	} else {
		c.Skip("test case needs to run singularity as root -- set ARVADOS_TEST_PRIVESC=sudo to enable this test")
	}
	s.executorSuite.TestEnableNetwork_Listen(c)
}

func (s *singularitySuite) TestInject(c *C) {
	path, err := exec.LookPath("nsenter")
	if err != nil || path != "/var/lib/arvados/bin/nsenter" {
		c.Skip("looks like /var/lib/arvados/bin/nsenter is not installed -- re-run `arvados-server install`?")
	}
	s.executorSuite.TestInject(c)
}

var _ = Suite(&singularityStubSuite{})

// singularityStubSuite tests don't really invoke singularity, so we
// can run them even if singularity is not installed.
type singularityStubSuite struct{}

func (s *singularityStubSuite) TestSingularityExecArgs(c *C) {
	e, err := newSingularityExecutor(c.Logf)
	c.Assert(err, IsNil)
	err = e.Create(containerSpec{
		WorkingDir:     "/WorkingDir",
		Env:            map[string]string{"FOO": "bar"},
		BindMounts:     map[string]bindmount{"/mnt": {HostPath: "/hostpath", ReadOnly: true}},
		EnableNetwork:  false,
		GPUStack:       "cuda",
		GPUDeviceCount: 3,
		VCPUs:          2,
		RAM:            12345678,
	})
	c.Check(err, IsNil)
	e.imageFilename = "/fake/image.sif"
	cmd := e.execCmd("./singularity")
	expectArgs := []string{"./singularity", "exec", "--containall", "--cleanenv", "--pwd=/WorkingDir", "--net", "--network=none", "--nv"}
	if cgroupSupport["cpu"] {
		expectArgs = append(expectArgs, "--cpus", "2")
	}
	if cgroupSupport["memory"] {
		expectArgs = append(expectArgs, "--memory", "12345678")
	}
	expectArgs = append(expectArgs, "--bind", "/hostpath:/mnt:ro", "/fake/image.sif")
	c.Check(cmd.Args, DeepEquals, expectArgs)
	c.Check(cmd.Env, DeepEquals, []string{
		"SINGULARITYENV_FOO=bar",
		"SINGULARITY_NO_EVAL=1",
		"XDG_RUNTIME_DIR=" + os.Getenv("XDG_RUNTIME_DIR"),
		"DBUS_SESSION_BUS_ADDRESS=" + os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
	})
}
