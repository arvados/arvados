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

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/arvados/cgofuse/fuse"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

var Command = &mountCommand{}

type mountCommand struct {
	// ready, if non-nil, will be closed when the mount is
	// initialized.  If ready is non-nil, it RunCommand() should
	// not be called more than once, or when ready is already
	// closed.  Only intended for testing.
	ready chan struct{}
	// It is safe to call Unmount only after ready has been
	// closed.
	Unmount func() (ok bool)
}

// RunCommand implements the subcommand "mount <path> [fuse options]".
//
// The "-d" fuse option (and perhaps other features) ignores the
// stderr argument and prints to os.Stderr instead.
func (c *mountCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "text", "info")
	defer logger.Debug("exiting")

	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	ro := flags.Bool("ro", false, "read-only")
	experimental := flags.Bool("experimental", false, "acknowledge this is an experimental command, and should not be used in production (required)")
	cacheSizeStr := flags.String("cache-size", "0", "cache size as percent of home filesystem size (\"5%\") or size (\"10GiB\") or 0 for automatic")
	logLevel := flags.String("log-level", "info", "logging level (debug, info, ...)")
	debug := flags.Bool("debug", false, "alias for -log-level=debug")
	pprof := flags.String("pprof", "", "serve Go profile data at `[addr]:port`")
	if ok, code := cmd.ParseFlags(flags, prog, args, "[FUSE mount options]", stderr); !ok {
		return code
	}
	if !*experimental {
		logger.Errorf("experimental command %q used without --experimental flag", prog)
		return 2
	}
	lvl, err := logrus.ParseLevel(*logLevel)
	if err != nil {
		logger.WithError(err).Error("invalid argument for -log-level flag")
		return 2
	}
	if *debug {
		lvl = logrus.DebugLevel
	}
	logger.SetLevel(lvl)
	if *pprof != "" {
		go func() {
			log.Println(http.ListenAndServe(*pprof, nil))
		}()
	}

	client := arvados.NewClientFromEnv()
	if err := yaml.Unmarshal([]byte(*cacheSizeStr), &client.DiskCacheSize); err != nil {
		logger.Errorf("error parsing -cache-size argument: %s", err)
		return 2
	}
	ac, err := arvadosclient.New(client)
	if err != nil {
		logger.Error(err)
		return 1
	}
	kc, err := keepclient.MakeKeepClient(ac)
	if err != nil {
		logger.Error(err)
		return 1
	}
	host := fuse.NewFileSystemHost(&keepFS{
		Client:     client,
		KeepClient: kc,
		ReadOnly:   *ro,
		Uid:        os.Getuid(),
		Gid:        os.Getgid(),
		Logger:     logger,
		ready:      c.ready,
	})
	c.Unmount = host.Unmount

	logger.WithField("mountargs", flags.Args()).Debug("mounting")
	ok := host.Mount("", flags.Args())
	if !ok {
		return 1
	}
	return 0
}
