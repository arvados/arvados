// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchstat

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

const logMsgPrefix = `(?m)(.*\n)*.* msg="`
const GiB = int64(1024 * 1024 * 1024)

type fakeStat struct {
	cgroupRoot string
	statName   string
	unit       string
	value      int64
}

var fakeRSS = fakeStat{
	cgroupRoot: "testdata/fakestat",
	statName:   "mem rss",
	unit:       "bytes",
	// Note this is the value of total_rss, not rss, because that's what should
	// always be reported for thresholds and maxima.
	value: 750 * 1024 * 1024,
}

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

func (s *suite) testRSSThresholds(c *C, rssPercentages []int64, alertCount int) {
	c.Assert(alertCount <= len(rssPercentages), Equals, true)
	rep := Reporter{
		CgroupRoot: fakeRSS.cgroupRoot,
		Logger:     s.logger,
		MemThresholds: map[string][]Threshold{
			"rss": NewThresholdsFromPercentages(GiB, rssPercentages),
		},
		PollPeriod:      time.Second * 10,
		ThresholdLogger: s.logger,
	}
	rep.Start()
	rep.Stop()
	logs := s.logbuf.String()
	c.Logf("%s", logs)

	for index, expectPercentage := range rssPercentages[:alertCount] {
		var logCheck Checker
		if index < alertCount {
			logCheck = Matches
		} else {
			logCheck = Not(Matches)
		}
		pattern := fmt.Sprintf(`%sContainer using over %d%% of memory \(rss %d/%d bytes\)"`,
			logMsgPrefix, expectPercentage, fakeRSS.value, GiB)
		c.Check(logs, logCheck, pattern)
	}
}

func (s *suite) TestZeroRSSThresholds(c *C) {
	s.testRSSThresholds(c, []int64{}, 0)
}

func (s *suite) TestOneRSSThresholdPassed(c *C) {
	s.testRSSThresholds(c, []int64{55}, 1)
}

func (s *suite) TestOneRSSThresholdNotPassed(c *C) {
	s.testRSSThresholds(c, []int64{85}, 0)
}

func (s *suite) TestMultipleRSSThresholdsNonePassed(c *C) {
	s.testRSSThresholds(c, []int64{95, 97, 99}, 0)
}

func (s *suite) TestMultipleRSSThresholdsSomePassed(c *C) {
	s.testRSSThresholds(c, []int64{60, 70, 80, 90}, 2)
}

func (s *suite) TestMultipleRSSThresholdsAllPassed(c *C) {
	s.testRSSThresholds(c, []int64{1, 2, 3}, 3)
}
