// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"context"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
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
	s.cluster.API.VocabularyPath = "foo"
	s.localdb.vocabularyCache = voc
}

func (s *CollectionSuite) TestCollectionCreateAndUpdateWithProperties(c *check.C) {
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

		// Create with properties
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

		// Create, then update with properties
		coll, err = s.localdb.CollectionCreate(ctx, arvados.CreateOptions{})
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

func (s *CollectionSuite) TestCollectionReplaceFiles(c *check.C) {
	ctx := auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.AdminToken}})
	foo, err := s.localdb.railsProxy.CollectionCreate(ctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt\n",
		}})
	c.Assert(err, check.IsNil)
	s.localdb.signCollection(ctx, &foo)
	foobarbaz, err := s.localdb.railsProxy.CollectionCreate(ctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": "./foo/bar 73feffa4b7f6bb68e44cf984c85f6e88+3 0:3:baz.txt\n",
		}})
	c.Assert(err, check.IsNil)
	s.localdb.signCollection(ctx, &foobarbaz)
	wazqux, err := s.localdb.railsProxy.CollectionCreate(ctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": "./waz d85b1213473c2fd7c2045020a6b9c62b+3 0:3:qux.txt\n",
		}})
	c.Assert(err, check.IsNil)
	s.localdb.signCollection(ctx, &wazqux)

	ctx = auth.NewContext(context.Background(), &auth.Credentials{Tokens: []string{arvadostest.ActiveTokenV2}})

	// Create using content from existing collections
	dst, err := s.localdb.CollectionCreate(ctx, arvados.CreateOptions{
		ReplaceFiles: map[string]string{
			"/f": foo.PortableDataHash + "/foo.txt",
			"/b": foobarbaz.PortableDataHash + "/foo/bar",
			"/q": wazqux.PortableDataHash + "/",
			"/w": wazqux.PortableDataHash + "/waz",
		},
		Attrs: map[string]interface{}{
			"owner_uuid": arvadostest.ActiveUserUUID,
		}})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, dst, "f", "b/baz.txt", "q/waz/qux.txt", "w/qux.txt")

	// Delete a file and a directory
	dst, err = s.localdb.CollectionUpdate(ctx, arvados.UpdateOptions{
		UUID: dst.UUID,
		ReplaceFiles: map[string]string{
			"/f":     "",
			"/q/waz": "",
		}})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, dst, "b/baz.txt", "q/", "w/qux.txt")

	// Move and copy content within collection
	dst, err = s.localdb.CollectionUpdate(ctx, arvados.UpdateOptions{
		UUID: dst.UUID,
		ReplaceFiles: map[string]string{
			// Note splicing content to /b/corge.txt but
			// removing everything else from /b
			"/b":              "",
			"/b/corge.txt":    dst.PortableDataHash + "/b/baz.txt",
			"/quux/corge.txt": dst.PortableDataHash + "/b/baz.txt",
		}})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, dst, "b/corge.txt", "q/", "w/qux.txt", "quux/corge.txt")

	// Remove everything except one file
	dst, err = s.localdb.CollectionUpdate(ctx, arvados.UpdateOptions{
		UUID: dst.UUID,
		ReplaceFiles: map[string]string{
			"/":            "",
			"/b/corge.txt": dst.PortableDataHash + "/b/corge.txt",
		}})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, dst, "b/corge.txt")

	// Copy entire collection to root
	dstcopy, err := s.localdb.CollectionCreate(ctx, arvados.CreateOptions{
		ReplaceFiles: map[string]string{
			"/": dst.PortableDataHash,
		}})
	c.Check(err, check.IsNil)
	c.Check(dstcopy.PortableDataHash, check.Equals, dst.PortableDataHash)
	s.expectFiles(c, dstcopy, "b/corge.txt")

	// Check invalid targets, sources, and combinations
	for _, badrepl := range []map[string]string{
		{
			"/foo/nope": dst.PortableDataHash + "/b",
			"/foo":      dst.PortableDataHash + "/b",
		},
		{
			"/foo":      dst.PortableDataHash + "/b",
			"/foo/nope": "",
		},
		{
			"/":     dst.PortableDataHash + "/",
			"/nope": "",
		},
		{
			"/":     dst.PortableDataHash + "/",
			"/nope": dst.PortableDataHash + "/b",
		},
		{"/bad/": ""},
		{"/./bad": ""},
		{"/b/./ad": ""},
		{"/b/../ad": ""},
		{"/b/.": ""},
		{".": ""},
		{"bad": ""},
		{"": ""},
		{"/bad": "/b"},
		{"/bad": "bad/b"},
		{"/bad": dst.UUID + "/b"},
	} {
		_, err = s.localdb.CollectionUpdate(ctx, arvados.UpdateOptions{
			UUID:         dst.UUID,
			ReplaceFiles: badrepl,
		})
		c.Logf("badrepl %#v\n... got err: %s", badrepl, err)
		c.Check(err, check.NotNil)
	}

	// Check conflicting replace_files and manifest_text
	_, err = s.localdb.CollectionUpdate(ctx, arvados.UpdateOptions{
		UUID:         dst.UUID,
		ReplaceFiles: map[string]string{"/": ""},
		Attrs: map[string]interface{}{
			"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:z\n",
		}})
	c.Logf("replace_files+manifest_text\n... got err: %s", err)
	c.Check(err, check.ErrorMatches, "ambiguous request: both.*replace_files.*manifest_text.*")
}

// expectFiles checks coll's directory structure against the given
// list of expected files and empty directories. An expected path with
// a trailing slash indicates an empty directory.
func (s *CollectionSuite) expectFiles(c *check.C, coll arvados.Collection, expected ...string) {
	client := arvados.NewClientFromEnv()
	ac, err := arvadosclient.New(client)
	c.Assert(err, check.IsNil)
	kc, err := keepclient.MakeKeepClient(ac)
	c.Assert(err, check.IsNil)
	cfs, err := coll.FileSystem(arvados.NewClientFromEnv(), kc)
	c.Assert(err, check.IsNil)
	var found []string
	nonemptydirs := map[string]bool{}
	fs.WalkDir(arvados.FS(cfs), "/", func(path string, d fs.DirEntry, err error) error {
		dir, _ := filepath.Split(path)
		nonemptydirs[dir] = true
		if d.IsDir() {
			if path != "/" {
				path += "/"
			}
			if !nonemptydirs[path] {
				nonemptydirs[path] = false
			}
		} else {
			found = append(found, path)
		}
		return nil
	})
	for d, nonempty := range nonemptydirs {
		if !nonempty {
			found = append(found, d)
		}
	}
	for i, path := range found {
		if path != "/" {
			found[i] = strings.TrimPrefix(path, "/")
		}
	}
	sort.Strings(found)
	sort.Strings(expected)
	c.Check(found, check.DeepEquals, expected)
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
