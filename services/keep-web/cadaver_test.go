// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"io"
	"os/exec"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

func (s *IntegrationSuite) TestWebdavWithCadaver(c *check.C) {
	basePath := "/c=" + arvadostest.FooAndBarFilesInDirUUID + "/t=" + arvadostest.ActiveToken + "/"
	type testcase struct {
		path  string
		cmd   string
		match string
	}
	for _, trial := range []testcase{
		{
			path:  basePath,
			cmd:   "ls\n",
			match: `(?ms).*dir1 *0 .*`,
		},
		{
			path:  basePath,
			cmd:   "ls dir1\n",
			match: `(?ms).*bar *3.*foo *3 .*`,
		},
		{
			path:  basePath + "_/dir1",
			cmd:   "ls\n",
			match: `(?ms).*bar *3.*foo *3 .*`,
		},
		{
			path:  basePath + "dir1/",
			cmd:   "ls\n",
			match: `(?ms).*bar *3.*foo *3 .*`,
		},
	} {
		c.Logf("%s %#v", "http://"+s.testServer.Addr, trial)
		cmd := exec.Command("cadaver", "http://"+s.testServer.Addr+trial.path)
		cmd.Stdin = bytes.NewBufferString(trial.cmd)
		stdout, err := cmd.StdoutPipe()
		c.Assert(err, check.Equals, nil)
		cmd.Stderr = cmd.Stdout
		go cmd.Start()

		var buf bytes.Buffer
		_, err = io.Copy(&buf, stdout)
		c.Check(err, check.Equals, nil)
		err = cmd.Wait()
		c.Check(err, check.Equals, nil)
		c.Check(buf.String(), check.Matches, trial.match)
	}
}
