// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

// Configured at: sdk/python/tests/run_test_server.py
const TestServerManagementToken = "e687950a23c3a9bceec28c6223a06c79"

func testLoadLegacyConfig(content []byte, mungeFlag string, c *check.C) (*arvados.Cluster, error) {
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}
	flags := flag.NewFlagSet("test", flag.ExitOnError)
	ldr := testLoader(c, "Clusters: {zzzzz: {}}", nil)
	ldr.SetupFlags(flags)
	args := ldr.MungeLegacyConfigArgs(ldr.Logger, []string{"-config", tmpfile.Name()}, mungeFlag)
	err = flags.Parse(args)
	c.Assert(err, check.IsNil)
	c.Assert(flags.NArg(), check.Equals, 0)
	cfg, err := ldr.Load()
	if err != nil {
		return nil, err
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (s *LoadSuite) TestLegacyVolumeDriverParameters(c *check.C) {
	logs := checkEquivalent(c, `
Clusters:
 z1111:
  Volumes:
   z1111-nyw5e-aaaaaaaaaaaaaaa:
    Driver: S3
    DriverParameters:
     AccessKey: exampleaccesskey
     SecretKey: examplesecretkey
     Region: foobar
     ReadTimeout: 1200s
`, `
Clusters:
 z1111:
  Volumes:
   z1111-nyw5e-aaaaaaaaaaaaaaa:
    Driver: S3
    DriverParameters:
     AccessKeyID: exampleaccesskey
     SecretAccessKey: examplesecretkey
     Region: foobar
     ReadTimeout: 1200s
`)
	c.Check(logs, check.Matches, `(?ms).*deprecated or unknown config entry: .*AccessKey.*`)
	c.Check(logs, check.Matches, `(?ms).*deprecated or unknown config entry: .*SecretKey.*`)
	c.Check(logs, check.Matches, `(?ms).*using your old config keys z1111\.Volumes\.z1111-nyw5e-aaaaaaaaaaaaaaa\.DriverParameters\.AccessKey/SecretKey -- but you should rename them to AccessKeyID/SecretAccessKey.*`)

	_, err := testLoader(c, `
Clusters:
 z1111:
  Volumes:
   z1111-nyw5e-aaaaaaaaaaaaaaa:
    Driver: S3
    DriverParameters:
     AccessKey: exampleaccesskey
     SecretKey: examplesecretkey
     AccessKeyID: exampleaccesskey
`, nil).Load()
	c.Check(err, check.ErrorMatches, `(?ms).*cannot use .*SecretKey.*and.*SecretAccessKey.*in z1111.Volumes.z1111-nyw5e-aaaaaaaaaaaaaaa.DriverParameters.*`)
}

func (s *LoadSuite) TestDeprecatedNodeProfilesToServices(c *check.C) {
	hostname, err := os.Hostname()
	c.Assert(err, check.IsNil)
	checkEquivalent(c, `
Clusters:
 z1111:
  NodeProfiles:
   "*":
    arvados-controller:
     listen: ":9004"
   `+hostname+`:
    arvados-api-server:
     listen: ":8000"
   dispatch-host:
    arvados-dispatch-cloud:
     listen: ":9006"
`, `
Clusters:
 z1111:
  Services:
   RailsAPI:
    InternalURLs:
     "http://localhost:8000": {}
   Controller:
    InternalURLs:
     "http://localhost:9004": {}
   DispatchCloud:
    InternalURLs:
     "http://dispatch-host:9006": {}
  NodeProfiles:
   "*":
    arvados-controller:
     listen: ":9004"
   `+hostname+`:
    arvados-api-server:
     listen: ":8000"
   dispatch-host:
    arvados-dispatch-cloud:
     listen: ":9006"
`)
}

func (s *LoadSuite) TestDeprecatedLoginBackend(c *check.C) {
	checkEquivalent(c, `
Clusters:
 z1111:
  Login:
   GoogleClientID: aaaa
   GoogleClientSecret: bbbb
   GoogleAlternateEmailAddresses: true
`, `
Clusters:
 z1111:
  Login:
   Google:
    Enable: true
    ClientID: aaaa
    ClientSecret: bbbb
    AlternateEmailAddresses: true
`)
	checkEquivalent(c, `
Clusters:
 z1111:
  Login:
   ProviderAppID: aaaa
   ProviderAppSecret: bbbb
`, `
Clusters:
 z1111:
  Login:
   SSO:
    Enable: true
    ProviderAppID: aaaa
    ProviderAppSecret: bbbb
`)
}

func (s *LoadSuite) TestLegacyKeepWebConfig(c *check.C) {
	content := []byte(`
{
	"Client": {
		"Scheme": "",
		"APIHost": "example.com",
		"AuthToken": "abcdefg",
	},
	"Listen": ":80",
	"AnonymousTokens": [
		"anonusertoken"
	],
	"AttachmentOnlyHost": "download.example.com",
	"TrustAllContent": true,
	"Cache": {
		"TTL": "1m",
		"UUIDTTL": "1s",
		"MaxCollectionEntries": 42,
		"MaxCollectionBytes": 1234567890,
		"MaxUUIDEntries": 100
	},
	"ManagementToken": "xyzzy"
}
`)
	cluster, err := testLoadLegacyConfig(content, "-legacy-keepweb-config", c)
	c.Assert(err, check.IsNil)

	c.Check(cluster.Services.Controller.ExternalURL, check.Equals, arvados.URL{Scheme: "https", Host: "example.com", Path: "/"})
	c.Check(cluster.SystemRootToken, check.Equals, "abcdefg")

	c.Check(cluster.Collections.WebDAVCache.TTL, check.Equals, arvados.Duration(60*time.Second))
	c.Check(cluster.Collections.WebDAVCache.UUIDTTL, check.Equals, arvados.Duration(time.Second))
	c.Check(cluster.Collections.WebDAVCache.MaxCollectionEntries, check.Equals, 42)
	c.Check(cluster.Collections.WebDAVCache.MaxCollectionBytes, check.Equals, int64(1234567890))
	c.Check(cluster.Collections.WebDAVCache.MaxUUIDEntries, check.Equals, 100)

	c.Check(cluster.Services.WebDAVDownload.ExternalURL, check.Equals, arvados.URL{Host: "download.example.com", Path: "/"})
	c.Check(cluster.Services.WebDAVDownload.InternalURLs[arvados.URL{Host: ":80"}], check.NotNil)
	c.Check(cluster.Services.WebDAV.InternalURLs[arvados.URL{Host: ":80"}], check.NotNil)

	c.Check(cluster.Collections.TrustAllContent, check.Equals, true)
	c.Check(cluster.Users.AnonymousUserToken, check.Equals, "anonusertoken")
	c.Check(cluster.ManagementToken, check.Equals, "xyzzy")
}

// Tests fix for https://dev.arvados.org/issues/15642
func (s *LoadSuite) TestLegacyKeepWebConfigDoesntDisableMissingItems(c *check.C) {
	content := []byte(`
{
	"Client": {
		"Scheme": "",
		"APIHost": "example.com",
		"AuthToken": "abcdefg",
	}
}
`)
	cluster, err := testLoadLegacyConfig(content, "-legacy-keepweb-config", c)
	c.Assert(err, check.IsNil)
	// The resulting ManagementToken should be the one set up on the test server.
	c.Check(cluster.ManagementToken, check.Equals, TestServerManagementToken)
}

func (s *LoadSuite) TestLegacyKeepproxyConfig(c *check.C) {
	f := "-legacy-keepproxy-config"
	content := []byte(fmtKeepproxyConfig("", true))
	cluster, err := testLoadLegacyConfig(content, f, c)

	c.Assert(err, check.IsNil)
	c.Assert(cluster, check.NotNil)
	c.Check(cluster.Services.Controller.ExternalURL, check.Equals, arvados.URL{Scheme: "https", Host: "example.com", Path: "/"})
	c.Check(cluster.SystemRootToken, check.Equals, "abcdefg")
	c.Check(cluster.ManagementToken, check.Equals, "xyzzy")
	c.Check(cluster.Services.Keepproxy.InternalURLs[arvados.URL{Host: ":80"}], check.Equals, arvados.ServiceInstance{})
	c.Check(cluster.Collections.DefaultReplication, check.Equals, 0)
	c.Check(cluster.API.KeepServiceRequestTimeout.String(), check.Equals, "15s")
	c.Check(cluster.SystemLogs.LogLevel, check.Equals, "debug")

	content = []byte(fmtKeepproxyConfig("", false))
	cluster, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.IsNil)
	c.Check(cluster.SystemLogs.LogLevel, check.Equals, "info")

	content = []byte(fmtKeepproxyConfig(`"DisableGet": true,`, true))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.NotNil)

	content = []byte(fmtKeepproxyConfig(`"DisablePut": true,`, true))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.NotNil)

	content = []byte(fmtKeepproxyConfig(`"PIDFile": "test",`, true))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.NotNil)

	content = []byte(fmtKeepproxyConfig(`"DisableGet": false, "DisablePut": false, "PIDFile": "",`, true))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.IsNil)
}

