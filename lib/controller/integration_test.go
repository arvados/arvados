// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/boot"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&IntegrationSuite{})

type IntegrationSuite struct {
	super        *boot.Supervisor
	oidcprovider *arvadostest.OIDCProvider
}

func (s *IntegrationSuite) SetUpSuite(c *check.C) {
	s.oidcprovider = arvadostest.NewOIDCProvider(c)
	s.oidcprovider.AuthEmail = "user@example.com"
	s.oidcprovider.AuthEmailVerified = true
	s.oidcprovider.AuthName = "Example User"
	s.oidcprovider.ValidClientID = "clientid"
	s.oidcprovider.ValidClientSecret = "clientsecret"

	hostport := map[string]string{}
	for _, id := range []string{"z1111", "z2222", "z3333"} {
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
    Containers:
      CloudVMs:
        Enable: true
        Driver: loopback
        BootProbeCommand: "rm -f /var/lock/crunch-run-broken"
        ProbeInterval: 1s
        PollInterval: 5s
        SyncInterval: 10s
        TimeoutIdle: 1s
        TimeoutBooting: 2s
      RuntimeEngine: singularity
      CrunchRunArgumentsList: ["--broken-node-hook", "true"]
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
        AcceptAccessToken: true
        AcceptAccessTokenScope: ""
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
}

func (s *IntegrationSuite) TearDownSuite(c *check.C) {
	if s.super != nil {
		s.super.Stop()
		s.super.Wait()
	}
}

func (s *IntegrationSuite) TestDefaultStorageClassesOnCollections(c *check.C) {
	conn := s.super.Conn("z1111")
	rootctx, _, _ := s.super.RootClients("z1111")
	userctx, _, kc, _ := s.super.UserClients("z1111", rootctx, c, conn, s.oidcprovider.AuthEmail, true)
	c.Assert(len(kc.DefaultStorageClasses) > 0, check.Equals, true)
	coll, err := conn.CollectionCreate(userctx, arvados.CreateOptions{})
	c.Assert(err, check.IsNil)
	c.Assert(coll.StorageClassesDesired, check.DeepEquals, kc.DefaultStorageClasses)
}

func (s *IntegrationSuite) TestGetCollectionByPDH(c *check.C) {
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	conn3 := s.super.Conn("z3333")
	userctx1, ac1, kc1, _ := s.super.UserClients("z1111", rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

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

// Tests bug #18004
func (s *IntegrationSuite) TestRemoteUserAndTokenCacheRace(c *check.C) {
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	rootctx2, _, _ := s.super.RootClients("z2222")
	conn2 := s.super.Conn("z2222")
	userctx1, _, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, "user2@example.com", true)

	var wg1, wg2 sync.WaitGroup
	creqs := 100

	// Make concurrent requests to z2222 with a local token to make sure more
	// than one worker is listening.
	wg1.Add(1)
	for i := 0; i < creqs; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			wg1.Wait()
			_, err := conn2.UserGetCurrent(rootctx2, arvados.GetOptions{})
			c.Check(err, check.IsNil, check.Commentf("warm up phase failed"))
		}()
	}
	wg1.Done()
	wg2.Wait()

	// Real test pass -- use a new remote token than the one used in the warm-up
	// phase.
	wg1.Add(1)
	for i := 0; i < creqs; i++ {
		wg2.Add(1)
		go func() {
			defer wg2.Done()
			wg1.Wait()
			// Retrieve the remote collection from cluster z2222.
			_, err := conn2.UserGetCurrent(userctx1, arvados.GetOptions{})
			c.Check(err, check.IsNil, check.Commentf("testing phase failed"))
		}()
	}
	wg1.Done()
	wg2.Wait()
}

