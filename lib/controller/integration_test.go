// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"context"
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
	booter        boot.Booter
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
      # LoginCluster: z1111
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
			booter: boot.Booter{
				SourcePath:           filepath.Join(cwd, "..", ".."),
				LibPath:              filepath.Join(cwd, "..", "..", "tmp"),
				ClusterType:          "test",
				ListenHost:           "127.0.0." + id[3:],
				ControllerAddr:       ":0",
				OwnTemporaryDatabase: true,
				Stderr:               &service.LogPrefixer{Writer: ctxlog.LogWriter(c.Log), Prefix: []byte("[" + id + "] ")},
			},
			config: *cfg,
		}
		s.testClusters[id].booter.Start(context.Background(), &s.testClusters[id].config)
	}
	for _, tc := range s.testClusters {
		au, ok := tc.booter.WaitReady()
		c.Assert(ok, check.Equals, true)
		u := url.URL(*au)
		tc.controllerURL = &u
	}
}

func (s *IntegrationSuite) TearDownSuite(c *check.C) {
	for _, c := range s.testClusters {
		c.booter.Stop()
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

func (s *IntegrationSuite) userClients(c *check.C, conn *rpc.Conn, rootctx context.Context, clusterID string, activate bool) (context.Context, *arvados.Client, *keepclient.KeepClient) {
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
	c.Logf("userToken: %q", userToken)
	ctx, ac, kc := s.clientsWithToken(clusterID, userToken)
	user, err := conn.UserGetCurrent(ctx, arvados.GetOptions{})
	if err != nil {
		panic(err)
	}
	_, err = conn.UserSetup(rootctx, arvados.UserSetupOptions{UUID: user.UUID})
	if err != nil {
		panic(err)
	}
	_, err = conn.UserActivate(rootctx, arvados.UserActivateOptions{UUID: user.UUID})
	if err != nil {
		panic(err)
	}
	user, err = conn.UserGetCurrent(ctx, arvados.GetOptions{})
	if err != nil {
		panic(err)
	}
	c.Logf("user: %#v", user)
	if !user.IsActive {
		c.Fatal("failed to activate user")
	}
	return ctx, ac, kc
}

func (s *IntegrationSuite) rootClients(clusterID string) (context.Context, *arvados.Client, *keepclient.KeepClient) {
	return s.clientsWithToken(clusterID, s.testClusters[clusterID].config.Clusters[clusterID].SystemRootToken)
}

func (s *IntegrationSuite) TestLoopDetection(c *check.C) {
	conn1 := s.conn("z1111")
	rootctx1, _, _ := s.rootClients("z1111")
	conn3 := s.conn("z3333")
	// rootctx3, _, _ := s.rootClients("z3333")

	userctx1, ac1, kc1 := s.userClients(c, conn1, rootctx1, "z1111", true)
	_, err := conn1.CollectionGet(userctx1, arvados.GetOptions{UUID: "1f4b0bc7583c2a7f9102c395f4ffc5e3+45"})
	c.Assert(err, check.ErrorMatches, `.*404 Not Found.*`)

	var coll1 arvados.Collection
	fs1, err := coll1.FileSystem(ac1, kc1)
	if err != nil {
		c.Error(err)
	}
	f, err := fs1.OpenFile("foo", os.O_CREATE|os.O_RDWR, 0777)
	f.Write([]byte("foo"))
	f.Close()
	mtxt, err := fs1.MarshalManifest(".")
	coll1, err = conn1.CollectionCreate(userctx1, arvados.CreateOptions{Attrs: map[string]interface{}{
		"manifest_text": mtxt,
	}})
	c.Assert(err, check.IsNil)
	coll, err := conn3.CollectionGet(userctx1, arvados.GetOptions{UUID: "1f4b0bc7583c2a7f9102c395f4ffc5e3+45"})
	c.Check(err, check.IsNil)
	c.Check(coll.PortableDataHash, check.Equals, "1f4b0bc7583c2a7f9102c395f4ffc5e3+45")
}
