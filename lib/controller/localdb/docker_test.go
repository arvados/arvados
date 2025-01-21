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

type tcpProxy struct {
	net.Listener
}

// newTCPProxy sets up a TCP proxy that forwards all connections to the
// given host and port. This allows the caller to run a docker container that
// can connect to cluster service on the test host's loopback interface.
//
// listenAddr is the IP address of the interface to listen on. Pass an empty
// string to listen on all interfaces.
//
// Caller is responsible for calling Close() on the returned tcpProxy.
func newTCPProxy(c *check.C, listenAddr, host, port string) *tcpProxy {
	target := net.JoinHostPort(host, port)
	ln, err := net.Listen("tcp", net.JoinHostPort(listenAddr, ""))
	c.Assert(err, check.IsNil)
	go func() {
		for {
			downstream, err := ln.Accept()
			if err != nil && strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			c.Assert(err, check.IsNil)
			go func() {
				c.Logf("tcpProxy accepted connection from %s", downstream.RemoteAddr().String())
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
	c.Logf("tcpProxy listening at %s", ln.Addr().String())
	return &tcpProxy{Listener: ln}
}

func (proxy *tcpProxy) Port() string {
	_, port, _ := net.SplitHostPort(proxy.Addr().String())
	return port
}

// newPgProxy sets up a tcpProxy for the cluster's PostgreSQL database.
func newPgProxy(c *check.C, cluster *arvados.Cluster, listenAddr string) *tcpProxy {
	host := cluster.PostgreSQL.Connection["host"]
	if host == "" {
		host = "localhost"
	}
	port := cluster.PostgreSQL.Connection["port"]
	if port == "" {
		port = "5432"
	}
	return newTCPProxy(c, listenAddr, host, port)
}

// newInternalProxy sets up a tcpProxy for an InternalURL of the given service.
func newInternalProxy(c *check.C, service arvados.Service, listenAddr string) *tcpProxy {
	for intURL, _ := range service.InternalURLs {
		host, port, err := net.SplitHostPort(intURL.Host)
		if err == nil && port != "" {
			return newTCPProxy(c, listenAddr, host, port)
		}
	}
	c.Fatal("no valid InternalURLs found for service")
	return nil
}