func (s *IntegrationSuite) TestS3WithFederatedToken(c *check.C) {
	if _, err := exec.LookPath("s3cmd"); err != nil {
		c.Skip("s3cmd not in PATH")
		return
	}

	testText := "IntegrationSuite.TestS3WithFederatedToken"

	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	userctx1, ac1, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)
	conn3 := s.super.Conn("z3333")

	createColl := func(clusterID string) arvados.Collection {
		_, ac, kc := s.super.ClientsWithToken(clusterID, ac1.AuthToken)
		var coll arvados.Collection
		fs, err := coll.FileSystem(ac, kc)
		c.Assert(err, check.IsNil)
		f, err := fs.OpenFile("test.txt", os.O_CREATE|os.O_RDWR, 0777)
		c.Assert(err, check.IsNil)
		_, err = io.WriteString(f, testText)
		c.Assert(err, check.IsNil)
		err = f.Close()
		c.Assert(err, check.IsNil)
		mtxt, err := fs.MarshalManifest(".")
		c.Assert(err, check.IsNil)
		coll, err = s.super.Conn(clusterID).CollectionCreate(userctx1, arvados.CreateOptions{Attrs: map[string]interface{}{
			"manifest_text": mtxt,
		}})
		c.Assert(err, check.IsNil)
		return coll
	}

	for _, trial := range []struct {
		clusterID string // create the collection on this cluster (then use z3333 to access it)
		token     string
	}{
		// Try the hardest test first: z3333 hasn't seen
		// z1111's token yet, and we're just passing the
		// opaque secret part, so z3333 has to guess that it
		// belongs to z1111.
		{"z1111", strings.Split(ac1.AuthToken, "/")[2]},
		{"z3333", strings.Split(ac1.AuthToken, "/")[2]},
		{"z1111", strings.Replace(ac1.AuthToken, "/", "_", -1)},
		{"z3333", strings.Replace(ac1.AuthToken, "/", "_", -1)},
	} {
		c.Logf("================ %v", trial)
		coll := createColl(trial.clusterID)

		cfgjson, err := conn3.ConfigGet(userctx1)
		c.Assert(err, check.IsNil)
		var cluster arvados.Cluster
		err = json.Unmarshal(cfgjson, &cluster)
		c.Assert(err, check.IsNil)

		c.Logf("TokenV2 is %s", ac1.AuthToken)
		host := cluster.Services.WebDAV.ExternalURL.Host
		s3args := []string{
			"--ssl", "--no-check-certificate",
			"--host=" + host, "--host-bucket=" + host,
			"--access_key=" + trial.token, "--secret_key=" + trial.token,
		}
		buf, err := exec.Command("s3cmd", append(s3args, "ls", "s3://"+coll.UUID)...).CombinedOutput()
		c.Check(err, check.IsNil)
		c.Check(string(buf), check.Matches, `.* `+fmt.Sprintf("%d", len(testText))+` +s3://`+coll.UUID+`/test.txt\n`)

		buf, _ = exec.Command("s3cmd", append(s3args, "get", "s3://"+coll.UUID+"/test.txt", c.MkDir()+"/tmpfile")...).CombinedOutput()
		// Command fails because we don't return Etag header.
		flen := strconv.Itoa(len(testText))
		c.Check(string(buf), check.Matches, `(?ms).*`+flen+` (bytes in|of `+flen+`).*`)
	}
}

func (s *IntegrationSuite) TestGetCollectionAsAnonymous(c *check.C) {
	conn1 := s.super.Conn("z1111")
	conn3 := s.super.Conn("z3333")
	rootctx1, rootac1, rootkc1 := s.super.RootClients("z1111")
	anonctx3, anonac3, _ := s.super.AnonymousClients("z3333")

	// Make sure anonymous token was set
	c.Assert(anonac3.AuthToken, check.Not(check.Equals), "")

	// Create the collection to find its PDH (but don't save it
	// anywhere yet)
	var coll1 arvados.Collection
	fs1, err := coll1.FileSystem(rootac1, rootkc1)
	c.Assert(err, check.IsNil)
	f, err := fs1.OpenFile("test.txt", os.O_CREATE|os.O_RDWR, 0777)
	c.Assert(err, check.IsNil)
	_, err = io.WriteString(f, "IntegrationSuite.TestGetCollectionAsAnonymous")
	c.Assert(err, check.IsNil)
	err = f.Close()
	c.Assert(err, check.IsNil)
	mtxt, err := fs1.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	pdh := arvados.PortableDataHash(mtxt)

	// Save the collection on cluster z1111.
	coll1, err = conn1.CollectionCreate(rootctx1, arvados.CreateOptions{Attrs: map[string]interface{}{
		"manifest_text": mtxt,
	}})
	c.Assert(err, check.IsNil)

	// Share it with the anonymous users group.
	var outLink arvados.Link
	err = rootac1.RequestAndDecode(&outLink, "POST", "/arvados/v1/links", nil,
		map[string]interface{}{"link": map[string]interface{}{
			"link_class": "permission",
			"name":       "can_read",
			"tail_uuid":  "z1111-j7d0g-anonymouspublic",
			"head_uuid":  coll1.UUID,
		},
		})
	c.Check(err, check.IsNil)

	// Current user should be z3 anonymous user
	outUser, err := anonac3.CurrentUser()
	c.Check(err, check.IsNil)
	c.Check(outUser.UUID, check.Equals, "z3333-tpzed-anonymouspublic")

	// Get the token uuid
	var outAuth arvados.APIClientAuthorization
	err = anonac3.RequestAndDecode(&outAuth, "GET", "/arvados/v1/api_client_authorizations/current", nil, nil)
	c.Check(err, check.IsNil)

	// Make a v2 token of the z3 anonymous user, and use it on z1
	_, anonac1, _ := s.super.ClientsWithToken("z1111", outAuth.TokenV2())
	outUser2, err := anonac1.CurrentUser()
	c.Check(err, check.IsNil)
	// z3 anonymous user will be mapped to the z1 anonymous user
	c.Check(outUser2.UUID, check.Equals, "z1111-tpzed-anonymouspublic")

	// Retrieve the collection (which is on z1) using anonymous from cluster z3333.
	coll, err := conn3.CollectionGet(anonctx3, arvados.GetOptions{UUID: coll1.UUID})
	c.Check(err, check.IsNil)
	c.Check(coll.PortableDataHash, check.Equals, pdh)
}

