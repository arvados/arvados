// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
	rw := httptest.NewRecorder()
	s.rtr.ServeHTTP(rw, req)
	c.Logf("response body: %s", rw.Body.String())
	var jresp map[string]interface{}
	err := json.Unmarshal(rw.Body.Bytes(), &jresp)
	c.Check(err, check.IsNil)
	return req, rw, jresp
}

func (s *RouterSuite) TestContainerList(c *check.C) {
	token := arvadostest.ActiveTokenV2

	_, rw, jresp := s.doRequest(c, token, "GET", `/arvados/v1/containers?limit=0`, nil, nil)
	c.Check(rw.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items_available"], check.FitsTypeOf, float64(0))
	c.Check(jresp["items_available"].(float64) > 2, check.Equals, true)
	c.Check(jresp["items"], check.HasLen, 0)

	_, rw, jresp = s.doRequest(c, token, "GET", `/arvados/v1/containers?limit=2&select=["uuid","command"]`, nil, nil)
	c.Check(rw.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items_available"], check.FitsTypeOf, float64(0))
	c.Check(jresp["items_available"].(float64) > 2, check.Equals, true)
	c.Check(jresp["items"], check.HasLen, 2)
	item0 := jresp["items"].([]interface{})[0].(map[string]interface{})
	c.Check(item0["uuid"], check.HasLen, 27)
	c.Check(item0["command"], check.FitsTypeOf, []interface{}{})
	c.Check(item0["command"].([]interface{})[0], check.FitsTypeOf, "")
	c.Check(item0["mounts"], check.IsNil)

	_, rw, jresp = s.doRequest(c, token, "GET", `/arvados/v1/containers`, nil, nil)
	c.Check(rw.Code, check.Equals, http.StatusOK)
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
	token := arvadostest.ActiveTokenV2
	_, rw, jresp := s.doRequest(c, token, "POST", "/arvados/v1/containers/"+uuid+"/lock", nil, nil)
	c.Check(rw.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.HasLen, 27)
	c.Check(jresp["state"], check.Equals, "Locked")
	_, rw, jresp = s.doRequest(c, token, "POST", "/arvados/v1/containers/"+uuid+"/lock", nil, nil)
	c.Check(rw.Code, check.Equals, http.StatusUnprocessableEntity)
	c.Check(rw.Body.String(), check.Not(check.Matches), `.*"uuid":.*`)
	_, rw, jresp = s.doRequest(c, token, "POST", "/arvados/v1/containers/"+uuid+"/unlock", nil, nil)
	c.Check(rw.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.HasLen, 27)
	c.Check(jresp["state"], check.Equals, "Queued")
	c.Check(jresp["environment"], check.IsNil)
	_, rw, jresp = s.doRequest(c, token, "POST", "/arvados/v1/containers/"+uuid+"/unlock", nil, nil)
	c.Check(rw.Code, check.Equals, http.StatusUnprocessableEntity)
	c.Check(jresp["uuid"], check.IsNil)
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
		_, rw, resp := s.doRequest(c, token, "GET", "/arvados/v1/containers/"+uuid+"?select="+string(j), nil, nil)
		c.Check(rw.Code, check.Equals, http.StatusOK)

		c.Check(resp["uuid"], check.HasLen, 27)
		c.Check(resp["command"], check.HasLen, 2)
		c.Check(resp["mounts"], check.IsNil)
		_, hasMounts := resp["mounts"]
		c.Check(hasMounts, check.Equals, false)
	}
}
