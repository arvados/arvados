// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
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
	c.Check(cmd.Args, DeepEquals, []string{"./singularity", "exec", "--containall", "--cleanenv", "--pwd", "/WorkingDir", "--net", "--network=none", "--nv", "--bind", "/hostpath:/mnt:ro", "/fake/image.sif"})
	c.Check(cmd.Env, DeepEquals, []string{"SINGULARITYENV_FOO=bar"})
}