// z3333 should forward the locally-issued anonymous user token to its login
// cluster z1111. That is no problem because the login cluster controller will
// map any anonymous user token to its local anonymous user.
//
// This needs to work because wb1 has a tendency to slap the local anonymous
// user token on every request as a reader_token, which gets folded into the
// request token list controller.
//
// Use a z1111 user token and the anonymous token from z3333 passed in as a
// reader_token to do a request on z3333, asking for the z1111 anonymous user
// object. The request will be forwarded to the z1111 cluster. The presence of
// the z3333 anonymous user token should not prohibit the request from being
// forwarded.
func (s *IntegrationSuite) TestForwardAnonymousTokenToLoginCluster(c *check.C) {
	conn1 := s.super.Conn("z1111")

	rootctx1, _, _ := s.super.RootClients("z1111")
	_, anonac3, _ := s.super.AnonymousClients("z3333")

	// Make a user connection to z3333 (using a z1111 user, because that's the login cluster)
	_, userac1, _, _ := s.super.UserClients("z3333", rootctx1, c, conn1, "user@example.com", true)

	// Get the anonymous user token for z3333
	var anon3Auth arvados.APIClientAuthorization
	err := anonac3.RequestAndDecode(&anon3Auth, "GET", "/arvados/v1/api_client_authorizations/current", nil, nil)
	c.Check(err, check.IsNil)

	var userList arvados.UserList
	where := make(map[string]string)
	where["uuid"] = "z1111-tpzed-anonymouspublic"
	err = userac1.RequestAndDecode(&userList, "GET", "/arvados/v1/users", nil,
		map[string]interface{}{
			"reader_tokens": []string{anon3Auth.TokenV2()},
			"where":         where,
		},
	)
	// The local z3333 anonymous token must be allowed to be forwarded to the login cluster
	c.Check(err, check.IsNil)

	userac1.AuthToken = "v2/z1111-gj3su-asdfasdfasdfasd/this-token-does-not-validate-so-anonymous-token-will-be-used-instead"
	err = userac1.RequestAndDecode(&userList, "GET", "/arvados/v1/users", nil,
		map[string]interface{}{
			"reader_tokens": []string{anon3Auth.TokenV2()},
			"where":         where,
		},
	)
	c.Check(err, check.IsNil)
}

// Get a token from the login cluster (z1111), use it to submit a
// container request on z2222.
func (s *IntegrationSuite) TestCreateContainerRequestWithFedToken(c *check.C) {
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	_, ac1, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

	// Use ac2 to get the discovery doc with a blank token, so the
	// SDK doesn't magically pass the z1111 token to z2222 before
	// we're ready to start our test.
	_, ac2, _ := s.super.ClientsWithToken("z2222", "")
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
	c.Assert(err, check.IsNil)
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
	c.Assert(err, check.IsNil)
	err = json.NewDecoder(resp.Body).Decode(&cr)
	c.Check(err, check.IsNil)
	c.Check(cr.UUID, check.Matches, "z2222-.*")
}

func (s *IntegrationSuite) TestCreateContainerRequestWithBadToken(c *check.C) {
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	_, ac1, _, au := s.super.UserClients("z1111", rootctx1, c, conn1, "user@example.com", true)

	tests := []struct {
		name         string
		token        string
		expectedCode int
	}{
		{"Good token", ac1.AuthToken, http.StatusOK},
		{"Bogus token", "abcdef", http.StatusUnauthorized},
		{"v1-looking token", "badtoken00badtoken00badtoken00badtoken00b", http.StatusUnauthorized},
		{"v2-looking token", "v2/" + au.UUID + "/badtoken00badtoken00badtoken00badtoken00b", http.StatusUnauthorized},
	}

	body, _ := json.Marshal(map[string]interface{}{
		"container_request": map[string]interface{}{
			"command":         []string{"echo"},
			"container_image": "d41d8cd98f00b204e9800998ecf8427e+0",
			"cwd":             "/",
			"output_path":     "/",
		},
	})

	for _, tt := range tests {
		c.Log(c.TestName() + " " + tt.name)
		ac1.AuthToken = tt.token
		req, err := http.NewRequest("POST", "https://"+ac1.APIHost+"/arvados/v1/container_requests", bytes.NewReader(body))
		c.Assert(err, check.IsNil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := ac1.Do(req)
		c.Assert(err, check.IsNil)
		c.Assert(resp.StatusCode, check.Equals, tt.expectedCode)
	}
}

func (s *IntegrationSuite) TestRequestIDHeader(c *check.C) {
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	userctx1, ac1, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, "user@example.com", true)

	coll, err := conn1.CollectionCreate(userctx1, arvados.CreateOptions{})
	c.Check(err, check.IsNil)
	specimen, err := conn1.SpecimenCreate(userctx1, arvados.CreateOptions{})
	c.Check(err, check.IsNil)

	tests := []struct {
		path            string
		reqIdProvided   bool
		notFoundRequest bool
	}{
		{"/arvados/v1/collections", false, false},
		{"/arvados/v1/collections", true, false},
		{"/arvados/v1/nonexistant", false, true},
		{"/arvados/v1/nonexistant", true, true},
		{"/arvados/v1/collections/" + coll.UUID, false, false},
		{"/arvados/v1/collections/" + coll.UUID, true, false},
		{"/arvados/v1/specimens/" + specimen.UUID, false, false},
		{"/arvados/v1/specimens/" + specimen.UUID, true, false},
		// new code path (lib/controller/router etc) - single-cluster request
		{"/arvados/v1/collections/z1111-4zz18-0123456789abcde", false, true},
		{"/arvados/v1/collections/z1111-4zz18-0123456789abcde", true, true},
		// new code path (lib/controller/router etc) - federated request
		{"/arvados/v1/collections/z2222-4zz18-0123456789abcde", false, true},
		{"/arvados/v1/collections/z2222-4zz18-0123456789abcde", true, true},
		// old code path (proxyRailsAPI) - single-cluster request
		{"/arvados/v1/specimens/z1111-j58dm-0123456789abcde", false, true},
		{"/arvados/v1/specimens/z1111-j58dm-0123456789abcde", true, true},
		// old code path (setupProxyRemoteCluster) - federated request
		{"/arvados/v1/workflows/z2222-7fd4e-0123456789abcde", false, true},
		{"/arvados/v1/workflows/z2222-7fd4e-0123456789abcde", true, true},
	}

	for _, tt := range tests {
		c.Log(c.TestName() + " " + tt.path)
		req, err := http.NewRequest("GET", "https://"+ac1.APIHost+tt.path, nil)
		c.Assert(err, check.IsNil)
		customReqId := "abcdeG"
		if !tt.reqIdProvided {
			c.Assert(req.Header.Get("X-Request-Id"), check.Equals, "")
		} else {
			req.Header.Set("X-Request-Id", customReqId)
		}
		resp, err := ac1.Do(req)
		c.Assert(err, check.IsNil)
		if tt.notFoundRequest {
			c.Check(resp.StatusCode, check.Equals, http.StatusNotFound)
		} else {
			c.Check(resp.StatusCode, check.Equals, http.StatusOK)
		}
		respHdr := resp.Header.Get("X-Request-Id")
		if tt.reqIdProvided {
			c.Check(respHdr, check.Equals, customReqId)
		} else {
			c.Check(respHdr, check.Matches, `req-[0-9a-zA-Z]{20}`)
		}
		if tt.notFoundRequest {
			var jresp httpserver.ErrorResponse
			err := json.NewDecoder(resp.Body).Decode(&jresp)
			c.Check(err, check.IsNil)
			c.Assert(jresp.Errors, check.HasLen, 1)
			c.Check(jresp.Errors[0], check.Matches, `.*\(`+respHdr+`\).*`)
		}
	}
}

