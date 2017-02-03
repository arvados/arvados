package main

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Squeue implements asynchronous polling monitor of the SLURM queue using the
// command 'squeue'.
type SqueueChecker struct {
	Period    time.Duration
	hasUUID   map[string]bool
	startOnce sync.Once
	done      chan struct{}
	sync.Cond
}

func squeueFunc() *exec.Cmd {
	return exec.Command("squeue", "--all", "--format=%j")
}

var squeueCmd = squeueFunc

// HasUUID checks if a given container UUID is in the slurm queue.
// This does not run squeue directly, but instead blocks until woken
// up by next successful update of squeue.
func (sqc *SqueueChecker) HasUUID(uuid string) bool {
	sqc.startOnce.Do(sqc.start)

	sqc.L.Lock()
	defer sqc.L.Unlock()

	// block until next squeue broadcast signaling an update.
	sqc.Wait()
	return sqc.hasUUID[uuid]
}

// Stop stops the squeue monitoring goroutine. Do not call HasUUID
// after calling Stop.
func (sqc *SqueueChecker) Stop() {
	if sqc.done != nil {
		close(sqc.done)
	}
}

// check gets the names of jobs in the SLURM queue (running and
// queued). If it succeeds, it updates squeue.hasUUID and wakes up any
// goroutines that are waiting in HasUUID().
func (sqc *SqueueChecker) check() {
	// Mutex between squeue sync and running sbatch or scancel.  This
	// establishes a sequence so that squeue doesn't run concurrently with
	// sbatch or scancel; the next update of squeue will occur only after
	// sbatch or scancel has completed.
	sqc.L.Lock()
	defer sqc.L.Unlock()

	cmd := squeueCmd()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error running %q %q: %s %q", cmd.Path, cmd.Args, err, stderr.String())
		return
	}

	uuids := strings.Split(stdout.String(), "\n")
	sqc.hasUUID = make(map[string]bool, len(uuids))
	for _, uuid := range uuids {
		sqc.hasUUID[uuid] = true
	}
	sqc.Broadcast()
}

// Initialize, and start a goroutine to call check() once per
// squeue.Period until terminated by calling Stop().
func (sqc *SqueueChecker) start() {
	sqc.L = &sync.Mutex{}
	sqc.done = make(chan struct{})
	go func() {
		ticker := time.NewTicker(sqc.Period)
		for {
			select {
			case <-sqc.done:
				ticker.Stop()
				return
			case <-ticker.C:
				sqc.check()
			}
		}
	}()
}
