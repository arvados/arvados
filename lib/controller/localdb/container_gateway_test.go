// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/controller/router"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/crunchrun"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&ContainerGatewaySuite{})

type ContainerGatewaySuite struct {
	localdbSuite
	ctrUUID string
	gw      *crunchrun.Gateway
}

func (s *ContainerGatewaySuite) SetUpTest(c *check.C) {
	s.localdbSuite.SetUpTest(c)

	s.ctrUUID = arvadostest.QueuedContainerUUID

	h := hmac.New(sha256.New, []byte(s.cluster.SystemRootToken))
	fmt.Fprint(h, s.ctrUUID)
	authKey := fmt.Sprintf("%x", h.Sum(nil))

	rtr := router.New(s.localdb, router.Config{})
	srv := httptest.NewUnstartedServer(rtr)
	srv.StartTLS()
	// the test setup doesn't use lib/service so
	// service.URLFromContext() returns nothing -- instead, this
	// is how we advertise our internal URL and enable
	// proxy-to-other-controller mode,
	forceInternalURLForTest = &arvados.URL{Scheme: "https", Host: srv.Listener.Addr().String()}
	ac := &arvados.Client{
		APIHost:   srv.Listener.Addr().String(),
		AuthToken: arvadostest.Dispatch1Token,
		Insecure:  true,
	}
	s.gw = &crunchrun.Gateway{
		ContainerUUID: s.ctrUUID,
		AuthSecret:    authKey,
		Address:       "localhost:0",
		Log:           ctxlog.TestLogger(c),
		Target:        crunchrun.GatewayTargetStub{},
		ArvadosClient: ac,
	}
	c.Assert(s.gw.Start(), check.IsNil)
	rootctx := ctrlctx.NewWithToken(s.ctx, s.cluster, s.cluster.SystemRootToken)
	// OK if this line fails (because state is already Running
	// from a previous test case) as long as the following line
	// succeeds:
	s.localdb.ContainerUpdate(rootctx, arvados.UpdateOptions{
		UUID: s.ctrUUID,
		Attrs: map[string]interface{}{
			"state": arvados.ContainerStateLocked}})
	_, err := s.localdb.ContainerUpdate(rootctx, arvados.UpdateOptions{
		UUID: s.ctrUUID,
		Attrs: map[string]interface{}{
			"state":           arvados.ContainerStateRunning,
			"gateway_address": s.gw.Address}})
	c.Assert(err, check.IsNil)

	s.cluster.Containers.ShellAccess.Admin = true
	s.cluster.Containers.ShellAccess.User = true
	_, err = s.db.Exec(`update containers set interactive_session_started=$1 where uuid=$2`, false, s.ctrUUID)
	c.Check(err, check.IsNil)
}

func (s *ContainerGatewaySuite) TestConfig(c *check.C) {
	for _, trial := range []struct {
		configAdmin bool
		configUser  bool
		sendToken   string
		errorCode   int
	}{
		{true, true, arvadostest.ActiveTokenV2, 0},
		{true, false, arvadostest.ActiveTokenV2, 503},
		{false, true, arvadostest.ActiveTokenV2, 0},
		{false, false, arvadostest.ActiveTokenV2, 503},
		{true, true, arvadostest.AdminToken, 0},
		{true, false, arvadostest.AdminToken, 0},
		{false, true, arvadostest.AdminToken, 403},
		{false, false, arvadostest.AdminToken, 503},
	} {
		c.Logf("trial %#v", trial)
		s.cluster.Containers.ShellAccess.Admin = trial.configAdmin
		s.cluster.Containers.ShellAccess.User = trial.configUser
		ctx := ctrlctx.NewWithToken(s.ctx, s.cluster, trial.sendToken)
		sshconn, err := s.localdb.ContainerSSH(ctx, arvados.ContainerSSHOptions{UUID: s.ctrUUID})
		if trial.errorCode == 0 {
			if !c.Check(err, check.IsNil) {
				continue
			}
			if !c.Check(sshconn.Conn, check.NotNil) {
				continue
			}
			sshconn.Conn.Close()
		} else {
			c.Check(err, check.NotNil)
			err, ok := err.(interface{ HTTPStatus() int })
			if c.Check(ok, check.Equals, true) {
				c.Check(err.HTTPStatus(), check.Equals, trial.errorCode)
			}
		}
	}
}

