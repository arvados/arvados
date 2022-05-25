// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"os"
	"os/exec"
	"testing"

	"gopkg.in/check.v1"
)

var buildimage string

func init() {
	os.Args = append(os.Args, "-test.timeout=30m") // kludge
}

type BuildSuite struct{}

var _ = check.Suite(&BuildSuite{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *BuildSuite) TestBuildAndInstall(c *check.C) {
	if testing.Short() {
		c.Skip("skipping docker tests in short mode")
	} else if _, err := exec.Command("docker", "info").CombinedOutput(); err != nil {
		c.Skip("skipping docker tests because docker is not available")
	} else if os.Getenv("ENABLE_DOCKER_TESTS") == "" {
		c.Skip("docker tests temporarily disabled if ENABLE_DOCKER_TESTS is not set, see https://dev.arvados.org/issues/15370#note-31")
	}

	tmpdir := c.MkDir()
	defer os.RemoveAll(tmpdir)
	err := os.Chmod(tmpdir, 0755)
	c.Assert(err, check.IsNil)

	cmd := exec.Command("go", "run", ".",
		"build",
		"-package-dir", tmpdir,
		"-package-version", "1.2.3~rc4",
		"-source", "../..",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	c.Check(err, check.IsNil)

	fi, err := os.Stat(tmpdir + "/arvados-server-easy_1.2.3~rc4_amd64.deb")
	c.Assert(err, check.IsNil)
	c.Logf("%#v", fi)

	buf, _ := exec.Command("ls", "-l", tmpdir).CombinedOutput()
	c.Logf("%s", buf)

	cmd = exec.Command("go", "run", ".",
		"testinstall",
		"-package-dir", tmpdir,
		"-package-version", "1.2.3~rc4",
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	c.Check(err, check.IsNil)

	err = os.RemoveAll(tmpdir)
	c.Check(err, check.IsNil)
}
