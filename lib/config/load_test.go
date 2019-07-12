// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
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

// Return a new Loader that reads cluster config from configdata
// (instead of the usual default /etc/arvados/config.yml), and logs to
// logdst or (if that's nil) c.Log.
func testLoader(c *check.C, configdata string, logdst io.Writer) *Loader {
	logger := ctxlog.TestLogger(c)
	if logdst != nil {
		lgr := logrus.New()
		lgr.Out = logdst
		logger = lgr
	}
	ldr := NewLoader(bytes.NewBufferString(configdata), logger)
	ldr.Path = "-"
	return ldr
}

type LoadSuite struct{}

func (s *LoadSuite) TestEmpty(c *check.C) {
	cfg, err := testLoader(c, "", nil).Load()
	c.Check(cfg, check.IsNil)
	c.Assert(err, check.ErrorMatches, `config does not define any clusters`)
}

func (s *LoadSuite) TestNoConfigs(c *check.C) {
	cfg, err := testLoader(c, `Clusters: {"z1111": {}}`, nil).Load()
	c.Assert(err, check.IsNil)
	c.Assert(cfg.Clusters, check.HasLen, 1)
	cc, err := cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(cc.ClusterID, check.Equals, "z1111")
	c.Check(cc.API.MaxRequestAmplification, check.Equals, 4)
	c.Check(cc.API.MaxItemsPerResponse, check.Equals, 1000)
}

func (s *LoadSuite) TestMungeLegacyConfigArgs(c *check.C) {
	f, err := ioutil.TempFile("", "")
	c.Check(err, check.IsNil)
	defer os.Remove(f.Name())
	io.WriteString(f, "Debug: true\n")
	oldfile := f.Name()

	f, err = ioutil.TempFile("", "")
	c.Check(err, check.IsNil)
	defer os.Remove(f.Name())
	io.WriteString(f, "Clusters: {aaaaa: {}}\n")
	newfile := f.Name()

	for _, trial := range []struct {
		argsIn  []string
		argsOut []string
	}{
		{
			[]string{"-config", oldfile},
			[]string{"-old-config", oldfile},
		},
		{
			[]string{"-config=" + oldfile},
			[]string{"-old-config=" + oldfile},
		},
		{
			[]string{"-config", newfile},
			[]string{"-config", newfile},
		},
		{
			[]string{"-config=" + newfile},
			[]string{"-config=" + newfile},
		},
		{
			[]string{"-foo", oldfile},
			[]string{"-foo", oldfile},
		},
		{
			[]string{"-foo=" + oldfile},
			[]string{"-foo=" + oldfile},
		},
		{
			[]string{"-foo", "-config=" + oldfile},
			[]string{"-foo", "-old-config=" + oldfile},
		},
		{
			[]string{"-foo", "bar", "-config=" + oldfile},
			[]string{"-foo", "bar", "-old-config=" + oldfile},
		},
		{
			[]string{"-foo=bar", "baz", "-config=" + oldfile},
			[]string{"-foo=bar", "baz", "-old-config=" + oldfile},
		},
		{
			[]string{"-config=/dev/null"},
			[]string{"-config=/dev/null"},
		},
		{
			[]string{"-config=-"},
			[]string{"-config=-"},
		},
		{
			[]string{"-config="},
			[]string{"-config="},
		},
		{
			[]string{"-foo=bar", "baz", "-config"},
			[]string{"-foo=bar", "baz", "-config"},
		},
		{
			[]string{},
			nil,
		},
	} {
		var logbuf bytes.Buffer
		logger := logrus.New()
		logger.Out = &logbuf

		var ldr Loader
		args := ldr.MungeLegacyConfigArgs(logger, trial.argsIn, "-old-config")
		c.Check(args, check.DeepEquals, trial.argsOut)
		if fmt.Sprintf("%v", trial.argsIn) != fmt.Sprintf("%v", trial.argsOut) {
			c.Check(logbuf.String(), check.Matches, `.*`+oldfile+` is not a cluster config file -- interpreting -config as -old-config.*\n`)
		}
	}
}