// We test the direct access to the database
// normally an integration test would not have a database access, but in this case we need
// to test tokens that are secret, so there is no API response that will give them back
func (s *IntegrationSuite) dbConn(c *check.C, clusterID string) (*sql.DB, *sql.Conn) {
	ctx := context.Background()
	db, err := sql.Open("postgres", s.super.Cluster(clusterID).PostgreSQL.Connection.String())
	c.Assert(err, check.IsNil)

	conn, err := db.Conn(ctx)
	c.Assert(err, check.IsNil)

	rows, err := conn.ExecContext(ctx, `SELECT 1`)
	c.Assert(err, check.IsNil)
	n, err := rows.RowsAffected()
	c.Assert(err, check.IsNil)
	c.Assert(n, check.Equals, int64(1))
	return db, conn
}

// TestRuntimeTokenInCR will test several different tokens in the runtime attribute
// and check the expected results accessing directly to the database if needed.
func (s *IntegrationSuite) TestRuntimeTokenInCR(c *check.C) {
	db, dbconn := s.dbConn(c, "z1111")
	defer db.Close()
	defer dbconn.Close()
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	userctx1, ac1, _, au := s.super.UserClients("z1111", rootctx1, c, conn1, "user@example.com", true)

	tests := []struct {
		name                 string
		token                string
		expectAToGetAValidCR bool
		expectedToken        *string
	}{
		{"Good token z1111 user", ac1.AuthToken, true, &ac1.AuthToken},
		{"Bogus token", "abcdef", false, nil},
		{"v1-looking token", "badtoken00badtoken00badtoken00badtoken00b", false, nil},
		{"v2-looking token", "v2/" + au.UUID + "/badtoken00badtoken00badtoken00badtoken00b", false, nil},
	}

	for _, tt := range tests {
		c.Log(c.TestName() + " " + tt.name)

		rq := map[string]interface{}{
			"command":         []string{"echo"},
			"container_image": "d41d8cd98f00b204e9800998ecf8427e+0",
			"cwd":             "/",
			"output_path":     "/",
			"runtime_token":   tt.token,
		}
		cr, err := conn1.ContainerRequestCreate(userctx1, arvados.CreateOptions{Attrs: rq})
		if tt.expectAToGetAValidCR {
			c.Check(err, check.IsNil)
			c.Check(cr, check.NotNil)
			c.Check(cr.UUID, check.Not(check.Equals), "")
		}

		if tt.expectedToken == nil {
			continue
		}

		c.Logf("cr.UUID: %s", cr.UUID)
		row := dbconn.QueryRowContext(rootctx1, `SELECT runtime_token from container_requests where uuid=$1`, cr.UUID)
		c.Check(row, check.NotNil)
		var token sql.NullString
		row.Scan(&token)
		if c.Check(token.Valid, check.Equals, true) {
			c.Check(token.String, check.Equals, *tt.expectedToken)
		}
	}
}

