// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import check "gopkg.in/check.v1"

var _ = check.Suite(&eventSuite{})

type eventSuite struct{}

func (*eventSuite) TestDetail(c *check.C) {
	e := &event{
		LogID: 17,
		db:    testDB(),
	}
	logRow := e.Detail()
	c.Assert(logRow, check.NotNil)
	c.Check(logRow, check.Equals, e.logRow)
	c.Check(logRow.UUID, check.Equals, "zzzzz-57u5n-containerlog006")
	c.Check(logRow.ObjectUUID, check.Equals, "zzzzz-dz642-logscontainer03")
	c.Check(logRow.EventType, check.Equals, "crunchstat")
	c.Check(logRow.Properties["text"], check.Equals, "2013-11-07_23:33:41 zzzzz-dz642-logscontainer03 29610 1 stderr crunchstat: cpu 1935.4300 user 59.4100 sys 8 cpus -- interval 10.0002 seconds 12.9900 user 0.9900 sys")
}
