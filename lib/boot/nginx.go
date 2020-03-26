// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
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
	vars := map[string]string{
		"LISTENHOST": super.ListenHost,
		"SSLCERT":    filepath.Join(super.SourcePath, "services", "api", "tmp", "self-signed.pem"), // TODO: root ca
		"SSLKEY":     filepath.Join(super.SourcePath, "services", "api", "tmp", "self-signed.key"), // TODO: root ca
		"ACCESSLOG":  filepath.Join(super.tempdir, "nginx_access.log"),
		"ERRORLOG":   filepath.Join(super.tempdir, "nginx_error.log"),
		"TMPDIR":     super.tempdir,
	}
	var err error
	for _, cmpt := range []struct {
		varname string
		svc     arvados.Service
	}{
		{"CONTROLLER", super.cluster.Services.Controller},
		{"KEEPWEB", super.cluster.Services.WebDAV},
		{"KEEPWEBDL", super.cluster.Services.WebDAVDownload},
		{"KEEPPROXY", super.cluster.Services.Keepproxy},
		{"GIT", super.cluster.Services.GitHTTP},
		{"WORKBENCH1", super.cluster.Services.Workbench1},
		{"WS", super.cluster.Services.Websocket},
	} {
		port, err := internalPort(cmpt.svc)
		if err != nil {
			return fmt.Errorf("%s internal port: %s (%v)", cmpt.varname, err, cmpt.svc)
		}
		if ok, err := addrIsLocal(net.JoinHostPort(super.ListenHost, port)); !ok || err != nil {
			return fmt.Errorf("urlIsLocal() failed for host %q port %q: %v", super.ListenHost, port, err)
		}
		vars[cmpt.varname+"PORT"] = port

		port, err = externalPort(cmpt.svc)
		if err != nil {
			return fmt.Errorf("%s external port: %s (%v)", cmpt.varname, err, cmpt.svc)
		}
		if ok, err := addrIsLocal(net.JoinHostPort(super.ListenHost, port)); !ok || err != nil {
			return fmt.Errorf("urlIsLocal() failed for host %q port %q: %v", super.ListenHost, port, err)
		}
		vars[cmpt.varname+"SSLPORT"] = port
	}
	tmpl, err := ioutil.ReadFile(filepath.Join(super.SourcePath, "sdk", "python", "tests", "nginx.conf"))
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
	super.waitShutdown.Add(1)
	go func() {
		defer super.waitShutdown.Done()
		fail(super.RunProgram(ctx, ".", nil, nil, nginx,
			"-g", "error_log stderr info;",
			"-g", "pid "+filepath.Join(super.tempdir, "nginx.pid")+";",
			"-c", conffile))
	}()
	return waitForConnect(ctx, super.cluster.Services.Controller.ExternalURL.Host)
}
