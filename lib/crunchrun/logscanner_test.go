// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&logScannerSuite{})

type logScannerSuite struct {
}

func (s *logScannerSuite) TestCallReportFuncOnce(c *check.C) {
	var reported []string
	ls := logScanner{
		Patterns: []string{"foobar", "barbaz"},
		ReportFunc: func(pattern string) {
			reported = append(reported, pattern)
		},
	}
	ls.Write([]byte("foo\nbar\nbar"))
	ls.Write([]byte("baz\nwaz\nqux"))
	ls.Write([]byte("\nfoobar\n"))
	c.Check(reported, check.DeepEquals, []string{"barbaz"})
}
