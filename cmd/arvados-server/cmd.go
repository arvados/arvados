// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// @Version 1.0.0
// @Title Arvados API
// @Description This is the Arvados API
// @ContactName Nico Cesar
// @ContactEmail nico@curii.com
// @ContactURL https://arvados.org
// @LicenseName Apache-2.0
// @LicenseURL https://doc.arvados.org/user/copying/copying.html
// @Security Authorization read write
// @SecurityScheme Authorization apiKey header Authorization

package main

import (
	"os"

	"git.arvados.org/arvados.git/lib/boot"
	"git.arvados.org/arvados.git/lib/cloud/cloudtest"
	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller"
	"git.arvados.org/arvados.git/lib/crunchrun"
	"git.arvados.org/arvados.git/lib/dispatchcloud"
	"git.arvados.org/arvados.git/lib/install"
	"git.arvados.org/arvados.git/lib/recovercollection"
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
		"install":            install.Command,
		"recover-collection": recovercollection.Command,
		"ws":                 ws.Command,
	})
)

func main() {
	os.Exit(handler.RunCommand(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
