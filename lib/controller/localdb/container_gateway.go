// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/hashicorp/yamux"
)

var (
	forceProxyForTest       = false
	forceInternalURLForTest *arvados.URL
)

// ContainerSSH returns a connection to the SSH server in the
// appropriate crunch-run process on the worker node where the
// specified container is running.
//
// If the returned error is nil, the caller is responsible for closing
// sshconn.Conn.
func (conn *Conn) ContainerSSH(ctx context.Context, opts arvados.ContainerSSHOptions) (sshconn arvados.ConnectionResponse, err error) {
	user, err := conn.railsProxy.UserGetCurrent(ctx, arvados.GetOptions{})
	if err != nil {
		return sshconn, err
	}
	ctr, err := conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{UUID: opts.UUID})
	if err != nil {
		return sshconn, err
	}
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{conn.cluster.SystemRootToken}})
	if !user.IsAdmin || !conn.cluster.Containers.ShellAccess.Admin {
		if !conn.cluster.Containers.ShellAccess.User {
			return sshconn, httpserver.ErrorWithStatus(errors.New("shell access is disabled in config"), http.StatusServiceUnavailable)
		}
		crs, err := conn.railsProxy.ContainerRequestList(ctxRoot, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"container_uuid", "=", opts.UUID}}})
		if err != nil {
			return sshconn, err
		}
		for _, cr := range crs.Items {
			if cr.ModifiedByUserUUID != user.UUID {
				return sshconn, httpserver.ErrorWithStatus(errors.New("permission denied: container is associated with requests submitted by other users"), http.StatusForbidden)
			}
		}
		if crs.ItemsAvailable != len(crs.Items) {
			return sshconn, httpserver.ErrorWithStatus(errors.New("incomplete response while checking permission"), http.StatusInternalServerError)
		}
	}

	conn.gwTunnelsLock.Lock()
	tunnel := conn.gwTunnels[opts.UUID]
	conn.gwTunnelsLock.Unlock()

	if ctr.State == arvados.ContainerStateQueued || ctr.State == arvados.ContainerStateLocked {
		return sshconn, httpserver.ErrorWithStatus(fmt.Errorf("container is not running yet (state is %q)", ctr.State), http.StatusServiceUnavailable)
	} else if ctr.State != arvados.ContainerStateRunning {
		return sshconn, httpserver.ErrorWithStatus(fmt.Errorf("container has ended (state is %q)", ctr.State), http.StatusGone)
	}

	// targetHost is the value we'll use in the Host header in our
	// "Upgrade: ssh" http request. It's just a placeholder
	// "localhost", unless we decide to connect directly, in which
	// case we'll set it to the gateway's external ip:host. (The
	// gateway doesn't even look at it, but we might as well.)
	targetHost := "localhost"
	myURL, _ := service.URLFromContext(ctx)

	var rawconn net.Conn
	if host, _, splitErr := net.SplitHostPort(ctr.GatewayAddress); splitErr == nil && host != "" && host != "127.0.0.1" {
		// If crunch-run provided a GatewayAddress like
		// "ipaddr:port", that means "ipaddr" is one of the
		// external interfaces where the gateway is
		// listening. In that case, it's the most
		// reliable/direct option, so we use it even if a
		// tunnel might also be available.
		targetHost = ctr.GatewayAddress
		rawconn, err = net.Dial("tcp", ctr.GatewayAddress)
		if err != nil {
			return sshconn, httpserver.ErrorWithStatus(err, http.StatusServiceUnavailable)
		}
	} else if tunnel != nil && !(forceProxyForTest && !opts.NoForward) {
		// If we can't connect directly, and the gateway has
		// established a yamux tunnel with us, connect through
		// the tunnel.
		//
		// ...except: forceProxyForTest means we are emulating
		// a situation where the gateway has established a
		// yamux tunnel with controller B, and the
		// ContainerSSH request arrives at controller A. If
		// opts.NoForward==false then we are acting as A, so
		// we pretend not to have a tunnel, and fall through
		// to the "tunurl" case below. If opts.NoForward==true
		// then the client is A and we are acting as B, so we
		// connect to our tunnel.
		rawconn, err = tunnel.Open()
		if err != nil {
			return sshconn, httpserver.ErrorWithStatus(err, http.StatusServiceUnavailable)
		}
	} else if ctr.GatewayAddress == "" {
		return sshconn, httpserver.ErrorWithStatus(errors.New("container is running but gateway is not available"), http.StatusServiceUnavailable)
	} else if tunurl := strings.TrimPrefix(ctr.GatewayAddress, "tunnel "); tunurl != ctr.GatewayAddress &&
		tunurl != "" &&
		tunurl != myURL.String() &&
		!opts.NoForward {
		// If crunch-run provided a GatewayAddress like
		// "tunnel https://10.0.0.10:1010/", that means the
		// gateway has established a yamux tunnel with the
		// controller process at the indicated InternalURL
		// (which isn't us, otherwise we would have had
		// "tunnel != nil" above). We need to proxy through to
		// the other controller process in order to use the
		// tunnel.
		for u := range conn.cluster.Services.Controller.InternalURLs {
			if u.String() == tunurl {
				ctxlog.FromContext(ctx).Debugf("proxying ContainerSSH request to other controller at %s", u)
				u := url.URL(u)
				arpc := rpc.NewConn(conn.cluster.ClusterID, &u, conn.cluster.TLS.Insecure, rpc.PassthroughTokenProvider)
				opts.NoForward = true
				return arpc.ContainerSSH(ctx, opts)
			}
		}
		ctxlog.FromContext(ctx).Warnf("container gateway provided a tunnel endpoint %s that is not one of Services.Controller.InternalURLs", tunurl)
		return sshconn, httpserver.ErrorWithStatus(errors.New("container gateway is running but tunnel endpoint is invalid"), http.StatusServiceUnavailable)
	} else {
		return sshconn, httpserver.ErrorWithStatus(errors.New("container gateway is running but tunnel is down"), http.StatusServiceUnavailable)
	}

	// crunch-run uses a self-signed / unverifiable TLS
	// certificate, so we use the following scheme to ensure we're
	// not talking to a MITM.
	//
	// 1. Compute ctrKey = HMAC-SHA256(sysRootToken,ctrUUID) --
	// this will be the same ctrKey that a-d-c supplied to
	// crunch-run in the GatewayAuthSecret env var.
	//
	// 2. Compute requestAuth = HMAC-SHA256(ctrKey,serverCert) and
	// send it to crunch-run as the X-Arvados-Authorization
	// header, proving that we know ctrKey. (Note a MITM cannot
	// replay the proof to a real crunch-run server, because the
	// real crunch-run server would have a different cert.)
	//
	// 3. Compute respondAuth = HMAC-SHA256(ctrKey,requestAuth)
	// and ensure the server returns it in the
	// X-Arvados-Authorization-Response header, proving that the
	// server knows ctrKey.
	var requestAuth, respondAuth string
	tlsconn := tls.Client(rawconn, &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("no certificate received, cannot compute authorization header")
			}
			h := hmac.New(sha256.New, []byte(conn.cluster.SystemRootToken))
			fmt.Fprint(h, opts.UUID)
			authKey := fmt.Sprintf("%x", h.Sum(nil))
			h = hmac.New(sha256.New, []byte(authKey))
			h.Write(rawCerts[0])
			requestAuth = fmt.Sprintf("%x", h.Sum(nil))
			h.Reset()
			h.Write([]byte(requestAuth))
			respondAuth = fmt.Sprintf("%x", h.Sum(nil))
			return nil
		},
	})
	err = tlsconn.HandshakeContext(ctx)
	if err != nil {
		return sshconn, httpserver.ErrorWithStatus(err, http.StatusBadGateway)
	}
	if respondAuth == "" {
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(errors.New("BUG: no respondAuth"), http.StatusInternalServerError)
	}
	bufr := bufio.NewReader(tlsconn)
	bufw := bufio.NewWriter(tlsconn)

	u := url.URL{
		Scheme: "http",
		Host:   targetHost,
		Path:   "/ssh",
	}
	postform := url.Values{
		"uuid":           {opts.UUID},
		"detach_keys":    {opts.DetachKeys},
		"login_username": {opts.LoginUsername},
		"no_forward":     {fmt.Sprintf("%v", opts.NoForward)},
	}
	postdata := postform.Encode()
	bufw.WriteString("POST " + u.String() + " HTTP/1.1\r\n")
	bufw.WriteString("Host: " + u.Host + "\r\n")
	bufw.WriteString("Upgrade: ssh\r\n")
	bufw.WriteString("X-Arvados-Authorization: " + requestAuth + "\r\n")
	bufw.WriteString("Content-Type: application/x-www-form-urlencoded\r\n")
	fmt.Fprintf(bufw, "Content-Length: %d\r\n", len(postdata))
	bufw.WriteString("\r\n")
	bufw.WriteString(postdata)
	bufw.Flush()
	resp, err := http.ReadResponse(bufr, &http.Request{Method: "POST"})
	if err != nil {
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(fmt.Errorf("error reading http response from gateway: %w", err), http.StatusBadGateway)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 1000))
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(fmt.Errorf("unexpected status %s %q", resp.Status, body), http.StatusBadGateway)
	}
	if strings.ToLower(resp.Header.Get("Upgrade")) != "ssh" ||
		strings.ToLower(resp.Header.Get("Connection")) != "upgrade" {
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(errors.New("bad upgrade"), http.StatusBadGateway)
	}
	if resp.Header.Get("X-Arvados-Authorization-Response") != respondAuth {
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(errors.New("bad X-Arvados-Authorization-Response header"), http.StatusBadGateway)
	}

	if !ctr.InteractiveSessionStarted {
		_, err = conn.railsProxy.ContainerUpdate(ctxRoot, arvados.UpdateOptions{
			UUID: opts.UUID,
			Attrs: map[string]interface{}{
				"interactive_session_started": true,
			},
		})
		if err != nil {
			tlsconn.Close()
			return sshconn, httpserver.ErrorWithStatus(err, http.StatusInternalServerError)
		}
	}

	sshconn.Conn = tlsconn
	sshconn.Bufrw = &bufio.ReadWriter{Reader: bufr, Writer: bufw}
	sshconn.Logger = ctxlog.FromContext(ctx)
	sshconn.Header = http.Header{"Upgrade": {"ssh"}}
	return sshconn, nil
}

