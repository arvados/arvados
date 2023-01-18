// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"crypto/tls"
	"encoding/json"

	"github.com/ghodss/yaml"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&ConfigSuite{})

type ConfigSuite struct{}

func (s *ConfigSuite) TestStringSetAsArray(c *check.C) {
	var cluster Cluster
	yaml.Unmarshal([]byte(`
API:
  DisabledAPIs: [jobs.list]`), &cluster)
	c.Check(len(cluster.API.DisabledAPIs), check.Equals, 1)
	_, ok := cluster.API.DisabledAPIs["jobs.list"]
	c.Check(ok, check.Equals, true)
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
	err := yaml.Unmarshal([]byte("Name: foo\nIncludedScratch: 4GB\nRAM: 4GiB\n"), &it)
	c.Check(err, check.IsNil)
	c.Check(int64(it.IncludedScratch), check.Equals, int64(4000000000))
	c.Check(int64(it.RAM), check.Equals, int64(4294967296))
}

func (s *ConfigSuite) TestInstanceTypeFixup(c *check.C) {
	for _, confdata := range []string{
		// Current format: map of entries
		`{foo4: {IncludedScratch: 4GB}, foo8: {ProviderType: foo_8, AddedScratch: 8GB}}`,
	} {
		c.Log(confdata)
		var itm InstanceTypeMap
		err := yaml.Unmarshal([]byte(confdata), &itm)
		c.Check(err, check.IsNil)

		c.Check(itm["foo4"].Name, check.Equals, "foo4")
		c.Check(itm["foo4"].ProviderType, check.Equals, "foo4")
		c.Check(itm["foo4"].Scratch, check.Equals, ByteSize(4000000000))
		c.Check(itm["foo4"].AddedScratch, check.Equals, ByteSize(0))
		c.Check(itm["foo4"].IncludedScratch, check.Equals, ByteSize(4000000000))

		c.Check(itm["foo8"].Name, check.Equals, "foo8")
		c.Check(itm["foo8"].ProviderType, check.Equals, "foo_8")
		c.Check(itm["foo8"].Scratch, check.Equals, ByteSize(8000000000))
		c.Check(itm["foo8"].AddedScratch, check.Equals, ByteSize(8000000000))
		c.Check(itm["foo8"].IncludedScratch, check.Equals, ByteSize(0))
	}
}

func (s *ConfigSuite) TestURLTrailingSlash(c *check.C) {
	var a, b map[URL]bool
	json.Unmarshal([]byte(`{"https://foo.example": true}`), &a)
	json.Unmarshal([]byte(`{"https://foo.example/": true}`), &b)
	c.Check(a, check.DeepEquals, b)
}

func (s *ConfigSuite) TestTLSVersion(c *check.C) {
	var v struct {
		Version TLSVersion
	}
	err := json.Unmarshal([]byte(`{"Version": 1.0}`), &v)
	c.Check(err, check.IsNil)
	c.Check(v.Version, check.Equals, TLSVersion(tls.VersionTLS10))

	err = json.Unmarshal([]byte(`{"Version": "1.3"}`), &v)
	c.Check(err, check.IsNil)
	c.Check(v.Version, check.Equals, TLSVersion(tls.VersionTLS13))

	err = json.Unmarshal([]byte(`{"Version": "1.345"}`), &v)
	c.Check(err, check.NotNil)
}
