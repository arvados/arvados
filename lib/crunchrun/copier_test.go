// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"syscall"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
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
	api, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.IsNil)
	s.log = bytes.Buffer{}
	s.cp = copier{
		client:        arvados.NewClientFromEnv(),
		arvClient:     api,
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
	s.cp.binds = append(s.cp.binds, bindtmp+":/mnt-w")

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
