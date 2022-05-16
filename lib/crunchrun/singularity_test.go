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
	// In production, executor will choose --network=bridge
	// because uid=0 under arvados-dispatch-cloud. But in test
	// cases, uid!=0, which means --network=bridge is conditional
	// on --fakeroot.
	uuc, err := os.ReadFile("/proc/sys/kernel/unprivileged_userns_clone")
	c.Check(err, check.IsNil)
	if string(uuc) == "0\n" {
		c.Skip("insufficient privileges to run this test case -- `singularity exec --fakeroot` requires /proc/sys/kernel/unprivileged_userns_clone = 1")
	}
	s.executor.(*singularityExecutor).fakeroot = true
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
	c.Check(cmd.Env, DeepEquals, []string{"SINGULARITYENV_FOO=bar"})
}
