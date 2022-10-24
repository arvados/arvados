// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/crunchstat"
)

const MaxLogLine = 1 << 14 // Child stderr lines >16KiB will be split

var (
	signalOnDeadPPID  int = 15
	ppidCheckInterval     = time.Second
	version               = "dev"
)

type logger interface {
	Printf(string, ...interface{})
}

func main() {
	reporter := crunchstat.Reporter{
		Logger: log.New(os.Stderr, "crunchstat: ", 0),
	}

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&reporter.CgroupRoot, "cgroup-root", "", "Root of cgroup tree")
	flags.StringVar(&reporter.CgroupParent, "cgroup-parent", "", "Name of container parent under cgroup")
	flags.StringVar(&reporter.CIDFile, "cgroup-cid", "", "Path to container id file")
	flags.IntVar(&signalOnDeadPPID, "signal-on-dead-ppid", signalOnDeadPPID, "Signal to send child if crunchstat's parent process disappears (0 to disable)")
	flags.DurationVar(&ppidCheckInterval, "ppid-check-interval", ppidCheckInterval, "Time between checks for parent process disappearance")
	pollMsec := flags.Int64("poll", 1000, "Reporting interval, in milliseconds")
	getVersion := flags.Bool("version", false, "Print version information and exit.")

	if ok, code := cmd.ParseFlags(flags, os.Args[0], os.Args[1:], "program [args ...]", os.Stderr); !ok {
		os.Exit(code)
	} else if *getVersion {
		fmt.Printf("crunchstat %s\n", version)
		return
	} else if flags.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "missing required argument: program (try -help)\n")
		os.Exit(2)
	}

	reporter.Logger.Printf("crunchstat %s started", version)

	if reporter.CgroupRoot == "" {
		reporter.Logger.Printf("error: must provide -cgroup-root")
		os.Exit(2)
	} else if signalOnDeadPPID < 0 {
		reporter.Logger.Printf("-signal-on-dead-ppid=%d is invalid (use a positive signal number, or 0 to disable)", signalOnDeadPPID)
		os.Exit(2)
	}
	reporter.PollPeriod = time.Duration(*pollMsec) * time.Millisecond

	reporter.Start()
	err := runCommand(flags.Args(), reporter.Logger)
	reporter.Stop()

	if err, ok := err.(*exec.ExitError); ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although
		// package syscall is generally platform dependent,
		// WaitStatus is defined for both Unix and Windows and
		// in both cases has an ExitStatus() method with the
		// same signature.
		if status, ok := err.Sys().(syscall.WaitStatus); ok {
			os.Exit(status.ExitStatus())
		} else {
			reporter.Logger.Printf("ExitError without WaitStatus: %v", err)
			os.Exit(1)
		}
	} else if err != nil {
		reporter.Logger.Printf("error running command: %v", err)
		os.Exit(1)
	}
}

func runCommand(argv []string, logger logger) error {
	cmd := exec.Command(argv[0], argv[1:]...)

	logger.Printf("Running %v", argv)

	// Child process will use our stdin and stdout pipes
	// (we close our copies below)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	// Forward SIGINT and SIGTERM to child process
	sigChan := make(chan os.Signal, 1)
	go func(sig <-chan os.Signal) {
		catch := <-sig
		if cmd.Process != nil {
			cmd.Process.Signal(catch)
		}
		logger.Printf("notice: caught signal: %v", catch)
	}(sigChan)
	signal.Notify(sigChan, syscall.SIGTERM)
	signal.Notify(sigChan, syscall.SIGINT)

	// Kill our child proc if our parent process disappears
	if signalOnDeadPPID != 0 {
		go sendSignalOnDeadPPID(ppidCheckInterval, signalOnDeadPPID, os.Getppid(), cmd, logger)
	}

	// Funnel stderr through our channel
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		logger.Printf("error in StderrPipe: %v", err)
		return err
	}

	// Run subprocess
	if err := cmd.Start(); err != nil {
		logger.Printf("error in cmd.Start: %v", err)
		return err
	}

	// Close stdin/stdout in this (parent) process
	os.Stdin.Close()
	os.Stdout.Close()

	err = copyPipeToChildLog(stderrPipe, log.New(os.Stderr, "", 0))
	if err != nil {
		cmd.Process.Kill()
		return err
	}

	return cmd.Wait()
}

func sendSignalOnDeadPPID(intvl time.Duration, signum, ppidOrig int, cmd *exec.Cmd, logger logger) {
	ticker := time.NewTicker(intvl)
	for range ticker.C {
		ppid := os.Getppid()
		if ppid == ppidOrig {
			continue
		}
		if cmd.Process == nil {
			// Child process isn't running yet
			continue
		}
		logger.Printf("notice: crunchstat ppid changed from %d to %d -- killing child pid %d with signal %d", ppidOrig, ppid, cmd.Process.Pid, signum)
		err := cmd.Process.Signal(syscall.Signal(signum))
		if err != nil {
			logger.Printf("error: sending signal: %s", err)
			continue
		}
		ticker.Stop()
		break
	}
}

func copyPipeToChildLog(in io.ReadCloser, logger logger) error {
	reader := bufio.NewReaderSize(in, MaxLogLine)
	var prefix string
	for {
		line, isPrefix, err := reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("error reading child stderr: %w", err)
		}
		var suffix string
		if isPrefix {
			suffix = "[...]"
		}
		logger.Printf("%s%s%s", prefix, string(line), suffix)
		// Set up prefix for following line
		if isPrefix {
			prefix = "[...]"
		} else {
			prefix = ""
		}
	}
	return in.Close()
}
