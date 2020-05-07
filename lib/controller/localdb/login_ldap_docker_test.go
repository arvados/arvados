// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Skip this slow test unless invoked as "go test -tags docker".
// +build docker

package localdb

import (
	"os"
	"os/exec"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&LDAPSuite{})

type LDAPSuite struct{}

func (s *LDAPSuite) TearDownSuite(c *check.C) {
	// Undo any changes/additions to the user database so they
	// don't affect subsequent tests.
	arvadostest.ResetEnv()
	c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
}

func (s *LDAPSuite) TestLoginLDAPViaPAM(c *check.C) {
	cmd := exec.Command("bash", "login_ldap_docker_test.sh")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "config_method=pam")
	err := cmd.Run()
	c.Check(err, check.IsNil)
}

func (s *LDAPSuite) TestLoginLDAPBuiltin(c *check.C) {
	cmd := exec.Command("bash", "login_ldap_docker_test.sh")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "config_method=ldap")
	err := cmd.Run()
	c.Check(err, check.IsNil)
}
