// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"runtime"
	"sort"
	"syscall"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/sirupsen/logrus"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&copierSuite{})

type copierSuite struct {
	cp  copier
	log bytes.Buffer
}

func (s *copierSuite) SetUpTest(c *check.C) {
	tmpdir := c.MkDir()
	s.log = bytes.Buffer{}

	cl, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.IsNil)
	kc, err := keepclient.MakeKeepClient(cl)
	c.Assert(err, check.IsNil)
	collfs, err := (&arvados.Collection{}).FileSystem(arvados.NewClientFromEnv(), kc)
	c.Assert(err, check.IsNil)

	s.cp = copier{
		client:        arvados.NewClientFromEnv(),
		keepClient:    kc,
		hostOutputDir: tmpdir,
		ctrOutputDir:  "/ctr/outdir",
		mounts: map[string]arvados.Mount{
			"/ctr/outdir": {Kind: "tmp"},
		},
		secretMounts: map[string]arvados.Mount{
			"/secret_text": {Kind: "text", Content: "xyzzy"},
		},
		logger: &logrus.Logger{Out: &s.log, Formatter: &logrus.TextFormatter{}, Level: logrus.InfoLevel},
		staged: collfs,
	}
}

func (s *copierSuite) TestEmptyOutput(c *check.C) {
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Check(s.cp.dirs, check.DeepEquals, []string(nil))
	c.Check(len(s.cp.files), check.Equals, 0)
}

func (s *copierSuite) TestEmptyWritableMount(c *check.C) {
	s.writeFileInOutputDir(c, ".arvados#collection", `{"manifest_text":""}`)
	s.cp.mounts[s.cp.ctrOutputDir] = arvados.Mount{
		Kind:     "collection",
		Writable: true,
	}

	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Assert(err, check.IsNil)
	c.Check(s.cp.dirs, check.DeepEquals, []string(nil))
	c.Check(len(s.cp.files), check.Equals, 0)
	rootdir, err := s.cp.staged.Open(".")
	c.Assert(err, check.IsNil)
	defer rootdir.Close()
	fis, err := rootdir.Readdir(-1)
	c.Assert(err, check.IsNil)
	c.Check(fis, check.HasLen, 0)
}

func (s *copierSuite) TestOutputCollectionWithOnlySubmounts(c *check.C) {
	s.writeFileInOutputDir(c, ".arvados#collection", `{"manifest_text":""}`)
	s.cp.mounts[s.cp.ctrOutputDir] = arvados.Mount{
		Kind:     "collection",
		Writable: true,
	}
	s.cp.mounts[path.Join(s.cp.ctrOutputDir, "foo")] = arvados.Mount{
		Kind:             "collection",
		Path:             "foo",
		PortableDataHash: arvadostest.FooCollectionPDH,
	}

	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Assert(err, check.IsNil)

	// s.cp.dirs and s.cp.files are empty, because nothing needs
	// to be copied from disk.
	c.Check(s.cp.dirs, check.DeepEquals, []string(nil))
	c.Check(len(s.cp.files), check.Equals, 0)

	// The "foo" file has already been copied from FooCollection
	// to s.cp.staged via Snapshot+Splice.
	rootdir, err := s.cp.staged.Open(".")
	c.Assert(err, check.IsNil)
	defer rootdir.Close()
	fis, err := rootdir.Readdir(-1)
	c.Assert(err, check.IsNil)
	c.Assert(fis, check.HasLen, 1)
	c.Check(fis[0].Size(), check.Equals, int64(3))
}

