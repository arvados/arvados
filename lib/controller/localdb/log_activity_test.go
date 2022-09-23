// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&activityPeriodSuite{})

type activityPeriodSuite struct{}

// The important thing is that, even when daylight savings time is
// making things difficult, the current period ends in the future.
func (*activityPeriodSuite) TestPeriod(c *check.C) {
	toronto, err := time.LoadLocation("America/Toronto")
	c.Assert(err, check.IsNil)

	format := "2006-01-02 15:04:05 MST"
	dststartday, err := time.ParseInLocation(format, "2022-03-13 00:00:00 EST", toronto)
	c.Assert(err, check.IsNil)
	dstendday, err := time.ParseInLocation(format, "2022-11-06 00:00:00 EDT", toronto)
	c.Assert(err, check.IsNil)

	for _, period := range []time.Duration{
		time.Minute * 13,
		time.Minute * 49,
		time.Hour,
		4 * time.Hour,
		48 * time.Hour,
	} {
		for offset := time.Duration(0); offset < 48*time.Hour; offset += 3 * time.Minute {
			t := dststartday.Add(offset)
			end := alignedPeriod(t, period)
			c.Check(end.After(t), check.Equals, true, check.Commentf("period %v offset %v", period, offset))

			t = dstendday.Add(offset)
			end = alignedPeriod(t, period)
			c.Check(end.After(t), check.Equals, true, check.Commentf("period %v offset %v", period, offset))
		}
	}
}