func (s *ContainerGatewaySuite) TestDirectTCP(c *check.C) {
	// Set up servers on a few TCP ports
	var addrs []string
	for i := 0; i < 3; i++ {
		ln, err := net.Listen("tcp", ":0")
		c.Assert(err, check.IsNil)
		defer ln.Close()
		addrs = append(addrs, ln.Addr().String())
		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				var gotAddr string
				fmt.Fscanf(conn, "%s\n", &gotAddr)
				c.Logf("stub server listening at %s received string %q from remote %s", ln.Addr().String(), gotAddr, conn.RemoteAddr())
				if gotAddr == ln.Addr().String() {
					fmt.Fprintf(conn, "%s\n", ln.Addr().String())
				}
				conn.Close()
			}
		}()
	}

	c.Logf("connecting to %s", s.gw.Address)
	sshconn, err := s.localdb.ContainerSSH(s.userctx, arvados.ContainerSSHOptions{UUID: s.ctrUUID})
	c.Assert(err, check.IsNil)
	c.Assert(sshconn.Conn, check.NotNil)
	defer sshconn.Conn.Close()
	conn, chans, reqs, err := ssh.NewClientConn(sshconn.Conn, "zzzz-dz642-abcdeabcdeabcde", &ssh.ClientConfig{
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil },
	})
	c.Assert(err, check.IsNil)
	client := ssh.NewClient(conn, chans, reqs)
	for _, expectAddr := range addrs {
		_, port, err := net.SplitHostPort(expectAddr)
		c.Assert(err, check.IsNil)

		c.Logf("trying foo:%s", port)
		{
			conn, err := client.Dial("tcp", "foo:"+port)
			c.Assert(err, check.IsNil)
			conn.SetDeadline(time.Now().Add(time.Second))
			buf, err := ioutil.ReadAll(conn)
			c.Check(err, check.IsNil)
			c.Check(string(buf), check.Equals, "")
		}

		c.Logf("trying localhost:%s", port)
		{
			conn, err := client.Dial("tcp", "localhost:"+port)
			c.Assert(err, check.IsNil)
			conn.SetDeadline(time.Now().Add(time.Second))
			conn.Write([]byte(expectAddr + "\n"))
			var gotAddr string
			fmt.Fscanf(conn, "%s\n", &gotAddr)
			c.Check(gotAddr, check.Equals, expectAddr)
		}
	}
}

func (s *ContainerGatewaySuite) setupLogCollection(c *check.C, files map[string]string) {
	client := arvados.NewClientFromEnv()
	ac, err := arvadosclient.New(client)
	c.Assert(err, check.IsNil)
	kc, err := keepclient.MakeKeepClient(ac)
	c.Assert(err, check.IsNil)
	cfs, err := (&arvados.Collection{}).FileSystem(client, kc)
	c.Assert(err, check.IsNil)
	for name, content := range files {
		for i, ch := range name {
			if ch == '/' {
				err := cfs.Mkdir("/"+name[:i], 0777)
				c.Assert(err, check.IsNil)
			}
		}
		f, err := cfs.OpenFile("/"+name, os.O_CREATE|os.O_WRONLY, 0777)
		c.Assert(err, check.IsNil)
		f.Write([]byte(content))
		err = f.Close()
		c.Assert(err, check.IsNil)
	}
	cfs.Sync()
	s.gw.LogCollection = cfs
}

func (s *ContainerGatewaySuite) TestContainerLogViaTunnel(c *check.C) {
	forceProxyForTest = true
	defer func() { forceProxyForTest = false }()

	s.gw = s.setupGatewayWithTunnel(c)
	s.setupLogCollection(c, map[string]string{
		"stderr.txt": "hello world\n",
	})

	for _, broken := range []bool{false, true} {
		c.Logf("broken=%v", broken)

		if broken {
			delete(s.cluster.Services.Controller.InternalURLs, *forceInternalURLForTest)
		} else {
			s.cluster.Services.Controller.InternalURLs[*forceInternalURLForTest] = arvados.ServiceInstance{}
			defer delete(s.cluster.Services.Controller.InternalURLs, *forceInternalURLForTest)
		}

		handler, err := s.localdb.ContainerLog(s.userctx, arvados.ContainerLogOptions{
			UUID:          s.ctrUUID,
			WebDAVOptions: arvados.WebDAVOptions{Path: "/stderr.txt"},
		})
		if broken {
			c.Check(err, check.ErrorMatches, `.*tunnel endpoint is invalid.*`)
			continue
		}
		c.Check(err, check.IsNil)
		c.Assert(handler, check.NotNil)
		r, err := http.NewRequestWithContext(s.userctx, "GET", "https://controller.example/arvados/v1/containers/"+s.ctrUUID+"/log/stderr.txt", nil)
		c.Assert(err, check.IsNil)
		r.Header.Set("Authorization", "Bearer "+arvadostest.ActiveTokenV2)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		resp := rec.Result()
		c.Check(resp.StatusCode, check.Equals, http.StatusOK)
		buf, err := ioutil.ReadAll(resp.Body)
		c.Check(err, check.IsNil)
		c.Check(string(buf), check.Equals, "hello world\n")
	}
}

