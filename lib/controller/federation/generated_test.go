// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"os/exec"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&UptodateSuite{})

type UptodateSuite struct{}

func (*UptodateSuite) TestUpToDate(c *check.C) {
	output, err := exec.Command("go", "run", "generate.go", "-check").CombinedOutput()
	if err != nil {
		c.Log(string(output))
		c.Error("generated.go is out of date -- run 'go generate' to update it")
	}
}
