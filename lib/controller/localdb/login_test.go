// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"encoding/json"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&loginSuite{})

type loginSuite struct{}

func (s *loginSuite) TestValidateLoginRedirectTarget(c *check.C) {
	var cluster arvados.Cluster
	for _, trial := range []struct {
		pass    bool
		wb1     string
		wb2     string
		trusted string
		target  string
	}{
		{true, "https://wb1.example/", "https://wb2.example/", "", "https://wb2.example/"},
		{true, "https://wb1.example:443/", "https://wb2.example:443/", "", "https://wb2.example/"},
		{true, "https://wb1.example:443/", "https://wb2.example:443/", "", "https://wb2.example"},
		{true, "https://wb1.example:443", "https://wb2.example:443", "", "https://wb2.example/"},
		{true, "http://wb1.example:80/", "http://wb2.example:80/", "", "http://wb2.example/"},
		{false, "https://wb1.example:80/", "https://wb2.example:80/", "", "https://wb2.example/"},
		{false, "https://wb1.example:1234/", "https://wb2.example:1234/", "", "https://wb2.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "", "https://bad.wb2.example/"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://good.wb2.example/", "https://good.wb2.example"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://good.wb2.example:443/", "https://good.wb2.example"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://good.wb2.example:443", "https://good.wb2.example/"},

		{true, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://ok.wildcard.example/"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://ok.ok.wildcard.example/"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://[ok.ok.wildcard.example]:443/"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://[*.wildcard.example]:443", "https://ok.ok.wildcard.example/"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example:443", "https://ok.wildcard.example/"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://ok.wildcard.example:443/"},
		{true, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example:443", "https://ok.wildcard.example:443/"},

		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "http://wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "http://.wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "http://wrongscheme.wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "http://wrongscheme.wildcard.example:443/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://wrongport.wildcard.example:80/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://notmatching-wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "http://notmatching.wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example:443", "https://attacker.example/ok.wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://attacker.example/ok.wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://attacker.example/?https://ok.wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*.wildcard.example", "https://attacker.example/#https://ok.wildcard.example/"},
		{false, "https://wb1.example/", "https://wb2.example/", "https://*-wildcard.example", "https://notsupported-wildcard.example/"},
	} {
		c.Logf("trial %+v", trial)
		// We use json.Unmarshal() to load the test strings
		// because we're testing behavior when the config file
		// contains string X.
		err := json.Unmarshal([]byte(`"`+trial.wb1+`"`), &cluster.Services.Workbench1.ExternalURL)
		c.Assert(err, check.IsNil)
		err = json.Unmarshal([]byte(`"`+trial.wb2+`"`), &cluster.Services.Workbench2.ExternalURL)
		c.Assert(err, check.IsNil)
		if trial.trusted != "" {
			err = json.Unmarshal([]byte(`{"`+trial.trusted+`": {}}`), &cluster.Login.TrustedClients)
			c.Assert(err, check.IsNil)
		}
		err = validateLoginRedirectTarget(&cluster, trial.target)
		c.Check(err == nil, check.Equals, trial.pass)
	}
}
