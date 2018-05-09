// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
)

func TestTLS(t *testing.T) {
	defer func() {
		theConfig.TLSKeyFile = ""
		theConfig.TLSCertificateFile = ""
	}()
	theConfig.TLSKeyFile = "../api/tmp/self-signed.key"
	theConfig.TLSCertificateFile = "../api/tmp/self-signed.pem"
	srv := &server{}
	srv.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	l, err := net.Listen("tcp", ":")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	go srv.Serve(l)
	defer srv.Shutdown(context.Background())
	c := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := c.Get("https://" + l.Addr().String() + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Error(err)
	}
	if !bytes.Equal(body, []byte("OK")) {
		t.Errorf("expected OK, got %q", body)
	}
}
