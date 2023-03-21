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
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"github.com/gorilla/mux"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&RouterSuite{})

type RouterSuite struct {
	rtr  *router
	stub arvadostest.APIStub
}

func (s *RouterSuite) SetUpTest(c *check.C) {
	s.stub = arvadostest.APIStub{}
	s.rtr = &router{
		mux:     mux.NewRouter(),
		backend: &s.stub,
	}
	s.rtr.addRoutes()
}

func (s *RouterSuite) TestOptions(c *check.C) {
	token := arvadostest.ActiveToken
	for _, trial := range []struct {
		comment      string // unparsed -- only used to help match test failures to trials
		method       string
		path         string
		header       http.Header
		body         string
		shouldStatus int // zero value means 200
		shouldCall   string
		withOptions  interface{}
	}{
		{
			method:      "GET",
			path:        "/arvados/v1/collections/" + arvadostest.FooCollection,
			shouldCall:  "CollectionGet",
			withOptions: arvados.GetOptions{UUID: arvadostest.FooCollection},
		},
		{
			method:      "PUT",
			path:        "/arvados/v1/collections/" + arvadostest.FooCollection,
			shouldCall:  "CollectionUpdate",
			withOptions: arvados.UpdateOptions{UUID: arvadostest.FooCollection},
		},
		{
			method:      "PATCH",
			path:        "/arvados/v1/collections/" + arvadostest.FooCollection,
			shouldCall:  "CollectionUpdate",
			withOptions: arvados.UpdateOptions{UUID: arvadostest.FooCollection},
		},
		{
			method:      "DELETE",
			path:        "/arvados/v1/collections/" + arvadostest.FooCollection,
			shouldCall:  "CollectionDelete",
			withOptions: arvados.DeleteOptions{UUID: arvadostest.FooCollection},
		},
		{
			method:      "POST",
			path:        "/arvados/v1/collections",
			shouldCall:  "CollectionCreate",
			withOptions: arvados.CreateOptions{},
		},
		{
			method:      "GET",
			path:        "/arvados/v1/collections",
			shouldCall:  "CollectionList",
			withOptions: arvados.ListOptions{Limit: -1},
		},
		{
			method:      "GET",
			path:        "/arvados/v1/api_client_authorizations",
			shouldCall:  "APIClientAuthorizationList",
			withOptions: arvados.ListOptions{Limit: -1},
		},
		{
			method:      "GET",
			path:        "/arvados/v1/collections?limit=123&offset=456&include_trash=true&include_old_versions=1",
			shouldCall:  "CollectionList",
			withOptions: arvados.ListOptions{Limit: 123, Offset: 456, IncludeTrash: true, IncludeOldVersions: true},
		},
		{
			method:      "POST",
			path:        "/arvados/v1/collections?limit=123&_method=GET",
			body:        `{"offset":456,"include_trash":true,"include_old_versions":true}`,
			shouldCall:  "CollectionList",
			withOptions: arvados.ListOptions{Limit: 123, Offset: 456, IncludeTrash: true, IncludeOldVersions: true},
		},
		{
			method:      "POST",
			path:        "/arvados/v1/collections?limit=123",
			body:        `{"offset":456,"include_trash":true,"include_old_versions":true}`,
			header:      http.Header{"X-Http-Method-Override": {"GET"}, "Content-Type": {"application/json"}},
			shouldCall:  "CollectionList",
			withOptions: arvados.ListOptions{Limit: 123, Offset: 456, IncludeTrash: true, IncludeOldVersions: true},
		},
		{
			method:      "POST",
			path:        "/arvados/v1/collections?limit=123",
			body:        "offset=456&include_trash=true&include_old_versions=1&_method=GET",
			header:      http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
			shouldCall:  "CollectionList",
			withOptions: arvados.ListOptions{Limit: 123, Offset: 456, IncludeTrash: true, IncludeOldVersions: true},
		},
		{
			comment:     "form-encoded expression filter in query string",
			method:      "GET",
			path:        "/arvados/v1/collections?filters=[%22(foo<bar)%22]",
			shouldCall:  "CollectionList",
			withOptions: arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"(foo<bar)", "=", true}}},
		},
		{
			comment:     "form-encoded expression filter in POST body",
			method:      "POST",
			path:        "/arvados/v1/collections",
			body:        "filters=[\"(foo<bar)\"]&_method=GET",
			header:      http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
			shouldCall:  "CollectionList",
			withOptions: arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"(foo<bar)", "=", true}}},
		},
		{
			comment:     "json-encoded expression filter in POST body",
			method:      "POST",
			path:        "/arvados/v1/collections?_method=GET",
			body:        `{"filters":["(foo<bar)",["bar","=","baz"]],"limit":2}`,
			header:      http.Header{"Content-Type": {"application/json"}},
			shouldCall:  "CollectionList",
			withOptions: arvados.ListOptions{Limit: 2, Filters: []arvados.Filter{{"(foo<bar)", "=", true}, {"bar", "=", "baz"}}},
		},
		{
			comment:     "json-encoded select param in query string",
			method:      "GET",
			path:        "/arvados/v1/collections/" + arvadostest.FooCollection + "?select=[%22portable_data_hash%22]",
			shouldCall:  "CollectionGet",
			withOptions: arvados.GetOptions{UUID: arvadostest.FooCollection, Select: []string{"portable_data_hash"}},
		},
		{
			method:       "PATCH",
			path:         "/arvados/v1/collections",
			shouldStatus: http.StatusMethodNotAllowed,
		},
		{
			method:       "PUT",
			path:         "/arvados/v1/collections",
			shouldStatus: http.StatusMethodNotAllowed,
		},
		{
			method:       "DELETE",
			path:         "/arvados/v1/collections",
			shouldStatus: http.StatusMethodNotAllowed,
		},
		{
			comment:    "container log webdav GET root",
			method:     "GET",
			path:       "/arvados/v1/containers/" + arvadostest.CompletedContainerUUID + "/log/",
			shouldCall: "ContainerLog",
			withOptions: arvados.ContainerLogOptions{
				UUID: arvadostest.CompletedContainerUUID,
				WebDAVOptions: arvados.WebDAVOptions{
					Method: "GET",
					Header: http.Header{"Authorization": {"Bearer " + arvadostest.ActiveToken}},
					Path:   "/"}},
		},
		{
			comment:    "container log webdav GET root without trailing slash",
			method:     "GET",
			path:       "/arvados/v1/containers/" + arvadostest.CompletedContainerUUID + "/log",
			shouldCall: "ContainerLog",
			withOptions: arvados.ContainerLogOptions{
				UUID: arvadostest.CompletedContainerUUID,
				WebDAVOptions: arvados.WebDAVOptions{
					Method: "GET",
					Header: http.Header{"Authorization": {"Bearer " + arvadostest.ActiveToken}},
					Path:   ""}},
		},
		{
			comment:    "container log webdav OPTIONS root",
			method:     "OPTIONS",
			path:       "/arvados/v1/containers/" + arvadostest.CompletedContainerUUID + "/log/",
			shouldCall: "ContainerLog",
			withOptions: arvados.ContainerLogOptions{
				UUID: arvadostest.CompletedContainerUUID,
				WebDAVOptions: arvados.WebDAVOptions{
					Method: "OPTIONS",
					Header: http.Header{"Authorization": {"Bearer " + arvadostest.ActiveToken}},
					Path:   "/"}},
		},
		{
			comment:    "container log webdav OPTIONS root without trailing slash",
			method:     "OPTIONS",
			path:       "/arvados/v1/containers/" + arvadostest.CompletedContainerUUID + "/log",
			shouldCall: "ContainerLog",
			withOptions: arvados.ContainerLogOptions{
				UUID: arvadostest.CompletedContainerUUID,
				WebDAVOptions: arvados.WebDAVOptions{
					Method: "OPTIONS",
					Header: http.Header{"Authorization": {"Bearer " + arvadostest.ActiveToken}},
					Path:   ""}},
		},
		{
			comment:    "container log webdav PROPFIND root",
			method:     "PROPFIND",
			path:       "/arvados/v1/containers/" + arvadostest.CompletedContainerUUID + "/log/",
			shouldCall: "ContainerLog",
			withOptions: arvados.ContainerLogOptions{
				UUID: arvadostest.CompletedContainerUUID,
				WebDAVOptions: arvados.WebDAVOptions{
					Method: "PROPFIND",
					Header: http.Header{"Authorization": {"Bearer " + arvadostest.ActiveToken}},
					Path:   "/"}},
		},
		{
			comment:    "container log webdav PROPFIND root without trailing slash",
			method:     "PROPFIND",
			path:       "/arvados/v1/containers/" + arvadostest.CompletedContainerUUID + "/log",
			shouldCall: "ContainerLog",
			withOptions: arvados.ContainerLogOptions{
				UUID: arvadostest.CompletedContainerUUID,
				WebDAVOptions: arvados.WebDAVOptions{
					Method: "PROPFIND",
					Header: http.Header{"Authorization": {"Bearer " + arvadostest.ActiveToken}},
					Path:   ""}},
		},
		{
			comment:    "container log webdav no_forward=true",
			method:     "GET",
			path:       "/arvados/v1/containers/" + arvadostest.CompletedContainerUUID + "/log/?no_forward=true",
			shouldCall: "ContainerLog",
			withOptions: arvados.ContainerLogOptions{
				UUID:      arvadostest.CompletedContainerUUID,
				NoForward: true,
				WebDAVOptions: arvados.WebDAVOptions{
					Method: "GET",
					Header: http.Header{"Authorization": {"Bearer " + arvadostest.ActiveToken}},
					Path:   "/"}},
		},
	} {
		// Reset calls captured in previous trial
		s.stub = arvadostest.APIStub{}

		c.Logf("trial: %+v", trial)
		comment := check.Commentf("trial comment: %s", trial.comment)

		_, rr := doRequest(c, s.rtr, token, trial.method, trial.path, trial.header, bytes.NewBufferString(trial.body), nil)
		if trial.shouldStatus == 0 {
			c.Check(rr.Code, check.Equals, http.StatusOK, comment)
		} else {
			c.Check(rr.Code, check.Equals, trial.shouldStatus, comment)
		}
		calls := s.stub.Calls(nil)
		if trial.shouldCall == "" {
			c.Check(calls, check.HasLen, 0, comment)
		} else if len(calls) != 1 {
			c.Check(calls, check.HasLen, 1, comment)
		} else {
			c.Check(calls[0].Method, isMethodNamed, trial.shouldCall, comment)
			c.Check(calls[0].Options, check.DeepEquals, trial.withOptions, comment)
		}
	}
}

