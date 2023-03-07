// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&LinkSuite{})

type LinkSuite struct {
	localdbSuite
}

func (s *LinkSuite) TestLinkCreateWithProperties(c *check.C) {
	s.setUpVocabulary(c, "")

	tests := []struct {
		name    string
		props   map[string]interface{}
		success bool
	}{
		{"Invalid prop key", map[string]interface{}{"Priority": "IDVALIMPORTANCES1"}, false},
		{"Invalid prop value", map[string]interface{}{"IDTAGIMPORTANCES": "high"}, false},
		{"Valid prop key & value", map[string]interface{}{"IDTAGIMPORTANCES": "IDVALIMPORTANCES1"}, true},
		{"Empty properties", map[string]interface{}{}, true},
	}
	for _, tt := range tests {
		c.Log(c.TestName()+" ", tt.name)

		lnk, err := s.localdb.LinkCreate(s.userctx, arvados.CreateOptions{
			Select: []string{"uuid", "properties"},
			Attrs: map[string]interface{}{
				"link_class": "star",
				"tail_uuid":  "zzzzz-j7d0g-publicfavorites",
				"head_uuid":  arvadostest.FooCollection,
				"properties": tt.props,
			}})
		if tt.success {
			c.Assert(err, check.IsNil)
			c.Assert(lnk.Properties, check.DeepEquals, tt.props)
		} else {
			c.Assert(err, check.NotNil)
		}
	}
}

func (s *LinkSuite) TestLinkUpdateWithProperties(c *check.C) {
	s.setUpVocabulary(c, "")

	tests := []struct {
		name    string
		props   map[string]interface{}
		success bool
	}{
		{"Invalid prop key", map[string]interface{}{"Priority": "IDVALIMPORTANCES1"}, false},
		{"Invalid prop value", map[string]interface{}{"IDTAGIMPORTANCES": "high"}, false},
		{"Valid prop key & value", map[string]interface{}{"IDTAGIMPORTANCES": "IDVALIMPORTANCES1"}, true},
		{"Empty properties", map[string]interface{}{}, true},
	}
	for _, tt := range tests {
		c.Log(c.TestName()+" ", tt.name)
		lnk, err := s.localdb.LinkCreate(s.userctx, arvados.CreateOptions{
			Attrs: map[string]interface{}{
				"link_class": "star",
				"tail_uuid":  "zzzzz-j7d0g-publicfavorites",
				"head_uuid":  arvadostest.FooCollection,
			},
		})
		c.Assert(err, check.IsNil)
		lnk, err = s.localdb.LinkUpdate(s.userctx, arvados.UpdateOptions{
			UUID:   lnk.UUID,
			Select: []string{"uuid", "properties"},
			Attrs: map[string]interface{}{
				"properties": tt.props,
			}})
		if tt.success {
			c.Assert(err, check.IsNil)
			c.Assert(lnk.Properties, check.DeepEquals, tt.props)
		} else {
			c.Assert(err, check.NotNil)
		}
	}
}
