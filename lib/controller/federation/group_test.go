// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"errors"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&GroupSuite{})

type GroupSuite struct {
	FederationSuite
}

func makeConn() (*Conn, *arvadostest.APIStub, *arvadostest.APIStub) {
	localAPIstub := &arvadostest.APIStub{Error: errors.New("No result")}
	remoteAPIstub := &arvadostest.APIStub{Error: errors.New("No result")}
	return &Conn{context.Background(), &arvados.Cluster{ClusterID: "local"}, localAPIstub, map[string]backend{"zzzzz": remoteAPIstub}}, localAPIstub, remoteAPIstub
}

func (s *UserSuite) TestGroupContents(c *check.C) {
	conn, localAPIstub, remoteAPIstub := makeConn()
	conn.GroupContents(s.ctx, arvados.GroupContentsOptions{UUID: "local-tpzed-xurymjxw79nv3jz"})
	c.Check(len(localAPIstub.Calls(nil)), check.Equals, 1)
	c.Check(len(remoteAPIstub.Calls(nil)), check.Equals, 0)

	conn, localAPIstub, remoteAPIstub = makeConn()
	conn.GroupContents(s.ctx, arvados.GroupContentsOptions{UUID: "zzzzz-tpzed-xurymjxw79nv3jz"})
	c.Check(len(localAPIstub.Calls(nil)), check.Equals, 1)
	c.Check(len(remoteAPIstub.Calls(nil)), check.Equals, 0)

	conn, localAPIstub, remoteAPIstub = makeConn()
	conn.GroupContents(s.ctx, arvados.GroupContentsOptions{UUID: "local-j7d0g-xurymjxw79nv3jz"})
	c.Check(len(localAPIstub.Calls(nil)), check.Equals, 1)
	c.Check(len(remoteAPIstub.Calls(nil)), check.Equals, 0)

	conn, localAPIstub, remoteAPIstub = makeConn()
	conn.GroupContents(s.ctx, arvados.GroupContentsOptions{UUID: "zzzzz-j7d0g-xurymjxw79nv3jz"})
	c.Check(len(localAPIstub.Calls(nil)), check.Equals, 0)
	c.Check(len(remoteAPIstub.Calls(nil)), check.Equals, 1)

	conn, localAPIstub, remoteAPIstub = makeConn()
	conn.GroupContents(s.ctx, arvados.GroupContentsOptions{UUID: "zzzzz-tpzed-xurymjxw79nv3jz", ClusterID: "zzzzz"})
	c.Check(len(localAPIstub.Calls(nil)), check.Equals, 0)
	c.Check(len(remoteAPIstub.Calls(nil)), check.Equals, 1)
}
