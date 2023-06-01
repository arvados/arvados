// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
	git_client "gopkg.in/src-d/go-git.v4/plumbing/transport/client"
	git_http "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

type GitMountSuite struct {
	tmpdir string
}

var _ = check.Suite(&GitMountSuite{})

func (s *GitMountSuite) SetUpTest(c *check.C) {
	var err error
	s.tmpdir, err = ioutil.TempDir("", "")
	c.Assert(err, check.IsNil)
	git_client.InstallProtocol("https", git_http.NewClient(arvados.InsecureHTTPClient))
}

func (s *GitMountSuite) TearDownTest(c *check.C) {
	err := os.RemoveAll(s.tmpdir)
	c.Check(err, check.IsNil)
}

// Commit fd3531f is crunch-run-tree-test
func (s *GitMountSuite) TestExtractTree(c *check.C) {
	gm := gitMount{
		Path:   "/",
		UUID:   arvadostest.Repository2UUID,
		Commit: "fd3531f42995344f36c30b79f55f27b502f3d344",
	}
	ac := arvados.NewClientFromEnv()
	err := gm.extractTree(ac, s.tmpdir, arvadostest.ActiveToken)
	c.Check(err, check.IsNil)

	fnm := filepath.Join(s.tmpdir, "dir1/dir2/file with mode 0644")
	data, err := ioutil.ReadFile(fnm)
	c.Check(err, check.IsNil)
	c.Check(data, check.DeepEquals, []byte{0, 1, 2, 3})
	fi, err := os.Stat(fnm)
	c.Check(err, check.IsNil)
	if err == nil {
		c.Check(fi.Mode(), check.Equals, os.FileMode(0644))
	}

	fnm = filepath.Join(s.tmpdir, "dir1/dir2/file with mode 0755")
	data, err = ioutil.ReadFile(fnm)
	c.Check(err, check.IsNil)
	c.Check(string(data), check.DeepEquals, "#!/bin/sh\nexec echo OK\n")
	fi, err = os.Stat(fnm)
	c.Check(err, check.IsNil)
	if err == nil {
		c.Check(fi.Mode(), check.Equals, os.FileMode(0755))
	}

	// Ensure there's no extra stuff like a ".git" dir
	s.checkTmpdirContents(c, []string{"dir1"})

	// Ensure tmpdir is world-readable and world-executable so the
	// UID inside the container can use it.
	fi, err = os.Stat(s.tmpdir)
	c.Check(err, check.IsNil)
	c.Check(fi.Mode()&os.ModePerm, check.Equals, os.FileMode(0755))
}

// Commit 5ebfab0 is not the tip of any branch or tag, but is
// reachable in branch "crunch-run-non-tip-test".
func (s *GitMountSuite) TestExtractNonTipCommit(c *check.C) {
	gm := gitMount{
		UUID:   arvadostest.Repository2UUID,
		Commit: "5ebfab0522851df01fec11ec55a6d0f4877b542e",
	}
	err := gm.extractTree(arvados.NewClientFromEnv(), s.tmpdir, arvadostest.ActiveToken)
	c.Check(err, check.IsNil)

	fnm := filepath.Join(s.tmpdir, "file only on testbranch")
	data, err := ioutil.ReadFile(fnm)
	c.Check(err, check.IsNil)
	c.Check(string(data), check.DeepEquals, "testfile\n")
}

func (s *GitMountSuite) TestNonexistentRepository(c *check.C) {
	gm := gitMount{
		Path:   "/",
		UUID:   "zzzzz-s0uqq-nonexistentrepo",
		Commit: "5ebfab0522851df01fec11ec55a6d0f4877b542e",
	}
	err := gm.extractTree(arvados.NewClientFromEnv(), s.tmpdir, arvadostest.ActiveToken)
	c.Check(err, check.NotNil)
	c.Check(err, check.ErrorMatches, ".*repository not found.*")

	s.checkTmpdirContents(c, []string{})
}

func (s *GitMountSuite) TestNonexistentCommit(c *check.C) {
	gm := gitMount{
		Path:   "/",
		UUID:   arvadostest.Repository2UUID,
		Commit: "bb66b6bb6b6bbb6b6b6b66b6b6b6b6b6b6b6b66b",
	}
	err := gm.extractTree(arvados.NewClientFromEnv(), s.tmpdir, arvadostest.ActiveToken)
	c.Check(err, check.NotNil)
	c.Check(err, check.ErrorMatches, ".*object not found.*")

	s.checkTmpdirContents(c, []string{})
}

func (s *GitMountSuite) TestGitUrlDiscoveryFails(c *check.C) {
	delete(discoveryMap, "gitUrl")
	gm := gitMount{
		Path:   "/",
		UUID:   arvadostest.Repository2UUID,
		Commit: "5ebfab0522851df01fec11ec55a6d0f4877b542e",
	}
	err := gm.extractTree(&arvados.Client{}, s.tmpdir, arvadostest.ActiveToken)
	c.Check(err, check.ErrorMatches, ".*error getting discovery doc.*")
}

func (s *GitMountSuite) TestInvalid(c *check.C) {
	for _, trial := range []struct {
		gm      gitMount
		matcher string
	}{
		{
			gm: gitMount{
				Path:   "/",
				UUID:   arvadostest.Repository2UUID,
				Commit: "abc123",
			},
			matcher: ".*SHA1.*",
		},
		{
			gm: gitMount{
				Path:           "/",
				UUID:           arvadostest.Repository2UUID,
				RepositoryName: arvadostest.Repository2Name,
				Commit:         "5ebfab0522851df01fec11ec55a6d0f4877b542e",
			},
			matcher: ".*repository_name.*",
		},
		{
			gm: gitMount{
				Path:   "/",
				GitURL: "https://localhost:0/" + arvadostest.Repository2Name + ".git",
				Commit: "5ebfab0522851df01fec11ec55a6d0f4877b542e",
			},
			matcher: ".*git_url.*",
		},
		{
			gm: gitMount{
				Path:   "/dir1/",
				UUID:   arvadostest.Repository2UUID,
				Commit: "5ebfab0522851df01fec11ec55a6d0f4877b542e",
			},
			matcher: ".*path.*",
		},
		{
			gm: gitMount{
				Path:   "/",
				Commit: "5ebfab0522851df01fec11ec55a6d0f4877b542e",
			},
			matcher: ".*UUID.*",
		},
		{
			gm: gitMount{
				Path:     "/",
				UUID:     arvadostest.Repository2UUID,
				Commit:   "5ebfab0522851df01fec11ec55a6d0f4877b542e",
				Writable: true,
			},
			matcher: ".*writable.*",
		},
	} {
		err := trial.gm.extractTree(arvados.NewClientFromEnv(), s.tmpdir, arvadostest.ActiveToken)
		c.Check(err, check.NotNil)
		s.checkTmpdirContents(c, []string{})

		err = trial.gm.validate()
		c.Check(err, check.ErrorMatches, trial.matcher)
	}
}

func (s *GitMountSuite) checkTmpdirContents(c *check.C, expect []string) {
	f, err := os.Open(s.tmpdir)
	c.Check(err, check.IsNil)
	names, err := f.Readdirnames(-1)
	c.Check(err, check.IsNil)
	c.Check(names, check.DeepEquals, expect)
}