// ContainerGatewayTunnel sets up a tunnel enabling us (controller) to
// connect to the caller's (crunch-run's) gateway server.
func (conn *Conn) ContainerGatewayTunnel(ctx context.Context, opts arvados.ContainerGatewayTunnelOptions) (resp arvados.ConnectionResponse, err error) {
	h := hmac.New(sha256.New, []byte(conn.cluster.SystemRootToken))
	fmt.Fprint(h, opts.UUID)
	authSecret := fmt.Sprintf("%x", h.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(authSecret), []byte(opts.AuthSecret)) != 1 {
		ctxlog.FromContext(ctx).Info("received incorrect auth_secret")
		return resp, httpserver.ErrorWithStatus(errors.New("authentication error"), http.StatusUnauthorized)
	}

	muxconn, clientconn := net.Pipe()
	tunnel, err := yamux.Server(muxconn, nil)
	if err != nil {
		clientconn.Close()
		return resp, httpserver.ErrorWithStatus(err, http.StatusInternalServerError)
	}

	conn.gwTunnelsLock.Lock()
	if conn.gwTunnels == nil {
		conn.gwTunnels = map[string]*yamux.Session{opts.UUID: tunnel}
	} else {
		conn.gwTunnels[opts.UUID] = tunnel
	}
	conn.gwTunnelsLock.Unlock()

	go func() {
		<-tunnel.CloseChan()
		conn.gwTunnelsLock.Lock()
		if conn.gwTunnels[opts.UUID] == tunnel {
			delete(conn.gwTunnels, opts.UUID)
		}
		conn.gwTunnelsLock.Unlock()
	}()

	// Assuming we're acting as the backend of an http server,
	// lib/controller/router will call resp's ServeHTTP handler,
	// which upgrades the incoming http connection to a raw socket
	// and connects it to our yamux.Server through our net.Pipe().
	resp.Conn = clientconn
	resp.Bufrw = &bufio.ReadWriter{Reader: bufio.NewReader(&bytes.Buffer{}), Writer: bufio.NewWriter(&bytes.Buffer{})}
	resp.Logger = ctxlog.FromContext(ctx)
	resp.Header = http.Header{"Upgrade": {"tunnel"}}
	if u, ok := service.URLFromContext(ctx); ok {
		resp.Header.Set("X-Arvados-Internal-Url", u.String())
	} else if forceInternalURLForTest != nil {
		resp.Header.Set("X-Arvados-Internal-Url", forceInternalURLForTest.String())
	}
	return
}
