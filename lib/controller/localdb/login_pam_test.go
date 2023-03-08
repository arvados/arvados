// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&PamSuite{})

type PamSuite struct {
	localdbSuite
}

func (s *PamSuite) SetUpTest(c *check.C) {
	s.localdbSuite.SetUpTest(c)
	s.cluster.Login.PAM.Enable = true
	s.cluster.Login.PAM.DefaultEmailDomain = "example.com"
	s.localdb.loginController = &pamLoginController{
		Cluster: s.cluster,
		Parent:  s.localdb,
	}
}

func (s *PamSuite) TestLoginFailure(c *check.C) {
	resp, err := s.localdb.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
		Username: "bogususername",
		Password: "boguspassword",
	})
	c.Check(err, check.ErrorMatches, `PAM: Authentication failure \(with username "bogususername" and password\)`)
	hs, ok := err.(interface{ HTTPStatus() int })
	if c.Check(ok, check.Equals, true) {
		c.Check(hs.HTTPStatus(), check.Equals, http.StatusUnauthorized)
	}
	c.Check(resp.APIToken, check.Equals, "")
}

// This test only runs if the ARVADOS_TEST_PAM_CREDENTIALS_FILE env
// var is set. The credentials file should contain a valid username
// and password, separated by \n.
//
// Depending on the host config, this test succeeds only if the test
// credentials are for the same account being used to run tests.
func (s *PamSuite) TestLoginSuccess(c *check.C) {
	testCredsFile := os.Getenv("ARVADOS_TEST_PAM_CREDENTIALS_FILE")
	if testCredsFile == "" {
		c.Skip("no test credentials file given in ARVADOS_TEST_PAM_CREDENTIALS_FILE")
		return
	}
	buf, err := ioutil.ReadFile(testCredsFile)
	c.Assert(err, check.IsNil)
	lines := strings.Split(string(buf), "\n")
	c.Assert(len(lines), check.Equals, 2, check.Commentf("credentials file %s should contain \"username\\npassword\"", testCredsFile))
	u, p := lines[0], lines[1]

	resp, err := s.localdb.UserAuthenticate(s.ctx, arvados.UserAuthenticateOptions{
		Username: u,
		Password: p,
	})
	c.Check(err, check.IsNil)
	c.Check(resp.APIToken, check.Not(check.Equals), "")
	c.Check(resp.UUID, check.Matches, `zzzzz-gj3su-.*`)
	c.Check(resp.Scopes, check.DeepEquals, []string{"all"})

	authinfo := getCallbackAuthInfo(c, s.railsSpy)
	c.Check(authinfo.Email, check.Equals, u+"@"+s.cluster.Login.PAM.DefaultEmailDomain)
	c.Check(authinfo.AlternateEmails, check.DeepEquals, []string(nil))
}
