// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ssh_executor

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"sync"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/test"
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

type ExecutorSuite struct{}

func (s *ExecutorSuite) TestExecute(c *check.C) {
	command := `foo 'bar' "baz"`
	stdinData := "foobar\nbaz\n"
	_, hostpriv := test.LoadTestKey(c, "../test/sshkey_vm")
	clientpub, clientpriv := test.LoadTestKey(c, "../test/sshkey_dispatch")
	for _, exitcode := range []int{0, 1, 2} {
		srv := &testTarget{
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
				AuthorizedKeys: []ssh.PublicKey{clientpub},
			},
		}
		err := srv.Start()
		c.Check(err, check.IsNil)
		c.Logf("srv address %q", srv.Address())
		defer srv.Close()

		exr := New(srv)
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

		// Use the test server's listening port.
		exr.SetTargetPort(srv.Port())

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