func (s *copierSuite) TestRepetitiveMountsInOutputDir(c *check.C) {
	var memstats0 runtime.MemStats
	runtime.ReadMemStats(&memstats0)

	s.writeFileInOutputDir(c, ".arvados#collection", `{"manifest_text":""}`)
	s.cp.mounts[s.cp.ctrOutputDir] = arvados.Mount{
		Kind:     "collection",
		Writable: true,
	}
	nmounts := 200
	ncollections := 1
	pdh := make([]string, ncollections)
	s.cp.manifestCache = make(map[string]string)
	for i := 0; i < ncollections; i++ {
		mtxt := arvadostest.FakeManifest(1, nmounts, 2, 4<<20)
		pdh[i] = arvados.PortableDataHash(mtxt)
		s.cp.manifestCache[pdh[i]] = mtxt
	}
	for i := 0; i < nmounts; i++ {
		filename := fmt.Sprintf("file%d", i)
		s.cp.mounts[path.Join(s.cp.ctrOutputDir, filename)] = arvados.Mount{
			Kind:             "collection",
			Path:             fmt.Sprintf("dir0/dir%d/file%d", i, i),
			PortableDataHash: pdh[i%ncollections],
		}
	}
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Assert(err, check.IsNil)

	// Files mounted under output dir have been copied from the
	// fake collections to s.cp.staged via Snapshot+Splice.
	rootdir, err := s.cp.staged.Open(".")
	c.Assert(err, check.IsNil)
	defer rootdir.Close()
	fis, err := rootdir.Readdir(-1)
	c.Assert(err, check.IsNil)
	c.Assert(fis, check.HasLen, nmounts)

	// nmounts -- Δalloc before -> Δalloc after fixing #22827
	// 500 -- 1542 MB -> 15 MB
	// 200 -- 254 MB -> 5 MB
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	delta := (int64(memstats.Alloc) - int64(memstats0.Alloc)) / 1000000
	c.Logf("Δalloc %d MB", delta)
	c.Check(delta < 40, check.Equals, true, check.Commentf("Δalloc %d MB is suspiciously high, expect ~ 5 MB", delta))
}

func (s *copierSuite) TestRegularFilesAndDirs(c *check.C) {
	err := os.MkdirAll(s.cp.hostOutputDir+"/dir1/dir2/dir3", 0755)
	c.Assert(err, check.IsNil)
	f, err := os.OpenFile(s.cp.hostOutputDir+"/dir1/foo", os.O_CREATE|os.O_WRONLY, 0644)
	c.Assert(err, check.IsNil)
	_, err = io.WriteString(f, "foo")
	c.Assert(err, check.IsNil)
	c.Assert(f.Close(), check.IsNil)
	err = syscall.Mkfifo(s.cp.hostOutputDir+"/dir1/fifo", 0644)
	c.Assert(err, check.IsNil)

	err = s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Check(s.cp.dirs, check.DeepEquals, []string{"/dir1", "/dir1/dir2", "/dir1/dir2/dir3"})
	c.Check(s.cp.files, check.DeepEquals, []filetodo{
		{src: os.DevNull, dst: "/dir1/dir2/dir3/.keep"},
		{src: s.cp.hostOutputDir + "/dir1/foo", dst: "/dir1/foo", size: 3},
	})
	c.Check(s.log.String(), check.Matches, `.* msg="Skipping unsupported file type \(mode 200000644\) in output dir: \\"/ctr/outdir/dir1/fifo\\""\n`)
}

func (s *copierSuite) TestSymlinkCycle(c *check.C) {
	c.Assert(os.Mkdir(s.cp.hostOutputDir+"/dir1", 0755), check.IsNil)
	c.Assert(os.Mkdir(s.cp.hostOutputDir+"/dir2", 0755), check.IsNil)
	c.Assert(os.Symlink("../dir2", s.cp.hostOutputDir+"/dir1/l_dir2"), check.IsNil)
	c.Assert(os.Symlink("../dir1", s.cp.hostOutputDir+"/dir2/l_dir1"), check.IsNil)
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.ErrorMatches, `.*cycle.*`)
}

func (s *copierSuite) TestSymlinkTargetMissing(c *check.C) {
	c.Assert(os.Symlink("./missing", s.cp.hostOutputDir+"/symlink"), check.IsNil)
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.ErrorMatches, `.*/ctr/outdir/missing.*`)
}

func (s *copierSuite) TestSymlinkTargetNotMounted(c *check.C) {
	c.Assert(os.Symlink("../boop", s.cp.hostOutputDir+"/symlink"), check.IsNil)
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.ErrorMatches, `.*/ctr/boop.*`)
}

func (s *copierSuite) TestSymlinkToSecret(c *check.C) {
	c.Assert(os.Symlink("/secret_text", s.cp.hostOutputDir+"/symlink"), check.IsNil)
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Check(len(s.cp.dirs), check.Equals, 0)
	c.Check(len(s.cp.files), check.Equals, 0)
}

func (s *copierSuite) TestSecretInOutputDir(c *check.C) {
	s.cp.secretMounts["/ctr/outdir/secret_text"] = s.cp.secretMounts["/secret_text"]
	s.writeFileInOutputDir(c, "secret_text", "xyzzy")
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Check(len(s.cp.dirs), check.Equals, 0)
	c.Check(len(s.cp.files), check.Equals, 0)
}

