// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"regexp"
	"strconv"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollectionSuite{})

type CollectionSuite struct {
	cluster  *arvados.Cluster
	localdb  *Conn
	railsSpy *arvadostest.Proxy
}

func (s *CollectionSuite) TearDownSuite(c *check.C) {
	// Undo any changes/additions to the user database so they
	// don't affect subsequent tests.
	arvadostest.ResetEnv()
	c.Check(arvados.NewClientFromEnv().RequestAndDecode(nil, "POST", "database/reset", nil, nil), check.IsNil)
}

func (s *CollectionSuite) SetUpTest(c *check.C) {
	cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
	c.Assert(err, check.IsNil)
	s.cluster, err = cfg.GetCluster("")
	c.Assert(err, check.IsNil)
	s.localdb = NewConn(s.cluster)
	s.railsSpy = arvadostest.NewProxy(c, s.cluster.Services.RailsAPI)
	*s.localdb.railsProxy = *rpc.NewConn(s.cluster.ClusterID, s.railsSpy.URL, true, rpc.PassthroughTokenProvider)
}

func (s *CollectionSuite) TearDownTest(c *check.C) {
	s.railsSpy.Close()
}

func (s *CollectionSuite) setUpVocabulary(c *check.C, testVocabulary string) {
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
	c.Assert(voc.Validate(), check.IsNil)
	s.cluster.API.VocabularyPath = "foo"
	s.localdb.vocabularyCache = voc
}

func (s *CollectionSuite) TestCollectionCreateWithProperties(c *check.C) {
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

		coll, err := s.localdb.CollectionCreate(ctx, arvados.CreateOptions{
			Select: []string{"uuid", "properties"},
			Attrs: map[string]interface{}{
				"properties": tt.props,
			}})
		if tt.success {
			c.Assert(err, check.IsNil)
			c.Assert(coll.Properties, check.DeepEquals, tt.props)
		} else {
			c.Assert(err, check.NotNil)
		}
	}
}

func (s *CollectionSuite) TestCollectionUpdateWithProperties(c *check.C) {
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
		coll, err := s.localdb.CollectionCreate(ctx, arvados.CreateOptions{})
		c.Assert(err, check.IsNil)
		coll, err = s.localdb.CollectionUpdate(ctx, arvados.UpdateOptions{
			UUID:   coll.UUID,
			Select: []string{"uuid", "properties"},
			Attrs: map[string]interface{}{
				"properties": tt.props,
			}})
		if tt.success {
			c.Assert(err, check.IsNil)
			c.Assert(coll.Properties, check.DeepEquals, tt.props)
		} else {
			c.Assert(err, check.NotNil)
		}
	}
}

