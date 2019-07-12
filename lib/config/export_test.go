// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"regexp"
	"strings"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&ExportSuite{})

type ExportSuite struct{}

func (s *ExportSuite) TestExport(c *check.C) {
	confdata := strings.Replace(string(DefaultYAML), "SAMPLE", "testkey", -1)
	cfg, err := testLoader(c, confdata, nil).Load()
	c.Assert(err, check.IsNil)
	cluster := cfg.Clusters["xxxxx"]
	cluster.ManagementToken = "abcdefg"

	var exported bytes.Buffer
	err = ExportJSON(&exported, &cluster)
	c.Check(err, check.IsNil)
	if err != nil {
		c.Logf("If all the new keys are safe, add these to whitelist in export.go:")
		for _, k := range regexp.MustCompile(`"[^"]*"`).FindAllString(err.Error(), -1) {
			c.Logf("\t%q: true,", strings.Replace(k, `"`, "", -1))
		}
	}
	c.Check(exported.String(), check.Not(check.Matches), `(?ms).*abcdefg.*`)
}
