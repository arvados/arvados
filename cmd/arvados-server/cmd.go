// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"git.arvados.org/arvados.git/lib/boot"
	"git.arvados.org/arvados.git/lib/cloud/cloudtest"
	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller"
	"git.arvados.org/arvados.git/lib/crunchrun"
	"git.arvados.org/arvados.git/lib/dispatchcloud"
	"git.arvados.org/arvados.git/lib/install"
	"git.arvados.org/arvados.git/lib/lsf"
	"git.arvados.org/arvados.git/lib/recovercollection"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/health"
	dispatchslurm "git.arvados.org/arvados.git/services/crunch-dispatch-slurm"
	"git.arvados.org/arvados.git/services/githttpd"
	keepbalance "git.arvados.org/arvados.git/services/keep-balance"
	keepweb "git.arvados.org/arvados.git/services/keep-web"
	"git.arvados.org/arvados.git/services/keepproxy"
	"git.arvados.org/arvados.git/services/keepstore"
	"git.arvados.org/arvados.git/services/ws"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	handler = cmd.Multi(map[string]cmd.Handler{
		"version":   cmd.Version,
		"-version":  cmd.Version,
		"--version": cmd.Version,

		"boot":               boot.Command,
		"check":              health.CheckCommand,
		"cloudtest":          cloudtest.Command,
		"config-check":       config.CheckCommand,
		"config-defaults":    config.DumpDefaultsCommand,
		"config-dump":        config.DumpCommand,
		"controller":         controller.Command,
		"crunch-run":         crunchrun.Command,
		"dispatch-cloud":     dispatchcloud.Command,
		"dispatch-lsf":       lsf.DispatchCommand,
		"dispatch-slurm":     dispatchslurm.Command,
		"git-httpd":          githttpd.Command,
		"health":             healthCommand,
		"install":            install.Command,
		"init":               install.InitCommand,
		"keep-balance":       keepbalance.Command,
		"keep-web":           keepweb.Command,
		"keepproxy":          keepproxy.Command,
		"keepstore":          keepstore.Command,
		"recover-collection": recovercollection.Command,
		"workbench2":         wb2command{},
		"ws":                 ws.Command,
	})
)

func main() {
	os.Exit(handler.RunCommand(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type wb2command struct{}

func (wb2command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) != 3 {
		fmt.Fprintf(stderr, "usage: %s api-host listen-addr app-dir\n", prog)
		return 1
	}
	configJSON, err := json.Marshal(map[string]string{"API_HOST": args[0]})
	if err != nil {
		fmt.Fprintf(stderr, "json.Marshal: %s\n", err)
		return 1
	}
	servefs := http.FileServer(http.Dir(args[2]))
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		for _, ent := range strings.Split(req.URL.Path, "/") {
			if ent == ".." {
				http.Error(w, "invalid URL path", http.StatusBadRequest)
				return
			}
		}
		fnm := filepath.Join(args[2], filepath.FromSlash(path.Clean("/"+req.URL.Path)))
		if _, err := os.Stat(fnm); os.IsNotExist(err) {
			req.URL.Path = "/"
		}
		servefs.ServeHTTP(w, req)
	}))
	mux.HandleFunc("/config.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Write(configJSON)
	})
	mux.HandleFunc("/_health/ping", func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, `{"health":"OK"}`)
	})
	err = http.ListenAndServe(args[1], mux)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	return 0
}

var healthCommand cmd.Handler = service.Command(arvados.ServiceNameHealth, func(ctx context.Context, cluster *arvados.Cluster, _ string, reg *prometheus.Registry) service.Handler {
	mClockSkew := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "arvados",
		Subsystem: "health",
		Name:      "clock_skew_seconds",
		Help:      "Clock skew observed in most recent health check",
	})
	reg.MustRegister(mClockSkew)
	return &health.Aggregator{
		Cluster:         cluster,
		MetricClockSkew: mClockSkew,
	}
})
