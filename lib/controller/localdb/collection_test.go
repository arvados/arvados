// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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

var _ = check.Suite(&replaceFilesSuite{})

type replaceFilesSuite struct {
	CollectionSuite
	client *arvados.Client
	ac     *arvadosclient.ArvadosClient
	kc     *keepclient.KeepClient
	foo    arvados.Collection // contains /foo.txt
	tmp    arvados.Collection // working collection, initially contains /foo.txt
}

func (s *replaceFilesSuite) SetUpSuite(c *check.C) {
	s.CollectionSuite.SetUpSuite(c)
	var err error
	s.client = arvados.NewClientFromEnv()
	s.ac, err = arvadosclient.New(s.client)
	c.Assert(err, check.IsNil)
	s.kc, err = keepclient.MakeKeepClient(s.ac)
	c.Assert(err, check.IsNil)
}

func (s *replaceFilesSuite) SetUpTest(c *check.C) {
	s.CollectionSuite.SetUpTest(c)
	// Unlike most test suites, we need to COMMIT our setup --
	// otherwise, when our tests start additional
	// transactions/connections, they won't see our setup.
	ctx, txFinish := ctrlctx.New(s.ctx, s.dbConnector.GetDB)
	defer txFinish(new(error))
	adminctx := ctrlctx.NewWithToken(ctx, s.cluster, arvadostest.AdminToken)
	var err error
	s.foo, err = s.localdb.railsProxy.CollectionCreate(adminctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt\n",
		}})
	c.Assert(err, check.IsNil)
	s.tmp, err = s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{
		ReplaceFiles: map[string]string{
			"/foo.txt": s.foo.PortableDataHash + "/foo.txt",
		},
		Attrs: map[string]interface{}{
			"owner_uuid": arvadostest.ActiveUserUUID,
		}})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, s.tmp, "foo.txt")
}

func (s *replaceFilesSuite) TestCollectionReplaceFiles(c *check.C) {
	adminctx := ctrlctx.NewWithToken(s.ctx, s.cluster, arvadostest.AdminToken)
	foobarbaz, err := s.localdb.railsProxy.CollectionCreate(adminctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": "./foo/bar 73feffa4b7f6bb68e44cf984c85f6e88+3 0:3:baz.txt\n",
		}})
	c.Assert(err, check.IsNil)
	wazqux, err := s.localdb.railsProxy.CollectionCreate(adminctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"owner_uuid":    arvadostest.ActiveUserUUID,
			"manifest_text": "./waz d85b1213473c2fd7c2045020a6b9c62b+3 0:3:qux.txt\n",
		}})
	c.Assert(err, check.IsNil)

	// Create using content from existing collections
	dst, err := s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{
		ReplaceFiles: map[string]string{
			"/f": s.foo.PortableDataHash + "/foo.txt",
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
}

func (s *replaceFilesSuite) TestMultipleRename(c *check.C) {
	adminctx := ctrlctx.NewWithToken(s.ctx, s.cluster, arvadostest.AdminToken)
	tmp, err := s.localdb.CollectionUpdate(adminctx, arvados.UpdateOptions{
		UUID: s.tmp.UUID,
		Attrs: map[string]interface{}{
			"manifest_text": ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:1:file1 0:2:file2 0:3:file3\n"}})
	c.Assert(err, check.IsNil)
	tmp, err = s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
		UUID: tmp.UUID,
		ReplaceFiles: map[string]string{
			"/file1":     "current/file2",
			"/file2":     "current/file3",
			"/file3":     "current/file1",
			"/dir/file1": "current/file1",
		}})
	c.Check(err, check.IsNil)
	s.expectFileSizes(c, tmp, map[string]int64{
		"file1":     2,
		"file2":     3,
		"file3":     1,
		"dir/file1": 1,
	})
}

func (s *replaceFilesSuite) TestConcurrentCopyFromPDH(c *check.C) {
	var wg sync.WaitGroup
	var expectFiles []string
	for i := 0; i < 10; i++ {
		fnm := fmt.Sprintf("copy%d.txt", i)
		expectFiles = append(expectFiles, fnm)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, txFinish := ctrlctx.New(s.ctx, s.dbConnector.GetDB)
			defer txFinish(new(error))
			userctx := ctrlctx.NewWithToken(ctx, s.cluster, arvadostest.ActiveTokenV2)
			_, err := s.localdb.CollectionUpdate(userctx, arvados.UpdateOptions{
				UUID: s.tmp.UUID,
				ReplaceFiles: map[string]string{
					"/" + fnm:  s.foo.PortableDataHash + "/foo.txt",
					"/foo.txt": "",
				}})
			c.Check(err, check.IsNil)
		}()
	}
	wg.Wait()
	// After N concurrent/overlapping requests to add different
	// files by copying from another collection, we should see all
	// N files.
	final, err := s.localdb.CollectionGet(s.userctx, arvados.GetOptions{UUID: s.tmp.UUID})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, final, expectFiles...)
}

