// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// package service provides a cmd.Handler that brings up a system service.
package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/prometheus/client_golang/prometheus"
	check "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&Suite{})

type Suite struct{}
type key int

const (
	contextKey key = iota
)

func (*Suite) TestCommand(c *check.C) {
	cf, err := ioutil.TempFile("", "cmd_test.")
	c.Assert(err, check.IsNil)
	defer os.Remove(cf.Name())
	defer cf.Close()
	fmt.Fprintf(cf, "Clusters:\n zzzzz:\n  SystemRootToken: abcde\n  NodeProfiles: {\"*\": {\"arvados-controller\": {Listen: \":1234\"}}}")

	healthCheck := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := Command(arvados.ServiceNameController, func(ctx context.Context, _ *arvados.Cluster, token string, reg *prometheus.Registry) Handler {
		c.Check(ctx.Value(contextKey), check.Equals, "bar")
		c.Check(token, check.Equals, "abcde")
		return &testHandler{ctx: ctx, healthCheck: healthCheck}
	})
	cmd.(*command).ctx = context.WithValue(ctx, contextKey, "bar")

	done := make(chan bool)
	var stdin, stdout, stderr bytes.Buffer

	go func() {
		cmd.RunCommand("arvados-controller", []string{"-config", cf.Name()}, &stdin, &stdout, &stderr)
		close(done)
	}()
	select {
	case <-healthCheck:
	case <-done:
		c.Error("command exited without health check")
	}
	cancel()
	c.Check(stdout.String(), check.Equals, "")
	c.Check(stderr.String(), check.Matches, `(?ms).*"msg":"CheckHealth called".*`)
}

func (*Suite) TestTLS(c *check.C) {
	cwd, err := os.Getwd()
	c.Assert(err, check.IsNil)

	stdin := bytes.NewBufferString(`
Clusters:
 zzzzz:
  SystemRootToken: abcde
  Services:
   Controller:
    ExternalURL: "https://localhost:12345"
    InternalURLs: {"https://localhost:12345": {}}
  TLS:
   Key: file://` + cwd + `/../../services/api/tmp/self-signed.key
   Certificate: file://` + cwd + `/../../services/api/tmp/self-signed.pem
`)

	called := make(chan bool)
	cmd := Command(arvados.ServiceNameController, func(ctx context.Context, _ *arvados.Cluster, token string, reg *prometheus.Registry) Handler {
		return &testHandler{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
			close(called)
		})}
	})

	exited := make(chan bool)
	var stdout, stderr bytes.Buffer
	go func() {
		cmd.RunCommand("arvados-controller", []string{"-config", "-"}, stdin, &stdout, &stderr)
		close(exited)
	}()
	got := make(chan bool)
	go func() {
		defer close(got)
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
		for range time.NewTicker(time.Millisecond).C {
			resp, err := client.Get("https://localhost:12345")
			if err != nil {
				c.Log(err)
				continue
			}
			body, err := ioutil.ReadAll(resp.Body)
			c.Check(err, check.IsNil)
			c.Logf("status %d, body %s", resp.StatusCode, string(body))
			c.Check(resp.StatusCode, check.Equals, http.StatusOK)
			break
		}
	}()
	select {
	case <-called:
	case <-exited:
		c.Error("command exited without calling handler")
	case <-time.After(time.Second):
		c.Error("timed out")
	}
	select {
	case <-got:
	case <-exited:
		c.Error("command exited before client received response")
	case <-time.After(time.Second):
		c.Error("timed out")
	}
	c.Log(stderr.String())
}

type testHandler struct {
	ctx         context.Context
	handler     http.Handler
	healthCheck chan bool
}

func (th *testHandler) Done() <-chan struct{}                            { return nil }
func (th *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) { th.handler.ServeHTTP(w, r) }
func (th *testHandler) CheckHealth() error {
	ctxlog.FromContext(th.ctx).Info("CheckHealth called")
	select {
	case th.healthCheck <- true:
	default:
	}
	return nil
}
