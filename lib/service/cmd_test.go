// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// package service provides a cmd.Handler that brings up a system service.
package service

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&Suite{})

type Suite struct{}

func (*Suite) TestCommand(c *check.C) {
	cf, err := ioutil.TempFile("", "cmd_test.")
	c.Assert(err, check.IsNil)
	defer os.Remove(cf.Name())
	defer cf.Close()
	fmt.Fprintf(cf, "Clusters:\n zzzzz:\n  SystemRootToken: abcde\n  NodeProfiles: {\"*\": {\"arvados-controller\": {Listen: \":1234\"}}}")

	healthCheck := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := Command(arvados.ServiceNameController, func(ctx context.Context, _ *arvados.Cluster, token string) Handler {
		c.Check(ctx.Value("foo"), check.Equals, "bar")
		c.Check(token, check.Equals, "abcde")
		return &testHandler{ctx: ctx, healthCheck: healthCheck}
	})
	cmd.(*command).ctx = context.WithValue(ctx, "foo", "bar")

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

type testHandler struct {
	ctx         context.Context
	healthCheck chan bool
}

func (th *testHandler) ServeHTTP(http.ResponseWriter, *http.Request) {}
func (th *testHandler) CheckHealth() error {
	ctxlog.FromContext(th.ctx).Info("CheckHealth called")
	select {
	case th.healthCheck <- true:
	default:
	}
	return nil
}
