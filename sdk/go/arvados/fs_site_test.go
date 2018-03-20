// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"net/http"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&SiteFSSuite{})

type SiteFSSuite struct {
	client *Client
	fs     CustomFileSystem
	kc     keepClient
}

func (s *SiteFSSuite) SetUpTest(c *check.C) {
	s.client = &Client{
		APIHost:   os.Getenv("ARVADOS_API_HOST"),
		AuthToken: arvadostest.ActiveToken,
		Insecure:  true,
	}
	s.kc = &keepClientStub{
		blocks: map[string][]byte{
			"3858f62230ac3c915f300c664312c63f": []byte("foobar"),
		}}
	s.fs = s.client.SiteFileSystem(s.kc)
}

func (s *SiteFSSuite) TestHttpFileSystemInterface(c *check.C) {
	_, ok := s.fs.(http.FileSystem)
	c.Check(ok, check.Equals, true)
}

func (s *SiteFSSuite) TestByIDEmpty(c *check.C) {
	f, err := s.fs.Open("/by_id")
	c.Assert(err, check.IsNil)
	fis, err := f.Readdir(-1)
	c.Check(len(fis), check.Equals, 0)
}

func (s *SiteFSSuite) TestByUUID(c *check.C) {
	f, err := s.fs.Open("/by_id")
	c.Assert(err, check.IsNil)
	fis, err := f.Readdir(-1)
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 0)

	err = s.fs.Mkdir("/by_id/"+arvadostest.FooCollection, 0755)
	c.Check(err, check.Equals, os.ErrExist)

	f, err = s.fs.Open("/by_id/" + arvadostest.NonexistentCollection)
	c.Assert(err, check.Equals, os.ErrNotExist)

	f, err = s.fs.Open("/by_id/" + arvadostest.FooCollection)
	c.Assert(err, check.IsNil)
	fis, err = f.Readdir(-1)
	var names []string
	for _, fi := range fis {
		names = append(names, fi.Name())
	}
	c.Check(names, check.DeepEquals, []string{"foo"})

	_, err = s.fs.OpenFile("/by_id/"+arvadostest.NonexistentCollection, os.O_RDWR|os.O_CREATE, 0755)
	c.Check(err, check.Equals, ErrInvalidOperation)
	err = s.fs.Rename("/by_id/"+arvadostest.FooCollection, "/by_id/beep")
	c.Check(err, check.Equals, ErrInvalidArgument)
	err = s.fs.Rename("/by_id/"+arvadostest.FooCollection+"/foo", "/by_id/beep")
	c.Check(err, check.Equals, ErrInvalidArgument)
	_, err = s.fs.Stat("/by_id/beep")
	c.Check(err, check.Equals, os.ErrNotExist)
	err = s.fs.Rename("/by_id/"+arvadostest.FooCollection+"/foo", "/by_id/"+arvadostest.FooCollection+"/bar")
	c.Check(err, check.IsNil)

	err = s.fs.Rename("/by_id", "/beep")
	c.Check(err, check.Equals, ErrInvalidArgument)
}
