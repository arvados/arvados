// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/ctrlctx"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollectionSuite{})

type CollectionSuite struct {
	localdbSuite
}

func (s *CollectionSuite) TestCollectionCreateAndUpdateWithProperties(c *check.C) {
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

		// Create with properties
		coll, err := s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{
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
		coll, err = s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{})
		c.Assert(err, check.IsNil)
		coll, err = s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
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
	adminctx := ctrlctx.NewWithToken(s.ctx, s.cluster, arvadostest.AdminToken)
	foo, err := s.localdb.railsProxy.CollectionCreate(adminctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt\n",
		}})
	c.Assert(err, check.IsNil)
	s.localdb.signCollection(adminctx, &foo)
	foobarbaz, err := s.localdb.railsProxy.CollectionCreate(adminctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": "./foo/bar 73feffa4b7f6bb68e44cf984c85f6e88+3 0:3:baz.txt\n",
		}})
	c.Assert(err, check.IsNil)
	s.localdb.signCollection(adminctx, &foobarbaz)
	wazqux, err := s.localdb.railsProxy.CollectionCreate(adminctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": "./waz d85b1213473c2fd7c2045020a6b9c62b+3 0:3:qux.txt\n",
		}})
	c.Assert(err, check.IsNil)
	s.localdb.signCollection(adminctx, &wazqux)

	// Create using content from existing collections
	dst, err := s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{
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
	dst, err = s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
		UUID: dst.UUID,
		ReplaceFiles: map[string]string{
			"/f":     "",
			"/q/waz": "",
		}})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, dst, "b/baz.txt", "q/", "w/qux.txt")

	// Move and copy content within collection
	dst, err = s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
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
	dst, err = s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
		UUID: dst.UUID,
		ReplaceFiles: map[string]string{
			"/":            "",
			"/b/corge.txt": dst.PortableDataHash + "/b/corge.txt",
		}})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, dst, "b/corge.txt")

	// Copy entire collection to root
	dstcopy, err := s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{
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
		_, err = s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
			UUID:         dst.UUID,
			ReplaceFiles: badrepl,
		})
		c.Logf("badrepl %#v\n... got err: %s", badrepl, err)
		c.Check(err, check.NotNil)
	}

	// Check conflicting replace_files and manifest_text
	_, err = s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
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
	cfs, err := coll.FileSystem(client, kc)
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

// Until #21701 it's hard to test from the outside whether the
// uuid_lock mechanism is effectively serializing concurrent
// replace_files updates to a single collection.  For now, we're
// really just checking that it doesn't cause updates to deadlock or
// anything like that.
func (s *CollectionSuite) TestCollectionUpdateLock(c *check.C) {
	adminctx := ctrlctx.NewWithToken(s.ctx, s.cluster, arvadostest.AdminToken)
	foo, err := s.localdb.railsProxy.CollectionCreate(adminctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt\n",
		}})
	c.Assert(err, check.IsNil)
	dst, err := s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{
		ReplaceFiles: map[string]string{
			"/foo.txt": foo.PortableDataHash + "/foo.txt",
		},
		Attrs: map[string]interface{}{
			"owner_uuid": arvadostest.ActiveUserUUID,
		}})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, dst, "foo.txt")

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		name1, name2 := "a", "b"
		if i&1 == 1 {
			name1, name2 = "b", "a"
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			upd, err := s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
				UUID: dst.UUID,
				ReplaceFiles: map[string]string{
					"/" + name1: foo.PortableDataHash + "/foo.txt",
					"/" + name2: "",
					"/foo.txt":  "",
				}})
			c.Assert(err, check.IsNil)
			s.expectFiles(c, upd, name1)
		}()
	}
	wg.Wait()
}

