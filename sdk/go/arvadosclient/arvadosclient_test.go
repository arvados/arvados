// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvadosclient

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&ServerRequiredSuite{})
var _ = Suite(&UnitSuite{})
var _ = Suite(&MockArvadosServerSuite{})

// Tests that require the Keep server running
type ServerRequiredSuite struct{}

func (s *ServerRequiredSuite) SetUpSuite(c *C) {
	arvadostest.StartAPI()
	arvadostest.StartKeep(2, false)
	RetryDelay = 0
}

func (s *ServerRequiredSuite) TearDownSuite(c *C) {
	arvadostest.StopKeep(2)
	arvadostest.StopAPI()
}

func (s *ServerRequiredSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

func (s *ServerRequiredSuite) TestMakeArvadosClientSecure(c *C) {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "")
	ac, err := MakeArvadosClient()
	c.Assert(err, Equals, nil)
	c.Check(ac.ApiServer, Equals, os.Getenv("ARVADOS_API_HOST"))
	c.Check(ac.ApiToken, Equals, os.Getenv("ARVADOS_API_TOKEN"))
	c.Check(ac.ApiInsecure, Equals, false)
}

func (s *ServerRequiredSuite) TestMakeArvadosClientInsecure(c *C) {
	os.Setenv("ARVADOS_API_HOST_INSECURE", "true")
	ac, err := MakeArvadosClient()
	c.Assert(err, Equals, nil)
	c.Check(ac.ApiInsecure, Equals, true)
	c.Check(ac.ApiServer, Equals, os.Getenv("ARVADOS_API_HOST"))
	c.Check(ac.ApiToken, Equals, os.Getenv("ARVADOS_API_TOKEN"))
	c.Check(ac.Client.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify, Equals, true)
}

func (s *ServerRequiredSuite) TestGetInvalidUUID(c *C) {
	arv, err := MakeArvadosClient()
	c.Assert(err, IsNil)

	getback := make(Dict)
	err = arv.Get("collections", "", nil, &getback)
	c.Assert(err, Equals, ErrInvalidArgument)
	c.Assert(len(getback), Equals, 0)

	err = arv.Get("collections", "zebra-moose-unicorn", nil, &getback)
	c.Assert(err, Equals, ErrInvalidArgument)
	c.Assert(len(getback), Equals, 0)

	err = arv.Get("collections", "acbd18db4cc2f85cedef654fccc4a4d8", nil, &getback)
	c.Assert(err, Equals, ErrInvalidArgument)
	c.Assert(len(getback), Equals, 0)
}

func (s *ServerRequiredSuite) TestGetValidUUID(c *C) {
	arv, err := MakeArvadosClient()
	c.Assert(err, IsNil)

	getback := make(Dict)
	err = arv.Get("collections", "zzzzz-4zz18-abcdeabcdeabcde", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)

	err = arv.Get("collections", "acbd18db4cc2f85cedef654fccc4a4d8+3", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)
}

func (s *ServerRequiredSuite) TestInvalidResourceType(c *C) {
	arv, err := MakeArvadosClient()
	c.Assert(err, IsNil)

	getback := make(Dict)
	err = arv.Get("unicorns", "zzzzz-zebra-unicorn7unicorn", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)

	err = arv.Update("unicorns", "zzzzz-zebra-unicorn7unicorn", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)

	err = arv.List("unicorns", nil, &getback)
	c.Assert(err, FitsTypeOf, APIServerError{})
	c.Assert(err.(APIServerError).HttpStatusCode, Equals, http.StatusNotFound)
	c.Assert(len(getback), Equals, 0)
}

func (s *ServerRequiredSuite) TestErrorResponse(c *C) {
	arv, _ := MakeArvadosClient()

	getback := make(Dict)

	{
		err := arv.Create("logs",
			Dict{"log": Dict{"bogus_attr": "foo"}},
			&getback)
		c.Assert(err, ErrorMatches, "arvados API server error: .*")
		c.Assert(err, ErrorMatches, ".*unknown attribute(: | ')bogus_attr.*")
		c.Assert(err, FitsTypeOf, APIServerError{})
		c.Assert(err.(APIServerError).HttpStatusCode, Equals, 422)
	}

	{
		err := arv.Create("bogus",
			Dict{"bogus": Dict{}},
			&getback)
		c.Assert(err, ErrorMatches, "arvados API server error: .*")
		c.Assert(err, ErrorMatches, ".*Path not found.*")
		c.Assert(err, FitsTypeOf, APIServerError{})
		c.Assert(err.(APIServerError).HttpStatusCode, Equals, 404)
	}
}