func (s *copierSuite) TestSymlinkToMountedCollection(c *check.C) {
	// simulate mounted read-only collection
	s.cp.mounts["/mnt"] = arvados.Mount{
		Kind:             "collection",
		PortableDataHash: arvadostest.FooCollectionPDH,
	}

	// simulate mounted writable collection
	bindtmp := c.MkDir()
	f, err := os.OpenFile(bindtmp+"/.arvados#collection", os.O_CREATE|os.O_WRONLY, 0644)
	c.Assert(err, check.IsNil)
	_, err = io.WriteString(f, `{"manifest_text":". 37b51d194a7513e45b56f6524f2d51f2+3 0:3:bar\n"}`)
	c.Assert(err, check.IsNil)
	c.Assert(f.Close(), check.IsNil)
	s.cp.mounts["/mnt-w"] = arvados.Mount{
		Kind:             "collection",
		PortableDataHash: arvadostest.FooCollectionPDH,
		Writable:         true,
	}
	s.cp.bindmounts = map[string]bindmount{
		"/mnt-w": bindmount{HostPath: bindtmp, ReadOnly: false},
	}

	c.Assert(os.Symlink("../../mnt", s.cp.hostOutputDir+"/l_dir"), check.IsNil)
	c.Assert(os.Symlink("/mnt/foo", s.cp.hostOutputDir+"/l_file"), check.IsNil)
	c.Assert(os.Symlink("/mnt-w/bar", s.cp.hostOutputDir+"/l_file_w"), check.IsNil)

	err = s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	s.checkStagedFile(c, "l_dir/foo", 3)
	s.checkStagedFile(c, "l_file", 3)
	s.checkStagedFile(c, "l_file_w", 3)
}

func (s *copierSuite) checkStagedFile(c *check.C, path string, size int64) {
	fi, err := s.cp.staged.Stat(path)
	if c.Check(err, check.IsNil) {
		c.Check(fi.Size(), check.Equals, size)
	}
}

func (s *copierSuite) TestSymlink(c *check.C) {
	hostfile := s.cp.hostOutputDir + "/dir1/file"

	err := os.MkdirAll(s.cp.hostOutputDir+"/dir1/dir2/dir3", 0755)
	c.Assert(err, check.IsNil)
	s.writeFileInOutputDir(c, "dir1/file", "file")
	for _, err := range []error{
		os.Symlink(s.cp.ctrOutputDir+"/dir1/file", s.cp.hostOutputDir+"/l_abs_file"),
		os.Symlink(s.cp.ctrOutputDir+"/dir1/dir2", s.cp.hostOutputDir+"/l_abs_dir2"),
		os.Symlink("../../dir1/file", s.cp.hostOutputDir+"/dir1/dir2/l_rel_file"),
		os.Symlink("dir1/file", s.cp.hostOutputDir+"/l_rel_file"),
		os.MkdirAll(s.cp.hostOutputDir+"/morelinks", 0755),
		os.Symlink("../dir1/dir2", s.cp.hostOutputDir+"/morelinks/l_rel_dir2"),
		os.Symlink("dir1/dir2/dir3", s.cp.hostOutputDir+"/l_rel_dir3"),
		// rel. symlink -> rel. symlink -> regular file
		os.Symlink("../dir1/dir2/l_rel_file", s.cp.hostOutputDir+"/morelinks/l_rel_l_rel_file"),
	} {
		c.Assert(err, check.IsNil)
	}

	err = s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Check(s.cp.dirs, check.DeepEquals, []string{
		"/dir1", "/dir1/dir2", "/dir1/dir2/dir3",
		"/l_abs_dir2", "/l_abs_dir2/dir3",
		"/l_rel_dir3",
		"/morelinks", "/morelinks/l_rel_dir2", "/morelinks/l_rel_dir2/dir3",
	})
	c.Check(s.cp.files, check.DeepEquals, []filetodo{
		{dst: "/dir1/dir2/dir3/.keep", src: os.DevNull},
		{dst: "/dir1/dir2/l_rel_file", src: hostfile, size: 4},
		{dst: "/dir1/file", src: hostfile, size: 4},
		{dst: "/l_abs_dir2/dir3/.keep", src: os.DevNull},
		{dst: "/l_abs_dir2/l_rel_file", src: hostfile, size: 4},
		{dst: "/l_abs_file", src: hostfile, size: 4},
		{dst: "/l_rel_dir3/.keep", src: os.DevNull},
		{dst: "/l_rel_file", src: hostfile, size: 4},
		{dst: "/morelinks/l_rel_dir2/dir3/.keep", src: os.DevNull},
		{dst: "/morelinks/l_rel_dir2/l_rel_file", src: hostfile, size: 4},
		{dst: "/morelinks/l_rel_l_rel_file", src: hostfile, size: 4},
	})
}

