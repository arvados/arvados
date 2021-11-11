// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package install

import (
	"bytes"
	"os/exec"
	"testing"

	"gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&Suite{})

type Suite struct{}

/*
	TestExtractGoVersion tests the grep/awk command used in
	tools/arvbox/bin/arvbox to extract the version of Go to install for
	bootstrapping `arvados-server`.

	If this test is changed, the arvbox code will also need to be updated.
*/
func (*Suite) TestExtractGoVersion(c *check.C) {
	script := `
  sourcepath="$(realpath ../..)"
  (cd ${sourcepath} && grep 'const goversion =' lib/install/deps.go |awk -F'"' '{print $2}')
	`
	cmd := exec.Command("bash", "-")
	cmd.Stdin = bytes.NewBufferString("set -ex -o pipefail\n" + script)
	cmdOutput, err := cmd.Output()
	c.Assert(err, check.IsNil)
	c.Assert(string(cmdOutput), check.Equals, goversion+"\n")
}
