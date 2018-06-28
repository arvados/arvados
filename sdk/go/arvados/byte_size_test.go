// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"github.com/ghodss/yaml"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&ByteSizeSuite{})

type ByteSizeSuite struct{}

func (s *ByteSizeSuite) TestUnmarshal(c *check.C) {
	for _, testcase := range []struct {
		in  string
		out int64
	}{
		{"0", 0},
		{"5", 5},
		{"5B", 5},
		{"5 B", 5},
		{" 4 KiB ", 4096},
		{"0K", 0},
		{"0Ki", 0},
		{"0 KiB", 0},
		{"4K", 4000},
		{"4KB", 4000},
		{"4Ki", 4096},
		{"4KiB", 4096},
		{"4MB", 4000000},
		{"4MiB", 4194304},
		{"4GB", 4000000000},
		{"4 GiB", 4294967296},
		{"4TB", 4000000000000},
		{"4TiB", 4398046511104},
		{"4PB", 4000000000000000},
		{"4PiB", 4503599627370496},
		{"4EB", 4000000000000000000},
		{"4EiB", 4611686018427387904},
	} {
		var n ByteSize
		err := yaml.Unmarshal([]byte(testcase.in+"\n"), &n)
		c.Check(err, check.IsNil)
		c.Check(int64(n), check.Equals, testcase.out)
	}
	for _, testcase := range []string{
		"B", "K", "KB", "KiB", "4BK", "4iB", "4A", "b", "4b", "4mB", "4m", "4mib", "4KIB", "4K iB", "4Ki B", "BB", "4BB",
		"400000 EB", // overflows int64
	} {
		var n ByteSize
		err := yaml.Unmarshal([]byte(testcase+"\n"), &n)
		c.Log(n)
		c.Log(err)
		c.Check(err, check.NotNil)
	}
}