func (s *ContainerGatewaySuite) TestContainerLogViaGateway(c *check.C) {
	s.testContainerLog(c, true)
}

func (s *ContainerGatewaySuite) TestContainerLogViaKeepWeb(c *check.C) {
	s.testContainerLog(c, false)
}

func (s *ContainerGatewaySuite) testContainerLog(c *check.C, viaGateway bool) {
	s.setupLogCollection(c, map[string]string{
		"stderr.txt":   "hello world\n",
		"a/b/c/d.html": "<html></html>\n",
	})
	if !viaGateway {
		rootctx := ctrlctx.NewWithToken(s.ctx, s.cluster, s.cluster.SystemRootToken)
		txt, err := s.gw.LogCollection.MarshalManifest(".")
		c.Assert(err, check.IsNil)
		coll, err := s.localdb.CollectionCreate(rootctx, arvados.CreateOptions{
			Attrs: map[string]interface{}{
				"manifest_text": txt,
			}})
		c.Assert(err, check.IsNil)
		_, err = s.localdb.ContainerUpdate(rootctx, arvados.UpdateOptions{
			UUID: s.ctrUUID,
			Attrs: map[string]interface{}{
				"log":             coll.PortableDataHash,
				"gateway_address": "",
			}})
		c.Assert(err, check.IsNil)
		// gateway_address="" above already ensures localdb
		// can't circumvent the keep-web proxy test by getting
		// content from the container gateway; this is just
		// extra insurance.
		s.gw.LogCollection = nil
	}
	for _, trial := range []struct {
		method       string
		path         string
		header       http.Header
		expectStatus int
		expectBodyRe string
		expectHeader http.Header
	}{
		{
			method:       "GET",
			path:         "/stderr.txt",
			expectStatus: http.StatusOK,
			expectBodyRe: "hello world\n",
			expectHeader: http.Header{
				"Content-Type": {"text/plain; charset=utf-8"},
			},
		},
		{
			method: "GET",
			path:   "/stderr.txt",
			header: http.Header{
				"Range": {"bytes=-6"},
			},
			expectStatus: http.StatusPartialContent,
			expectBodyRe: "world\n",
			expectHeader: http.Header{
				"Content-Type":  {"text/plain; charset=utf-8"},
				"Content-Range": {"bytes 6-11/12"},
			},
		},
		{
			method:       "OPTIONS",
			path:         "/stderr.txt",
			expectStatus: http.StatusOK,
			expectBodyRe: "",
			expectHeader: http.Header{
				"Dav":   {"1, 2"},
				"Allow": {"OPTIONS, LOCK, GET, HEAD, POST, DELETE, PROPPATCH, COPY, MOVE, UNLOCK, PROPFIND, PUT"},
			},
		},
		{
			method:       "PROPFIND",
			path:         "",
			expectStatus: http.StatusMultiStatus,
			expectBodyRe: `.*\Q<D:displayname>stderr.txt</D:displayname>\E.*`,
			expectHeader: http.Header{
				"Content-Type": {"text/xml; charset=utf-8"},
			},
		},
		{
			method:       "PROPFIND",
			path:         "/a/b/c/",
			expectStatus: http.StatusMultiStatus,
			expectBodyRe: `.*\Q<D:displayname>d.html</D:displayname>\E.*`,
			expectHeader: http.Header{
				"Content-Type": {"text/xml; charset=utf-8"},
			},
		},
		{
			method:       "GET",
			path:         "/a/b/c/d.html",
			expectStatus: http.StatusOK,
			expectBodyRe: "<html></html>\n",
			expectHeader: http.Header{
				"Content-Type": {"text/html; charset=utf-8"},
			},
		},
	} {
		c.Logf("trial %#v", trial)
		handler, err := s.localdb.ContainerLog(s.userctx, arvados.ContainerLogOptions{
			UUID:          s.ctrUUID,
			WebDAVOptions: arvados.WebDAVOptions{Path: trial.path},
		})
		c.Assert(err, check.IsNil)
		c.Assert(handler, check.NotNil)
		r, err := http.NewRequestWithContext(s.userctx, trial.method, "https://controller.example/arvados/v1/containers/"+s.ctrUUID+"/log"+trial.path, nil)
		c.Assert(err, check.IsNil)
		for k := range trial.header {
			r.Header.Set(k, trial.header.Get(k))
		}
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		resp := rec.Result()
		c.Check(resp.StatusCode, check.Equals, trial.expectStatus)
		for k := range trial.expectHeader {
			c.Check(resp.Header.Get(k), check.Equals, trial.expectHeader.Get(k))
		}
		buf, err := ioutil.ReadAll(resp.Body)
		c.Check(err, check.IsNil)
		c.Check(string(buf), check.Matches, trial.expectBodyRe)
	}
}

