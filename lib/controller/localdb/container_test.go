// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
)

var _ = Suite(&containerSuite{})

type containerSuite struct {
	localdbSuite
	topcr     arvados.ContainerRequest
	topc      arvados.Container
	starttime time.Time
}

func (s *containerSuite) crAttrs(c *C) map[string]interface{} {
	return map[string]interface{}{
		"container_image":     arvadostest.DockerImage112PDH,
		"command":             []string{c.TestName(), fmt.Sprintf("%d", s.starttime.UnixMilli()), "top"},
		"output_path":         "/out",
		"priority":            1,
		"state":               "Committed",
		"container_count_max": 1,
		"runtime_constraints": arvados.RuntimeConstraints{
			RAM:   1,
			VCPUs: 1,
		},
		"mounts": map[string]arvados.Mount{
			"/out": arvados.Mount{},
		},
	}
}

func (s *containerSuite) SetUpTest(c *C) {
	s.localdbSuite.SetUpTest(c)
	var err error
	s.topcr, err = s.localdb.ContainerRequestCreate(s.userctx, arvados.CreateOptions{Attrs: s.crAttrs(c)})
	c.Assert(err, IsNil)
	s.topc, err = s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.topcr.ContainerUUID})
	c.Assert(err, IsNil)
	c.Assert(int(s.topc.Priority), Not(Equals), 0)
	c.Logf("topcr %s topc %s", s.topcr.UUID, s.topc.UUID)
	s.starttime = time.Now()
}

func (s *containerSuite) syncUpdatePriority(c *C) {
	// Sending 1x to the "update now" channel starts an update;
	// sending again fills the channel while the first update is
	// running; sending a third time blocks until the worker
	// receives the 2nd send, i.e., guarantees that the first
	// update has finished.
	s.localdb.wantContainerPriorityUpdate <- struct{}{}
	s.localdb.wantContainerPriorityUpdate <- struct{}{}
	s.localdb.wantContainerPriorityUpdate <- struct{}{}
}

func (s *containerSuite) TestUpdatePriorityShouldBeNonZero(c *C) {
	_, err := s.db.Exec("update containers set priority=0 where uuid=$1", s.topc.UUID)
	c.Assert(err, IsNil)
	topc, err := s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.topc.UUID})
	c.Assert(err, IsNil)
	c.Assert(int(topc.Priority), Equals, 0)
	s.syncUpdatePriority(c)
	topc, err = s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.topc.UUID})
	c.Assert(err, IsNil)
	c.Check(int(topc.Priority), Not(Equals), 0)
}

func (s *containerSuite) TestUpdatePriorityShouldBeZero(c *C) {
	_, err := s.db.Exec("update container_requests set priority=0 where uuid=$1", s.topcr.UUID)
	c.Assert(err, IsNil)
	topc, err := s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.topc.UUID})
	c.Assert(err, IsNil)
	c.Assert(int(topc.Priority), Not(Equals), 0)
	s.syncUpdatePriority(c)
	topc, err = s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.topc.UUID})
	c.Assert(err, IsNil)
	c.Check(int(topc.Priority), Equals, 0)
}