var _ = check.Suite(&RouterIntegrationSuite{})

type RouterIntegrationSuite struct {
	rtr *router
}

func (s *RouterIntegrationSuite) SetUpTest(c *check.C) {
	cluster := &arvados.Cluster{}
	cluster.TLS.Insecure = true
	arvadostest.SetServiceURL(&cluster.Services.RailsAPI, "https://"+os.Getenv("ARVADOS_TEST_API_HOST"))
	url, _ := url.Parse("https://" + os.Getenv("ARVADOS_TEST_API_HOST"))
	s.rtr = New(rpc.NewConn("zzzzz", url, true, rpc.PassthroughTokenProvider), Config{})
}

func (s *RouterIntegrationSuite) TearDownSuite(c *check.C) {
	err := arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil)
	c.Check(err, check.IsNil)
}

func (s *RouterIntegrationSuite) TestCollectionResponses(c *check.C) {
	token := arvadostest.ActiveTokenV2

	// Check "get collection" response has "kind" key
	jresp := map[string]interface{}{}
	_, rr := doRequest(c, s.rtr, token, "GET", `/arvados/v1/collections`, nil, bytes.NewBufferString(`{"include_trash":true}`), jresp)
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
		jresp := map[string]interface{}{}
		_, rr = doRequest(c, s.rtr, token, "GET", `/arvados/v1/collections`, nil, bytes.NewBufferString(`{"where":{"uuid":["`+arvadostest.FooCollection+`"]}`+selectj+`}`), jresp)
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
	jresp = map[string]interface{}{}
	_, rr = doRequest(c, s.rtr, token, "POST", `/arvados/v1/collections`, http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}, bytes.NewBufferString(`ensure_unique_name=true`), jresp)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.FitsTypeOf, "")
	c.Check(jresp["kind"], check.Equals, "arvados#collection")
}

