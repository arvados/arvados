// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

// package service provides a cmd.Handler that brings up a system service.
package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
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

func (*Suite) TestGetListenAddress(c *check.C) {
	// Find an available port on the testing host, so the test
	// cases don't get confused by "already in use" errors.
	listener, err := net.Listen("tcp", ":")
	c.Assert(err, check.IsNil)
	_, unusedPort, err := net.SplitHostPort(listener.Addr().String())
	c.Assert(err, check.IsNil)
	listener.Close()

	defer os.Unsetenv("ARVADOS_SERVICE_INTERNAL_URL")
	for idx, trial := range []struct {
		// internalURL => listenURL, both with trailing "/"
		// because config loader always adds it
		internalURLs     map[string]string
		envVar           string
		expectErrorMatch string
		expectLogsMatch  string
		expectListen     string
		expectInternal   string
	}{
		{
			internalURLs:   map[string]string{"http://localhost:" + unusedPort + "/": ""},
			expectListen:   "http://localhost:" + unusedPort + "/",
			expectInternal: "http://localhost:" + unusedPort + "/",
		},
		{ // implicit port 80 in InternalURLs
			internalURLs:     map[string]string{"http://localhost/": ""},
			expectErrorMatch: `.*:80: bind: permission denied`,
		},
		{ // implicit port 443 in InternalURLs
			internalURLs:   map[string]string{"https://host.example/": "http://localhost:" + unusedPort + "/"},
			expectListen:   "http://localhost:" + unusedPort + "/",
			expectInternal: "https://host.example/",
		},
		{ // implicit port 443 in ListenURL
			internalURLs:     map[string]string{"wss://host.example/": "wss://localhost/"},
			expectErrorMatch: `.*:443: bind: permission denied`,
		},
		{
			internalURLs:   map[string]string{"https://hostname.example/": "http://localhost:8000/"},
			expectListen:   "http://localhost:8000/",
			expectInternal: "https://hostname.example/",
		},
		{
			internalURLs: map[string]string{
				"https://hostname1.example/": "http://localhost:12435/",
				"https://hostname2.example/": "http://localhost:" + unusedPort + "/",
			},
			envVar:         "https://hostname2.example", // note this works despite missing trailing "/"
			expectListen:   "http://localhost:" + unusedPort + "/",
			expectInternal: "https://hostname2.example/",
		},
		{ // cannot listen on any of the ListenURLs
			internalURLs: map[string]string{
				"https://hostname1.example/": "http://1.2.3.4:" + unusedPort + "/",
				"https://hostname2.example/": "http://1.2.3.4:" + unusedPort + "/",
			},
			expectErrorMatch: "configuration does not enable the \"arvados-controller\" service on this host",
		},
		{ // cannot listen on any of the (implied) ListenURLs
			internalURLs: map[string]string{
				"https://1.2.3.4/": "",
				"https://1.2.3.5/": "",
			},
			expectErrorMatch: "configuration does not enable the \"arvados-controller\" service on this host",
		},
		{ // impossible port number
			internalURLs: map[string]string{
				"https://host.example/": "http://0.0.0.0:1234567",
			},
			expectErrorMatch: `.*:1234567: listen tcp: address 1234567: invalid port`,
		},
		{
			// env var URL not mentioned in config = obey env var, with warning
			internalURLs:    map[string]string{"https://hostname1.example/": "http://localhost:8000/"},
			envVar:          "https://hostname2.example",
			expectListen:    "https://hostname2.example/",
			expectInternal:  "https://hostname2.example/",
			expectLogsMatch: `.*\Qpossible configuration error: listening on https://hostname2.example/ (from $ARVADOS_SERVICE_INTERNAL_URL) even though configuration does not have a matching InternalURLs entry\E.*\n`,
		},
		{
			// env var + empty config = obey env var, with warning
			envVar:          "https://hostname.example",
			expectListen:    "https://hostname.example/",
			expectInternal:  "https://hostname.example/",
			expectLogsMatch: `.*\Qpossible configuration error: listening on https://hostname.example/ (from $ARVADOS_SERVICE_INTERNAL_URL) even though configuration does not have a matching InternalURLs entry\E.*\n`,
		},
	} {
		c.Logf("trial %d %+v", idx, trial)
		os.Setenv("ARVADOS_SERVICE_INTERNAL_URL", trial.envVar)
		var logbuf bytes.Buffer
		log := ctxlog.New(&logbuf, "text", "info")
		services := arvados.Services{Controller: arvados.Service{InternalURLs: map[arvados.URL]arvados.ServiceInstance{}}}
		for k, v := range trial.internalURLs {
			u, err := url.Parse(k)
			c.Assert(err, check.IsNil)
			si := arvados.ServiceInstance{}
			if v != "" {
				u, err := url.Parse(v)
				c.Assert(err, check.IsNil)
				si.ListenURL = arvados.URL(*u)
			}
			services.Controller.InternalURLs[arvados.URL(*u)] = si
		}
		listenURL, internalURL, err := getListenAddr(services, "arvados-controller", log)
		if trial.expectLogsMatch != "" {
			c.Check(logbuf.String(), check.Matches, trial.expectLogsMatch)
		}
		if trial.expectErrorMatch != "" {
			c.Check(err, check.ErrorMatches, trial.expectErrorMatch)
			continue
		}
		if !c.Check(err, check.IsNil) {
			continue
		}
		c.Check(listenURL.String(), check.Equals, trial.expectListen)
		c.Check(internalURL.String(), check.Equals, trial.expectInternal)
	}
}

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

