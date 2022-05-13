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
	"regexp"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&LoadSuite{})

var emptyConfigYAML = `Clusters: {"z1111": {}}`

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

func (s *LoadSuite) SetUpSuite(c *check.C) {
	os.Unsetenv("ARVADOS_API_HOST")
	os.Unsetenv("ARVADOS_API_HOST_INSECURE")
	os.Unsetenv("ARVADOS_API_TOKEN")
}

func (s *LoadSuite) TestEmpty(c *check.C) {
	cfg, err := testLoader(c, "", nil).Load()
	c.Check(cfg, check.IsNil)
	c.Assert(err, check.ErrorMatches, `config does not define any clusters`)
}

func (s *LoadSuite) TestNoConfigs(c *check.C) {
	cfg, err := testLoader(c, emptyConfigYAML, nil).Load()
	c.Assert(err, check.IsNil)
	c.Assert(cfg.Clusters, check.HasLen, 1)
	cc, err := cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(cc.ClusterID, check.Equals, "z1111")
	c.Check(cc.API.MaxRequestAmplification, check.Equals, 4)
	c.Check(cc.API.MaxItemsPerResponse, check.Equals, 1000)
}

func (s *LoadSuite) TestNullKeyDoesNotOverrideDefault(c *check.C) {
	ldr := testLoader(c, `{"Clusters":{"z1111":{"API":}}}`, nil)
	ldr.SkipDeprecated = true
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	c1, err := cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(c1.ClusterID, check.Equals, "z1111")
	c.Check(c1.API.MaxRequestAmplification, check.Equals, 4)
	c.Check(c1.API.MaxItemsPerResponse, check.Equals, 1000)
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
	io.WriteString(f, emptyConfigYAML)
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
		c.Assert(err, check.IsNil)
		_, hasSample := cc.InstanceTypes["SAMPLE"]
		c.Check(hasSample, check.Equals, false)
		if strings.Contains(yaml, "Foo") {
			c.Check(cc.InstanceTypes["Foo"].RAM, check.Equals, arvados.ByteSize(12345000000))
			c.Check(cc.InstanceTypes["Foo"].Price, check.Equals, 0.0)
		}
	}
}