func (s *RouterIntegrationSuite) TestMaxRequestSize(c *check.C) {
	token := arvadostest.ActiveTokenV2
	for _, maxRequestSize := range []int{
		// Ensure 5M limit is enforced.
		5000000,
		// Ensure 50M limit is enforced, and that a >25M body
		// is accepted even though the default Go request size
		// limit is 10M.
		50000000,
	} {
		s.rtr.config.MaxRequestSize = maxRequestSize
		okstr := "a"
		for len(okstr) < maxRequestSize/2 {
			okstr = okstr + okstr
		}

		hdr := http.Header{"Content-Type": {"application/x-www-form-urlencoded"}}

		body := bytes.NewBufferString(url.Values{"foo_bar": {okstr}}.Encode())
		_, rr := doRequest(c, s.rtr, token, "POST", `/arvados/v1/collections`, hdr, body, nil)
		c.Check(rr.Code, check.Equals, http.StatusOK)

		body = bytes.NewBufferString(url.Values{"foo_bar": {okstr + okstr}}.Encode())
		_, rr = doRequest(c, s.rtr, token, "POST", `/arvados/v1/collections`, hdr, body, nil)
		c.Check(rr.Code, check.Equals, http.StatusRequestEntityTooLarge)
	}
}

func (s *RouterIntegrationSuite) TestContainerList(c *check.C) {
	token := arvadostest.ActiveTokenV2

	jresp := map[string]interface{}{}
	_, rr := doRequest(c, s.rtr, token, "GET", `/arvados/v1/containers?limit=0`, nil, nil, jresp)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items_available"], check.FitsTypeOf, float64(0))
	c.Check(jresp["items_available"].(float64) > 2, check.Equals, true)
	c.Check(jresp["items"], check.NotNil)
	c.Check(jresp["items"], check.HasLen, 0)

	jresp = map[string]interface{}{}
	_, rr = doRequest(c, s.rtr, token, "GET", `/arvados/v1/containers?filters=[["uuid","in",[]]]`, nil, nil, jresp)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items_available"], check.Equals, float64(0))
	c.Check(jresp["items"], check.NotNil)
	c.Check(jresp["items"], check.HasLen, 0)

	jresp = map[string]interface{}{}
	_, rr = doRequest(c, s.rtr, token, "GET", `/arvados/v1/containers?limit=2&select=["uuid","command"]`, nil, nil, jresp)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["items_available"], check.FitsTypeOf, float64(0))
	c.Check(jresp["items_available"].(float64) > 2, check.Equals, true)
	c.Check(jresp["items"], check.HasLen, 2)
	item0 := jresp["items"].([]interface{})[0].(map[string]interface{})
	c.Check(item0["uuid"], check.HasLen, 27)
	c.Check(item0["command"], check.FitsTypeOf, []interface{}{})
	c.Check(item0["command"].([]interface{})[0], check.FitsTypeOf, "")
	c.Check(item0["mounts"], check.IsNil)

	jresp = map[string]interface{}{}
	_, rr = doRequest(c, s.rtr, token, "GET", `/arvados/v1/containers`, nil, nil, jresp)
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

