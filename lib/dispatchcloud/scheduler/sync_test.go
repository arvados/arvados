// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"context"
	"time"

	"git.arvados.org/arvados.git/lib/dispatchcloud/test"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

// Ensure the scheduler expunges containers from the queue when they
// are no longer relevant (completed and not running, queued with
// priority 0, etc).
func (*SchedulerSuite) TestForgetIrrelevantContainers(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	pool := stubPool{}
	queue := test.Queue{
		ChooseType: chooseType,
		Containers: []arvados.Container{
			{
				UUID:     test.ContainerUUID(1),
				Priority: 0,
				State:    arvados.ContainerStateQueued,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
			{
				UUID:     test.ContainerUUID(2),
				Priority: 12345,
				State:    arvados.ContainerStateComplete,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
		},
	}
	queue.Update()

	ents, _ := queue.Entries()
	c.Check(ents, check.HasLen, 1)

	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond, 0)
	sch.sync()

	ents, _ = queue.Entries()
	c.Check(ents, check.HasLen, 0)
}

func (*SchedulerSuite) TestCancelOrphanedContainers(c *check.C) {
	ctx := ctxlog.Context(context.Background(), ctxlog.TestLogger(c))
	pool := stubPool{
		unalloc: map[arvados.InstanceType]int{test.InstanceType(1): 1},
		unknown: map[arvados.InstanceType]int{test.InstanceType(1): 1},
	}
	queue := test.Queue{
		ChooseType: chooseType,
		Containers: []arvados.Container{
			{
				UUID:     test.ContainerUUID(1),
				Priority: 0,
				State:    arvados.ContainerStateRunning,
				RuntimeConstraints: arvados.RuntimeConstraints{
					VCPUs: 1,
					RAM:   1 << 30,
				},
			},
		},
	}
	queue.Update()

	ents, _ := queue.Entries()
	c.Check(ents, check.HasLen, 1)

	sch := New(ctx, arvados.NewClientFromEnv(), &queue, &pool, nil, time.Millisecond, time.Millisecond, 0)

	// Sync shouldn't cancel the container because it might be
	// running on the VM with state=="unknown".
	//
	// (Cancel+forget happens asynchronously and requires multiple
	// sync() calls, so even after 10x sync-and-sleep iterations,
	// we aren't 100% confident that sync isn't trying to
	// cancel. But in the test environment, the goroutines started
	// by sync() access stubs and therefore run quickly, so it
	// works fine in practice. We accept that if the code is
	// broken, the test will still pass occasionally.)
	for i := 0; i < 10; i++ {
		sch.sync()
		time.Sleep(time.Millisecond)
	}
	ents, _ = queue.Entries()
	c.Check(ents, check.HasLen, 1)
	c.Check(ents[test.ContainerUUID(1)].Container.State, check.Equals, arvados.ContainerStateRunning)

	// Sync should cancel & forget the container when the
	// "unknown" node goes away.
	//
	// (As above, cancel+forget is async and requires multiple
	// sync() calls, but stubs are fast so in practice this takes
	// much less than 1s to complete.)
	pool.unknown = nil
	for deadline := time.Now().Add(time.Second); ; time.Sleep(time.Millisecond) {
		sch.sync()
		ents, _ = queue.Entries()
		if len(ents) == 0 || time.Now().After(deadline) {
			break
		}
	}
	c.Check(ents, check.HasLen, 0)
}
