// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"

	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"gopkg.in/check.v1"
)

type DockerSuite struct {
	tmpdir   string
	hostip   string
	proxyln  net.Listener
	proxysrv *http.Server
}

var _ = check.Suite(&DockerSuite{})

func Test(t *testing.T) { check.TestingT(t) }

func (s *DockerSuite) SetUpSuite(c *check.C) {
	if testing.Short() {
		c.Skip("skipping docker tests in short mode")
	} else if _, err := exec.Command("docker", "info").CombinedOutput(); err != nil {
		c.Skip("skipping docker tests because docker is not available")
	}

	s.tmpdir = c.MkDir()

	// The integration-testing controller listens on the loopback
	// interface, so it won't be reachable directly from the
	// docker container -- so here we run a proxy on 0.0.0.0 for
	// the duration of the test.
	hostips, err := exec.Command("hostname", "-I").Output()
	c.Assert(err, check.IsNil)
	s.hostip = strings.Split(strings.Trim(string(hostips), "\n"), " ")[0]
	ln, err := net.Listen("tcp", s.hostip+":0")
	c.Assert(err, check.IsNil)
	s.proxyln = ln
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "https", Host: os.Getenv("ARVADOS_API_HOST")})
	proxy.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	s.proxysrv = &http.Server{Handler: proxy}
	go s.proxysrv.ServeTLS(ln, "../../services/api/tmp/self-signed.pem", "../../services/api/tmp/self-signed.key")

	// Build a pam module to install & configure in the docker
	// container.
	cmd := exec.Command("go", "build", "-buildmode=c-shared", "-o", s.tmpdir+"/pam_arvados.so")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	c.Assert(err, check.IsNil)

	// Build the testclient program that will (from inside the
	// docker container) configure the system to use the above PAM
	// config, and then try authentication.
	cmd = exec.Command("go", "build", "-o", s.tmpdir+"/testclient", "./testclient.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	c.Assert(err, check.IsNil)
}

func (s *DockerSuite) TearDownSuite(c *check.C) {
	if s.proxysrv != nil {
		s.proxysrv.Close()
	}
	if s.proxyln != nil {
		s.proxyln.Close()
	}
}

func (s *DockerSuite) SetUpTest(c *check.C) {
	// Write a PAM config file that uses our proxy as
	// ARVADOS_API_HOST.
	proxyhost := s.proxyln.Addr().String()
	confdata := fmt.Sprintf(`Name: Arvados authentication
Default: yes
Priority: 256
Auth-Type: Primary
Auth:
	[success=end default=ignore]	/usr/lib/pam_arvados.so %s testvm2.shell insecure
Auth-Initial:
	[success=end default=ignore]	/usr/lib/pam_arvados.so %s testvm2.shell insecure
`, proxyhost, proxyhost)
	err := ioutil.WriteFile(s.tmpdir+"/conffile", []byte(confdata), 0755)
	c.Assert(err, check.IsNil)
}

func (s *DockerSuite) runTestClient(c *check.C, args ...string) (stdout, stderr *bytes.Buffer, err error) {

	cmd := exec.Command("docker", append([]string{
		"run", "--rm",
		"--hostname", "testvm2.shell",
		"--add-host", "zzzzz.arvadosapi.com:" + s.hostip,
		"-v", s.tmpdir + "/pam_arvados.so:/usr/lib/pam_arvados.so:ro",
		"-v", s.tmpdir + "/conffile:/usr/share/pam-configs/arvados:ro",
		"-v", s.tmpdir + "/testclient:/testclient:ro",
		"debian:bullseye",
		"/testclient"}, args...)...)
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	return
}

func (s *DockerSuite) TestSuccess(c *check.C) {
	stdout, stderr, err := s.runTestClient(c, "try", "active", arvadostest.ActiveTokenV2)
	c.Check(err, check.IsNil)
	c.Logf("%s", stderr.String())
	c.Check(stdout.String(), check.Equals, "")
	c.Check(stderr.String(), check.Matches, `(?ms).*authentication succeeded.*`)
}

func (s *DockerSuite) TestFailure(c *check.C) {
	for _, trial := range []struct {
		label    string
		username string
		token    string
	}{
		{"bad token", "active", arvadostest.ActiveTokenV2 + "badtoken"},
		{"empty token", "active", ""},
		{"empty username", "", arvadostest.ActiveTokenV2},
		{"wrong username", "wrongusername", arvadostest.ActiveTokenV2},
	} {
		c.Logf("trial: %s", trial.label)
		stdout, stderr, err := s.runTestClient(c, "try", trial.username, trial.token)
		c.Logf("%s", stderr.String())
		c.Check(err, check.NotNil)
		c.Check(stdout.String(), check.Equals, "")
		c.Check(stderr.String(), check.Matches, `(?ms).*authentication failed.*`)
	}
}

func (s *DockerSuite) TestDefaultHostname(c *check.C) {
	confdata := fmt.Sprintf(`Name: Arvados authentication
Default: yes
Priority: 256
Auth-Type: Primary
Auth:
	[success=end default=ignore]	/usr/lib/pam_arvados.so %s - insecure debug
Auth-Initial:
	[success=end default=ignore]	/usr/lib/pam_arvados.so %s - insecure debug
`, s.proxyln.Addr().String(), s.proxyln.Addr().String())
	err := ioutil.WriteFile(s.tmpdir+"/conffile", []byte(confdata), 0755)
	c.Assert(err, check.IsNil)

	stdout, stderr, err := s.runTestClient(c, "try", "active", arvadostest.ActiveTokenV2)
	c.Check(err, check.IsNil)
	c.Logf("%s", stderr.String())
	c.Check(stdout.String(), check.Equals, "")
	c.Check(stderr.String(), check.Matches, `(?ms).*authentication succeeded.*`)
}
