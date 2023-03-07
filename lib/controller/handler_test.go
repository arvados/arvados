// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&HandlerSuite{})

type HandlerSuite struct {
	cluster *arvados.Cluster
	handler *Handler
	logbuf  *bytes.Buffer
	ctx     context.Context
	cancel  context.CancelFunc
}

func (s *HandlerSuite) SetUpTest(c *check.C) {
	s.logbuf = &bytes.Buffer{}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.ctx = ctxlog.Context(s.ctx, ctxlog.New(io.MultiWriter(os.Stderr, s.logbuf), "json", "debug"))
	s.cluster = &arvados.Cluster{
		ClusterID:  "zzzzz",
		PostgreSQL: integrationTestCluster().PostgreSQL,
	}
	s.cluster.API.RequestTimeout = arvados.Duration(5 * time.Minute)
	s.cluster.TLS.Insecure = true
	arvadostest.SetServiceURL(&s.cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	arvadostest.SetServiceURL(&s.cluster.Services.Controller, "http://localhost:/")
	s.handler = newHandler(s.ctx, s.cluster, "", prometheus.NewRegistry()).(*Handler)
}

func (s *HandlerSuite) TearDownTest(c *check.C) {
	s.cancel()
}

func (s *HandlerSuite) TestConfigExport(c *check.C) {
	s.cluster.ManagementToken = "secret"
	s.cluster.SystemRootToken = "secret"
	s.cluster.Collections.BlobSigning = true
	s.cluster.Collections.BlobSigningTTL = arvados.Duration(23 * time.Second)
	for _, method := range []string{"GET", "OPTIONS"} {
		req := httptest.NewRequest(method, "/arvados/v1/config", nil)
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Log(resp.Body.String())
		if !c.Check(resp.Code, check.Equals, http.StatusOK) {
			continue
		}
		c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, `*`)
		c.Check(resp.Header().Get("Access-Control-Allow-Methods"), check.Matches, `.*\bGET\b.*`)
		c.Check(resp.Header().Get("Access-Control-Allow-Headers"), check.Matches, `.+`)
		if method == "OPTIONS" {
			c.Check(resp.Body.String(), check.HasLen, 0)
			continue
		}
		var cluster arvados.Cluster
		err := json.Unmarshal(resp.Body.Bytes(), &cluster)
		c.Check(err, check.IsNil)
		c.Check(cluster.ManagementToken, check.Equals, "")
		c.Check(cluster.SystemRootToken, check.Equals, "")
		c.Check(cluster.Collections.BlobSigning, check.Equals, true)
		c.Check(cluster.Collections.BlobSigningTTL, check.Equals, arvados.Duration(23*time.Second))
	}
}

func (s *HandlerSuite) TestVocabularyExport(c *check.C) {
	voc := `{
		"strict_tags": false,
		"tags": {
			"IDTAGIMPORTANCE": {
				"strict": false,
				"labels": [{"label": "Importance"}],
				"values": {
					"HIGH": {
						"labels": [{"label": "High"}]
					},
					"LOW": {
						"labels": [{"label": "Low"}]
					}
				}
			}
		}
	}`
	f, err := os.CreateTemp("", "test-vocabulary-*.json")
	c.Assert(err, check.IsNil)
	defer os.Remove(f.Name())
	_, err = f.WriteString(voc)
	c.Assert(err, check.IsNil)
	f.Close()
	s.cluster.API.VocabularyPath = f.Name()
	for _, method := range []string{"GET", "OPTIONS"} {
		c.Log(c.TestName()+" ", method)
		req := httptest.NewRequest(method, "/arvados/v1/vocabulary", nil)
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Log(resp.Body.String())
		if !c.Check(resp.Code, check.Equals, http.StatusOK) {
			continue
		}
		c.Check(resp.Header().Get("Access-Control-Allow-Origin"), check.Equals, `*`)
		c.Check(resp.Header().Get("Access-Control-Allow-Methods"), check.Matches, `.*\bGET\b.*`)
		c.Check(resp.Header().Get("Access-Control-Allow-Headers"), check.Matches, `.+`)
		if method == "OPTIONS" {
			c.Check(resp.Body.String(), check.HasLen, 0)
			continue
		}
		var expectedVoc, receivedVoc *arvados.Vocabulary
		err := json.Unmarshal([]byte(voc), &expectedVoc)
		c.Check(err, check.IsNil)
		err = json.Unmarshal(resp.Body.Bytes(), &receivedVoc)
		c.Check(err, check.IsNil)
		c.Check(receivedVoc, check.DeepEquals, expectedVoc)
	}
}