// TestIntermediateCluster will send a container request to
// one cluster with another cluster as the destination
// and check the tokens are being handled properly
func (s *IntegrationSuite) TestIntermediateCluster(c *check.C) {
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	uctx1, ac1, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, "user@example.com", true)

	tests := []struct {
		name                 string
		token                string
		expectedRuntimeToken string
		expectedUUIDprefix   string
	}{
		{"Good token z1111 user sending a CR to z2222", ac1.AuthToken, "", "z2222-xvhdp-"},
	}

	for _, tt := range tests {
		c.Log(c.TestName() + " " + tt.name)
		rq := map[string]interface{}{
			"command":         []string{"echo"},
			"container_image": "d41d8cd98f00b204e9800998ecf8427e+0",
			"cwd":             "/",
			"output_path":     "/",
			"runtime_token":   tt.token,
		}
		cr, err := conn1.ContainerRequestCreate(uctx1, arvados.CreateOptions{ClusterID: "z2222", Attrs: rq})

		c.Check(err, check.IsNil)
		c.Check(strings.HasPrefix(cr.UUID, tt.expectedUUIDprefix), check.Equals, true)
		c.Check(cr.RuntimeToken, check.Equals, tt.expectedRuntimeToken)
	}
}

// Test for #17785
func (s *IntegrationSuite) TestFederatedApiClientAuthHandling(c *check.C) {
	rootctx1, rootclnt1, _ := s.super.RootClients("z1111")
	conn1 := s.super.Conn("z1111")

	// Make sure LoginCluster is properly configured
	for _, cls := range []string{"z1111", "z3333"} {
		c.Check(
			s.super.Cluster(cls).Login.LoginCluster,
			check.Equals, "z1111",
			check.Commentf("incorrect LoginCluster config on cluster %q", cls))
	}
	// Get user's UUID & attempt to create a token for it on the remote cluster
	_, _, _, user := s.super.UserClients("z1111", rootctx1, c, conn1,
		"user@example.com", true)
	_, rootclnt3, _ := s.super.ClientsWithToken("z3333", rootclnt1.AuthToken)
	var resp arvados.APIClientAuthorization
	err := rootclnt3.RequestAndDecode(
		&resp, "POST", "arvados/v1/api_client_authorizations", nil,
		map[string]interface{}{
			"api_client_authorization": map[string]string{
				"owner_uuid": user.UUID,
			},
		},
	)
	c.Assert(err, check.IsNil)
	c.Assert(resp.APIClientID, check.Not(check.Equals), 0)
	newTok := resp.TokenV2()
	c.Assert(newTok, check.Not(check.Equals), "")

	// Confirm the token is from z1111
	c.Assert(strings.HasPrefix(newTok, "v2/z1111-gj3su-"), check.Equals, true)

	// Confirm the token works and is from the correct user
	_, rootclnt3bis, _ := s.super.ClientsWithToken("z3333", newTok)
	var curUser arvados.User
	err = rootclnt3bis.RequestAndDecode(
		&curUser, "GET", "arvados/v1/users/current", nil, nil,
	)
	c.Assert(err, check.IsNil)
	c.Assert(curUser.UUID, check.Equals, user.UUID)

	// Request the ApiClientAuthorization list using the new token
	_, userClient, _ := s.super.ClientsWithToken("z3333", newTok)
	var acaLst arvados.APIClientAuthorizationList
	err = userClient.RequestAndDecode(
		&acaLst, "GET", "arvados/v1/api_client_authorizations", nil, nil,
	)
	c.Assert(err, check.IsNil)
}

