// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"bytes"
	"io/ioutil"
	"os"
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CmdSuite{})

type CmdSuite struct {
	mnt string
}

func (s *CmdSuite) SetUpTest(c *check.C) {
	tmpdir, err := ioutil.TempDir("", "")
	c.Assert(err, check.IsNil)
	s.mnt = tmpdir
}

func (s *CmdSuite) TearDownTest(c *check.C) {
	c.Check(os.RemoveAll(s.mnt), check.IsNil)
}

func (s *CmdSuite) TestMount(c *check.C) {
	exited := make(chan int)
	stdin := bytes.NewBufferString("stdin")
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	mountCmd := cmd{ready: make(chan struct{})}
	ready := false
	go func() {
		exited <- mountCmd.RunCommand("test mount", []string{"--experimental", s.mnt}, stdin, stdout, stderr)
	}()
	go func() {
		<-mountCmd.ready
		ready = true
		ok := mountCmd.Unmount()
		c.Check(ok, check.Equals, true)
	}()
	select {
	case <-time.After(5 * time.Second):
		c.Fatal("timed out")
	case errCode, ok := <-exited:
		c.Check(ok, check.Equals, true)
		c.Check(errCode, check.Equals, 0)
	}
	c.Check(ready, check.Equals, true)
	c.Check(stdout.String(), check.Equals, "")
	// stdin should not have been read
	c.Check(stdin.String(), check.Equals, "stdin")
}