func (*Suite) TestDumpRequests(c *check.C) {
	defer func(orig time.Duration) { requestQueueDumpCheckInterval = orig }(requestQueueDumpCheckInterval)
	requestQueueDumpCheckInterval = time.Second / 10

	tmpdir := c.MkDir()
	cf, err := ioutil.TempFile(tmpdir, "cmd_test.")
	c.Assert(err, check.IsNil)
	defer os.Remove(cf.Name())
	defer cf.Close()

	max := 1
	fmt.Fprintf(cf, `
Clusters:
 zzzzz:
  SystemRootToken: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  ManagementToken: bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
  API: {MaxConcurrentRequests: %d}
  SystemLogs: {RequestQueueDumpDirectory: %q}
  Services:
   Controller:
    ExternalURL: "http://localhost:12345"
    InternalURLs: {"http://localhost:12345": {}}
`, max, tmpdir)

	started := make(chan bool, max+1)
	hold := make(chan bool)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started <- true
		<-hold
	})
	healthCheck := make(chan bool, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := Command(arvados.ServiceNameController, func(ctx context.Context, _ *arvados.Cluster, token string, reg *prometheus.Registry) Handler {
		return &testHandler{ctx: ctx, handler: handler, healthCheck: healthCheck}
	})
	cmd.(*command).ctx = context.WithValue(ctx, contextKey, "bar")

	exited := make(chan bool)
	var stdin, stdout, stderr bytes.Buffer

	go func() {
		cmd.RunCommand("arvados-controller", []string{"-config", cf.Name()}, &stdin, &stdout, &stderr)
		close(exited)
	}()
	select {
	case <-healthCheck:
	case <-exited:
		c.Logf("%s", stderr.String())
		c.Error("command exited without health check")
	}
	client := http.Client{}
	deadline := time.Now().Add(time.Second * 2)
	for i := 0; i < max+1; i++ {
		go func() {
			resp, err := client.Get("http://localhost:12345/testpath")
			for err != nil && strings.Contains(err.Error(), "dial tcp") && deadline.After(time.Now()) {
				time.Sleep(time.Second / 100)
				resp, err = client.Get("http://localhost:12345/testpath")
			}
			if c.Check(err, check.IsNil) {
				c.Logf("resp StatusCode %d", resp.StatusCode)
			}
		}()
	}
	for i := 0; i < max; i++ {
		select {
		case <-started:
		case <-time.After(time.Second):
			c.Logf("%s", stderr.String())
			panic("timed out")
		}
	}
	for {
		j, err := os.ReadFile(tmpdir + "/arvados-controller-requests.json")
		if os.IsNotExist(err) && deadline.After(time.Now()) {
			time.Sleep(time.Second / 100)
			continue
		}
		c.Check(err, check.IsNil)
		c.Logf("%s", stderr.String())
		c.Logf("%s", string(j))

		var loaded []struct{ URL string }
		err = json.Unmarshal(j, &loaded)
		c.Check(err, check.IsNil)
		c.Check(loaded, check.HasLen, max)
		c.Check(loaded[0].URL, check.Equals, "/testpath")
		break
	}
	close(hold)
	cancel()

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