func (s *copierSuite) TestUnsupportedOutputMount(c *check.C) {
	s.cp.mounts["/ctr/outdir"] = arvados.Mount{Kind: "waz"}
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.NotNil)
}

func (s *copierSuite) TestUnsupportedMountKindBelow(c *check.C) {
	s.cp.mounts["/ctr/outdir/dirk"] = arvados.Mount{Kind: "waz"}
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.NotNil)
}

func (s *copierSuite) TestWritableMountBelow(c *check.C) {
	s.cp.mounts["/ctr/outdir/mount"] = arvados.Mount{
		Kind:             "collection",
		PortableDataHash: arvadostest.FooCollectionPDH,
		Writable:         true,
	}
	c.Assert(os.MkdirAll(s.cp.hostOutputDir+"/mount", 0755), check.IsNil)
	s.writeFileInOutputDir(c, "file", "file")
	s.writeFileInOutputDir(c, "mount/foo", "foo")

	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Check(s.cp.dirs, check.DeepEquals, []string{"/mount"})
	c.Check(s.cp.files, check.DeepEquals, []filetodo{
		{src: s.cp.hostOutputDir + "/file", dst: "/file", size: 4},
		{src: s.cp.hostOutputDir + "/mount/foo", dst: "/mount/foo", size: 3},
	})
}

