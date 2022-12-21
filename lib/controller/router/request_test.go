// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

type testReq struct {
	method   string
	path     string
	token    string // default is ActiveTokenV2; use noToken to omit
	param    map[string]interface{}
	attrs    map[string]interface{}
	attrsKey string
	header   http.Header

	// variations on request formatting
	json            bool
	jsonAttrsTop    bool
	jsonStringParam bool
	tokenInBody     bool
	tokenInQuery    bool
	noContentType   bool

	body *bytes.Buffer
}

const noToken = "(no token)"

func (tr *testReq) Request() *http.Request {
	param := map[string]interface{}{}
	for k, v := range tr.param {
		param[k] = v
	}

	if tr.body != nil {
		// caller provided a buffer
	} else if tr.json {
		if tr.jsonAttrsTop {
			for k, v := range tr.attrs {
				if tr.jsonStringParam {
					j, err := json.Marshal(v)
					if err != nil {
						panic(err)
					}
					param[k] = string(j)
				} else {
					param[k] = v
				}
			}
		} else if tr.attrs != nil {
			if tr.jsonStringParam {
				j, err := json.Marshal(tr.attrs)
				if err != nil {
					panic(err)
				}
				param[tr.attrsKey] = string(j)
			} else {
				param[tr.attrsKey] = tr.attrs
			}
		}
		tr.body = bytes.NewBuffer(nil)
		err := json.NewEncoder(tr.body).Encode(param)
		if err != nil {
			panic(err)
		}
	} else {
		values := make(url.Values)
		for k, v := range param {
			if vs, ok := v.(string); ok && !tr.jsonStringParam {
				values.Set(k, vs)
			} else {
				jv, err := json.Marshal(v)
				if err != nil {
					panic(err)
				}
				values.Set(k, string(jv))
			}
		}
		if tr.attrs != nil {
			jattrs, err := json.Marshal(tr.attrs)
			if err != nil {
				panic(err)
			}
			values.Set(tr.attrsKey, string(jattrs))
		}
		tr.body = bytes.NewBuffer(nil)
		io.WriteString(tr.body, values.Encode())
	}
	method := tr.method
	if method == "" {
		method = "GET"
	}
	path := tr.path
	if path == "" {
		path = "example/test/path"
	}
	req := httptest.NewRequest(method, "https://an.example/"+path, tr.body)
	token := tr.token
	if token == "" {
		token = arvadostest.ActiveTokenV2
	}
	if token != noToken {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if tr.json {
		req.Header.Set("Content-Type", "application/json")
	} else if tr.header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range tr.header {
		req.Header[k] = append([]string(nil), v...)
	}
	return req
}

func (tr *testReq) bodyContent() string {
	return string(tr.body.Bytes())
}

func (s *RouterSuite) TestAttrsInBody(c *check.C) {
	attrs := map[string]interface{}{"foo": "bar"}

	multipartBody := new(bytes.Buffer)
	multipartWriter := multipart.NewWriter(multipartBody)
	multipartWriter.WriteField("attrs", `{"foo":"bar"}`)
	multipartWriter.Close()

	for _, tr := range []testReq{
		{attrsKey: "model_name", json: true, attrs: attrs},
		{attrsKey: "model_name", json: true, attrs: attrs, jsonAttrsTop: true},
		{attrsKey: "model_name", json: true, attrs: attrs, jsonAttrsTop: true, jsonStringParam: true},
		{attrsKey: "model_name", json: true, attrs: attrs, jsonAttrsTop: false, jsonStringParam: true},
		{body: multipartBody, header: http.Header{"Content-Type": []string{multipartWriter.FormDataContentType()}}},
	} {
		c.Logf("tr: %#v", tr)
		req := tr.Request()
		params, err := s.rtr.loadRequestParams(req, tr.attrsKey)
		c.Logf("params: %#v", params)
		c.Assert(err, check.IsNil)
		c.Check(params, check.NotNil)
		c.Assert(params["attrs"], check.FitsTypeOf, map[string]interface{}{})
		c.Check(params["attrs"].(map[string]interface{})["foo"], check.Equals, "bar")
	}
}

func (s *RouterSuite) TestBoolParam(c *check.C) {
	testKey := "ensure_unique_name"

	for i, tr := range []testReq{
		{method: "POST", param: map[string]interface{}{testKey: false}, json: true},
		{method: "POST", param: map[string]interface{}{testKey: false}},
		{method: "POST", param: map[string]interface{}{testKey: "false"}},
		{method: "POST", param: map[string]interface{}{testKey: "0"}},
		{method: "POST", param: map[string]interface{}{testKey: ""}},
	} {
		c.Logf("#%d, tr: %#v", i, tr)
		req := tr.Request()
		c.Logf("tr.body: %s", tr.bodyContent())
		params, err := s.rtr.loadRequestParams(req, tr.attrsKey)
		c.Logf("params: %#v", params)
		c.Assert(err, check.IsNil)
		c.Check(params, check.NotNil)
		c.Check(params[testKey], check.Equals, false)
	}

	for i, tr := range []testReq{
		{method: "POST", param: map[string]interface{}{testKey: true}, json: true},
		{method: "POST", param: map[string]interface{}{testKey: true}},
		{method: "POST", param: map[string]interface{}{testKey: "true"}},
		{method: "POST", param: map[string]interface{}{testKey: "1"}},
	} {
		c.Logf("#%d, tr: %#v", i, tr)
		req := tr.Request()
		c.Logf("tr.body: %s", tr.bodyContent())
		params, err := s.rtr.loadRequestParams(req, tr.attrsKey)
		c.Logf("params: %#v", params)
		c.Assert(err, check.IsNil)
		c.Check(params, check.NotNil)
		c.Check(params[testKey], check.Equals, true)
	}
}

func (s *RouterSuite) TestOrderParam(c *check.C) {
	for i, tr := range []testReq{
		{method: "POST", param: map[string]interface{}{"order": ""}, json: true},
		{method: "POST", param: map[string]interface{}{"order": ""}, json: false},
		{method: "POST", param: map[string]interface{}{"order": []string{}}, json: true},
		{method: "POST", param: map[string]interface{}{"order": []string{}}, json: false},
		{method: "POST", param: map[string]interface{}{}, json: true},
		{method: "POST", param: map[string]interface{}{}, json: false},
	} {
		c.Logf("#%d, tr: %#v", i, tr)
		req := tr.Request()
		params, err := s.rtr.loadRequestParams(req, tr.attrsKey)
		c.Assert(err, check.IsNil)
		c.Assert(params, check.NotNil)
		if order, ok := params["order"]; ok && order != nil {
			c.Check(order, check.DeepEquals, []interface{}{})
		}
	}

	for i, tr := range []testReq{
		{method: "POST", param: map[string]interface{}{"order": "foo,bar desc"}, json: true},
		{method: "POST", param: map[string]interface{}{"order": "foo,bar desc"}, json: false},
		{method: "POST", param: map[string]interface{}{"order": "[\"foo\", \"bar desc\"]"}, json: false},
		{method: "POST", param: map[string]interface{}{"order": []string{"foo", "bar desc"}}, json: true},
		{method: "POST", param: map[string]interface{}{"order": []string{"foo", "bar desc"}}, json: false},
	} {
		c.Logf("#%d, tr: %#v", i, tr)
		req := tr.Request()
		params, err := s.rtr.loadRequestParams(req, tr.attrsKey)
		c.Assert(err, check.IsNil)
		if _, ok := params["order"].([]string); ok {
			c.Check(params["order"], check.DeepEquals, []string{"foo", "bar desc"})
		} else {
			c.Check(params["order"], check.DeepEquals, []interface{}{"foo", "bar desc"})
		}
	}
}