func fmtKeepproxyConfig(param string, debugLog bool) string {
	return fmt.Sprintf(`
{
	"Client": {
		"Scheme": "",
		"APIHost": "example.com",
		"AuthToken": "abcdefg",
		"Insecure": false
	},
	"Listen": ":80",
	"DefaultReplicas": 0,
	"Timeout": "15s",
	"Debug": %t,
	%s
	"ManagementToken": "xyzzy"
}
`, debugLog, param)
}

func (s *LoadSuite) TestLegacyArvGitHttpdConfig(c *check.C) {
	content := []byte(`
{
	"Client": {
		"Scheme": "",
		"APIHost": "example.com",
		"AuthToken": "abcdefg",
	},
	"Listen": ":9000",
	"GitCommand": "/test/git",
	"GitoliteHome": "/test/gitolite",
	"RepoRoot": "/test/reporoot",
	"ManagementToken": "xyzzy"
}
`)
	f := "-legacy-git-httpd-config"
	cluster, err := testLoadLegacyConfig(content, f, c)

	c.Assert(err, check.IsNil)
	c.Assert(cluster, check.NotNil)
	c.Check(cluster.Services.Controller.ExternalURL, check.Equals, arvados.URL{Scheme: "https", Host: "example.com", Path: "/"})
	c.Check(cluster.SystemRootToken, check.Equals, "abcdefg")
	c.Check(cluster.ManagementToken, check.Equals, "xyzzy")
	c.Check(cluster.Git.GitCommand, check.Equals, "/test/git")
	c.Check(cluster.Git.GitoliteHome, check.Equals, "/test/gitolite")
	c.Check(cluster.Git.Repositories, check.Equals, "/test/reporoot")
	c.Check(cluster.Services.Keepproxy.InternalURLs[arvados.URL{Host: ":9000"}], check.Equals, arvados.ServiceInstance{})
}

