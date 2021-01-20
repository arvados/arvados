// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"net/url"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"gopkg.in/check.v1"
)

// TestCluster stores a working test cluster data
type TestCluster struct {
	Super         Supervisor
	Config        arvados.Config
	ControllerURL *url.URL
	ClusterID     string
}

type logger struct {
	loggerfunc func(...interface{})
}

func (l logger) Log(args ...interface{}) {
	l.loggerfunc(args)
}

// NewTestCluster loads the provided configuration, and sets up a test cluster
// ready for being started.
func NewTestCluster(srcPath, clusterID string, cfg *arvados.Config, listenHost string, logWriter func(...interface{})) *TestCluster {
	return &TestCluster{
		Super: Supervisor{
			SourcePath:           srcPath,
			ClusterType:          "test",
			ListenHost:           listenHost,
			ControllerAddr:       ":0",
			OwnTemporaryDatabase: true,
			Stderr: &service.LogPrefixer{
				Writer: ctxlog.LogWriter(logWriter),
				Prefix: []byte("[" + clusterID + "] ")},
		},
		Config:    *cfg,
		ClusterID: clusterID,
	}
}

// Start the test cluster.
func (tc *TestCluster) Start() {
	tc.Super.Start(context.Background(), &tc.Config, "-")
}

// WaitReady waits for all components to report healthy, and finishes setting
// up the TestCluster struct.
func (tc *TestCluster) WaitReady() bool {
	au, ok := tc.Super.WaitReady()
	if !ok {
		return ok
	}
	u := url.URL(*au)
	tc.ControllerURL = &u
	return ok
}

// ClientsWithToken returns Context, Arvados.Client and keepclient structs
// initialized to connect to the cluster with the supplied Arvados token.
func (tc *TestCluster) ClientsWithToken(token string) (context.Context, *arvados.Client, *keepclient.KeepClient) {
	cl := tc.Config.Clusters[tc.ClusterID]
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

// UserClients logs in as a user called "example", get the user's API token,
// initialize clients with the API token, set up the user and
// optionally activate the user.  Return client structs for
// communicating with the cluster on behalf of the 'example' user.
func (tc *TestCluster) UserClients(rootctx context.Context, c *check.C, conn *rpc.Conn, authEmail string, activate bool) (context.Context, *arvados.Client, *keepclient.KeepClient, arvados.User) {
	login, err := conn.UserSessionCreate(rootctx, rpc.UserSessionCreateOptions{
		ReturnTo: ",https://example.com",
		AuthInfo: rpc.UserSessionAuthInfo{
			Email:     authEmail,
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
	ctx, ac, kc := tc.ClientsWithToken(userToken)
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
	return ctx, ac, kc, user
}

// RootClients returns Context, arvados.Client and keepclient structs initialized
// to communicate with the cluster as the system root user.
func (tc *TestCluster) RootClients() (context.Context, *arvados.Client, *keepclient.KeepClient) {
	return tc.ClientsWithToken(tc.Config.Clusters[tc.ClusterID].SystemRootToken)
}

// AnonymousClients returns Context, arvados.Client and keepclient structs initialized
// to communicate with the cluster as the anonymous user.
func (tc *TestCluster) AnonymousClients() (context.Context, *arvados.Client, *keepclient.KeepClient) {
	return tc.ClientsWithToken(tc.Config.Clusters[tc.ClusterID].Users.AnonymousUserToken)
}

// Conn gets rpc connection struct initialized to communicate with the
// specified cluster.
func (tc *TestCluster) Conn() *rpc.Conn {
	return rpc.NewConn(tc.ClusterID, tc.ControllerURL, true, rpc.PassthroughTokenProvider)
}
