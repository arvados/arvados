// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

func (s *IntegrationSuite) TestCadaverHTTPAuth(c *check.C) {
	s.testCadaver(c, arvadostest.ActiveToken, func(newCollection arvados.Collection) (string, string, string) {
		r := "/c=" + arvadostest.FooAndBarFilesInDirUUID + "/"
		w := "/c=" + newCollection.UUID + "/"
		pdh := "/c=" + strings.Replace(arvadostest.FooAndBarFilesInDirPDH, "+", "-", -1) + "/"
		return r, w, pdh
	}, nil)
}

func (s *IntegrationSuite) TestCadaverPathAuth(c *check.C) {
	s.testCadaver(c, "", func(newCollection arvados.Collection) (string, string, string) {
		r := "/c=" + arvadostest.FooAndBarFilesInDirUUID + "/t=" + arvadostest.ActiveToken + "/"
		w := "/c=" + newCollection.UUID + "/t=" + arvadostest.ActiveToken + "/"
		pdh := "/c=" + strings.Replace(arvadostest.FooAndBarFilesInDirPDH, "+", "-", -1) + "/t=" + arvadostest.ActiveToken + "/"
		return r, w, pdh
	}, nil)
}

func (s *IntegrationSuite) TestCadaverUserProject(c *check.C) {
	rpath := "/users/active/foo_file_in_dir/"
	s.testCadaver(c, arvadostest.ActiveToken, func(newCollection arvados.Collection) (string, string, string) {
		wpath := "/users/active/" + newCollection.Name
		pdh := "/c=" + strings.Replace(arvadostest.FooAndBarFilesInDirPDH, "+", "-", -1) + "/"
		return rpath, wpath, pdh
	}, func(path string) bool {
		// Skip tests that rely on writes, because /users/
		// tree is read-only.
		return !strings.HasPrefix(path, rpath) || strings.HasPrefix(path, rpath+"_/")
	})
}

