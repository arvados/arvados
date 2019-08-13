// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"flag"
	"io/ioutil"
	"os"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

func (s *LoadSuite) TestDeprecatedNodeProfilesToServices(c *check.C) {
	hostname, err := os.Hostname()
	c.Assert(err, check.IsNil)
	s.checkEquivalent(c, `
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
		"MaxPermissionEntries": 100,
		"MaxUUIDEntries": 100
	},
	"ManagementToken": "xyzzy"
}
`)
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		c.Error(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		c.Error(err)
	}
	if err := tmpfile.Close(); err != nil {
		c.Error(err)
	}
	flags := flag.NewFlagSet("keep-web", flag.ExitOnError)
	ldr := testLoader(c, "Clusters: {zzzzz: {}}", nil)
	ldr.SetupFlags(flags)
	args := ldr.MungeLegacyConfigArgs(ldr.Logger, []string{"-config", tmpfile.Name()}, "-legacy-keepweb-config")
	flags.Parse(args)
	cfg, err := ldr.Load()
	if err != nil {
		c.Error(err)
	}
	c.Check(cfg, check.NotNil)
	cluster, err := cfg.GetCluster("")
	if err != nil {
		c.Error(err)
	}
	c.Check(cluster, check.NotNil)

	c.Check(cluster.Services.Controller.ExternalURL, check.Equals, arvados.URL{Scheme: "https", Host: "example.com"})
	c.Check(cluster.SystemRootToken, check.Equals, "abcdefg")

	c.Check(cluster.Collections.WebDAVCache.TTL, check.Equals, arvados.Duration(60*time.Second))
	c.Check(cluster.Collections.WebDAVCache.UUIDTTL, check.Equals, arvados.Duration(time.Second))
	c.Check(cluster.Collections.WebDAVCache.MaxCollectionEntries, check.Equals, 42)
	c.Check(cluster.Collections.WebDAVCache.MaxCollectionBytes, check.Equals, int64(1234567890))
	c.Check(cluster.Collections.WebDAVCache.MaxPermissionEntries, check.Equals, 100)
	c.Check(cluster.Collections.WebDAVCache.MaxUUIDEntries, check.Equals, 100)

	c.Check(cluster.Services.WebDAVDownload.ExternalURL, check.Equals, arvados.URL{Host: "download.example.com"})
	c.Check(cluster.Services.WebDAVDownload.InternalURLs[arvados.URL{Host: ":80"}], check.NotNil)
	c.Check(cluster.Services.WebDAV.InternalURLs[arvados.URL{Host: ":80"}], check.NotNil)

	c.Check(cluster.Collections.TrustAllContent, check.Equals, true)
	c.Check(cluster.Users.AnonymousUserToken, check.Equals, "anonusertoken")
	c.Check(cluster.ManagementToken, check.Equals, "xyzzy")
}
