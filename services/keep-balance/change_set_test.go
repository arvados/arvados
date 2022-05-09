// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepbalance

import (
	"encoding/json"

	"git.arvados.org/arvados.git/sdk/go/arvados"

	check "gopkg.in/check.v1"
)

var _ = check.Suite(&changeSetSuite{})

type changeSetSuite struct{}

func (s *changeSetSuite) TestJSONFormat(c *check.C) {
	mnt := &KeepMount{
		KeepMount: arvados.KeepMount{
			UUID: "zzzzz-mount-abcdefghijklmno"}}
	srv := &KeepService{
		KeepService: arvados.KeepService{
			UUID:           "zzzzz-bi6l4-000000000000001",
			ServiceType:    "disk",
			ServiceSSLFlag: false,
			ServiceHost:    "keep1.zzzzz.arvadosapi.com",
			ServicePort:    25107}}

	buf, err := json.Marshal([]Pull{{
		SizedDigest: arvados.SizedDigest("acbd18db4cc2f85cedef654fccc4a4d8+3"),
		To:          mnt,
		From:        srv}})
	c.Check(err, check.IsNil)
	c.Check(string(buf), check.Equals, `[{"locator":"acbd18db4cc2f85cedef654fccc4a4d8","servers":["http://keep1.zzzzz.arvadosapi.com:25107"],"mount_uuid":"zzzzz-mount-abcdefghijklmno"}]`)

	buf, err = json.Marshal([]Trash{{
		SizedDigest: arvados.SizedDigest("acbd18db4cc2f85cedef654fccc4a4d8+3"),
		From:        mnt,
		Mtime:       123456789}})
	c.Check(err, check.IsNil)
	c.Check(string(buf), check.Equals, `[{"locator":"acbd18db4cc2f85cedef654fccc4a4d8","block_mtime":123456789,"mount_uuid":"zzzzz-mount-abcdefghijklmno"}]`)
}
