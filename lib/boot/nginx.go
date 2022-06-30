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
	"path/filepath"
	"regexp"
	"strings"

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
		"LISTENHOST":       "0.0.0.0",
		"UPSTREAMHOST":     super.ListenHost,
		"SSLCERT":          filepath.Join(super.tempdir, "server.crt"),
		"SSLKEY":           filepath.Join(super.tempdir, "server.key"),
		"ACCESSLOG":        filepath.Join(super.tempdir, "nginx_access.log"),
		"ERRORLOG":         filepath.Join(super.tempdir, "nginx_error.log"),
		"TMPDIR":           super.wwwtempdir,
		"ARVADOS_API_HOST": super.cluster.Services.Controller.ExternalURL.Host,
	}
	u := url.URL(super.cluster.Services.Controller.ExternalURL)
	ctrlHost := u.Hostname()
	if strings.HasPrefix(super.cluster.TLS.Certificate, "file:/") && strings.HasPrefix(super.cluster.TLS.Key, "file:/") {
		vars["SSLCERT"] = filepath.Clean(super.cluster.TLS.Certificate[5:])
		vars["SSLKEY"] = filepath.Clean(super.cluster.TLS.Key[5:])
	} else if f, err := os.Open("/var/lib/acme/live/" + ctrlHost + "/privkey"); err == nil {
		f.Close()
		vars["SSLCERT"] = "/var/lib/acme/live/" + ctrlHost + "/cert"
		vars["SSLKEY"] = "/var/lib/acme/live/" + ctrlHost + "/privkey"
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
		{"WORKBENCH2", super.cluster.Services.Workbench2},
		{"WS", super.cluster.Services.Websocket},
	} {
		var host, port string
		if len(cmpt.svc.InternalURLs) == 0 {
			// We won't run this service, but we need an
			// upstream port to write in our templated
			// nginx config. Choose a port that will
			// return 502 Bad Gateway.
			port = "9"
		} else if host, port, err = internalPort(cmpt.svc); err != nil {
			return fmt.Errorf("%s internal port: %w (%v)", cmpt.varname, err, cmpt.svc)
		} else if ok, err := addrIsLocal(net.JoinHostPort(host, port)); !ok || err != nil {
			return fmt.Errorf("%s addrIsLocal() failed for host %q port %q: %v", cmpt.varname, host, port, err)
		}
		vars[cmpt.varname+"PORT"] = port

		port, err = externalPort(cmpt.svc)
		if err != nil {
			return fmt.Errorf("%s external port: %w (%v)", cmpt.varname, err, cmpt.svc)
		}
		listenAddr := net.JoinHostPort(super.ListenHost, port)
		if ok, err := addrIsLocal(listenAddr); !ok || err != nil {
			return fmt.Errorf("%s addrIsLocal(%q) failed: %w", cmpt.varname, listenAddr, err)
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

	configs := "error_log stderr info; "
	configs += "pid " + filepath.Join(super.wwwtempdir, "nginx.pid") + "; "
	configs += "user www-data; "

	super.waitShutdown.Add(1)
	go func() {
		defer super.waitShutdown.Done()
		fail(super.RunProgram(ctx, ".", runOptions{}, nginx, "-g", configs, "-c", conffile))
	}()
	// Choose one of the ports where Nginx should listen, and wait
	// here until we can connect. If ExternalURL is https://foo
	// (with no port) then we connect to "foo:https"
	testurl := url.URL(super.cluster.Services.Controller.ExternalURL)
	if testurl.Port() == "" {
		testurl.Host = net.JoinHostPort(testurl.Host, testurl.Scheme)
	}
	return waitForConnect(ctx, testurl.Host)
}
