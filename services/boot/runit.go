package main

import (
	"bytes"
	"context"
	"fmt"
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

	script := &bytes.Buffer{}
	fmt.Fprintf(script, "#!/bin/sh\n\nexec %q", r.cmd)
	for _, arg := range r.args {
		fmt.Fprintf(script, " %q", arg)
	}
	fmt.Fprintf(script, " 2>&1\n")

	return atomicWriteFile(path.Join(r.svdir(ctx), "run"), script.Bytes(), 0755)
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