// Tests fix for https://dev.arvados.org/issues/15642
func (s *LoadSuite) TestLegacyArvGitHttpdConfigDoesntDisableMissingItems(c *check.C) {
	content := []byte(`
{
	"Client": {
		"Scheme": "",
		"APIHost": "example.com",
		"AuthToken": "abcdefg",
	}
}
`)
	cluster, err := testLoadLegacyConfig(content, "-legacy-git-httpd-config", c)
	c.Assert(err, check.IsNil)
	// The resulting ManagementToken should be the one set up on the test server.
	c.Check(cluster.ManagementToken, check.Equals, TestServerManagementToken)
}

func (s *LoadSuite) TestLegacyKeepBalanceConfig(c *check.C) {
	f := "-legacy-keepbalance-config"
	content := []byte(fmtKeepBalanceConfig(""))
	cluster, err := testLoadLegacyConfig(content, f, c)

	c.Assert(err, check.IsNil)
	c.Assert(cluster, check.NotNil)
	c.Check(cluster.ManagementToken, check.Equals, "xyzzy")
	c.Check(cluster.Services.Keepbalance.InternalURLs[arvados.URL{Host: ":80"}], check.Equals, arvados.ServiceInstance{})
	c.Check(cluster.Collections.BalanceCollectionBuffers, check.Equals, 1000)
	c.Check(cluster.Collections.BalanceCollectionBatch, check.Equals, 100000)
	c.Check(cluster.Collections.BalancePeriod.String(), check.Equals, "10m")
	c.Check(cluster.Collections.BlobMissingReport, check.Equals, "testfile")
	c.Check(cluster.API.KeepServiceRequestTimeout.String(), check.Equals, "30m")

	content = []byte(fmtKeepBalanceConfig(`"KeepServiceTypes":["disk"],`))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.IsNil)

	content = []byte(fmtKeepBalanceConfig(`"KeepServiceTypes":[],`))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.IsNil)

	content = []byte(fmtKeepBalanceConfig(`"KeepServiceTypes":["proxy"],`))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.NotNil)

	content = []byte(fmtKeepBalanceConfig(`"KeepServiceTypes":["disk", "proxy"],`))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.NotNil)

	content = []byte(fmtKeepBalanceConfig(`"KeepServiceList":{},`))
	_, err = testLoadLegacyConfig(content, f, c)
	c.Check(err, check.NotNil)
}

func fmtKeepBalanceConfig(param string) string {
	return fmt.Sprintf(`
{
	"Client": {
		"Scheme": "",
		"APIHost": "example.com",
		"AuthToken": "abcdefg",
		"Insecure": false
	},
	"Listen": ":80",
	%s
	"RunPeriod": "10m",
	"CollectionBatchSize": 100000,
	"CollectionBuffers": 1000,
	"RequestTimeout": "30m",
	"ManagementToken": "xyzzy",
	"LostBlocksFile": "testfile"
}
`, param)
}
