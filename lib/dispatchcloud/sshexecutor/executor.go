// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Package sshexecutor provides an implementation of pool.Executor
// using a long-lived multiplexed SSH session.
package sshexecutor

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"golang.org/x/crypto/ssh"
)

var ErrNoAddress = errors.New("instance has no address")

// New returns a new Executor, using the given target.
func New(t cloud.ExecutorTarget) *Executor {
	return &Executor{target: t}
}

// An Executor uses a multiplexed SSH connection to execute shell
// commands on a remote target. It reconnects automatically after
// errors.
//
// When setting up a connection, the Executor accepts whatever host
// key is provided by the remote server, then passes the received key
// and the SSH connection to the target's VerifyHostKey method before
// executing commands on the connection.
//
// A zero Executor must not be used before calling SetTarget.
//
// An Executor must not be copied.
type Executor struct {
	target     cloud.ExecutorTarget
	targetPort string
	targetUser string
	signers    []ssh.Signer
	mtx        sync.RWMutex // controls access to instance after creation

	client      *ssh.Client
	clientErr   error
	clientOnce  sync.Once     // initialized private state
	clientSetup chan bool     // len>0 while client setup is in progress
	hostKey     ssh.PublicKey // most recent host key that passed verification, if any
}

// SetSigners updates the set of private keys that will be offered to
// the target next time the Executor sets up a new connection.
func (exr *Executor) SetSigners(signers ...ssh.Signer) {
	exr.mtx.Lock()
	defer exr.mtx.Unlock()
	exr.signers = signers
}

// SetTarget sets the current target. The new target will be used next
// time a new connection is set up; until then, the Executor will
// continue to use the existing target.
//
// The new target is assumed to represent the same host as the
// previous target, although its address and host key might differ.
func (exr *Executor) SetTarget(t cloud.ExecutorTarget) {
	exr.mtx.Lock()
	defer exr.mtx.Unlock()
	exr.target = t
}

// SetTargetPort sets the default port (name or number) to connect
// to. This is used only when the address returned by the target's
// Address() method does not specify a port. If the given port is
// empty (or SetTargetPort is not called at all), the default port is
// "ssh".
func (exr *Executor) SetTargetPort(port string) {
	exr.mtx.Lock()
	defer exr.mtx.Unlock()
	exr.targetPort = port
}

// Target returns the current target.
func (exr *Executor) Target() cloud.ExecutorTarget {
	exr.mtx.RLock()
	defer exr.mtx.RUnlock()
	return exr.target
}

// Execute runs cmd on the target. If an existing connection is not
// usable, it sets up a new connection to the current target.
func (exr *Executor) Execute(env map[string]string, cmd string, stdin io.Reader) ([]byte, []byte, error) {
	session, err := exr.newSession()
	if err != nil {
		return nil, nil, err
	}
	defer session.Close()
	for k, v := range env {
		err = session.Setenv(k, v)
		if err != nil {
			return nil, nil, err
		}
	}
	var stdout, stderr bytes.Buffer
	session.Stdin = stdin
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run(cmd)
	return stdout.Bytes(), stderr.Bytes(), err
}

// Close shuts down any active connections.
func (exr *Executor) Close() {
	// Ensure exr is initialized
	exr.sshClient(false)

	exr.clientSetup <- true
	if exr.client != nil {
		defer exr.client.Close()
	}
	exr.client, exr.clientErr = nil, errors.New("closed")
	<-exr.clientSetup
}

// Create a new SSH session. If session setup fails or the SSH client
// hasn't been setup yet, setup a new SSH client and try again.
func (exr *Executor) newSession() (*ssh.Session, error) {
	try := func(create bool) (*ssh.Session, error) {
		client, err := exr.sshClient(create)
		if err != nil {
			return nil, err
		}
		return client.NewSession()
	}
	session, err := try(false)
	if err != nil {
		session, err = try(true)
	}
	return session, err
}

// Get the latest SSH client. If another goroutine is in the process
// of setting one up, wait for it to finish and return its result (or
// the last successfully setup client, if it fails).
func (exr *Executor) sshClient(create bool) (*ssh.Client, error) {
	exr.clientOnce.Do(func() {
		exr.clientSetup = make(chan bool, 1)
		exr.clientErr = errors.New("client not yet created")
	})
	defer func() { <-exr.clientSetup }()
	select {
	case exr.clientSetup <- true:
		if create {
			client, err := exr.setupSSHClient()
			if err == nil || exr.client == nil {
				if exr.client != nil {
					// Hang up the previous
					// (non-working) client
					go exr.client.Close()
				}
				exr.client, exr.clientErr = client, err
			}
			if err != nil {
				return nil, err
			}
		}
	default:
		// Another goroutine is doing the above case.  Wait
		// for it to finish and return whatever it leaves in
		// wkr.client.
		exr.clientSetup <- true
	}
	return exr.client, exr.clientErr
}

func (exr *Executor) TargetHostPort() (string, string) {
	addr := exr.Target().Address()
	if addr == "" {
		return "", ""
	}
	h, p, err := net.SplitHostPort(addr)
	if err != nil || p == "" {
		// Target address does not specify a port.  Use
		// targetPort, or "ssh".
		if h == "" {
			h = addr
		}
		if p = exr.targetPort; p == "" {
			p = "ssh"
		}
	}
	return h, p
}

// Create a new SSH client.
func (exr *Executor) setupSSHClient() (*ssh.Client, error) {
	addr := net.JoinHostPort(exr.TargetHostPort())
	if addr == ":" {
		return nil, ErrNoAddress
	}
	var receivedKey ssh.PublicKey
	client, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
		User: exr.Target().RemoteUser(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(exr.signers...),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			receivedKey = key
			return nil
		},
		Timeout: time.Minute,
	})
	if err != nil {
		return nil, err
	} else if receivedKey == nil {
		return nil, errors.New("BUG: key was never provided to HostKeyCallback")
	}

	if exr.hostKey == nil || !bytes.Equal(exr.hostKey.Marshal(), receivedKey.Marshal()) {
		err = exr.Target().VerifyHostKey(receivedKey, client)
		if err != nil {
			return nil, err
		}
		exr.hostKey = receivedKey
	}
	return client, nil
}
