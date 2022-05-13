// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

const defaultSpread int64 = 10

// wantNice calculates appropriate nice values for a set of SLURM
// jobs. The returned slice will have len(jobs) elements.
//
// spread is a positive amount of space to leave between adjacent
// priorities when making adjustments. Generally, increasing spread
// reduces the total number of adjustments made. A smaller spread
// produces lower nice values, which is useful for old SLURM versions
// with a limited "nice" range and for sites where SLURM is also
// running non-Arvados jobs with low nice values.
//
// If spread<1, a sensible default (10) is used.
func wantNice(jobs []*slurmJob, spread int64) []int64 {
	if len(jobs) == 0 {
		return nil
	}

	if spread < 1 {
		spread = defaultSpread
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
		} else if space := target - job.priority; space >= 0 && space < (spread-1)*10 {
			// Ordering is correct, and interval isn't too
			// large. Leave existing nice value alone.
			renice[i] = job.nice
			target = job.priority
		} else {
			target -= (spread - 1)
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