func (s *replaceFilesSuite) TestConcurrentCopyFromProvidedManifestText(c *check.C) {
	blockLocator := strings.Split(s.tmp.ManifestText, " ")[1]
	var wg sync.WaitGroup
	expectFileSizes := make(map[string]int64)
	for i := 0; i < 10; i++ {
		fnm := fmt.Sprintf("upload%d.txt", i)
		expectFileSizes[fnm] = 2
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, txFinish := ctrlctx.New(s.ctx, s.dbConnector.GetDB)
			defer txFinish(new(error))
			userctx := ctrlctx.NewWithToken(ctx, s.cluster, arvadostest.ActiveTokenV2)
			_, err := s.localdb.CollectionUpdate(userctx, arvados.UpdateOptions{
				UUID: s.tmp.UUID,
				Attrs: map[string]interface{}{
					"manifest_text": ". " + blockLocator + " 0:2:" + fnm + "\n",
				},
				ReplaceFiles: map[string]string{
					"/" + fnm:  "manifest_text/" + fnm,
					"/foo.txt": "",
				}})
			c.Check(err, check.IsNil)
		}()
	}
	wg.Wait()
	// After N concurrent/overlapping requests to add different
	// files, we should see all N files.
	final, err := s.localdb.CollectionGet(s.userctx, arvados.GetOptions{UUID: s.tmp.UUID})
	c.Assert(err, check.IsNil)
	s.expectFileSizes(c, final, expectFileSizes)
}

func (s *replaceFilesSuite) TestUnusedManifestText_Create(c *check.C) {
	blockLocator := strings.Split(s.tmp.ManifestText, " ")[1]
	_, err := s.localdb.CollectionCreate(s.userctx, arvados.CreateOptions{
		Attrs: map[string]interface{}{
			"manifest_text": ". " + blockLocator + " 0:3:foo\n",
		},
		ReplaceFiles: map[string]string{
			"/foo.txt": "",
		}})
	c.Check(err, check.ErrorMatches, `.*manifest_text.*would not be used.*`)
}

func (s *replaceFilesSuite) TestUnusedManifestText_Update(c *check.C) {
	blockLocator := strings.Split(s.tmp.ManifestText, " ")[1]
	_, err := s.localdb.CollectionUpdate(s.userctx, arvados.UpdateOptions{
		UUID: s.tmp.UUID,
		Attrs: map[string]interface{}{
			"manifest_text": ". " + blockLocator + " 0:3:foo\n",
		},
		ReplaceFiles: map[string]string{
			"/foo.txt": "",
		}})
	c.Check(err, check.ErrorMatches, `.*manifest_text.*would not be used.*`)
}

func (s *replaceFilesSuite) TestConcurrentRename(c *check.C) {
	var wg sync.WaitGroup
	var renamed atomic.Int32
	n := 10
	errors := make(chan error, n)
	var newnameOK string
	for i := 0; i < n; i++ {
		newname := fmt.Sprintf("newname%d.txt", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, txFinish := ctrlctx.New(s.ctx, s.dbConnector.GetDB)
			defer txFinish(new(error))
			userctx := ctrlctx.NewWithToken(ctx, s.cluster, arvadostest.ActiveTokenV2)
			upd, err := s.localdb.CollectionUpdate(userctx, arvados.UpdateOptions{
				UUID: s.tmp.UUID,
				ReplaceFiles: map[string]string{
					"/" + newname: "current/foo.txt",
					"/foo.txt":    "",
				}})
			if err != nil {
				errors <- err
			} else {
				renamed.Add(1)
				s.expectFiles(c, upd, newname)
				newnameOK = newname
			}
		}()
	}
	wg.Wait()
	// N concurrent/overlapping attempts to rename foo.txt should
	// have succeed exactly one time, and the final collection
	// content should correspond to the operation that returned
	// success.
	if !c.Check(int(renamed.Load()), check.Equals, 1) {
		close(errors)
		for err := range errors {
			c.Logf("err: %s", err)
		}
		return
	}
	c.Assert(newnameOK, check.Not(check.Equals), "")
	final, err := s.localdb.CollectionGet(s.userctx, arvados.GetOptions{UUID: s.tmp.UUID})
	c.Assert(err, check.IsNil)
	s.expectFiles(c, final, newnameOK)
}

// expectFiles checks coll's directory structure against the given
// list of expected files and empty directories. An expected path with
// a trailing slash indicates an empty directory.
func (s *replaceFilesSuite) expectFiles(c *check.C, coll arvados.Collection, expected ...string) {
	expectSizes := make(map[string]int64)
	for _, path := range expected {
		expectSizes[path] = -1
	}
	s.expectFileSizes(c, coll, expectSizes)
}

// expectFileSizes checks coll's directory structure against the given
// map of path->size.  An expected path with a trailing slash
// indicates an empty directory.  An expected size of -1 indicates the
// file size does not need to be checked.
func (s *replaceFilesSuite) expectFileSizes(c *check.C, coll arvados.Collection, expected map[string]int64) {
	cfs, err := coll.FileSystem(s.client, s.kc)
	c.Assert(err, check.IsNil)
	found := make(map[string]int64)
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
			fi, err := d.Info()
			c.Assert(err, check.IsNil)
			found[path] = fi.Size()
		}
		return nil
	})
	for d, nonempty := range nonemptydirs {
		if !nonempty {
			found[d] = 0
		}
	}
	for path, size := range found {
		if trimmed := strings.TrimPrefix(path, "/"); trimmed != path && trimmed != "" {
			found[trimmed] = size
			delete(found, path)
			path = trimmed
		}
		if expected[path] == -1 {
			// Path is expected to exist, and -1 means we
			// aren't supposed to check the size.  Change
			// "found size" to -1 as well, so this entry
			// will pass the DeepEquals check below.
			found[path] = -1
		}
	}
	c.Check(found, check.DeepEquals, expected)
}
