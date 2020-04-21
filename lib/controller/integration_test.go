// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"context"
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"git.arvados.org/arvados.git/lib/boot"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&IntegrationSuite{})

type testCluster struct {
	super         boot.Supervisor
	config        arvados.Config
	controllerURL *url.URL
}

type IntegrationSuite struct {
	testClusters map[string]*testCluster
}

func (s *IntegrationSuite) SetUpSuite(c *check.C) {
	if forceLegacyAPI14 {
		c.Skip("heavy integration tests don't run with forceLegacyAPI14")
		return
	}

	cwd, _ := os.Getwd()
	s.testClusters = map[string]*testCluster{
		"z1111": nil,
		"z2222": nil,
		"z3333": nil,
	}
	hostport := map[string]string{}
	for id := range s.testClusters {
		hostport[id] = func() string {
			// TODO: Instead of expecting random ports on
			// 127.0.0.11, 22, 33 to be race-safe, try
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
    Login:
      LoginCluster: z1111
    SystemLogs:
      Format: text
    RemoteClusters:
      z1111:
        Host: ` + hostport["z1111"] + `
        Scheme: https
        Insecure: true
        Proxy: true
        ActivateUsers: true
      z2222:
        Host: ` + hostport["z2222"] + `
        Scheme: https
        Insecure: true
        Proxy: true
        ActivateUsers: true
      z3333:
        Host: ` + hostport["z3333"] + `
        Scheme: https
        Insecure: true
        Proxy: true
        ActivateUsers: true
`
		loader := config.NewLoader(bytes.NewBufferString(yaml), ctxlog.TestLogger(c))
		loader.Path = "-"
		loader.SkipLegacy = true
		loader.SkipAPICalls = true
		cfg, err := loader.Load()
		c.Assert(err, check.IsNil)
		s.testClusters[id] = &testCluster{
			super: boot.Supervisor{
				SourcePath:           filepath.Join(cwd, "..", ".."),
				ClusterType:          "test",
				ListenHost:           "127.0.0." + id[3:],
				ControllerAddr:       ":0",
				OwnTemporaryDatabase: true,
				Stderr:               &service.LogPrefixer{Writer: ctxlog.LogWriter(c.Log), Prefix: []byte("[" + id + "] ")},
			},
			config: *cfg,
		}
		s.testClusters[id].super.Start(context.Background(), &s.testClusters[id].config, "-")
	}
	for _, tc := range s.testClusters {
		au, ok := tc.super.WaitReady()
		c.Assert(ok, check.Equals, true)
		u := url.URL(*au)
		tc.controllerURL = &u
	}
}

func (s *IntegrationSuite) TearDownSuite(c *check.C) {
	for _, c := range s.testClusters {
		c.super.Stop()
	}
}

func (s *IntegrationSuite) conn(clusterID string) *rpc.Conn {
	return rpc.NewConn(clusterID, s.testClusters[clusterID].controllerURL, true, rpc.PassthroughTokenProvider)
}

func (s *IntegrationSuite) clientsWithToken(clusterID string, token string) (context.Context, *arvados.Client, *keepclient.KeepClient) {
	cl := s.testClusters[clusterID].config.Clusters[clusterID]
	ctx := auth.NewContext(context.Background(), auth.NewCredentials(token))
	ac, err := arvados.NewClientFromConfig(&cl)
	if err != nil {
		panic(err)
	}
	ac.AuthToken = token
	arv, err := arvadosclient.New(ac)
	if err != nil {
		panic(err)
	}
	kc := keepclient.New(arv)
	return ctx, ac, kc
}

func (s *IntegrationSuite) userClients(rootctx context.Context, c *check.C, conn *rpc.Conn, clusterID string, activate bool) (context.Context, *arvados.Client, *keepclient.KeepClient) {
	login, err := conn.UserSessionCreate(rootctx, rpc.UserSessionCreateOptions{
		ReturnTo: ",https://example.com",
		AuthInfo: rpc.UserSessionAuthInfo{
			Email:     "user@example.com",
			FirstName: "Example",
			LastName:  "User",
			Username:  "example",
		},
	})
	c.Assert(err, check.IsNil)
	redirURL, err := url.Parse(login.RedirectLocation)
	c.Assert(err, check.IsNil)
	userToken := redirURL.Query().Get("api_token")
	c.Logf("user token: %q", userToken)
	ctx, ac, kc := s.clientsWithToken(clusterID, userToken)
	user, err := conn.UserGetCurrent(ctx, arvados.GetOptions{})
	c.Assert(err, check.IsNil)
	_, err = conn.UserSetup(rootctx, arvados.UserSetupOptions{UUID: user.UUID})
	c.Assert(err, check.IsNil)
	if activate {
		_, err = conn.UserActivate(rootctx, arvados.UserActivateOptions{UUID: user.UUID})
		c.Assert(err, check.IsNil)
		user, err = conn.UserGetCurrent(ctx, arvados.GetOptions{})
		c.Assert(err, check.IsNil)
		c.Logf("user UUID: %q", user.UUID)
		if !user.IsActive {
			c.Fatalf("failed to activate user -- %#v", user)
		}
	}
	return ctx, ac, kc
}

func (s *IntegrationSuite) rootClients(clusterID string) (context.Context, *arvados.Client, *keepclient.KeepClient) {
	return s.clientsWithToken(clusterID, s.testClusters[clusterID].config.Clusters[clusterID].SystemRootToken)
}

func (s *IntegrationSuite) TestGetCollectionByPDH(c *check.C) {
	conn1 := s.conn("z1111")
	rootctx1, _, _ := s.rootClients("z1111")
	conn3 := s.conn("z3333")
	userctx1, ac1, kc1 := s.userClients(rootctx1, c, conn1, "z1111", true)

	// Create the collection to find its PDH (but don't save it
	// anywhere yet)
	var coll1 arvados.Collection
	fs1, err := coll1.FileSystem(ac1, kc1)
	c.Assert(err, check.IsNil)
	f, err := fs1.OpenFile("test.txt", os.O_CREATE|os.O_RDWR, 0777)
	c.Assert(err, check.IsNil)
	_, err = io.WriteString(f, "IntegrationSuite.TestGetCollectionByPDH")
	c.Assert(err, check.IsNil)
	err = f.Close()
	c.Assert(err, check.IsNil)
	mtxt, err := fs1.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	pdh := arvados.PortableDataHash(mtxt)

	// Looking up the PDH before saving returns 404 if cycle
	// detection is working.
	_, err = conn1.CollectionGet(userctx1, arvados.GetOptions{UUID: pdh})
	c.Assert(err, check.ErrorMatches, `.*404 Not Found.*`)

	// Save the collection on cluster z1111.
	coll1, err = conn1.CollectionCreate(userctx1, arvados.CreateOptions{Attrs: map[string]interface{}{
		"manifest_text": mtxt,
	}})
	c.Assert(err, check.IsNil)

	// Retrieve the collection from cluster z3333.
	coll, err := conn3.CollectionGet(userctx1, arvados.GetOptions{UUID: pdh})
	c.Check(err, check.IsNil)
	c.Check(coll.PortableDataHash, check.Equals, pdh)
}

// Test for bug #16263
func (s *IntegrationSuite) TestListUsers(c *check.C) {
	rootctx1, _, _ := s.rootClients("z1111")
	conn1 := s.conn("z1111")
	conn3 := s.conn("z3333")

	// Make sure LoginCluster is properly configured
	for cls := range s.testClusters {
		c.Check(
			s.testClusters[cls].config.Clusters[cls].Login.LoginCluster,
			check.Equals, "z1111",
			check.Commentf("incorrect LoginCluster config on cluster %q", cls))
	}
	// Make sure z1111 has users with NULL usernames
	lst, err := conn1.UserList(rootctx1, arvados.ListOptions{Limit: -1})
	nullUsername := false
	c.Assert(err, check.IsNil)
	c.Assert(len(lst.Items), check.Not(check.Equals), 0)
	for _, user := range lst.Items {
		if user.Username == "" {
			nullUsername = true
		}
	}
	c.Assert(nullUsername, check.Equals, true)
	// Ask for the user list on z3333 using z1111's system root token
	_, err = conn3.UserList(rootctx1, arvados.ListOptions{Limit: -1})
	c.Assert(err, check.IsNil, check.Commentf("getting user list: %q", err))
}

// Test for bug #16263
func (s *IntegrationSuite) TestListUsersWithMaxLimit(c *check.C) {
	rootctx1, _, _ := s.rootClients("z1111")
	conn3 := s.conn("z3333")
	maxLimit := int64(math.MaxInt64)

	// Make sure LoginCluster is properly configured
	for cls := range s.testClusters {
		c.Check(
			s.testClusters[cls].config.Clusters[cls].Login.LoginCluster,
			check.Equals, "z1111",
			check.Commentf("incorrect LoginCluster config on cluster %q", cls))
	}

	// Ask for the user list on z3333 using z1111's system root token and
	// limit: max int64 value.
	_, err := conn3.UserList(rootctx1, arvados.ListOptions{Limit: maxLimit})
	c.Assert(err, check.IsNil, check.Commentf("getting user list: %q", err))
}