// Test for bug #18076
func (s *IntegrationSuite) TestStaleCachedUserRecord(c *check.C) {
	rootctx1, _, _ := s.super.RootClients("z1111")
	_, rootclnt3, _ := s.super.RootClients("z3333")
	conn1 := s.super.Conn("z1111")
	conn3 := s.super.Conn("z3333")

	// Make sure LoginCluster is properly configured
	for _, cls := range []string{"z1111", "z3333"} {
		c.Check(
			s.super.Cluster(cls).Login.LoginCluster,
			check.Equals, "z1111",
			check.Commentf("incorrect LoginCluster config on cluster %q", cls))
	}

	for testCaseNr, testCase := range []struct {
		name           string
		withRepository bool
	}{
		{"User without local repository", false},
		{"User with local repository", true},
	} {
		c.Log(c.TestName() + " " + testCase.name)
		// Create some users, request them on the federated cluster so they're cached.
		var users []arvados.User
		for userNr := 0; userNr < 2; userNr++ {
			_, _, _, user := s.super.UserClients("z1111",
				rootctx1,
				c,
				conn1,
				fmt.Sprintf("user%d%d@example.com", testCaseNr, userNr),
				true)
			c.Assert(user.Username, check.Not(check.Equals), "")
			users = append(users, user)

			lst, err := conn3.UserList(rootctx1, arvados.ListOptions{Limit: -1})
			c.Assert(err, check.Equals, nil)
			userFound := false
			for _, fedUser := range lst.Items {
				if fedUser.UUID == user.UUID {
					c.Assert(fedUser.Username, check.Equals, user.Username)
					userFound = true
					break
				}
			}
			c.Assert(userFound, check.Equals, true)

			if testCase.withRepository {
				var repo interface{}
				err = rootclnt3.RequestAndDecode(
					&repo, "POST", "arvados/v1/repositories", nil,
					map[string]interface{}{
						"repository": map[string]string{
							"name":       fmt.Sprintf("%s/test", user.Username),
							"owner_uuid": user.UUID,
						},
					},
				)
				c.Assert(err, check.IsNil)
			}
		}

		// Swap the usernames
		_, err := conn1.UserUpdate(rootctx1, arvados.UpdateOptions{
			UUID: users[0].UUID,
			Attrs: map[string]interface{}{
				"username": "",
			},
		})
		c.Assert(err, check.Equals, nil)
		_, err = conn1.UserUpdate(rootctx1, arvados.UpdateOptions{
			UUID: users[1].UUID,
			Attrs: map[string]interface{}{
				"username": users[0].Username,
			},
		})
		c.Assert(err, check.Equals, nil)
		_, err = conn1.UserUpdate(rootctx1, arvados.UpdateOptions{
			UUID: users[0].UUID,
			Attrs: map[string]interface{}{
				"username": users[1].Username,
			},
		})
		c.Assert(err, check.Equals, nil)

		// Re-request the list on the federated cluster & check for updates
		lst, err := conn3.UserList(rootctx1, arvados.ListOptions{Limit: -1})
		c.Assert(err, check.Equals, nil)
		var user0Found, user1Found bool
		for _, user := range lst.Items {
			if user.UUID == users[0].UUID {
				user0Found = true
				c.Assert(user.Username, check.Equals, users[1].Username)
			} else if user.UUID == users[1].UUID {
				user1Found = true
				c.Assert(user.Username, check.Equals, users[0].Username)
			}
		}
		c.Assert(user0Found, check.Equals, true)
		c.Assert(user1Found, check.Equals, true)
	}
}

