// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"github.com/ghodss/yaml"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&ConfigSuite{})

type ConfigSuite struct{}

func (s *ConfigSuite) TestInstanceTypesAsArray(c *check.C) {
	var cluster Cluster
	yaml.Unmarshal([]byte("InstanceTypes:\n- Name: foo\n"), &cluster)
	c.Check(len(cluster.InstanceTypes), check.Equals, 1)
	c.Check(cluster.InstanceTypes["foo"].Name, check.Equals, "foo")
}

func (s *ConfigSuite) TestInstanceTypesAsHash(c *check.C) {
	var cluster Cluster
	yaml.Unmarshal([]byte("InstanceTypes:\n  foo:\n    ProviderType: bar\n"), &cluster)
	c.Check(len(cluster.InstanceTypes), check.Equals, 1)
	c.Check(cluster.InstanceTypes["foo"].Name, check.Equals, "foo")
	c.Check(cluster.InstanceTypes["foo"].ProviderType, check.Equals, "bar")
}

func (s *ConfigSuite) TestInstanceTypeSize(c *check.C) {
	var it InstanceType
	err := yaml.Unmarshal([]byte("Name: foo\nScratch: 4GB\nRAM: 4GiB\n"), &it)
	c.Check(err, check.IsNil)
	c.Check(int64(it.Scratch), check.Equals, int64(4000000000))
	c.Check(int64(it.RAM), check.Equals, int64(4294967296))
}
