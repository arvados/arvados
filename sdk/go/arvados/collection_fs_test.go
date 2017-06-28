// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"io"
	"net/http"
	"os"
	"testing"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollectionFSSuite{})

type CollectionFSSuite struct {
	client *Client
	coll   Collection
	fs     http.FileSystem
}

func (s *CollectionFSSuite) SetUpTest(c *check.C) {
	s.client = NewClientFromEnv()
	err := s.client.RequestAndDecode(&s.coll, "GET", "arvados/v1/collections/"+arvadostest.FooAndBarFilesInDirUUID, nil, nil)
	c.Assert(err, check.IsNil)
	s.fs = s.coll.FileSystem(s.client, nil)
}

func (s *CollectionFSSuite) TestReaddirFull(c *check.C) {
	f, err := s.fs.Open("/dir1")
	c.Assert(err, check.IsNil)

	st, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(2))
	c.Check(st.IsDir(), check.Equals, true)

	fis, err := f.Readdir(0)
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 2)
	if len(fis) > 0 {
		c.Check(fis[0].Size(), check.Equals, int64(3))
	}
}

func (s *CollectionFSSuite) TestReaddirLimited(c *check.C) {
	f, err := s.fs.Open("./dir1")
	c.Assert(err, check.IsNil)

	fis, err := f.Readdir(1)
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 1)
	if len(fis) > 0 {
		c.Check(fis[0].Size(), check.Equals, int64(3))
	}

	fis, err = f.Readdir(1)
	c.Check(err, check.Equals, io.EOF)
	c.Check(len(fis), check.Equals, 1)
	if len(fis) > 0 {
		c.Check(fis[0].Size(), check.Equals, int64(3))
	}

	fis, err = f.Readdir(1)
	c.Check(len(fis), check.Equals, 0)
	c.Check(err, check.NotNil)
	c.Check(err, check.Equals, io.EOF)

	f, err = s.fs.Open("dir1")
	c.Assert(err, check.IsNil)
	fis, err = f.Readdir(1)
	c.Check(len(fis), check.Equals, 1)
	c.Assert(err, check.IsNil)
	fis, err = f.Readdir(2)
	c.Check(len(fis), check.Equals, 1)
	c.Assert(err, check.Equals, io.EOF)
	fis, err = f.Readdir(2)
	c.Check(len(fis), check.Equals, 0)
	c.Assert(err, check.Equals, io.EOF)
}

func (s *CollectionFSSuite) TestPathMunge(c *check.C) {
	for _, path := range []string{".", "/", "./", "///", "/../", "/./.."} {
		f, err := s.fs.Open(path)
		c.Assert(err, check.IsNil)

		st, err := f.Stat()
		c.Assert(err, check.IsNil)
		c.Check(st.Size(), check.Equals, int64(1))
		c.Check(st.IsDir(), check.Equals, true)
	}
	for _, path := range []string{"/dir1", "dir1", "./dir1", "///dir1//.//", "../dir1/../dir1/"} {
		c.Logf("%q", path)
		f, err := s.fs.Open(path)
		c.Assert(err, check.IsNil)

		st, err := f.Stat()
		c.Assert(err, check.IsNil)
		c.Check(st.Size(), check.Equals, int64(2))
		c.Check(st.IsDir(), check.Equals, true)
	}
}

func (s *CollectionFSSuite) TestNotExist(c *check.C) {
	for _, path := range []string{"/no", "no", "./no", "n/o", "/n/o"} {
		f, err := s.fs.Open(path)
		c.Assert(f, check.IsNil)
		c.Assert(err, check.NotNil)
		c.Assert(os.IsNotExist(err), check.Equals, true)
	}
}

func (s *CollectionFSSuite) TestOpenFile(c *check.C) {
	c.Skip("cannot test files with nil keepclient")

	f, err := s.fs.Open("/foo.txt")
	c.Assert(err, check.IsNil)
	st, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(st.Size(), check.Equals, int64(3))
}

// Gocheck boilerplate
func Test(t *testing.T) {
	check.TestingT(t)
}
