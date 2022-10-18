// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&DurationSuite{})

type DurationSuite struct{}

func (s *DurationSuite) TestMarshalJSON(c *check.C) {
	var d struct {
		D Duration
	}
	err := json.Unmarshal([]byte(`{"D":"1.234s"}`), &d)
	c.Check(err, check.IsNil)
	c.Check(d.D, check.Equals, Duration(time.Second+234*time.Millisecond))
	buf, err := json.Marshal(d)
	c.Check(err, check.IsNil)
	c.Check(string(buf), check.Equals, `{"D":"1.234s"}`)

	for _, trial := range []struct {
		seconds int
		out     string
	}{
		{30, "30s"},
		{60, "1m"},
		{120, "2m"},
		{150, "2m30s"},
		{3600, "1h"},
		{7201, "2h1s"},
		{360600, "100h10m"},
		{360610, "100h10m10s"},
	} {
		buf, err := json.Marshal(Duration(time.Duration(trial.seconds) * time.Second))
		c.Check(err, check.IsNil)
		c.Check(string(buf), check.Equals, `"`+trial.out+`"`)
	}
}

func (s *DurationSuite) TestUnmarshalJSON(c *check.C) {
	var d struct {
		D Duration
	}
	err := json.Unmarshal([]byte(`{"D":1.234}`), &d)
	c.Check(err, check.ErrorMatches, `.*missing unit in duration "?1\.234"?`)
	err = json.Unmarshal([]byte(`{"D":"1.234"}`), &d)
	c.Check(err, check.ErrorMatches, `.*missing unit in duration "?1\.234"?`)
	err = json.Unmarshal([]byte(`{"D":"1"}`), &d)
	c.Check(err, check.ErrorMatches, `.*missing unit in duration "?1"?`)
	err = json.Unmarshal([]byte(`{"D":"foobar"}`), &d)
	c.Check(err, check.ErrorMatches, `.*invalid duration "?foobar"?`)
	err = json.Unmarshal([]byte(`{"D":"60s"}`), &d)
	c.Check(err, check.IsNil)
	c.Check(d.D.Duration(), check.Equals, time.Minute)

	d.D = Duration(time.Second)
	err = json.Unmarshal([]byte(`{"D":"0"}`), &d)
	c.Check(err, check.IsNil)
	c.Check(d.D.Duration(), check.Equals, time.Duration(0))

	d.D = Duration(time.Second)
	err = json.Unmarshal([]byte(`{"D":0}`), &d)
	c.Check(err, check.IsNil)
	c.Check(d.D.Duration(), check.Equals, time.Duration(0))
}
