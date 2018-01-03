// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"flag"
	"io"
	"log"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"github.com/curoverse/cgofuse/fuse"
)

var Command = cmd{}

type cmd struct{}

func (cmd) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := log.New(stderr, prog+" ", 0)
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	ro := flags.Bool("ro", false, "read-only")
	err := flags.Parse(args)
	if err != nil {
		logger.Print(err)
		return 2
	}

	client := arvados.NewClientFromEnv()
	ac, err := arvadosclient.New(client)
	if err != nil {
		logger.Print(err)
		return 1
	}
	kc, err := keepclient.MakeKeepClient(ac)
	if err != nil {
		logger.Print(err)
		return 1
	}
	host := fuse.NewFileSystemHost(&keepFS{
		Client:     client,
		KeepClient: kc,
		ReadOnly:   *ro,
		Uid:        os.Getuid(),
		Gid:        os.Getgid(),
	})
	notOK := host.Mount("", flags.Args())
	if notOK {
		return 1
	} else {
		return 0
	}
}
