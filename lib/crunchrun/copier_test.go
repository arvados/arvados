// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"sort"
	"syscall"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
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
	s.cp = copier{
		client:        arvados.NewClientFromEnv(),
		hostOutputDir: tmpdir,
		ctrOutputDir:  "/ctr/outdir",
		mounts: map[string]arvados.Mount{
			"/ctr/outdir": {Kind: "tmp"},
		},
		secretMounts: map[string]arvados.Mount{
			"/secret_text": {Kind: "text", Content: "xyzzy"},
		},
		logger: &logrus.Logger{Out: &s.log, Formatter: &logrus.TextFormatter{}, Level: logrus.InfoLevel},
	}
}

func (s *copierSuite) TestEmptyOutput(c *check.C) {
	err := s.cp.walkMount("", s.cp.ctrOutputDir, 10, true)
	c.Check(err, check.IsNil)
	c.Check(s.cp.dirs, check.DeepEquals, []string(nil))
	c.Check(len(s.cp.files), check.Equals, 0)
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
	bindtmp, err := ioutil.TempDir("", "crunch-run.test.")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(bindtmp)
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
	c.Check(s.cp.manifest, check.Matches, `(?ms)\./l_dir acbd\S+ 0:3:foo\n\. acbd\S+ 0:3:l_file\n\. 37b5\S+ 0:3:l_file_w\n`)
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

func (s *copierSuite) writeFileInOutputDir(c *check.C, path, data string) {
	f, err := os.OpenFile(s.cp.hostOutputDir+"/"+path, os.O_CREATE|os.O_WRONLY, 0644)
	c.Assert(err, check.IsNil)
	_, err = io.WriteString(f, data)
	c.Assert(err, check.IsNil)
	c.Assert(f.Close(), check.IsNil)
}

// applyGlobsToFilesAndDirs uses the same glob-matching code as
// applyGlobsToCollectionFS, so we don't need to test all of the same
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
		cp := copier{globs: trial.globs}
		err = cp.applyGlobsToCollectionFS(collfs)
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
