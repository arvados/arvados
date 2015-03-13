package main

import (
	"net"
	"net/http"
	"net/http/cgi"
	"sync"
	"time"
)

type server struct {
	http.Server
	Addr     string // host:port where the server is listening.
	err      error
	cond     *sync.Cond
	done     bool
	listener *net.TCPListener
	wantDown bool
}

func (srv *server) Start() error {
	gitHandler := &cgi.Handler{
		Path: theConfig.GitCommand,
		Dir:  theConfig.Root,
		Env: []string{
			"GIT_PROJECT_ROOT=" + theConfig.Root,
			"GIT_HTTP_EXPORT_ALL=",
		},
		InheritEnv: []string{"PATH"},
		Args:       []string{"http-backend"},
	}

	// The rest of the work here is essentially
	// http.ListenAndServe() with two more features: (1) whoever
	// called Start() can discover which address:port we end up
	// listening to -- which makes listening on ":0" useful in
	// test suites -- and (2) the server can be shut down without
	// killing the process -- which is useful in test cases, and
	// makes it possible to shut down gracefully on SIGTERM
	// without killing active connections.

	addr, err := net.ResolveTCPAddr("tcp", theConfig.Addr)
	if err != nil {
		return err
	}
	srv.listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return err
	}
	srv.Addr = srv.listener.Addr().String()
	mux := http.NewServeMux()
	mux.Handle("/", &authHandler{gitHandler})
	srv.Handler = mux

	mutex := &sync.RWMutex{}
	srv.cond = sync.NewCond(mutex.RLocker())
	go func() {
		err = srv.Serve(tcpKeepAliveListener{srv.listener})
		if !srv.wantDown {
			srv.err = err
		}
		mutex.Lock()
		srv.done = true
		srv.cond.Broadcast()
		mutex.Unlock()
	}()
	return nil
}

// Wait returns when the server has shut down.
func (srv *server) Wait() error {
	srv.cond.L.Lock()
	defer srv.cond.L.Unlock()
	for !srv.done {
		srv.cond.Wait()
	}
	return srv.err
}

// Close shuts down the server and returns when it has stopped.
func (srv *server) Close() error {
	srv.wantDown = true
	srv.listener.Close()
	return srv.Wait()
}

// tcpKeepAliveListener is copied from net/http because not exported.
//
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
