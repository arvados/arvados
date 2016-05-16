package arvados

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
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
