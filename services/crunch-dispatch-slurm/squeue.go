package main

import (
	"bufio"
	"log"
	"os/exec"
	"sync"
	"time"
)

// Squeue implements asynchronous polling monitor of the SLURM queue using the
// command 'squeue'.
type Squeue struct {
	squeueContents []string
	squeueDone     chan struct{}
	squeueCond     *sync.Cond
	SlurmLock      sync.Mutex
}

// squeueFunc
func squeueFunc() *exec.Cmd {
	return exec.Command("squeue", "--all", "--format=%j")
}

var squeueCmd = squeueFunc

// RunSqueue runs squeue once and captures the output.  If it succeeds, set
// "squeueContents" and then wake up any goroutines waiting squeueCond in
// CheckSqueue().  If there was an error, log it and leave the threads blocked.
func (squeue *Squeue) RunSqueue() {
	var newSqueueContents []string

	// Mutex between squeue sync and running sbatch or scancel.  This
	// establishes a sequence so that squeue doesn't run concurrently with
	// sbatch or scancel; the next update of squeue will occur only after
	// sbatch or scancel has completed.
	squeue.SlurmLock.Lock()
	defer squeue.SlurmLock.Unlock()

	// Also ensure unlock on all return paths

	cmd := squeueCmd()
	sq, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe for squeue: %v", err)
		return
	}
	err = cmd.Start()
	if err != nil {
		log.Printf("Error running squeue: %v", err)
		return
	}
	scanner := bufio.NewScanner(sq)
	for scanner.Scan() {
		newSqueueContents = append(newSqueueContents, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		cmd.Wait()
		log.Printf("Error reading from squeue pipe: %v", err)
		return
	}

	err = cmd.Wait()
	if err != nil {
		log.Printf("Error running squeue: %v", err)
		return
	}

	squeue.squeueCond.L.Lock()
	squeue.squeueContents = newSqueueContents
	squeue.squeueCond.Broadcast()
	squeue.squeueCond.L.Unlock()
}

// CheckSqueue checks if a given container UUID is in the slurm queue.  This
// does not run squeue directly, but instead blocks until woken up by next
// successful update of squeue.
func (squeue *Squeue) CheckSqueue(uuid string) bool {
	squeue.squeueCond.L.Lock()
	// block until next squeue broadcast signaling an update.
	squeue.squeueCond.Wait()
	contents := squeue.squeueContents
	squeue.squeueCond.L.Unlock()

	for _, k := range contents {
		if k == uuid {
			return true
		}
	}
	return false
}

// StartMonitor starts the squeue monitoring goroutine.
func (squeue *Squeue) StartMonitor(pollInterval time.Duration) {
	squeue.squeueCond = sync.NewCond(&sync.Mutex{})
	squeue.squeueDone = make(chan struct{})
	go squeue.SyncSqueue(pollInterval)
}

// Done stops the squeue monitoring goroutine.
func (squeue *Squeue) Done() {
	squeue.squeueDone <- struct{}{}
	close(squeue.squeueDone)
}

// SyncSqueue periodically polls RunSqueue() at the given duration until
// terminated by calling Done().
func (squeue *Squeue) SyncSqueue(pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-squeue.squeueDone:
			return
		case <-ticker.C:
			squeue.RunSqueue()
		}
	}
}
