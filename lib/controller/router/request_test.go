// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package router

import (
	"bytes"
	"net/http/httptest"

	check "gopkg.in/check.v1"
)

func (s *RouterSuite) TestAttrsInBody(c *check.C) {
	for _, body := range []string{
		`{"foo":"bar"}`,
		`{"model_name": {"foo":"bar"}}`,
	} {
		c.Logf("body: %s", body)
		req := httptest.NewRequest("POST", "https://an.example/ctrl", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		params, err := s.rtr.loadRequestParams(req, "model_name")
		c.Assert(err, check.IsNil)
		c.Logf("params: %#v", params)
		c.Check(params, check.NotNil)
		c.Assert(params["attrs"], check.FitsTypeOf, map[string]interface{}{})
		c.Check(params["attrs"].(map[string]interface{})["foo"], check.Equals, "bar")
	}
}
