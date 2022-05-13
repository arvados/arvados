// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

import (
	. "gopkg.in/check.v1"
)

var _ = Suite(&PrioritySuite{})

type PrioritySuite struct{}

func (s *PrioritySuite) TestReniceCorrect(c *C) {
	for _, test := range []struct {
		spread int64
		in     []*slurmJob
		out    []int64
	}{
		{
			0,
			nil,
			nil,
		},
		{
			0,
			[]*slurmJob{},
			nil,
		},
		{
			10,
			[]*slurmJob{{priority: 4294000111, nice: 10000}},
			[]int64{0},
		},
		{
			10,
			[]*slurmJob{
				{priority: 4294000111, nice: 10000},
				{priority: 4294000111, nice: 10000},
				{priority: 4294000111, nice: 10000},
				{priority: 4294000111, nice: 10000},
			},
			[]int64{0, 10, 20, 30},
		},
		{ // smaller spread than necessary, but correctly ordered => leave nice alone
			10,
			[]*slurmJob{
				{priority: 4294000113, nice: 0},
				{priority: 4294000112, nice: 1},
				{priority: 4294000111, nice: 99},
			},
			[]int64{0, 1, 99},
		},
		{ // larger spread than necessary, but less than 10x => leave nice alone
			10,
			[]*slurmJob{
				{priority: 4294000144, nice: 0},
				{priority: 4294000122, nice: 20},
				{priority: 4294000111, nice: 30},
			},
			[]int64{0, 20, 30},
		},
		{ // > 10x spread => reduce nice to achieve spread=10
			10,
			[]*slurmJob{
				{priority: 4000, nice: 0},    // max pri 4000
				{priority: 3000, nice: 999},  // max pri 3999
				{priority: 2000, nice: 1998}, // max pri 3998
			},
			[]int64{0, 9, 18},
		},
		{ // > 10x spread, but spread=10 is impossible without negative nice
			10,
			[]*slurmJob{
				{priority: 4000, nice: 0},    // max pri 4000
				{priority: 3000, nice: 500},  // max pri 3500
				{priority: 2000, nice: 2000}, // max pri 4000
			},
			[]int64{0, 0, 510},
		},
		{ // default spread, needs reorder
			0,
			[]*slurmJob{
				{priority: 4000, nice: 0}, // max pri 4000
				{priority: 5000, nice: 0}, // max pri 5000
				{priority: 6000, nice: 0}, // max pri 6000
			},
			[]int64{0, 1000 + defaultSpread, 2000 + defaultSpread*2},
		},
		{ // minimum spread
			1,
			[]*slurmJob{
				{priority: 4000, nice: 0}, // max pri 4000
				{priority: 5000, nice: 0}, // max pri 5000
				{priority: 6000, nice: 0}, // max pri 6000
				{priority: 3000, nice: 0}, // max pri 3000
			},
			[]int64{0, 1001, 2002, 0},
		},
	} {
		c.Logf("spread=%d %+v -> %+v", test.spread, test.in, test.out)
		c.Check(wantNice(test.in, test.spread), DeepEquals, test.out)

		if len(test.in) == 0 {
			continue
		}
		// After making the adjustments, calling wantNice
		// again should return the same recommendations.
		updated := make([]*slurmJob, len(test.in))
		for i, in := range test.in {
			updated[i] = &slurmJob{
				nice:     test.out[i],
				priority: in.priority + in.nice - test.out[i],
			}
		}
		c.Check(wantNice(updated, test.spread), DeepEquals, test.out)
	}
}

func (s *PrioritySuite) TestReniceChurn(c *C) {
	const spread = 10
	jobs := make([]*slurmJob, 1000)
	for i := range jobs {
		jobs[i] = &slurmJob{priority: 4294000000 - int64(i), nice: 10000}
	}
	adjustments := 0
	queue := jobs
	for len(queue) > 0 {
		renice := wantNice(queue, spread)
		for i := range queue {
			if renice[i] == queue[i].nice {
				continue
			}
			queue[i].priority += queue[i].nice - renice[i]
			queue[i].nice = renice[i]
			adjustments++
		}
		queue = queue[1:]
	}
	c.Logf("processed queue of %d with %d renice ops", len(jobs), adjustments)
	c.Check(adjustments < len(jobs)*len(jobs)/10, Equals, true)
}
