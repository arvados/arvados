package main

import (
	"net/http"
	"testing"
)

type basicAuthTestCase struct {
	hdr  string
	user string
	pass string
	ok   bool
}

func TestBasicAuth(t *testing.T) {
	tests := []basicAuthTestCase{
		{"Basic Zm9vOmJhcg==", "foo", "bar", true},
		{"Bogus Zm9vOmJhcg==", "", "", false},
		{"Zm9vOmJhcg==", "", "", false},
		{"Basic", "", "", false},
		{"", "", "", false},
	}
	for _, test := range tests {
		if u, p, ok := BasicAuth(&http.Request{Header: map[string][]string{
			"Authorization": {test.hdr},
		}}); u != test.user || p != test.pass || ok != test.ok {
			t.Error("got:", u, p, ok, "expected:", test.user, test.pass, test.ok, "from:", test.hdr)
		}
	}
}
