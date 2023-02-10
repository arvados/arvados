// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchstat

import (
	"bytes"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&suite{
	logger: logrus.New(),
})

type suite struct {
	logbuf bytes.Buffer
	logger *logrus.Logger
}

func (s *suite) SetUpSuite(c *C) {
	s.logger.Out = &s.logbuf
}

func (s *suite) SetUpTest(c *C) {
	s.logbuf.Reset()
}

func (s *suite) TestReadAllOrWarnFail(c *C) {
	rep := Reporter{Logger: s.logger}

	// The special file /proc/self/mem can be opened for
	// reading, but reading from byte 0 returns an error.
	f, err := os.Open("/proc/self/mem")
	c.Assert(err, IsNil)
	defer f.Close()
	_, err = rep.readAllOrWarn(f)
	c.Check(err, NotNil)
	c.Check(s.logbuf.String(), Matches, ".* msg=\"warning: read /proc/self/mem: .*\n")
}

func (s *suite) TestReadAllOrWarnSuccess(c *C) {
	rep := Reporter{Logger: s.logger}

	f, err := os.Open("./crunchstat_test.go")
	c.Assert(err, IsNil)
	defer f.Close()
	data, err := rep.readAllOrWarn(f)
	c.Check(err, IsNil)
	c.Check(string(data), Matches, "(?ms).*\npackage crunchstat\n.*")
	c.Check(s.logbuf.String(), Equals, "")
}

func (s *suite) TestReportPIDs(c *C) {
	r := Reporter{
		Logger:     s.logger,
		CgroupRoot: "/sys/fs/cgroup",
		PollPeriod: time.Second,
	}
	r.Start()
	r.ReportPID("init", 1)
	r.ReportPID("test_process", os.Getpid())
	r.ReportPID("nonexistent", 12345) // should be silently ignored/omitted
	for deadline := time.Now().Add(10 * time.Second); ; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			c.Error("timed out")
			break
		}
		if m := regexp.MustCompile(`(?ms).*procmem \d+ init (\d+) test_process.*`).FindSubmatch(s.logbuf.Bytes()); len(m) > 0 {
			size, err := strconv.ParseInt(string(m[1]), 10, 64)
			c.Check(err, IsNil)
			// Expect >1 MiB and <100 MiB -- otherwise we
			// are probably misinterpreting /proc/N/stat
			// or multiplying by the wrong page size.
			c.Check(size > 1000000, Equals, true)
			c.Check(size < 100000000, Equals, true)
			break
		}
	}
	c.Logf("%s", s.logbuf.String())
}
