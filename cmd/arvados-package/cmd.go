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
		"fpm":         cmdFunc(fpm),
		"testinstall": cmdFunc(testinstall),
		"_install":    install.Command, // internal use
	})
)

func main() {
	os.Exit(handler.RunCommand(os.Args[0], os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type cmdFunc func(ctx context.Context, opts opts, stdin io.Reader, stdout, stderr io.Writer) error

func (cf cmdFunc) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "text", "info")
	ctx := ctxlog.Context(context.Background(), logger)
	opts, err := parseFlags(args)
	if err != nil {
		logger.WithError(err).Error("error parsing command line flags")
		return 1
	}
	err = cf(ctx, opts, stdin, stdout, stderr)
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
}

func parseFlags(args []string) (opts, error) {
	opts := opts{
		TargetOS: "debian:10",
	}
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.StringVar(&opts.PackageVersion, "package-version", opts.PackageVersion, "package version to build/test, like \"1.2.3\"")
	flags.StringVar(&opts.SourceDir, "source", opts.SourceDir, "arvados source tree location")
	flags.StringVar(&opts.PackageDir, "package-dir", opts.PackageDir, "destination directory for new package (default is cwd)")
	flags.StringVar(&opts.PackageChown, "package-chown", opts.PackageChown, "desired uid:gid for new package (default is current user:group)")
	flags.StringVar(&opts.TargetOS, "target-os", opts.TargetOS, "target operating system vendor:version")
	flags.BoolVar(&opts.RebuildImage, "rebuild-image", opts.RebuildImage, "rebuild docker image(s) instead of using existing")
	err := flags.Parse(args)
	if err != nil {
		return opts, err
	}
	if len(flags.Args()) > 0 {
		return opts, fmt.Errorf("unrecognized command line arguments: %v", flags.Args())
	}
	if opts.SourceDir == "" {
		d, err := os.Getwd()
		if err != nil {
			return opts, fmt.Errorf("Getwd: %w", err)
		}
		opts.SourceDir = d
	}
	opts.PackageDir = filepath.Clean(opts.PackageDir)
	return opts, nil
}
