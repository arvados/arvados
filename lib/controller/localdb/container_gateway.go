// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

// ContainerSSH returns a connection to the SSH server in the
// appropriate crunch-run process on the worker node where the
// specified container is running.
//
// If the returned error is nil, the caller is responsible for closing
// sshconn.Conn.
func (conn *Conn) ContainerSSH(ctx context.Context, opts arvados.ContainerSSHOptions) (sshconn arvados.ContainerSSHConnection, err error) {
	user, err := conn.railsProxy.UserGetCurrent(ctx, arvados.GetOptions{})
	if err != nil {
		return
	}
	ctr, err := conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{UUID: opts.UUID})
	if err != nil {
		return
	}
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{conn.cluster.SystemRootToken}})
	if !user.IsAdmin || !conn.cluster.Containers.ShellAccess.Admin {
		if !conn.cluster.Containers.ShellAccess.User {
			err = httpserver.ErrorWithStatus(errors.New("shell access is disabled in config"), http.StatusServiceUnavailable)
			return
		}
		var crs arvados.ContainerRequestList
		crs, err = conn.railsProxy.ContainerRequestList(ctxRoot, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"container_uuid", "=", opts.UUID}}})
		if err != nil {
			return
		}
		for _, cr := range crs.Items {
			if cr.ModifiedByUserUUID != user.UUID {
				err = httpserver.ErrorWithStatus(errors.New("permission denied: container is associated with requests submitted by other users"), http.StatusForbidden)
				return
			}
		}
		if crs.ItemsAvailable != len(crs.Items) {
			err = httpserver.ErrorWithStatus(errors.New("incomplete response while checking permission"), http.StatusInternalServerError)
			return
		}
	}

	switch ctr.State {
	case arvados.ContainerStateQueued, arvados.ContainerStateLocked:
		err = httpserver.ErrorWithStatus(fmt.Errorf("container is not running yet (state is %q)", ctr.State), http.StatusServiceUnavailable)
		return
	case arvados.ContainerStateRunning:
		if ctr.GatewayAddress == "" {
			err = httpserver.ErrorWithStatus(errors.New("container is running but gateway is not available -- installation problem or feature not supported"), http.StatusServiceUnavailable)
			return
		}
	default:
		err = httpserver.ErrorWithStatus(fmt.Errorf("container has ended (state is %q)", ctr.State), http.StatusGone)
		return
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
	netconn, err := tls.Dial("tcp", ctr.GatewayAddress, &tls.Config{
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
	if err != nil {
		err = httpserver.ErrorWithStatus(err, http.StatusBadGateway)
		return
	}
	if respondAuth == "" {
		err = httpserver.ErrorWithStatus(errors.New("BUG: no respondAuth"), http.StatusInternalServerError)
		return
	}
	bufr := bufio.NewReader(netconn)
	bufw := bufio.NewWriter(netconn)

	u := url.URL{
		Scheme: "http",
		Host:   ctr.GatewayAddress,
		Path:   "/ssh",
	}
	bufw.WriteString("GET " + u.String() + " HTTP/1.1\r\n")
	bufw.WriteString("Host: " + u.Host + "\r\n")
	bufw.WriteString("Upgrade: ssh\r\n")
	bufw.WriteString("X-Arvados-Target-Uuid: " + opts.UUID + "\r\n")
	bufw.WriteString("X-Arvados-Authorization: " + requestAuth + "\r\n")
	bufw.WriteString("X-Arvados-Detach-Keys: " + opts.DetachKeys + "\r\n")
	bufw.WriteString("X-Arvados-Login-Username: " + opts.LoginUsername + "\r\n")
	bufw.WriteString("\r\n")
	bufw.Flush()
	resp, err := http.ReadResponse(bufr, &http.Request{Method: "GET"})
	if err != nil {
		err = httpserver.ErrorWithStatus(fmt.Errorf("error reading http response from gateway: %w", err), http.StatusBadGateway)
		netconn.Close()
		return
	}
	if resp.Header.Get("X-Arvados-Authorization-Response") != respondAuth {
		err = httpserver.ErrorWithStatus(errors.New("bad X-Arvados-Authorization-Response header"), http.StatusBadGateway)
		netconn.Close()
		return
	}
	if strings.ToLower(resp.Header.Get("Upgrade")) != "ssh" ||
		strings.ToLower(resp.Header.Get("Connection")) != "upgrade" {
		err = httpserver.ErrorWithStatus(errors.New("bad upgrade"), http.StatusBadGateway)
		netconn.Close()
		return
	}

	if !ctr.InteractiveSessionStarted {
		_, err = conn.railsProxy.ContainerUpdate(ctxRoot, arvados.UpdateOptions{
			UUID: opts.UUID,
			Attrs: map[string]interface{}{
				"interactive_session_started": true,
			},
		})
		if err != nil {
			netconn.Close()
			return
		}
	}

	sshconn.Conn = netconn
	sshconn.Bufrw = &bufio.ReadWriter{Reader: bufr, Writer: bufw}
	sshconn.Logger = ctxlog.FromContext(ctx)
	return
}
