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
	mnt    string
	stderr *bytes.Buffer
}

func (s *CmdSuite) SetUpTest(c *check.C) {
	tmpdir, err := ioutil.TempDir("", "")
	c.Assert(err, check.IsNil)
	s.mnt = tmpdir
	s.stderr = bytes.NewBuffer(nil)
}

func (s *CmdSuite) TearDownTest(c *check.C) {
	c.Assert(os.RemoveAll(s.mnt), check.IsNil)
}

func (s *CmdSuite) TestMount(c *check.C) {
	s.mountAndCheck(c, []string{}, func() {
		f, err := os.Open(s.mnt + "/by_id/" + arvadostest.FooCollection)
		c.Assert(err, check.IsNil)
		dirnames, err := f.Readdirnames(-1)
		c.Assert(err, check.IsNil)
		c.Assert(dirnames, check.DeepEquals, []string{"foo"})
		f.Close()

		buf, err := ioutil.ReadFile(s.mnt + "/by_id/" + arvadostest.FooCollection + "/.arvados#collection")
		c.Assert(err, check.IsNil)
		var m map[string]interface{}
		err = json.Unmarshal(buf, &m)
		c.Assert(err, check.IsNil)
		c.Assert(m["manifest_text"], check.Matches, `\. acbd.* 0:3:foo\n`)

		_, err = os.Open(s.mnt + "/by_id/zzzzz-4zz18-does-not-exist")
		c.Assert(os.IsNotExist(err), check.Equals, true)
	})
}

func (s *CmdSuite) TestMountById(c *check.C) {
	s.mountAndCheck(c, []string{"--mount-by-id", "by_id_test"}, func() {
		f, err := os.Open(s.mnt + "/by_id_test/" + arvadostest.FooCollection)
		c.Assert(err, check.IsNil)
		dirnames, err := f.Readdirnames(-1)
		c.Assert(err, check.IsNil)
		c.Assert(dirnames, check.DeepEquals, []string{"foo"})
		f.Close()
	})
}

func (s *CmdSuite) TestCrunchstatLogger(c *check.C) {
	s.mountAndCheck(c, []string{"--crunchstat-interval", "0.01"}, func() {
		data := make([]byte, 2048)
		for i := range data {
			data[i] = byte(i % 256)
		}
		collectionPath := s.mnt + "/by_id/" + arvadostest.FooCollection + "/testfile"

		os.WriteFile(collectionPath, data, 0644)
		os.ReadFile(collectionPath)
		time.Sleep(20 * time.Millisecond)

		// Check that any logging has occurred
		logs := s.stderr.String()
		c.Assert(strings.Contains(logs, "blkio:0:0 2048 write 2048 read"), check.Equals, true)
		c.Assert(strings.Contains(logs, "crunchstat: fuseop:open 1 count"), check.Equals, true)
	})
}

func (s *CmdSuite) mountAndCheck(c *check.C, testArgs []string, testFunc func()) {
	exited := make(chan int)
	stdin := bytes.NewBufferString("stdin")
	stdout := bytes.NewBuffer(nil)
	mountCmd := mountCommand{ready: make(chan struct{})}
	ready := false
	args := append(append([]string{}, testArgs...), "-experimental", s.mnt)
	go func() {
		exited <- mountCmd.RunCommand("test mount", args, stdin, stdout, s.stderr)
	}()
	go func() {
		<-mountCmd.ready
		defer func() {
			ok := mountCmd.Unmount()
			c.Assert(ok, check.Equals, true)

			//If stderr was populated during the test, check that logging stops after unmount.
			if len(s.stderr.Bytes()) > 0 {
				len1 := s.stderr.Len()
				time.Sleep(100 * time.Millisecond)
				len2 := s.stderr.Len()
				c.Assert(len1, check.Equals, len2)
			}
		}()
		testFunc()
		ready = true
	}()
	select {
	case <-time.After(5 * time.Second):
		c.Fatal("timed out")
	case errCode, ok := <-exited:
		c.Assert(ok, check.Equals, true)
		c.Assert(errCode, check.Equals, 0)
	}
	c.Assert(ready, check.Equals, true)
	c.Assert(stdout.String(), check.Equals, "")
	// stdin should not have been read
	c.Assert(stdin.String(), check.Equals, "stdin")
}
