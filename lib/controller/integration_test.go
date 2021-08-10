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
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"git.arvados.org/arvados.git/lib/boot"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&IntegrationSuite{})

type IntegrationSuite struct {
	testClusters map[string]*boot.TestCluster
	oidcprovider *arvadostest.OIDCProvider
}

func (s *IntegrationSuite) SetUpSuite(c *check.C) {
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
}

func (s *IntegrationSuite) TearDownSuite(c *check.C) {
	for _, c := range s.testClusters {
		c.Super.Stop()
	}
}

func (s *IntegrationSuite) TestGetCollectionByPDH(c *check.C) {
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	conn3 := s.testClusters["z3333"].Conn()
	userctx1, ac1, kc1, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

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
func (s *IntegrationSuite) TestRemoteTokenCacheRace(c *check.C) {
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	rootctx2, _, _ := s.testClusters["z2222"].RootClients()
	conn2 := s.testClusters["z2222"].Conn()
	userctx1, _, _, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, "user2@example.com", true)

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
			c.Check(err, check.IsNil)
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
			c.Check(err, check.IsNil)
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

	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	userctx1, ac1, _, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)
	conn3 := s.testClusters["z3333"].Conn()

	createColl := func(clusterID string) arvados.Collection {
		_, ac, kc := s.testClusters[clusterID].ClientsWithToken(ac1.AuthToken)
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
		coll, err = s.testClusters[clusterID].Conn().CollectionCreate(userctx1, arvados.CreateOptions{Attrs: map[string]interface{}{
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
	conn1 := s.testClusters["z1111"].Conn()
	conn3 := s.testClusters["z3333"].Conn()
	rootctx1, rootac1, rootkc1 := s.testClusters["z1111"].RootClients()
	anonctx3, anonac3, _ := s.testClusters["z3333"].AnonymousClients()

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
	_, anonac1, _ := s.testClusters["z1111"].ClientsWithToken(outAuth.TokenV2())
	outUser2, err := anonac1.CurrentUser()
	c.Check(err, check.IsNil)
	// z3 anonymous user will be mapped to the z1 anonymous user
	c.Check(outUser2.UUID, check.Equals, "z1111-tpzed-anonymouspublic")

	// Retrieve the collection (which is on z1) using anonymous from cluster z3333.
	coll, err := conn3.CollectionGet(anonctx3, arvados.GetOptions{UUID: coll1.UUID})
	c.Check(err, check.IsNil)
	c.Check(coll.PortableDataHash, check.Equals, pdh)
}

// Get a token from the login cluster (z1111), use it to submit a
// container request on z2222.
func (s *IntegrationSuite) TestCreateContainerRequestWithFedToken(c *check.C) {
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	_, ac1, _, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

	// Use ac2 to get the discovery doc with a blank token, so the
	// SDK doesn't magically pass the z1111 token to z2222 before
	// we're ready to start our test.
	_, ac2, _ := s.testClusters["z2222"].ClientsWithToken("")
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
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	_, ac1, _, au := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, "user@example.com", true)

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
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	userctx1, ac1, _, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, "user@example.com", true)

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
		{"/arvados/v1/collections/z1111-4zz18-0123456789abcde", false, true},
		{"/arvados/v1/collections/z1111-4zz18-0123456789abcde", true, true},
		{"/arvados/v1/specimens/z1111-j58dm-0123456789abcde", false, true},
		{"/arvados/v1/specimens/z1111-j58dm-0123456789abcde", true, true},
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
		if !tt.reqIdProvided {
			c.Check(resp.Header.Get("X-Request-Id"), check.Matches, "^req-[0-9a-zA-Z]{20}$")
			if tt.notFoundRequest {
				var jresp httpserver.ErrorResponse
				err := json.NewDecoder(resp.Body).Decode(&jresp)
				c.Check(err, check.IsNil)
				c.Assert(jresp.Errors, check.HasLen, 1)
				c.Check(jresp.Errors[0], check.Matches, "^.*(req-[0-9a-zA-Z]{20}).*$")
			}
		} else {
			c.Check(resp.Header.Get("X-Request-Id"), check.Equals, customReqId)
			if tt.notFoundRequest {
				var jresp httpserver.ErrorResponse
				err := json.NewDecoder(resp.Body).Decode(&jresp)
				c.Check(err, check.IsNil)
				c.Assert(jresp.Errors, check.HasLen, 1)
				c.Check(jresp.Errors[0], check.Matches, "^.*("+customReqId+").*$")
			}
		}
	}
}

// We test the direct access to the database
// normally an integration test would not have a database access, but in this case we need
// to test tokens that are secret, so there is no API response that will give them back
func (s *IntegrationSuite) dbConn(c *check.C, clusterID string) (*sql.DB, *sql.Conn) {
	ctx := context.Background()
	db, err := sql.Open("postgres", s.testClusters[clusterID].Super.Cluster().PostgreSQL.Connection.String())
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
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	userctx1, ac1, _, au := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, "user@example.com", true)

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
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	uctx1, ac1, _, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, "user@example.com", true)

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

// Test for bug #16263
func (s *IntegrationSuite) TestListUsers(c *check.C) {
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	conn1 := s.testClusters["z1111"].Conn()
	conn3 := s.testClusters["z3333"].Conn()
	userctx1, _, _, _ := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

	// Make sure LoginCluster is properly configured
	for cls := range s.testClusters {
		c.Check(
			s.testClusters[cls].Config.Clusters[cls].Login.LoginCluster,
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

func (s *IntegrationSuite) TestSetupUserWithVM(c *check.C) {
	conn1 := s.testClusters["z1111"].Conn()
	conn3 := s.testClusters["z3333"].Conn()
	rootctx1, rootac1, _ := s.testClusters["z1111"].RootClients()

	// Create user on LoginCluster z1111
	_, _, _, user := s.testClusters["z1111"].UserClients(rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

	// Make a new root token (because rootClients() uses SystemRootToken)
	var outAuth arvados.APIClientAuthorization
	err := rootac1.RequestAndDecode(&outAuth, "POST", "/arvados/v1/api_client_authorizations", nil, nil)
	c.Check(err, check.IsNil)

	// Make a v2 root token to communicate with z3333
	rootctx3, rootac3, _ := s.testClusters["z3333"].ClientsWithToken(outAuth.TokenV2())

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
	conn1 := s.testClusters["z1111"].Conn()
	rootctx1, _, _ := s.testClusters["z1111"].RootClients()
	s.testClusters["z1111"].UserClients(rootctx1, c, conn1, s.oidcprovider.AuthEmail, true)

	accesstoken := s.oidcprovider.ValidAccessToken()

	for _, clusterID := range []string{"z1111", "z2222"} {

		var coll arvados.Collection

		// Write some file data and create a collection
		{
			c.Logf("save collection to %s", clusterID)

			conn := s.testClusters[clusterID].Conn()
			ctx, ac, kc := s.testClusters[clusterID].ClientsWithToken(accesstoken)

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

			conn := s.testClusters[readClusterID].Conn()
			ctx, ac, kc := s.testClusters[readClusterID].ClientsWithToken(accesstoken)

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
