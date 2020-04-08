// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Skip this slow test unless invoked as "go test -tags docker".
// +build docker

package localdb

import (
	"os"
	"os/exec"

	check "gopkg.in/check.v1"
)

func (s *PamSuite) TestLoginLDAPViaPAM(c *check.C) {
	cmd := exec.Command("bash", "login_pam_docker_test.sh")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	c.Check(err, check.IsNil)
}
