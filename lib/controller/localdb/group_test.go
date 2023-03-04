// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&GroupSuite{})

type GroupSuite struct {
	cluster  *arvados.Cluster
	localdb  *Conn
	railsSpy *arvadostest.Proxy
}

func (s *GroupSuite) SetUpSuite(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.localdb = NewConn(context.Background(), s.cluster, (&ctrlctx.DBConnector{PostgreSQL: s.cluster.PostgreSQL}).GetDB)
	s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	*s.localdb.railsProxy = *rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)
}

func (s *GroupSuite) TearDownSuite(c *check.C) {
	s.railsSpy.Close()
	// Undo any changes/additions to the user database so they
	// don't affect subsequent tests.
	arvadostest.ResetEnv()
	c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
}

func (s *GroupSuite) setUpVocabulary(c *check.C, testVocabulary string) {
	if testVocabulary == "" {
		testVocabulary = `{
			"strict_tags": false,
			"tags": {
				"IDTAGIMPORTANCES": {
					"strict": true,
					"labels": [{"label": "Importance"}, {"label": "Priority"}],
					"values": {
						"IDVALIMPORTANCES1": { "labels": [{"label": "Critical"}, {"label": "Urgent"}, {"label": "High"}] },
						"IDVALIMPORTANCES2": { "labels": [{"label": "Normal"}, {"label": "Moderate"}] },
						"IDVALIMPORTANCES3": { "labels": [{"label": "Low"}] }
					}
				}
			}
		}`
	}
	voc, err := arvados.NewVocabulary([]byte(testVocabulary), []string{})
	c.Assert(err, check.IsNil)
	s.localdb.vocabularyCache = voc
	s.cluster.API.VocabularyPath = "foo"
}

func (s *GroupSuite) TestGroupCreateWithProperties(c *check.C) {
	s.setUpVocabulary(c, "")
	ctx := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})

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

		grp, err := s.localdb.GroupCreate(ctx, arvados.CreateOptions{
			Select: []string{"uuid", "properties"},
			Attrs: map[string]interface{}{
				"group_class": "project",
				"properties":  tt.props,
			}})
		if tt.success {
			c.Assert(err, check.IsNil)
			c.Assert(grp.Properties, check.DeepEquals, tt.props)
		} else {
			c.Assert(err, check.NotNil)
		}
	}
}

func (s *GroupSuite) TestGroupUpdateWithProperties(c *check.C) {
	s.setUpVocabulary(c, "")
	ctx := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})

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
		grp, err := s.localdb.GroupCreate(ctx, arvados.CreateOptions{
			Attrs: map[string]interface{}{
				"group_class": "project",
			},
		})
		c.Assert(err, check.IsNil)
		grp, err = s.localdb.GroupUpdate(ctx, arvados.UpdateOptions{
			UUID:   grp.UUID,
			Select: []string{"uuid", "properties"},
			Attrs: map[string]interface{}{
				"properties": tt.props,
			}})
		if tt.success {
			c.Assert(err, check.IsNil)
			c.Assert(grp.Properties, check.DeepEquals, tt.props)
		} else {
			c.Assert(err, check.NotNil)
		}
	}
}

