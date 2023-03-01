// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchstat

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
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
	cgroupRoot string
	logbuf     bytes.Buffer
	logger     *logrus.Logger
}

func (s *suite) SetUpSuite(c *C) {
	s.logger.Out = &s.logbuf
}

func (s *suite) SetUpTest(c *C) {
	s.cgroupRoot = ""
	s.logbuf.Reset()
}

func (s *suite) tempCgroup(c *C, sourceDir string) error {
	tempDir := c.MkDir()
	dirents, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, dirent := range dirents {
		srcData, err := os.ReadFile(path.Join(sourceDir, dirent.Name()))
		if err != nil {
			return err
		}
		destPath := path.Join(tempDir, dirent.Name())
		err = os.WriteFile(destPath, srcData, 0o600)
		if err != nil {
			return err
		}
	}
	s.cgroupRoot = tempDir
	return nil
}

func (s *suite) addPidToCgroup(pid int) error {
	if s.cgroupRoot == "" {
		return errors.New("cgroup has not been set up for this test")
	}
	procsPath := path.Join(s.cgroupRoot, "cgroup.procs")
	procsFile, err := os.OpenFile(procsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	pidLine := strconv.Itoa(pid) + "\n"
	_, err = procsFile.Write([]byte(pidLine))
	if err != nil {
		procsFile.Close()
		return err
	}
	return procsFile.Close()
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

func (s *suite) TestLogMaxima(c *C) {
	err := s.tempCgroup(c, fakeRSS.cgroupRoot)
	c.Assert(err, IsNil)
	rep := Reporter{
		CgroupRoot: s.cgroupRoot,
		Logger:     s.logger,
		PollPeriod: time.Second * 10,
		TempDir:    s.cgroupRoot,
	}
	rep.Start()
	rep.Stop()
	rep.LogMaxima(s.logger, map[string]int64{"rss": GiB})
	logs := s.logbuf.String()
	c.Logf("%s", logs)

	expectRSS := fmt.Sprintf(`Maximum container memory rss usage was %d%%, %d/%d bytes`,
		100*fakeRSS.value/GiB, fakeRSS.value, GiB)
	for _, expected := range []string{
		`Maximum disk usage was \d+%, \d+/\d+ bytes`,
		`Maximum container memory cache usage was 73400320 bytes`,
		`Maximum container memory swap usage was 320 bytes`,
		`Maximum container memory pgmajfault usage was 20 faults`,
		expectRSS,
	} {
		pattern := logMsgPrefix + expected + `"`
		c.Check(logs, Matches, pattern)
	}
}

func (s *suite) TestLogProcessMemMax(c *C) {
	err := s.tempCgroup(c, fakeRSS.cgroupRoot)
	c.Assert(err, IsNil)
	pid := os.Getpid()
	err = s.addPidToCgroup(pid)
	c.Assert(err, IsNil)

	rep := Reporter{
		CgroupRoot: s.cgroupRoot,
		Logger:     s.logger,
		PollPeriod: time.Second * 10,
		TempDir:    s.cgroupRoot,
	}
	rep.ReportPID("test-run", pid)
	rep.Start()
	rep.Stop()
	rep.LogProcessMemMax(s.logger)
	logs := s.logbuf.String()
	c.Logf("%s", logs)

	pattern := logMsgPrefix + `Maximum test-run memory rss usage was \d+ bytes"`
	c.Check(logs, Matches, pattern)
}
