// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package ws

import (
	"bytes"
	"encoding/json"
	"net/url"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&eventSuite{})

type eventSuite struct{}

func (*eventSuite) TestDetail(c *check.C) {
	e := &event{
		LogID:  17,
		DB:     testDB(),
		Logger: ctxlog.TestLogger(c),
	}
	logRow := e.Detail()
	c.Assert(logRow, check.NotNil)
	c.Check(logRow, check.Equals, e.logRow)
	c.Check(logRow.UUID, check.Equals, "zzzzz-57u5n-containerlog006")
	c.Check(logRow.ObjectUUID, check.Equals, "zzzzz-dz642-logscontainer03")
	c.Check(logRow.EventType, check.Equals, "crunchstat")
	c.Check(logRow.Properties["text"], check.Equals, "2013-11-07_23:33:41 zzzzz-dz642-logscontainer03 29610 1 stderr crunchstat: cpu 1935.4300 user 59.4100 sys 8 cpus -- interval 10.0002 seconds 12.9900 user 0.9900 sys")
}

func (*eventSuite) TestDetail_Properties(c *check.C) {
	ac := arvados.NewClientFromEnv()
	jsondata, err := json.Marshal(map[string]interface{}{
		"object_uuid": arvadostest.RunningContainerUUID,
		"event_type":  "blip",
		"properties":  nil,
	})
	c.Assert(err, check.IsNil)
	var lg arvados.Log
	err = ac.RequestAndDecode(&lg, "POST", "arvados/v1/logs", bytes.NewBufferString(url.Values{"log": []string{string(jsondata)}}.Encode()), nil)
	c.Assert(err, check.IsNil)
	defer testDB().Exec(`delete from logs where id=$1`, lg.ID)

	enoprop := &event{
		LogID:  lg.ID,
		DB:     testDB(),
		Logger: ctxlog.TestLogger(c),
	}
	logRow := enoprop.Detail()
	c.Assert(logRow, check.NotNil)
	c.Check(logRow.Properties, check.DeepEquals, map[string]interface{}{})

	_, err = testDB().Exec(`update logs set properties='bad properties' where id=$1`, lg.ID)
	c.Assert(err, check.IsNil)
	ebadprop := &event{
		LogID:  lg.ID,
		DB:     testDB(),
		Logger: ctxlog.TestLogger(c),
	}
	logRow = ebadprop.Detail()
	c.Check(logRow, check.IsNil)
}
