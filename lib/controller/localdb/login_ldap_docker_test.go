// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&LDAPDockerSuite{})

type LDAPDockerSuite struct {
	localdbSuite
	tmpdir     string
	netName    string
	netAddr    string
	pgProxy    *tcpProxy
	railsProxy *tcpProxy
}

func (s *LDAPDockerSuite) setUpDockerNetwork() (string, error) {
	netName := "arvados-net-" + path.Base(path.Dir(s.tmpdir))
	cmd := exec.Command("docker", "network", "create", netName)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return netName, nil
}

func (s *LDAPDockerSuite) ipFromCmd(cmd *exec.Cmd) (string, error) {
	cmd.Stderr = os.Stderr
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	stdout := bufio.NewReader(outPipe)
	var ip net.IP
	var readErr error
	for readErr == nil {
		var line string
		line, readErr = stdout.ReadString('\n')
		if lineIP := net.ParseIP(strings.TrimSuffix(line, "\n")); lineIP != nil {
			ip = lineIP
		}
	}
	if err := cmd.Wait(); err != nil {
		return "", err
	} else if readErr != io.EOF {
		return "", readErr
	} else {
		return ip.String(), nil
	}
}

func (s *LDAPDockerSuite) SetUpSuite(c *check.C) {
	s.localdbSuite.SetUpSuite(c)
	s.tmpdir = c.MkDir()
	var err error
	s.netName, err = s.setUpDockerNetwork()
	c.Assert(err, check.IsNil)
	s.netAddr, err = s.ipFromCmd(exec.Command("docker", "network", "inspect",
		"--format", "{{(index .IPAM.Config 0).Gateway}}", s.netName))
	c.Assert(err, check.IsNil)
	setup := exec.Command("login_ldap_docker_test/setup_suite.sh", s.netName, s.tmpdir)
	setup.Stderr = os.Stderr
	err = setup.Run()
	c.Assert(err, check.IsNil)
}

func (s *LDAPDockerSuite) TearDownSuite(c *check.C) {
	if s.netName != "" {
		cmd := exec.Command("login_ldap_docker_test/teardown_suite.sh", s.netName)
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		c.Check(err, check.IsNil)
	}
	s.localdbSuite.TearDownSuite(c)
}

func (s *LDAPDockerSuite) setUpConfig(c *check.C) {
	src, err := os.Open(os.Getenv("ARVADOS_CONFIG"))
	c.Assert(err, check.IsNil)
	defer src.Close()
	dst, err := os.Create(path.Join(s.tmpdir, "arvados.yml"))
	c.Assert(err, check.IsNil)
	_, err = io.Copy(dst, src)
	dst.Close()
	c.Assert(err, check.IsNil)

	pgconn := &map[string]interface{}{
		"host": s.netAddr,
		"port": s.pgProxy.Port(),
	}
	err = s.updateConfig(".Clusters.zzzzz.PostgreSQL.Connection |= (. * $arg)", pgconn)
	c.Assert(err, check.IsNil)
	intVal := make(map[string]string)
	intURLs := make(map[string]interface{})
	railsURL := "https://" + net.JoinHostPort(s.netAddr, s.railsProxy.Port())
	intURLs[railsURL] = &intVal
	err = s.updateConfig(".Clusters.zzzzz.Services.RailsAPI.InternalURLs = $arg", &intURLs)
	c.Assert(err, check.IsNil)
	intURLs = make(map[string]interface{})
	intURLs["http://0.0.0.0:80"] = &intVal
	err = s.updateConfig(".Clusters.zzzzz.Services.Controller.InternalURLs = $arg", &intURLs)
	c.Assert(err, check.IsNil)
}