func (s *HandlerSuite) TestVocabularyFailedCheckStatus(c *check.C) {
	voc := `{
		"strict_tags": false,
		"tags": {
			"IDTAGIMPORTANCE": {
				"strict": true,
				"labels": [{"label": "Importance"}],
				"values": {
					"HIGH": {
						"labels": [{"label": "High"}]
					},
					"LOW": {
						"labels": [{"label": "Low"}]
					}
				}
			}
		}
	}`
	f, err := os.CreateTemp("", "test-vocabulary-*.json")
	c.Assert(err, check.IsNil)
	defer os.Remove(f.Name())
	_, err = f.WriteString(voc)
	c.Assert(err, check.IsNil)
	f.Close()
	s.cluster.API.VocabularyPath = f.Name()

	req := httptest.NewRequest("POST", "/arvados/v1/collections",
		strings.NewReader(`{
			"collection": {
				"properties": {
					"IDTAGIMPORTANCE": "Critical"
				}
			}
		}`))
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	req.Header.Set("Content-type", "application/json")

	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Log(resp.Body.String())
	c.Assert(resp.Code, check.Equals, http.StatusBadRequest)
	var jresp httpserver.ErrorResponse
	err = json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Assert(len(jresp.Errors), check.Equals, 1)
	c.Check(jresp.Errors[0], check.Matches, `.*tag value.*is not valid for key.*`)
}

func (s *HandlerSuite) TestProxyDiscoveryDoc(c *check.C) {
	req := httptest.NewRequest("GET", "/discovery/v1/apis/arvados/v1/rest", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var dd arvados.DiscoveryDocument
	err := json.Unmarshal(resp.Body.Bytes(), &dd)
	c.Check(err, check.IsNil)
	c.Check(dd.BlobSignatureTTL, check.Not(check.Equals), int64(0))
	c.Check(dd.BlobSignatureTTL > 0, check.Equals, true)
	c.Check(len(dd.Resources), check.Not(check.Equals), 0)
	c.Check(len(dd.Schemas), check.Not(check.Equals), 0)
}

// Handler should give up and exit early if request context is
// cancelled due to client hangup, httpserver.HandlerWithDeadline,
// etc.
func (s *HandlerSuite) TestRequestCancel(c *check.C) {
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest("GET", "/discovery/v1/apis/arvados/v1/rest", nil).WithContext(ctx)
	resp := httptest.NewRecorder()
	cancel()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusBadGateway)
	var jresp httpserver.ErrorResponse
	err := json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Assert(len(jresp.Errors), check.Equals, 1)
	c.Check(jresp.Errors[0], check.Matches, `.*context canceled`)
}

func (s *HandlerSuite) TestProxyWithoutToken(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
	jresp := map[string]interface{}{}
	err := json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Check(jresp["errors"], check.FitsTypeOf, []interface{}{})
}