func (s *ServerRequiredSuite) TestAPIDiscovery_Get_defaultCollectionReplication(c *C) {
	arv, err := MakeArvadosClient()
	c.Assert(err, IsNil)
	value, err := arv.Discovery("defaultCollectionReplication")
	c.Assert(err, IsNil)
	c.Assert(value, NotNil)
}

func (s *ServerRequiredSuite) TestAPIDiscovery_Get_noSuchParameter(c *C) {
	arv, err := MakeArvadosClient()
	c.Assert(err, IsNil)
	value, err := arv.Discovery("noSuchParameter")
	c.Assert(err, NotNil)
	c.Assert(value, IsNil)
}

func (s *ServerRequiredSuite) TestCreateLarge(c *C) {
	arv, err := MakeArvadosClient()
	c.Assert(err, IsNil)

	txt := arvados.SignLocator("d41d8cd98f00b204e9800998ecf8427e+0", arv.ApiToken, time.Now().Add(time.Minute), time.Minute, []byte(arvadostest.SystemRootToken))
	// Ensure our request body is bigger than the Go http server's
	// default max size, 10 MB.
	for len(txt) < 12000000 {
		txt = txt + " " + txt
	}
	txt = ". " + txt + " 0:0:foo\n"

	resp := Dict{}
	err = arv.Create("collections", Dict{
		"ensure_unique_name": true,
		"collection": Dict{
			"is_trashed":    true,
			"name":          "test",
			"manifest_text": txt,
		},
	}, &resp)
	c.Check(err, IsNil)
	c.Check(resp["portable_data_hash"], Not(Equals), "")
	c.Check(resp["portable_data_hash"], Not(Equals), "d41d8cd98f00b204e9800998ecf8427e+0")
}

type UnitSuite struct{}

func (s *UnitSuite) TestUUIDMatch(c *C) {
	c.Assert(UUIDMatch("zzzzz-tpzed-000000000000000"), Equals, true)
	c.Assert(UUIDMatch("zzzzz-zebra-000000000000000"), Equals, true)
	c.Assert(UUIDMatch("00000-00000-zzzzzzzzzzzzzzz"), Equals, true)
	c.Assert(UUIDMatch("ZEBRA-HORSE-AFRICANELEPHANT"), Equals, false)
	c.Assert(UUIDMatch(" zzzzz-tpzed-000000000000000"), Equals, false)
	c.Assert(UUIDMatch("d41d8cd98f00b204e9800998ecf8427e"), Equals, false)
	c.Assert(UUIDMatch("d41d8cd98f00b204e9800998ecf8427e+0"), Equals, false)
	c.Assert(UUIDMatch(""), Equals, false)
}

