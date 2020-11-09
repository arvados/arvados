// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/exec"

	"git.arvados.org/arvados.git/lib/install"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/sirupsen/logrus"
)

type buildPackage struct{}

func (bld buildPackage) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "text", "info")
	err := (&builder{
		PackageVersion: "0.0.0",
		logger:         logger,
	}).run(context.Background(), prog, args, stdin, stdout, stderr)
	if err != nil {
		logger.WithError(err).Error("failed")
		return 1
	}
	return 0
}

type builder struct {
	PackageVersion string
	SourcePath     string
	OutputDir      string
	SkipInstall    bool
	logger         logrus.FieldLogger
}

func (bldr *builder) run(ctx context.Context, prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.StringVar(&bldr.PackageVersion, "package-version", bldr.PackageVersion, "package version")
	flags.StringVar(&bldr.SourcePath, "source", bldr.SourcePath, "source tree location")
	flags.StringVar(&bldr.OutputDir, "output-directory", bldr.OutputDir, "destination directory for new package (default is cwd)")
	flags.BoolVar(&bldr.SkipInstall, "skip-install", bldr.SkipInstall, "skip install step, assume you have already run 'arvados-server install -type package'")
	err := flags.Parse(args)
	if err != nil {
		return err
	}
	if len(flags.Args()) > 0 {
		return fmt.Errorf("unrecognized command line arguments: %v", flags.Args())
	}
	if !bldr.SkipInstall {
		exitcode := install.Command.RunCommand("arvados-server install", []string{
			"-type", "package",
			"-package-version", bldr.PackageVersion,
			"-source", bldr.SourcePath,
		}, stdin, stdout, stderr)
		if exitcode != 0 {
			return fmt.Errorf("arvados-server install failed: exit code %d", exitcode)
		}
	}
	cmd := exec.Command("/var/lib/arvados/bin/gem", "install", "--user", "--no-document", "fpm")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("gem install fpm: %w", err)
	}

	format := "deb" // TODO: rpm

	cmd = exec.Command("/root/.gem/ruby/2.5.0/bin/fpm",
		"--name", "arvados-server-easy",
		"--version", bldr.PackageVersion,
		"--input-type", "dir",
		"--output-type", format)
	deps, err := install.ProductionDependencies()
	if err != nil {
		return err
	}
	for _, pkg := range deps {
		cmd.Args = append(cmd.Args, "--depends", pkg)
	}
	cmd.Args = append(cmd.Args,
		"--deb-use-file-permissions",
		"--rpm-use-file-permissions",
		"--exclude", "var/lib/arvados/go",
		"/var/lib/arvados",
		"/var/www/.gem",
		"/var/www/.passenger",
		"/var/www/.bundle",
	)
	fmt.Fprintf(stderr, "... %s\n", cmd.Args)
	cmd.Dir = bldr.OutputDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