func (s *CollectionSuite) TestSignatures(c *check.C) {
	ctx := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})

	resp, err := s.localdb.CollectionGet(ctx, arvados.GetOptions{UUID: arvadostest.FooCollection})
	c.Check(err, check.IsNil)
	c.Check(resp.ManifestText, check.Matches, `(?ms).* acbd[^ ]*\+3\+A[0-9a-f]+@[0-9a-f]+ 0:.*`)
	s.checkSignatureExpiry(c, resp.ManifestText, time.Hour*24*7*2)

	resp, err = s.localdb.CollectionGet(ctx, arvados.GetOptions{UUID: arvadostest.FooCollection, Select: []string{"manifest_text"}})
	c.Check(err, check.IsNil)
	c.Check(resp.ManifestText, check.Matches, `(?ms).* acbd[^ ]*\+3\+A[0-9a-f]+@[0-9a-f]+ 0:.*`)

	lresp, err := s.localdb.CollectionList(ctx, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", arvadostest.FooCollection}}})
	c.Check(err, check.IsNil)
	if c.Check(lresp.Items, check.HasLen, 1) {
		c.Check(lresp.Items[0].UUID, check.Equals, arvadostest.FooCollection)
		c.Check(lresp.Items[0].ManifestText, check.Equals, "")
		c.Check(lresp.Items[0].UnsignedManifestText, check.Equals, "")
	}

	lresp, err = s.localdb.CollectionList(ctx, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", arvadostest.FooCollection}}, Select: []string{"manifest_text"}})
	c.Check(err, check.IsNil)
	if c.Check(lresp.Items, check.HasLen, 1) {
		c.Check(lresp.Items[0].ManifestText, check.Matches, `(?ms).* acbd[^ ]*\+3\+A[0-9a-f]+@[0-9a-f]+ 0:.*`)
		c.Check(lresp.Items[0].UnsignedManifestText, check.Equals, "")
	}

	lresp, err = s.localdb.CollectionList(ctx, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", arvadostest.FooCollection}}, Select: []string{"unsigned_manifest_text"}})
	c.Check(err, check.IsNil)
	if c.Check(lresp.Items, check.HasLen, 1) {
		c.Check(lresp.Items[0].ManifestText, check.Equals, "")
		c.Check(lresp.Items[0].UnsignedManifestText, check.Matches, `(?ms).* acbd[^ ]*\+3 0:.*`)
	}

	// early trash date causes lower signature TTL (even if
	// trash_at and is_trashed fields are unselected)
	trashed, err := s.localdb.CollectionCreate(ctx, arvados.CreateOptions{
		Select: []string{"uuid", "manifest_text"},
		Attrs: map[string]interface{}{
			"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
			"trash_at":      time.Now().UTC().Add(time.Hour),
		}})
	c.Assert(err, check.IsNil)
	s.checkSignatureExpiry(c, trashed.ManifestText, time.Hour)
	resp, err = s.localdb.CollectionGet(ctx, arvados.GetOptions{UUID: trashed.UUID})
	c.Assert(err, check.IsNil)
	s.checkSignatureExpiry(c, resp.ManifestText, time.Hour)

	// distant future trash date does not cause higher signature TTL
	trashed, err = s.localdb.CollectionUpdate(ctx, arvados.UpdateOptions{
		UUID: trashed.UUID,
		Attrs: map[string]interface{}{
			"trash_at": time.Now().UTC().Add(time.Hour * 24 * 365),
		}})
	c.Assert(err, check.IsNil)
	s.checkSignatureExpiry(c, trashed.ManifestText, time.Hour*24*7*2)
	resp, err = s.localdb.CollectionGet(ctx, arvados.GetOptions{UUID: trashed.UUID})
	c.Assert(err, check.IsNil)
	s.checkSignatureExpiry(c, resp.ManifestText, time.Hour*24*7*2)

	// Make sure groups/contents doesn't return manifest_text with
	// collections (if it did, we'd need to sign it).
	gresp, err := s.localdb.GroupContents(ctx, arvados.GroupContentsOptions{
		Limit:   -1,
		Filters: []arvados.Filter{{"uuid", "=", arvadostest.FooCollection}},
		Select:  []string{"uuid", "manifest_text"},
	})
	if err != nil {
		c.Check(err, check.ErrorMatches, `.*Invalid attribute.*manifest_text.*`)
	} else if c.Check(gresp.Items, check.HasLen, 1) {
		c.Check(gresp.Items[0].(map[string]interface{})["uuid"], check.Equals, arvadostest.FooCollection)
		c.Check(gresp.Items[0].(map[string]interface{})["manifest_text"], check.Equals, nil)
	}
}

func (s *CollectionSuite) checkSignatureExpiry(c *check.C, manifestText string, expectedTTL time.Duration) {
	m := regexp.MustCompile(`@([[:xdigit:]]+)`).FindStringSubmatch(manifestText)
	c.Assert(m, check.HasLen, 2)
	sigexp, err := strconv.ParseInt(m[1], 16, 64)
	c.Assert(err, check.IsNil)
	expectedExp := time.Now().Add(expectedTTL).Unix()
	c.Check(sigexp > expectedExp-60, check.Equals, true)
	c.Check(sigexp <= expectedExp, check.Equals, true)
}

func (s *CollectionSuite) TestSignaturesDisabled(c *check.C) {
	s.localdb.cluster.Collections.BlobSigning = false
	ctx := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})

	resp, err := s.localdb.CollectionGet(ctx, arvados.GetOptions{UUID: arvadostest.FooCollection})
	c.Check(err, check.IsNil)
	c.Check(resp.ManifestText, check.Matches, `(?ms).* acbd[^ +]*\+3 0:.*`)
}