func (s *LDAPDockerSuite) updateConfig(expr string, arg *map[string]interface{}) error {
	jsonArg, err := json.Marshal(arg)
	if err != nil {
		return err
	}
	cmd := exec.Command("yq", "-yi",
		"--argjson", "arg", string(jsonArg),
		expr, path.Join(s.tmpdir, "arvados.yml"))
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (s *LDAPDockerSuite) enableLogin(key string) error {
	login := make(map[string]interface{})
	login["Test"] = &map[string]bool{"Enable": false}
	login[key] = &map[string]bool{"Enable": true}
	return s.updateConfig(".Clusters.zzzzz.Login |= (. * $arg)", &login)
}

func (s *LDAPDockerSuite) SetUpTest(c *check.C) {
	s.localdbSuite.SetUpTest(c)
	s.pgProxy = newPgProxy(c, s.cluster, s.netAddr)
	s.railsProxy = newInternalProxy(c, s.cluster.Services.RailsAPI, s.netAddr)
	s.setUpConfig(c)
}

func (s *LDAPDockerSuite) TearDownTest(c *check.C) {
	cidPath := path.Join(s.tmpdir, "controller.cid")
	if cid, err := os.ReadFile(cidPath); err == nil {
		cmd := exec.Command("docker", "stop", strings.TrimSpace(string(cid)))
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		c.Check(err, check.IsNil)
	}
	if err := os.Remove(cidPath); err != nil {
		c.Check(err, check.Equals, os.ErrNotExist)
	}
	s.railsProxy.Close()
	s.pgProxy.Close()
	s.localdbSuite.TearDownTest(c)
}

func (s *LDAPDockerSuite) startController(args ...string) (*url.URL, error) {
	args = append([]string{s.netName, s.tmpdir}, args...)
	cmd := exec.Command("login_ldap_docker_test/start_controller_container.sh", args...)
	ip, err := s.ipFromCmd(cmd)
	if err != nil {
		return nil, err
	}
	return &url.URL{
		Scheme: "http",
		Host:   ip,
	}, nil
}

func (s *LDAPDockerSuite) parseResponse(resp *http.Response, body any) error {
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 400 {
		return json.Unmarshal(respBody, body)
	}
	errBody := make(map[string]interface{})
	err = json.Unmarshal(respBody, &errBody)
	if err != nil {
		return fmt.Errorf("%s: error unmarshaling error response: %w", resp.Status, err)
	}
	errors, ok := errBody["errors"]
	if !ok {
		return fmt.Errorf("%s: error response did not include 'errors' key", resp.Status)
	}
	errList, ok := errors.([]interface{})
	if !ok {
		return fmt.Errorf("%s: error response 'errors' was not an array", resp.Status)
	} else if len(errList) == 0 {
		return fmt.Errorf("%s: error response with empty 'errors'", resp.Status)
	} else {
		return fmt.Errorf("%s: %s", resp.Status, errList[0])
	}
}

func (s *LDAPDockerSuite) authenticate(server *url.URL, username, password string) (*arvados.APIClientAuthorization, error) {
	reqURL := server.JoinPath("/arvados/v1/users/authenticate").String()
	reqValues := url.Values{
		"username": {username},
		"password": {password},
	}
	resp, err := http.PostForm(reqURL, reqValues)
	if err != nil {
		return nil, err
	}
	token := &arvados.APIClientAuthorization{}
	err = s.parseResponse(resp, token)
	return token, err
}

func (s *LDAPDockerSuite) getCurrentUser(server *url.URL, token string) (*arvados.User, error) {
	reqURL := server.JoinPath("/arvados/v1/users/current").String()
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	user := &arvados.User{}
	err = s.parseResponse(resp, user)
	return user, err
}

func (s *LDAPDockerSuite) TestLoginPAMCallingLDAP(c *check.C) {
	err := s.enableLogin("PAM")
	c.Assert(err, check.IsNil)
	setupPath, err := filepath.Abs("login_ldap_docker_test/setup_pam_test.sh")
	c.Assert(err, check.IsNil)
	arvURL, err := s.startController("-v", setupPath+":/setup.sh:ro")
	c.Assert(err, check.IsNil)
	_, err = s.authenticate(arvURL, "foo-bar", "nosecret")
	c.Check(err, check.ErrorMatches, `401 Unauthorized: PAM: Authentication failure \(with username "foo-bar" and password\)`)
	_, err = s.authenticate(arvURL, "expired", "secret")
	c.Check(err, check.ErrorMatches, `401 Unauthorized: PAM: Authentication failure; "Your account has expired; please contact your system administrator\."`)
	aca, err := s.authenticate(arvURL, "foo-bar", "secret")
	if c.Check(err, check.IsNil) {
		user, err := s.getCurrentUser(arvURL, aca.TokenV2())
		if c.Check(err, check.IsNil) {
			// Check PAMDefaultEmailDomain was propagated as expected
			c.Check(user.Email, check.Equals, "foo-bar@example.com")
		}
	}
}

func (s *LDAPDockerSuite) TestLoginLDAPBuiltin(c *check.C) {
	err := s.enableLogin("LDAP")
	c.Assert(err, check.IsNil)
	arvURL, err := s.startController()
	c.Assert(err, check.IsNil)
	_, err = s.authenticate(arvURL, "foo-bar", "nosecret")
	c.Check(err, check.ErrorMatches, `401 Unauthorized: LDAP: Authentication failure \(with username "foo-bar" and password\)`)
	aca, err := s.authenticate(arvURL, "foo-bar", "secret")
	if c.Check(err, check.IsNil) {
		user, err := s.getCurrentUser(arvURL, aca.TokenV2())
		if c.Check(err, check.IsNil) {
			// User fields come from LDAP attributes
			c.Check(user.FirstName, check.Equals, "Foo")
			c.Check(user.LastName, check.Equals, "Bar")
			// "-" character removed by RailsAPI
			c.Check(user.Username, check.Equals, "foobar")
			c.Check(user.Email, check.Equals, "foo-bar-baz@example.com")
		}
	}
}
