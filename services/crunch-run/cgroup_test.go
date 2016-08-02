package main

import (
	. "gopkg.in/check.v1"
)

type CgroupSuite struct{}

var _ = Suite(&CgroupSuite{})

func (s *CgroupSuite) TestFindCgroup(c *C) {
	for _, s := range []string{"devices", "cpu", "cpuset"} {
		g := findCgroup(s)
		c.Check(g, Not(Equals), "")
		c.Logf("cgroup(%q) == %q", s, g)
	}
}