func (s *CollectionSuite) TestSignatures(c *check.C) {
	resp, err := s.localdb.CollectionGet(s.userctx, arvados.GetOptions{UUID: arvadostest.FooCollection})
	c.Check(err, check.IsNil)
	c.Check(resp.ManifestText, check.Matches, `(?ms).* acbd[^ ]*\+3\+A[0-9a-f]+@[0-9a-f]+ 0:.*`)
	s.checkSignatureExpiry(c, resp.ManifestText, time.Hour*24*7*2)

	resp, err = s.localdb.CollectionGet(s.userctx, arvados.GetOptions{UUID: arvadostest.FooCollection, Select: []string{"manifest_text"}})
	c.Check(err, check.IsNil)
	c.Check(resp.ManifestText, check.Matches, `(?ms).* acbd[^ ]*\+3\+A[0-9a-f]+@[0-9a-f]+ 0:.*`)

	lresp, err := s.localdb.CollectionList(s.userctx, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", arvadostest.FooCollection}}})
	c.Check(err, check.IsNil)
	if c.Check(lresp.Items, check.HasLen, 1) {
		c.Check(lresp.Items[0].UUID, check.Equals, arvadostest.FooCollection)
		c.Check(lresp.Items[0].ManifestText, check.Equals, "")
		c.Check(lresp.Items[0].UnsignedManifestText, check.Equals, "")
	}

	lresp, err = s.localdb.CollectionList(s.userctx, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", arvadostest.FooCollection}}, Select: []string{"manifest_text"}})
	c.Check(err, check.IsNil)
	if c.Check(lresp.Items, check.HasLen, 1) {
		c.Check(lresp.Items[0].ManifestText, check.Matches, `(?ms).* acbd[^ ]*\+3\+A[0-9a-f]+@[0-9a-f]+ 0:.*`)
		c.Check(lresp.Items[0].UnsignedManifestText, check.Equals, "")
	}

	lresp, err = s.localdb.CollectionList(s.userctx, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"uuid", "=", arvadostest.FooCollection}}, Select: []string{"unsigned_manifest_text"}})
	c.Check(err, check.IsNil)
	if c.Check(lresp.Items, check.HasLen, 1) {
		c.Check(lresp.Items[0].ManifestText, check.Equals, "")
		c.Check(lresp.Items[0].UnsignedManifestText, check.Matches, `(?ms).* acbd[^ ]*\+3 0:.*`)
	}

	// early trash date causes lower signature TTL (even if
	// trash_at and is_trashed fields are unselected)
	trashed, err := s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{
		Select: []string{"uuid", "manifest_text"},
		Attrs: map[string]interface{}{
			"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n",
			"trash_at":      time.Now().UTC().Add(time.Hour),
		}})
	c.Assert(err, check.IsNil)
	s.checkSignatureExpiry(c, trashed.ManifestText, time.Hour)
	resp, err = s.localdb.CollectionGet(s.userctx, arvados.GetOptions{UUID: trashed.UUID})
	c.Assert(err, check.IsNil)
	s.checkSignatureExpiry(c, resp.ManifestText, time.Hour)

	// distant future trash date does not cause higher signature TTL
	trashed, err = s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
		UUID: trashed.UUID,
		Attrs: map[string]interface{}{
			"trash_at": time.Now().UTC().Add(time.Hour * 24 * 365),
		}})
	c.Assert(err, check.IsNil)
	s.checkSignatureExpiry(c, trashed.ManifestText, time.Hour*24*7*2)
	resp, err = s.localdb.CollectionGet(s.userctx, arvados.GetOptions{UUID: trashed.UUID})
	c.Assert(err, check.IsNil)
	s.checkSignatureExpiry(c, resp.ManifestText, time.Hour*24*7*2)

	// Make sure groups/contents doesn't return manifest_text with
	// collections (if it did, we'd need to sign it).
	gresp, err := s.localdb.GroupContents(s.userctx, arvados.GroupContentsOptions{
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
	resp, err := s.localdb.CollectionGet(s.userctx, arvados.GetOptions{UUID: arvadostest.FooCollection})
	c.Check(err, check.IsNil)
	c.Check(resp.ManifestText, check.Matches, `(?ms).* acbd[^ +]*\+3 0:.*`)
}