func (s *HandlerSuite) TestProxyWithToken(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var u arvados.User
	err := json.Unmarshal(resp.Body.Bytes(), &u)
	c.Check(err, check.IsNil)
	c.Check(u.UUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *HandlerSuite) TestProxyWithTokenInRequestBody(c *check.C) {
	req := httptest.NewRequest("POST", "/arvados/v1/users/current", strings.NewReader(url.Values{
		"_method":   {"GET"},
		"api_token": {arvadostest.ActiveToken},
	}.Encode()))
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var u arvados.User
	err := json.Unmarshal(resp.Body.Bytes(), &u)
	c.Check(err, check.IsNil)
	c.Check(u.UUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *HandlerSuite) TestProxyNotFound(c *check.C) {
	req := httptest.NewRequest("GET", "/arvados/v1/xyzzy", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	jresp := map[string]interface{}{}
	err := json.Unmarshal(resp.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	c.Check(jresp["errors"], check.FitsTypeOf, []interface{}{})
}

func (s *HandlerSuite) TestLogoutGoogle(c *check.C) {
	s.cluster.Services.Workbench2.ExternalURL = arvados.URL{Scheme: "https", Host: "wb2.example", Path: "/"}
	s.cluster.Login.Google.Enable = true
	s.cluster.Login.Google.ClientID = "test"
	req := httptest.NewRequest("GET", "https://0.0.0.0:1/logout?return_to=https://wb2.example/", nil)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	if !c.Check(resp.Code, check.Equals, http.StatusFound) {
		c.Log(resp.Body.String())
	}
	c.Check(resp.Header().Get("Location"), check.Equals, "https://wb2.example/")
}

func (s *HandlerSuite) TestValidateV1APIToken(c *check.C) {
	c.Assert(s.handler.CheckHealth(), check.IsNil)
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	user, ok, err := s.handler.validateAPItoken(req, arvadostest.ActiveToken)
	c.Assert(err, check.IsNil)
	c.Check(ok, check.Equals, true)
	c.Check(user.Authorization.UUID, check.Equals, arvadostest.ActiveTokenUUID)
	c.Check(user.Authorization.APIToken, check.Equals, arvadostest.ActiveToken)
	c.Check(user.Authorization.Scopes, check.DeepEquals, []string{"all"})
	c.Check(user.UUID, check.Equals, arvadostest.ActiveUserUUID)
}

func (s *HandlerSuite) TestValidateV2APIToken(c *check.C) {
	c.Assert(s.handler.CheckHealth(), check.IsNil)
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	user, ok, err := s.handler.validateAPItoken(req, arvadostest.ActiveTokenV2)
	c.Assert(err, check.IsNil)
	c.Check(ok, check.Equals, true)
	c.Check(user.Authorization.UUID, check.Equals, arvadostest.ActiveTokenUUID)
	c.Check(user.Authorization.APIToken, check.Equals, arvadostest.ActiveToken)
	c.Check(user.Authorization.Scopes, check.DeepEquals, []string{"all"})
	c.Check(user.UUID, check.Equals, arvadostest.ActiveUserUUID)
	c.Check(user.Authorization.TokenV2(), check.Equals, arvadostest.ActiveTokenV2)
}

func (s *HandlerSuite) TestValidateRemoteToken(c *check.C) {
	saltedToken, err := auth.SaltToken(arvadostest.ActiveTokenV2, "abcde")
	c.Assert(err, check.IsNil)
	for _, trial := range []struct {
		code  int
		token string
	}{
		{http.StatusOK, saltedToken},
		{http.StatusUnauthorized, "bogus"},
	} {
		req := httptest.NewRequest("GET", "https://0.0.0.0:1/arvados/v1/users/current?remote=abcde", nil)
		req.Header.Set("Authorization", "Bearer "+trial.token)
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		if !c.Check(resp.Code, check.Equals, trial.code) {
			c.Logf("HTTP %d: %s", resp.Code, resp.Body.String())
		}
	}
}

func (s *HandlerSuite) TestLogTokenUUID(c *check.C) {
	req := httptest.NewRequest("GET", "https://0.0.0.0/arvados/v1/users/current", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveTokenV2)
	req = req.WithContext(s.ctx)
	resp := httptest.NewRecorder()
	httpserver.LogRequests(s.handler).ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(s.logbuf.String(), check.Matches, `(?ms).*"tokenUUIDs":\["`+strings.Split(arvadostest.ActiveTokenV2, "/")[1]+`"\].*`)
}

func (s *HandlerSuite) TestCreateAPIToken(c *check.C) {
	c.Assert(s.handler.CheckHealth(), check.IsNil)
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	auth, err := s.handler.createAPItoken(req, arvadostest.ActiveUserUUID, nil)
	c.Assert(err, check.IsNil)
	c.Check(auth.Scopes, check.DeepEquals, []string{"all"})

	user, ok, err := s.handler.validateAPItoken(req, auth.TokenV2())
	c.Assert(err, check.IsNil)
	c.Check(ok, check.Equals, true)
	c.Check(user.Authorization.UUID, check.Equals, auth.UUID)
	c.Check(user.Authorization.APIToken, check.Equals, auth.APIToken)
	c.Check(user.Authorization.Scopes, check.DeepEquals, []string{"all"})
	c.Check(user.UUID, check.Equals, arvadostest.ActiveUserUUID)
	c.Check(user.Authorization.TokenV2(), check.Equals, auth.TokenV2())
}

func (s *HandlerSuite) CheckObjectType(c *check.C, url string, token string, skippedFields map[string]bool) {
	var proxied, direct map[string]interface{}
	var err error

	// Get collection from controller
	req := httptest.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Assert(resp.Code, check.Equals, http.StatusOK,
		check.Commentf("Wasn't able to get data from the controller at %q: %q", url, resp.Body.String()))
	err = json.Unmarshal(resp.Body.Bytes(), &proxied)
	c.Check(err, check.Equals, nil)

	// Get collection directly from RailsAPI
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp2, err := client.Get(s.cluster.Services.RailsAPI.ExternalURL.String() + url + "/?api_token=" + token)
	c.Check(err, check.Equals, nil)
	c.Assert(resp2.StatusCode, check.Equals, http.StatusOK,
		check.Commentf("Wasn't able to get data from the RailsAPI at %q", url))
	defer resp2.Body.Close()
	db, err := ioutil.ReadAll(resp2.Body)
	c.Check(err, check.Equals, nil)
	err = json.Unmarshal(db, &direct)
	c.Check(err, check.Equals, nil)

	// Check that all RailsAPI provided keys exist on the controller response.
	for k := range direct {
		if _, ok := skippedFields[k]; ok {
			continue
		} else if val, ok := proxied[k]; !ok {
			c.Errorf("%s's key %q missing on controller's response.", direct["kind"], k)
		} else if direct["kind"] == "arvados#collection" && k == "manifest_text" {
			// Tokens differ from request to request
			c.Check(strings.Split(val.(string), "+A")[0], check.Equals, strings.Split(direct[k].(string), "+A")[0])
		} else {
			c.Check(val, check.DeepEquals, direct[k],
				check.Commentf("RailsAPI %s key %q's value %q differs from controller's %q.", direct["kind"], k, direct[k], val))
		}
	}
}

func (s *HandlerSuite) TestGetObjects(c *check.C) {
	// Get the 1st keep service's uuid from the running test server.
	req := httptest.NewRequest("GET", "/arvados/v1/keep_services/", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.AdminToken)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Assert(resp.Code, check.Equals, http.StatusOK)
	var ksList arvados.KeepServiceList
	json.Unmarshal(resp.Body.Bytes(), &ksList)
	c.Assert(len(ksList.Items), check.Not(check.Equals), 0)
	ksUUID := ksList.Items[0].UUID
	// Create a new token for the test user so that we're not comparing
	// the ones from the fixtures.
	req = httptest.NewRequest("POST", "/arvados/v1/api_client_authorizations",
		strings.NewReader(`{
			"api_client_authorization": {
				"owner_uuid": "`+arvadostest.AdminUserUUID+`",
				"created_by_ip_address": "::1",
				"last_used_by_ip_address": "::1",
				"default_owner_uuid": "`+arvadostest.AdminUserUUID+`"
			}
		}`))
	req.Header.Set("Authorization", "Bearer "+arvadostest.SystemRootToken)
	req.Header.Set("Content-type", "application/json")
	resp = httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Assert(resp.Code, check.Equals, http.StatusOK,
		check.Commentf("%s", resp.Body.String()))
	var auth arvados.APIClientAuthorization
	json.Unmarshal(resp.Body.Bytes(), &auth)
	c.Assert(auth.UUID, check.Not(check.Equals), "")

	testCases := map[string]map[string]bool{
		"api_clients/" + arvadostest.TrustedWorkbenchAPIClientUUID:     nil,
		"api_client_authorizations/" + auth.UUID:                       {"href": true, "modified_by_client_uuid": true, "modified_by_user_uuid": true},
		"authorized_keys/" + arvadostest.AdminAuthorizedKeysUUID:       nil,
		"collections/" + arvadostest.CollectionWithUniqueWordsUUID:     {"href": true},
		"containers/" + arvadostest.RunningContainerUUID:               nil,
		"container_requests/" + arvadostest.QueuedContainerRequestUUID: nil,
		"groups/" + arvadostest.AProjectUUID:                           nil,
		"keep_services/" + ksUUID:                                      nil,
		"links/" + arvadostest.ActiveUserCanReadAllUsersLinkUUID:       nil,
		"logs/" + arvadostest.CrunchstatForRunningJobLogUUID:           nil,
		"nodes/" + arvadostest.IdleNodeUUID:                            nil,
		"repositories/" + arvadostest.ArvadosRepoUUID:                  nil,
		"users/" + arvadostest.ActiveUserUUID:                          {"href": true},
		"virtual_machines/" + arvadostest.TestVMUUID:                   nil,
		"workflows/" + arvadostest.WorkflowWithDefinitionYAMLUUID:      nil,
	}
	for url, skippedFields := range testCases {
		c.Logf("Testing %q", url)
		s.CheckObjectType(c, "/arvados/v1/"+url, auth.TokenV2(), skippedFields)
	}
}

func (s *HandlerSuite) TestRedactRailsAPIHostFromErrors(c *check.C) {
	req := httptest.NewRequest("GET", "https://0.0.0.0:1/arvados/v1/collections/zzzzz-4zz18-abcdefghijklmno", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	var jresp struct {
		Errors []string
	}
	c.Log(resp.Body.String())
	c.Assert(json.NewDecoder(resp.Body).Decode(&jresp), check.IsNil)
	c.Assert(jresp.Errors, check.HasLen, 1)
	c.Check(jresp.Errors[0], check.Matches, `.*//railsapi\.internal/arvados/v1/collections/.*: 404 Not Found.*`)
	c.Check(jresp.Errors[0], check.Not(check.Matches), `(?ms).*127.0.0.1.*`)
}

func (s *HandlerSuite) TestTrashSweep(c *check.C) {
	s.cluster.SystemRootToken = arvadostest.SystemRootToken
	s.cluster.Collections.TrashSweepInterval = arvados.Duration(time.Second / 10)
	s.handler.CheckHealth()
	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})
	coll, err := s.handler.federation.CollectionCreate(ctx, arvados.CreateOptions{Attrs: map[string]interface{}{"name": "test trash sweep"}, EnsureUniqueName: true})
	c.Assert(err, check.IsNil)
	defer s.handler.federation.CollectionDelete(ctx, arvados.DeleteOptions{UUID: coll.UUID})
	db, err := s.handler.dbConnector.GetDB(s.ctx)
	c.Assert(err, check.IsNil)
	_, err = db.ExecContext(s.ctx, `update collections set trash_at = $1, delete_at = $2 where uuid = $3`, time.Now().UTC().Add(time.Second/10), time.Now().UTC().Add(time.Hour), coll.UUID)
	c.Assert(err, check.IsNil)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if time.Now().After(deadline) {
			c.Log("timed out")
			c.FailNow()
		}
		updated, err := s.handler.federation.CollectionGet(ctx, arvados.GetOptions{UUID: coll.UUID, IncludeTrash: true})
		c.Assert(err, check.IsNil)
		if updated.IsTrashed {
			break
		}
		time.Sleep(time.Second / 10)
	}
}

func (s *HandlerSuite) TestContainerLogSweep(c *check.C) {
	s.cluster.SystemRootToken = arvadostest.SystemRootToken
	s.cluster.Containers.Logging.SweepInterval = arvados.Duration(time.Second / 10)
	s.handler.CheckHealth()
	ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})
	logentry, err := s.handler.federation.LogCreate(ctx, arvados.CreateOptions{Attrs: map[string]interface{}{
		"object_uuid": arvadostest.CompletedContainerUUID,
		"event_type":  "stderr",
		"properties": map[string]interface{}{
			"text": "test trash sweep\n",
		},
	}})
	c.Assert(err, check.IsNil)
	defer s.handler.federation.LogDelete(ctx, arvados.DeleteOptions{UUID: logentry.UUID})
	deadline := time.Now().Add(5 * time.Second)
	for {
		if time.Now().After(deadline) {
			c.Log("timed out")
			c.FailNow()
		}
		logentries, err := s.handler.federation.LogList(ctx, arvados.ListOptions{Filters: []arvados.Filter{{"uuid", "=", logentry.UUID}}, Limit: -1})
		c.Assert(err, check.IsNil)
		if len(logentries.Items) == 0 {
			break
		}
		time.Sleep(time.Second / 10)
	}
}