func (s *GroupSuite) TestCanWriteCanManageResponses(c *check.C) {
	ctxUser1 := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})
	ctxUser2 := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.SpectatorToken}})
	ctxAdmin := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.AdminToken}})
	project, err := s.localdb.GroupCreate(ctxUser1, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"group_class": "project",
		},
	})
	c.Assert(err, check.IsNil)
	c.Check(project.CanWrite, check.Equals, true)
	c.Check(project.CanManage, check.Equals, true)

	subproject, err := s.localdb.GroupCreate(ctxUser1, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":  project.UUID,
			"group_class": "project",
		},
	})
	c.Assert(err, check.IsNil)
	c.Check(subproject.CanWrite, check.Equals, true)
	c.Check(subproject.CanManage, check.Equals, true)

	projlist, err := s.localdb.GroupList(ctxUser1, arvados.ListOptions{
		Limit:   -1,
		Filters: []arvados.Filter{{"uuid", "in", []string{project.UUID, subproject.UUID}}},
	})
	c.Assert(err, check.IsNil)
	c.Assert(projlist.Items, check.HasLen, 2)
	for _, p := range projlist.Items {
		c.Check(p.CanWrite, check.Equals, true)
		c.Check(p.CanManage, check.Equals, true)
	}

	// Give 2nd user permission to read
	permlink, err := s.localdb.LinkCreate(ctxAdmin, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"link_class": "permission",
			"name":       "can_read",
			"tail_uuid":  arvadostest.SpectatorUserUUID,
			"head_uuid":  project.UUID,
		},
	})
	c.Assert(err, check.IsNil)

	// As 2nd user: can read, cannot manage, cannot write
	project2, err := s.localdb.GroupGet(ctxUser2, arvados.GetOptions{UUID: project.UUID})
	c.Assert(err, check.IsNil)
	c.Check(project2.CanWrite, check.Equals, false)
	c.Check(project2.CanManage, check.Equals, false)

	_, err = s.localdb.LinkUpdate(ctxAdmin, arvados.UpdateOptions{
		UUID: permlink.UUID,
		Attrs: map[string]interface{}{
			"name": "can_write",
		},
	})
	c.Assert(err, check.IsNil)

	// As 2nd user: cannot manage, can write
	project2, err = s.localdb.GroupGet(ctxUser2, arvados.GetOptions{UUID: project.UUID})
	c.Assert(err, check.IsNil)
	c.Check(project2.CanWrite, check.Equals, true)
	c.Check(project2.CanManage, check.Equals, false)

	// As owner: after freezing, can manage (owner), cannot write (frozen)
	project, err = s.localdb.GroupUpdate(ctxUser1, arvados.UpdateOptions{
		UUID: project.UUID,
		Attrs: map[string]interface{}{
			"frozen_by_uuid": arvadostest.ActiveUserUUID,
		}})
	c.Assert(err, check.IsNil)
	c.Check(project.CanWrite, check.Equals, false)
	c.Check(project.CanManage, check.Equals, true)

	// As admin: can manage (admin), cannot write (frozen)
	project, err = s.localdb.GroupGet(ctxAdmin, arvados.GetOptions{UUID: project.UUID})
	c.Assert(err, check.IsNil)
	c.Check(project.CanWrite, check.Equals, false)
	c.Check(project.CanManage, check.Equals, true)

	// As 2nd user: cannot manage (perm), cannot write (frozen)
	project2, err = s.localdb.GroupGet(ctxUser2, arvados.GetOptions{UUID: project.UUID})
	c.Assert(err, check.IsNil)
	c.Check(project2.CanWrite, check.Equals, false)
	c.Check(project2.CanManage, check.Equals, false)

	// After upgrading perm to "manage", as 2nd user: can manage (perm), cannot write (frozen)
	_, err = s.localdb.LinkUpdate(ctxAdmin, arvados.UpdateOptions{
		UUID: permlink.UUID,
		Attrs: map[string]interface{}{
			"name": "can_manage",
		},
	})
	c.Assert(err, check.IsNil)
	project2, err = s.localdb.GroupGet(ctxUser2, arvados.GetOptions{UUID: project.UUID})
	c.Assert(err, check.IsNil)
	c.Check(project2.CanWrite, check.Equals, false)
	c.Check(project2.CanManage, check.Equals, true)

	// 2nd user can also manage (but not write) the subject inside the frozen project
	subproject2, err := s.localdb.GroupGet(ctxUser2, arvados.GetOptions{UUID: subproject.UUID})
	c.Assert(err, check.IsNil)
	c.Check(subproject2.CanWrite, check.Equals, false)
	c.Check(subproject2.CanManage, check.Equals, true)

	u, err := s.localdb.UserGet(ctxUser1, arvados.GetOptions{
		UUID: arvadostest.ActiveUserUUID,
	})
	c.Assert(err, check.IsNil)
	c.Check(u.CanWrite, check.Equals, true)
	c.Check(u.CanManage, check.Equals, true)

	for _, selectParam := range [][]string{
		nil,
		{"can_write", "can_manage"},
	} {
		c.Logf("selectParam: %+v", selectParam)
		ulist, err := s.localdb.UserList(ctxUser1, arvados.ListOptions{
			Limit:   -1,
			Filters: []arvados.Filter{{"uuid", "=", arvadostest.ActiveUserUUID}},
			Select:  selectParam,
		})
		c.Assert(err, check.IsNil)
		c.Assert(ulist.Items, check.HasLen, 1)
		c.Logf("%+v", ulist.Items)
		for _, u := range ulist.Items {
			c.Check(u.CanWrite, check.Equals, true)
			c.Check(u.CanManage, check.Equals, true)
		}
	}
}
