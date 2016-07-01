package arvados

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"testing"
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

func TestCurrentUser(t *testing.T) {
	t.Parallel()
	stub := &stubTransport{
		Responses: map[string]string{
			"/arvados/v1/users/current": `{"uuid":"zzzzz-abcde-012340123401234"}`,
		},
	}
	c := &Client{
		Client: &http.Client{
			Transport: stub,
		},
		APIHost:   "zzzzz.arvadosapi.com",
		AuthToken: "xyzzy",
	}
	u, err := c.CurrentUser()
	if err != nil {
		t.Fatal(err)
	}
	if x := "zzzzz-abcde-012340123401234"; u.UUID != x {
		t.Errorf("got uuid %q, expected %q", u.UUID, x)
	}
	if len(stub.Requests) < 1 {
		t.Fatal("empty stub.Requests")
	}
	hdr := stub.Requests[len(stub.Requests)-1].Header
	if hdr.Get("Authorization") != "OAuth2 xyzzy" {
		t.Errorf("got headers %+q, expected Authorization header", hdr)
	}

	c.Client.Transport = &errorTransport{}
	u, err = c.CurrentUser()
	if err == nil {
		t.Errorf("got nil error, expected something awful")
	}
}

func TestAnythingToValues(t *testing.T) {
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
		t.Logf("%#v", tc.in)
		out, err := anythingToValues(tc.in)
		switch {
		case tc.ok == nil:
			if err == nil {
				t.Errorf("got %#v, expected error", out)
			}
		case err != nil:
			t.Errorf("got err %#v, expected nil", err)
		case !tc.ok(out):
			t.Errorf("got %#v but tc.ok() says that is wrong", out)
		}
	}
}
