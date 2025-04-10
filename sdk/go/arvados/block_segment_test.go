// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"

	. "gopkg.in/check.v1"
)

var _ = Suite(&blockSegmentSuite{})

type blockSegmentSuite struct{}

func (s *blockSegmentSuite) TestMarshal(c *C) {
	dst, err := json.Marshal(map[BlockSegment]BlockSegment{
		BlockSegment{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1", 0, 1}: BlockSegment{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+3", 2, 1},
	})
	c.Check(err, IsNil)
	c.Check(string(dst), Equals, `{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1 0 1":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+3 2 1"}`)
}

func (s *blockSegmentSuite) TestUnmarshal(c *C) {
	var dst struct {
		F map[BlockSegment]BlockSegment
	}
	err := json.Unmarshal([]byte(`{"f": {"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1 0 1": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+3 2 1"}}`), &dst)
	c.Check(err, IsNil)
	c.Check(dst.F, HasLen, 1)
	for k, v := range dst.F {
		c.Check(k, Equals, BlockSegment{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1", 0, 1})
		c.Check(v, Equals, BlockSegment{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+3", 2, 1})
	}
}

func (s *blockSegmentSuite) TestRoundTrip(c *C) {
	orig := map[BlockSegment]BlockSegment{
		BlockSegment{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1", 0, 1}:   BlockSegment{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+50", 0, 1},
		BlockSegment{"cccccccccccccccccccccccccccccccc+49", 0, 49}: BlockSegment{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+50", 1, 49},
	}
	j, err := json.Marshal(orig)
	c.Check(err, IsNil)
	var dst map[BlockSegment]BlockSegment
	err = json.Unmarshal(j, &dst)
	c.Check(err, IsNil)
	c.Check(dst, DeepEquals, orig)
}
