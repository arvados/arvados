// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net"
	"net/http"
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
		if id != "z3333" {
			yaml += `      z3333:
        Host: ` + hostport["z3333"] + `
        Scheme: https
        Insecure: true
        Proxy: true
        ActivateUsers: true
`
		}

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

// Get a token from the login cluster (z1111), use it to submit a
// container request on z2222.
func (s *IntegrationSuite) TestCreateContainerRequestWithFedToken(c *check.C) {
	conn1 := s.conn("z1111")
	rootctx1, _, _ := s.rootClients("z1111")
	_, ac1, _ := s.userClients(rootctx1, c, conn1, "z1111", true)

	// Use ac2 to get the discovery doc with a blank token, so the
	// SDK doesn't magically pass the z1111 token to z2222 before
	// we're ready to start our test.
	_, ac2, _ := s.clientsWithToken("z2222", "")
	var dd map[string]interface{}
	err := ac2.RequestAndDecode(&dd, "GET", "discovery/v1/apis/arvados/v1/rest", nil, nil)
	c.Assert(err, check.IsNil)

	var (
		body bytes.Buffer
		req  *http.Request
		resp *http.Response
		u    arvados.User
		cr   arvados.ContainerRequest
	)
	json.NewEncoder(&body).Encode(map[string]interface{}{
		"container_request": map[string]interface{}{
			"command":         []string{"echo"},
			"container_image": "d41d8cd98f00b204e9800998ecf8427e+0",
			"cwd":             "/",
			"output_path":     "/",
		},
	})
	ac2.AuthToken = ac1.AuthToken

	c.Log("...post CR with good (but not yet cached) token")
	cr = arvados.ContainerRequest{}
	req, err = http.NewRequest("POST", "https://"+ac2.APIHost+"/arvados/v1/container_requests", bytes.NewReader(body.Bytes()))
	c.Assert(err, check.IsNil)
	req.Header.Set("Content-Type", "application/json")
	err = ac2.DoAndDecode(&cr, req)
	c.Logf("err == %#v", err)

	c.Log("...get user with good token")
	u = arvados.User{}
	req, err = http.NewRequest("GET", "https://"+ac2.APIHost+"/arvados/v1/users/current", nil)
	c.Assert(err, check.IsNil)
	err = ac2.DoAndDecode(&u, req)
	c.Check(err, check.IsNil)
	c.Check(u.UUID, check.Matches, "z1111-tpzed-.*")

	c.Log("...post CR with good cached token")
	cr = arvados.ContainerRequest{}
	req, err = http.NewRequest("POST", "https://"+ac2.APIHost+"/arvados/v1/container_requests", bytes.NewReader(body.Bytes()))
	c.Assert(err, check.IsNil)
	req.Header.Set("Content-Type", "application/json")
	err = ac2.DoAndDecode(&cr, req)
	c.Check(err, check.IsNil)
	c.Check(cr.UUID, check.Matches, "z2222-.*")

	c.Log("...post with good cached token ('OAuth2 ...')")
	cr = arvados.ContainerRequest{}
	req, err = http.NewRequest("POST", "https://"+ac2.APIHost+"/arvados/v1/container_requests", bytes.NewReader(body.Bytes()))
	c.Assert(err, check.IsNil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "OAuth2 "+ac2.AuthToken)
	resp, err = arvados.InsecureHTTPClient.Do(req)
	if c.Check(err, check.IsNil) {
		err = json.NewDecoder(resp.Body).Decode(&cr)
		c.Check(err, check.IsNil)
		c.Check(cr.UUID, check.Matches, "z2222-.*")
	}
}

// Test for bug #16263
func (s *IntegrationSuite) TestListUsers(c *check.C) {
	rootctx1, _, _ := s.rootClients("z1111")
	conn1 := s.conn("z1111")
	conn3 := s.conn("z3333")
	userctx1, _, _ := s.userClients(rootctx1, c, conn1, "z1111", true)

	// Make sure LoginCluster is properly configured
	for cls := range s.testClusters {
		c.Check(
			s.testClusters[cls].config.Clusters[cls].Login.LoginCluster,
			check.Equals, "z1111",
			check.Commentf("incorrect LoginCluster config on cluster %q", cls))
	}
	// Make sure z1111 has users with NULL usernames
	lst, err := conn1.UserList(rootctx1, arvados.ListOptions{
		Limit: math.MaxInt64, // check that large limit works (see #16263)
	})
	nullUsername := false
	c.Assert(err, check.IsNil)
	c.Assert(len(lst.Items), check.Not(check.Equals), 0)
	for _, user := range lst.Items {
		if user.Username == "" {
			nullUsername = true
		}
	}
	c.Assert(nullUsername, check.Equals, true)

	user1, err := conn1.UserGetCurrent(userctx1, arvados.GetOptions{})
	c.Assert(err, check.IsNil)
	c.Check(user1.IsActive, check.Equals, true)

	// Ask for the user list on z3333 using z1111's system root token
	lst, err = conn3.UserList(rootctx1, arvados.ListOptions{Limit: -1})
	c.Assert(err, check.IsNil)
	found := false
	for _, user := range lst.Items {
		if user.UUID == user1.UUID {
			c.Check(user.IsActive, check.Equals, true)
			found = true
			break
		}
	}
	c.Check(found, check.Equals, true)

	// Deactivate user acct on z1111
	_, err = conn1.UserUnsetup(rootctx1, arvados.GetOptions{UUID: user1.UUID})
	c.Assert(err, check.IsNil)

	// Get user list from z3333, check the returned z1111 user is
	// deactivated
	lst, err = conn3.UserList(rootctx1, arvados.ListOptions{Limit: -1})
	c.Assert(err, check.IsNil)
	found = false
	for _, user := range lst.Items {
		if user.UUID == user1.UUID {
			c.Check(user.IsActive, check.Equals, false)
			found = true
			break
		}
	}
	c.Check(found, check.Equals, true)

	// Deactivated user can see is_active==false via "get current
	// user" API
	user1, err = conn3.UserGetCurrent(userctx1, arvados.GetOptions{})
	c.Assert(err, check.IsNil)
	c.Check(user1.IsActive, check.Equals, false)
}
