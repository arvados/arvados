// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

import (
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
)

type Slurm interface {
	Batch(script io.Reader, args []string) error
	Cancel(name string) error
	QueueCommand(args []string) *exec.Cmd
	Release(name string) error
	Renice(name string, nice int64) error
}

type slurmCLI struct {
	runSemaphore chan bool
}

func NewSlurmCLI() *slurmCLI {
	return &slurmCLI{
		runSemaphore: make(chan bool, 3),
	}
}

func (scli *slurmCLI) Batch(script io.Reader, args []string) error {
	return scli.run(script, "sbatch", args)
}

func (scli *slurmCLI) Cancel(name string) error {
	for _, args := range [][]string{
		// If the slurm job hasn't started yet, remove it from
		// the queue.
		{"--state=pending"},
		// If the slurm job has started, send SIGTERM. If we
		// cancel a running job without a --signal argument,
		// slurm will send SIGTERM and then (after some
		// site-configured interval) SIGKILL. This would kill
		// crunch-run without stopping the container, which we
		// don't want.
		{"--batch", "--signal=TERM", "--state=running"},
		{"--batch", "--signal=TERM", "--state=suspended"},
	} {
		err := scli.run(nil, "scancel", append([]string{"--name=" + name}, args...))
		if err != nil {
			// scancel exits 0 if no job matches the given
			// name and state. Any error from scancel here
			// really indicates something is wrong.
			return err
		}
	}
	return nil
}

func (scli *slurmCLI) QueueCommand(args []string) *exec.Cmd {
	return exec.Command("squeue", args...)
}

func (scli *slurmCLI) Release(name string) error {
	return scli.run(nil, "scontrol", []string{"release", "Name=" + name})
}

func (scli *slurmCLI) Renice(name string, nice int64) error {
	return scli.run(nil, "scontrol", []string{"update", "JobName=" + name, fmt.Sprintf("Nice=%d", nice)})
}

func (scli *slurmCLI) run(stdin io.Reader, prog string, args []string) error {
	scli.runSemaphore <- true
	defer func() { <-scli.runSemaphore }()
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
