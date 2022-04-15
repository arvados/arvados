// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"net"
	"os"
	"time"

	"git.arvados.org/arvados.git/lib/boot"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&FederationSuite{})

var origAPIHost, origAPIToken string

type FederationSuite struct {
	super *boot.Supervisor
}

func (s *FederationSuite) SetUpSuite(c *check.C) {
	origAPIHost = os.Getenv("ARVADOS_API_HOST")
	origAPIToken = os.Getenv("ARVADOS_API_TOKEN")

	hostport := map[string]string{}
	for _, id := range []string{"z1111", "z2222"} {
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
	yaml := "Clusters:\n"
	for id := range hostport {
		yaml += `
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
      PAM:
        Enable: true
`
		} else {
			yaml += `
    Login:
      LoginCluster: z1111
`
		}
	}
	s.super = &boot.Supervisor{
		ClusterType:          "test",
		ConfigYAML:           yaml,
		Stderr:               ctxlog.LogWriter(c.Log),
		NoWorkbench1:         true,
		NoWorkbench2:         true,
		OwnTemporaryDatabase: true,
	}

	// Give up if startup takes longer than 3m
	timeout := time.AfterFunc(3*time.Minute, s.super.Stop)
	defer timeout.Stop()
	s.super.Start(context.Background())
	ok := s.super.WaitReady()
	c.Assert(ok, check.Equals, true)

	// Activate user, make it admin.
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	userctx1, _, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, "admin@example.com", true)
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
	s.super.Stop()
	_ = os.Setenv("ARVADOS_API_HOST", origAPIHost)
	_ = os.Setenv("ARVADOS_API_TOKEN", origAPIToken)
}

func (s *FederationSuite) TestGroupSyncingOnFederatedCluster(c *check.C) {
	// Get admin user's V2 token
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	userctx1, _, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, "admin@example.com", true)
	user1Auth, err := conn1.APIClientAuthorizationCurrent(userctx1, arvados.GetOptions{})
	c.Check(err, check.IsNil)
	userV2Token := user1Auth.TokenV2()

	// Get federated admin clients on z2222 to set up environment
	conn2 := s.super.Conn("z2222")
	userctx2, userac2, _ := s.super.ClientsWithToken("z2222", userV2Token)
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
			Operand:  s.super.Cluster("z2222").ClusterID + "-tpzed-000000000000000",
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
