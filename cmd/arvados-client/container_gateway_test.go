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
	"io"
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
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
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
	gw := crunchrun.Gateway{
		ContainerUUID: uuid,
		Address:       "0.0.0.0:0",
		AuthSecret:    authSecret,
		Log:           ctxlog.TestLogger(c),
		// Just forward connections to localhost instead of a
		// container, so we can test without running a
		// container.
		Target: crunchrun.GatewayTargetStub{},
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
	stdin, err := cmd.StdinPipe()
	c.Assert(err, check.IsNil)
	go fmt.Fprintln(stdin, "data appears on stdin, but stdin does not close; cmd should exit anyway, not hang")
	time.AfterFunc(5*time.Second, func() {
		c.Errorf("timed out -- remote end is probably hung waiting for us to close stdin")
		stdin.Close()
	})
	c.Check(cmd.Run(), check.IsNil)
	c.Check(stdout.String(), check.Equals, "ok\n")

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

func (s *ClientSuite) TestContainerLog(c *check.C) {
	arvadostest.StartKeep(2, true)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(20*time.Second))
	defer cancel()

	rpcconn := rpc.NewConn("",
		&url.URL{
			Scheme: "https",
			Host:   os.Getenv("ARVADOS_API_HOST"),
		},
		true,
		func(context.Context) ([]string, error) {
			return []string{arvadostest.SystemRootToken}, nil
		})
	imageColl, err := rpcconn.CollectionCreate(ctx, arvados.CreateOptions{Attrs: map[string]interface{}{
		"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855.tar\n",
	}})
	c.Assert(err, check.IsNil)
	c.Logf("imageColl %+v", imageColl)
	cr, err := rpcconn.ContainerRequestCreate(ctx, arvados.CreateOptions{Attrs: map[string]interface{}{
		"state":           "Committed",
		"command":         []string{"echo", fmt.Sprintf("%d", time.Now().Unix())},
		"container_image": imageColl.PortableDataHash,
		"cwd":             "/",
		"output_path":     "/",
		"priority":        1,
		"runtime_constraints": arvados.RuntimeConstraints{
			VCPUs: 1,
			RAM:   1000000000,
		},
		"container_count_max": 1,
	}})
	c.Assert(err, check.IsNil)
	h := hmac.New(sha256.New, []byte(arvadostest.SystemRootToken))
	fmt.Fprint(h, cr.ContainerUUID)
	authSecret := fmt.Sprintf("%x", h.Sum(nil))

	coll := arvados.Collection{}
	client := arvados.NewClientFromEnv()
	ac, err := arvadosclient.New(client)
	c.Assert(err, check.IsNil)
	kc, err := keepclient.MakeKeepClient(ac)
	c.Assert(err, check.IsNil)
	cfs, err := coll.FileSystem(client, kc)
	c.Assert(err, check.IsNil)

	c.Log("running logs command on queued container")
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "go", "run", ".", "logs", "-poll=250ms", cr.UUID)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.SystemRootToken)
	cmd.Stdout = io.MultiWriter(&stdout, os.Stderr)
	cmd.Stderr = io.MultiWriter(&stderr, os.Stderr)
	err = cmd.Start()
	c.Assert(err, check.Equals, nil)

	c.Log("changing container state to Locked")
	_, err = rpcconn.ContainerUpdate(ctx, arvados.UpdateOptions{UUID: cr.ContainerUUID, Attrs: map[string]interface{}{
		"state": arvados.ContainerStateLocked,
	}})
	c.Assert(err, check.IsNil)
	c.Log("starting gateway")
	gw := crunchrun.Gateway{
		ContainerUUID: cr.ContainerUUID,
		Address:       "0.0.0.0:0",
		AuthSecret:    authSecret,
		Log:           ctxlog.TestLogger(c),
		Target:        crunchrun.GatewayTargetStub{},
		LogCollection: cfs,
	}
	err = gw.Start()
	c.Assert(err, check.IsNil)
	c.Log("updating container gateway address")
	_, err = rpcconn.ContainerUpdate(ctx, arvados.UpdateOptions{UUID: cr.ContainerUUID, Attrs: map[string]interface{}{
		"gateway_address": gw.Address,
		"state":           arvados.ContainerStateRunning,
	}})
	c.Assert(err, check.IsNil)

	fCrunchrun, err := cfs.OpenFile("crunch-run.txt", os.O_CREATE|os.O_WRONLY, 0777)
	c.Assert(err, check.IsNil)
	_, err = fmt.Fprintln(fCrunchrun, "line 1 of crunch-run.txt")
	c.Assert(err, check.IsNil)
	fStderr, err := cfs.OpenFile("stderr.txt", os.O_CREATE|os.O_WRONLY, 0777)
	c.Assert(err, check.IsNil)
	_, err = fmt.Fprintln(fStderr, "line 1 of stderr")
	c.Assert(err, check.IsNil)
	time.Sleep(time.Second * 2)
	_, err = fmt.Fprintln(fCrunchrun, "line 2 of crunch-run.txt")
	c.Assert(err, check.IsNil)
	_, err = fmt.Fprintln(fStderr, "--end--")
	c.Assert(err, check.IsNil)

	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline) && !strings.Contains(stdout.String(), "--end--"); time.Sleep(time.Second / 10) {
	}
	c.Check(stdout.String(), check.Matches, `(?ms).*--end--\n.*`)

	mtxt, err := cfs.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	savedLog, err := rpcconn.CollectionCreate(ctx, arvados.CreateOptions{Attrs: map[string]interface{}{
		"manifest_text": mtxt,
	}})
	c.Assert(err, check.IsNil)
	_, err = rpcconn.ContainerUpdate(ctx, arvados.UpdateOptions{UUID: cr.ContainerUUID, Attrs: map[string]interface{}{
		"state":     arvados.ContainerStateComplete,
		"log":       savedLog.PortableDataHash,
		"output":    "d41d8cd98f00b204e9800998ecf8427e+0",
		"exit_code": 0,
	}})
	c.Assert(err, check.IsNil)

	err = cmd.Wait()
	c.Check(err, check.IsNil)
	// Ensure controller doesn't cheat by fetching data from the
	// gateway after the container is complete.
	gw.LogCollection = nil

	c.Logf("re-running logs command on completed container")
	{
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
		defer cancel()
		cmd := exec.CommandContext(ctx, "go", "run", ".", "logs", cr.UUID)
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.SystemRootToken)
		buf, err := cmd.CombinedOutput()
		c.Check(err, check.Equals, nil)
		c.Check(string(buf), check.Matches, `(?ms).*--end--\n`)
	}
}
