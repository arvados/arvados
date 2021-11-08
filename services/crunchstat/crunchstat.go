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

	"git.arvados.org/arvados.git/lib/crunchstat"
)

const MaxLogLine = 1 << 14 // Child stderr lines >16KiB will be split

var (
	signalOnDeadPPID  int = 15
	ppidCheckInterval     = time.Second
	version               = "dev"
)

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

	err := flags.Parse(os.Args[1:])
	if err == flag.ErrHelp {
		return
	} else if err != nil {
		reporter.Logger.Print(err)
		os.Exit(2)
	}

	// Print version information if requested
	if *getVersion {
		fmt.Printf("crunchstat %s\n", version)
		return
	}

	if flags.NArg() == 0 {
		fmt.Fprintf(flags.Output(), "Usage: %s [options] program [args...]\n\nOptions:\n", os.Args[0])
		flags.PrintDefaults()
		os.Exit(2)
	}

	reporter.Logger.Printf("crunchstat %s started", version)

	if reporter.CgroupRoot == "" {
		reporter.Logger.Fatal("error: must provide -cgroup-root")
	} else if signalOnDeadPPID < 0 {
		reporter.Logger.Fatalf("-signal-on-dead-ppid=%d is invalid (use a positive signal number, or 0 to disable)", signalOnDeadPPID)
	}
	reporter.PollPeriod = time.Duration(*pollMsec) * time.Millisecond

	reporter.Start()
	err = runCommand(flags.Args(), reporter.Logger)
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
			reporter.Logger.Fatalln("ExitError without WaitStatus:", err)
		}
	} else if err != nil {
		reporter.Logger.Fatalln("error in cmd.Wait:", err)
	}
}

func runCommand(argv []string, logger *log.Logger) error {
	cmd := exec.Command(argv[0], argv[1:]...)

	logger.Println("Running", argv)

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
		logger.Println("notice: caught signal:", catch)
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
		logger.Fatalln("error in StderrPipe:", err)
	}

	// Run subprocess
	if err := cmd.Start(); err != nil {
		logger.Fatalln("error in cmd.Start:", err)
	}

	// Close stdin/stdout in this (parent) process
	os.Stdin.Close()
	os.Stdout.Close()

	copyPipeToChildLog(stderrPipe, log.New(os.Stderr, "", 0))

	return cmd.Wait()
}

func sendSignalOnDeadPPID(intvl time.Duration, signum, ppidOrig int, cmd *exec.Cmd, logger *log.Logger) {
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

func copyPipeToChildLog(in io.ReadCloser, logger *log.Logger) {
	reader := bufio.NewReaderSize(in, MaxLogLine)
	var prefix string
	for {
		line, isPrefix, err := reader.ReadLine()
		if err == io.EOF {
			break
		} else if err != nil {
			logger.Fatal("error reading child stderr:", err)
		}
		var suffix string
		if isPrefix {
			suffix = "[...]"
		}
		logger.Print(prefix, string(line), suffix)
		// Set up prefix for following line
		if isPrefix {
			prefix = "[...]"
		} else {
			prefix = ""
		}
	}
	in.Close()
}