// Check some glob-matching edge cases. In particular, check that
// patterns like "foo/**" do not match regular files named "foo"
// (unless of course they are inside a directory named "foo").
func (s *copierSuite) TestMatchGlobs(c *check.C) {
	s.cp.globs = []string{"foo*/**"}
	c.Check(s.cp.matchGlobs("foo", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("food", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("foo", false), check.Equals, false)
	c.Check(s.cp.matchGlobs("food", false), check.Equals, false)
	c.Check(s.cp.matchGlobs("foo/bar", false), check.Equals, true)
	c.Check(s.cp.matchGlobs("food/bar", false), check.Equals, true)
	c.Check(s.cp.matchGlobs("foo/bar", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("food/bar", true), check.Equals, true)

	s.cp.globs = []string{"ba[!/]/foo*/**"}
	c.Check(s.cp.matchGlobs("bar/foo", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("bar/food", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("bar/foo", false), check.Equals, false)
	c.Check(s.cp.matchGlobs("bar/food", false), check.Equals, false)
	c.Check(s.cp.matchGlobs("bar/foo/z\\[", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("bar/food/z\\[", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("bar/foo/z\\[", false), check.Equals, true)
	c.Check(s.cp.matchGlobs("bar/food/z\\[", false), check.Equals, true)

	s.cp.globs = []string{"waz/**/foo*/**"}
	c.Check(s.cp.matchGlobs("waz/quux/foo", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("waz/quux/food", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("waz/quux/foo", false), check.Equals, false)
	c.Check(s.cp.matchGlobs("waz/quux/food", false), check.Equals, false)
	c.Check(s.cp.matchGlobs("waz/quux/foo/foo", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("waz/quux/food/foo", true), check.Equals, true)
	c.Check(s.cp.matchGlobs("waz/quux/foo/foo", false), check.Equals, true)
	c.Check(s.cp.matchGlobs("waz/quux/food/foo", false), check.Equals, true)

	s.cp.globs = []string{"foo/**/*"}
	c.Check(s.cp.matchGlobs("foo", false), check.Equals, false)
	c.Check(s.cp.matchGlobs("foo/bar", false), check.Equals, true)
	c.Check(s.cp.matchGlobs("foo/bar/baz", false), check.Equals, true)
	c.Check(s.cp.matchGlobs("foo/bar/baz/waz", false), check.Equals, true)
}

func (s *copierSuite) TestSubtreeCouldMatch(c *check.C) {
	for _, trial := range []struct {
		mount string // relative to output dir
		glob  string
		could bool
	}{
		{mount: "abc", glob: "*"},
		{mount: "abc", glob: "abc/*", could: true},
		{mount: "abc", glob: "a*/**", could: true},
		{mount: "abc", glob: "**", could: true},
		{mount: "abc", glob: "*/*", could: true},
		{mount: "abc", glob: "**/*.txt", could: true},
		{mount: "abc/def", glob: "*"},
		{mount: "abc/def", glob: "*/*"},
		{mount: "abc/def", glob: "*/*.txt"},
		{mount: "abc/def", glob: "*/*/*", could: true},
		{mount: "abc/def", glob: "**", could: true},
		{mount: "abc/def", glob: "**/bar", could: true},
		{mount: "abc/def", glob: "abc/**", could: true},
		{mount: "abc/def/ghi", glob: "*c/**/bar", could: true},
		{mount: "abc/def/ghi", glob: "*c/*f/bar"},
		{mount: "abc/def/ghi", glob: "abc/d[^/]f/ghi/*", could: true},
	} {
		c.Logf("=== %+v", trial)
		got := (&copier{
			globs: []string{trial.glob},
		}).subtreeCouldMatch(trial.mount)
		c.Check(got, check.Equals, trial.could)
	}
}

func (s *copierSuite) TestCopyFromLargeCollection_Readonly(c *check.C) {
	s.testCopyFromLargeCollection(c, false)
}

func (s *copierSuite) TestCopyFromLargeCollection_Writable(c *check.C) {
	s.testCopyFromLargeCollection(c, true)
}

func (s *copierSuite) testCopyFromLargeCollection(c *check.C, writable bool) {
	bindtmp := c.MkDir()
	mtxt := arvadostest.FakeManifest(100, 100, 2, 4<<20)
	pdh := arvados.PortableDataHash(mtxt)
	json, err := json.Marshal(arvados.Collection{ManifestText: mtxt, PortableDataHash: pdh})
	c.Assert(err, check.IsNil)
	err = os.WriteFile(bindtmp+"/.arvados#collection", json, 0644)
	// This symlink tricks walkHostFS into calling walkMount on
	// the fakecollection dir. If we did the obvious thing instead
	// (i.e., mount a collection under the output dir) walkMount
	// would see that our fakecollection dir is actually a regular
	// directory, conclude that the mount has been deleted and
	// replaced by a regular directory tree, and process the tree
	// as regular files, bypassing the manifest-copying code path
	// we're trying to test.
	err = os.Symlink("/fakecollection", s.cp.hostOutputDir+"/fakecollection")
	c.Assert(err, check.IsNil)
	s.cp.mounts["/fakecollection"] = arvados.Mount{
		Kind:             "collection",
		PortableDataHash: pdh,
		Writable:         writable,
	}
	s.cp.bindmounts = map[string]bindmount{
		"/fakecollection": bindmount{HostPath: bindtmp, ReadOnly: !writable},
	}
	s.cp.manifestCache = map[string]string{pdh: mtxt}
	err = s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Log(s.log.String())

	// Check some files to ensure they were copied properly.
	// Specifically, arbitrarily check every 17th file in every
	// 13th dir.  (This is better than checking all of the files
	// only in that it's less likely to show up as a distracting
	// signal in CPU profiling.)
	for i := 0; i < 100; i += 13 {
		for j := 0; j < 100; j += 17 {
			fnm := fmt.Sprintf("/fakecollection/dir%d/dir%d/file%d", i, j, j)
			_, err := s.cp.staged.Stat(fnm)
			c.Assert(err, check.IsNil, check.Commentf("%s", fnm))
		}
	}
}

func (s *copierSuite) TestMountBelowExcludedByGlob(c *check.C) {
	bindtmp := c.MkDir()
	s.cp.mounts["/ctr/outdir/include/includer"] = arvados.Mount{
		Kind:             "collection",
		PortableDataHash: arvadostest.FooCollectionPDH,
	}
	s.cp.mounts["/ctr/outdir/include/includew"] = arvados.Mount{
		Kind:             "collection",
		PortableDataHash: arvadostest.FooCollectionPDH,
		Writable:         true,
	}
	s.cp.mounts["/ctr/outdir/exclude/excluder"] = arvados.Mount{
		Kind:             "collection",
		PortableDataHash: arvadostest.FooCollectionPDH,
	}
	s.cp.mounts["/ctr/outdir/exclude/excludew"] = arvados.Mount{
		Kind:             "collection",
		PortableDataHash: arvadostest.FooCollectionPDH,
		Writable:         true,
	}
	s.cp.mounts["/ctr/outdir/nonexistent/collection"] = arvados.Mount{
		// As extra assurance, plant a collection that will
		// fail if copier attempts to load its manifest.  (For
		// performance reasons it's important that copier
		// doesn't try to load the manifest before deciding
		// not to copy the contents.)
		Kind:             "collection",
		PortableDataHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa+1234",
	}
	s.cp.globs = []string{
		"?ncl*/*r/*",
		"*/?ncl*/**",
	}
	c.Assert(os.MkdirAll(s.cp.hostOutputDir+"/include/includer", 0755), check.IsNil)
	c.Assert(os.MkdirAll(s.cp.hostOutputDir+"/include/includew", 0755), check.IsNil)
	c.Assert(os.MkdirAll(s.cp.hostOutputDir+"/exclude/excluder", 0755), check.IsNil)
	c.Assert(os.MkdirAll(s.cp.hostOutputDir+"/exclude/excludew", 0755), check.IsNil)
	s.writeFileInOutputDir(c, "include/includew/foo", "foo")
	s.writeFileInOutputDir(c, "exclude/excludew/foo", "foo")
	s.cp.bindmounts = map[string]bindmount{
		"/ctr/outdir/include/includew": bindmount{HostPath: bindtmp, ReadOnly: false},
	}
	s.cp.bindmounts = map[string]bindmount{
		"/ctr/outdir/include/excludew": bindmount{HostPath: bindtmp, ReadOnly: false},
	}

	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Log(s.log.String())

	// Note it's OK that "/exclude" is not excluded by walkMount:
	// it is just a local filesystem directory, not a mount point
	// that's expensive to walk.  In real-life usage, it will be
	// removed from cp.dirs before any copying happens.
	c.Check(s.cp.dirs, check.DeepEquals, []string{"/exclude", "/include", "/include/includew"})
	c.Check(s.cp.files, check.DeepEquals, []filetodo{
		{src: s.cp.hostOutputDir + "/include/includew/foo", dst: "/include/includew/foo", size: 3},
	})
	manifest, err := s.cp.staged.MarshalManifest(".")
	c.Assert(err, check.IsNil)
	c.Check(manifest, check.Matches, `(?ms).*\./include/includer .*`)
	c.Check(manifest, check.Not(check.Matches), `(?ms).*exclude.*`)
	c.Check(s.log.String(), check.Matches, `(?ms).*not copying \\"exclude/excluder\\".*`)
	c.Check(s.log.String(), check.Matches, `(?ms).*not copying \\"nonexistent/collection\\".*`)
}

func (s *copierSuite) writeFileInOutputDir(c *check.C, path, data string) {
	f, err := os.OpenFile(s.cp.hostOutputDir+"/"+path, os.O_CREATE|os.O_WRONLY, 0644)
	c.Assert(err, check.IsNil)
	_, err = io.WriteString(f, data)
	c.Assert(err, check.IsNil)
	c.Assert(f.Close(), check.IsNil)
}

// applyGlobsToFilesAndDirs uses the same glob-matching code as
// applyGlobsToStaged, so we don't need to test all of the same
// glob-matching behavior covered in TestApplyGlobsToCollectionFS.  We
// do need to check that (a) the glob is actually being used to filter
// out files, and (b) non-matching dirs still included if and only if
// they are ancestors of matching files.
func (s *copierSuite) TestApplyGlobsToFilesAndDirs(c *check.C) {
	dirs := []string{"dir1", "dir1/dir11", "dir1/dir12", "dir2"}
	files := []string{"dir1/file11", "dir1/dir11/file111", "dir2/file2"}
	for _, trial := range []struct {
		globs []string
		dirs  []string
		files []string
	}{
		{
			globs: []string{},
			dirs:  append([]string{}, dirs...),
			files: append([]string{}, files...),
		},
		{
			globs: []string{"**"},
			dirs:  append([]string{}, dirs...),
			files: append([]string{}, files...),
		},
		{
			globs: []string{"**/file111"},
			dirs:  []string{"dir1", "dir1/dir11"},
			files: []string{"dir1/dir11/file111"},
		},
		{
			globs: []string{"nothing"},
			dirs:  nil,
			files: nil,
		},
		{
			globs: []string{"**/dir12"},
			dirs:  []string{"dir1", "dir1/dir12"},
			files: nil,
		},
		{
			globs: []string{"**/file*"},
			dirs:  []string{"dir1", "dir1/dir11", "dir2"},
			files: append([]string{}, files...),
		},
		{
			globs: []string{"**/dir1[12]"},
			dirs:  []string{"dir1", "dir1/dir11", "dir1/dir12"},
			files: nil,
		},
		{
			globs: []string{"**/dir1[^2]"},
			dirs:  []string{"dir1", "dir1/dir11"},
			files: nil,
		},
		{
			globs: []string{"dir1/**"},
			dirs:  []string{"dir1", "dir1/dir11", "dir1/dir12"},
			files: []string{"dir1/file11", "dir1/dir11/file111"},
		},
	} {
		c.Logf("=== globs: %q", trial.globs)
		cp := copier{
			globs: trial.globs,
			dirs:  dirs,
		}
		for _, path := range files {
			cp.files = append(cp.files, filetodo{dst: path})
		}
		cp.applyGlobsToFilesAndDirs()
		var gotFiles []string
		for _, file := range cp.files {
			gotFiles = append(gotFiles, file.dst)
		}
		c.Check(cp.dirs, check.DeepEquals, trial.dirs)
		c.Check(gotFiles, check.DeepEquals, trial.files)
	}
}

func (s *copierSuite) TestApplyGlobsToCollectionFS(c *check.C) {
	for _, trial := range []struct {
		globs  []string
		expect []string
	}{
		{
			globs:  nil,
			expect: []string{"foo", "bar", "baz/quux", "baz/parent1/item1"},
		},
		{
			globs:  []string{"foo"},
			expect: []string{"foo"},
		},
		{
			globs:  []string{"baz/parent1/item1"},
			expect: []string{"baz/parent1/item1"},
		},
		{
			globs:  []string{"**"},
			expect: []string{"foo", "bar", "baz/quux", "baz/parent1/item1"},
		},
		{
			globs:  []string{"**/*"},
			expect: []string{"foo", "bar", "baz/quux", "baz/parent1/item1"},
		},
		{
			globs:  []string{"*"},
			expect: []string{"foo", "bar"},
		},
		{
			globs:  []string{"baz"},
			expect: nil,
		},
		{
			globs:  []string{"b*/**"},
			expect: []string{"baz/quux", "baz/parent1/item1"},
		},
		{
			globs:  []string{"baz"},
			expect: nil,
		},
		{
			globs:  []string{"baz/**"},
			expect: []string{"baz/quux", "baz/parent1/item1"},
		},
		{
			globs:  []string{"baz/*"},
			expect: []string{"baz/quux"},
		},
		{
			globs:  []string{"baz/**/*uu?"},
			expect: []string{"baz/quux"},
		},
		{
			globs:  []string{"**/*m1"},
			expect: []string{"baz/parent1/item1"},
		},
		{
			globs:  []string{"*/*/*/**/*1"},
			expect: nil,
		},
		{
			globs:  []string{"f*", "**/q*"},
			expect: []string{"foo", "baz/quux"},
		},
		{
			globs:  []string{"\\"}, // invalid pattern matches nothing
			expect: nil,
		},
		{
			globs:  []string{"\\", "foo"},
			expect: []string{"foo"},
		},
		{
			globs:  []string{"foo/**"},
			expect: nil,
		},
		{
			globs:  []string{"foo*/**"},
			expect: nil,
		},
	} {
		c.Logf("=== globs: %q", trial.globs)
		collfs, err := (&arvados.Collection{ManifestText: ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo 0:0:bar 0:0:baz/quux 0:0:baz/parent1/item1\n"}).FileSystem(nil, nil)
		c.Assert(err, check.IsNil)
		cp := copier{globs: trial.globs, staged: collfs}
		err = cp.applyGlobsToStaged()
		if !c.Check(err, check.IsNil) {
			continue
		}
		var got []string
		fs.WalkDir(arvados.FS(collfs), "", func(path string, ent fs.DirEntry, err error) error {
			if !ent.IsDir() {
				got = append(got, path)
			}
			return nil
		})
		sort.Strings(got)
		sort.Strings(trial.expect)
		c.Check(got, check.DeepEquals, trial.expect)
	}
}
