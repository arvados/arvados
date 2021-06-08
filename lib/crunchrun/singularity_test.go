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
