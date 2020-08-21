// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"io"
	"net"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

type pgproxy struct {
	net.Listener
}

// newPgProxy sets up a TCP proxy, listening on all interfaces, that
// forwards all connections to the cluster's PostgreSQL server. This
// allows the caller to run a docker container that can connect to a
// postgresql instance that listens on the test host's loopback
// interface.
//
// Caller is responsible for calling Close() on the returned pgproxy.
func newPgProxy(c *check.C, cluster *arvados.Cluster) *pgproxy {
	host := cluster.PostgreSQL.Connection["host"]
	if host == "" {
		host = "localhost"
	}
	port := cluster.PostgreSQL.Connection["port"]
	if port == "" {
		port = "5432"
	}
	target := net.JoinHostPort(host, port)

	ln, err := net.Listen("tcp", ":")
	c.Assert(err, check.IsNil)
	go func() {
		for {
			downstream, err := ln.Accept()
			if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			c.Assert(err, check.IsNil)
			go func() {
				c.Logf("pgproxy accepted connection from %s", downstream.RemoteAddr().String())
				defer downstream.Close()
				upstream, err := net.Dial("tcp", target)
				if err != nil {
					c.Logf("net.Dial(%q): %s", target, err)
					return
				}
				defer upstream.Close()
				go io.Copy(downstream, upstream)
				io.Copy(upstream, downstream)
			}()
		}
	}()
	c.Logf("pgproxy listening at %s", ln.Addr().String())
	return &pgproxy{Listener: ln}
}

func (proxy *pgproxy) Port() string {
	_, port, _ := net.SplitHostPort(proxy.Addr().String())
	return port
}
