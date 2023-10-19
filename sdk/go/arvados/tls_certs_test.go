// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"os/exec"

	check "gopkg.in/check.v1"
)

type tlsCertsSuite struct{}

var _ = check.Suite(&tlsCertsSuite{})

func (s *tlsCertsSuite) TestCustomCert(c *check.C) {
	certfile := "/etc/arvados/ca-certificates.crt"
	if _, err := os.Stat(certfile); err != nil {
		c.Skip("custom cert file " + certfile + " does not exist")
	}
	out, err := exec.Command("bash", "-c", "SSL_CERT_FILE= go run tls_certs_test_showenv.go").CombinedOutput()
	c.Logf("%s", out)
	c.Assert(err, check.IsNil)
	c.Check(string(out), check.Equals, certfile+"\n")

	out, err = exec.Command("bash", "-c", "SSL_CERT_FILE=/dev/null go run tls_certs_test_showenv.go").CombinedOutput()
	c.Logf("%s", out)
	c.Assert(err, check.IsNil)
	c.Check(string(out), check.Equals, "/dev/null\n")
}
