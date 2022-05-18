// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

const slurm15NiceLimit int64 = 10000

type slurmJob struct {
	uuid         string
	wantPriority int64
	priority     int64 // current slurm priority (incorporates nice value)
	nice         int64 // current slurm nice value
	hitNiceLimit bool
}

// SqueueChecker implements asynchronous polling monitor of the SLURM queue
// using the command 'squeue'.
type SqueueChecker struct {
	Logger         logger
	Period         time.Duration
	PrioritySpread int64
	Slurm          Slurm
	queue          map[string]*slurmJob
	startOnce      sync.Once
	done           chan struct{}
	lock           sync.RWMutex
	notify         sync.Cond
}

// HasUUID checks if a given container UUID is in the slurm queue.
// This does not run squeue directly, but instead blocks until woken
// up by next successful update of squeue.
func (sqc *SqueueChecker) HasUUID(uuid string) bool {
	sqc.startOnce.Do(sqc.start)

	sqc.lock.RLock()
	defer sqc.lock.RUnlock()

	// block until next squeue broadcast signaling an update.
	sqc.notify.Wait()
	_, exists := sqc.queue[uuid]
	return exists
}

// SetPriority sets or updates the desired (Arvados) priority for a
// container.
func (sqc *SqueueChecker) SetPriority(uuid string, want int64) {
	sqc.startOnce.Do(sqc.start)

	sqc.lock.RLock()
	job := sqc.queue[uuid]
	if job == nil {
		// Wait in case the slurm job was just submitted and
		// will appear in the next squeue update.
		sqc.notify.Wait()
		job = sqc.queue[uuid]
	}
	needUpdate := job != nil && job.wantPriority != want
	sqc.lock.RUnlock()

	if needUpdate {
		sqc.lock.Lock()
		job.wantPriority = want
		sqc.lock.Unlock()
	}
}

// adjust slurm job nice values as needed to ensure slurm priority
// order matches Arvados priority order.
func (sqc *SqueueChecker) reniceAll() {
	// This is slow (it shells out to scontrol many times) and no
	// other goroutines update sqc.queue or any of the job fields
	// we use here, so we don't acquire a lock.
	jobs := make([]*slurmJob, 0, len(sqc.queue))
	for _, j := range sqc.queue {
		if j.wantPriority == 0 {
			// SLURM job with unknown Arvados priority
			// (perhaps it's not an Arvados job)
			continue
		}
		if j.priority <= 2*slurm15NiceLimit {
			// SLURM <= 15.x implements "hold" by setting
			// priority to 0. If we include held jobs
			// here, we'll end up trying to push other
			// jobs below them using negative priority,
			// which won't help anything.
			continue
		}
		jobs = append(jobs, j)
	}

	sort.Slice(jobs, func(i, j int) bool {
		if jobs[i].wantPriority != jobs[j].wantPriority {
			return jobs[i].wantPriority > jobs[j].wantPriority
		}
		// break ties with container uuid --
		// otherwise, the ordering would change from
		// one interval to the next, and we'd do many
		// pointless slurm queue rearrangements.
		return jobs[i].uuid > jobs[j].uuid
	})
	renice := wantNice(jobs, sqc.PrioritySpread)
	for i, job := range jobs {
		niceNew := renice[i]
		if job.hitNiceLimit && niceNew > slurm15NiceLimit {
			niceNew = slurm15NiceLimit
		}
		if niceNew == job.nice {
			continue
		}
		err := sqc.Slurm.Renice(job.uuid, niceNew)
		if err != nil && niceNew > slurm15NiceLimit && strings.Contains(err.Error(), "Invalid nice value") {
			sqc.Logger.Warnf("container %q clamping nice values at %d, priority order will not be correct -- see https://dev.arvados.org/projects/arvados/wiki/SLURM_integration#Limited-nice-values-SLURM-15", job.uuid, slurm15NiceLimit)
			job.hitNiceLimit = true
		}
	}
}

// Stop stops the squeue monitoring goroutine. Do not call HasUUID
// after calling Stop.
func (sqc *SqueueChecker) Stop() {
	if sqc.done != nil {
		close(sqc.done)
	}
}

// check gets the names of jobs in the SLURM queue (running and
// queued). If it succeeds, it updates sqc.queue and wakes up any
// goroutines that are waiting in HasUUID() or All().
func (sqc *SqueueChecker) check() {
	cmd := sqc.Slurm.QueueCommand([]string{"--all", "--noheader", "--format=%j %y %Q %T %r"})
	stdout, stderr := &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		sqc.Logger.Warnf("Error running %q %q: %s %q", cmd.Path, cmd.Args, err, stderr.String())
		return
	}

	lines := strings.Split(stdout.String(), "\n")
	newq := make(map[string]*slurmJob, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var uuid, state, reason string
		var n, p int64
		if _, err := fmt.Sscan(line, &uuid, &n, &p, &state, &reason); err != nil {
			sqc.Logger.Warnf("ignoring unparsed line in squeue output: %q", line)
			continue
		}

		// No other goroutines write to jobs' priority or nice
		// fields, so we can read and write them without
		// locks.
		replacing, ok := sqc.queue[uuid]
		if !ok {
			replacing = &slurmJob{uuid: uuid}
		}
		replacing.priority = p
		replacing.nice = n
		newq[uuid] = replacing

		if state == "PENDING" && ((reason == "BadConstraints" && p <= 2*slurm15NiceLimit) || reason == "launch failed requeued held") && replacing.wantPriority > 0 {
			// When using SLURM 14.x or 15.x, our queued
			// jobs land in this state when "scontrol
			// reconfigure" invalidates their feature
			// constraints by clearing all node features.
			// They stay in this state even after the
			// features reappear, until we run "scontrol
			// release {jobid}". Priority is usually 0 in
			// this state, but sometimes (due to a race
			// with nice adjustments?) it's a small
			// positive value.
			//
			// "scontrol release" is silent and successful
			// regardless of whether the features have
			// reappeared, so rather than second-guessing
			// whether SLURM is ready, we just keep trying
			// this until it works.
			//
			// "launch failed requeued held" seems to be
			// another manifestation of this problem,
			// resolved the same way.
			sqc.Logger.Printf("releasing held job %q (priority=%d, state=%q, reason=%q)", uuid, p, state, reason)
			sqc.Slurm.Release(uuid)
		} else if state != "RUNNING" && p <= 2*slurm15NiceLimit && replacing.wantPriority > 0 {
			sqc.Logger.Warnf("job %q has low priority %d, nice %d, state %q, reason %q", uuid, p, n, state, reason)
		}
	}
	sqc.lock.Lock()
	sqc.queue = newq
	sqc.lock.Unlock()
	sqc.notify.Broadcast()
}

// Initialize, and start a goroutine to call check() once per
// squeue.Period until terminated by calling Stop().
func (sqc *SqueueChecker) start() {
	sqc.notify.L = sqc.lock.RLocker()
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
				sqc.reniceAll()
				select {
				case <-ticker.C:
					// If this iteration took
					// longer than sqc.Period,
					// consume the next tick and
					// wait. Otherwise we would
					// starve other goroutines.
				default:
				}
			}
		}
	}()
}

// All waits for the next squeue invocation, and returns all job
// names reported by squeue.
func (sqc *SqueueChecker) All() []string {
	sqc.startOnce.Do(sqc.start)
	sqc.lock.RLock()
	defer sqc.lock.RUnlock()
	sqc.notify.Wait()
	var uuids []string
	for u := range sqc.queue {
		uuids = append(uuids, u)
	}
	return uuids
}
