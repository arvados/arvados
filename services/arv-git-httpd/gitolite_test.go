package main

import (
	"os"
	"os/exec"
	"io/ioutil"
	"log"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&GitoliteSuite{})

// GitoliteSuite tests need an API server, an arv-git-httpd server,
// and a repository hosted by gitolite.
type GitoliteSuite struct {
	IntegrationSuite
	gitoliteHome string
}

func (s *GitoliteSuite) SetUpTest(c *check.C) {
	var err error
	s.gitoliteHome, err = ioutil.TempDir("", "arv-git-httpd")
	c.Assert(err, check.Equals, nil)

	runGitolite := func(prog string, args ...string) {
		c.Log(prog, " ", args)
		cmd := exec.Command(prog, args...)
		cmd.Dir = s.gitoliteHome
		cmd.Env = append(os.Environ(), "HOME=" + s.gitoliteHome)
		diags, err := cmd.CombinedOutput()
		c.Log(string(diags))
		c.Assert(err, check.Equals, nil)
	}

	runGitolite("gitolite", "setup", "--admin", "root")

	s.tmpRepoRoot = s.gitoliteHome + "/repositories"
	s.Config = &config{
		Addr:       ":0",
		GitCommand: "/usr/share/gitolite3/gitolite-shell",
		Root:       s.tmpRepoRoot,
	}
	s.IntegrationSuite.SetUpTest(c)

	// Install the gitolite hooks in the bare repo we made in
	// (*IntegrationTest)SetUpTest() -- see 2.2.4 at
	// http://gitolite.com/gitolite/gitolite.html
	runGitolite("gitolite", "setup")

	os.Setenv("GITOLITE_HTTP_HOME", s.gitoliteHome)
	os.Setenv("GL_BYPASS_ACCESS_CHECKS", "1")
}

func (s *GitoliteSuite) TearDownTest(c *check.C) {
	// We really want Unsetenv here, but it's not worth forcing an
	// upgrade to Go 1.4.
	os.Setenv("GITOLITE_HTTP_HOME", "")
	os.Setenv("GL_BYPASS_ACCESS_CHECKS", "")
	s.IntegrationSuite.TearDownTest(c)
}

func (s *GitoliteSuite) TestFetch(c *check.C) {
	log.Printf("gitolite_test: TestFetch()")
	err := s.RunGit(c, activeToken, "fetch", "active/foo.git")
	c.Check(err, check.Equals, nil)
}

func (s *GitoliteSuite) TestFetchUnreadable(c *check.C) {
	log.Printf("gitolite_test: TestFetchUnreadable()")
	err := s.RunGit(c, anonymousToken, "fetch", "active/foo.git")
	c.Check(err, check.ErrorMatches, `.* not found.*`)
}

func (s *GitoliteSuite) TestPush(c *check.C) {
	log.Printf("gitolite_test: TestPush()")
	err := s.RunGit(c, activeToken, "push", "active/foo.git")
	c.Check(err, check.Equals, nil)

	// Check that the commit hash appears in the gitolite log, as
	// assurance that the gitolite hooks really did run.

	sha1, err := exec.Command("git", "--git-dir", s.tmpWorkdir + "/.git",
		"log", "-n1", "--format=%H").CombinedOutput()
	c.Logf("git-log in workdir: %q", string(sha1))
	c.Assert(err, check.Equals, nil)
	c.Assert(len(sha1), check.Equals, 41)

	gitoliteLog, err := exec.Command("grep", "-r", string(sha1[:40]), s.gitoliteHome + "/.gitolite/logs").CombinedOutput()
	c.Check(err, check.Equals, nil)
	c.Logf("gitolite log message: %q", string(gitoliteLog))
}

func (s *GitoliteSuite) TestPushUnwritable(c *check.C) {
	log.Printf("gitolite_test: TestPushUnwritable()")
	err := s.RunGit(c, spectatorToken, "push", "active/foo.git")
	c.Check(err, check.ErrorMatches, `.*HTTP code = 403.*`)
}
