// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"errors"
	"path/filepath"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// Run a service using the arvados-server binary.
//
// In future this will bring up the service in the current process,
// but for now (at least until the subcommand handlers get a shutdown
// mechanism) it starts a child process using the arvados-server
// binary, which the supervisor is assumed to have installed in
// {super.tempdir}/bin/.
type runServiceCommand struct {
	name    string           // arvados-server subcommand, e.g., "controller"
	svc     arvados.Service  // cluster.Services.* entry with the desired InternalURLs
	depends []supervisedTask // wait for these tasks before starting
}

func (runner runServiceCommand) String() string {
	return runner.name
}

func (runner runServiceCommand) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	binfile := filepath.Join(super.bindir, "arvados-server")
	err := super.RunProgram(ctx, super.bindir, runOptions{}, binfile, "-version")
	if err != nil {
		return err
	}
	super.wait(ctx, createCertificates{})
	super.wait(ctx, runner.depends...)
	for u := range runner.svc.InternalURLs {
		u := u
		if islocal, err := addrIsLocal(u.Host); err != nil {
			return err
		} else if !islocal {
			continue
		}
		super.waitShutdown.Add(1)
		go func() {
			defer super.waitShutdown.Done()
			fail(super.RunProgram(ctx, super.tempdir, runOptions{
				env: []string{
					"ARVADOS_SERVICE_INTERNAL_URL=" + u.String(),
					// Child process should not
					// try to tell systemd that we
					// are ready.
					"NOTIFY_SOCKET=",
				},
			}, binfile, runner.name, "-config", super.configfile))
		}()
	}
	return nil
}

// Run a Go service that isn't bundled in arvados-server.
type runGoProgram struct {
	src     string           // source dir, e.g., "services/keepproxy"
	svc     arvados.Service  // cluster.Services.* entry with the desired InternalURLs
	depends []supervisedTask // wait for these tasks before starting
}

func (runner runGoProgram) String() string {
	_, basename := filepath.Split(runner.src)
	return basename
}

func (runner runGoProgram) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	if len(runner.svc.InternalURLs) == 0 {
		return errors.New("bug: runGoProgram needs non-empty svc.InternalURLs")
	}

	binfile, err := super.installGoProgram(ctx, runner.src)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	err = super.RunProgram(ctx, super.tempdir, runOptions{}, binfile, "-version")
	if err != nil {
		return err
	}

	super.wait(ctx, createCertificates{})
	super.wait(ctx, runner.depends...)
	for u := range runner.svc.InternalURLs {
		u := u
		if islocal, err := addrIsLocal(u.Host); err != nil {
			return err
		} else if !islocal {
			continue
		}
		super.waitShutdown.Add(1)
		go func() {
			defer super.waitShutdown.Done()
			fail(super.RunProgram(ctx, super.tempdir, runOptions{env: []string{"ARVADOS_SERVICE_INTERNAL_URL=" + u.String()}}, binfile))
		}()
	}
	return nil
}
