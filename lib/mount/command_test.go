// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
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
	mountCmd := mountCommand{ready: make(chan struct{})}
	ready := false
	go func() {
		exited <- mountCmd.RunCommand("test mount", []string{"--experimental", s.mnt}, stdin, stdout, stderr)
	}()
	go func() {
		<-mountCmd.ready
		ready = true

		f, err := os.Open(s.mnt + "/by_id/" + arvadostest.FooCollection)
		if c.Check(err, check.IsNil) {
			dirnames, err := f.Readdirnames(-1)
			c.Check(err, check.IsNil)
			c.Check(dirnames, check.DeepEquals, []string{"foo"})
			f.Close()
		}

		buf, err := ioutil.ReadFile(s.mnt + "/by_id/" + arvadostest.FooCollection + "/.arvados#collection")
		if c.Check(err, check.IsNil) {
			var m map[string]interface{}
			err = json.Unmarshal(buf, &m)
			c.Check(err, check.IsNil)
			c.Check(m["manifest_text"], check.Matches, `\. acbd.* 0:3:foo\n`)
		}

		_, err = os.Open(s.mnt + "/by_id/zzzzz-4zz18-does-not-exist")
		c.Check(os.IsNotExist(err), check.Equals, true)

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

func (s *CmdSuite) TestCrunchstatLogger(c *check.C) {
	exited := make(chan int)
	stdin := bytes.NewBufferString("stdin")
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	mountCmd := mountCommand{ready: make(chan struct{})}
	go func() {
		exited <- mountCmd.RunCommand("test mount", []string{"--experimental", "--crunchstat-interval", "0.01", s.mnt}, stdin, stdout, stderr)
	}()
	go func() {
		<-mountCmd.ready

		// Check that crunchstat ticker is running
		time.Sleep(20 * time.Millisecond)
		logs := stderr.String()
		c.Check(strings.Contains(logs, "crunchstat"), check.Equals, true)

		ok := mountCmd.Unmount()
		c.Check(ok, check.Equals, true)

		// Check that logging has stopped
		stderrLen1 := stderr.Len()
		time.Sleep(100 * time.Millisecond)
		stderrLen2 := stderr.Len()
		c.Check(stderrLen2, check.Equals, stderrLen1)
	}()

	select {
	case <-time.After(5 * time.Second):
		c.Fatal("timed out")
	case errCode, ok := <-exited:
		c.Check(ok, check.Equals, true)
		c.Check(errCode, check.Equals, 0)
	}
}
