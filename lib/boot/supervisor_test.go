// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"net"
	"testing"

	check "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	check.TestingT(t)
}

type supervisorSuite struct{}

var _ = check.Suite(&supervisorSuite{})

func (s *supervisorSuite) TestAddrIsLocal(c *check.C) {
	is, err := addrIsLocal("0.0.0.0:0")
	c.Check(err, check.IsNil)
	c.Check(is, check.Equals, true)

	is, err = addrIsLocal("127.0.0.1:9")
	c.Check(err, check.IsNil)
	c.Check(is, check.Equals, true)

	is, err = addrIsLocal("127.0.0.127:32767")
	c.Check(err, check.IsNil)
	c.Check(is, check.Equals, true)

	is, err = addrIsLocal("[::1]:32767")
	c.Check(err, check.IsNil)
	c.Check(is, check.Equals, true)

	is, err = addrIsLocal("8.8.8.8:32767")
	c.Check(err, check.IsNil)
	c.Check(is, check.Equals, false)

	is, err = addrIsLocal("example.com:32767")
	c.Check(err, check.IsNil)
	c.Check(is, check.Equals, false)

	is, err = addrIsLocal("1.2.3.4.5:32767")
	c.Check(err, check.NotNil)

	ln, err := net.Listen("tcp", ":")
	c.Assert(err, check.IsNil)
	defer ln.Close()
	is, err = addrIsLocal(ln.Addr().String())
	c.Check(err, check.IsNil)
	c.Check(is, check.Equals, true)

}