func (s *IntegrationSuite) testCadaver(c *check.C, password string, pathFunc func(arvados.Collection) (string, string, string), skip func(string) bool) {
	testdata := []byte("the human tragedy consists in the necessity of living with the consequences of actions performed under the pressure of compulsions we do not understand")

	tempdir, err := ioutil.TempDir("", "keep-web-test-")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(tempdir)

	localfile, err := ioutil.TempFile(tempdir, "localfile")
	c.Assert(err, check.IsNil)
	localfile.Write(testdata)

	emptyfile, err := ioutil.TempFile(tempdir, "emptyfile")
	c.Assert(err, check.IsNil)

	checkfile, err := ioutil.TempFile(tempdir, "checkfile")
	c.Assert(err, check.IsNil)

	var newCollection arvados.Collection
	arv := arvados.NewClientFromEnv()
	arv.AuthToken = arvadostest.ActiveToken
	err = arv.RequestAndDecode(&newCollection, "POST", "arvados/v1/collections", nil, map[string]interface{}{"collection": map[string]interface{}{}})
	c.Assert(err, check.IsNil)

	readPath, writePath, pdhPath := pathFunc(newCollection)

	matchToday := time.Now().Format("Jan +2")

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
			match: `(?ms).*bar *3.*foo +3 +Feb +\d+ +2014.*`,
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
			cmd:   "move testfile \"test &#!%20 file\"\n",
			match: `(?ms).*Moving .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "move \"test &#!%20 file\" testfile\n",
			match: `(?ms).*Moving .* succeeded.*`,
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
			cmd:   "lock newdir0/testfile\n",
			match: `(?ms).*Locking .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "unlock newdir0/testfile\nasdf\n",
			match: `(?ms).*Unlocking .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "ls\n",
			match: `(?ms).*newdir0.* 0 +` + matchToday + ` \d+:\d+\n.*`,
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
			cmd:   "move newdir1/ newdir1x/\n",
			match: `(?ms).*Moving .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "move newdir1x newdir1\n",
			match: `(?ms).*Moving .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "move newdir0/testfile newdir1/\n",
			match: `(?ms).*Moving .* succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "move newdir1 newdir1/\n",
			match: `(?ms).*Moving .* failed.*`,
		},
		{
			path:  writePath,
			cmd:   "get newdir1/testfile '" + checkfile.Name() + "'\n",
			match: `(?ms).*succeeded.*`,
			data:  testdata,
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
			cmd:   "copy newdir2/testfile2 testfile3\n",
			match: `(?ms).*succeeded.*`,
		},
		{
			path:  writePath,
			cmd:   "get testfile3 '" + checkfile.Name() + "'\n",
			match: `(?ms).*succeeded.*`,
			data:  testdata,
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
			path:  pdhPath,
			cmd:   "put '" + localfile.Name() + "' foo\n",
			match: `(?ms).*Uploading .* failed:.*405 Method Not Allowed.*`,
		},
		{
			path:  pdhPath,
			cmd:   "move foo bar\n",
			match: `(?ms).*Moving .* failed:.*405 Method Not Allowed.*`,
		},
		{
			path:  pdhPath,
			cmd:   "copy foo bar\n",
			match: `(?ms).*Copying .* failed:.*405 Method Not Allowed.*`,
		},
		{
			path:  pdhPath,
			cmd:   "delete foo\n",
			match: `(?ms).*Deleting .* failed:.*405 Method Not Allowed.*`,
		},
		{
			path:  pdhPath,
			cmd:   "lock foo\n",
			match: `(?ms).*Locking .* failed:.*405 Method Not Allowed.*`,
		},
	} {
		c.Logf("%s %+v", s.testServer.URL, trial)
		if skip != nil && skip(trial.path) {
			c.Log("(skip)")
			continue
		}

		os.Remove(checkfile.Name())

		stdout := s.runCadaver(c, password, trial.path, trial.cmd)
		c.Check(stdout, check.Matches, trial.match)

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

func (s *IntegrationSuite) TestCadaverByID(c *check.C) {
	for _, path := range []string{"/by_id", "/by_id/"} {
		stdout := s.runCadaver(c, arvadostest.ActiveToken, path, "ls")
		c.Check(stdout, check.Matches, `(?ms).*collection is empty.*`)
	}
	for _, path := range []string{
		"/by_id/" + arvadostest.FooCollectionPDH,
		"/by_id/" + arvadostest.FooCollectionPDH + "/",
		"/by_id/" + arvadostest.FooCollection,
		"/by_id/" + arvadostest.FooCollection + "/",
	} {
		stdout := s.runCadaver(c, arvadostest.ActiveToken, path, "ls")
		c.Check(stdout, check.Matches, `(?ms).*\s+foo\s+3 .*`)
	}
}

func (s *IntegrationSuite) TestCadaverUsersDir(c *check.C) {
	for _, path := range []string{"/"} {
		stdout := s.runCadaver(c, arvadostest.ActiveToken, path, "ls")
		c.Check(stdout, check.Matches, `(?ms).*Coll:\s+by_id\s+0 .*`)
		c.Check(stdout, check.Matches, `(?ms).*Coll:\s+users\s+0 .*`)
	}
	for _, path := range []string{"/users", "/users/"} {
		stdout := s.runCadaver(c, arvadostest.ActiveToken, path, "ls")
		c.Check(stdout, check.Matches, `(?ms).*Coll:\s+active.*`)
	}
	for _, path := range []string{"/users/active", "/users/active/"} {
		stdout := s.runCadaver(c, arvadostest.ActiveToken, path, "ls")
		c.Check(stdout, check.Matches, `(?ms).*Coll:\s+A Project\s+0 .*`)
		c.Check(stdout, check.Matches, `(?ms).*Coll:\s+bar_file\s+0 .*`)
	}
	for _, path := range []string{"/users/admin", "/users/doesnotexist", "/users/doesnotexist/"} {
		stdout := s.runCadaver(c, arvadostest.ActiveToken, path, "ls")
		c.Check(stdout, check.Matches, `(?ms).*404 Not Found.*`)
	}
}

func (s *IntegrationSuite) runCadaver(c *check.C, password, path, stdin string) string {
	tempdir, err := ioutil.TempDir("", "keep-web-test-")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(tempdir)

	cmd := exec.Command("cadaver", s.testServer.URL+path)
	if password != "" {
		// cadaver won't try username/password authentication
		// unless the server responds 401 to an
		// unauthenticated request, which it only does in
		// AttachmentOnlyHost, TrustAllContent, and
		// per-collection vhost cases.
		s.handler.Cluster.Services.WebDAVDownload.ExternalURL.Host = s.testServer.URL[7:]

		cmd.Env = append(os.Environ(), "HOME="+tempdir)
		f, err := os.OpenFile(filepath.Join(tempdir, ".netrc"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		c.Assert(err, check.IsNil)
		_, err = fmt.Fprintf(f, "default login none password %s\n", password)
		c.Assert(err, check.IsNil)
		c.Assert(f.Close(), check.IsNil)
	}
	cmd.Stdin = bytes.NewBufferString(stdin)
	stdout, err := cmd.StdoutPipe()
	c.Assert(err, check.Equals, nil)
	cmd.Stderr = cmd.Stdout
	go cmd.Start()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, stdout)
	c.Check(err, check.Equals, nil)
	err = cmd.Wait()
	c.Check(err, check.Equals, nil)
	return buf.String()
}
