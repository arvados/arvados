// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

import (
	"time"

	"github.com/sirupsen/logrus"
	. "gopkg.in/check.v1"
)

var _ = Suite(&SqueueSuite{})

type SqueueSuite struct{}

func (s *SqueueSuite) TestReleasePending(c *C) {
	uuids := []string{
		"zzzzz-dz642-fake0fake0fake0",
		"zzzzz-dz642-fake1fake1fake1",
		"zzzzz-dz642-fake2fake2fake2",
	}
	slurm := &slurmFake{
		queue: uuids[0] + " 10000 4294000000 PENDING Resources\n" + uuids[1] + " 10000 4294000111 PENDING Resources\n" + uuids[2] + " 10000 0 PENDING BadConstraints\n",
	}
	sqc := &SqueueChecker{
		Logger: logrus.StandardLogger(),
		Slurm:  slurm,
		Period: time.Hour,
	}
	sqc.startOnce.Do(sqc.start)
	defer sqc.Stop()

	done := make(chan struct{})
	go func() {
		for _, u := range uuids {
			sqc.SetPriority(u, 1)
		}
		close(done)
	}()
	callUntilReady(sqc.check, done)

	slurm.didRelease = nil
	sqc.check()
	c.Check(slurm.didRelease, DeepEquals, []string{uuids[2]})
}

func (s *SqueueSuite) TestReniceAll(c *C) {
	uuids := []string{"zzzzz-dz642-fake0fake0fake0", "zzzzz-dz642-fake1fake1fake1", "zzzzz-dz642-fake2fake2fake2"}
	for _, test := range []struct {
		spread int64
		squeue string
		want   map[string]int64
		expect [][]string
	}{
		{
			spread: 1,
			squeue: uuids[0] + " 10000 4294000000 PENDING Resources\n",
			want:   map[string]int64{uuids[0]: 1},
			expect: [][]string{{uuids[0], "0"}},
		},
		{ // fake0 priority is too high
			spread: 1,
			squeue: uuids[0] + " 10000 4294000777 PENDING Resources\n" + uuids[1] + " 10000 4294000444 PENDING Resources\n",
			want:   map[string]int64{uuids[0]: 1, uuids[1]: 999},
			expect: [][]string{{uuids[1], "0"}, {uuids[0], "334"}},
		},
		{ // specify spread
			spread: 100,
			squeue: uuids[0] + " 10000 4294000777 PENDING Resources\n" + uuids[1] + " 10000 4294000444 PENDING Resources\n",
			want:   map[string]int64{uuids[0]: 1, uuids[1]: 999},
			expect: [][]string{{uuids[1], "0"}, {uuids[0], "433"}},
		},
		{ // ignore fake2 because SetPriority() not called
			spread: 1,
			squeue: uuids[0] + " 10000 4294000000 PENDING Resources\n" + uuids[1] + " 10000 4294000111 PENDING Resources\n" + uuids[2] + " 10000 4294000222 PENDING Resources\n",
			want:   map[string]int64{uuids[0]: 999, uuids[1]: 1},
			expect: [][]string{{uuids[0], "0"}, {uuids[1], "112"}},
		},
		{ // ignore fake2 because slurm priority=0
			spread: 1,
			squeue: uuids[0] + " 10000 4294000000 PENDING Resources\n" + uuids[1] + " 10000 4294000111 PENDING Resources\n" + uuids[2] + " 10000 0 PENDING Resources\n",
			want:   map[string]int64{uuids[0]: 999, uuids[1]: 1, uuids[2]: 997},
			expect: [][]string{{uuids[0], "0"}, {uuids[1], "112"}},
		},
	} {
		c.Logf("spread=%d squeue=%q want=%v -> expect=%v", test.spread, test.squeue, test.want, test.expect)
		slurm := &slurmFake{
			queue: test.squeue,
		}
		sqc := &SqueueChecker{
			Logger:         logrus.StandardLogger(),
			Slurm:          slurm,
			PrioritySpread: test.spread,
			Period:         time.Hour,
		}
		sqc.startOnce.Do(sqc.start)
		sqc.check()
		for uuid, pri := range test.want {
			sqc.SetPriority(uuid, pri)
		}
		sqc.reniceAll()
		c.Check(slurm.didRenice, DeepEquals, test.expect)
		sqc.Stop()
	}
}

