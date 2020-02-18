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

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type installPassenger struct {
	src     string
	depends []bootTask
}

func (runner installPassenger) String() string {
	return "installPassenger:" + runner.src
}

func (runner installPassenger) Run(ctx context.Context, fail func(error), boot *Booter) error {
	err := boot.wait(ctx, runner.depends...)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = boot.RunProgram(ctx, runner.src, &buf, nil, "gem", "list", "--details", "bundler")
	if err != nil {
		return err
	}
	for _, version := range []string{"1.11.0", "1.17.3", "2.0.2"} {
		if !strings.Contains(buf.String(), "("+version+")") {
			err = boot.RunProgram(ctx, runner.src, nil, nil, "gem", "install", "--user", "bundler:1.11", "bundler:1.17.3", "bundler:2.0.2")
			if err != nil {
				return err
			}
			break
		}
	}
	err = boot.RunProgram(ctx, runner.src, nil, nil, "bundle", "install", "--jobs", "4", "--path", filepath.Join(os.Getenv("HOME"), ".gem"))
	if err != nil {
		return err
	}
	err = boot.RunProgram(ctx, runner.src, nil, nil, "bundle", "exec", "passenger-config", "build-native-support")
	if err != nil {
		return err
	}
	err = boot.RunProgram(ctx, runner.src, nil, nil, "bundle", "exec", "passenger-config", "install-standalone-runtime")
	if err != nil {
		return err
	}
	err = boot.RunProgram(ctx, runner.src, nil, nil, "bundle", "exec", "passenger-config", "validate-install")
	if err != nil {
		return err
	}
	return nil
}

type runPassenger struct {
	src     string
	svc     arvados.Service
	depends []bootTask
}

func (runner runPassenger) String() string {
	return "runPassenger:" + runner.src
}

func (runner runPassenger) Run(ctx context.Context, fail func(error), boot *Booter) error {
	err := boot.wait(ctx, runner.depends...)
	if err != nil {
		return err
	}
	port, err := internalPort(runner.svc)
	if err != nil {
		return fmt.Errorf("bug: no InternalURLs for component %q: %v", runner, runner.svc.InternalURLs)
	}
	go func() {
		err = boot.RunProgram(ctx, runner.src, nil, nil, "bundle", "exec", "passenger", "start", "-p", port)
		fail(err)
	}()
	return nil
}
