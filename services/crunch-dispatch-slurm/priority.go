// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import "git.curoverse.com/arvados.git/sdk/go/arvados"

type slurmJob struct {
	ctr      *arvados.Container
	priority int64 // current slurm priority (incorporates nice value)
	nice     int64 // current slurm nice value
}

// wantNice calculates appropriate nice values for a set of SLURM
// jobs. The returned slice will have len(jobs) elements.
//
// spread is a non-negative amount of space to leave between adjacent
// priorities when making adjustments. Generally, increasing spread
// reduces the total number of adjustments made. A smaller spread
// produces lower nice values, which is useful for old SLURM versions
// with a limited "nice" range and for sites where SLURM is also
// running non-Arvados jobs with low nice values.
func wantNice(jobs []slurmJob, spread int64) []int64 {
	if len(jobs) == 0 {
		return nil
	}
	renice := make([]int64, len(jobs))

	// highest usable priority (without going out of order)
	var target int64
	for i, job := range jobs {
		if i == 0 {
			// renice[0] is always zero, so our highest
			// priority container gets the highest
			// possible slurm priority.
			target = job.priority + job.nice
		} else if space := target - job.priority; space >= 0 && space < spread*10 {
			// Ordering is correct, and interval isn't too
			// large. Leave existing nice value alone.
			renice[i] = job.nice
			target = job.priority
		} else {
			target -= spread
			if possible := job.priority + job.nice; target > possible {
				// renice[i] is already 0, that's the
				// best we can do
				target = possible
			} else {
				renice[i] = possible - target
			}
		}
		target--
	}
	return renice
}
