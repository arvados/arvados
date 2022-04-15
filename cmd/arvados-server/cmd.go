// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

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
	"git.arvados.org/arvados.git/services/githttpd"
	"git.arvados.org/arvados.git/services/keepproxy"
	"git.arvados.org/arvados.git/services/keepstore"
	"git.arvados.org/arvados.git/services/ws"
)

var (
	handler = cmd.Multi(map[string]cmd.Handler{
		"version":   cmd.Version,
		"-version":  cmd.Version,
		"--version": cmd.Version,

		"boot":               boot.Command,
		"cloudtest":          cloudtest.Command,
		"config-check":       config.CheckCommand,
		"config-defaults":    config.DumpDefaultsCommand,
		"config-dump":        config.DumpCommand,
		"controller":         controller.Command,
		"crunch-run":         crunchrun.Command,
		"dispatch-cloud":     dispatchcloud.Command,
		"dispatch-lsf":       lsf.DispatchCommand,
		"git-httpd":          githttpd.Command,
		"install":            install.Command,
		"init":               install.InitCommand,
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
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(args[2])))
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
