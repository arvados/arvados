// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/install"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

var (
	handler = cmd.Multi(map[string]cmd.Handler{
		"version":   cmd.Version,
		"-version":  cmd.Version,
		"--version": cmd.Version,

		"build":       cmdFunc(build),
		"testinstall": cmdFunc(testinstall),
		"_fpm":        cmdFunc(fpm),    // internal use
		"_install":    install.Command, // internal use
	})
)

func main() {
	if len(os.Args) < 2 || strings.HasPrefix(os.Args[1], "-") {
		parseFlags(os.Args[0], []string{"-help"}, os.Stderr)
		os.Exit(2)
	}
	os.Exit(handler.RunCommand(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type cmdFunc func(ctx context.Context, opts opts, stdin io.Reader, stdout, stderr io.Writer) error

func (cf cmdFunc) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "text", "info")
	ctx := ctxlog.Context(context.Background(), logger)
	opts, ok, code := parseFlags(prog, args, stderr)
	if !ok {
		return code
	}
	err := cf(ctx, opts, stdin, stdout, stderr)
	if err != nil {
		logger.WithError(err).Error("failed")
		return 1
	}
	return 0
}

type opts struct {
	PackageVersion string
	PackageDir     string
	PackageChown   string
	RebuildImage   bool
	SourceDir      string
	TargetOS       string
	Maintainer     string
	Vendor         string
	Live           string
}

func parseFlags(prog string, args []string, stderr io.Writer) (_ opts, ok bool, exitCode int) {
	opts := opts{
		SourceDir:  ".",
		TargetOS:   "debian:10",
		Maintainer: "Arvados Package Maintainers <packaging@arvados.org>",
		Vendor:     "The Arvados Project",
	}
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.StringVar(&opts.PackageVersion, "package-version", opts.PackageVersion, "package version to build/test, like \"1.2.3\"")
	flags.StringVar(&opts.SourceDir, "source", opts.SourceDir, "arvados source tree location")
	flags.StringVar(&opts.PackageDir, "package-dir", opts.PackageDir, "destination directory for new package (default is cwd)")
	flags.StringVar(&opts.PackageChown, "package-chown", opts.PackageChown, "desired uid:gid for new package (default is current user:group)")
	flags.StringVar(&opts.TargetOS, "target-os", opts.TargetOS, "target operating system vendor:version")
	flags.StringVar(&opts.Maintainer, "package-maintainer", opts.Maintainer, "maintainer to be listed in package metadata")
	flags.StringVar(&opts.Vendor, "package-vendor", opts.Vendor, "vendor to be listed in package metadata")
	flags.StringVar(&opts.Live, "live", opts.Live, "run controller at https://`example.com`, use host's /var/lib/acme/live certificates, wait for ^C before shutting down")
	flags.BoolVar(&opts.RebuildImage, "rebuild-image", opts.RebuildImage, "rebuild docker image(s) instead of using existing")
	flags.Usage = func() {
		fmt.Fprint(flags.Output(), `Usage: arvados-package <subcommand> [options]

Subcommands:
	build
		use a docker container to build a package from a checked
		out version of the arvados source tree
	testinstall
		use a docker container to install a package and confirm
		the resulting installation is functional
	version
		show program version

Internally used subcommands:
	_fpm
		build a package
	_install
		equivalent to "arvados-server install"

Automation/integration notes:
	The first time a given machine runs "build" or "testinstall" (and
	any time the -rebuild-image is used), new docker images are built,
	which is quite slow. If you use on-demand VMs to run automated builds,
	run "build" and "testinstall" once when setting up your initial VM
	image, and be prepared to rebuild that VM image when package-building
	slows down (this will happen when new dependencies are introduced).

	The "build" subcommand, if successful, also runs
	dpkg-scanpackages to create/replace Packages.gz in the package
	dir. This enables the "testinstall" subcommand to list the
	package dir as a source in /etc/apt/sources.*.

Options:
`)
		flags.PrintDefaults()
	}
	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		return opts, false, code
	}
	if opts.SourceDir == "" {
		d, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "error getting current working directory: %s\n", err)
			return opts, false, 1
		}
		opts.SourceDir = d
	}
	opts.PackageDir = filepath.Clean(opts.PackageDir)
	abs, err := filepath.Abs(opts.SourceDir)
	if err != nil {
		fmt.Fprintf(stderr, "error resolving source dir %q: %s\n", opts.SourceDir, err)
		return opts, false, 1
	}
	opts.SourceDir = abs
	return opts, true, 0
}
