// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollectionSuite{})

type CollectionSuite struct{}

func (s *CollectionSuite) TestSizedDigests(c *check.C) {
	coll := Collection{ManifestText: ". d41d8cd98f00b204e9800998ecf8427e+0 acbd18db4cc2f85cedef654fccc4a4d8+3 73feffa4b7f6bb68e44cf984c85f6e88+3+Z+K@xyzzy 0:0:foo 0:3:bar 3:3:baz\n"}
	sd, err := coll.SizedDigests()
	c.Check(err, check.IsNil)
	c.Check(sd, check.DeepEquals, []SizedDigest{"acbd18db4cc2f85cedef654fccc4a4d8+3", "73feffa4b7f6bb68e44cf984c85f6e88+3"})

	coll = Collection{ManifestText: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n. acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:bar\n. 73feffa4b7f6bb68e44cf984c85f6e88+3+Z+K@xyzzy 0:3:baz\n"}
	sd, err = coll.SizedDigests()
	c.Check(err, check.IsNil)
	c.Check(sd, check.DeepEquals, []SizedDigest{"acbd18db4cc2f85cedef654fccc4a4d8+3", "73feffa4b7f6bb68e44cf984c85f6e88+3"})

	coll = Collection{ManifestText: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n"}
	sd, err = coll.SizedDigests()
	c.Check(err, check.IsNil)
	c.Check(sd, check.HasLen, 0)

	coll = Collection{ManifestText: "", PortableDataHash: "d41d8cd98f00b204e9800998ecf8427e+0"}
	sd, err = coll.SizedDigests()
	c.Check(err, check.IsNil)
	c.Check(sd, check.HasLen, 0)
}
