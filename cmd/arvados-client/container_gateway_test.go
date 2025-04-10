// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
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
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&shellSuite{})

type shellSuite struct {
	gobindir    string
	homedir     string
	runningUUID string
}

func (s *shellSuite) SetUpSuite(c *check.C) {
	tmpdir := c.MkDir()
	s.gobindir = tmpdir + "/bin"
	c.Check(os.Mkdir(s.gobindir, 0777), check.IsNil)
	s.homedir = tmpdir + "/home"
	c.Check(os.Mkdir(s.homedir, 0777), check.IsNil)

	// We explicitly build a client binary in our tempdir here,
	// instead of using "go run .", because (a) we're going to
	// invoke the same binary several times, and (b) we're going
	// to change $HOME to a temp dir in some of the tests, which
	// would force "go run ." to recompile the world instead of
	// using the cached object files in the real $HOME.
	c.Logf("building arvados-client binary in %s", s.gobindir)
	cmd := exec.Command("go", "install", ".")
	cmd.Env = append(os.Environ(), "GOBIN="+s.gobindir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	c.Assert(cmd.Run(), check.IsNil)

	s.runningUUID = arvadostest.RunningContainerUUID
	h := hmac.New(sha256.New, []byte(arvadostest.SystemRootToken))
	fmt.Fprint(h, s.runningUUID)
	authSecret := fmt.Sprintf("%x", h.Sum(nil))
	gw := crunchrun.Gateway{
		ContainerUUID: s.runningUUID,
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
	_, err = rpcconn.ContainerUpdate(context.TODO(), arvados.UpdateOptions{UUID: s.runningUUID, Attrs: map[string]interface{}{
		"gateway_address": gw.Address,
	}})
	c.Assert(err, check.IsNil)
}

func (s *shellSuite) TearDownSuite(c *check.C) {
	c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
}

func (s *shellSuite) TestShellGatewayNotAvailable(c *check.C) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(s.gobindir+"/arvados-client", "shell", arvadostest.QueuedContainerUUID, "-o", "controlpath=none", "echo", "ok")
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.Check(cmd.Run(), check.NotNil)
	c.Log(stderr.String())
	c.Check(stderr.String(), check.Matches, `(?ms).*container is not running yet \(state is "Queued"\).*`)
}

func (s *shellSuite) TestShellGatewayUsingEnvVars(c *check.C) {
	s.testShellGateway(c, false)
}
func (s *shellSuite) TestShellGatewayUsingSettingsConf(c *check.C) {
	s.testShellGateway(c, true)
}
func (s *shellSuite) testShellGateway(c *check.C, useSettingsConf bool) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(
		s.gobindir+"/arvados-client", "shell", s.runningUUID,
		"-o", "controlpath=none",
		"-o", "userknownhostsfile="+s.homedir+"/known_hosts",
		"echo", "ok")
	if useSettingsConf {
		settings := "ARVADOS_API_HOST=" + os.Getenv("ARVADOS_API_HOST") + "\nARVADOS_API_TOKEN=" + arvadostest.ActiveTokenV2 + "\nARVADOS_API_HOST_INSECURE=true\n"
		err := os.MkdirAll(s.homedir+"/.config/arvados", 0777)
		c.Assert(err, check.IsNil)
		err = os.WriteFile(s.homedir+"/.config/arvados/settings.conf", []byte(settings), 0777)
		c.Assert(err, check.IsNil)
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, "ARVADOS_") && !strings.HasPrefix(kv, "HOME=") {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		cmd.Env = append(cmd.Env, "HOME="+s.homedir)
	} else {
		err := os.Remove(s.homedir + "/.config/arvados/settings.conf")
		if !os.IsNotExist(err) {
			c.Assert(err, check.IsNil)
		}
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	c.Assert(err, check.IsNil)
	go fmt.Fprintln(stdin, "data appears on stdin, but stdin does not close; cmd should exit anyway, not hang")
	timeout := time.AfterFunc(5*time.Second, func() {
		c.Errorf("timed out -- remote end is probably hung waiting for us to close stdin")
		stdin.Close()
	})
	c.Logf("cmd.Args: %s", cmd.Args)
	c.Check(cmd.Run(), check.IsNil)
	timeout.Stop()
	c.Check(stdout.String(), check.Equals, "ok\n")
}

func stubHTTPTarget(c *check.C) *httpserver.Server {
	c.Log("setting up an http server")
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
	err := httpTarget.Start()
	c.Assert(err, check.IsNil)
	return httpTarget
}

func (s *shellSuite) TestShellGatewayPortForwarding(c *check.C) {
	httpTarget := stubHTTPTarget(c)

	ln, err := net.Listen("tcp", ":0")
	c.Assert(err, check.IsNil)
	_, forwardedPort, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()

	c.Log("connecting")
	var stdout, stderr bytes.Buffer
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	cmd := exec.CommandContext(ctx,
		s.gobindir+"/arvados-client", "shell", s.runningUUID,
		"-L", forwardedPort+":"+httpTarget.Addr,
		"-o", "controlpath=none",
		"-o", "userknownhostsfile="+s.homedir+"/known_hosts",
		"-N",
	)
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.ActiveTokenV2)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	c.Logf("cmd.Args: %s", cmd.Args)
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

// This test is arguably misplaced: arvados-client does not (yet?)
// have a "do http request against container X port Y" feature, so
// we're not really testing arvados-client here.  However, (a) it
// might have one someday, and (b) testing the same http server setup
// via both access mechanisms might help troubleshoot if one of them
// fails.
func (s *shellSuite) TestGatewayHTTPProxy(c *check.C) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	httpTarget := stubHTTPTarget(c)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	_, port, _ := net.SplitHostPort(httpTarget.Addr)
	req, err := http.NewRequestWithContext(ctx, "GET", "https://"+os.Getenv("ARVADOS_API_HOST")+"/foo", nil)
	c.Assert(err, check.IsNil)
	req.AddCookie(&http.Cookie{Name: "arvados_api_token", Value: auth.EncodeTokenCookie([]byte(arvadostest.ActiveTokenV2))})
	req.Host = s.runningUUID + "-" + port + ".example.com"
	resp, err := client.Do(req)
	c.Assert(err, check.IsNil)
	c.Check(resp.StatusCode, check.Equals, http.StatusOK)
	body, err := ioutil.ReadAll(resp.Body)
	c.Check(err, check.IsNil)
	c.Check(string(body), check.Equals, "bar baz\n")
}

