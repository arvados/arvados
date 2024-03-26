// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"testing"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/arvados/cgofuse/fuse"
	check "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}

var _ = check.Suite(&FSSuite{})

type FSSuite struct{}

func (*FSSuite) TestFuseInterface(c *check.C) {
	var _ fuse.FileSystemInterface = &keepFS{}
}

func (*FSSuite) TestOpendir(c *check.C) {
	client := arvados.NewClientFromEnv()
	ac, err := arvadosclient.New(client)
	c.Assert(err, check.IsNil)
	kc, err := keepclient.MakeKeepClient(ac)
	c.Assert(err, check.IsNil)

	var fs fuse.FileSystemInterface = &keepFS{
		Client:     client,
		KeepClient: kc,
		Logger:     ctxlog.TestLogger(c),
	}
	fs.Init()
	errc, fh := fs.Opendir("/by_id")
	c.Check(errc, check.Equals, 0)
	c.Check(fh, check.Not(check.Equals), uint64(0))
	c.Check(fh, check.Not(check.Equals), invalidFH)
	errc, fh = fs.Opendir("/bogus")
	c.Check(errc, check.Equals, -fuse.ENOENT)
	c.Check(fh, check.Equals, invalidFH)
}
