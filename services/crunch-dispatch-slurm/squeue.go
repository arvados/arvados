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
type Squeue struct {
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
func (squeue *Squeue) HasUUID(uuid string) bool {
	squeue.startOnce.Do(squeue.start)

	squeue.L.Lock()
	defer squeue.L.Unlock()

	// block until next squeue broadcast signaling an update.
	squeue.Wait()
	return squeue.hasUUID[uuid]
}

// Stop stops the squeue monitoring goroutine. Do not call HasUUID
// after calling Stop.
func (squeue *Squeue) Stop() {
	if squeue.done != nil {
		close(squeue.done)
	}
}

// check gets the names of jobs in the SLURM queue (running and
// queued). If it succeeds, it updates squeue.hasUUID and wakes up any
// goroutines that are waiting in HasUUID().
func (squeue *Squeue) check() {
	// Mutex between squeue sync and running sbatch or scancel.  This
	// establishes a sequence so that squeue doesn't run concurrently with
	// sbatch or scancel; the next update of squeue will occur only after
	// sbatch or scancel has completed.
	squeue.L.Lock()
	defer squeue.L.Unlock()

	cmd := squeueCmd()
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		log.Printf("Error running %q %q: %s %q", cmd.Path, cmd.Args, err, stderr.String())
		return
	}

	uuids := strings.Split(stdout.String(), "\n")
	squeue.hasUUID = make(map[string]bool, len(uuids))
	for _, uuid := range uuids {
		squeue.hasUUID[uuid] = true
	}
	squeue.Broadcast()
}

// Initialize, and start a goroutine to call check() once per
// squeue.Period until terminated by calling Stop().
func (squeue *Squeue) start() {
	squeue.L = &sync.Mutex{}
	squeue.done = make(chan struct{})
	go func() {
		ticker := time.NewTicker(squeue.Period)
		for {
			select {
			case <-squeue.done:
				ticker.Stop()
				return
			case <-ticker.C:
				squeue.check()
			}
		}
	}()
}