var _ = check.Suite(&logsSuite{})

type logsSuite struct{}

func (s *logsSuite) TestContainerRequestLog(c *check.C) {
	arvadostest.StartKeep(2, true)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(30*time.Second))
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
	cmd := exec.CommandContext(ctx, "go", "run", ".", "logs", "-f", "-poll=250ms", cr.UUID)
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

	const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"
	fCrunchrun, err := cfs.OpenFile("crunch-run.txt", os.O_CREATE|os.O_WRONLY, 0777)
	c.Assert(err, check.IsNil)
	_, err = fmt.Fprintf(fCrunchrun, "%s line 1 of crunch-run.txt\n", time.Now().UTC().Format(rfc3339NanoFixed))
	c.Assert(err, check.IsNil)
	fStderr, err := cfs.OpenFile("stderr.txt", os.O_CREATE|os.O_WRONLY, 0777)
	c.Assert(err, check.IsNil)
	_, err = fmt.Fprintf(fStderr, "%s line 1 of stderr\n", time.Now().UTC().Format(rfc3339NanoFixed))
	c.Assert(err, check.IsNil)

	{
		// Without "-f", just show the existing logs and
		// exit. Timeout needs to be long enough for "go run".
		ctxNoFollow, cancel := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
		defer cancel()
		cmdNoFollow := exec.CommandContext(ctxNoFollow, "go", "run", ".", "logs", "-poll=250ms", cr.UUID)
		buf, err := cmdNoFollow.CombinedOutput()
		c.Check(err, check.IsNil)
		c.Check(string(buf), check.Matches, `(?ms).*line 1 of stderr\n`)
	}

	time.Sleep(time.Second * 2)
	_, err = fmt.Fprintf(fCrunchrun, "%s line 2 of crunch-run.txt", time.Now().UTC().Format(rfc3339NanoFixed))
	c.Assert(err, check.IsNil)
	_, err = fmt.Fprintf(fStderr, "%s --end--", time.Now().UTC().Format(rfc3339NanoFixed))
	c.Assert(err, check.IsNil)

	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline) && !strings.Contains(stdout.String(), "--end--"); time.Sleep(time.Second / 10) {
	}
	c.Check(stdout.String(), check.Matches, `(?ms).*stderr\.txt +20\S+Z --end--\n.*`)

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
		cmd := exec.CommandContext(ctx, "go", "run", ".", "logs", "-f", cr.UUID)
		cmd.Env = append(cmd.Env, os.Environ()...)
		cmd.Env = append(cmd.Env, "ARVADOS_API_TOKEN="+arvadostest.SystemRootToken)
		buf, err := cmd.CombinedOutput()
		c.Check(err, check.Equals, nil)
		c.Check(string(buf), check.Matches, `(?ms).*--end--\n`)
	}
}
