// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"os"
	"os/exec"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

func (s *ClientSuite) TestShellGatewayNotAvailable(c *check.C) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".", "shell", arvadostest.QueuedContainerUUID, "-o", "controlpath=none", "echo", "ok")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.Check(cmd.Run(), check.NotNil)
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*gateway is not available, container is queued.*`)
}
