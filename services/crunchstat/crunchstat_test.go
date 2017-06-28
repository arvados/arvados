// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Test that CopyPipeToChildLog works even on lines longer than
// bufio.MaxScanTokenSize.
func TestCopyPipeToChildLogLongLines(t *testing.T) {
	logger, logBuf := bufLogger()

	pipeIn, pipeOut := io.Pipe()
	copied := make(chan bool)
	go func() {
		copyPipeToChildLog(pipeIn, logger)
		close(copied)
	}()

	sentBytes := make([]byte, bufio.MaxScanTokenSize+MaxLogLine+(1<<22))
	go func() {
		pipeOut.Write([]byte("before\n"))

		for i := range sentBytes {
			// Some bytes that aren't newlines:
			sentBytes[i] = byte((rand.Int() & 0xff) | 0x80)
		}
		sentBytes[len(sentBytes)-1] = '\n'
		pipeOut.Write(sentBytes)

		pipeOut.Write([]byte("after"))
		pipeOut.Close()
	}()

	if before, err := logBuf.ReadBytes('\n'); err != nil || string(before) != "before\n" {
		t.Fatalf("\"before\n\" not received (got \"%s\", %s)", before, err)
	}

	var receivedBytes []byte
	done := false
	for !done {
		line, err := logBuf.ReadBytes('\n')
		if err != nil {
			t.Fatal(err)
		}
		if len(line) >= 5 && string(line[0:5]) == "[...]" {
			if receivedBytes == nil {
				t.Fatal("Beginning of line reported as continuation")
			}
			line = line[5:]
		}
		if len(line) >= 6 && string(line[len(line)-6:]) == "[...]\n" {
			line = line[:len(line)-6]
		} else {
			done = true
		}
		receivedBytes = append(receivedBytes, line...)
	}
	if bytes.Compare(receivedBytes, sentBytes) != 0 {
		t.Fatalf("sent %d bytes, got %d different bytes", len(sentBytes), len(receivedBytes))
	}

	if after, err := logBuf.ReadBytes('\n'); err != nil || string(after) != "after\n" {
		t.Fatalf("\"after\n\" not received (got \"%s\", %s)", after, err)
	}

	select {
	case <-time.After(time.Second):
		t.Fatal("Timeout")
	case <-copied:
		// Done.
	}
}

func bufLogger() (*log.Logger, *bufio.Reader) {
	r, w := io.Pipe()
	logger := log.New(w, "", 0)
	return logger, bufio.NewReader(r)
}

func TestSignalOnDeadPPID(t *testing.T) {
	if !testDeadParent(t, 0) {
		t.Fatal("child should still be alive after parent dies")
	}
	if testDeadParent(t, 15) {
		t.Fatal("child should have been killed when parent died")
	}
}

// testDeadParent returns true if crunchstat's child proc is still
// alive after its parent dies.
func testDeadParent(t *testing.T, signum int) bool {
	var err error
	var bin, childlockfile, parentlockfile *os.File
	for _, f := range []**os.File{&bin, &childlockfile, &parentlockfile} {
		*f, err = ioutil.TempFile("", "crunchstat_")
		if err != nil {
			t.Fatal(err)
		}
		defer (*f).Close()
		defer os.Remove((*f).Name())
	}

	bin.Close()
	err = exec.Command("go", "build", "-o", bin.Name()).Run()
	if err != nil {
		t.Fatal(err)
	}

	err = syscall.Flock(int(parentlockfile.Fd()), syscall.LOCK_EX)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("bash", "-c", `
set -e
"$BINFILE" -cgroup-root=/none -ppid-check-interval=10ms -signal-on-dead-ppid="$SIGNUM" bash -c '
    set -e
    unlock() {
        flock --unlock "$CHILDLOCKFD"
        kill %1
    }
    trap unlock TERM
    flock --exclusive "$CHILDLOCKFD"
    echo -n "$$" > "$CHILDLOCKFILE"
    flock --unlock "$PARENTLOCKFD"
    sleep 20 </dev/null >/dev/null 2>/dev/null &
    wait %1
    unlock
' &

# wait for inner bash to start, to ensure $BINFILE has seen this bash proc as its initial PPID
flock --exclusive "$PARENTLOCKFILE" true
`)
	cmd.Env = append(os.Environ(),
		"SIGNUM="+fmt.Sprintf("%d", signum),
		"PARENTLOCKFD=3",
		"PARENTLOCKFILE="+parentlockfile.Name(),
		"CHILDLOCKFD=4",
		"CHILDLOCKFILE="+childlockfile.Name(),
		"BINFILE="+bin.Name())
	cmd.ExtraFiles = []*os.File{parentlockfile, childlockfile}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Start()
	defer cmd.Wait()

	var wg sync.WaitGroup
	wg.Add(2)
	defer wg.Wait()
	for _, rdr := range []io.ReadCloser{stderr, stdout} {
		go func(rdr io.ReadCloser) {
			defer wg.Done()
			buf := make([]byte, 1024)
			for {
				n, err := rdr.Read(buf)
				if n > 0 {
					t.Logf("%s", buf[:n])
				}
				if err != nil {
					return
				}
			}
		}(rdr)
	}

	// Wait until inner bash process releases parentlockfile
	// (which means it has locked childlockfile and written its
	// PID)
	err = exec.Command("flock", "--exclusive", parentlockfile.Name(), "true").Run()
	if err != nil {
		t.Fatal(err)
	}

	childDone := make(chan bool)
	go func() {
		// Notify the main thread when the inner bash process
		// releases its lock on childlockfile (which means
		// either its sleep process ended or it received a
		// TERM signal).
		t0 := time.Now()
		err = exec.Command("flock", "--exclusive", childlockfile.Name(), "true").Run()
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("child done after %s", time.Since(t0))
		close(childDone)
	}()

	select {
	case <-time.After(500 * time.Millisecond):
		// Inner bash process is still alive after the timeout
		// period. Kill it now, so our stdout and stderr pipes
		// can finish and we don't leave a mess of child procs
		// behind.
		buf, err := ioutil.ReadFile(childlockfile.Name())
		if err != nil {
			t.Fatal(err)
		}
		var childPID int
		_, err = fmt.Sscanf(string(buf), "%d", &childPID)
		if err != nil {
			t.Fatal(err)
		}
		child, err := os.FindProcess(childPID)
		if err != nil {
			t.Fatal(err)
		}
		child.Signal(syscall.Signal(15))
		return true

	case <-childDone:
		// Inner bash process ended soon after its grandparent
		// ended.
		return false
	}
}
