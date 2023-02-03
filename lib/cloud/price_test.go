// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package cloud

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type cloudSuite struct{}

var _ = Suite(&cloudSuite{})

func (s *cloudSuite) TestNormalizePriceHistory(c *C) {
	t0, err := time.Parse(time.RFC3339, "2023-01-01T01:00:00Z")
	c.Assert(err, IsNil)
	h := []InstancePrice{
		{t0.Add(1 * time.Minute), 1.0},
		{t0.Add(4 * time.Minute), 1.2}, // drop: unchanged price
		{t0.Add(5 * time.Minute), 1.1},
		{t0.Add(3 * time.Minute), 1.2},
		{t0.Add(5 * time.Minute), 1.1}, // drop: duplicate
		{t0.Add(2 * time.Minute), 1.0}, // drop: out of order, unchanged price
	}
	c.Check(NormalizePriceHistory(h), DeepEquals, []InstancePrice{h[2], h[3], h[0]})
}
