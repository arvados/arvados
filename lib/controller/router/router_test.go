// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&RouterSuite{})

type RouterSuite struct {
	rtr *router
}

func (s *RouterSuite) SetUpTest(c *check.C) {
	cluster := &arvados.Cluster{
		TLS: arvados.TLS{Insecure: true},
	}
	arvadostest.SetServiceURL(&cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	s.rtr = New(cluster)
}

func (s *RouterSuite) TearDownTest(c *check.C) {
	err := arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil)
	c.Check(err, check.IsNil)
}

func (s *RouterSuite) doRequest(c *check.C, token, method, path string, hdrs http.Header, body io.Reader) (*http.Request, *httptest.ResponseRecorder, map[string]interface{}) {
	req := httptest.NewRequest(method, path, body)
	for k, v := range hdrs {
		req.Header[k] = v
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	s.rtr.ServeHTTP(rr, req)
	c.Logf("response body: %s", rr.Body.String())
	var jresp map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	return req, rr, jresp
}

func (s *RouterSuite) TestCollectionResponses(c *check.C) {
	token := arvadostest.ActiveTokenV2

	// Check "get collection" response has "kind" key
	_, rr, jresp := s.doRequest(c, token, "GET", `/arvados/v1/collections`, nil, bytes.NewBufferString(`{"include_trash":true}`))
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items"], check.FitsTypeOf, []interface{}{})
	c.Check(jresp["kind"], check.Equals, "arvados#collectionList")
	c.Check(jresp["items"].([]interface{})[0].(map[string]interface{})["kind"], check.Equals, "arvados#collection")

	// Check items in list response have a "kind" key regardless
	// of whether a uuid/pdh is selected.
	for _, selectj := range []string{
		``,
		`,"select":["portable_data_hash"]`,
		`,"select":["name"]`,
		`,"select":["uuid"]`,
	} {
		_, rr, jresp = s.doRequest(c, token, "GET", `/arvados/v1/collections`, nil, bytes.NewBufferString(`{"where":{"uuid":["`+arvadostest.FooCollection+`"]}`+selectj+`}`))
		c.Check(rr.Code, check.Equals, http.StatusOK)
		c.Check(jresp["items"], check.FitsTypeOf, []interface{}{})
		c.Check(jresp["items_available"], check.FitsTypeOf, float64(0))
		c.Check(jresp["kind"], check.Equals, "arvados#collectionList")
		item0 := jresp["items"].([]interface{})[0].(map[string]interface{})
		c.Check(item0["kind"], check.Equals, "arvados#collection")
		if selectj == "" || strings.Contains(selectj, "portable_data_hash") {
			c.Check(item0["portable_data_hash"], check.Equals, arvadostest.FooCollectionPDH)
		} else {
			c.Check(item0["portable_data_hash"], check.IsNil)
		}
		if selectj == "" || strings.Contains(selectj, "name") {
			c.Check(item0["name"], check.FitsTypeOf, "")
		} else {
			c.Check(item0["name"], check.IsNil)
		}
		if selectj == "" || strings.Contains(selectj, "uuid") {
			c.Check(item0["uuid"], check.Equals, arvadostest.FooCollection)
		} else {
			c.Check(item0["uuid"], check.IsNil)
		}
	}

	// Check "create collection" response has "kind" key
	_, rr, jresp = s.doRequest(c, token, "POST", `/arvados/v1/collections`, http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}, bytes.NewBufferString(`ensure_unique_name=true`))
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.FitsTypeOf, "")
	c.Check(jresp["kind"], check.Equals, "arvados#collection")
}

