// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"net/http"
	"os"
	"time"

	check "gopkg.in/check.v1"
)

const (
	// Importing arvadostest would be an import cycle, so these
	// fixtures are duplicated here [until fs moves to a separate
	// package].
	fixtureActiveToken                  = "3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi"
	fixtureAProjectUUID                 = "zzzzz-j7d0g-v955i6s2oi1cbso"
	fixtureThisFilterGroupUUID          = "zzzzz-j7d0g-thisfiltergroup"
	fixtureAFilterGroupTwoUUID          = "zzzzz-j7d0g-afiltergrouptwo"
	fixtureAFilterGroupThreeUUID        = "zzzzz-j7d0g-filtergroupthre"
	fixtureAFilterGroupFourUUID         = "zzzzz-j7d0g-filtergroupfour"
	fixtureAFilterGroupFiveUUID         = "zzzzz-j7d0g-filtergroupfive"
	fixtureFooAndBarFilesInDirUUID      = "zzzzz-4zz18-foonbarfilesdir"
	fixtureFooCollectionName            = "zzzzz-4zz18-fy296fx3hot09f7 added sometime"
	fixtureFooCollectionPDH             = "1f4b0bc7583c2a7f9102c395f4ffc5e3+45"
	fixtureFooCollection                = "zzzzz-4zz18-fy296fx3hot09f7"
	fixtureNonexistentCollection        = "zzzzz-4zz18-totallynotexist"
	fixtureStorageClassesDesiredArchive = "zzzzz-4zz18-3t236wr12769qqa"
	fixtureBlobSigningKey               = "zfhgfenhffzltr9dixws36j1yhksjoll2grmku38mi7yxd66h5j4q9w4jzanezacp8s6q0ro3hxakfye02152hncy6zml2ed0uc"
	fixtureBlobSigningTTL               = 336 * time.Hour
)

var _ = check.Suite(&SiteFSSuite{})

func init() {
	// Enable DebugLocksPanicMode sometimes. Don't enable it all
	// the time, though -- it adds many calls to time.Sleep(),
	// which could hide different bugs.
	if time.Now().Second()&1 == 0 {
		DebugLocksPanicMode = true
	}
}

type SiteFSSuite struct {
	client *Client
	fs     CustomFileSystem
	kc     keepClient
}

func (s *SiteFSSuite) SetUpTest(c *check.C) {
	s.client = &Client{
		APIHost:   os.Getenv("ARVADOS_API_HOST"),
		AuthToken: fixtureActiveToken,
		Insecure:  true,
	}
	s.kc = &keepClientStub{
		blocks: map[string][]byte{
			"3858f62230ac3c915f300c664312c63f": []byte("foobar"),
		},
		sigkey:    fixtureBlobSigningKey,
		sigttl:    fixtureBlobSigningTTL,
		authToken: fixtureActiveToken,
	}
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
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 0)
}

func (s *SiteFSSuite) TestUpdateStorageClasses(c *check.C) {
	f, err := s.fs.OpenFile("/by_id/"+fixtureStorageClassesDesiredArchive+"/newfile", os.O_CREATE|os.O_RDWR, 0777)
	c.Assert(err, check.IsNil)
	_, err = f.Write([]byte("nope"))
	c.Assert(err, check.IsNil)
	err = f.Close()
	c.Assert(err, check.IsNil)
	err = s.fs.Sync()
	c.Assert(err, check.ErrorMatches, `.*stub does not write storage class "archive"`)
}

func (s *SiteFSSuite) TestByUUIDAndPDH(c *check.C) {
	f, err := s.fs.Open("/by_id")
	c.Assert(err, check.IsNil)
	fis, err := f.Readdir(-1)
	c.Check(err, check.IsNil)
	c.Check(len(fis), check.Equals, 0)

	err = s.fs.Mkdir("/by_id/"+fixtureFooCollection, 0755)
	c.Check(err, check.Equals, os.ErrExist)

	f, err = s.fs.Open("/by_id/" + fixtureNonexistentCollection)
	c.Assert(err, check.Equals, os.ErrNotExist)

	for _, path := range []string{
		fixtureFooCollection,
		fixtureFooCollectionPDH,
		fixtureAProjectUUID + "/" + fixtureFooCollectionName,
	} {
		f, err = s.fs.Open("/by_id/" + path)
		c.Assert(err, check.IsNil)
		fis, err = f.Readdir(-1)
		c.Assert(err, check.IsNil)
		var names []string
		for _, fi := range fis {
			names = append(names, fi.Name())
		}
		c.Check(names, check.DeepEquals, []string{"foo"})
	}

	f, err = s.fs.Open("/by_id/" + fixtureAProjectUUID + "/A Subproject/baz_file")
	c.Assert(err, check.IsNil)
	fis, err = f.Readdir(-1)
	c.Assert(err, check.IsNil)
	var names []string
	for _, fi := range fis {
		names = append(names, fi.Name())
	}
	c.Check(names, check.DeepEquals, []string{"baz"})

	_, err = s.fs.OpenFile("/by_id/"+fixtureNonexistentCollection, os.O_RDWR|os.O_CREATE, 0755)
	c.Check(err, check.Equals, ErrInvalidArgument)
	err = s.fs.Rename("/by_id/"+fixtureFooCollection, "/by_id/beep")
	c.Check(err, check.Equals, ErrInvalidArgument)
	err = s.fs.Rename("/by_id/"+fixtureFooCollection+"/foo", "/by_id/beep")
	c.Check(err, check.Equals, ErrInvalidArgument)
	_, err = s.fs.Stat("/by_id/beep")
	c.Check(err, check.Equals, os.ErrNotExist)
	err = s.fs.Rename("/by_id/"+fixtureFooCollection+"/foo", "/by_id/"+fixtureFooCollection+"/bar")
	c.Check(err, check.IsNil)

	err = s.fs.Rename("/by_id", "/beep")
	c.Check(err, check.Equals, ErrInvalidArgument)
}
