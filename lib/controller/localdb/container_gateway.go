// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

// ContainerSSH returns a connection to the SSH server in the
// appropriate crunch-run process on the worker node where the
// specified container is running.
//
// If the returned error is nil, the caller is responsible for closing
// sshconn.Conn.
func (conn *Conn) ContainerSSH(ctx context.Context, opts arvados.ContainerSSHOptions) (sshconn arvados.ContainerSSHConnection, err error) {
	ctr, err := conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{UUID: opts.UUID})
	if err != nil {
		return
	}
	if ctr.GatewayAddress == "" || ctr.State != arvados.ContainerStateRunning {
		err = httpserver.ErrorWithStatus(fmt.Errorf("gateway is not available, container is %s", strings.ToLower(string(ctr.State))), http.StatusBadGateway)
		return
	}
	netconn, err := net.Dial("tcp", ctr.GatewayAddress)
	if err != nil {
		return
	}
	bufr := bufio.NewReader(netconn)
	bufw := bufio.NewWriter(netconn)

	// Note this auth header does not protect from replay/mitm
	// attacks (TODO: use TLS for that). It only authenticates us
	// to crunch-run.
	h := hmac.New(sha256.New, []byte(conn.cluster.SystemRootToken))
	fmt.Fprint(h, "%s", opts.UUID)
	auth := fmt.Sprintf("%x", h.Sum(nil))

	u := url.URL{
		Scheme: "http",
		Host:   ctr.GatewayAddress,
		Path:   "/ssh",
	}
	bufw.WriteString("GET " + u.String() + " HTTP/1.1\r\n")
	bufw.WriteString("Host: " + u.Host + "\r\n")
	bufw.WriteString("Upgrade: ssh\r\n")
	bufw.WriteString("X-Arvados-Target-Uuid: " + opts.UUID + "\r\n")
	bufw.WriteString("X-Arvados-Authorization: " + auth + "\r\n")
	bufw.WriteString("X-Arvados-Detach-Keys: " + opts.DetachKeys + "\r\n")
	bufw.WriteString("\r\n")
	bufw.Flush()
	resp, err := http.ReadResponse(bufr, &http.Request{Method: "GET"})
	if err != nil {
		err = fmt.Errorf("error reading http response from gateway: %w", err)
		netconn.Close()
		return
	}
	if strings.ToLower(resp.Header.Get("Upgrade")) != "ssh" ||
		strings.ToLower(resp.Header.Get("Connection")) != "upgrade" {
		err = httpserver.ErrorWithStatus(errors.New("bad upgrade"), http.StatusBadGateway)
		netconn.Close()
		return
	}
	sshconn.Conn = netconn
	sshconn.Bufrw = &bufio.ReadWriter{Reader: bufr, Writer: bufw}
	return
}
