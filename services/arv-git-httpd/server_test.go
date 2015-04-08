package main

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&IntegrationSuite{})

// IntegrationSuite tests need an API server and an arv-git-httpd server
type IntegrationSuite struct {
	tmpRepoRoot string
	tmpWorkdir  string
	testServer  *server
}

func (s *IntegrationSuite) TestPathVariants(c *check.C) {
	s.makeArvadosRepo(c)
	// Spectator token
	os.Setenv("ARVADOS_API_TOKEN", "zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu")
	for _, repo := range []string{"active/foo.git", "active/foo/.git", "arvados.git", "arvados/.git"} {
		err := s.runGit(c, "fetch", repo)
		c.Assert(err, check.Equals, nil)
	}
}

func (s *IntegrationSuite) TestReadonly(c *check.C) {
	// Spectator token
	os.Setenv("ARVADOS_API_TOKEN", "zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu")
	err := s.runGit(c, "fetch", "active/foo.git")
	c.Assert(err, check.Equals, nil)
	err = s.runGit(c, "push", "active/foo.git", "master:newbranchfail")
	c.Assert(err, check.ErrorMatches, `.*HTTP code = 403.*`)
	_, err = os.Stat(s.tmpRepoRoot + "/zzzzz-s0uqq-382brsig8rp3666/.git/refs/heads/newbranchfail")
	c.Assert(err, check.FitsTypeOf, &os.PathError{})
}

func (s *IntegrationSuite) TestReadwrite(c *check.C) {
	// Active user token
	os.Setenv("ARVADOS_API_TOKEN", "3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi")
	err := s.runGit(c, "fetch", "active/foo.git")
	c.Assert(err, check.Equals, nil)
	err = s.runGit(c, "push", "active/foo.git", "master:newbranch")
	c.Assert(err, check.Equals, nil)
	_, err = os.Stat(s.tmpRepoRoot + "/zzzzz-s0uqq-382brsig8rp3666/.git/refs/heads/newbranch")
	c.Assert(err, check.Equals, nil)
}

func (s *IntegrationSuite) TestNonexistent(c *check.C) {
	// Spectator token
	os.Setenv("ARVADOS_API_TOKEN", "zw2f4gwx8hw8cjre7yp6v1zylhrhn3m5gvjq73rtpwhmknrybu")
	err := s.runGit(c, "fetch", "thisrepodoesnotexist.git")
	c.Assert(err, check.ErrorMatches, `.* not found.*`)
}

func (s *IntegrationSuite) TestNoPermission(c *check.C) {
	// Anonymous token
	os.Setenv("ARVADOS_API_TOKEN", "4kg6k6lzmp9kj4cpkcoxie964cmvjahbt4fod9zru44k4jqdmi")
	for _, repo := range []string{"active/foo.git", "active/foo/.git"} {
		err := s.runGit(c, "fetch", repo)
		c.Assert(err, check.ErrorMatches, `.* not found.*`)
	}
}

func (s *IntegrationSuite) SetUpSuite(c *check.C) {
	arvadostest.StartAPI()
}

func (s *IntegrationSuite) SetUpTest(c *check.C) {
	arvadostest.ResetEnv()
	s.testServer = &server{}
	var err error
	s.tmpRepoRoot, err = ioutil.TempDir("", "arv-git-httpd")
	c.Assert(err, check.Equals, nil)
	s.tmpWorkdir, err = ioutil.TempDir("", "arv-git-httpd")
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("git", "init", s.tmpRepoRoot+"/zzzzz-s0uqq-382brsig8rp3666").Output()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("sh", "-c", "cd "+s.tmpRepoRoot+"/zzzzz-s0uqq-382brsig8rp3666 && echo test >test && git add test && git -c user.name=Foo -c user.email=Foo commit -am 'foo: test'").CombinedOutput()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("git", "init", s.tmpWorkdir).Output()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("sh", "-c", "cd "+s.tmpWorkdir+" && echo work >work && git add work && git -c user.name=Foo -c user.email=Foo commit -am 'workdir: test'").CombinedOutput()
	c.Assert(err, check.Equals, nil)

	theConfig = &config{
		Addr:       ":",
		GitCommand: "/usr/bin/git",
		Root:       s.tmpRepoRoot,
	}
	err = s.testServer.Start()
	c.Assert(err, check.Equals, nil)

	// Clear ARVADOS_API_TOKEN after starting up the server, to
	// make sure arv-git-httpd doesn't use it.
	os.Setenv("ARVADOS_API_TOKEN", "")

	_, err = exec.Command("git", "config",
		"--file", s.tmpWorkdir+"/.git/config",
		"credential.http://"+s.testServer.Addr+"/.helper",
		"!cred(){ echo password=$ARVADOS_API_TOKEN; };cred").Output()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("git", "config",
		"--file", s.tmpWorkdir+"/.git/config",
		"credential.http://"+s.testServer.Addr+"/.username",
		"none").Output()
	c.Assert(err, check.Equals, nil)
}

func (s *IntegrationSuite) TearDownTest(c *check.C) {
	var err error
	if s.testServer != nil {
		err = s.testServer.Close()
	}
	c.Check(err, check.Equals, nil)
	if s.tmpRepoRoot != "" {
		err = os.RemoveAll(s.tmpRepoRoot)
		c.Check(err, check.Equals, nil)
	}
	if s.tmpWorkdir != "" {
		err = os.RemoveAll(s.tmpWorkdir)
		c.Check(err, check.Equals, nil)
	}
}

func (s *IntegrationSuite) runGit(c *check.C, gitCmd, repo string, args ...string) error {
	cwd, err := os.Getwd()
	c.Assert(err, check.Equals, nil)
	defer os.Chdir(cwd)
	os.Chdir(s.tmpWorkdir)

	gitargs := append([]string{
		gitCmd, "http://" + s.testServer.Addr + "/" + repo,
	}, args...)
	cmd := exec.Command("git", gitargs...)
	w, err := cmd.StdinPipe()
	c.Assert(err, check.Equals, nil)
	go w.Close()
	output, err := cmd.CombinedOutput()
	c.Log("git ", gitargs, " => ", err)
	if err != nil && len(output) > 0 {
		// If messages appeared on stderr, they are more
		// helpful than the err returned by CombinedOutput().
		//
		// Easier to match error strings without newlines:
		err = errors.New(strings.Replace(string(output), "\n", " // ", -1))
	}
	return err
}

// Make a bare arvados repo at {tmpRepoRoot}/arvados.git
func (s *IntegrationSuite) makeArvadosRepo(c *check.C) {
	_, err := exec.Command("git", "init", "--bare", s.tmpRepoRoot+"/zzzzz-s0uqq-arvadosrepo0123.git").Output()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("git", "--git-dir", s.tmpRepoRoot+"/zzzzz-s0uqq-arvadosrepo0123.git", "fetch", "../../.git", "master:master").Output()
	c.Assert(err, check.Equals, nil)
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
