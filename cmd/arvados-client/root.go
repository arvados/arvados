// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

// rootCommand runs another command using API connection info and
// SystemRootToken from the system config file instead of the caller's
// environment vars.
type rootCommand struct{}

func (rootCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	ldr := config.NewLoader(stdin, ctxlog.New(stderr, "text", "info"))
	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	ldr.SetupFlags(flags)
	if ok, code := cmd.ParseFlags(flags, prog, args, "subcommand ...", stderr); !ok {
		return code
	}
	cfg, err := ldr.Load()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	os.Setenv("ARVADOS_API_HOST", cluster.Services.Controller.ExternalURL.Host)
	os.Setenv("ARVADOS_API_TOKEN", cluster.SystemRootToken)
	if cluster.TLS.Insecure {
		os.Setenv("ARVADOS_API_HOST_INSECURE", "1")
	} else {
		os.Unsetenv("ARVADOS_API_HOST_INSECURE")
	}
	return handler.RunCommand(prog, flags.Args(), stdin, stdout, stderr)
}
