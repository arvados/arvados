// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/crunchrun"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	check "gopkg.in/check.v1"
)

func (s *ClientSuite) TestShellGatewayNotAvailable(c *check.C) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".", "shell", arvadostest.QueuedContainerUUID, "-o", "controlpath=none", "echo", "ok")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.Check(cmd.Run(), check.NotNil)
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*container is not running yet \(state is "Queued"\).*`)
}

func (s *ClientSuite) TestShellGateway(c *check.C) {
	defer func() {
		c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
	}()
	uuid := arvadostest.QueuedContainerUUID
	h := hmac.New(sha256.New, []byte(arvadostest.SystemRootToken))
	fmt.Fprint(h, uuid)
	authSecret := fmt.Sprintf("%x", h.Sum(nil))
	dcid := "theperthcountyconspiracy"
	gw := crunchrun.Gateway{
		DockerContainerID: &dcid,
		ContainerUUID:     uuid,
		Address:           "0.0.0.0:0",
		AuthSecret:        authSecret,
		// Just forward connections to localhost instead of a
		// container, so we can test without running a
		// container.
		ContainerIPAddress: func() (string, error) { return "0.0.0.0", nil },
	}
	err := gw.Start()
	c.Assert(err, check.IsNil)

	rpcconn := rpc.NewConn("",
		&url.URL{
			Scheme: "https",
			Host:   os.Getenv("ARVADOS_API_HOST"),
		},
		true,
		func(context.Context) ([]string, error) {
			return []string{arvadostest.SystemRootToken}, nil
		})
	_, err = rpcconn.ContainerUpdate(context.TODO(), arvados.UpdateOptions{UUID: uuid, Attrs: map[string]interface{}{
		"state": arvados.ContainerStateLocked,
	}})
	c.Assert(err, check.IsNil)
	_, err = rpcconn.ContainerUpdate(context.TODO(), arvados.UpdateOptions{UUID: uuid, Attrs: map[string]interface{}{
		"state":           arvados.ContainerStateRunning,
		"gateway_address": gw.Address,
	}})
	c.Assert(err, check.IsNil)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("go", "run", ".", "shell", uuid, "-o", "controlpath=none", "-o", "userknownhostsfile="+c.MkDir()+"/known_hosts", "echo", "ok")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.Check(cmd.Run(), check.NotNil)
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*(No such container: theperthcountyconspiracy|exec: \"docker\": executable file not found in \$PATH).*`)

	// Set up an http server, and try using "arvados-client shell"
	// to forward traffic to it.
	httpTarget := &httpserver.Server{}
	httpTarget.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.Logf("httpTarget.Handler: incoming request: %s %s", r.Method, r.URL)
		if r.URL.Path == "/foo" {
			fmt.Fprintln(w, "bar baz")
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	err = httpTarget.Start()
	c.Assert(err, check.IsNil)

	ln, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, forwardedPort, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()

	stdout.Reset()
	stderr.Reset()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	cmd = exec.CommandContext(ctx,
		"go", "run", ".", "shell", uuid,
		"-L", forwardedPort+":"+httpTarget.Addr,
		"-o", "controlpath=none",
		"-o", "userknownhostsfile="+c.MkDir()+"/known_hosts",
		"-N",
	)
	c.Logf("cmd.Args: %s", cmd.Args)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Start()

	forwardedURL := fmt.Sprintf("http://localhost:%s/foo", forwardedPort)

	for range time.NewTicker(time.Second / 20).C {
		resp, err := http.Get(forwardedURL)
		if err != nil {
			if !strings.Contains(err.Error(), "connect") {
				c.Fatal(err)
			} else if ctx.Err() != nil {
				if cmd.Process.Signal(syscall.Signal(0)) != nil {
					c.Error("OpenSSH exited")
				} else {
					c.Errorf("timed out trying to connect: %s", err)
				}
				c.Logf("OpenSSH stdout:\n%s", stdout.String())
				c.Logf("OpenSSH stderr:\n%s", stderr.String())
				c.FailNow()
			}
			// Retry until OpenSSH starts listening
			continue
		}
		c.Check(resp.StatusCode, check.Equals, http.StatusOK)
		body, err := ioutil.ReadAll(resp.Body)
		c.Check(err, check.IsNil)
		c.Check(string(body), check.Equals, "bar baz\n")
		break
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(forwardedURL)
			if !c.Check(err, check.IsNil) {
				return
			}
			body, err := ioutil.ReadAll(resp.Body)
			c.Check(err, check.IsNil)
			c.Check(string(body), check.Equals, "bar baz\n")
		}()
	}
	wg.Wait()
}
