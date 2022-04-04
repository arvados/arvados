// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing/iotest"

	check "gopkg.in/check.v1"
)

type stubTransport struct {
	Responses map[string]string
	Requests  []http.Request
	sync.Mutex
}

func (stub *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	stub.Lock()
	stub.Requests = append(stub.Requests, *req)
	stub.Unlock()

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    req,
	}
	str := stub.Responses[req.URL.Path]
	if str == "" {
		resp.Status = "404 Not Found"
		resp.StatusCode = 404
		str = "{}"
	}
	buf := bytes.NewBufferString(str)
	resp.Body = ioutil.NopCloser(buf)
	resp.ContentLength = int64(buf.Len())
	return resp, nil
}

type errorTransport struct{}

func (stub *errorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("something awful happened")
}

type timeoutTransport struct {
	response []byte
}

func (stub *timeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Request:    req,
		Body:       ioutil.NopCloser(iotest.TimeoutReader(bytes.NewReader(stub.response))),
	}, nil
}

var _ = check.Suite(&clientSuite{})

type clientSuite struct{}

func (*clientSuite) TestCurrentUser(c *check.C) {
	stub := &stubTransport{
		Responses: map[string]string{
			"/arvados/v1/users/current": `{"uuid":"zzzzz-abcde-012340123401234"}`,
		},
	}
	client := &Client{
		Client: &http.Client{
			Transport: stub,
		},
		APIHost:   "zzzzz.arvadosapi.com",
		AuthToken: "xyzzy",
	}
	u, err := client.CurrentUser()
	c.Check(err, check.IsNil)
	c.Check(u.UUID, check.Equals, "zzzzz-abcde-012340123401234")
	c.Check(stub.Requests, check.Not(check.HasLen), 0)
	hdr := stub.Requests[len(stub.Requests)-1].Header
	c.Check(hdr.Get("Authorization"), check.Equals, "OAuth2 xyzzy")

	client.Client.Transport = &errorTransport{}
	u, err = client.CurrentUser()
	c.Check(err, check.NotNil)
}

func (*clientSuite) TestAnythingToValues(c *check.C) {
	type testCase struct {
		in interface{}
		// ok==nil means anythingToValues should return an
		// error, otherwise it's a func that returns true if
		// out is correct
		ok func(out url.Values) bool
	}
	for _, tc := range []testCase{
		{
			in: map[string]interface{}{"foo": "bar"},
			ok: func(out url.Values) bool {
				return out.Get("foo") == "bar"
			},
		},
		{
			in: map[string]interface{}{"foo": 2147483647},
			ok: func(out url.Values) bool {
				return out.Get("foo") == "2147483647"
			},
		},
		{
			in: map[string]interface{}{"foo": 1.234},
			ok: func(out url.Values) bool {
				return out.Get("foo") == "1.234"
			},
		},
		{
			in: map[string]interface{}{"foo": "1.234"},
			ok: func(out url.Values) bool {
				return out.Get("foo") == "1.234"
			},
		},
		{
			in: map[string]interface{}{"foo": map[string]interface{}{"bar": 1.234}},
			ok: func(out url.Values) bool {
				return out.Get("foo") == `{"bar":1.234}`
			},
		},
		{
			in: url.Values{"foo": {"bar"}},
			ok: func(out url.Values) bool {
				return out.Get("foo") == "bar"
			},
		},
		{
			in: 1234,
			ok: nil,
		},
		{
			in: []string{"foo"},
			ok: nil,
		},
	} {
		c.Logf("%#v", tc.in)
		out, err := anythingToValues(tc.in)
		if tc.ok == nil {
			c.Check(err, check.NotNil)
			continue
		}
		c.Check(err, check.IsNil)
		c.Check(tc.ok(out), check.Equals, true)
	}
}

func (*clientSuite) TestLoadConfig(c *check.C) {
	oldenv := os.Environ()
	defer func() {
		os.Clearenv()
		for _, s := range oldenv {
			i := strings.IndexRune(s, '=')
			os.Setenv(s[:i], s[i+1:])
		}
	}()

	tmp := c.MkDir()
	os.Setenv("HOME", tmp)
	for _, s := range os.Environ() {
		if strings.HasPrefix(s, "ARVADOS_") {
			i := strings.IndexRune(s, '=')
			os.Unsetenv(s[:i])
		}
	}
	os.Mkdir(tmp+"/.config", 0777)
	os.Mkdir(tmp+"/.config/arvados", 0777)

	// Use $HOME/.config/arvados/settings.conf if no env vars are
	// set
	os.WriteFile(tmp+"/.config/arvados/settings.conf", []byte(`
		ARVADOS_API_HOST = localhost:1
		ARVADOS_API_TOKEN = token_from_settings_file1
	`), 0777)
	client := NewClientFromEnv()
	c.Check(client.AuthToken, check.Equals, "token_from_settings_file1")
	c.Check(client.APIHost, check.Equals, "localhost:1")
	c.Check(client.Insecure, check.Equals, false)

	// ..._INSECURE=true, comments, ignored lines in settings.conf
	os.WriteFile(tmp+"/.config/arvados/settings.conf", []byte(`
		(ignored) = (ignored)
		#ARVADOS_API_HOST = localhost:2
		ARVADOS_API_TOKEN = token_from_settings_file2
		ARVADOS_API_HOST_INSECURE = true
	`), 0777)
	client = NewClientFromEnv()
	c.Check(client.AuthToken, check.Equals, "token_from_settings_file2")
	c.Check(client.APIHost, check.Equals, "")
	c.Check(client.Insecure, check.Equals, true)

	// Environment variables override settings.conf
	os.Setenv("ARVADOS_API_HOST", "[::]:3")
	os.Setenv("ARVADOS_API_HOST_INSECURE", "0")
	client = NewClientFromEnv()
	c.Check(client.AuthToken, check.Equals, "token_from_settings_file2")
	c.Check(client.APIHost, check.Equals, "[::]:3")
	c.Check(client.Insecure, check.Equals, false)
}
