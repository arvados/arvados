// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package githttpd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&GitoliteSuite{})

// GitoliteSuite tests need an API server, an arvados-git-httpd
// server, and a repository hosted by gitolite.
type GitoliteSuite struct {
	IntegrationSuite
	gitoliteHome string
}

func (s *GitoliteSuite) SetUpTest(c *check.C) {
	var err error
	s.gitoliteHome, err = ioutil.TempDir("", "githttp")
	c.Assert(err, check.Equals, nil)

	runGitolite := func(prog string, args ...string) {
		c.Log(prog, " ", args)
		cmd := exec.Command(prog, args...)
		cmd.Dir = s.gitoliteHome
		cmd.Env = []string{"HOME=" + s.gitoliteHome}
		for _, e := range os.Environ() {
			if !strings.HasPrefix(e, "HOME=") {
				cmd.Env = append(cmd.Env, e)
			}
		}
		diags, err := cmd.CombinedOutput()
		c.Log(string(diags))
		c.Assert(err, check.Equals, nil)
	}

	runGitolite("gitolite", "setup", "--admin", "root")

	s.tmpRepoRoot = s.gitoliteHome + "/repositories"

	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.Equals, nil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.Equals, nil)

	s.cluster.Services.GitHTTP.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Host: "localhost:0"}: {}}
	s.cluster.TLS.Insecure = true
	s.cluster.Git.GitCommand = "/usr/share/gitolite3/gitolite-shell"
	s.cluster.Git.GitoliteHome = s.gitoliteHome
	s.cluster.Git.Repositories = s.tmpRepoRoot

	s.IntegrationSuite.SetUpTest(c)

	// Install the gitolite hooks in the bare repo we made in
	// (*IntegrationTest)SetUpTest() -- see 2.2.4 at
	// http://gitolite.com/gitolite/gitolite.html
	runGitolite("gitolite", "setup")
}

func (s *GitoliteSuite) TearDownTest(c *check.C) {
	// We really want Unsetenv here, but it's not worth forcing an
	// upgrade to Go 1.4.
	os.Setenv("GITOLITE_HTTP_HOME", "")
	os.Setenv("GL_BYPASS_ACCESS_CHECKS", "")
	if s.gitoliteHome != "" {
		err := os.RemoveAll(s.gitoliteHome)
		c.Check(err, check.Equals, nil)
	}
	s.IntegrationSuite.TearDownTest(c)
}

func (s *GitoliteSuite) TestFetch(c *check.C) {
	err := s.RunGit(c, activeToken, "fetch", "active/foo.git", "refs/heads/main")
	c.Check(err, check.Equals, nil)
}

func (s *GitoliteSuite) TestFetchUnreadable(c *check.C) {
	err := s.RunGit(c, anonymousToken, "fetch", "active/foo.git")
	c.Check(err, check.ErrorMatches, `.* not found.*`)
}

func (s *GitoliteSuite) TestPush(c *check.C) {
	err := s.RunGit(c, activeToken, "push", "active/foo.git", "main:gitolite-push")
	c.Check(err, check.Equals, nil)

	// Check that the commit hash appears in the gitolite log, as
	// assurance that the gitolite hooks really did run.

	sha1, err := exec.Command("git", "--git-dir", s.tmpWorkdir+"/.git",
		"log", "-n1", "--format=%H").CombinedOutput()
	c.Logf("git-log in workdir: %q", string(sha1))
	c.Assert(err, check.Equals, nil)
	c.Assert(len(sha1), check.Equals, 41)

	gitoliteLog, err := exec.Command("grep", "-r", string(sha1[:40]), s.gitoliteHome+"/.gitolite/logs").CombinedOutput()
	c.Check(err, check.Equals, nil)
	c.Logf("gitolite log message: %q", string(gitoliteLog))
}

func (s *GitoliteSuite) TestPushUnwritable(c *check.C) {
	err := s.RunGit(c, spectatorToken, "push", "active/foo.git", "main:gitolite-push-fail")
	c.Check(err, check.ErrorMatches, `.*HTTP (code = )?403.*`)
}
