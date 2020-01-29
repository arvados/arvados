// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

func runNginx(ctx context.Context, boot *Booter) error {
	vars := map[string]string{
		"SSLCERT":   filepath.Join(boot.SourcePath, "services", "api", "tmp", "self-signed.pem"), // TODO: root ca
		"SSLKEY":    filepath.Join(boot.SourcePath, "services", "api", "tmp", "self-signed.key"), // TODO: root ca
		"ACCESSLOG": filepath.Join(boot.tempdir, "nginx_access.log"),
		"ERRORLOG":  filepath.Join(boot.tempdir, "nginx_error.log"),
		"TMPDIR":    boot.tempdir,
	}
	var err error
	for _, cmpt := range []struct {
		varname string
		svc     arvados.Service
	}{
		{"CONTROLLER", boot.cluster.Services.Controller},
		{"KEEPWEB", boot.cluster.Services.WebDAV},
		{"KEEPWEBDL", boot.cluster.Services.WebDAVDownload},
		{"KEEPPROXY", boot.cluster.Services.Keepproxy},
		{"GIT", boot.cluster.Services.GitHTTP},
		{"WS", boot.cluster.Services.Websocket},
	} {
		vars[cmpt.varname+"PORT"], err = internalPort(cmpt.svc)
		if err != nil {
			return fmt.Errorf("%s internal port: %s (%v)", cmpt.varname, err, cmpt.svc)
		}
		vars[cmpt.varname+"SSLPORT"], err = externalPort(cmpt.svc)
		if err != nil {
			return fmt.Errorf("%s external port: %s (%v)", cmpt.varname, err, cmpt.svc)
		}
	}
	tmpl, err := ioutil.ReadFile(filepath.Join(boot.SourcePath, "sdk", "python", "tests", "nginx.conf"))
	if err != nil {
		return err
	}
	conf := regexp.MustCompile(`{{.*?}}`).ReplaceAllStringFunc(string(tmpl), func(src string) string {
		if len(src) < 4 {
			return src
		}
		return vars[src[2:len(src)-2]]
	})
	conffile := filepath.Join(boot.tempdir, "nginx.conf")
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
	return boot.RunProgram(ctx, ".", nil, nil, nginx,
		"-g", "error_log stderr info;",
		"-g", "pid "+filepath.Join(boot.tempdir, "nginx.pid")+";",
		"-c", conffile)
}
