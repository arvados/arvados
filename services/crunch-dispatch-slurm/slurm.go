// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
)

type Slurm interface {
	Cancel(name string) error
	Renice(name string, nice int) error
	QueueCommand(args []string) *exec.Cmd
	Batch(script io.Reader, args []string) error
}

type slurmCLI struct{}

func (scli *slurmCLI) Batch(script io.Reader, args []string) error {
	return scli.run(script, "sbatch", args)
}

func (scli *slurmCLI) Cancel(name string) error {
	return scli.run(nil, "scancel", []string{"--name=" + name})
}

func (scli *slurmCLI) QueueCommand(args []string) *exec.Cmd {
	return exec.Command("squeue", args...)
}

func (scli *slurmCLI) Renice(name string, nice int) error {
	return scli.run(nil, "scontrol", []string{"update", "JobName=" + name, fmt.Sprintf("Nice=%d", nice)})
}

func (scli *slurmCLI) run(stdin io.Reader, prog string, args []string) error {
	cmd := exec.Command(prog, args...)
	cmd.Stdin = stdin
	out, err := cmd.CombinedOutput()
	outTrim := strings.TrimSpace(string(out))
	if err != nil || len(out) > 0 {
		log.Printf("%q %q: %q", cmd.Path, cmd.Args, outTrim)
	}
	if err != nil {
		err = fmt.Errorf("%s: %s (%q)", cmd.Path, err, outTrim)
	}
	return err
}
