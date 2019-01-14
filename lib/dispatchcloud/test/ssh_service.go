// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

func LoadTestKey(c *check.C, fnm string) (ssh.PublicKey, ssh.Signer) {
	rawpubkey, err := ioutil.ReadFile(fnm + ".pub")
	c.Assert(err, check.IsNil)
	pubkey, _, _, _, err := ssh.ParseAuthorizedKey(rawpubkey)
	c.Assert(err, check.IsNil)
	rawprivkey, err := ioutil.ReadFile(fnm)
	c.Assert(err, check.IsNil)
	privkey, err := ssh.ParsePrivateKey(rawprivkey)
	c.Assert(err, check.IsNil)
	return pubkey, privkey
}

// An SSHExecFunc handles an "exec" session on a multiplexed SSH
// connection.
type SSHExecFunc func(command string, stdin io.Reader, stdout, stderr io.Writer) uint32

// An SSHService accepts SSH connections on an available TCP port and
// passes clients' "exec" sessions to the provided SSHExecFunc.
type SSHService struct {
	Exec           SSHExecFunc
	HostKey        ssh.Signer
	AuthorizedKeys []ssh.PublicKey

	listener net.Listener
	conn     *ssh.ServerConn
	setup    sync.Once
	mtx      sync.Mutex
	started  chan bool
	closed   bool
	err      error
}

// Address returns the host:port where the SSH server is listening. It
// returns "" if called before the server is ready to accept
// connections.
func (ss *SSHService) Address() string {
	ss.setup.Do(ss.start)
	ss.mtx.Lock()
	ln := ss.listener
	ss.mtx.Unlock()
	if ln == nil {
		return ""
	}
	return ln.Addr().String()
}

// Close shuts down the server and releases resources. Established
// connections are unaffected.
func (ss *SSHService) Close() {
	ss.Start()
	ss.mtx.Lock()
	ln := ss.listener
	ss.closed = true
	ss.mtx.Unlock()
	if ln != nil {
		ln.Close()
	}
}

// Start returns when the server is ready to accept connections.
func (ss *SSHService) Start() error {
	ss.setup.Do(ss.start)
	<-ss.started
	return ss.err
}

func (ss *SSHService) start() {
	ss.started = make(chan bool)
	go ss.run()
}

func (ss *SSHService) run() {
	defer close(ss.started)
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			for _, ak := range ss.AuthorizedKeys {
				if bytes.Equal(ak.Marshal(), pubKey.Marshal()) {
					return &ssh.Permissions{}, nil
				}
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}
	config.AddHostKey(ss.HostKey)

	listener, err := net.Listen("tcp", ":")
	if err != nil {
		ss.err = err
		return
	}

	ss.mtx.Lock()
	ss.listener = listener
	ss.mtx.Unlock()

	go func() {
		for {
			nConn, err := listener.Accept()
			if err != nil && strings.Contains(err.Error(), "use of closed network connection") && ss.closed {
				return
			} else if err != nil {
				log.Printf("accept: %s", err)
				return
			}
			go ss.serveConn(nConn, config)
		}
	}()
}

func (ss *SSHService) serveConn(nConn net.Conn, config *ssh.ServerConfig) {
	defer nConn.Close()
	conn, newchans, reqs, err := ssh.NewServerConn(nConn, config)
	if err != nil {
		log.Printf("ssh.NewServerConn: %s", err)
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)
	for newch := range newchans {
		if newch.ChannelType() != "session" {
			newch.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		ch, reqs, err := newch.Accept()
		if err != nil {
			log.Printf("accept channel: %s", err)
			return
		}
		var execReq struct {
			Command string
		}
		go func() {
			for req := range reqs {
				if req.Type == "exec" && execReq.Command == "" {
					req.Reply(true, nil)
					ssh.Unmarshal(req.Payload, &execReq)
					go func() {
						var resp struct {
							Status uint32
						}
						resp.Status = ss.Exec(execReq.Command, ch, ch, ch.Stderr())
						ch.SendRequest("exit-status", false, ssh.Marshal(&resp))
						ch.Close()
					}()
				}
			}
		}()
	}
}
