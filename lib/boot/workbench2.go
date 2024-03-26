// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net"
	"os"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type runWorkbench2 struct {
	svc arvados.Service
}

func (runner runWorkbench2) String() string {
	return "runWorkbench2"
}

func (runner runWorkbench2) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	host, port, err := internalPort(runner.svc)
	if err != nil {
		return fmt.Errorf("bug: no internalPort for %q: %v (%#v)", runner, err, runner.svc)
	}
	super.waitShutdown.Add(1)
	go func() {
		defer super.waitShutdown.Done()
		if super.ClusterType == "production" {
			err = super.RunProgram(ctx, "/var/lib/arvados/workbench2", runOptions{
				user: "www-data",
			}, "arvados-server", "workbench2", super.cluster.Services.Controller.ExternalURL.Host, net.JoinHostPort(host, port), ".")
		} else {
			// super.SourcePath might be readonly, so for
			// dev/test mode we make a copy in a writable
			// dir.
			livedir := super.wwwtempdir + "/workbench2"
			if err := super.RunProgram(ctx, super.SourcePath+"/services/workbench2", runOptions{}, "rsync", "-a", "--delete-after", super.SourcePath+"/services/workbench2/", livedir); err != nil {
				fail(err)
				return
			}
			if err = os.Mkdir(livedir+"/public/_health", 0777); err != nil && !errors.Is(err, fs.ErrExist) {
				fail(err)
				return
			}
			if err = ioutil.WriteFile(livedir+"/public/_health/ping", []byte(`{"health":"OK"}`), 0666); err != nil {
				fail(err)
				return
			}

			stdinr, stdinw := io.Pipe()
			defer stdinw.Close()
			go func() {
				<-ctx.Done()
				stdinw.Close()
			}()
			err = super.RunProgram(ctx, livedir, runOptions{
				env: []string{
					"CI=true",
					"HTTPS=false",
					"PORT=" + port,
					"REACT_APP_ARVADOS_API_HOST=" + super.cluster.Services.Controller.ExternalURL.Host,
				},
				// If we don't connect stdin, "yarn start" just exits.
				stdin: stdinr,
			}, "yarn", "start")
			fail(errors.New("`yarn start` exited"))
		}
		fail(err)
	}()
	return nil
}
