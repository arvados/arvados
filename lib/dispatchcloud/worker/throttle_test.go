// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package worker

import (
	"errors"
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&ThrottleSuite{})

type ThrottleSuite struct{}

func (s *ThrottleSuite) TestRateLimitError(c *check.C) {
	var t throttle
	c.Check(t.Error(), check.IsNil)
	t.ErrorUntil(errors.New("wait"), time.Now().Add(time.Second), nil)
	c.Check(t.Error(), check.NotNil)
	t.ErrorUntil(nil, time.Now(), nil)
	c.Check(t.Error(), check.IsNil)

	notified := false
	t.ErrorUntil(errors.New("wait"), time.Now().Add(time.Millisecond), func() { notified = true })
	c.Check(t.Error(), check.NotNil)
	time.Sleep(time.Millisecond * 10)
	c.Check(t.Error(), check.IsNil)
	c.Check(notified, check.Equals, true)
}
