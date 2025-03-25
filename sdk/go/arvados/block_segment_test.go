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

func (s *blockSegmentSuite) TestUnmarshal(c *C) {
	var dst struct {
		F map[BlockSegment]BlockSegment
	}
	err := json.Unmarshal([]byte(`{"f": {"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1 0 1": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+2 1 2"}}`), &dst)
	c.Check(err, IsNil)
	c.Check(dst.F, HasLen, 1)
	for k, v := range dst.F {
		c.Check(k, Equals, BlockSegment{"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1", 0, 1})
		c.Check(v, Equals, BlockSegment{"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb+2", 1, 2})
	}
}
