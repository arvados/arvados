// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package scheduler

import (
	"context"
	"time"

	"git.curoverse.com/arvados.git/lib/dispatchcloud/test"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/ctxlog"
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

	sch := New(ctx, &queue, &pool, time.Millisecond, time.Millisecond)
	sch.sync()

	ents, _ = queue.Entries()
	c.Check(ents, check.HasLen, 0)
}
