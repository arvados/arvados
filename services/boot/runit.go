package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

type runitService struct {
	name string
	cmd  string
	args []string
}

func (r *runitService) Start(ctx context.Context) error {
	if err := installRunit.Boot(ctx); err != nil {
		return err
	}
	svdir := r.svdir(ctx)
	if err := os.MkdirAll(svdir, 0755); err != nil {
		return err
	}
	tmp, err := ioutil.TempFile(svdir, "run~")
	if err != nil {
		return err
	}
	fmt.Fprintf(tmp, "#!/bin/sh\n\nexec %q", r.cmd)
	for _, arg := range r.args {
		fmt.Fprintf(tmp, " %q", arg)
	}
	fmt.Fprintf(tmp, " 2>&1\n")
	tmp.Close()
	if err := os.Chmod(tmp.Name(), 0755); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path.Join(svdir, "run")); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

func (r *runitService) Running(ctx context.Context) (bool, error) {
	if err := installRunit.Boot(ctx); err != nil {
		return false, err
	}
	return runStatusCmd("sv", "stat", r.svdir(ctx))
}

func (r *runitService) svdir(ctx context.Context) string {
	return path.Join(cfg(ctx).RunitSvDir, r.name)
}
