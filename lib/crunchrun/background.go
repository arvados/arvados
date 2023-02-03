// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"encoding/json"
	"fmt"
	"io"
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
	brokenfile = "crunch-run-broken"
	pricesfile = "crunch-run-prices.json"
)

// procinfo is saved in each process's lockfile.
type procinfo struct {
	UUID string
	PID  int
}

// Detach acquires a lock for the given uuid, and starts the current
// program as a child process (with -no-detach prepended to the given
// arguments so the child knows not to detach again). The lock is
// passed along to the child process.
//
// Stdout and stderr in the child process are sent to the systemd
// journal using the systemd-cat program.
func Detach(uuid string, prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return exitcode(stderr, detach(uuid, prog, args, stdin, stdout))
}
func detach(uuid string, prog string, args []string, stdin io.Reader, stdout io.Writer) error {
	lockfile, err := func() (*os.File, error) {
		// We must hold the dir-level lock between
		// opening/creating the lockfile and acquiring LOCK_EX
		// on it, to avoid racing with the ListProcess's
		// alive-checking and garbage collection.
		dirlock, err := lockall()
		if err != nil {
			return nil, err
		}
		defer dirlock.Close()
		lockfilename := filepath.Join(lockdir, lockprefix+uuid+locksuffix)
		lockfile, err := os.OpenFile(lockfilename, os.O_CREATE|os.O_RDWR, 0700)
		if err != nil {
			return nil, fmt.Errorf("open %s: %s", lockfilename, err)
		}
		err = syscall.Flock(int(lockfile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err != nil {
			lockfile.Close()
			return nil, fmt.Errorf("lock %s: %s", lockfilename, err)
		}
		return lockfile, nil
	}()
	if err != nil {
		return err
	}
	defer lockfile.Close()
	lockfile.Truncate(0)

	execargs := append([]string{"-no-detach"}, args...)
	if strings.HasSuffix(prog, " crunch-run") {
		// invoked as "/path/to/arvados-server crunch-run"
		// (see arvados/lib/cmd.Multi)
		execargs = append([]string{strings.TrimSuffix(prog, " crunch-run"), "crunch-run"}, execargs...)
	} else {
		// invoked as "/path/to/crunch-run"
		execargs = append([]string{prog}, execargs...)
	}
	if _, err := exec.LookPath("systemd-cat"); err == nil {
		execargs = append([]string{
			// Here, if the inner systemd-cat can't exec
			// crunch-run, it writes an error message to
			// stderr, and the outer systemd-cat writes it
			// to the journal where the operator has a
			// chance to discover it. (If we only used one
			// systemd-cat command, it would be up to us
			// to report the error -- but we are going to
			// detach and exit, not wait for something to
			// appear on stderr.)  Note these systemd-cat
			// calls don't result in additional processes
			// -- they just connect stderr/stdout to
			// sockets and call exec().
			"systemd-cat", "--identifier=crunch-run",
			"systemd-cat", "--identifier=crunch-run",
		}, execargs...)
	}

	cmd := exec.Command(execargs[0], execargs[1:]...)
	// Child inherits lockfile.
	cmd.ExtraFiles = []*os.File{lockfile}
	// Ensure child isn't interrupted even if we receive signals
	// from parent (sshd) while sending lockfile content to
	// caller.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// We need to manage our own OS pipe here to ensure the child
	// process reads all of our stdin pipe before we return.
	piper, pipew, err := os.Pipe()
	if err != nil {
		return err
	}
	defer pipew.Close()
	cmd.Stdin = piper
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("exec %s: %s", cmd.Path, err)
	}
	_, err = io.Copy(pipew, stdin)
	if err != nil {
		return err
	}
	err = pipew.Close()
	if err != nil {
		return err
	}

	w := io.MultiWriter(stdout, lockfile)
	return json.NewEncoder(w).Encode(procinfo{
		UUID: uuid,
		PID:  cmd.Process.Pid,
	})
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
		return fmt.Errorf("open %s: %s", path, err)
	}
	defer f.Close()

	var pi procinfo
	err = json.NewDecoder(f).Decode(&pi)
	if err != nil {
		return fmt.Errorf("decode %s: %s", path, err)
	}

	if pi.UUID != uuid || pi.PID == 0 {
		return fmt.Errorf("%s: bogus procinfo: %+v", path, pi)
	}

	proc, err := os.FindProcess(pi.PID)
	if err != nil {
		// FindProcess should have succeeded, even if the
		// process does not exist.
		return fmt.Errorf("%s: find process %d: %s", uuid, pi.PID, err)
	}

	// Send the requested signal once, then send signal 0 a few
	// times.  When proc.Signal() returns an error (process no
	// longer exists), return success.  If that doesn't happen
	// within 1 second, return an error.
	err = proc.Signal(signal)
	for deadline := time.Now().Add(time.Second); err == nil && time.Now().Before(deadline); time.Sleep(time.Second / 100) {
		err = proc.Signal(syscall.Signal(0))
	}
	if err == nil {
		// Reached deadline without a proc.Signal() error.
		return fmt.Errorf("%s: pid %d: sent signal %d (%s) but process is still alive", uuid, pi.PID, signal, signal)
	}
	fmt.Fprintf(stderr, "%s: pid %d: %s\n", uuid, pi.PID, err)
	return nil
}

