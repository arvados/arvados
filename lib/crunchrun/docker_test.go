// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"os/exec"
	"time"

	. "gopkg.in/check.v1"
)

var _ = Suite(&dockerSuite{})

type dockerSuite struct {
	executorSuite
}

func (s *dockerSuite) SetUpSuite(c *C) {
	_, err := exec.LookPath("docker")
	if err != nil {
		c.Skip("looks like docker is not installed")
	}
	s.newExecutor = func(c *C) {
		exec.Command("docker", "rm", "zzzzz-zzzzz-zzzzzzzzzzzzzzz").Run()
		var err error
		s.executor, err = newDockerExecutor("zzzzz-zzzzz-zzzzzzzzzzzzzzz", c.Logf, time.Second/2)
		c.Assert(err, IsNil)
	}
}