// Test for bug #16263
func (s *IntegrationSuite) TestListUsers(c *check.C) {
	rootctx1, _, _ := s.super.RootClients("z1111")
	conn1 := s.super.Conn("z1111")
	conn3 := s.super.Conn("z3333")
	userctx1, _, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

	// Make sure LoginCluster is properly configured
	for _, cls := range []string{"z1111", "z2222", "z3333"} {
		c.Check(
			s.super.Cluster(cls).Login.LoginCluster,
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
			break
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

	// Deactivated user no longer has working token
	user1, err = conn3.UserGetCurrent(userctx1, arvados.GetOptions{})
	c.Assert(err, check.ErrorMatches, `.*401 Unauthorized.*`)
}

func (s *IntegrationSuite) TestSetupUserWithVM(c *check.C) {
	conn1 := s.super.Conn("z1111")
	conn3 := s.super.Conn("z3333")
	rootctx1, rootac1, _ := s.super.RootClients("z1111")

	// Create user on LoginCluster z1111
	_, _, _, user := s.super.UserClients("z1111", rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

	// Make a new root token (because rootClients() uses SystemRootToken)
	var outAuth arvados.APIClientAuthorization
	err := rootac1.RequestAndDecode(&outAuth, "POST", "/arvados/v1/api_client_authorizations", nil, nil)
	c.Check(err, check.IsNil)

	// Make a v2 root token to communicate with z3333
	rootctx3, rootac3, _ := s.super.ClientsWithToken("z3333", outAuth.TokenV2())

	// Create VM on z3333
	var outVM arvados.VirtualMachine
	err = rootac3.RequestAndDecode(&outVM, "POST", "/arvados/v1/virtual_machines", nil,
		map[string]interface{}{"virtual_machine": map[string]interface{}{
			"hostname": "example",
		},
		})
	c.Check(outVM.UUID[0:5], check.Equals, "z3333")
	c.Check(err, check.IsNil)

	// Make sure z3333 user list is up to date
	_, err = conn3.UserList(rootctx3, arvados.ListOptions{Limit: 1000})
	c.Check(err, check.IsNil)

	// Try to set up user on z3333 with the VM
	_, err = conn3.UserSetup(rootctx3, arvados.UserSetupOptions{UUID: user.UUID, VMUUID: outVM.UUID})
	c.Check(err, check.IsNil)

	var outLinks arvados.LinkList
	err = rootac3.RequestAndDecode(&outLinks, "GET", "/arvados/v1/links", nil,
		arvados.ListOptions{
			Limit: 1000,
			Filters: []arvados.Filter{
				{
					Attr:     "tail_uuid",
					Operator: "=",
					Operand:  user.UUID,
				},
				{
					Attr:     "head_uuid",
					Operator: "=",
					Operand:  outVM.UUID,
				},
				{
					Attr:     "name",
					Operator: "=",
					Operand:  "can_login",
				},
				{
					Attr:     "link_class",
					Operator: "=",
					Operand:  "permission",
				}}})
	c.Check(err, check.IsNil)

	c.Check(len(outLinks.Items), check.Equals, 1)
}

func (s *IntegrationSuite) TestOIDCAccessTokenAuth(c *check.C) {
	conn1 := s.super.Conn("z1111")
	rootctx1, _, _ := s.super.RootClients("z1111")
	s.super.UserClients("z1111", rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

	accesstoken := s.oidcprovider.ValidAccessToken()

	for _, clusterID := range []string{"z1111", "z2222"} {

		var coll arvados.Collection

		// Write some file data and create a collection
		{
			c.Logf("save collection to %s", clusterID)

			conn := s.super.Conn(clusterID)
			ctx, ac, kc := s.super.ClientsWithToken(clusterID, accesstoken)

			fs, err := coll.FileSystem(ac, kc)
			c.Assert(err, check.IsNil)
			f, err := fs.OpenFile("test.txt", os.O_CREATE|os.O_RDWR, 0777)
			c.Assert(err, check.IsNil)
			_, err = io.WriteString(f, "IntegrationSuite.TestOIDCAccessTokenAuth")
			c.Assert(err, check.IsNil)
			err = f.Close()
			c.Assert(err, check.IsNil)
			mtxt, err := fs.MarshalManifest(".")
			c.Assert(err, check.IsNil)
			coll, err = conn.CollectionCreate(ctx, arvados.CreateOptions{Attrs: map[string]interface{}{
				"manifest_text": mtxt,
			}})
			c.Assert(err, check.IsNil)
		}

		// Read the collection & file data -- both from the
		// cluster where it was created, and from the other
		// cluster.
		for _, readClusterID := range []string{"z1111", "z2222", "z3333"} {
			c.Logf("retrieve %s from %s", coll.UUID, readClusterID)

			conn := s.super.Conn(readClusterID)
			ctx, ac, kc := s.super.ClientsWithToken(readClusterID, accesstoken)

			user, err := conn.UserGetCurrent(ctx, arvados.GetOptions{})
			c.Assert(err, check.IsNil)
			c.Check(user.FullName, check.Equals, "Example User")
			readcoll, err := conn.CollectionGet(ctx, arvados.GetOptions{UUID: coll.UUID})
			c.Assert(err, check.IsNil)
			c.Check(readcoll.ManifestText, check.Not(check.Equals), "")
			fs, err := readcoll.FileSystem(ac, kc)
			c.Assert(err, check.IsNil)
			f, err := fs.Open("test.txt")
			c.Assert(err, check.IsNil)
			buf, err := ioutil.ReadAll(f)
			c.Assert(err, check.IsNil)
			c.Check(buf, check.DeepEquals, []byte("IntegrationSuite.TestOIDCAccessTokenAuth"))
		}
	}
}

// z3333 should not forward a locally-issued container runtime token,
// associated with a z1111 user, to its login cluster z1111. z1111
// would only call back to z3333 and then reject the response because
// the user ID does not match the token prefix. See
// dev.arvados.org/issues/18346
func (s *IntegrationSuite) TestForwardRuntimeTokenToLoginCluster(c *check.C) {
	db3, db3conn := s.dbConn(c, "z3333")
	defer db3.Close()
	defer db3conn.Close()
	rootctx1, _, _ := s.super.RootClients("z1111")
	rootctx3, _, _ := s.super.RootClients("z3333")
	conn1 := s.super.Conn("z1111")
	conn3 := s.super.Conn("z3333")
	userctx1, _, _, _ := s.super.UserClients("z1111", rootctx1, c, conn1, "user@example.com", true)

	user1, err := conn1.UserGetCurrent(userctx1, arvados.GetOptions{})
	c.Assert(err, check.IsNil)
	c.Logf("user1 %+v", user1)

	imageColl, err := conn3.CollectionCreate(userctx1, arvados.CreateOptions{Attrs: map[string]interface{}{
		"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.tar\n",
	}})
	c.Assert(err, check.IsNil)
	c.Logf("imageColl %+v", imageColl)

	cr, err := conn3.ContainerRequestCreate(userctx1, arvados.CreateOptions{Attrs: map[string]interface{}{
		"state":           "Committed",
		"command":         []string{"echo"},
		"container_image": imageColl.PortableDataHash,
		"cwd":             "/",
		"output_path":     "/",
		"priority":        1,
		"runtime_constraints": arvados.RuntimeConstraints{
			VCPUs: 1,
			RAM:   1000000000,
		},
	}})
	c.Assert(err, check.IsNil)
	c.Logf("container request %+v", cr)
	ctr, err := conn3.ContainerLock(rootctx3, arvados.GetOptions{UUID: cr.ContainerUUID})
	c.Assert(err, check.IsNil)
	c.Logf("container %+v", ctr)

	// We could use conn3.ContainerAuth() here, but that API
	// hasn't been added to sdk/go/arvados/api.go yet.
	row := db3conn.QueryRowContext(context.Background(), `SELECT api_token from api_client_authorizations where uuid=$1`, ctr.AuthUUID)
	c.Check(row, check.NotNil)
	var val sql.NullString
	row.Scan(&val)
	c.Assert(val.Valid, check.Equals, true)
	runtimeToken := "v2/" + ctr.AuthUUID + "/" + val.String
	ctrctx, _, _ := s.super.ClientsWithToken("z3333", runtimeToken)
	c.Logf("container runtime token %+v", runtimeToken)

	_, err = conn3.UserGet(ctrctx, arvados.GetOptions{UUID: user1.UUID})
	c.Assert(err, check.NotNil)
	c.Check(err, check.ErrorMatches, `request failed: .* 401 Unauthorized: cannot use a locally issued token to forward a request to our login cluster \(z1111\)`)
	c.Check(err, check.Not(check.ErrorMatches), `(?ms).*127\.0\.0\.11.*`)
}

func (s *IntegrationSuite) TestRunTrivialContainer(c *check.C) {
	outcoll := s.runContainer(c, "z1111", map[string]interface{}{
		"command":             []string{"sh", "-c", "touch \"/out/hello world\" /out/ohai"},
		"container_image":     "busybox:uclibc",
		"cwd":                 "/tmp",
		"environment":         map[string]string{},
		"mounts":              map[string]arvados.Mount{"/out": {Kind: "tmp", Capacity: 10000}},
		"output_path":         "/out",
		"runtime_constraints": arvados.RuntimeConstraints{RAM: 100000000, VCPUs: 1},
		"priority":            1,
		"state":               arvados.ContainerRequestStateCommitted,
	}, 0)
	c.Check(outcoll.ManifestText, check.Matches, `\. d41d8.* 0:0:hello\\040world 0:0:ohai\n`)
	c.Check(outcoll.PortableDataHash, check.Equals, "8fa5dee9231a724d7cf377c5a2f4907c+65")
}

func (s *IntegrationSuite) runContainer(c *check.C, clusterID string, ctrSpec map[string]interface{}, expectExitCode int) arvados.Collection {
	conn := s.super.Conn(clusterID)
	rootctx, _, _ := s.super.RootClients(clusterID)
	_, ac, kc, _ := s.super.UserClients(clusterID, rootctx, c, conn, s.oidcprovider.AuthEmail, true)

	c.Log("[docker load]")
	out, err := exec.Command("docker", "load", "--input", arvadostest.BusyboxDockerImage(c)).CombinedOutput()
	c.Logf("[docker load output] %s", out)
	c.Check(err, check.IsNil)

	c.Log("[arv-keepdocker]")
	akd := exec.Command("arv-keepdocker", "--no-resume", "busybox:uclibc")
	akd.Env = append(os.Environ(), "ARVADOS_API_HOST="+ac.APIHost, "ARVADOS_API_HOST_INSECURE=1", "ARVADOS_API_TOKEN="+ac.AuthToken)
	out, err = akd.CombinedOutput()
	c.Logf("[arv-keepdocker output]\n%s", out)
	c.Check(err, check.IsNil)

	var cr arvados.ContainerRequest
	err = ac.RequestAndDecode(&cr, "POST", "/arvados/v1/container_requests", nil, map[string]interface{}{
		"container_request": ctrSpec,
	})
	c.Assert(err, check.IsNil)

	showlogs := func(collectionID string) {
		var logcoll arvados.Collection
		err = ac.RequestAndDecode(&logcoll, "GET", "/arvados/v1/collections/"+collectionID, nil, nil)
		c.Assert(err, check.IsNil)
		cfs, err := logcoll.FileSystem(ac, kc)
		c.Assert(err, check.IsNil)
		fs.WalkDir(arvados.FS(cfs), "/", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() || strings.HasPrefix(path, "/log for container") {
				return nil
			}
			f, err := cfs.Open(path)
			c.Assert(err, check.IsNil)
			defer f.Close()
			buf, err := ioutil.ReadAll(f)
			c.Assert(err, check.IsNil)
			c.Logf("=== %s\n%s\n", path, buf)
			return nil
		})
	}

	var ctr arvados.Container
	var lastState arvados.ContainerState
	deadline := time.Now().Add(time.Minute)
wait:
	for ; ; lastState = ctr.State {
		if time.Now().After(deadline) {
			c.Errorf("timed out, container request state is %q", cr.State)
			showlogs(ctr.Log)
			c.FailNow()
		}
		err = ac.RequestAndDecode(&ctr, "GET", "/arvados/v1/containers/"+cr.ContainerUUID, nil, nil)
		if err != nil {
			// container req is being auto-retried with a new container uuid
			ac.RequestAndDecode(&cr, "GET", "/arvados/v1/container_requests/"+cr.UUID, nil, nil)
			c.Assert(err, check.IsNil)
			time.Sleep(time.Second / 2)
			continue
		}
		switch ctr.State {
		case lastState:
			time.Sleep(time.Second / 2)
		case arvados.ContainerStateComplete:
			break wait
		case arvados.ContainerStateQueued, arvados.ContainerStateLocked, arvados.ContainerStateRunning:
			c.Logf("container state changed to %q", ctr.State)
		default:
			c.Errorf("unexpected container state %q", ctr.State)
			showlogs(ctr.Log)
			c.FailNow()
		}
	}
	c.Check(ctr.ExitCode, check.Equals, 0)

	err = ac.RequestAndDecode(&cr, "GET", "/arvados/v1/container_requests/"+cr.UUID, nil, nil)
	c.Assert(err, check.IsNil)

	showlogs(cr.LogUUID)

	var outcoll arvados.Collection
	err = ac.RequestAndDecode(&outcoll, "GET", "/arvados/v1/collections/"+cr.OutputUUID, nil, nil)
	c.Assert(err, check.IsNil)
	return outcoll
}