// If a limited nice range prevents desired priority adjustments, give
// up and clamp nice to 10K.
func (s *SqueueSuite) TestReniceInvalidNiceValue(c *C) {
	uuids := []string{"zzzzz-dz642-fake0fake0fake0", "zzzzz-dz642-fake1fake1fake1", "zzzzz-dz642-fake2fake2fake2"}
	slurm := &slurmFake{
		queue:         uuids[0] + " 0 4294000222 PENDING Resources\n" + uuids[1] + " 0 4294555222 PENDING Resources\n",
		rejectNice10K: true,
	}
	sqc := &SqueueChecker{
		Logger:         logrus.StandardLogger(),
		Slurm:          slurm,
		PrioritySpread: 1,
		Period:         time.Hour,
	}
	sqc.startOnce.Do(sqc.start)
	sqc.check()
	sqc.SetPriority(uuids[0], 2)
	sqc.SetPriority(uuids[1], 1)

	// First attempt should renice to 555001, which will fail
	sqc.reniceAll()
	c.Check(slurm.didRenice, DeepEquals, [][]string{{uuids[1], "555001"}})

	// Next attempt should renice to 10K, which will succeed
	sqc.reniceAll()
	c.Check(slurm.didRenice, DeepEquals, [][]string{{uuids[1], "555001"}, {uuids[1], "10000"}})
	// ...so we'll change the squeue response to reflect the
	// updated priority+nice, and make sure sqc sees that...
	slurm.queue = uuids[0] + " 0 4294000222 PENDING Resources\n" + uuids[1] + " 10000 4294545222 PENDING Resources\n"
	sqc.check()

	// Next attempt should leave nice alone because it's already
	// at the 10K limit
	sqc.reniceAll()
	c.Check(slurm.didRenice, DeepEquals, [][]string{{uuids[1], "555001"}, {uuids[1], "10000"}})

	// Back to normal if desired nice value falls below 10K
	slurm.queue = uuids[0] + " 0 4294000222 PENDING Resources\n" + uuids[1] + " 10000 4294000111 PENDING Resources\n"
	sqc.check()
	sqc.reniceAll()
	c.Check(slurm.didRenice, DeepEquals, [][]string{{uuids[1], "555001"}, {uuids[1], "10000"}, {uuids[1], "9890"}})

	sqc.Stop()
}

// If the given UUID isn't in the slurm queue yet, SetPriority()
// should wait for it to appear on the very next poll, then give up.
func (s *SqueueSuite) TestSetPriorityBeforeQueued(c *C) {
	uuidGood := "zzzzz-dz642-fake0fake0fake0"
	uuidBad := "zzzzz-dz642-fake1fake1fake1"

	slurm := &slurmFake{}
	sqc := &SqueueChecker{
		Logger: logrus.StandardLogger(),
		Slurm:  slurm,
		Period: time.Hour,
	}
	sqc.startOnce.Do(sqc.start)
	sqc.Stop()
	sqc.check()

	done := make(chan struct{})
	go func() {
		sqc.SetPriority(uuidGood, 123)
		sqc.SetPriority(uuidBad, 345)
		close(done)
	}()
	c.Check(sqc.queue[uuidGood], IsNil)
	c.Check(sqc.queue[uuidBad], IsNil)
	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			slurm.queue = uuidGood + " 0 12345 PENDING Resources\n"
			sqc.check()

			// Avoid immediately selecting this case again
			// on the next iteration if check() took
			// longer than one tick.
			select {
			case <-tick.C:
			default:
			}
		case <-timeout.C:
			c.Fatal("timed out")
		case <-done:
			c.Assert(sqc.queue[uuidGood], NotNil)
			c.Check(sqc.queue[uuidGood].wantPriority, Equals, int64(123))
			c.Check(sqc.queue[uuidBad], IsNil)
			return
		}
	}
}

func callUntilReady(fn func(), done <-chan struct{}) {
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-done:
			return
		case <-tick.C:
			fn()
		}
	}
}
