// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"
)

func busyboxDockerImage(c *C) string {
	fnm := "busybox_uclibc.tar"
	cachedir := c.MkDir()
	cachefile := cachedir + "/" + fnm
	if _, err := os.Stat(cachefile); err == nil {
		return cachefile
	}

	f, err := ioutil.TempFile(cachedir, "")
	c.Assert(err, IsNil)
	defer f.Close()
	defer os.Remove(f.Name())

	resp, err := http.Get("https://cache.arvados.org/" + fnm)
	c.Assert(err, IsNil)
	defer resp.Body.Close()
	_, err = io.Copy(f, resp.Body)
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)
	err = os.Rename(f.Name(), cachefile)
	c.Assert(err, IsNil)

	return cachefile
}

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
	err := s.executor.LoadImage("", busyboxDockerImage(c), arvados.Container{}, "", nil)
	c.Assert(err, IsNil)
}

func (s *executorSuite) TearDownTest(c *C) {
	s.executor.Close()
}

func (s *executorSuite) TestExecTrivialContainer(c *C) {
	s.spec.Command = []string{"echo", "ok"}
	s.checkRun(c, 0)
	c.Check(s.stdout.String(), Equals, "ok\n")
	c.Check(s.stderr.String(), Equals, "")
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

func (s *executorSuite) checkRun(c *C, expectCode int) {
	c.Assert(s.executor.Create(s.spec), IsNil)
	c.Assert(s.executor.Start(), IsNil)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	code, err := s.executor.Wait(ctx)
	c.Assert(err, IsNil)
	c.Check(code, Equals, expectCode)
}
