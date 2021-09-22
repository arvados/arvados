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
		ReportFunc: func(pattern, detail string) {
			reported = append(reported, pattern, detail)
		},
	}
	ls.Write([]byte("foo\nbar\n2021-01-01T00:00:00.000Z: bar"))
	ls.Write([]byte("baz: it's a detail\nwaz\nqux"))
	ls.Write([]byte("\nfoobar\n"))
	c.Check(reported, check.DeepEquals, []string{"barbaz", "2021-01-01T00:00:00.000Z: barbaz: it's a detail"})
}

func (s *logScannerSuite) TestOneWritePerLine(c *check.C) {
	var reported []string
	ls := logScanner{
		Patterns: []string{"barbaz"},
		ReportFunc: func(pattern, detail string) {
			reported = append(reported, pattern, detail)
		},
	}
	ls.Write([]byte("foo\n"))
	ls.Write([]byte("2021-01-01T00:00:00.000Z: barbaz: it's a detail\n"))
	ls.Write([]byte("waz\n"))
	c.Check(reported, check.DeepEquals, []string{"barbaz", "2021-01-01T00:00:00.000Z: barbaz: it's a detail"})
}

func (s *logScannerSuite) TestNoDetail(c *check.C) {
	var reported []string
	ls := logScanner{
		Patterns: []string{"barbaz"},
		ReportFunc: func(pattern, detail string) {
			reported = append(reported, pattern, detail)
		},
	}
	ls.Write([]byte("foo\n"))
	ls.Write([]byte("barbaz\n"))
	ls.Write([]byte("waz\n"))
	c.Check(reported, check.DeepEquals, []string{"barbaz", "barbaz"})
}
