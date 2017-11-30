// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"testing"

	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

type TestSuite struct {
}

var _ = Suite(&TestSuite{})

func (s *TestSuite) TestVersion(c *C) {
	// Default version string when Version is not set
	c.Assert(Version, Equals, "")
	c.Assert(GetVersion(), Equals, "dev")
	// Simulate linker flag setting Version var
	Version = "1.0.0"
	c.Assert(GetVersion(), Equals, "1.0.0")
}
