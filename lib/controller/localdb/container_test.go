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
	"strings"
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
	containerPriorityUpdateInterval = 2 * time.Second
	s.localdbSuite.SetUpTest(c)
	s.starttime = time.Now()
	var err error
	s.topcr, err = s.localdb.ContainerRequestCreate(s.userctx, arvados.CreateOptions{Attrs: s.crAttrs(c)})
	c.Assert(err, IsNil)
	s.topc, err = s.localdb.ContainerGet(s.userctx, arvados.GetOptions{UUID: s.topcr.ContainerUUID})
	c.Assert(err, IsNil)
	c.Assert(int(s.topc.Priority), Not(Equals), 0)
	c.Logf("topcr %s topc %s", s.topcr.UUID, s.topc.UUID)
}

func (s *containerSuite) TearDownTest(c *C) {
	containerPriorityUpdateInterval = 5 * time.Minute
	s.localdbSuite.TearDownTest(c)
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
	testCtx, testCancel := context.WithDeadline(s.ctx, time.Now().Add(30*time.Second))
	defer testCancel()
	adminCtx := ctrlctx.NewWithToken(testCtx, s.cluster, s.cluster.SystemRootToken)

	childCR := func(parent arvados.ContainerRequest, arg string) arvados.ContainerRequest {
		attrs := s.crAttrs(c)
		attrs["command"] = []string{c.TestName(), fmt.Sprintf("%d", s.starttime.UnixMilli()), arg}
		cr, err := s.localdb.ContainerRequestCreate(s.userctx, arvados.CreateOptions{Attrs: attrs})
		c.Assert(err, IsNil)
		_, err = s.db.Exec("update container_requests set requesting_container_uuid=$1 where uuid=$2", parent.ContainerUUID, cr.UUID)
		c.Assert(err, IsNil)
		_, err = s.localdb.ContainerUpdate(adminCtx, arvados.UpdateOptions{
			UUID:  cr.ContainerUUID,
			Attrs: map[string]interface{}{"state": "Locked"},
		})
		c.Assert(err, IsNil)
		_, err = s.localdb.ContainerUpdate(adminCtx, arvados.UpdateOptions{
			UUID:  cr.ContainerUUID,
			Attrs: map[string]interface{}{"state": "Running"},
		})
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

	// Set priority=0 on a parent+child, plus 18 other randomly
	// selected containers in the tree
	//
	// First entries of needfix are allcrs[1] (which is "i 0") and
	// allcrs[2] ("i 0 j 0") -- we want to make sure to get at
	// least one parent/child pair -- and the rest were chosen
	// randomly.
	needfix := []int{1, 2, 23, 12, 20, 14, 13, 15, 7, 17, 6, 22, 21, 11, 1, 17, 18}
	for n, i := range needfix {
		needfix[n] = i
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

	// Flood railsapi with priority updates. This can cause
	// database deadlock: one call acquires row locks in the order
	// {i0j0, i0, i0j1}, while another call acquires row locks in
	// the order {i0j1, i0, i0j0}.
	deadlockCtx, deadlockCancel := context.WithDeadline(adminCtx, time.Now().Add(30*time.Second))
	defer deadlockCancel()
	for _, cr := range allcrs {
		if strings.Contains(cr.Command[2], " j ") && !strings.Contains(cr.Command[2], " k ") {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for _, p := range []int{1, 2, 3, 4} {
					var err error
					for {
						_, err = s.localdb.ContainerRequestUpdate(deadlockCtx, arvados.UpdateOptions{
							UUID: cr.UUID,
							Attrs: map[string]interface{}{
								"priority": p,
							},
						})
						if err != nil && strings.Contains(err.Error(), "TRDeadlockDetected") {
							c.Logf("Deadlock detected (will retry): %q", err)
							time.Sleep(time.Duration(rand.Intn(int(time.Second / 4))))
							continue
						}
						c.Check(err, IsNil)
						break
					}
				}
			}()
		}
	}
	wg.Wait()

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
				Limit:   10,
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
			time.Sleep(time.Second / 10)
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
			for i, cr := range allcrs {
				var ctr arvados.Container
				var command string
				err = s.db.QueryRowContext(s.ctx, `select cr.priority, cr.state, cr.container_uuid, c.state, c.priority, cr.command
					from container_requests cr
					left join containers c on cr.container_uuid = c.uuid
					where cr.uuid=$1`, cr.UUID).Scan(&cr.Priority, &cr.State, &ctr.UUID, &ctr.State, &ctr.Priority, &command)
				c.Check(err, IsNil)
				c.Logf("allcrs[%d] cr.pri %d %s c.pri %d %s cr.uuid %s c.uuid %s cmd %s", i, cr.Priority, cr.State, ctr.Priority, ctr.State, cr.UUID, ctr.UUID, command)
			}
			c.Fatal("timed out")
		}
		done := true
		for _, cr := range allcrs {
			var priority int
			var crstate, command, ctrUUID string
			var parent sql.NullString
			err := s.db.QueryRowContext(s.ctx, `select state, priority, container_uuid, requesting_container_uuid, command
				from container_requests where uuid=$1`, cr.UUID).Scan(&crstate, &priority, &ctrUUID, &parent, &command)
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
