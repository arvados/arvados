// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	check "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&Suite{})

type Suite struct{}

func (s *Suite) TestLogRequests(c *check.C) {
	captured := &bytes.Buffer{}
	log := logrus.New()
	log.Out = captured
	log.Formatter = &logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	}

	h := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("hello world"))
	})
	req, err := http.NewRequest("GET", "https://foo.example/bar", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4:12345")
	c.Assert(err, check.IsNil)
	resp := httptest.NewRecorder()
	AddRequestIDs(LogRequests(log, h)).ServeHTTP(resp, req)

	dec := json.NewDecoder(captured)

	gotReq := make(map[string]interface{})
	err = dec.Decode(&gotReq)
	c.Logf("%#v", gotReq)
	c.Check(gotReq["RequestID"], check.Matches, "req-[a-z0-9]{20}")
	c.Check(gotReq["reqForwardedFor"], check.Equals, "1.2.3.4:12345")
	c.Check(gotReq["msg"], check.Equals, "request")

	gotResp := make(map[string]interface{})
	err = dec.Decode(&gotResp)
	c.Logf("%#v", gotResp)
	c.Check(gotResp["RequestID"], check.Equals, gotReq["RequestID"])
	c.Check(gotResp["reqForwardedFor"], check.Equals, "1.2.3.4:12345")
	c.Check(gotResp["msg"], check.Equals, "response")

	c.Assert(gotResp["time"], check.FitsTypeOf, "")
	_, err = time.Parse(time.RFC3339Nano, gotResp["time"].(string))
	c.Check(err, check.IsNil)

	for _, key := range []string{"timeToStatus", "timeWriteBody", "timeTotal"} {
		c.Assert(gotResp[key], check.FitsTypeOf, float64(0))
		c.Check(gotResp[key].(float64), check.Not(check.Equals), float64(0))
	}
}
