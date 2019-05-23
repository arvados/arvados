// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
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

func (s *LoadSuite) TestSampleKeys(c *check.C) {
	for _, yaml := range []string{
		`{"Clusters":{"z1111":{}}}`,
		`{"Clusters":{"z1111":{"InstanceTypes":{"Foo":{"RAM": "12345M"}}}}}`,
	} {
		cfg, err := Load(bytes.NewBufferString(yaml), ctxlog.TestLogger(c))
		c.Assert(err, check.IsNil)
		cc, err := cfg.GetCluster("z1111")
		_, hasSample := cc.InstanceTypes["SAMPLE"]
		c.Check(hasSample, check.Equals, false)
		if strings.Contains(yaml, "Foo") {
			c.Check(cc.InstanceTypes["Foo"].RAM, check.Equals, arvados.ByteSize(12345000000))
			c.Check(cc.InstanceTypes["Foo"].Price, check.Equals, 0.0)
		}
	}
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

func (s *LoadSuite) TestDeprecatedOrUnknownWarning(c *check.C) {
	var logbuf bytes.Buffer
	logger := logrus.New()
	logger.Out = &logbuf
	_, err := Load(bytes.NewBufferString(`
Clusters:
  zzzzz:
    postgresql: {}
    BadKey: {}
    Containers: {}
    RemoteClusters:
      z2222:
        Host: z2222.arvadosapi.com
        Proxy: true
        BadKey: badValue
`), logger)
	c.Assert(err, check.IsNil)
	logs := strings.Split(strings.TrimSuffix(logbuf.String(), "\n"), "\n")
	for _, log := range logs {
		c.Check(log, check.Matches, `.*deprecated or unknown config entry:.*BadKey.*`)
	}
	c.Check(logs, check.HasLen, 2)
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
	c.Check(err, check.ErrorMatches, `Clusters.zzzzz.PostgreSQL.Connection: multiple entries for "(dbname|host)".*`)
}

func (s *LoadSuite) TestBadType(c *check.C) {
	for _, data := range []string{`
Clusters:
 zzzzz:
  PostgreSQL: true
`, `
Clusters:
 zzzzz:
  PostgreSQL:
   ConnectionPool: true
`, `
Clusters:
 zzzzz:
  PostgreSQL:
   ConnectionPool: "foo"
`, `
Clusters:
 zzzzz:
  PostgreSQL:
   ConnectionPool: []
`, `
Clusters:
 zzzzz:
  PostgreSQL:
   ConnectionPool: [] # {foo: bar} isn't caught here; we rely on config-check
`,
	} {
		c.Log(data)
		v, err := Load(bytes.NewBufferString(data), ctxlog.TestLogger(c))
		if v != nil {
			c.Logf("%#v", v.Clusters["zzzzz"].PostgreSQL.ConnectionPool)
		}
		c.Check(err, check.ErrorMatches, `.*cannot unmarshal .*PostgreSQL.*`)
	}
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