// ListProcesses lists UUIDs of active crunch-run processes.
func ListProcesses(stdin io.Reader, stdout, stderr io.Writer) int {
	if buf, err := io.ReadAll(stdin); err == nil && len(buf) > 0 {
		// write latest pricing data to disk where
		// current/future crunch-run processes can load it
		fnm := filepath.Join(lockdir, pricesfile)
		fnmtmp := fmt.Sprintf("%s~%d", fnm, os.Getpid())
		err := os.WriteFile(fnmtmp, buf, 0777)
		if err != nil {
			fmt.Fprintf(stderr, "error writing price data to %s: %s", fnmtmp, err)
		} else if err = os.Rename(fnmtmp, fnm); err != nil {
			fmt.Fprintf(stderr, "error renaming %s to %s: %s", fnmtmp, fnm, err)
			os.Remove(fnmtmp)
		}
	}
	// filepath.Walk does not follow symlinks, so we must walk
	// lockdir+"/." in case lockdir itself is a symlink.
	walkdir := lockdir + "/."
	return exitcode(stderr, filepath.Walk(walkdir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && path != walkdir {
			return filepath.SkipDir
		}
		if name := info.Name(); name == brokenfile {
			fmt.Fprintln(stdout, "broken")
			return nil
		} else if !strings.HasPrefix(name, lockprefix) || !strings.HasSuffix(name, locksuffix) {
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

		// Ensure other processes don't acquire this lockfile
		// after we have decided it is abandoned but before we
		// have deleted it.
		dirlock, err := lockall()
		if err != nil {
			return err
		}
		err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
		if err == nil {
			// lockfile is stale
			err := os.Remove(path)
			dirlock.Close()
			if err != nil {
				fmt.Fprintf(stderr, "unlink %s: %s\n", f.Name(), err)
			}
			return nil
		}
		dirlock.Close()

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

		proc, err := os.FindProcess(pi.PID)
		if err != nil {
			// FindProcess should have succeeded, even if the
			// process does not exist.
			fmt.Fprintf(stderr, "%s: find process %d: %s", path, pi.PID, err)
			return nil
		}
		err = proc.Signal(syscall.SIGUSR2)
		if err != nil {
			// Process is dead, even though lockfile was
			// still locked. Most likely a stuck arv-mount
			// process that inherited the lock from
			// crunch-run. Report container UUID as
			// "stale".
			fmt.Fprintln(stdout, pi.UUID, "stale")
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

// Acquire a dir-level lock. Must be held while creating or deleting
// container-specific lockfiles, to avoid races during the intervals
// when those container-specific lockfiles are open but not locked.
//
// Caller releases the lock by closing the returned file.
func lockall() (*os.File, error) {
	lockfile := filepath.Join(lockdir, lockprefix+"all"+locksuffix)
	f, err := os.OpenFile(lockfile, os.O_CREATE|os.O_RDWR, 0700)
	if err != nil {
		return nil, fmt.Errorf("open %s: %s", lockfile, err)
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("lock %s: %s", lockfile, err)
	}
	return f, nil
}