func (s *HandlerSuite) TestLogActivity(c *check.C) {
	s.cluster.SystemRootToken = arvadostest.SystemRootToken
	s.cluster.Users.ActivityLoggingPeriod = arvados.Duration(24 * time.Hour)
	s.handler.CheckHealth()

	testServer := newServerFromIntegrationTestEnv(c)
	testServer.Server.Handler = httpserver.AddRequestIDs(httpserver.LogRequests(s.handler))
	c.Assert(testServer.Start(), check.IsNil)
	defer testServer.Close()

	u, _ := url.Parse("http://" + testServer.Addr)
	client := rpc.NewConn(s.cluster.ClusterID, u, true, rpc.PassthroughTokenProvider)

	starttime := time.Now()
	for i := 0; i < 4; i++ {
		for _, token := range []string{
			arvadostest.ActiveTokenV2,
			arvadostest.ActiveToken,
			arvadostest.SpectatorToken,
		} {
			ctx := auth.NewContext(s.ctx, &auth.Credentials{Tokens: []string{token}})
			_, err := client.CollectionList(ctx, arvados.ListOptions{})
			c.Assert(err, check.IsNil)
		}
	}
	db, err := s.handler.dbConnector.GetDB(s.ctx)
	c.Assert(err, check.IsNil)
	for _, userUUID := range []string{arvadostest.ActiveUserUUID, arvadostest.SpectatorUserUUID} {
		var rows int
		err = db.QueryRowContext(s.ctx, `select count(uuid) from logs where object_uuid = $1 and event_at > $2`, arvadostest.ActiveUserUUID, starttime.UTC()).Scan(&rows)
		c.Assert(err, check.IsNil)
		c.Check(rows, check.Equals, 1, check.Commentf("expect 1 row for user uuid %s", userUUID))
	}
}