func (s *LoadSuite) TestMultipleClusters(c *check.C) {
	ldr := testLoader(c, `{"Clusters":{"z1111":{},"z2222":{}}}`, nil)
	ldr.SkipDeprecated = true
	cfg, err := ldr.Load()
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
    ManagementToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    SystemRootToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    Collections:
     BlobSigningKey: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
    PostgreSQL: {}
    BadKey1: {}
    Containers:
      RunTimeEngine: abc
    RemoteClusters:
      z2222:
        Host: z2222.arvadosapi.com
        Proxy: true
        BadKey2: badValue
    Services:
      KeepStore:
        InternalURLs:
          "http://host.example:12345": {}
      Keepstore:
        InternalURLs:
          "http://host.example:12345":
            RendezVous: x
    ServiceS:
      Keepstore:
        InternalURLs:
          "http://host.example:12345": {}
    Volumes:
      zzzzz-nyw5e-aaaaaaaaaaaaaaa: {}
`, &logbuf).Load()
	c.Assert(err, check.IsNil)
	c.Log(logbuf.String())
	logs := strings.Split(strings.TrimSuffix(logbuf.String(), "\n"), "\n")
	for _, log := range logs {
		c.Check(log, check.Matches, `.*deprecated or unknown config entry:.*(RunTimeEngine.*RuntimeEngine|BadKey1|BadKey2|KeepStore|ServiceS|RendezVous).*`)
	}
	c.Check(logs, check.HasLen, 6)
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
	var logbuf bytes.Buffer
	loader := testLoader(c, string(DefaultYAML), &logbuf)
	cfg, err := loader.Load()
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

	c.Check(logbuf.String(), check.Matches, `(?ms).*SystemRootToken: secret token is not set.*`)
	c.Check(logbuf.String(), check.Matches, `(?ms).*ManagementToken: secret token is not set.*`)
	c.Check(logbuf.String(), check.Matches, `(?ms).*Collections.BlobSigningKey: secret token is not set.*`)
	logbuf.Reset()
	loader.logExtraKeys(loaded, supplied, "")
	c.Check(logbuf.String(), check.Equals, "")
}

func (s *LoadSuite) TestNoWarningsForDumpedConfig(c *check.C) {
	var logbuf bytes.Buffer
	cfg, err := testLoader(c, `
Clusters:
 zzzzz:
  ManagementToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  SystemRootToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  Collections:
   BlobSigningKey: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  InstanceTypes:
   abc:
    IncludedScratch: 123456
`, &logbuf).Load()
	c.Assert(err, check.IsNil)
	yaml, err := yaml.Marshal(cfg)
	c.Assert(err, check.IsNil)
	// Well, *nearly* no warnings. SourceTimestamp and
	// SourceSHA256 are included in a config-dump, but not
	// expected in a real config file.
	yaml = regexp.MustCompile(`(^|\n)(Source(Timestamp|SHA256): .*?\n)+`).ReplaceAll(yaml, []byte("$1"))
	cfgDumped, err := testLoader(c, string(yaml), &logbuf).Load()
	c.Assert(err, check.IsNil)
	// SourceTimestamp and SourceSHA256 aren't expected to be
	// preserved through dump+load
	cfgDumped.SourceTimestamp = cfg.SourceTimestamp
	cfgDumped.SourceSHA256 = cfg.SourceSHA256
	c.Check(cfg, check.DeepEquals, cfgDumped)
	c.Check(logbuf.String(), check.Equals, "")
}

func (s *LoadSuite) TestUnacceptableTokens(c *check.C) {
	for _, trial := range []struct {
		short      bool
		configPath string
		example    string
	}{
		{false, "SystemRootToken", "SystemRootToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_b_c"},
		{false, "ManagementToken", "ManagementToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa b c"},
		{false, "ManagementToken", "ManagementToken: \"$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabc\""},
		{false, "Collections.BlobSigningKey", "Collections: {BlobSigningKey: \"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa⛵\"}"},
		{true, "SystemRootToken", "SystemRootToken: a_b_c"},
		{true, "ManagementToken", "ManagementToken: a b c"},
		{true, "ManagementToken", "ManagementToken: \"$abc\""},
		{true, "Collections.BlobSigningKey", "Collections: {BlobSigningKey: \"⛵\"}"},
	} {
		c.Logf("trying bogus config: %s", trial.example)
		_, err := testLoader(c, "Clusters:\n zzzzz:\n  "+trial.example, nil).Load()
		c.Check(err, check.ErrorMatches, `Clusters.zzzzz.`+trial.configPath+`: unacceptable characters in token.*`)
	}
}

func (s *LoadSuite) TestPostgreSQLKeyConflict(c *check.C) {
	_, err := testLoader(c, `
Clusters:
 zzzzz:
  PostgreSQL:
   Connection:
     DBName: dbname
     Host: host
`, nil).Load()
	c.Check(err, check.ErrorMatches, `Clusters.zzzzz.PostgreSQL.Connection: multiple entries for "(dbname|host)".*`)
}

func (s *LoadSuite) TestBadClusterIDs(c *check.C) {
	for _, data := range []string{`
Clusters:
 123456:
  ManagementToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  SystemRootToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  Collections:
   BlobSigningKey: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
`, `
Clusters:
 12345:
  ManagementToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  SystemRootToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  Collections:
   BlobSigningKey: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  RemoteClusters:
   Zzzzz:
    Host: Zzzzz.arvadosapi.com
    Proxy: true
`, `
Clusters:
 abcde:
  ManagementToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  SystemRootToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  Collections:
   BlobSigningKey: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  Login:
   LoginCluster: zz-zz
`,
	} {
		c.Log(data)
		v, err := testLoader(c, data, nil).Load()
		if v != nil {
			c.Logf("%#v", v.Clusters)
		}
		c.Check(err, check.ErrorMatches, `.*cluster ID should be 5 alphanumeric characters.*`)
	}
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
	checkEquivalent(c, `# config has old keys only
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
	checkEquivalent(c, `# config has both old and new keys; old values win
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

func checkEquivalent(c *check.C, goty, expectedy string) string {
	var logbuf bytes.Buffer
	gotldr := testLoader(c, goty, &logbuf)
	expectedldr := testLoader(c, expectedy, nil)
	checkEquivalentLoaders(c, gotldr, expectedldr)
	return logbuf.String()
}

func checkEqualYAML(c *check.C, got, expected interface{}) {
	expectedyaml, err := yaml.Marshal(expected)
	c.Assert(err, check.IsNil)
	gotyaml, err := yaml.Marshal(got)
	c.Assert(err, check.IsNil)
	if !bytes.Equal(gotyaml, expectedyaml) {
		cmd := exec.Command("diff", "-u", "--label", "expected", "--label", "got", "/dev/fd/3", "/dev/fd/4")
		for _, y := range [][]byte{expectedyaml, gotyaml} {
			pr, pw, err := os.Pipe()
			c.Assert(err, check.IsNil)
			defer pr.Close()
			go func(data []byte) {
				pw.Write(data)
				pw.Close()
			}(y)
			cmd.ExtraFiles = append(cmd.ExtraFiles, pr)
		}
		diff, err := cmd.CombinedOutput()
		// diff should report differences and exit non-zero.
		c.Check(err, check.NotNil)
		c.Log(string(diff))
		c.Error("got != expected; see diff (-expected +got) above")
	}
}

func checkEquivalentLoaders(c *check.C, gotldr, expectedldr *Loader) {
	got, err := gotldr.Load()
	c.Assert(err, check.IsNil)
	expected, err := expectedldr.Load()
	c.Assert(err, check.IsNil)
	// The inputs generally aren't even files, so SourceTimestamp
	// can't be expected to match.
	got.SourceTimestamp = expected.SourceTimestamp
	// Obviously the content isn't identical -- otherwise we
	// wouldn't need to check that it's equivalent.
	got.SourceSHA256 = expected.SourceSHA256
	checkEqualYAML(c, got, expected)
}

func checkListKeys(path string, x interface{}) (err error) {
	v := reflect.Indirect(reflect.ValueOf(x))
	switch v.Kind() {
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			k := iter.Key()
			if k.Kind() == reflect.String {
				if err = checkListKeys(path+"."+k.String(), iter.Value().Interface()); err != nil {
					return
				}
			}
		}
		return

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			val := v.Field(i)
			structField := v.Type().Field(i)
			fieldname := structField.Name
			endsWithList := strings.HasSuffix(fieldname, "List")
			isAnArray := structField.Type.Kind() == reflect.Slice
			if endsWithList != isAnArray {
				if endsWithList {
					err = fmt.Errorf("%s.%s ends with 'List' but field is not an array (type %v)", path, fieldname, val.Kind())
					return
				}
				if isAnArray && structField.Type.Elem().Kind() != reflect.Uint8 {
					err = fmt.Errorf("%s.%s is an array but field name does not end in 'List' (slice of %v)", path, fieldname, structField.Type.Elem().Kind())
					return
				}
			}
			if val.CanInterface() {
				checkListKeys(path+"."+fieldname, val.Interface())
			}
		}
	}
	return
}

func (s *LoadSuite) TestListKeys(c *check.C) {
	v1 := struct {
		EndInList []string
	}{[]string{"a", "b"}}
	var m1 = make(map[string]interface{})
	m1["c"] = &v1
	if err := checkListKeys("", m1); err != nil {
		c.Error(err)
	}

	v2 := struct {
		DoesNot []string
	}{[]string{"a", "b"}}
	var m2 = make(map[string]interface{})
	m2["c"] = &v2
	if err := checkListKeys("", m2); err == nil {
		c.Errorf("Should have produced an error")
	}

	v3 := struct {
		EndInList string
	}{"a"}
	var m3 = make(map[string]interface{})
	m3["c"] = &v3
	if err := checkListKeys("", m3); err == nil {
		c.Errorf("Should have produced an error")
	}

	loader := testLoader(c, string(DefaultYAML), nil)
	cfg, err := loader.Load()
	c.Assert(err, check.IsNil)
	if err := checkListKeys("", cfg); err != nil {
		c.Error(err)
	}
}

func (s *LoadSuite) TestImplicitStorageClasses(c *check.C) {
	// If StorageClasses and Volumes.*.StorageClasses are all
	// empty, there is a default storage class named "default".
	ldr := testLoader(c, `{"Clusters":{"z1111":{}}}`, nil)
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	cc, err := cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(cc.StorageClasses, check.HasLen, 1)
	c.Check(cc.StorageClasses["default"].Default, check.Equals, true)
	c.Check(cc.StorageClasses["default"].Priority, check.Equals, 0)

	// The implicit "default" storage class is used by all
	// volumes.
	ldr = testLoader(c, `
Clusters:
 z1111:
  Volumes:
   z: {}`, nil)
	cfg, err = ldr.Load()
	c.Assert(err, check.IsNil)
	cc, err = cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(cc.StorageClasses, check.HasLen, 1)
	c.Check(cc.StorageClasses["default"].Default, check.Equals, true)
	c.Check(cc.StorageClasses["default"].Priority, check.Equals, 0)
	c.Check(cc.Volumes["z"].StorageClasses["default"], check.Equals, true)

	// The "default" storage class isn't implicit if any classes
	// are configured explicitly.
	ldr = testLoader(c, `
Clusters:
 z1111:
  StorageClasses:
   local:
    Default: true
    Priority: 111
  Volumes:
   z:
    StorageClasses:
     local: true`, nil)
	cfg, err = ldr.Load()
	c.Assert(err, check.IsNil)
	cc, err = cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(cc.StorageClasses, check.HasLen, 1)
	c.Check(cc.StorageClasses["local"].Default, check.Equals, true)
	c.Check(cc.StorageClasses["local"].Priority, check.Equals, 111)

	// It is an error for a volume to refer to a storage class
	// that isn't listed in StorageClasses.
	ldr = testLoader(c, `
Clusters:
 z1111:
  StorageClasses:
   local:
    Default: true
    Priority: 111
  Volumes:
   z:
    StorageClasses:
     nx: true`, nil)
	_, err = ldr.Load()
	c.Assert(err, check.ErrorMatches, `z: volume refers to storage class "nx" that is not defined.*`)

	// It is an error for a volume to refer to a storage class
	// that isn't listed in StorageClasses ... even if it's
	// "default", which would exist implicitly if it weren't
	// referenced explicitly by a volume.
	ldr = testLoader(c, `
Clusters:
 z1111:
  Volumes:
   z:
    StorageClasses:
     default: true`, nil)
	_, err = ldr.Load()
	c.Assert(err, check.ErrorMatches, `z: volume refers to storage class "default" that is not defined.*`)

	// If the "default" storage class is configured explicitly, it
	// is not used implicitly by any volumes, even if it's the
	// only storage class.
	var logbuf bytes.Buffer
	ldr = testLoader(c, `
Clusters:
 z1111:
  StorageClasses:
   default:
    Default: true
    Priority: 111
  Volumes:
   z: {}`, &logbuf)
	_, err = ldr.Load()
	c.Assert(err, check.ErrorMatches, `z: volume has no StorageClasses listed`)

	// If StorageClasses are configured explicitly, there must be
	// at least one with Default: true. (Calling one "default" is
	// not sufficient.)
	ldr = testLoader(c, `
Clusters:
 z1111:
  StorageClasses:
   default:
    Priority: 111
  Volumes:
   z:
    StorageClasses:
     default: true`, nil)
	_, err = ldr.Load()
	c.Assert(err, check.ErrorMatches, `there is no default storage class.*`)
}

func (s *LoadSuite) TestPreemptiblePriceFactor(c *check.C) {
	yaml := `
Clusters:
 z1111:
  InstanceTypes:
   Type1:
    RAM: 12345M
    VCPUs: 8
    Price: 1.23
 z2222:
  Containers:
   PreemptiblePriceFactor: 0.5
  InstanceTypes:
   Type1:
    RAM: 12345M
    VCPUs: 8
    Price: 1.23
 z3333:
  Containers:
   PreemptiblePriceFactor: 0.5
  InstanceTypes:
   Type1:
    RAM: 12345M
    VCPUs: 8
    Price: 1.23
   Type1.preemptible: # higher price than the auto-added variant would use -- should generate warning
    ProviderType: Type1
    RAM: 12345M
    VCPUs: 8
    Price: 1.23
    Preemptible: true
   Type2:
    RAM: 23456M
    VCPUs: 16
    Price: 2.46
   Type2.preemptible: # identical to the auto-added variant -- so no warning
    ProviderType: Type2
    RAM: 23456M
    VCPUs: 16
    Price: 1.23
    Preemptible: true
`
	var logbuf bytes.Buffer
	cfg, err := testLoader(c, yaml, &logbuf).Load()
	c.Assert(err, check.IsNil)
	cc, err := cfg.GetCluster("z1111")
	c.Assert(err, check.IsNil)
	c.Check(cc.InstanceTypes["Type1"].Price, check.Equals, 1.23)
	c.Check(cc.InstanceTypes, check.HasLen, 1)

	cc, err = cfg.GetCluster("z2222")
	c.Assert(err, check.IsNil)
	c.Check(cc.InstanceTypes["Type1"].Preemptible, check.Equals, false)
	c.Check(cc.InstanceTypes["Type1"].Price, check.Equals, 1.23)
	c.Check(cc.InstanceTypes["Type1.preemptible"].Preemptible, check.Equals, true)
	c.Check(cc.InstanceTypes["Type1.preemptible"].Price, check.Equals, 1.23/2)
	c.Check(cc.InstanceTypes["Type1.preemptible"].ProviderType, check.Equals, "Type1")
	c.Check(cc.InstanceTypes, check.HasLen, 2)

	cc, err = cfg.GetCluster("z3333")
	c.Assert(err, check.IsNil)
	// Don't overwrite the explicitly configured preemptible variant
	c.Check(cc.InstanceTypes["Type1.preemptible"].Price, check.Equals, 1.23)
	c.Check(cc.InstanceTypes, check.HasLen, 4)
	c.Check(logbuf.String(), check.Matches, `(?ms).*Clusters\.z3333\.InstanceTypes\[Type1\.preemptible\]: already exists, so not automatically adding a preemptible variant of Type1.*`)
	c.Check(logbuf.String(), check.Not(check.Matches), `(?ms).*Type2\.preemptible.*`)
	c.Check(logbuf.String(), check.Not(check.Matches), `(?ms).*(z1111|z2222)[^\n]*InstanceTypes.*`)
}

func (s *LoadSuite) TestSourceTimestamp(c *check.C) {
	conftime, err := time.Parse(time.RFC3339, "2022-03-04T05:06:07-08:00")
	c.Assert(err, check.IsNil)
	confdata := `Clusters: {zzzzz: {}}`
	conffile := c.MkDir() + "/config.yml"
	ioutil.WriteFile(conffile, []byte(confdata), 0777)
	tv := unix.NsecToTimeval(conftime.UnixNano())
	unix.Lutimes(conffile, []unix.Timeval{tv, tv})
	for _, trial := range []struct {
		configarg  string
		expectTime time.Time
	}{
		{"-", time.Now()},
		{conffile, conftime},
	} {
		c.Logf("trial: %+v", trial)
		ldr := NewLoader(strings.NewReader(confdata), ctxlog.TestLogger(c))
		ldr.Path = trial.configarg
		cfg, err := ldr.Load()
		c.Assert(err, check.IsNil)
		c.Check(cfg.SourceTimestamp, check.Equals, cfg.SourceTimestamp.UTC())
		c.Check(cfg.SourceTimestamp, check.Equals, ldr.sourceTimestamp)
		c.Check(int(cfg.SourceTimestamp.Sub(trial.expectTime).Seconds()), check.Equals, 0)
		c.Check(int(ldr.loadTimestamp.Sub(time.Now()).Seconds()), check.Equals, 0)

		var buf bytes.Buffer
		reg := prometheus.NewRegistry()
		ldr.RegisterMetrics(reg)
		enc := expfmt.NewEncoder(&buf, expfmt.FmtText)
		got, _ := reg.Gather()
		for _, mf := range got {
			enc.Encode(mf)
		}
		c.Check(buf.String(), check.Matches, `# HELP .*
# TYPE .*
arvados_config_load_timestamp_seconds{sha256="83aea5d82eb1d53372cd65c936c60acc1c6ef946e61977bbca7cfea709d201a8"} \Q`+fmt.Sprintf("%g", float64(ldr.loadTimestamp.UnixNano())/1e9)+`\E
# HELP .*
# TYPE .*
arvados_config_source_timestamp_seconds{sha256="83aea5d82eb1d53372cd65c936c60acc1c6ef946e61977bbca7cfea709d201a8"} \Q`+fmt.Sprintf("%g", float64(cfg.SourceTimestamp.UnixNano())/1e9)+`\E
`)
	}
}
