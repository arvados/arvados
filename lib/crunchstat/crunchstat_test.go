// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchstat

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&suite{})

type suite struct{}

func bufLogger() (*log.Logger, *bufio.Reader) {
	r, w := io.Pipe()
	logger := log.New(w, "", 0)
	return logger, bufio.NewReader(r)
}

func (s *suite) TestReadAllOrWarnFail(c *C) {
	logger, rcv := bufLogger()
	rep := Reporter{Logger: logger}

	done := make(chan bool)
	var msg []byte
	var err error
	go func() {
		msg, err = rcv.ReadBytes('\n')
		close(done)
	}()
	{
		// The special file /proc/self/mem can be opened for
		// reading, but reading from byte 0 returns an error.
		f, err := os.Open("/proc/self/mem")
		if err != nil {
			c.Fatalf("Opening /proc/self/mem: %s", err)
		}
		if x, err := rep.readAllOrWarn(f); err == nil {
			c.Fatalf("Expected error, got %v", x)
		}
	}
	<-done
	if err != nil {
		c.Fatal(err)
	} else if matched, err := regexp.MatchString("^warning: read /proc/self/mem: .*", string(msg)); err != nil || !matched {
		c.Fatalf("Expected error message about unreadable file, got \"%s\"", msg)
	}
}

func (s *suite) TestReadAllOrWarnSuccess(c *C) {
	rep := Reporter{Logger: log.New(os.Stderr, "", 0)}

	f, err := os.Open("./crunchstat_test.go")
	if err != nil {
		c.Fatalf("Opening ./crunchstat_test.go: %s", err)
	}
	data, err := rep.readAllOrWarn(f)
	if err != nil {
		c.Fatalf("got error %s", err)
	}
	if matched, err := regexp.MatchString("\npackage crunchstat\n", string(data)); err != nil || !matched {
		c.Fatalf("data failed regexp: err %v, matched %v", err, matched)
	}
}

func (s *suite) TestReportPIDs(c *C) {
	var logbuf bytes.Buffer
	logger := logrus.New()
	logger.Out = &logbuf
	r := Reporter{
		Logger:     logger,
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
		if regexp.MustCompile(`(!?ms).*procmem \d+ init \d+ test_process.*`).MatchString(logbuf.String()) {
			break
		}
	}
	c.Logf("%s", logbuf.String())
}
