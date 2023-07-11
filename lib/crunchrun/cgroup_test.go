// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"fmt"
	"os/exec"

	. "gopkg.in/check.v1"
)

type CgroupSuite struct{}

var _ = Suite(&CgroupSuite{})

func (s *CgroupSuite) TestFindCgroup(c *C) {
	if buf, err := exec.Command("stat", "-ftc", "%T", "/sys/fs/cgroup").CombinedOutput(); err != nil {
		c.Skip(fmt.Sprintf("cannot stat /sys/fs/cgroup: %s", err))
	} else if string(buf) == "cgroup2fs\n" {
		c.Skip("cannot test cgroups v1 feature because this system is using cgroups v2 unified mode")
	}
	for _, s := range []string{"devices", "cpu", "cpuset"} {
		g, err := findCgroup(s)
		if c.Check(err, IsNil) {
			c.Check(g, Not(Equals), "", Commentf("subsys %q", s))
		}
		c.Logf("cgroup(%q) == %q", s, g)
	}
}
