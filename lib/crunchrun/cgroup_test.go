// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

type CgroupSuite struct{}

var _ = Suite(&CgroupSuite{})

func (s *CgroupSuite) TestFindCgroup(c *C) {
	var testfiles []string
	buf, err := exec.Command("find", "../crunchstat/testdata", "-name", "cgroup", "-type", "f").Output()
	c.Assert(err, IsNil)
	for _, testfile := range bytes.Split(buf, []byte{'\n'}) {
		if len(testfile) > 0 {
			testfiles = append(testfiles, string(testfile))
		}
	}
	testfiles = append(testfiles, "/proc/self/cgroup")

	tmpdir := c.MkDir()
	err = os.MkdirAll(tmpdir+"/proc/self", 0777)
	c.Assert(err, IsNil)
	fsys := os.DirFS(tmpdir)

	for _, trial := range []struct {
		match  string // if non-empty, only check testfiles containing this string
		subsys string
		expect string // empty means "any" (we never actually expect empty string)
	}{
		{"debian11", "blkio", "/user.slice/user-1000.slice/session-5424.scope"},
		{"debian12", "cpuacct", "/user.slice/user-1000.slice/session-4.scope"},
		{"debian12", "bogus-does-not-matter", "/user.slice/user-1000.slice/session-4.scope"},
		{"ubuntu1804", "blkio", "/user.slice"},
		{"ubuntu1804", "cpuacct", "/user.slice"},
		{"", "cpu", ""},
		{"", "cpuset", ""},
		{"", "devices", ""},
		{"", "bogus-does-not-matter", ""},
	} {
		for _, testfile := range testfiles {
			if !strings.Contains(testfile, trial.match) {
				continue
			}
			c.Logf("trial %+v testfile %s", trial, testfile)

			// Copy cgroup file into our fake proc/self/ dir
			buf, err := os.ReadFile(testfile)
			c.Assert(err, IsNil)
			err = os.WriteFile(tmpdir+"/proc/self/cgroup", buf, 0777)
			c.Assert(err, IsNil)

			cgroup, err := findCgroup(fsys, trial.subsys)
			if !c.Check(err, IsNil) {
				continue
			}
			c.Logf("\tcgroup = %q", cgroup)
			c.Check(cgroup, Not(Equals), "")
			if trial.expect != "" {
				c.Check(cgroup, Equals, trial.expect)
			}
		}
	}
}

func (s *CgroupSuite) TestCgroupSupport(c *C) {
	var logbuf bytes.Buffer
	logger := logrus.New()
	logger.Out = &logbuf
	checkCgroupSupport(logger.Printf)
	c.Check(logbuf.String(), Equals, "")
	c.Check(cgroupSupport, NotNil)
	c.Check(cgroupSupport["memory"], Equals, true)
	c.Check(cgroupSupport["entropy"], Equals, false)
}

func (s *CgroupSuite) TestCgroup1Support(c *C) {
	tmpdir := c.MkDir()
	err := os.MkdirAll(tmpdir+"/proc/self", 0777)
	c.Assert(err, IsNil)
	err = os.WriteFile(tmpdir+"/proc/self/cgroup", []byte(`12:blkio:/user.slice
11:perf_event:/
10:freezer:/
9:pids:/user.slice/user-1000.slice/session-5.scope
8:hugetlb:/
7:rdma:/
6:cpu,cpuacct:/user.slice
5:devices:/user.slice
4:memory:/user.slice/user-1000.slice/session-5.scope
3:net_cls,net_prio:/
2:cpuset:/
1:name=systemd:/user.slice/user-1000.slice/session-5.scope
0::/user.slice/user-1000.slice/session-5.scope
`), 0777)
	c.Assert(err, IsNil)
	cgroupSupport = map[string]bool{}
	ok := checkCgroup1Support(os.DirFS(tmpdir), c.Logf)
	c.Check(ok, Equals, true)
	c.Check(cgroupSupport, DeepEquals, map[string]bool{
		"blkio":        true,
		"cpu":          true,
		"cpuacct":      true,
		"cpuset":       true,
		"devices":      true,
		"freezer":      true,
		"hugetlb":      true,
		"memory":       true,
		"name=systemd": true,
		"net_cls":      true,
		"net_prio":     true,
		"perf_event":   true,
		"pids":         true,
		"rdma":         true,
	})
}
