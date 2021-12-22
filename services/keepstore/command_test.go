// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"bytes"
	"io/ioutil"
	"os"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CommandSuite{})

type CommandSuite struct{}

func (*CommandSuite) TestLegacyConfigPath(c *check.C) {
	var stdin, stdout, stderr bytes.Buffer
	tmp, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(tmp.Name())
	tmp.Write([]byte("Listen: \"1.2.3.4.5:invalidport\"\n"))
	tmp.Close()
	exited := runCommand("keepstore", []string{"-config", tmp.Name()}, &stdin, &stdout, &stderr)
	c.Check(exited, check.Equals, 1)
	c.Check(stderr.String(), check.Matches, `(?ms).*unable to migrate Listen value.*`)
}
