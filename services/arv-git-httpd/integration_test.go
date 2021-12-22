// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

// IntegrationSuite tests need an API server and an arv-git-httpd
// server. See GitSuite and GitoliteSuite.
type IntegrationSuite struct {
	tmpRepoRoot string
	tmpWorkdir  string
	testServer  *server
	cluster     *arvados.Cluster
}

func (s *IntegrationSuite) SetUpTest(c *check.C) {
	arvadostest.ResetEnv()

	var err error
	if s.tmpRepoRoot == "" {
		s.tmpRepoRoot, err = ioutil.TempDir("", "arv-git-httpd")
		c.Assert(err, check.Equals, nil)
	}
	s.tmpWorkdir, err = ioutil.TempDir("", "arv-git-httpd")
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("git", "init", "--bare", s.tmpRepoRoot+"/zzzzz-s0uqq-382brsig8rp3666.git").Output()
	c.Assert(err, check.Equals, nil)
	// we need git 2.28 to specify the initial branch with -b; Buster only has 2.20; so we do it in 2 steps
	_, err = exec.Command("git", "init", s.tmpWorkdir).Output()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("sh", "-c", "cd "+s.tmpWorkdir+" && git checkout -b main").Output()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("sh", "-c", "cd "+s.tmpWorkdir+" && echo initial >initial && git add initial && git -c user.name=Initial -c user.email=Initial commit -am 'foo: initial commit'").CombinedOutput()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("sh", "-c", "cd "+s.tmpWorkdir+" && git push "+s.tmpRepoRoot+"/zzzzz-s0uqq-382brsig8rp3666.git main:main").CombinedOutput()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("sh", "-c", "cd "+s.tmpWorkdir+" && echo work >work && git add work && git -c user.name=Foo -c user.email=Foo commit -am 'workdir: test'").CombinedOutput()
	c.Assert(err, check.Equals, nil)

	if s.cluster == nil {
		cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
		c.Assert(err, check.Equals, nil)
		s.cluster, err = cfg.GetCluster("")
		c.Assert(err, check.Equals, nil)

		s.cluster.Services.GitHTTP.InternalURLs = map[arvados.URL]arvados.ServiceInstance{{Host: "localhost:0"}: {}}
		s.cluster.TLS.Insecure = true
		s.cluster.Git.GitCommand = "/usr/bin/git"
		s.cluster.Git.Repositories = s.tmpRepoRoot
		s.cluster.ManagementToken = arvadostest.ManagementToken
	}

	s.testServer = &server{cluster: s.cluster}
	err = s.testServer.Start()
	c.Assert(err, check.Equals, nil)

	_, err = exec.Command("git", "config",
		"--file", s.tmpWorkdir+"/.git/config",
		"credential.http://"+s.testServer.Addr+"/.helper",
		"!cred(){ cat >/dev/null; if [ \"$1\" = get ]; then echo password=$ARVADOS_API_TOKEN; fi; };cred").Output()
	c.Assert(err, check.Equals, nil)
	_, err = exec.Command("git", "config",
		"--file", s.tmpWorkdir+"/.git/config",
		"credential.http://"+s.testServer.Addr+"/.username",
		"none").Output()
	c.Assert(err, check.Equals, nil)

	// Clear ARVADOS_API_* env vars before starting up the server,
	// to make sure arv-git-httpd doesn't use them or complain
	// about them being missing.
	os.Unsetenv("ARVADOS_API_HOST")
	os.Unsetenv("ARVADOS_API_HOST_INSECURE")
	os.Unsetenv("ARVADOS_API_TOKEN")
}

func (s *IntegrationSuite) TearDownTest(c *check.C) {
	var err error
	if s.testServer != nil {
		err = s.testServer.Close()
	}
	c.Check(err, check.Equals, nil)
	s.testServer = nil

	if s.tmpRepoRoot != "" {
		err = os.RemoveAll(s.tmpRepoRoot)
		c.Check(err, check.Equals, nil)
	}
	s.tmpRepoRoot = ""

	if s.tmpWorkdir != "" {
		err = os.RemoveAll(s.tmpWorkdir)
		c.Check(err, check.Equals, nil)
	}
	s.tmpWorkdir = ""

	s.cluster = nil
}

func (s *IntegrationSuite) RunGit(c *check.C, token, gitCmd, repo string, args ...string) error {
	cwd, err := os.Getwd()
	c.Assert(err, check.Equals, nil)
	defer os.Chdir(cwd)
	os.Chdir(s.tmpWorkdir)

	gitargs := append([]string{
		gitCmd, "http://" + s.testServer.Addr + "/" + repo,
	}, args...)
	cmd := exec.Command("git", gitargs...)
	cmd.Env = append(os.Environ(), "ARVADOS_API_TOKEN="+token)
	w, err := cmd.StdinPipe()
	c.Assert(err, check.Equals, nil)
	w.Close()
	output, err := cmd.CombinedOutput()
	c.Log("git ", gitargs, " => ", err)
	c.Log(string(output))
	if err != nil && len(output) > 0 {
		// If messages appeared on stderr, they are more
		// helpful than the err returned by CombinedOutput().
		//
		// Easier to match error strings without newlines:
		err = errors.New(strings.Replace(string(output), "\n", " // ", -1))
	}
	return err
}
