// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	. "gopkg.in/check.v1"
)

type CgroupSuite struct{}

var _ = Suite(&CgroupSuite{})

func (s *CgroupSuite) TestFindCgroup(c *C) {
	for _, s := range []string{"devices", "cpu", "cpuset"} {
		g, err := findCgroup(s)
		if c.Check(err, IsNil) {
			c.Check(g, Not(Equals), "", Commentf("subsys %q", s))
		}
		c.Logf("cgroup(%q) == %q", s, g)
	}
}
