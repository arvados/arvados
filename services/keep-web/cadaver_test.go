// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

func (s *IntegrationSuite) TestWebdavWithCadaver(c *check.C) {
	testdata := []byte("the human tragedy consists in the necessity of living with the consequences of actions performed under the pressure of compulsions we do not understand")

	localfile, err := ioutil.TempFile("", "localfile")
	c.Assert(err, check.IsNil)
	defer os.Remove(localfile.Name())
	localfile.Write(testdata)

	emptyfile, err := ioutil.TempFile("", "emptyfile")
	c.Assert(err, check.IsNil)
	defer os.Remove(emptyfile.Name())

	checkfile, err := ioutil.TempFile("", "checkfile")
	c.Assert(err, check.IsNil)
	defer os.Remove(checkfile.Name())

	var newCollection arvados.Collection
	arv := arvados.NewClientFromEnv()
	arv.AuthToken = arvadostest.ActiveToken
	err = arv.RequestAndDecode(&newCollection, "POST", "/arvados/v1/collections", bytes.NewBufferString(url.Values{"collection": {"{}"}}.Encode()), nil)
	c.Assert(err, check.IsNil)
	writePath := "/c=" + newCollection.UUID + "/t=" + arv.AuthToken + "/"

	readPath := "/c=" + arvadostest.FooAndBarFilesInDirUUID + "/t=" + arvadostest.ActiveToken + "/"
	type testcase struct {
		path  string
		cmd   string
		match string
		data  []byte
	}
	for _, trial := range []testcase{
		{
			path:  readPath,
			cmd:   "ls\n",
			match: `(?ms).*dir1 *0 .*`,
		},
		{
			path:  readPath,
			cmd:   "ls dir1\n",
			match: `(?ms).*bar *3.*foo *3 .*`,
		},
		{
			path:  readPath + "_/dir1",
			cmd:   "ls\n",
			match: `(?ms).*bar *3.*foo *3 .*`,
		},
		{
			path:  readPath + "dir1/",
			cmd:   "ls\n",
			match: `(?ms).*bar *3.*foo *3 .*`,
		},
		{
			path:  writePath,
			cmd:   "get emptyfile '" + checkfile.Name() + "'\n",
			match: `(?ms).*Not Found.*`,
		},
		{
			path:  writePath,
			cmd:   "put '" + emptyfile.Name() + "' emptyfile\n",
			match: `(?ms).*Uploading .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "get emptyfile '" + checkfile.Name() + "'\n",
			match: `(?ms).*Downloading .* succeeded.*`,
			data:  []byte{},
		},
		{
			path:  writePath,
			cmd:   "put '" + localfile.Name() + "' testfile\n",
			match: `(?ms).*Uploading .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "get testfile '" + checkfile.Name() + "'\n",
			match: `(?ms).*succeeded.*`,
			data:  testdata,
		},
		{
			path:  writePath,
			cmd:   "move testfile newdir0/\n",
			match: `(?ms).*Moving .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "move testfile newdir0/\n",
			match: `(?ms).*Moving .* failed.*`,
		},
		{
			path:  writePath,
			cmd:   "ls\n",
			match: `(?ms).*newdir0.* 0 .*`,
		},
		{
			path:  writePath,
			cmd:   "move newdir0/testfile emptyfile/bogus/\n",
			match: `(?ms).*Moving .* failed.*`,
		},
		{
			path:  writePath,
			cmd:   "mkcol newdir1\n",
			match: `(?ms).*Creating .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "move newdir0/testfile newdir1/\n",
			match: `(?ms).*Moving .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "put '" + localfile.Name() + "' newdir1/testfile1\n",
			match: `(?ms).*Uploading .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "mkcol newdir2\n",
			match: `(?ms).*Creating .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "put '" + localfile.Name() + "' newdir2/testfile2\n",
			match: `(?ms).*Uploading .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "get newdir2/testfile2 '" + checkfile.Name() + "'\n",
			match: `(?ms).*succeeded.*`,
			data:  testdata,
		},
		{
			path:  writePath,
			cmd:   "rmcol newdir2\n",
			match: `(?ms).*Deleting collection .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "get newdir2/testfile2 '" + checkfile.Name() + "'\n",
			match: `(?ms).*Downloading .* failed.*`,
		},
		{
			path:  "/c=" + arvadostest.UserAgreementCollection + "/t=" + arv.AuthToken + "/",
			cmd:   "put '" + localfile.Name() + "' foo\n",
			match: `(?ms).*Uploading .* failed:.*403 Forbidden.*`,
		},
		{
			path:  "/c=" + strings.Replace(arvadostest.FooAndBarFilesInDirPDH, "+", "-", -1) + "/t=" + arv.AuthToken + "/",
			cmd:   "put '" + localfile.Name() + "' foo\n",
			match: `(?ms).*Uploading .* failed:.*405 Method Not Allowed.*`,
		},
	} {
		c.Logf("%s %+v", "http://"+s.testServer.Addr, trial)

		os.Remove(checkfile.Name())

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

		if trial.data == nil {
			continue
		}
		checkfile, err = os.Open(checkfile.Name())
		c.Assert(err, check.IsNil)
		checkfile.Seek(0, os.SEEK_SET)
		got, err := ioutil.ReadAll(checkfile)
		c.Check(got, check.DeepEquals, trial.data)
		c.Check(err, check.IsNil)
	}
}