func (s *RouterIntegrationSuite) TestContainerLock(c *check.C) {
	uuid := arvadostest.QueuedContainerUUID
	token := arvadostest.AdminToken

	jresp := map[string]interface{}{}
	_, rr := doRequest(c, s.rtr, token, "POST", "/arvados/v1/containers/"+uuid+"/lock", nil, nil, jresp)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.HasLen, 27)
	c.Check(jresp["state"], check.Equals, "Locked")

	_, rr = doRequest(c, s.rtr, token, "POST", "/arvados/v1/containers/"+uuid+"/lock", nil, nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusUnprocessableEntity)
	c.Check(rr.Body.String(), check.Not(check.Matches), `.*"uuid":.*`)

	jresp = map[string]interface{}{}
	_, rr = doRequest(c, s.rtr, token, "POST", "/arvados/v1/containers/"+uuid+"/unlock", nil, nil, jresp)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["uuid"], check.HasLen, 27)
	c.Check(jresp["state"], check.Equals, "Queued")
	c.Check(jresp["environment"], check.IsNil)

	jresp = map[string]interface{}{}
	_, rr = doRequest(c, s.rtr, token, "POST", "/arvados/v1/containers/"+uuid+"/unlock", nil, nil, jresp)
	c.Check(rr.Code, check.Equals, http.StatusUnprocessableEntity)
	c.Check(jresp["uuid"], check.IsNil)
}

func (s *RouterIntegrationSuite) TestWritableBy(c *check.C) {
	jresp := map[string]interface{}{}
	_, rr := doRequest(c, s.rtr, arvadostest.ActiveTokenV2, "GET", `/arvados/v1/users/`+arvadostest.ActiveUserUUID, nil, nil, jresp)
	c.Check(rr.Code, check.Equals, http.StatusOK)
	c.Check(jresp["writable_by"], check.DeepEquals, []interface{}{"zzzzz-tpzed-000000000000000", "zzzzz-tpzed-xurymjxw79nv3jz", "zzzzz-j7d0g-48foin4vonvc2at"})
}

