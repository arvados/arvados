// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"net"
	"os"
	"path/filepath"

	"git.arvados.org/arvados.git/lib/boot"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&FederationSuite{})

var origAPIHost, origAPIToken string

type FederationSuite struct {
	testClusters map[string]*boot.TestCluster
	oidcprovider *arvadostest.OIDCProvider
}

func (s *FederationSuite) SetUpSuite(c *check.C) {
	origAPIHost = os.Getenv("ARVADOS_API_HOST")
	origAPIToken = os.Getenv("ARVADOS_API_TOKEN")

	cwd, _ := os.Getwd()

	s.oidcprovider = arvadostest.NewOIDCProvider(c)
	s.oidcprovider.AuthEmail = "user@example.com"
	s.oidcprovider.AuthEmailVerified = true
	s.oidcprovider.AuthName = "Example User"
	s.oidcprovider.ValidClientID = "clientid"
	s.oidcprovider.ValidClientSecret = "clientsecret"

	s.testClusters = map[string]*boot.TestCluster{
		"z1111": nil,
		"z2222": nil,
	}
	hostport := map[string]string{}
	for id := range s.testClusters {
		hostport[id] = func() string {
			// TODO: Instead of expecting random ports on
			// 127.0.0.11, 22 to be race-safe, try
			// different 127.x.y.z until finding one that
			// isn't in use.
			ln, err := net.Listen("tcp", ":0")
			c.Assert(err, check.IsNil)
			ln.Close()
			_, port, err := net.SplitHostPort(ln.Addr().String())
			c.Assert(err, check.IsNil)
			return "127.0.0." + id[3:] + ":" + port
		}()
	}
	for id := range s.testClusters {
		yaml := `Clusters:
  ` + id + `:
    Services:
      Controller:
        ExternalURL: https://` + hostport[id] + `
    TLS:
      Insecure: true
    SystemLogs:
      Format: text
    RemoteClusters:
      z1111:
        Host: ` + hostport["z1111"] + `
        Scheme: https
        Insecure: true
        Proxy: true
        ActivateUsers: true
`
		if id != "z2222" {
			yaml += `      z2222:
        Host: ` + hostport["z2222"] + `
        Scheme: https
        Insecure: true
        Proxy: true
        ActivateUsers: true
`
		}
		if id == "z1111" {
			yaml += `
    Login:
      LoginCluster: z1111
      OpenIDConnect:
        Enable: true
        Issuer: ` + s.oidcprovider.Issuer.URL + `
        ClientID: ` + s.oidcprovider.ValidClientID + `
        ClientSecret: ` + s.oidcprovider.ValidClientSecret + `
        EmailClaim: email
        EmailVerifiedClaim: email_verified
`
		} else {
			yaml += `
    Login:
      LoginCluster: z1111
`
		}

		loader := config.NewLoader(bytes.NewBufferString(yaml), ctxlog.TestLogger(c))
		loader.Path = "-"
		loader.SkipLegacy = true
		loader.SkipAPICalls = true
		cfg, err := loader.Load()
		c.Assert(err, check.IsNil)
		tc := boot.NewTestCluster(
			filepath.Join(cwd, "..", ".."),
			id, cfg, "127.0.0."+id[3:], c.Log)
		tc.Super.NoWorkbench1 = true
		tc.Start()
		s.testClusters[id] = tc
	}
	for _, tc := range s.testClusters {
		ok := tc.WaitReady()
		c.Assert(ok, check.Equals, true)
	}

	// Activate user, make it admin.
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	userctx1, _, _, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)
	user1, err := conn1.UserGetCurrent(userctx1, arvados.GetOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(user1.IsAdmin, check.Equals, false)
	user1, err = conn1.UserUpdate(rootctx1, arvados.UpdateOptions{
		UUID: user1.UUID,
		Attrs: map[string]interface{}{
			"is_admin": true,
		},
	})
	c.Assert(err, check.IsNil)
	c.Assert(user1.IsAdmin, check.Equals, true)
}

func (s *FederationSuite) TearDownSuite(c *check.C) {
	for _, c := range s.testClusters {
		c.Super.Stop()
	}
	_ = os.Setenv("ARVADOS_API_HOST", origAPIHost)
	_ = os.Setenv("ARVADOS_API_TOKEN", origAPIToken)
}

func (s *FederationSuite) TestGroupSyncingOnFederatedCluster(c *check.C) {
	// Get admin user's V2 token
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	userctx1, _, _, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)
	user1Auth, err := conn1.APIClientAuthorizationCurrent(userctx1, arvados.GetOptions{})
	c.Check(err, check.IsNil)
	userV2Token := user1Auth.TokenV2()

	// Get federated admin clients on z2222 to set up environment
	conn2 := s.testClusters["z2222"].Conn()
	userctx2, userac2, _ := s.testClusters["z2222"].ClientsWithToken(userV2Token)
	user2, err := conn2.UserGetCurrent(userctx2, arvados.GetOptions{})
	c.Check(err, check.IsNil)
	c.Check(user2.IsAdmin, check.Equals, true)

	// Set up environment for sync-groups using admin user credentials on z2222
	err = os.Setenv("ARVADOS_API_HOST", userac2.APIHost)
	c.Assert(err, check.IsNil)
	err = os.Setenv("ARVADOS_API_TOKEN", userac2.AuthToken)
	c.Assert(err, check.IsNil)

	// Check that no parent group is created
	gl := arvados.GroupList{}
	params := arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "owner_uuid",
			Operator: "=",
			Operand:  s.testClusters["z2222"].ClusterID + "-tpzed-000000000000000",
		}, {
			Attr:     "name",
			Operator: "=",
			Operand:  "Externally synchronized groups",
		}},
	}
	err = userac2.RequestAndDecode(&gl, "GET", "/arvados/v1/groups", nil, params)
	c.Assert(err, check.IsNil)
	c.Assert(gl.ItemsAvailable, check.Equals, 0)

	// Set up config, confirm that the parent group was created
	os.Args = []string{"cmd", "somefile.csv"}
	config, err := GetConfig()
	c.Assert(err, check.IsNil)
	userac2.RequestAndDecode(&gl, "GET", "/arvados/v1/groups", nil, params)
	c.Assert(gl.ItemsAvailable, check.Equals, 1)

	// Run the tool with custom config
	data := [][]string{
		{"TestGroup1", user2.Email},
	}
	tmpfile, err := MakeTempCSVFile(data)
	c.Assert(err, check.IsNil)
	defer os.Remove(tmpfile.Name()) // clean up
	config.Path = tmpfile.Name()
	err = doMain(&config)
	c.Assert(err, check.IsNil)
	// Check the group was created correctly, and has the user as a member
	groupUUID, err := RemoteGroupExists(&config, "TestGroup1")
	c.Assert(err, check.IsNil)
	c.Assert(groupUUID, check.Not(check.Equals), "")
	c.Assert(GroupMembershipExists(config.Client, user2.UUID, groupUUID, "can_write"), check.Equals, true)
}
