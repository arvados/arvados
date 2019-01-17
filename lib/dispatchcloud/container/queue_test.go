// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package container

import (
	"sync"
	"testing"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&IntegrationSuite{})

type IntegrationSuite struct{}

func (*IntegrationSuite) TestControllerBackedQueue(c *check.C) {
	client := arvados.NewClientFromEnv()
	cq := NewQueue(logrus.StandardLogger(), nil, testTypeChooser, client)

	err := cq.Update()
	c.Check(err, check.IsNil)

	ents, threshold := cq.Entries()
	c.Check(len(ents), check.Not(check.Equals), 0)
	c.Check(time.Since(threshold) < time.Minute, check.Equals, true)
	c.Check(time.Since(threshold) > 0, check.Equals, true)

	_, ok := ents[arvadostest.QueuedContainerUUID]
	c.Check(ok, check.Equals, true)

	var wg sync.WaitGroup
	for uuid, ent := range ents {
		c.Check(ent.Container.UUID, check.Equals, uuid)
		c.Check(ent.InstanceType.Name, check.Equals, "testType")
		c.Check(ent.Container.State, check.Equals, arvados.ContainerStateQueued)
		c.Check(ent.Container.Priority > 0, check.Equals, true)

		ctr, ok := cq.Get(uuid)
		c.Check(ok, check.Equals, true)
		c.Check(ctr.UUID, check.Equals, uuid)

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cq.Unlock(uuid)
			c.Check(err, check.NotNil)
			err = cq.Lock(uuid)
			c.Check(err, check.IsNil)
			ctr, ok := cq.Get(uuid)
			c.Check(ok, check.Equals, true)
			c.Check(ctr.State, check.Equals, arvados.ContainerStateLocked)
			err = cq.Lock(uuid)
			c.Check(err, check.NotNil)
			err = cq.Unlock(uuid)
			c.Check(err, check.IsNil)
			ctr, ok = cq.Get(uuid)
			c.Check(ok, check.Equals, true)
			c.Check(ctr.State, check.Equals, arvados.ContainerStateQueued)
			err = cq.Unlock(uuid)
			c.Check(err, check.NotNil)
		}()
	}
	wg.Wait()

	err = cq.Cancel(arvadostest.CompletedContainerUUID)
	c.Check(err, check.ErrorMatches, `.*State cannot change from Complete to Cancelled.*`)
}

func testTypeChooser(ctr *arvados.Container) (arvados.InstanceType, error) {
	return arvados.InstanceType{Name: "testType"}, nil
}
