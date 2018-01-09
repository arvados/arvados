// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"

	"git.curoverse.com/arvados.git/lib/cli"
	"git.curoverse.com/arvados.git/lib/cmd"
)

var (
	version                = "dev"
	cmdVersion cmd.Handler = versionCmd{}
	handler                = cmd.Multi(map[string]cmd.Handler{
		"-e":        cmdVersion,
		"version":   cmdVersion,
		"-version":  cmdVersion,
		"--version": cmdVersion,

		"copy":     cli.Copy,
		"create":   cli.Create,
		"edit":     cli.Edit,
		"get":      cli.Get,
		"keep":     cli.Keep,
		"pipeline": cli.Pipeline,
		"run":      cli.Run,
		"tag":      cli.Tag,
		"ws":       cli.Ws,

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
	})
)

type versionCmd struct{}

func (versionCmd) RunCommand(prog string, args []string, _ io.Reader, stdout, _ io.Writer) int {
	prog = regexp.MustCompile(` -*version$`).ReplaceAllLiteralString(prog, "")
	fmt.Fprintf(stdout, "%s %s (%s)\n", prog, version, runtime.Version())
	return 0
}

func fixLegacyArgs(args []string) []string {
	flags, _ := cli.LegacyFlagSet()
	return cmd.SubcommandToFront(args, flags)
}

func main() {
	os.Exit(handler.RunCommand(os.Args[0], fixLegacyArgs(os.Args[1:]), os.Stdin, os.Stdout, os.Stderr))
}
