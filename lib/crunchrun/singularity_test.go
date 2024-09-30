// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"os"
	"os/exec"

	. "gopkg.in/check.v1"
	check "gopkg.in/check.v1"
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
	uuc, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	c.Assert(err, check.IsNil)
	if string(uuc) == "0\n" {
		c.Skip("insufficient privileges to run singularity tests -- `singularity exec --fakeroot` requires /proc/sys/kernel/unprivileged_userns_clone = 1")
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

func (s *singularitySuite) TestIPAddress(c *C) {
	// With modern iptables, singularity (as of 4.2.1) cannot
	// enable networking when invoked by a regular user. Under
	// arvados-dispatch-cloud, crunch-run runs as root, so it's
	// OK. For testing, assuming tests are not running as root, we
	// use sudo -- but only if requested via environment variable.
	if os.Getuid() != 0 {
		if os.Getenv("ARVADOS_TEST_USE_SUDO") != "" {
			s.executor.(*singularityExecutor).sudo = true
		} else {
			c.Skip("test case needs to run singularity as root -- set ARVADOS_TEST_USE_SUDO=1 to enable this test")
		}
	}
	s.executorSuite.TestIPAddress(c)
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
		WorkingDir:      "/WorkingDir",
		Env:             map[string]string{"FOO": "bar"},
		BindMounts:      map[string]bindmount{"/mnt": {HostPath: "/hostpath", ReadOnly: true}},
		EnableNetwork:   false,
		CUDADeviceCount: 3,
	})
	c.Check(err, IsNil)
	e.imageFilename = "/fake/image.sif"
	cmd := e.execCmd("./singularity")
	c.Check(cmd.Args, DeepEquals, []string{"./singularity", "exec", "--containall", "--cleanenv", "--pwd=/WorkingDir", "--net", "--network=none", "--nv", "--bind", "/hostpath:/mnt:ro", "/fake/image.sif"})
	c.Check(cmd.Env, DeepEquals, []string{"SINGULARITYENV_FOO=bar", "SINGULARITY_NO_EVAL=1"})
}
