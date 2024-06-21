// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package diagnostics

import (
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&suite{})

type suite struct{}

func (*suite) TestGetSHA2FromImageData(c *C) {
	imageSHA2, err := getSHA2FromImageData(HelloWorldDockerImage)
	c.Check(err, IsNil)
	c.Check(imageSHA2, Matches, `[0-9a-f]{64}`)
}
