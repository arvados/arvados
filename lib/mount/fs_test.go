// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"os"
	"syscall"
	"testing"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/arvados/cgofuse/fuse"
	. "gopkg.in/check.v1"
)

// Gocheck boilerplate
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&FSSuite{})

type FSSuite struct {
	fs *keepFS
}

func (s *FSSuite) SetUpTest(c *C) {
	client := arvados.NewClientFromEnv()
	ac, err := arvadosclient.New(client)
	c.Assert(err, IsNil)
	kc, err := keepclient.MakeKeepClient(ac)
	c.Assert(err, IsNil)
	s.fs = &keepFS{
		Client:     client,
		KeepClient: kc,
		Logger:     ctxlog.TestLogger(c),
	}
	s.fs.Init()
}

func (s *FSSuite) TestFuseInterface(c *C) {
	var _ fuse.FileSystemInterface = s.fs
}

func (s *FSSuite) TestOpendir(c *C) {
	errc, fh := s.fs.Opendir("/by_id")
	c.Check(errc, Equals, 0)
	c.Check(fh, Not(Equals), uint64(0))
	c.Check(fh, Not(Equals), invalidFH)
	errc, fh = s.fs.Opendir("/bogus")
	c.Check(errc, Equals, -fuse.ENOENT)
	c.Check(fh, Equals, invalidFH)
}

func (s *FSSuite) TestMknod_ReadOnly(c *C) {
	s.fs.ReadOnly = true
	path := "/by_id/" + arvadostest.FooCollection + "/z"
	errc := s.fs.Mknod(path, syscall.S_IFREG, 0)
	c.Check(errc, Equals, -fuse.EROFS)
}

func (s *FSSuite) TestMknod(c *C) {
	path := "/by_id/" + arvadostest.FooCollection + "/z"
	_, err := s.fs.root.Stat(path)
	c.Assert(err, Equals, os.ErrNotExist)

	// Should return error if mode indicates unsupported file type
	for _, mode := range []uint32{
		syscall.S_IFCHR,
		syscall.S_IFBLK,
		syscall.S_IFIFO,
		syscall.S_IFSOCK,
	} {
		errc := s.fs.Mknod(path, mode, 0)
		c.Check(errc, Equals, -fuse.ENOSYS)
		_, err := s.fs.root.Stat(path)
		c.Check(err, Equals, os.ErrNotExist)
	}

	// Should create file and return 0 if mode indicates regular
	// file
	errc := s.fs.Mknod(path, syscall.S_IFREG|0o644, 0)
	c.Check(errc, Equals, 0)
	_, err = s.fs.root.Stat(path)
	c.Check(err, IsNil)

	// Special case: "Zero file type is equivalent to type
	// S_IFREG." cf. mknod(2)
	errc = s.fs.Mknod(path+"2", 0o644, 0)
	c.Check(errc, Equals, 0)
	_, err = s.fs.root.Stat(path + "2")
	c.Check(err, IsNil)

	// Should return error if target exists
	errc = s.fs.Mknod(path, syscall.S_IFREG|0o644, 0)
	c.Check(errc, Equals, -fuse.EEXIST)
}