func (s *containerSuite) TestUpdatePriorityMultiLevelWorkflow(c *C) {
	childCR := func(parent arvados.ContainerRequest, arg string) arvados.ContainerRequest {
		attrs := s.crAttrs(c)
		attrs["command"] = []string{c.TestName(), fmt.Sprintf("%d", s.starttime.UnixMilli()), arg}
		cr, err := s.localdb.ContainerRequestCreate(s.userctx, arvados.CreateOptions{Attrs: attrs})
		c.Assert(err, IsNil)
		_, err = s.db.Exec("update container_requests set requesting_container_uuid=$1 where uuid=$2", parent.ContainerUUID, cr.UUID)
		c.Assert(err, IsNil)
		return cr
	}
	// Build a tree of container requests and containers (3 levels
	// deep below s.topcr)
	allcrs := []arvados.ContainerRequest{s.topcr}
	for i := 0; i < 2; i++ {
		cri := childCR(s.topcr, fmt.Sprintf("i %d", i))
		allcrs = append(allcrs, cri)
		for j := 0; j < 3; j++ {
			crj := childCR(cri, fmt.Sprintf("i %d j %d", i, j))
			allcrs = append(allcrs, crj)
			for k := 0; k < 4; k++ {
				crk := childCR(crj, fmt.Sprintf("i %d j %d k %d", i, j, k))
				allcrs = append(allcrs, crk)
			}
		}
	}

	testCtx, testCancel := context.WithDeadline(s.ctx, time.Now().Add(time.Second*20))
	defer testCancel()

	// Set priority=0 on a parent+child, plus 18 other randomly
	// selected containers in the tree
	adminCtx := ctrlctx.NewWithToken(testCtx, s.cluster, s.cluster.SystemRootToken)
	needfix := make([]int, 20)
	running := make(map[int]bool)
	for n := range needfix {
		var i int // which container are we going to run & then set priority=0
		if n < 2 {
			// first two are allcrs[1] (which is "i 0")
			// and allcrs[2] (which is "i 0 j 0")
			i = n + 1
		} else {
			// rest are random
			i = rand.Intn(len(allcrs))
		}
		needfix[n] = i
		if !running[i] {
			_, err := s.localdb.ContainerUpdate(adminCtx, arvados.UpdateOptions{
				UUID:  allcrs[i].ContainerUUID,
				Attrs: map[string]interface{}{"state": "Locked"},
			})
			c.Assert(err, IsNil)
			_, err = s.localdb.ContainerUpdate(adminCtx, arvados.UpdateOptions{
				UUID:  allcrs[i].ContainerUUID,
				Attrs: map[string]interface{}{"state": "Running"},
			})
			c.Assert(err, IsNil)
			running[i] = true
		}
		res, err := s.db.Exec("update containers set priority=0 where uuid=$1", allcrs[i].ContainerUUID)
		c.Assert(err, IsNil)
		updated, err := res.RowsAffected()
		c.Assert(err, IsNil)
		if n == 0 {
			c.Assert(int(updated), Equals, 1)
		}
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	chaosCtx, chaosCancel := context.WithCancel(adminCtx)
	defer chaosCancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Flood the api with ContainerUpdate calls for the
		// same containers that need to have their priority
		// fixed
		for chaosCtx.Err() == nil {
			n := rand.Intn(len(needfix))
			_, err := s.localdb.ContainerUpdate(chaosCtx, arvados.UpdateOptions{
				UUID: allcrs[needfix[n]].ContainerUUID,
				Attrs: map[string]interface{}{
					"runtime_status": map[string]string{
						"info": time.Now().Format(time.RFC3339Nano),
					},
				},
			})
			if !errors.Is(err, context.Canceled) {
				c.Check(err, IsNil)
			}
		}
	}()
	// Find and fix the containers with wrong priority
	s.syncUpdatePriority(c)
	// Ensure they all got fixed
	for _, cr := range allcrs {
		var priority int
		err := s.db.QueryRow("select priority from containers where uuid=$1", cr.ContainerUUID).Scan(&priority)
		c.Assert(err, IsNil)
		c.Check(priority, Not(Equals), 0)
	}

	chaosCancel()

	// Simulate cascading cancellation of the entire tree. For
	// this we need a goroutine to notice and cancel containers
	// with state=Running and priority=0, and cancel them
	// (this is normally done by a dispatcher).
	dispCtx, dispCancel := context.WithCancel(adminCtx)
	defer dispCancel()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for dispCtx.Err() == nil {
			needcancel, err := s.localdb.ContainerList(dispCtx, arvados.ListOptions{
				Limit:   1,
				Filters: []arvados.Filter{{"state", "=", "Running"}, {"priority", "=", 0}},
			})
			if errors.Is(err, context.Canceled) {
				break
			}
			c.Assert(err, IsNil)
			for _, ctr := range needcancel.Items {
				_, err := s.localdb.ContainerUpdate(dispCtx, arvados.UpdateOptions{
					UUID: ctr.UUID,
					Attrs: map[string]interface{}{
						"state": "Cancelled",
					},
				})
				c.Assert(err, IsNil)
			}
		}
	}()

	_, err := s.localdb.ContainerRequestUpdate(s.userctx, arvados.UpdateOptions{
		UUID: s.topcr.UUID,
		Attrs: map[string]interface{}{
			"priority": 0,
		},
	})
	c.Assert(err, IsNil)

	c.Logf("waiting for all %d containers to have priority=0 after cancelling top level CR", len(allcrs))
	for {
		time.Sleep(time.Second / 2)
		if testCtx.Err() != nil {
			c.Fatal("timed out")
		}
		done := true
		for _, cr := range allcrs {
			var priority int
			var crstate, command, ctrUUID string
			var parent sql.NullString
			err := s.db.QueryRowContext(s.ctx, "select state, priority, command, container_uuid, requesting_container_uuid from container_requests where uuid=$1", cr.UUID).Scan(&crstate, &priority, &command, &ctrUUID, &parent)
			if errors.Is(err, context.Canceled) {
				break
			}
			c.Assert(err, IsNil)
			if crstate == "Committed" && priority > 0 {
				c.Logf("container request %s (%s; parent=%s) still has state %s priority %d", cr.UUID, command, parent.String, crstate, priority)
				done = false
				break
			}
			err = s.db.QueryRowContext(s.ctx, "select priority, command from containers where uuid=$1", cr.ContainerUUID).Scan(&priority, &command)
			if errors.Is(err, context.Canceled) {
				break
			}
			c.Assert(err, IsNil)
			if priority > 0 {
				c.Logf("container %s (%s) still has priority %d", cr.ContainerUUID, command, priority)
				done = false
				break
			}
		}
		if done {
			c.Logf("success -- all %d containers have priority=0", len(allcrs))
			break
		}
	}
}