func (s *LoadSuite) TestSampleKeys(c *check.C) {
	for _, yaml := range []string{
		`{"Clusters":{"z1111":{}}}`,
		`{"Clusters":{"z1111":{"InstanceTypes":{"Foo":{"RAM": "12345M"}}}}}`,
	} {
		cfg, err := testLoader(c, yaml, nil).Load()
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
	cfg, err := testLoader(c, `{"Clusters":{"z1111":{},"z2222":{}}}`, nil).Load()
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
	_, err := testLoader(c, `
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
`, &logbuf).Load()
	c.Assert(err, check.IsNil)
	logs := strings.Split(strings.TrimSuffix(logbuf.String(), "\n"), "\n")
	for _, log := range logs {
		c.Check(log, check.Matches, `.*deprecated or unknown config entry:.*BadKey.*`)
	}
	c.Check(logs, check.HasLen, 2)
}

func (s *LoadSuite) checkSAMPLEKeys(c *check.C, path string, x interface{}) {
	v := reflect.Indirect(reflect.ValueOf(x))
	switch v.Kind() {
	case reflect.Map:
		var stringKeys, sampleKey bool
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() == reflect.String {
				stringKeys = true
				if k.String() == "SAMPLE" || k.String() == "xxxxx" {
					sampleKey = true
					s.checkSAMPLEKeys(c, path+"."+k.String(), iter.Value().Interface())
				}
			}
		}
		if stringKeys && !sampleKey {
			c.Errorf("%s is a map with string keys (type %T) but config.default.yml has no SAMPLE key", path, x)
		}
		return
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			val := v.Field(i)
			if val.CanInterface() {
				s.checkSAMPLEKeys(c, path+"."+v.Type().Field(i).Name, val.Interface())
			}
		}
	}
}

func (s *LoadSuite) TestDefaultConfigHasAllSAMPLEKeys(c *check.C) {
	cfg, err := Load(bytes.NewBuffer(DefaultYAML), ctxlog.TestLogger(c))
	c.Assert(err, check.IsNil)
	s.checkSAMPLEKeys(c, "", cfg)
}

func (s *LoadSuite) TestNoUnrecognizedKeysInDefaultConfig(c *check.C) {
	var logbuf bytes.Buffer
	var supplied map[string]interface{}
	yaml.Unmarshal(DefaultYAML, &supplied)

	loader := testLoader(c, string(DefaultYAML), &logbuf)
	cfg, err := loader.Load()
	c.Assert(err, check.IsNil)
	var loaded map[string]interface{}
	buf, err := yaml.Marshal(cfg)
	c.Assert(err, check.IsNil)
	err = yaml.Unmarshal(buf, &loaded)
	c.Assert(err, check.IsNil)

	loader.logExtraKeys(loaded, supplied, "")
	c.Check(logbuf.String(), check.Equals, "")
}

func (s *LoadSuite) TestNoWarningsForDumpedConfig(c *check.C) {
	var logbuf bytes.Buffer
	logger := logrus.New()
	logger.Out = &logbuf
	cfg, err := testLoader(c, `{"Clusters":{"zzzzz":{}}}`, &logbuf).Load()
	c.Assert(err, check.IsNil)
	yaml, err := yaml.Marshal(cfg)
	c.Assert(err, check.IsNil)
	cfgDumped, err := testLoader(c, string(yaml), &logbuf).Load()
	c.Assert(err, check.IsNil)
	c.Check(cfg, check.DeepEquals, cfgDumped)
	c.Check(logbuf.String(), check.Equals, "")
}

func (s *LoadSuite) TestPostgreSQLKeyConflict(c *check.C) {
	_, err := testLoader(c, `
Clusters:
 zzzzz:
  postgresql:
   connection:
     DBName: dbname
     Host: host
`, nil).Load()
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
		v, err := testLoader(c, data, nil).Load()
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
	got, err := testLoader(c, goty, nil).Load()
	c.Assert(err, check.IsNil)
	expected, err := testLoader(c, expectedy, nil).Load()
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
