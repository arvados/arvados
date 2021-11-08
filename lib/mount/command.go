// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"flag"
	"io"
	"log"
	"net/http"

	// pprof is only imported to register its HTTP handlers
	_ "net/http/pprof"
	"os"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/arvados/cgofuse/fuse"
)

var Command = &cmd{}

type cmd struct {
	// ready, if non-nil, will be closed when the mount is
	// initialized.  If ready is non-nil, it RunCommand() should
	// not be called more than once, or when ready is already
	// closed.
	ready chan struct{}
	// It is safe to call Unmount only after ready has been
	// closed.
	Unmount func() (ok bool)
}

// RunCommand implements the subcommand "mount <path> [fuse options]".
//
// The "-d" fuse option (and perhaps other features) ignores the
// stderr argument and prints to os.Stderr instead.
func (c *cmd) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := log.New(stderr, prog+" ", 0)
	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	ro := flags.Bool("ro", false, "read-only")
	experimental := flags.Bool("experimental", false, "acknowledge this is an experimental command, and should not be used in production (required)")
	blockCache := flags.Int("block-cache", 4, "read cache size (number of 64MiB blocks)")
	pprof := flags.String("pprof", "", "serve Go profile data at `[addr]:port`")
	err := flags.Parse(args)
	if err != nil {
		logger.Print(err)
		return 2
	}
	if !*experimental {
		logger.Printf("error: experimental command %q used without --experimental flag", prog)
		return 2
	}
	if *pprof != "" {
		go func() {
			log.Println(http.ListenAndServe(*pprof, nil))
		}()
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
	kc.BlockCache = &keepclient.BlockCache{MaxBlocks: *blockCache}
	host := fuse.NewFileSystemHost(&keepFS{
		Client:     client,
		KeepClient: kc,
		ReadOnly:   *ro,
		Uid:        os.Getuid(),
		Gid:        os.Getgid(),
		ready:      c.ready,
	})
	c.Unmount = host.Unmount
	ok := host.Mount("", flags.Args())
	if !ok {
		return 1
	}
	return 0
}
