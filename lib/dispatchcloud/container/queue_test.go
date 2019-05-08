// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package container

import (
	"errors"
	"os"
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

func logger() logrus.FieldLogger {
	logger := logrus.StandardLogger()
	if os.Getenv("ARVADOS_DEBUG") != "" {
		logger.SetLevel(logrus.DebugLevel)
	}
	return logger
}

type IntegrationSuite struct{}

func (suite *IntegrationSuite) TearDownTest(c *check.C) {
	err := arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil)
	c.Check(err, check.IsNil)
}

func (suite *IntegrationSuite) TestGetLockUnlockCancel(c *check.C) {
	typeChooser := func(ctr *arvados.Container) (arvados.InstanceType, error) {
		return arvados.InstanceType{Name: "testType"}, nil
	}

	client := arvados.NewClientFromEnv()
	cq := NewQueue(logger(), nil, typeChooser, client)

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
			c.Check(err, check.ErrorMatches, ".*cannot unlock when Queued*.")

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

			err = cq.Cancel(uuid)
			c.Check(err, check.IsNil)
			ctr, ok = cq.Get(uuid)
			c.Check(ok, check.Equals, true)
			c.Check(ctr.State, check.Equals, arvados.ContainerStateCancelled)
			err = cq.Lock(uuid)
			c.Check(err, check.NotNil)
		}()
	}
	wg.Wait()
}

func (suite *IntegrationSuite) TestCancelIfNoInstanceType(c *check.C) {
	errorTypeChooser := func(ctr *arvados.Container) (arvados.InstanceType, error) {
		return arvados.InstanceType{}, errors.New("no suitable instance type")
	}

	client := arvados.NewClientFromEnv()
	cq := NewQueue(logger(), nil, errorTypeChooser, client)

	var ctr arvados.Container
	err := client.RequestAndDecode(&ctr, "GET", "arvados/v1/containers/"+arvadostest.QueuedContainerUUID, nil, nil)
	c.Check(err, check.IsNil)
	c.Check(ctr.State, check.Equals, arvados.ContainerStateQueued)

	cq.Update()

	// Wait for the cancel operation to take effect. Container
	// will have state=Cancelled or just disappear from the queue.
	suite.waitfor(c, time.Second, func() bool {
		err := client.RequestAndDecode(&ctr, "GET", "arvados/v1/containers/"+arvadostest.QueuedContainerUUID, nil, nil)
		return err == nil && ctr.State == arvados.ContainerStateCancelled
	})
	c.Check(ctr.RuntimeStatus["error"], check.Equals, `no suitable instance type`)
}

func (suite *IntegrationSuite) waitfor(c *check.C, timeout time.Duration, fn func() bool) {
	defer func() {
		c.Check(fn(), check.Equals, true)
	}()
	deadline := time.Now().Add(timeout)
	for !fn() && time.Now().Before(deadline) {
		time.Sleep(timeout / 1000)
	}
}
