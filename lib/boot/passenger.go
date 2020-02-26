// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// Don't trust "passenger-config" (or "bundle install") to handle
// concurrent installs.
var passengerInstallMutex sync.Mutex

// Install a Rails application's dependencies, including phusion
// passenger.
type installPassenger struct {
	src     string
	depends []supervisedTask
}

func (runner installPassenger) String() string {
	return "installPassenger:" + runner.src
}

func (runner installPassenger) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	err := super.wait(ctx, runner.depends...)
	if err != nil {
		return err
	}

	passengerInstallMutex.Lock()
	defer passengerInstallMutex.Unlock()

	var buf bytes.Buffer
	err = super.RunProgram(ctx, runner.src, &buf, nil, "gem", "list", "--details", "bundler")
	if err != nil {
		return err
	}
	for _, version := range []string{"1.11.0", "1.17.3", "2.0.2"} {
		if !strings.Contains(buf.String(), "("+version+")") {
			err = super.RunProgram(ctx, runner.src, nil, nil, "gem", "install", "--user", "bundler:1.11", "bundler:1.17.3", "bundler:2.0.2")
			if err != nil {
				return err
			}
			break
		}
	}
	err = super.RunProgram(ctx, runner.src, nil, nil, "bundle", "install", "--jobs", "4", "--path", filepath.Join(os.Getenv("HOME"), ".gem"))
	if err != nil {
		return err
	}
	err = super.RunProgram(ctx, runner.src, nil, nil, "bundle", "exec", "passenger-config", "build-native-support")
	if err != nil {
		return err
	}
	err = super.RunProgram(ctx, runner.src, nil, nil, "bundle", "exec", "passenger-config", "install-standalone-runtime")
	if err != nil {
		return err
	}
	err = super.RunProgram(ctx, runner.src, nil, nil, "bundle", "exec", "passenger-config", "validate-install")
	if err != nil {
		return err
	}
	return nil
}

type runPassenger struct {
	src     string
	svc     arvados.Service
	depends []supervisedTask
}

func (runner runPassenger) String() string {
	return "runPassenger:" + runner.src
}

func (runner runPassenger) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	err := super.wait(ctx, runner.depends...)
	if err != nil {
		return err
	}
	port, err := internalPort(runner.svc)
	if err != nil {
		return fmt.Errorf("bug: no internalPort for %q: %v (%#v)", runner, err, runner.svc)
	}
	loglevel := "4"
	if lvl, ok := map[string]string{
		"debug":   "5",
		"info":    "4",
		"warn":    "2",
		"warning": "2",
		"error":   "1",
		"fatal":   "0",
		"panic":   "0",
	}[super.cluster.SystemLogs.LogLevel]; ok {
		loglevel = lvl
	}
	super.waitShutdown.Add(1)
	go func() {
		defer super.waitShutdown.Done()
		err = super.RunProgram(ctx, runner.src, nil, nil, "bundle", "exec",
			"passenger", "start",
			"-p", port,
			"--log-file", "/dev/stderr",
			"--log-level", loglevel,
			"--no-friendly-error-pages",
			"--pid-file", filepath.Join(super.tempdir, "passenger."+strings.Replace(runner.src, "/", "_", -1)+".pid"))
		fail(err)
	}()
	return nil
}
