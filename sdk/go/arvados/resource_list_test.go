// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"time"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&filterEncodingSuite{})

type filterEncodingSuite struct{}

func (s *filterEncodingSuite) TestMarshalNanoseconds(c *check.C) {
	t0 := time.Now()
	t0str := t0.Format(time.RFC3339Nano)
	buf, err := json.Marshal([]Filter{
		{Attr: "modified_at", Operator: "=", Operand: t0}})
	c.Assert(err, check.IsNil)
	c.Check(string(buf), check.Equals, `[["modified_at","=","`+t0str+`"]]`)
}

func (s *filterEncodingSuite) TestMarshalNil(c *check.C) {
	buf, err := json.Marshal([]Filter{
		{Attr: "modified_at", Operator: "=", Operand: nil}})
	c.Assert(err, check.IsNil)
	c.Check(string(buf), check.Equals, `[["modified_at","=",null]]`)
}

func (s *filterEncodingSuite) TestUnmarshalNil(c *check.C) {
	buf := []byte(`["modified_at","=",null]`)
	var f Filter
	err := f.UnmarshalJSON(buf)
	c.Assert(err, check.IsNil)
	c.Check(f, check.DeepEquals, Filter{Attr: "modified_at", Operator: "=", Operand: nil})
}

func (s *filterEncodingSuite) TestMarshalBoolean(c *check.C) {
	buf, err := json.Marshal([]Filter{
		{Attr: "is_active", Operator: "=", Operand: true}})
	c.Assert(err, check.IsNil)
	c.Check(string(buf), check.Equals, `[["is_active","=",true]]`)
}

func (s *filterEncodingSuite) TestUnmarshalBoolean(c *check.C) {
	buf := []byte(`["is_active","=",true]`)
	var f Filter
	err := f.UnmarshalJSON(buf)
	c.Assert(err, check.IsNil)
	c.Check(f, check.DeepEquals, Filter{Attr: "is_active", Operator: "=", Operand: true})
}

func (s *filterEncodingSuite) TestUnmarshalBooleanExpression(c *check.C) {
	buf := []byte(`"(foo < bar)"`)
	var f Filter
	err := f.UnmarshalJSON(buf)
	c.Assert(err, check.IsNil)
	c.Check(f, check.DeepEquals, Filter{Attr: "(foo < bar)", Operator: "=", Operand: true})
}