func (s *RouterSuite) TestContainerList(c *check.C) {
	token := arvadostest.ActiveTokenV2

	_, rr, jresp := s.doRequest(c, token, "GET", `/arvados/v1/containers?limit=0`, nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items_available"], check.FitsTypeOf, float64(0))
	c.Check(jresp["items_available"].(float64) > 2, check.Equals, true)
	c.Check(jresp["items"], check.HasLen, 0)

	_, rr, jresp = s.doRequest(c, token, "GET", `/arvados/v1/containers?limit=2&select=["uuid","command"]`, nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items_available"], check.FitsTypeOf, float64(0))
	c.Check(jresp["items_available"].(float64) > 2, check.Equals, true)
	c.Check(jresp["items"], check.HasLen, 2)
	item0 := jresp["items"].([]interface{})[0].(map[string]interface{})
	c.Check(item0["uuid"], check.HasLen, 27)
	c.Check(item0["command"], check.FitsTypeOf, []interface{}{})
	c.Check(item0["command"].([]interface{})[0], check.FitsTypeOf, "")
	c.Check(item0["mounts"], check.IsNil)

	_, rr, jresp = s.doRequest(c, token, "GET", `/arvados/v1/containers`, nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items_available"], check.FitsTypeOf, float64(0))
	c.Check(jresp["items_available"].(float64) > 2, check.Equals, true)
	avail := int(jresp["items_available"].(float64))
	c.Check(jresp["items"], check.HasLen, avail)
	item0 = jresp["items"].([]interface{})[0].(map[string]interface{})
	c.Check(item0["uuid"], check.HasLen, 27)
	c.Check(item0["command"], check.FitsTypeOf, []interface{}{})
	c.Check(item0["command"].([]interface{})[0], check.FitsTypeOf, "")
	c.Check(item0["mounts"], check.NotNil)
}

func (s *RouterSuite) TestContainerLock(c *check.C) {
	uuid := arvadostest.QueuedContainerUUID
	token := arvadostest.AdminToken
	_, rr, jresp := s.doRequest(c, token, "POST", "/arvados/v1/containers/"+uuid+"/lock", nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.HasLen, 27)
	c.Check(jresp["state"], check.Equals, "Locked")
	_, rr, jresp = s.doRequest(c, token, "POST", "/arvados/v1/containers/"+uuid+"/lock", nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusUnprocessableEntity)
	c.Check(rr.Body.String(), check.Not(check.Matches), `.*"uuid":.*`)
	_, rr, jresp = s.doRequest(c, token, "POST", "/arvados/v1/containers/"+uuid+"/unlock", nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.HasLen, 27)
	c.Check(jresp["state"], check.Equals, "Queued")
	c.Check(jresp["environment"], check.IsNil)
	_, rr, jresp = s.doRequest(c, token, "POST", "/arvados/v1/containers/"+uuid+"/unlock", nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusUnprocessableEntity)
	c.Check(jresp["uuid"], check.IsNil)
}

func (s *RouterSuite) TestFullTimestampsInResponse(c *check.C) {
	uuid := arvadostest.CollectionReplicationDesired2Confirmed2UUID
	token := arvadostest.ActiveTokenV2

	_, rr, jresp := s.doRequest(c, token, "GET", `/arvados/v1/collections/`+uuid, nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.Equals, uuid)
	expectNS := map[string]int{
		"created_at":  596506000, // fixture says 596506247, but truncated by postgresql
		"modified_at": 596338000, // fixture says 596338465, but truncated by postgresql
	}
	for key, ns := range expectNS {
		mt, ok := jresp[key].(string)
		c.Logf("jresp[%q] == %q", key, mt)
		c.Assert(ok, check.Equals, true)
		t, err := time.Parse(time.RFC3339Nano, mt)
		c.Check(err, check.IsNil)
		c.Check(t.Nanosecond(), check.Equals, ns)
	}
}

func (s *RouterSuite) TestSelectParam(c *check.C) {
	uuid := arvadostest.QueuedContainerUUID
	token := arvadostest.ActiveTokenV2
	for _, sel := range [][]string{
		{"uuid", "command"},
		{"uuid", "command", "uuid"},
		{"", "command", "uuid"},
	} {
		j, err := json.Marshal(sel)
		c.Assert(err, check.IsNil)
		_, rr, resp := s.doRequest(c, token, "GET", "/arvados/v1/containers/"+uuid+"?select="+string(j), nil, nil)
		c.Check(rr.Code, check.Equals, http.StatusOK)

		c.Check(resp["kind"], check.Equals, "arvados#container")
		c.Check(resp["uuid"], check.HasLen, 27)
		c.Check(resp["command"], check.HasLen, 2)
		c.Check(resp["mounts"], check.IsNil)
		_, hasMounts := resp["mounts"]
		c.Check(hasMounts, check.Equals, false)
	}
}

func (s *RouterSuite) TestCORS(c *check.C) {
	token := arvadostest.ActiveTokenV2
	req := (&testReq{
		method: "OPTIONS",
		path:   "arvados/v1/collections/" + arvadostest.FooCollection,
		header: http.Header{"Origin": {"https://example.com"}},
		token:  token,
	}).Request()
	rr := httptest.NewRecorder()
	s.rtr.ServeHTTP(rr, req)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(rr.Body.String(), check.HasLen, 0)
	c.Check(rr.Result().Header.Get("Access-Control-Allow-Origin"), check.Equals, "*")
	for _, hdr := range []string{"Authorization", "Content-Type"} {
		c.Check(rr.Result().Header.Get("Access-Control-Allow-Headers"), check.Matches, ".*"+hdr+".*")
	}
	for _, method := range []string{"GET", "HEAD", "PUT", "POST", "DELETE"} {
		c.Check(rr.Result().Header.Get("Access-Control-Allow-Methods"), check.Matches, ".*"+method+".*")
	}

	for _, unsafe := range []string{"login", "logout", "auth", "auth/foo", "login/?blah"} {
		req := (&testReq{
			method: "OPTIONS",
			path:   unsafe,
			header: http.Header{"Origin": {"https://example.com"}},
			token:  token,
		}).Request()
		rr := httptest.NewRecorder()
		s.rtr.ServeHTTP(rr, req)
		c.Check(rr.Code, check.Equals, http.StatusOK)
		c.Check(rr.Body.String(), check.HasLen, 0)
		c.Check(rr.Result().Header.Get("Access-Control-Allow-Origin"), check.Equals, "")
		c.Check(rr.Result().Header.Get("Access-Control-Allow-Methods"), check.Equals, "")
		c.Check(rr.Result().Header.Get("Access-Control-Allow-Headers"), check.Equals, "")

		req = (&testReq{
			method: "POST",
			path:   unsafe,
			header: http.Header{"Origin": {"https://example.com"}},
			token:  token,
		}).Request()
		rr = httptest.NewRecorder()
		s.rtr.ServeHTTP(rr, req)
		c.Check(rr.Result().Header.Get("Access-Control-Allow-Origin"), check.Equals, "")
		c.Check(rr.Result().Header.Get("Access-Control-Allow-Methods"), check.Equals, "")
		c.Check(rr.Result().Header.Get("Access-Control-Allow-Headers"), check.Equals, "")
	}
}
