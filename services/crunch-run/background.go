// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	lockdir    = "/var/lock"
	lockprefix = "crunch-run-"
	locksuffix = ".lock"
)

// procinfo is saved in each process's lockfile.
type procinfo struct {
	UUID   string
	PID    int
	Stdout string
	Stderr string
}

// Detach acquires a lock for the given uuid, and starts the current
// program as a child process (with -detached prepended to the given
// arguments so the child knows not to detach again). The lock is
// passed along to the child process.
func Detach(uuid string, args []string, stdout, stderr io.Writer) int {
	return exitcode(stderr, detach(uuid, args, stdout, stderr))
}
func detach(uuid string, args []string, stdout, stderr io.Writer) error {
	lockfile, err := os.OpenFile(filepath.Join(lockdir, lockprefix+uuid+locksuffix), os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return err
	}
	defer lockfile.Close()
	err = syscall.Flock(int(lockfile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return err
	}
	lockfile.Truncate(0)

	outfile, err := ioutil.TempFile("", "crunch-run-"+uuid+"-stdout-")
	if err != nil {
		return err
	}
	defer outfile.Close()
	errfile, err := ioutil.TempFile("", "crunch-run-"+uuid+"-stderr-")
	if err != nil {
		os.Remove(outfile.Name())
		return err
	}
	defer errfile.Close()

	cmd := exec.Command(args[0], append([]string{"-detached"}, args[1:]...)...)
	cmd.Stdout = outfile
	cmd.Stderr = errfile
	// Child inherits lockfile.
	cmd.ExtraFiles = []*os.File{lockfile}
	// Ensure child isn't interrupted even if we receive signals
	// from parent (sshd) while sending lockfile content to
	// caller.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err = cmd.Start()
	if err != nil {
		os.Remove(outfile.Name())
		os.Remove(errfile.Name())
		return err
	}

	w := io.MultiWriter(stdout, lockfile)
	err = json.NewEncoder(w).Encode(procinfo{
		PID:    cmd.Process.Pid,
		Stdout: outfile.Name(),
		Stderr: errfile.Name(),
	})
	if err != nil {
		os.Remove(outfile.Name())
		os.Remove(errfile.Name())
		return err
	}
	return nil
}

// KillProcess finds the crunch-run process corresponding to the given
// uuid, and sends the given signal to it. It then waits up to 1
// second for the process to die. It returns 0 if the process is
// successfully killed or didn't exist in the first place.
func KillProcess(uuid string, signal syscall.Signal, stdout, stderr io.Writer) int {
	return exitcode(stderr, kill(uuid, signal, stdout, stderr))
}

func kill(uuid string, signal syscall.Signal, stdout, stderr io.Writer) error {
	path := filepath.Join(lockdir, lockprefix+uuid+locksuffix)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	defer f.Close()

	var pi procinfo
	err = json.NewDecoder(f).Decode(&pi)
	if err != nil {
		return fmt.Errorf("%s: %s\n", path, err)
	}

	if pi.UUID != uuid || pi.PID == 0 {
		return fmt.Errorf("%s: bogus procinfo: %+v", path, pi)
	}

	proc, err := os.FindProcess(pi.PID)
	if err != nil {
		return err
	}

	err = proc.Signal(signal)
	for deadline := time.Now().Add(time.Second); err == nil && time.Now().Before(deadline); time.Sleep(time.Second / 100) {
		err = proc.Signal(syscall.Signal(0))
	}
	if err == nil {
		return fmt.Errorf("pid %d: sent signal %d (%s) but process is still alive", pi.PID, signal, signal)
	}
	fmt.Fprintf(stderr, "pid %d: %s\n", pi.PID, err)
	return nil
}

// List UUIDs of active crunch-run processes.
func ListProcesses(stdout, stderr io.Writer) int {
	return exitcode(stderr, filepath.Walk(lockdir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return filepath.SkipDir
		}
		if name := info.Name(); !strings.HasPrefix(name, lockprefix) || !strings.HasSuffix(name, locksuffix) {
			return nil
		}
		if info.Size() == 0 {
			// race: process has opened/locked but hasn't yet written pid/uuid
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		// TODO: Do this check without risk of disrupting lock
		// acquisition during races, e.g., by connecting to a
		// unix socket or checking /proc/$pid/fd/$n ->
		// lockfile.
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
		if err == nil {
			// lockfile is stale
			err := os.Remove(path)
			if err != nil {
				fmt.Fprintln(stderr, err)
			}
			return nil
		}

		var pi procinfo
		err = json.NewDecoder(f).Decode(&pi)
		if err != nil {
			fmt.Fprintf(stderr, "%s: %s\n", path, err)
			return nil
		}
		if pi.UUID == "" || pi.PID == 0 {
			fmt.Fprintf(stderr, "%s: bogus procinfo: %+v", path, pi)
			return nil
		}

		fmt.Fprintln(stdout, pi.UUID)
		return nil
	}))
}

// If err is nil, return 0 ("success"); otherwise, print err to stderr
// and return 1.
func exitcode(stderr io.Writer, err error) int {
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
