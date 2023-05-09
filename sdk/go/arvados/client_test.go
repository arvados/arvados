// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing/iotest"
	"time"

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

var _ = check.Suite(&clientRetrySuite{})

type clientRetrySuite struct {
	server     *httptest.Server
	client     Client
	reqs       []*http.Request
	respStatus chan int
	respDelay  time.Duration

	origLimiterQuietPeriod time.Duration
}

func (s *clientRetrySuite) SetUpTest(c *check.C) {
	// Test server: delay and return errors until a final status
	// appears on the respStatus channel.
	s.origLimiterQuietPeriod = requestLimiterQuietPeriod
	requestLimiterQuietPeriod = time.Second / 100
	s.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.reqs = append(s.reqs, r)
		delay := s.respDelay
		if delay == 0 {
			delay = time.Duration(rand.Int63n(int64(time.Second / 10)))
		}
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case code, ok := <-s.respStatus:
			if !ok {
				code = http.StatusOK
			}
			w.WriteHeader(code)
			w.Write([]byte(`{}`))
		case <-timer.C:
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	s.reqs = nil
	s.respStatus = make(chan int, 1)
	s.client = Client{
		APIHost:   s.server.URL[8:],
		AuthToken: "zzz",
		Insecure:  true,
		Timeout:   2 * time.Second,
	}
}

func (s *clientRetrySuite) TearDownTest(c *check.C) {
	s.server.Close()
	requestLimiterQuietPeriod = s.origLimiterQuietPeriod
}

func (s *clientRetrySuite) TestOK(c *check.C) {
	s.respStatus <- http.StatusOK
	err := s.client.RequestAndDecode(&struct{}{}, http.MethodGet, "test", nil, nil)
	c.Check(err, check.IsNil)
	c.Check(s.reqs, check.HasLen, 1)
}

func (s *clientRetrySuite) TestNetworkError(c *check.C) {
	// Close the stub server to produce a "connection refused" error.
	s.server.Close()

	start := time.Now()
	timeout := time.Second
	ctx, cancel := context.WithDeadline(context.Background(), start.Add(timeout))
	defer cancel()
	s.client.Timeout = timeout * 2
	err := s.client.RequestAndDecodeContext(ctx, &struct{}{}, http.MethodGet, "test", nil, nil)
	c.Check(err, check.ErrorMatches, `.*dial tcp .* connection refused.*`)
	delta := time.Since(start)
	c.Check(delta > timeout, check.Equals, true, check.Commentf("time.Since(start) == %v, timeout = %v", delta, timeout))
}

func (s *clientRetrySuite) TestNonRetryableError(c *check.C) {
	s.respStatus <- http.StatusBadRequest
	err := s.client.RequestAndDecode(&struct{}{}, http.MethodGet, "test", nil, nil)
	c.Check(err, check.ErrorMatches, `.*400 Bad Request.*`)
	c.Check(s.reqs, check.HasLen, 1)
}

func (s *clientRetrySuite) TestNonRetryableAfter503s(c *check.C) {
	time.AfterFunc(time.Second, func() { s.respStatus <- http.StatusNotFound })
	err := s.client.RequestAndDecode(&struct{}{}, http.MethodGet, "test", nil, nil)
	c.Check(err, check.ErrorMatches, `.*404 Not Found.*`)
}

func (s *clientRetrySuite) TestOKAfter503s(c *check.C) {
	start := time.Now()
	delay := time.Second
	time.AfterFunc(delay, func() { s.respStatus <- http.StatusOK })
	err := s.client.RequestAndDecode(&struct{}{}, http.MethodGet, "test", nil, nil)
	c.Check(err, check.IsNil)
	c.Check(len(s.reqs) > 1, check.Equals, true, check.Commentf("len(s.reqs) == %d", len(s.reqs)))
	c.Check(time.Since(start) > delay, check.Equals, true)
}

func (s *clientRetrySuite) TestTimeoutAfter503(c *check.C) {
	s.respStatus <- http.StatusServiceUnavailable
	s.respDelay = time.Second * 2
	s.client.Timeout = time.Second / 2
	err := s.client.RequestAndDecode(&struct{}{}, http.MethodGet, "test", nil, nil)
	c.Check(err, check.ErrorMatches, `.*503 Service Unavailable.*`)
	c.Check(s.reqs, check.HasLen, 2)
}

func (s *clientRetrySuite) Test503Forever(c *check.C) {
	err := s.client.RequestAndDecode(&struct{}{}, http.MethodGet, "test", nil, nil)
	c.Check(err, check.ErrorMatches, `.*503 Service Unavailable.*`)
	c.Check(len(s.reqs) > 1, check.Equals, true, check.Commentf("len(s.reqs) == %d", len(s.reqs)))
}

func (s *clientRetrySuite) TestContextAlreadyCanceled(c *check.C) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := s.client.RequestAndDecodeContext(ctx, &struct{}{}, http.MethodGet, "test", nil, nil)
	c.Check(err, check.Equals, context.Canceled)
}
