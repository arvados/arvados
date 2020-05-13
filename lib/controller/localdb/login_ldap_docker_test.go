// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"os"
	"os/exec"
	"testing"

	check "gopkg.in/check.v1"
)

func haveDocker() bool {
	_, err := exec.Command("docker", "info").CombinedOutput()
	return err == nil
}

func (s *LDAPSuite) TestLoginLDAPViaPAM(c *check.C) {
	if testing.Short() {
		c.Skip("skipping docker test in short mode")
	}
	if !haveDocker() {
		c.Skip("skipping docker test because docker is not available")
	}
	cmd := exec.Command("bash", "login_ldap_docker_test.sh")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "config_method=pam")
	err := cmd.Run()
	c.Check(err, check.IsNil)
}

func (s *LDAPSuite) TestLoginLDAPBuiltin(c *check.C) {
	if testing.Short() {
		c.Skip("skipping docker test in short mode")
	}
	if !haveDocker() {
		c.Skip("skipping docker test because docker is not available")
	}
	cmd := exec.Command("bash", "login_ldap_docker_test.sh")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "config_method=ldap")
	err := cmd.Run()
	c.Check(err, check.IsNil)
}