func (s *UnitSuite) TestPDHMatch(c *C) {
	c.Assert(PDHMatch("zzzzz-tpzed-000000000000000"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+0"), Equals, true)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+12345"), Equals, true)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e 12345"), Equals, false)
	c.Assert(PDHMatch("D41D8CD98F00B204E9800998ECF8427E+12345"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+12345 "), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+abcdef"), Equals, false)
	c.Assert(PDHMatch("da39a3ee5e6b4b0d3255bfef95601890afd80709"), Equals, false)
	c.Assert(PDHMatch("da39a3ee5e6b4b0d3255bfef95601890afd80709+0"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427+12345"), Equals, false)
	c.Assert(PDHMatch("d41d8cd98f00b204e9800998ecf8427e+12345\n"), Equals, false)
	c.Assert(PDHMatch("+12345"), Equals, false)
	c.Assert(PDHMatch(""), Equals, false)
}

// Tests that use mock arvados server
type MockArvadosServerSuite struct{}

func (s *MockArvadosServerSuite) SetUpSuite(c *C) {
	RetryDelay = 0
}

func (s *MockArvadosServerSuite) SetUpTest(c *C) {
	arvadostest.ResetEnv()
}

type APIServer struct {
	listener net.Listener
	url      string
}

func RunFakeArvadosServer(st http.Handler) (api APIServer, err error) {
	api.listener, err = net.ListenTCP("tcp", &net.TCPAddr{Port: 0})
	if err != nil {
		return
	}
	api.url = api.listener.Addr().String()
	go http.Serve(api.listener, st)
	return
}

type APIStub struct {
	method        string
	retryAttempts int
	expected      int
	respStatus    []int
	responseBody  []string
}

func (h *APIStub) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if req.URL.Path == "/redirect-loop" {
		http.Redirect(resp, req, "/redirect-loop", http.StatusFound)
		return
	}
	if h.respStatus[h.retryAttempts] < 0 {
		// Fail the client's Do() by starting a redirect loop
		http.Redirect(resp, req, "/redirect-loop", http.StatusFound)
	} else {
		resp.WriteHeader(h.respStatus[h.retryAttempts])
		resp.Write([]byte(h.responseBody[h.retryAttempts]))
	}
	h.retryAttempts++
}

func (s *MockArvadosServerSuite) TestWithRetries(c *C) {
	for _, stub := range []APIStub{
		{
			"get", 0, 200, []int{200, 500}, []string{`{"ok":"ok"}`, ``},
		},
		{
			"create", 0, 200, []int{200, 500}, []string{`{"ok":"ok"}`, ``},
		},
		{
			"get", 0, 500, []int{500, 500, 500, 200}, []string{``, ``, ``, `{"ok":"ok"}`},
		},
		{
			"create", 0, 500, []int{500, 500, 500, 200}, []string{``, ``, ``, `{"ok":"ok"}`},
		},
		{
			"update", 0, 500, []int{500, 500, 500, 200}, []string{``, ``, ``, `{"ok":"ok"}`},
		},
		{
			"delete", 0, 500, []int{500, 500, 500, 200}, []string{``, ``, ``, `{"ok":"ok"}`},
		},
		{
			"get", 0, 502, []int{500, 500, 502, 200}, []string{``, ``, ``, `{"ok":"ok"}`},
		},
		{
			"create", 0, 502, []int{500, 500, 502, 200}, []string{``, ``, ``, `{"ok":"ok"}`},
		},
		{
			"get", 0, 200, []int{500, 500, 200}, []string{``, ``, `{"ok":"ok"}`},
		},
		{
			"create", 0, 200, []int{500, 500, 200}, []string{``, ``, `{"ok":"ok"}`},
		},
		{
			"delete", 0, 200, []int{500, 500, 200}, []string{``, ``, `{"ok":"ok"}`},
		},
		{
			"update", 0, 200, []int{500, 500, 200}, []string{``, ``, `{"ok":"ok"}`},
		},
		{
			"get", 0, 401, []int{401, 200}, []string{``, `{"ok":"ok"}`},
		},
		{
			"create", 0, 401, []int{401, 200}, []string{``, `{"ok":"ok"}`},
		},
		{
			"get", 0, 404, []int{404, 200}, []string{``, `{"ok":"ok"}`},
		},
		{
			"get", 0, 401, []int{500, 401, 200}, []string{``, ``, `{"ok":"ok"}`},
		},

		// Response code -1 simulates an HTTP/network error
		// (i.e., Do() returns an error; there is no HTTP
		// response status code).

		// Succeed on second retry
		{
			"get", 0, 200, []int{-1, -1, 200}, []string{``, ``, `{"ok":"ok"}`},
		},
		// "POST" is not safe to retry: fail after one error
		{
			"create", 0, -1, []int{-1, 200}, []string{``, `{"ok":"ok"}`},
		},
	} {
		api, err := RunFakeArvadosServer(&stub)
		c.Check(err, IsNil)

		defer api.listener.Close()

		arv := ArvadosClient{
			Scheme:      "http",
			ApiServer:   api.url,
			ApiToken:    "abc123",
			ApiInsecure: true,
			Client:      &http.Client{Transport: &http.Transport{}},
			Retries:     2}

		getback := make(Dict)
		switch stub.method {
		case "get":
			err = arv.Get("collections", "zzzzz-4zz18-znfnqtbbv4spc3w", nil, &getback)
		case "create":
			err = arv.Create("collections",
				Dict{"collection": Dict{"name": "testing"}},
				&getback)
		case "update":
			err = arv.Update("collections", "zzzzz-4zz18-znfnqtbbv4spc3w",
				Dict{"collection": Dict{"name": "testing"}},
				&getback)
		case "delete":
			err = arv.Delete("pipeline_templates", "zzzzz-4zz18-znfnqtbbv4spc3w", nil, &getback)
		}

		switch stub.expected {
		case 200:
			c.Check(err, IsNil)
			c.Check(getback["ok"], Equals, "ok")
		case -1:
			c.Check(err, NotNil)
			c.Check(err, ErrorMatches, `.*stopped after \d+ redirects`)
		default:
			c.Check(err, NotNil)
			c.Check(err, ErrorMatches, fmt.Sprintf("arvados API server error: %d.*", stub.expected))
			c.Check(err.(APIServerError).HttpStatusCode, Equals, stub.expected)
		}
	}
}
