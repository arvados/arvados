// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&ContainerRequestSuite{})

type ContainerRequestSuite struct {
	localdbSuite
}

func (s *ContainerRequestSuite) TestCRCreateWithProperties(c *check.C) {
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

		cnt, err := s.localdb.ContainerRequestCreate(ctx, arvados.CreateOptions{
			Select: []string{"uuid", "properties"},
			Attrs: map[string]interface{}{
				"command":         []string{"echo", "foo"},
				"container_image": "arvados/apitestfixture:latest",
				"cwd":             "/tmp",
				"environment":     map[string]string{},
				"mounts": map[string]interface{}{
					"/out": map[string]interface{}{
						"kind":     "tmp",
						"capacity": 1000000,
					},
				},
				"output_path": "/out",
				"runtime_constraints": map[string]interface{}{
					"vcpus": 1,
					"ram":   2,
				},
				"properties": tt.props,
			}})
		if tt.success {
			c.Assert(err, check.IsNil)
			c.Assert(cnt.Properties, check.DeepEquals, tt.props)
		} else {
			c.Assert(err, check.NotNil)
		}
	}
}

func (s *ContainerRequestSuite) TestCRUpdateWithProperties(c *check.C) {
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
		cnt, err := s.localdb.ContainerRequestCreate(ctx, arvados.CreateOptions{
			Attrs: map[string]interface{}{
				"command":         []string{"echo", "foo"},
				"container_image": "arvados/apitestfixture:latest",
				"cwd":             "/tmp",
				"environment":     map[string]string{},
				"mounts": map[string]interface{}{
					"/out": map[string]interface{}{
						"kind":     "tmp",
						"capacity": 1000000,
					},
				},
				"output_path": "/out",
				"runtime_constraints": map[string]interface{}{
					"vcpus": 1,
					"ram":   2,
				},
			},
		})
		c.Assert(err, check.IsNil)
		cnt, err = s.localdb.ContainerRequestUpdate(ctx, arvados.UpdateOptions{
			UUID:   cnt.UUID,
			Select: []string{"uuid", "properties"},
			Attrs: map[string]interface{}{
				"properties": tt.props,
			}})
		if tt.success {
			c.Assert(err, check.IsNil)
			c.Assert(cnt.Properties, check.DeepEquals, tt.props)
		} else {
			c.Assert(err, check.NotNil)
		}
	}
}
