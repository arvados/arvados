// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"git.arvados.org/arvados.git/lib/cli"
	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/costanalyzer"
	"git.arvados.org/arvados.git/lib/deduplicationreport"
	"git.arvados.org/arvados.git/lib/diagnostics"
	"git.arvados.org/arvados.git/lib/mount"
)

var (
	handler = cmd.Multi(map[string]cmd.Handler{
		"-e":        cmd.Version,
		"version":   cmd.Version,
		"-version":  cmd.Version,
		"--version": cmd.Version,

		"copy":   cli.Copy,
		"create": cli.Create,
		"edit":   cli.Edit,
		"get":    cli.Get,
		"keep":   cli.Keep,
		"tag":    cli.Tag,
		"ws":     cli.Ws,

		"api_client_authorization": cli.APICall,
		"api_client":               cli.APICall,
		"authorized_key":           cli.APICall,
		"collection":               cli.APICall,
		"container":                cli.APICall,
		"container_request":        cli.APICall,
		"group":                    cli.APICall,
		"human":                    cli.APICall,
		"job":                      cli.APICall,
		"job_task":                 cli.APICall,
		"keep_disk":                cli.APICall,
		"keep_service":             cli.APICall,
		"link":                     cli.APICall,
		"log":                      cli.APICall,
		"node":                     cli.APICall,
		"pipeline_instance":        cli.APICall,
		"pipeline_template":        cli.APICall,
		"repository":               cli.APICall,
		"specimen":                 cli.APICall,
		"trait":                    cli.APICall,
		"user_agreement":           cli.APICall,
		"user":                     cli.APICall,
		"virtual_machine":          cli.APICall,
		"workflow":                 cli.APICall,

		"connect-ssh":          connectSSHCommand{},
		"costanalyzer":         costanalyzer.Command,
		"deduplication-report": deduplicationreport.Command,
		"diagnostics":          diagnostics.Command{},
		"logs":                 logsCommand{},
		"mount":                mount.Command,
		"shell":                shellCommand{},
		"sudo":                 sudoCommand{},
	})
)

func fixLegacyArgs(args []string) []string {
	flags, _ := cli.LegacyFlagSet()
	return cmd.SubcommandToFront(args, flags)
}

func main() {
	os.Exit(handler.RunCommand(os.Args[0], fixLegacyArgs(os.Args[1:]), os.Stdin, os.Stdout, os.Stderr))
}
