// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&LoadSuite{})

type LoadSuite struct{}

func (s *LoadSuite) TestEmpty(c *check.C) {
	cfg, err := Load(&bytes.Buffer{}, ctxlog.TestLogger(c))
	c.Check(cfg, check.IsNil)
	c.Assert(err, check.ErrorMatches, `config does not define any clusters`)
}

func (s *LoadSuite) TestNoConfigs(c *check.C) {
	cfg, err := Load(bytes.NewBufferString(`Clusters: {"z1111": {}}`), ctxlog.TestLogger(c))
	c.Assert(err, check.IsNil)
	c.Assert(cfg.Clusters, check.HasLen, 1)
	cc, err := cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(cc.ClusterID, check.Equals, "z1111")
	c.Check(cc.API.MaxRequestAmplification, check.Equals, 4)
	c.Check(cc.API.MaxItemsPerResponse, check.Equals, 1000)
}

func (s *LoadSuite) TestMultipleClusters(c *check.C) {
	cfg, err := Load(bytes.NewBufferString(`{"Clusters":{"z1111":{},"z2222":{}}}`), ctxlog.TestLogger(c))
	c.Assert(err, check.IsNil)
	c1, err := cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(c1.ClusterID, check.Equals, "z1111")
	c2, err := cfg.GetCluster("z2222")
	c.Assert(err, check.IsNil)
	c.Check(c2.ClusterID, check.Equals, "z2222")
}

func (s *LoadSuite) TestPostgreSQLKeyConflict(c *check.C) {
	_, err := Load(bytes.NewBufferString(`
Clusters:
 zzzzz:
  postgresql:
   connection:
     dbname: dbname
     host: host
`), ctxlog.TestLogger(c))
	c.Assert(err, check.ErrorMatches, `Clusters.zzzzz.PostgreSQL.Connection: multiple keys with .*"(dbname|host)".*`)
}

func (s *LoadSuite) TestMovedKeys(c *check.C) {
	s.checkEquivalent(c, `# config has old keys only
Clusters:
 zzzzz:
  RequestLimits:
   MultiClusterRequestConcurrency: 3
   MaxItemsPerResponse: 999
`, `
Clusters:
 zzzzz:
  API:
   MaxRequestAmplification: 3
   MaxItemsPerResponse: 999
`)
	s.checkEquivalent(c, `# config has both old and new keys; old values win
Clusters:
 zzzzz:
  RequestLimits:
   MultiClusterRequestConcurrency: 0
   MaxItemsPerResponse: 555
  API:
   MaxRequestAmplification: 3
   MaxItemsPerResponse: 999
`, `
Clusters:
 zzzzz:
  API:
   MaxRequestAmplification: 0
   MaxItemsPerResponse: 555
`)
}

func (s *LoadSuite) checkEquivalent(c *check.C, goty, expectedy string) {
	got, err := Load(bytes.NewBufferString(goty), ctxlog.TestLogger(c))
	c.Assert(err, check.IsNil)
	expected, err := Load(bytes.NewBufferString(expectedy), ctxlog.TestLogger(c))
	c.Assert(err, check.IsNil)
	if !c.Check(got, check.DeepEquals, expected) {
		cmd := exec.Command("diff", "-u", "--label", "expected", "--label", "got", "/dev/fd/3", "/dev/fd/4")
		for _, obj := range []interface{}{expected, got} {
			y, _ := yaml.Marshal(obj)
			pr, pw, err := os.Pipe()
			c.Assert(err, check.IsNil)
			defer pr.Close()
			go func() {
				io.Copy(pw, bytes.NewBuffer(y))
				pw.Close()
			}()
			cmd.ExtraFiles = append(cmd.ExtraFiles, pr)
		}
		diff, err := cmd.CombinedOutput()
		c.Log(string(diff))
		c.Check(err, check.IsNil)
	}
}
