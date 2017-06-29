// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package dispatch

import (
	"testing"
	"time"

	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&ThrottleTestSuite{})

type ThrottleTestSuite struct{}

func (*ThrottleTestSuite) TestThrottle(c *check.C) {
	uuid := "zzzzz-zzzzz-zzzzzzzzzzzzzzz"
	t0 := throttle{}
	c.Check(t0.Check(uuid), check.Equals, true)
	c.Check(t0.Check(uuid), check.Equals, true)

	tNs := throttle{hold: time.Nanosecond}
	c.Check(tNs.Check(uuid), check.Equals, true)
	time.Sleep(time.Microsecond)
	c.Check(tNs.Check(uuid), check.Equals, true)

	tMin := throttle{hold: time.Minute}
	c.Check(tMin.Check(uuid), check.Equals, true)
	c.Check(tMin.Check(uuid), check.Equals, false)
	c.Check(tMin.Check(uuid), check.Equals, false)
	tMin.seen[uuid].last = time.Now().Add(-time.Hour)
	c.Check(tMin.Check(uuid), check.Equals, true)
	c.Check(tMin.Check(uuid), check.Equals, false)
}
