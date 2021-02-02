// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// Run an Nginx process that proxies the supervisor's configured
// ExternalURLs to the appropriate InternalURLs.
type runNginx struct{}

func (runNginx) String() string {
	return "nginx"
}

func (runNginx) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	err := super.wait(ctx, createCertificates{})
	if err != nil {
		return err
	}
	vars := map[string]string{
		"LISTENHOST": super.ListenHost,
		"SSLCERT":    filepath.Join(super.tempdir, "server.crt"),
		"SSLKEY":     filepath.Join(super.tempdir, "server.key"),
		"ACCESSLOG":  filepath.Join(super.tempdir, "nginx_access.log"),
		"ERRORLOG":   filepath.Join(super.tempdir, "nginx_error.log"),
		"TMPDIR":     super.wwwtempdir,
	}
	for _, cmpt := range []struct {
		varname string
		svc     arvados.Service
	}{
		{"CONTROLLER", super.cluster.Services.Controller},
		{"KEEPWEB", super.cluster.Services.WebDAV},
		{"KEEPWEBDL", super.cluster.Services.WebDAVDownload},
		{"KEEPPROXY", super.cluster.Services.Keepproxy},
		{"GIT", super.cluster.Services.GitHTTP},
		{"HEALTH", super.cluster.Services.Health},
		{"WORKBENCH1", super.cluster.Services.Workbench1},
		{"WS", super.cluster.Services.Websocket},
	} {
		port, err := internalPort(cmpt.svc)
		if err != nil {
			return fmt.Errorf("%s internal port: %w (%v)", cmpt.varname, err, cmpt.svc)
		}
		if ok, err := addrIsLocal(net.JoinHostPort(super.ListenHost, port)); !ok || err != nil {
			return fmt.Errorf("urlIsLocal() failed for host %q port %q: %v", super.ListenHost, port, err)
		}
		vars[cmpt.varname+"PORT"] = port

		port, err = externalPort(cmpt.svc)
		if err != nil {
			return fmt.Errorf("%s external port: %w (%v)", cmpt.varname, err, cmpt.svc)
		}
		if ok, err := addrIsLocal(net.JoinHostPort(super.ListenHost, port)); !ok || err != nil {
			return fmt.Errorf("urlIsLocal() failed for host %q port %q: %v", super.ListenHost, port, err)
		}
		vars[cmpt.varname+"SSLPORT"] = port
	}
	var conftemplate string
	if super.ClusterType == "production" {
		conftemplate = "/var/lib/arvados/share/nginx.conf"
	} else {
		conftemplate = filepath.Join(super.SourcePath, "sdk", "python", "tests", "nginx.conf")
	}
	tmpl, err := ioutil.ReadFile(conftemplate)
	if err != nil {
		return err
	}
	conf := regexp.MustCompile(`{{.*?}}`).ReplaceAllStringFunc(string(tmpl), func(src string) string {
		if len(src) < 4 {
			return src
		}
		return vars[src[2:len(src)-2]]
	})
	conffile := filepath.Join(super.tempdir, "nginx.conf")
	err = ioutil.WriteFile(conffile, []byte(conf), 0755)
	if err != nil {
		return err
	}
	nginx := "nginx"
	if _, err := exec.LookPath(nginx); err != nil {
		for _, dir := range []string{"/sbin", "/usr/sbin", "/usr/local/sbin"} {
			if _, err = os.Stat(dir + "/nginx"); err == nil {
				nginx = dir + "/nginx"
				break
			}
		}
	}

	args := []string{
		"-g", "error_log stderr info;",
		"-g", "pid " + filepath.Join(super.wwwtempdir, "nginx.pid") + ";",
		"-c", conffile,
	}
	// Nginx ignores "user www-data;" when running as a non-root
	// user... except that it causes it to ignore our other -g
	// options. So we still have to decide for ourselves whether
	// it's needed.
	if u, err := user.Current(); err != nil {
		return fmt.Errorf("user.Current(): %w", err)
	} else if u.Uid == "0" {
		args = append([]string{"-g", "user www-data;"}, args...)
	}

	super.waitShutdown.Add(1)
	go func() {
		defer super.waitShutdown.Done()
		fail(super.RunProgram(ctx, ".", runOptions{}, nginx, args...))
	}()
	// Choose one of the ports where Nginx should listen, and wait
	// here until we can connect. If ExternalURL is https://foo (with no port) then we connect to "foo:https"
	testurl := url.URL(super.cluster.Services.Controller.ExternalURL)
	if testurl.Port() == "" {
		testurl.Host = net.JoinHostPort(testurl.Host, testurl.Scheme)
	}
	return waitForConnect(ctx, testurl.Host)
}