func (s *ContainerGatewaySuite) TestConnect(c *check.C) {
	c.Logf("connecting to %s", s.gw.Address)
	sshconn, err := s.localdb.ContainerSSH(s.userctx, arvados.ContainerSSHOptions{UUID: s.ctrUUID})
	c.Assert(err, check.IsNil)
	c.Assert(sshconn.Conn, check.NotNil)
	defer sshconn.Conn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Receive text banner
		buf := make([]byte, 12)
		_, err := io.ReadFull(sshconn.Conn, buf)
		c.Check(err, check.IsNil)
		c.Check(string(buf), check.Equals, "SSH-2.0-Go\r\n")

		// Send text banner
		_, err = sshconn.Conn.Write([]byte("SSH-2.0-Fake\r\n"))
		c.Check(err, check.IsNil)

		// Receive binary
		_, err = io.ReadFull(sshconn.Conn, buf[:4])
		c.Check(err, check.IsNil)

		// If we can get this far into an SSH handshake...
		c.Logf("was able to read %x -- success, tunnel is working", buf[:4])
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		c.Fail()
	}
	ctr, err := s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.ctrUUID})
	c.Check(err, check.IsNil)
	c.Check(ctr.InteractiveSessionStarted, check.Equals, true)
}

func (s *ContainerGatewaySuite) TestConnectFail(c *check.C) {
	c.Log("trying with no token")
	ctx := ctrlctx.NewWithToken(s.ctx, s.cluster, "")
	_, err := s.localdb.ContainerSSH(ctx, arvados.ContainerSSHOptions{UUID: s.ctrUUID})
	c.Check(err, check.ErrorMatches, `.* 401 .*`)

	c.Log("trying with anonymous token")
	ctx = ctrlctx.NewWithToken(s.ctx, s.cluster, arvadostest.AnonymousToken)
	_, err = s.localdb.ContainerSSH(ctx, arvados.ContainerSSHOptions{UUID: s.ctrUUID})
	c.Check(err, check.ErrorMatches, `.* 404 .*`)
}

func (s *ContainerGatewaySuite) TestCreateTunnel(c *check.C) {
	// no AuthSecret
	conn, err := s.localdb.ContainerGatewayTunnel(s.userctx, arvados.ContainerGatewayTunnelOptions{
		UUID: s.ctrUUID,
	})
	c.Check(err, check.ErrorMatches, `authentication error`)
	c.Check(conn.Conn, check.IsNil)

	// bogus AuthSecret
	conn, err = s.localdb.ContainerGatewayTunnel(s.userctx, arvados.ContainerGatewayTunnelOptions{
		UUID:       s.ctrUUID,
		AuthSecret: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	})
	c.Check(err, check.ErrorMatches, `authentication error`)
	c.Check(conn.Conn, check.IsNil)

	// good AuthSecret
	conn, err = s.localdb.ContainerGatewayTunnel(s.userctx, arvados.ContainerGatewayTunnelOptions{
		UUID:       s.ctrUUID,
		AuthSecret: s.gw.AuthSecret,
	})
	c.Check(err, check.IsNil)
	c.Check(conn.Conn, check.NotNil)
}

func (s *ContainerGatewaySuite) TestConnectThroughTunnelWithProxyOK(c *check.C) {
	forceProxyForTest = true
	defer func() { forceProxyForTest = false }()
	s.cluster.Services.Controller.InternalURLs[*forceInternalURLForTest] = arvados.ServiceInstance{}
	defer delete(s.cluster.Services.Controller.InternalURLs, *forceInternalURLForTest)
	s.testConnectThroughTunnel(c, "")
}

