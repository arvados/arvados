// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package sshexecutor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"testing"
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"golang.org/x/crypto/ssh"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&ExecutorSuite{})

type testTarget struct {
	test.SSHService
}

func (*testTarget) VerifyHostKey(ssh.PublicKey, *ssh.Client) error {
	return nil
}

// Address returns the wrapped SSHService's host, with the port
// stripped. This ensures the executor won't work until
// SetTargetPort() is called -- see (*testTarget)Port().
func (tt *testTarget) Address() string {
	h, _, err := net.SplitHostPort(tt.SSHService.Address())
	if err != nil {
		panic(err)
	}
	return h
}

func (tt *testTarget) Port() string {
	_, p, err := net.SplitHostPort(tt.SSHService.Address())
	if err != nil {
		panic(err)
	}
	return p
}

type mitmTarget struct {
	test.SSHService
}

func (*mitmTarget) VerifyHostKey(key ssh.PublicKey, client *ssh.Client) error {
	return fmt.Errorf("host key failed verification: %#v", key)
}

type ExecutorSuite struct{}

func (s *ExecutorSuite) TestBadHostKey(c *check.C) {
	_, hostpriv := test.LoadTestKey(c, "../test/sshkey_vm")
	clientpub, clientpriv := test.LoadTestKey(c, "../test/sshkey_dispatch")
	target := &mitmTarget{
		SSHService: test.SSHService{
			Exec: func(map[string]string, string, io.Reader, io.Writer, io.Writer) uint32 {
				c.Error("Target Exec func called even though host key verification failed")
				return 0
			},
			HostKey:        hostpriv,
			AuthorizedUser: "username",
			AuthorizedKeys: []ssh.PublicKey{clientpub},
		},
	}

	err := target.Start()
	c.Check(err, check.IsNil)
	c.Logf("target address %q", target.Address())
	defer target.Close()

	exr := New(target)
	exr.SetSigners(clientpriv)

	_, _, err = exr.Execute(nil, "true", nil)
	c.Check(err, check.ErrorMatches, "host key failed verification: .*")
}

func (s *ExecutorSuite) TestExecute(c *check.C) {
	command := `foo 'bar' "baz"`
	stdinData := "foobar\nbaz\n"
	_, hostpriv := test.LoadTestKey(c, "../test/sshkey_vm")
	clientpub, clientpriv := test.LoadTestKey(c, "../test/sshkey_dispatch")
	for _, exitcode := range []int{0, 1, 2} {
		target := &testTarget{
			SSHService: test.SSHService{
				Exec: func(env map[string]string, cmd string, stdin io.Reader, stdout, stderr io.Writer) uint32 {
					c.Check(env["TESTVAR"], check.Equals, "test value")
					c.Check(cmd, check.Equals, command)
					var wg sync.WaitGroup
					wg.Add(2)
					go func() {
						io.WriteString(stdout, "stdout\n")
						wg.Done()
					}()
					go func() {
						io.WriteString(stderr, "stderr\n")
						wg.Done()
					}()
					buf, err := ioutil.ReadAll(stdin)
					wg.Wait()
					c.Check(err, check.IsNil)
					if err != nil {
						return 99
					}
					_, err = stdout.Write(buf)
					c.Check(err, check.IsNil)
					return uint32(exitcode)
				},
				HostKey:        hostpriv,
				AuthorizedUser: "username",
				AuthorizedKeys: []ssh.PublicKey{clientpub},
			},
		}
		err := target.Start()
		c.Check(err, check.IsNil)
		c.Logf("target address %q", target.Address())
		defer target.Close()

		exr := New(target)
		exr.SetSigners(clientpriv)

		// Use the default target port (ssh). Execute will
		// return a connection error or an authentication
		// error, depending on whether the test host is
		// running an SSH server.
		_, _, err = exr.Execute(nil, command, nil)
		c.Check(err, check.ErrorMatches, `.*(unable to authenticate|connection refused).*`)

		// Use a bogus target port. Execute will return a
		// connection error.
		exr.SetTargetPort("0")
		_, _, err = exr.Execute(nil, command, nil)
		c.Check(err, check.ErrorMatches, `.*connection refused.*`)
		c.Check(errors.As(err, new(*net.OpError)), check.Equals, true)

		// Use the test server's listening port.
		exr.SetTargetPort(target.Port())

		done := make(chan bool)
		go func() {
			stdout, stderr, err := exr.Execute(map[string]string{"TESTVAR": "test value"}, command, bytes.NewBufferString(stdinData))
			if exitcode == 0 {
				c.Check(err, check.IsNil)
			} else {
				c.Check(err, check.NotNil)
				err, ok := err.(*ssh.ExitError)
				c.Assert(ok, check.Equals, true)
				c.Check(err.ExitStatus(), check.Equals, exitcode)
			}
			c.Check(stdout, check.DeepEquals, []byte("stdout\n"+stdinData))
			c.Check(stderr, check.DeepEquals, []byte("stderr\n"))
			close(done)
		}()

		timeout := time.NewTimer(time.Second)
		select {
		case <-done:
		case <-timeout.C:
			c.Fatal("timed out")
		}
	}
}
