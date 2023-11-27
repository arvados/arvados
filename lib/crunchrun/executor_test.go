// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/diagnostics"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
)

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

// embedded by dockerSuite and singularitySuite so they can share
// tests.
type executorSuite struct {
	newExecutor func(*C) // embedding struct's SetUpSuite method must set this
	executor    containerExecutor
	spec        containerSpec
	stdout      bytes.Buffer
	stderr      bytes.Buffer
}

func (s *executorSuite) SetUpTest(c *C) {
	s.newExecutor(c)
	s.stdout = bytes.Buffer{}
	s.stderr = bytes.Buffer{}
	s.spec = containerSpec{
		Image:       "busybox:uclibc",
		VCPUs:       1,
		WorkingDir:  "",
		Env:         map[string]string{"PATH": "/bin:/usr/bin"},
		NetworkMode: "default",
		Stdout:      nopWriteCloser{&s.stdout},
		Stderr:      nopWriteCloser{&s.stderr},
	}
	err := s.executor.LoadImage("", arvadostest.BusyboxDockerImage(c), arvados.Container{}, "", nil)
	c.Assert(err, IsNil)
}

func (s *executorSuite) TearDownTest(c *C) {
	s.executor.Close()
}

func (s *executorSuite) TestExecTrivialContainer(c *C) {
	c.Logf("Using container runtime: %s", s.executor.Runtime())
	s.spec.Command = []string{"echo", "ok"}
	s.checkRun(c, 0)
	c.Check(s.stdout.String(), Equals, "ok\n")
	c.Check(s.stderr.String(), Equals, "")
}

func (s *executorSuite) TestDiagnosticsImage(c *C) {
	s.newExecutor(c)
	imagefile := c.MkDir() + "/hello-world.tar"
	err := ioutil.WriteFile(imagefile, diagnostics.HelloWorldDockerImage, 0777)
	c.Assert(err, IsNil)
	err = s.executor.LoadImage("", imagefile, arvados.Container{}, "", nil)
	c.Assert(err, IsNil)

	c.Logf("Using container runtime: %s", s.executor.Runtime())
	s.spec.Image = "hello-world"
	s.spec.Command = []string{"/hello"}
	s.checkRun(c, 0)
	c.Check(s.stdout.String(), Matches, `(?ms)\nHello from Docker!\n.*`)
}

func (s *executorSuite) TestExitStatus(c *C) {
	s.spec.Command = []string{"false"}
	s.checkRun(c, 1)
}

func (s *executorSuite) TestSignalExitStatus(c *C) {
	if _, isdocker := s.executor.(*dockerExecutor); isdocker {
		// It's not quite this easy to make busybox kill
		// itself in docker where it's pid 1.
		c.Skip("kill -9 $$ doesn't work on busybox with pid=1 in docker")
		return
	}
	s.spec.Command = []string{"sh", "-c", "kill -9 $$"}
	s.checkRun(c, 0x80+9)
}

func (s *executorSuite) TestExecStop(c *C) {
	s.spec.Command = []string{"sh", "-c", "sleep 10; echo ok"}
	err := s.executor.Create(s.spec)
	c.Assert(err, IsNil)
	err = s.executor.Start()
	c.Assert(err, IsNil)
	go func() {
		time.Sleep(time.Second / 10)
		s.executor.Stop()
	}()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	code, err := s.executor.Wait(ctx)
	c.Check(code, Not(Equals), 0)
	c.Check(err, IsNil)
	c.Check(s.stdout.String(), Equals, "")
	c.Check(s.stderr.String(), Equals, "")
}

func (s *executorSuite) TestExecCleanEnv(c *C) {
	s.spec.Command = []string{"env"}
	s.checkRun(c, 0)
	c.Check(s.stderr.String(), Equals, "")
	got := map[string]string{}
	for _, kv := range strings.Split(s.stdout.String(), "\n") {
		if kv == "" {
			continue
		}
		kv := strings.SplitN(kv, "=", 2)
		switch kv[0] {
		case "HOSTNAME", "HOME":
			// docker sets these by itself
		case "LD_LIBRARY_PATH", "SINGULARITY_NAME", "PWD", "LANG", "SHLVL", "SINGULARITY_INIT", "SINGULARITY_CONTAINER":
			// singularity sets these by itself (cf. https://sylabs.io/guides/3.5/user-guide/environment_and_metadata.html)
		case "SINGULARITY_APPNAME":
			// singularity also sets this by itself (v3.5.2, but not v3.7.4)
		case "PROMPT_COMMAND", "PS1", "SINGULARITY_BIND", "SINGULARITY_COMMAND", "SINGULARITY_ENVIRONMENT":
			// singularity also sets these by itself (v3.7.4)
		case "SINGULARITY_NO_EVAL":
			// our singularity driver sets this to control
			// singularity behavior, and it gets passed
			// through to the container
		default:
			got[kv[0]] = kv[1]
		}
	}
	c.Check(got, DeepEquals, s.spec.Env)
}
func (s *executorSuite) TestExecEnableNetwork(c *C) {
	for _, enable := range []bool{false, true} {
		s.SetUpTest(c)
		s.spec.Command = []string{"ip", "route"}
		s.spec.EnableNetwork = enable
		s.checkRun(c, 0)
		if enable {
			c.Check(s.stdout.String(), Matches, "(?ms).*default via.*")
		} else {
			c.Check(s.stdout.String(), Equals, "")
		}
	}
}

