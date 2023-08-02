// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchstat

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

const logMsgPrefix = `(?m)(.*\n)*.* msg="`

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&suite{})

type testdatasource struct {
	fspath string
	pid    int
}

func (s testdatasource) Pid() int {
	return s.pid
}
func (s testdatasource) FS() fs.FS {
	return os.DirFS(s.fspath)
}

// To generate a test case for a new OS target, build
// cmd/arvados-server and run
//
//	arvados-server crunchstat -dump ./testdata/example1234 sleep 2
var testdata = map[string]testdatasource{
	"debian11":   {fspath: "testdata/debian11", pid: 4153022},
	"debian12":   {fspath: "testdata/debian12", pid: 1115883},
	"ubuntu1804": {fspath: "testdata/ubuntu1804", pid: 2523},
	"ubuntu2004": {fspath: "testdata/ubuntu2004", pid: 1360},
	"ubuntu2204": {fspath: "testdata/ubuntu2204", pid: 1967},
}

type suite struct {
	logbuf                bytes.Buffer
	logger                *logrus.Logger
	debian12MemoryCurrent int64
}

func (s *suite) SetUpSuite(c *C) {
	s.logger = logrus.New()
	s.logger.Out = &s.logbuf

	buf, err := os.ReadFile("testdata/debian12/sys/fs/cgroup/user.slice/user-1000.slice/session-4.scope/memory.current")
	c.Assert(err, IsNil)
	_, err = fmt.Sscanf(string(buf), "%d", &s.debian12MemoryCurrent)
	c.Assert(err, IsNil)
}

func (s *suite) SetUpTest(c *C) {
	s.logbuf.Reset()
}

// Report stats for the current (go test) process's cgroup, using the
// test host's real procfs/sysfs.
func (s *suite) TestReportCurrent(c *C) {
	r := Reporter{
		Pid:        os.Getpid,
		Logger:     s.logger,
		PollPeriod: time.Second,
	}
	r.Start()
	defer r.Stop()
	checkPatterns := []string{
		`(?ms).*rss.*`,
		`(?ms).*net:.*`,
		`(?ms).*blkio:.*`,
		`(?ms).* [\d.]+ user [\d.]+ sys ` + fmt.Sprintf("%d", runtime.NumCPU()) + ` cpus -- .*`,
	}
	for deadline := time.Now().Add(4 * time.Second); !c.Failed(); time.Sleep(time.Millisecond) {
		done := true
		for _, pattern := range checkPatterns {
			if m := regexp.MustCompile(pattern).FindSubmatch(s.logbuf.Bytes()); len(m) == 0 {
				done = false
				if time.Now().After(deadline) {
					c.Errorf("timed out waiting for %s", pattern)
				}
			}
		}
		if done {
			break
		}
	}
	c.Logf("%s", s.logbuf.String())
}

// Report stats for a the current (go test) process.
func (s *suite) TestReportPIDs(c *C) {
	r := Reporter{
		Pid:        func() int { return 1 },
		Logger:     s.logger,
		PollPeriod: time.Second,
	}
	r.Start()
	defer r.Stop()
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

func (s *suite) TestAllTestdata(c *C) {
	for platform, datasource := range testdata {
		s.logbuf.Reset()
		c.Logf("=== %s", platform)
		rep := Reporter{
			Pid:             datasource.Pid,
			FS:              datasource.FS(),
			Logger:          s.logger,
			PollPeriod:      time.Second,
			ThresholdLogger: s.logger,
			Debug:           true,
		}
		rep.Start()
		rep.Stop()
		logs := s.logbuf.String()
		c.Logf("%s", logs)
		c.Check(logs, Matches, `(?ms).* \d\d+ rss\\n.*`)
		c.Check(logs, Matches, `(?ms).*blkio:\d+:\d+ \d+ write \d+ read\\n.*`)
		c.Check(logs, Matches, `(?ms).*net:\S+ \d+ tx \d+ rx\\n.*`)
		c.Check(logs, Matches, `(?ms).* [\d.]+ user [\d.]+ sys [2-9]\d* cpus.*`)
	}
}

func (s *suite) testRSSThresholds(c *C, rssPercentages []int64, alertCount int) {
	c.Assert(alertCount <= len(rssPercentages), Equals, true)
	rep := Reporter{
		Pid:    testdata["debian12"].Pid,
		FS:     testdata["debian12"].FS(),
		Logger: s.logger,
		MemThresholds: map[string][]Threshold{
			"rss": NewThresholdsFromPercentages(s.debian12MemoryCurrent*3/2, rssPercentages),
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
			logMsgPrefix, expectPercentage, s.debian12MemoryCurrent, s.debian12MemoryCurrent*3/2)
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
	s.testRSSThresholds(c, []int64{45, 60, 75, 90}, 2)
}

func (s *suite) TestMultipleRSSThresholdsAllPassed(c *C) {
	s.testRSSThresholds(c, []int64{1, 2, 3}, 3)
}

func (s *suite) TestLogMaxima(c *C) {
	rep := Reporter{
		Pid:        testdata["debian12"].Pid,
		FS:         testdata["debian12"].FS(),
		Logger:     s.logger,
		PollPeriod: time.Second * 10,
		TempDir:    "/",
	}
	rep.Start()
	rep.Stop()
	rep.LogMaxima(s.logger, map[string]int64{"rss": s.debian12MemoryCurrent * 3 / 2})
	logs := s.logbuf.String()
	c.Logf("%s", logs)

	expectRSS := fmt.Sprintf(`Maximum container memory rss usage was %d%%, %d/%d bytes`,
		66, s.debian12MemoryCurrent, s.debian12MemoryCurrent*3/2)
	for _, expected := range []string{
		`Maximum disk usage was \d+%, \d+/\d+ bytes`,
		`Maximum container memory swap usage was \d\d+ bytes`,
		`Maximum container memory pgmajfault usage was \d\d+ faults`,
		expectRSS,
	} {
		pattern := logMsgPrefix + expected + `"`
		c.Check(logs, Matches, pattern)
	}
}

func (s *suite) TestLogProcessMemMax(c *C) {
	rep := Reporter{
		Pid:        os.Getpid,
		Logger:     s.logger,
		PollPeriod: time.Second * 10,
	}
	rep.ReportPID("test-run", os.Getpid())
	rep.Start()
	rep.Stop()
	rep.LogProcessMemMax(s.logger)
	logs := s.logbuf.String()
	c.Logf("%s", logs)

	pattern := logMsgPrefix + `Maximum test-run memory rss usage was \d+ bytes"`
	c.Check(logs, Matches, pattern)
}
