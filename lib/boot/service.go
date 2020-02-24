// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"path/filepath"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type runServiceCommand struct {
	name    string
	svc     arvados.Service
	depends []bootTask
}

func (runner runServiceCommand) String() string {
	return runner.name
}

func (runner runServiceCommand) Run(ctx context.Context, fail func(error), boot *Booter) error {
	boot.wait(ctx, runner.depends...)
	go func() {
		var u arvados.URL
		for u = range runner.svc.InternalURLs {
		}
		fail(boot.RunProgram(ctx, boot.tempdir, nil, []string{"ARVADOS_SERVICE_INTERNAL_URL=" + u.String()}, "arvados-server", runner.name, "-config", boot.configfile))
	}()
	return nil
}

type runGoProgram struct {
	src     string
	svc     arvados.Service
	depends []bootTask
}

func (runner runGoProgram) String() string {
	_, basename := filepath.Split(runner.src)
	return basename
}

func (runner runGoProgram) Run(ctx context.Context, fail func(error), boot *Booter) error {
	boot.wait(ctx, runner.depends...)
	bindir := filepath.Join(boot.tempdir, "bin")
	err := boot.RunProgram(ctx, runner.src, nil, []string{"GOBIN=" + bindir}, "go", "install")
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, basename := filepath.Split(runner.src)
	binfile := filepath.Join(bindir, basename)

	if len(runner.svc.InternalURLs) > 0 {
		// Run one for each URL
		for u := range runner.svc.InternalURLs {
			u := u
			boot.waitShutdown.Add(1)
			go func() {
				defer boot.waitShutdown.Done()
				fail(boot.RunProgram(ctx, boot.tempdir, nil, []string{"ARVADOS_SERVICE_INTERNAL_URL=" + u.String()}, binfile))
			}()
		}
	} else {
		// Just run one
		boot.waitShutdown.Add(1)
		go func() {
			defer boot.waitShutdown.Done()
			fail(boot.RunProgram(ctx, boot.tempdir, nil, nil, binfile))
		}()
	}
	return nil
}
