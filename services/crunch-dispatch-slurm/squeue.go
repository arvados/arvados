package main

import (
	"bufio"
	"log"
	"os/exec"
	"sync"
	"time"
)

type Squeue struct {
	squeueContents []string
	squeueDone     chan struct{}
	squeueError    error
	squeueCond     *sync.Cond
	SlurmLock      sync.Mutex
}

// squeueFunc
func squeueFunc() *exec.Cmd {
	return exec.Command("squeue", "--format=%j")
}

var squeueCmd = squeueFunc

func (squeue *Squeue) RunSqueue() error {
	var newSqueueContents []string

	// Mutex between squeue sync and running sbatch or scancel.  This
	// establishes a sequence so that squeue doesn't run concurrently with
	// sbatch or scancel; the next update of squeue will occur only after
	// sbatch or scancel has completed.
	squeueUpdater.SlurmLock.Lock()
	defer squeueUpdater.SlurmLock.Unlock()

	// Also ensure unlock on all return paths
	defer squeueUpdater.squeueCond.L.Unlock()

	cmd := squeueCmd()
	sq, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe for squeue: %v", err)
		squeueUpdater.squeueCond.L.Lock()
		squeueUpdater.squeueError = err
		return err
	}
	cmd.Start()
	scanner := bufio.NewScanner(sq)
	for scanner.Scan() {
		newSqueueContents = append(newSqueueContents, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		cmd.Wait()
		log.Printf("Error reading from squeue pipe: %v", err)
		squeueUpdater.squeueCond.L.Lock()
		squeueUpdater.squeueError = err
		return err
	}

	err = cmd.Wait()
	if err != nil {
		log.Printf("Error running squeue: %v", err)
		squeueUpdater.squeueCond.L.Lock()
		squeueUpdater.squeueError = err
		return err
	}

	squeueUpdater.squeueCond.L.Lock()
	squeueUpdater.squeueError = nil
	squeueUpdater.squeueContents = newSqueueContents
	squeueUpdater.squeueCond.Broadcast()

	return nil
}

// Check if a container UUID is in the slurm queue.  This will block until the
// next successful update from SLURM.
func (squeue *Squeue) CheckSqueue(uuid string) (bool, error) {
	squeueUpdater.squeueCond.L.Lock()
	// block until next squeue broadcast signaling an update.
	squeueUpdater.squeueCond.Wait()
	if squeueUpdater.squeueError != nil {
		e := squeueUpdater.squeueError
		squeueUpdater.squeueCond.L.Unlock()
		return false, e
	}
	contents := squeueUpdater.squeueContents
	squeueUpdater.squeueCond.L.Unlock()

	for _, k := range contents {
		if k == uuid {
			return true, nil
		}
	}
	return false, nil
}

func (squeue *Squeue) StartMonitor(pollInterval time.Duration) {
	squeueUpdater.squeueCond = sync.NewCond(&sync.Mutex{})
	squeueUpdater.squeueDone = make(chan struct{})
	squeueUpdater.RunSqueue()
	go squeueUpdater.SyncSqueue(pollInterval)
}

func (squeue *Squeue) Done() {
	squeueUpdater.squeueDone <- struct{}{}
	close(squeueUpdater.squeueDone)
}

func (squeue *Squeue) SyncSqueue(pollInterval time.Duration) {
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-squeueUpdater.squeueDone:
			return
		case <-ticker.C:
			squeueUpdater.RunSqueue()
		}
	}
}
