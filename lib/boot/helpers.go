// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"net/url"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"gopkg.in/check.v1"
)

// ClientsWithToken returns Context, Arvados.Client and keepclient structs
// initialized to connect to the cluster with the supplied Arvados token.
func (super *Supervisor) ClientsWithToken(clusterID, token string) (context.Context, *arvados.Client, *keepclient.KeepClient) {
	cl := super.cluster
	if super.children != nil {
		cl = super.children[clusterID].cluster
	} else if clusterID != cl.ClusterID {
		panic("bad clusterID " + clusterID)
	}
	ctx := auth.NewContext(super.ctx, auth.NewCredentials(token))
	ac, err := arvados.NewClientFromConfig(cl)
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
func (super *Supervisor) UserClients(clusterID string, rootctx context.Context, c *check.C, conn *rpc.Conn, authEmail string, activate bool) (context.Context, *arvados.Client, *keepclient.KeepClient, arvados.User) {
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
	ctx, ac, kc := super.ClientsWithToken(clusterID, userToken)
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
func (super *Supervisor) RootClients(clusterID string) (context.Context, *arvados.Client, *keepclient.KeepClient) {
	return super.ClientsWithToken(clusterID, super.Cluster(clusterID).SystemRootToken)
}

// AnonymousClients returns Context, arvados.Client and keepclient structs initialized
// to communicate with the cluster as the anonymous user.
func (super *Supervisor) AnonymousClients(clusterID string) (context.Context, *arvados.Client, *keepclient.KeepClient) {
	return super.ClientsWithToken(clusterID, super.Cluster(clusterID).Users.AnonymousUserToken)
}

// Conn gets rpc connection struct initialized to communicate with the
// specified cluster.
func (super *Supervisor) Conn(clusterID string) *rpc.Conn {
	controllerURL := url.URL(super.Cluster(clusterID).Services.Controller.ExternalURL)
	return rpc.NewConn(clusterID, &controllerURL, true, rpc.PassthroughTokenProvider)
}