func (s *ContainerGatewaySuite) TestConnectThroughTunnelWithProxyError(c *check.C) {
	forceProxyForTest = true
	defer func() { forceProxyForTest = false }()
	// forceInternalURLForTest will not be usable because it isn't
	// listed in s.cluster.Services.Controller.InternalURLs
	s.testConnectThroughTunnel(c, `.*tunnel endpoint is invalid.*`)
}

func (s *ContainerGatewaySuite) TestConnectThroughTunnelNoProxyOK(c *check.C) {
	s.testConnectThroughTunnel(c, "")
}

func (s *ContainerGatewaySuite) setupGatewayWithTunnel(c *check.C) *crunchrun.Gateway {
	rootctx := ctrlctx.NewWithToken(s.ctx, s.cluster, s.cluster.SystemRootToken)
	// Until the tunnel starts up, set gateway_address to a value
	// that can't work. We want to ensure the only way we can
	// reach the gateway is through the tunnel.
	tungw := &crunchrun.Gateway{
		ContainerUUID: s.ctrUUID,
		AuthSecret:    s.gw.AuthSecret,
		Log:           ctxlog.TestLogger(c),
		Target:        crunchrun.GatewayTargetStub{},
		ArvadosClient: s.gw.ArvadosClient,
		UpdateTunnelURL: func(url string) {
			c.Logf("UpdateTunnelURL(%q)", url)
			gwaddr := "tunnel " + url
			s.localdb.ContainerUpdate(rootctx, arvados.UpdateOptions{
				UUID: s.ctrUUID,
				Attrs: map[string]interface{}{
					"gateway_address": gwaddr}})
		},
	}
	c.Assert(tungw.Start(), check.IsNil)

	// We didn't supply an external hostname in the Address field,
	// so Start() should assign a local address.
	host, _, err := net.SplitHostPort(tungw.Address)
	c.Assert(err, check.IsNil)
	c.Check(host, check.Equals, "127.0.0.1")

	_, err = s.localdb.ContainerUpdate(rootctx, arvados.UpdateOptions{
		UUID: s.ctrUUID,
		Attrs: map[string]interface{}{
			"state": arvados.ContainerStateRunning,
		}})
	c.Assert(err, check.IsNil)

	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(time.Second / 2) {
		ctr, err := s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.ctrUUID})
		c.Assert(err, check.IsNil)
		c.Check(ctr.InteractiveSessionStarted, check.Equals, false)
		c.Logf("ctr.GatewayAddress == %s", ctr.GatewayAddress)
		if strings.HasPrefix(ctr.GatewayAddress, "tunnel ") {
			break
		}
	}
	return tungw
}

func (s *ContainerGatewaySuite) testConnectThroughTunnel(c *check.C, expectErrorMatch string) {
	s.setupGatewayWithTunnel(c)
	c.Log("connecting to gateway through tunnel")
	arpc := rpc.NewConn("", &url.URL{Scheme: "https", Host: s.gw.ArvadosClient.APIHost}, true, rpc.PassthroughTokenProvider)
	sshconn, err := arpc.ContainerSSH(s.userctx, arvados.ContainerSSHOptions{UUID: s.ctrUUID})
	if expectErrorMatch != "" {
		c.Check(err, check.ErrorMatches, expectErrorMatch)
		return
	}
	c.Assert(err, check.IsNil)
	c.Assert(sshconn.Conn, check.NotNil)
	defer sshconn.Conn.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Receive text banner
		buf := make([]byte, 12)
		_, err := io.ReadFull(sshconn.Conn, buf)
		c.Check(err, check.IsNil)
		c.Check(string(buf), check.Equals, "SSH-2.0-Go\r\n")

		// Send text banner
		_, err = sshconn.Conn.Write([]byte("SSH-2.0-Fake\r\n"))
		c.Check(err, check.IsNil)

		// Receive binary
		_, err = io.ReadFull(sshconn.Conn, buf[:4])
		c.Check(err, check.IsNil)

		// If we can get this far into an SSH handshake...
		c.Logf("was able to read %x -- success, tunnel is working", buf[:4])
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		c.Fail()
	}
	ctr, err := s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.ctrUUID})
	c.Check(err, check.IsNil)
	c.Check(ctr.InteractiveSessionStarted, check.Equals, true)
}