func (s *RouterIntegrationSuite) TestFullTimestampsInResponse(c *check.C) {
	uuid := arvadostest.CollectionReplicationDesired2Confirmed2UUID
	token := arvadostest.ActiveTokenV2

	jresp := map[string]interface{}{}
	_, rr := doRequest(c, s.rtr, token, "GET", `/arvados/v1/collections/`+uuid, nil, nil, jresp)
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

func (s *RouterIntegrationSuite) TestSelectParam(c *check.C) {
	uuid := arvadostest.QueuedContainerUUID
	token := arvadostest.ActiveTokenV2
	// GET
	for _, sel := range [][]string{
		{"uuid", "command"},
		{"uuid", "command", "uuid"},
	} {
		j, err := json.Marshal(sel)
		c.Assert(err, check.IsNil)
		jresp := map[string]interface{}{}
		_, rr := doRequest(c, s.rtr, token, "GET", "/arvados/v1/containers/"+uuid+"?select="+string(j), nil, nil, jresp)
		c.Check(rr.Code, check.Equals, http.StatusOK)

		c.Check(jresp["kind"], check.Equals, "arvados#container")
		c.Check(jresp["uuid"], check.HasLen, 27)
		c.Check(jresp["command"], check.HasLen, 2)
		c.Check(jresp["mounts"], check.IsNil)
		_, hasMounts := jresp["mounts"]
		c.Check(hasMounts, check.Equals, false)
	}
	// POST & PUT
	uuid = arvadostest.FooCollection
	j, err := json.Marshal([]string{"uuid", "description"})
	c.Assert(err, check.IsNil)
	for _, method := range []string{"PUT", "POST"} {
		desc := "Today is " + time.Now().String()
		reqBody := "{\"description\":\"" + desc + "\"}"
		jresp := map[string]interface{}{}
		var rr *httptest.ResponseRecorder
		if method == "PUT" {
			_, rr = doRequest(c, s.rtr, token, method, "/arvados/v1/collections/"+uuid+"?select="+string(j), nil, bytes.NewReader([]byte(reqBody)), jresp)
		} else {
			_, rr = doRequest(c, s.rtr, token, method, "/arvados/v1/collections?select="+string(j), nil, bytes.NewReader([]byte(reqBody)), jresp)
		}
		c.Check(rr.Code, check.Equals, http.StatusOK)
		c.Check(jresp["kind"], check.Equals, "arvados#collection")
		c.Check(jresp["uuid"], check.HasLen, 27)
		c.Check(jresp["description"], check.Equals, desc)
		c.Check(jresp["manifest_text"], check.IsNil)
	}
}

func (s *RouterIntegrationSuite) TestHEAD(c *check.C) {
	_, rr := doRequest(c, s.rtr, arvadostest.ActiveTokenV2, "HEAD", "/arvados/v1/containers/"+arvadostest.QueuedContainerUUID, nil, nil, nil)
	c.Check(rr.Code, check.Equals, http.StatusOK)
}

func (s *RouterIntegrationSuite) TestRouteNotFound(c *check.C) {
	token := arvadostest.ActiveTokenV2
	req := (&testReq{
		method: "POST",
		path:   "arvados/v1/collections/" + arvadostest.FooCollection + "/error404pls",
		token:  token,
	}).Request()
	rr := httptest.NewRecorder()
	s.rtr.ServeHTTP(rr, req)
	c.Check(rr.Code, check.Equals, http.StatusNotFound)
	c.Logf("body: %q", rr.Body.String())
	var j map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &j)
	c.Check(err, check.IsNil)
	c.Logf("decoded: %v", j)
	c.Assert(j["errors"], check.FitsTypeOf, []interface{}{})
	c.Check(j["errors"].([]interface{})[0], check.Equals, "API endpoint not found")
}

func (s *RouterIntegrationSuite) TestCORS(c *check.C) {
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
	for _, method := range []string{"GET", "HEAD", "PUT", "POST", "PATCH", "DELETE"} {
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

func doRequest(c *check.C, rtr http.Handler, token, method, path string, hdrs http.Header, body io.Reader, jresp map[string]interface{}) (*http.Request, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, body)
	for k, v := range hdrs {
		req.Header[k] = v
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	rtr.ServeHTTP(rr, req)
	c.Logf("response body: %s", rr.Body.String())
	if jresp != nil {
		err := json.Unmarshal(rr.Body.Bytes(), &jresp)
		c.Check(err, check.IsNil)
	}
	return req, rr
}
