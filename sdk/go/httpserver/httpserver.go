// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Server struct {
	http.Server
	Addr     string // host:port where the server is listening.
	err      error
	cond     *sync.Cond
	running  bool
	listener *net.TCPListener
	wantDown bool
}

// Start is essentially (*http.Server)ListenAndServe() with two more
// features: (1) by the time Start() returns, Addr is changed to the
// address:port we ended up listening to -- which makes listening on
// ":0" useful in test suites -- and (2) the server can be shut down
// without killing the process -- which is useful in test cases, and
// makes it possible to shut down gracefully on SIGTERM without
// killing active connections.
func (srv *Server) Start() error {
	addr, err := net.ResolveTCPAddr("tcp", srv.Addr)
	if err != nil {
		return err
	}
	srv.listener, err = listenTCP("tcp", addr)
	if err != nil {
		return err
	}
	srv.Addr = srv.listener.Addr().String()

	mutex := &sync.RWMutex{}
	srv.cond = sync.NewCond(mutex.RLocker())
	srv.running = true
	go func() {
		lnr := tcpKeepAliveListener{srv.listener}
		if srv.TLSConfig != nil {
			err = srv.ServeTLS(lnr, "", "")
		} else {
			err = srv.Serve(lnr)
		}
		if !srv.wantDown {
			srv.err = err
		}
		mutex.Lock()
		srv.running = false
		srv.cond.Broadcast()
		mutex.Unlock()
	}()
	return nil
}

// Close shuts down the server and returns when it has stopped.
func (srv *Server) Close() error {
	srv.wantDown = true
	srv.listener.Close()
	return srv.Wait()
}

// Wait returns when the server has shut down.
func (srv *Server) Wait() error {
	if srv.cond == nil {
		return nil
	}
	srv.cond.L.Lock()
	defer srv.cond.L.Unlock()
	for srv.running {
		srv.cond.Wait()
	}
	return srv.err
}

// tcpKeepAliveListener is copied from net/http because not exported.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

// net.ListenTCP, but retry after "address already in use" for up to 5
// minutes if running inside the arvados test suite.
func listenTCP(network string, addr *net.TCPAddr) (*net.TCPListener, error) {
	if os.Getenv("ARVADOS_TEST_API_HOST") == "" {
		return net.ListenTCP("tcp", addr)
	}
	timeout := 5 * time.Minute
	deadline := time.Now().Add(timeout)
	logged := false
	for {
		ln, err := net.ListenTCP("tcp", addr)
		if err != nil && strings.Contains(err.Error(), "address already in use") && time.Now().Before(deadline) {
			if !logged {
				log.Printf("listenTCP: retrying up to %v after error: %s", timeout, err)
				logged = true
			}
			time.Sleep(time.Second)
			continue
		}
		return ln, err
	}
}