func (s *executorSuite) TestExecWorkingDir(c *C) {
	s.spec.WorkingDir = "/tmp"
	s.spec.Command = []string{"sh", "-c", "pwd"}
	s.checkRun(c, 0)
	c.Check(s.stdout.String(), Equals, "/tmp\n")
}

func (s *executorSuite) TestExecStdoutStderr(c *C) {
	s.spec.Command = []string{"sh", "-c", "echo foo; echo -n bar >&2; echo baz; echo waz >&2"}
	s.checkRun(c, 0)
	c.Check(s.stdout.String(), Equals, "foo\nbaz\n")
	c.Check(s.stderr.String(), Equals, "barwaz\n")
}

func (s *executorSuite) TestIPAddress(c *C) {
	// Listen on an available port on the host.
	ln, err := net.Listen("tcp", net.JoinHostPort("0.0.0.0", "0"))
	c.Assert(err, IsNil)
	defer ln.Close()
	_, port, err := net.SplitHostPort(ln.Addr().String())
	c.Assert(err, IsNil)

	// Start a container that listens on the same port number that
	// is already in use on the host.
	s.spec.Command = []string{"nc", "-l", "-p", port, "-e", "printf", `HTTP/1.1 418 I'm a teapot\r\n\r\n`}
	s.spec.EnableNetwork = true
	c.Assert(s.executor.Create(s.spec), IsNil)
	c.Assert(s.executor.Start(), IsNil)
	starttime := time.Now()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	for ctx.Err() == nil {
		time.Sleep(time.Second / 10)
		_, err := s.executor.IPAddress()
		if err == nil {
			break
		}
	}
	// When we connect to the port using s.executor.IPAddress(),
	// we should reach the nc process running inside the
	// container, not the net.Listen() running outside the
	// container, even though both listen on the same port.
	ip, err := s.executor.IPAddress()
	if c.Check(err, IsNil) && c.Check(ip, Not(Equals), "") {
		req, err := http.NewRequest("BREW", "http://"+net.JoinHostPort(ip, port), nil)
		c.Assert(err, IsNil)
		resp, err := http.DefaultClient.Do(req)
		c.Assert(err, IsNil)
		c.Check(resp.StatusCode, Equals, http.StatusTeapot)
	}

	s.executor.Stop()
	code, _ := s.executor.Wait(ctx)
	c.Logf("container ran for %v", time.Now().Sub(starttime))
	c.Check(code, Equals, -1)

	c.Logf("stdout:\n%s\n\n", s.stdout.String())
	c.Logf("stderr:\n%s\n\n", s.stderr.String())
}

func (s *executorSuite) TestInject(c *C) {
	hostdir := c.MkDir()
	c.Assert(os.WriteFile(hostdir+"/testfile", []byte("first tube"), 0777), IsNil)
	mountdir := fmt.Sprintf("/injecttest-%d", os.Getpid())
	s.spec.Command = []string{"sleep", "10"}
	s.spec.BindMounts = map[string]bindmount{mountdir: {HostPath: hostdir, ReadOnly: true}}
	c.Assert(s.executor.Create(s.spec), IsNil)
	c.Assert(s.executor.Start(), IsNil)
	starttime := time.Now()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Second))
	defer cancel()

	// Allow InjectCommand to fail a few times while the container
	// is starting
	for ctx.Err() == nil {
		_, err := s.executor.InjectCommand(ctx, "", "root", false, []string{"true"})
		if err == nil {
			break
		}
		time.Sleep(time.Second / 10)
	}

	injectcmd := []string{"cat", mountdir + "/testfile"}
	cmd, err := s.executor.InjectCommand(ctx, "", "root", false, injectcmd)
	c.Assert(err, IsNil)
	out, err := cmd.CombinedOutput()
	c.Logf("inject %s => %q", injectcmd, out)
	c.Check(err, IsNil)
	c.Check(string(out), Equals, "first tube")

	s.executor.Stop()
	code, _ := s.executor.Wait(ctx)
	c.Logf("container ran for %v", time.Now().Sub(starttime))
	c.Check(code, Equals, -1)
}

func (s *executorSuite) checkRun(c *C, expectCode int) {
	c.Assert(s.executor.Create(s.spec), IsNil)
	c.Assert(s.executor.Start(), IsNil)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	code, err := s.executor.Wait(ctx)
	c.Assert(err, IsNil)
	c.Check(code, Equals, expectCode)
}
