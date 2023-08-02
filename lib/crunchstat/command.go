// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchstat

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os/exec"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
)

var Command = command{}

type command struct{}

func (command) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet(prog, flag.ExitOnError)
	poll := flags.Duration("poll", 10*time.Second, "reporting interval")
	debug := flags.Bool("debug", false, "show additional debug info")
	dump := flags.String("dump", "", "save snapshot of OS files in given `directory` (for creating test cases)")
	getVersion := flags.Bool("version", false, "print version information and exit")

	if ok, code := cmd.ParseFlags(flags, prog, args, "program [args ...]", stderr); !ok {
		return code
	} else if *getVersion {
		fmt.Printf("%s %s\n", prog, cmd.Version.String())
		return 0
	} else if flags.NArg() == 0 {
		fmt.Fprintf(stderr, "missing required argument: program (try -help)\n")
		return 2
	}

	reporter := &Reporter{
		Logger:     log.New(stderr, prog+": ", 0),
		Debug:      *debug,
		PollPeriod: *poll,
	}
	reporter.Logger.Printf("%s %s", prog, cmd.Version.String())
	reporter.Logger.Printf("running %v", flags.Args())
	cmd := exec.Command(flags.Arg(0), flags.Args()[1:]...)

	// Child process will use our stdin and stdout pipes (we close
	// our copies below)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	// Child process stderr and our stats will both go to stderr
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		reporter.Logger.Printf("error in cmd.Start: %v", err)
		return 1
	}
	reporter.Pid = func() int {
		return cmd.Process.Pid
	}
	reporter.Start()
	defer reporter.Stop()
	if stdin, ok := stdin.(io.Closer); ok {
		stdin.Close()
	}
	if stdout, ok := stdout.(io.Closer); ok {
		stdout.Close()
	}

	failed := false
	if *dump != "" {
		err := reporter.dumpSourceFiles(*dump)
		if err != nil {
			fmt.Fprintf(stderr, "error dumping source files: %s\n", err)
			failed = true
		}
	}

	err := cmd.Wait()

	if err, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although
		// package syscall is generally platform dependent,
		// WaitStatus is defined for both Unix and Windows and
		// in both cases has an ExitStatus() method with the
		// same signature.
		if status, ok := err.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		} else {
			reporter.Logger.Printf("ExitError without WaitStatus: %v", err)
			return 1
		}
	} else if err != nil {
		reporter.Logger.Printf("error running command: %v", err)
		return 1
	}

	if failed {
		return 1
	}
	return 0
}