func (s *HandlerSuite) TestLogLimiting(c *check.C) {
	s.handler.Cluster.API.MaxConcurrentRequests = 2
	s.handler.Cluster.API.LogCreateRequestFraction = 0.5

	logreq := httptest.NewRequest("POST", "/arvados/v1/logs", strings.NewReader(`{
			"log": {
                          "event_type": "test"
			}
		}`))
	logreq.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)

	// Log create succeeds
	for i := 0; i < 2; i++ {
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, logreq)
		c.Check(resp.Code, check.Equals, http.StatusOK)
		var lg arvados.Log
		err := json.Unmarshal(resp.Body.Bytes(), &lg)
		c.Check(err, check.IsNil)
		c.Check(lg.UUID, check.Matches, "zzzzz-57u5n-.*")
	}

	// Pretend there's a log create in flight
	s.handler.limitLogCreate <- struct{}{}

	// Log create should be rejected now
	resp := httptest.NewRecorder()
	s.handler.ServeHTTP(resp, logreq)
	c.Check(resp.Code, check.Equals, http.StatusServiceUnavailable)

	// Other requests still succeed
	req := httptest.NewRequest("GET", "/arvados/v1/users/current", nil)
	req.Header.Set("Authorization", "Bearer "+arvadostest.ActiveToken)
	resp = httptest.NewRecorder()
	s.handler.ServeHTTP(resp, req)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var u arvados.User
	err := json.Unmarshal(resp.Body.Bytes(), &u)
	c.Check(err, check.IsNil)
	c.Check(u.UUID, check.Equals, arvadostest.ActiveUserUUID)

	// log create still fails
	resp = httptest.NewRecorder()
	s.handler.ServeHTTP(resp, logreq)
	c.Check(resp.Code, check.Equals, http.StatusServiceUnavailable)

	// Pretend in-flight log is done
	<-s.handler.limitLogCreate

	// log create succeeds again
	resp = httptest.NewRecorder()
	s.handler.ServeHTTP(resp, logreq)
	c.Check(resp.Code, check.Equals, http.StatusOK)

}
