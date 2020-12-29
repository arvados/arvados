// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gopkg.in/check.v1"
)

var buildimage string

func init() {
	os.Args = append(os.Args, "-test.timeout=30m") // kludge

	// This enables a hack to speed up repeated tests: hit "docker
	// commit --pause {containername} checkpointtag" after the
	// test container has downloaded/compiled some stuff, then run
	// tests with "-test.buildimage=checkpointtag" next time to
	// retry/resume/update from that point.
	flag.StringVar(&buildimage, "test.buildimage", "debian:10", "docker image to use when running buildpackage")
}

type BuildSuite struct{}

var _ = check.Suite(&BuildSuite{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *BuildSuite) TestBuildAndInstall(c *check.C) {
	if testing.Short() {
		c.Skip("skipping docker tests in short mode")
	} else if _, err := exec.Command("docker", "info").CombinedOutput(); err != nil {
		c.Skip("skipping docker tests because docker is not available")
	}
	tmpdir := c.MkDir()
	defer os.RemoveAll(tmpdir)

	err := os.Mkdir(tmpdir+"/pkg", 0755)
	c.Assert(err, check.IsNil)
	err = os.Mkdir(tmpdir+"/bin", 0755)
	c.Assert(err, check.IsNil)

	cmd := exec.Command("go", "install")
	cmd.Env = append(append([]string(nil), os.Environ()...), "GOPATH="+tmpdir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	c.Assert(err, check.IsNil)

	srctree, err := filepath.Abs("../..")
	c.Assert(err, check.IsNil)

	cmd = exec.Command("docker", "run", "--rm",
		"-v", tmpdir+"/pkg:/pkg",
		"-v", tmpdir+"/bin/arvados-package:/arvados-package:ro",
		"-v", srctree+":/usr/local/src/arvados:ro",
		buildimage,
		"/arvados-package", "build",
		"-package-version", "0.9.99",
		"-source", "/usr/local/src/arvados",
		"-output-directory", "/pkg")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	c.Assert(err, check.IsNil)

	fi, err := os.Stat(tmpdir + "/pkg/arvados-server-easy_0.9.99_amd64.deb")
	c.Assert(err, check.IsNil)
	c.Logf("%#v", fi)
}
