// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

type spiedRequest struct {
	method string
	path   string
	params map[string]interface{}
}

type spyingClient struct {
	*Client
	calls []spiedRequest
}

func (sc *spyingClient) RequestAndDecode(dst interface{}, method, path string, body io.Reader, params interface{}) error {
	var paramsCopy map[string]interface{}
	var buf bytes.Buffer
	json.NewEncoder(&buf).Encode(params)
	json.NewDecoder(&buf).Decode(&paramsCopy)
	sc.calls = append(sc.calls, spiedRequest{
		method: method,
		path:   path,
		params: paramsCopy,
	})
	return sc.Client.RequestAndDecode(dst, method, path, body, params)
}

func (s *SiteFSSuite) TestCurrentUserHome(c *check.C) {
	s.fs.MountProject("home", "")
	s.testHomeProject(c, "/home")
}

func (s *SiteFSSuite) TestUsersDir(c *check.C) {
	s.testHomeProject(c, "/users/active")
}

func (s *SiteFSSuite) testHomeProject(c *check.C, path string) {
	f, err := s.fs.Open(path)
	c.Assert(err, check.IsNil)
	fis, err := f.Readdir(-1)
	c.Check(len(fis), check.Not(check.Equals), 0)

	ok := false
	for _, fi := range fis {
		c.Check(fi.Name(), check.Not(check.Equals), "")
		if fi.Name() == "A Project" {
			ok = true
		}
	}
	c.Check(ok, check.Equals, true)

	f, err = s.fs.Open(path + "/A Project/..")
	c.Assert(err, check.IsNil)
	fi, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(fi.IsDir(), check.Equals, true)
	_, basename := filepath.Split(path)
	c.Check(fi.Name(), check.Equals, basename)

	f, err = s.fs.Open(path + "/A Project/A Subproject")
	c.Assert(err, check.IsNil)
	fi, err = f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(fi.IsDir(), check.Equals, true)

	for _, nx := range []string{
		path + "/Unrestricted public data",
		path + "/Unrestricted public data/does not exist",
		path + "/A Project/does not exist",
	} {
		c.Log(nx)
		f, err = s.fs.Open(nx)
		c.Check(err, check.NotNil)
		c.Check(os.IsNotExist(err), check.Equals, true)
	}
}

func (s *SiteFSSuite) TestProjectReaddirAfterLoadOne(c *check.C) {
	f, err := s.fs.Open("/users/active/A Project/A Subproject")
	c.Assert(err, check.IsNil)
	defer f.Close()
	f, err = s.fs.Open("/users/active/A Project/Project does not exist")
	c.Assert(err, check.NotNil)
	f, err = s.fs.Open("/users/active/A Project/A Subproject")
	c.Assert(err, check.IsNil)
	defer f.Close()
	f, err = s.fs.Open("/users/active/A Project")
	c.Assert(err, check.IsNil)
	defer f.Close()
	fis, err := f.Readdir(-1)
	c.Assert(err, check.IsNil)
	c.Logf("%#v", fis)
	var foundSubproject, foundCollection bool
	for _, fi := range fis {
		switch fi.Name() {
		case "A Subproject":
			foundSubproject = true
		case "collection_to_move_around":
			foundCollection = true
		}
	}
	c.Check(foundSubproject, check.Equals, true)
	c.Check(foundCollection, check.Equals, true)
}

func (s *SiteFSSuite) TestSlashInName(c *check.C) {
	badCollection := Collection{
		Name:      "bad/collection",
		OwnerUUID: arvadostest.AProjectUUID,
	}
	err := s.client.RequestAndDecode(&badCollection, "POST", "arvados/v1/collections", s.client.UpdateBody(&badCollection), nil)
	c.Assert(err, check.IsNil)
	defer s.client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+badCollection.UUID, nil, nil)

	badProject := Group{
		Name:       "bad/project",
		GroupClass: "project",
		OwnerUUID:  arvadostest.AProjectUUID,
	}
	err = s.client.RequestAndDecode(&badProject, "POST", "arvados/v1/groups", s.client.UpdateBody(&badProject), nil)
	c.Assert(err, check.IsNil)
	defer s.client.RequestAndDecode(nil, "DELETE", "arvados/v1/groups/"+badProject.UUID, nil, nil)

	dir, err := s.fs.Open("/users/active/A Project")
	c.Assert(err, check.IsNil)
	fis, err := dir.Readdir(-1)
	c.Check(err, check.IsNil)
	for _, fi := range fis {
		c.Logf("fi.Name() == %q", fi.Name())
		c.Check(strings.Contains(fi.Name(), "/"), check.Equals, false)
	}
}

func (s *SiteFSSuite) TestProjectUpdatedByOther(c *check.C) {
	s.fs.MountProject("home", "")

	project, err := s.fs.OpenFile("/home/A Project", 0, 0)
	c.Assert(err, check.IsNil)

	_, err = s.fs.Open("/home/A Project/oob")
	c.Check(err, check.NotNil)

	oob := Collection{
		Name:      "oob",
		OwnerUUID: arvadostest.AProjectUUID,
	}
	err = s.client.RequestAndDecode(&oob, "POST", "arvados/v1/collections", s.client.UpdateBody(&oob), nil)
	c.Assert(err, check.IsNil)
	defer s.client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+oob.UUID, nil, nil)

	err = project.Sync()
	c.Check(err, check.IsNil)
	f, err := s.fs.Open("/home/A Project/oob")
	c.Assert(err, check.IsNil)
	fi, err := f.Stat()
	c.Assert(err, check.IsNil)
	c.Check(fi.IsDir(), check.Equals, true)
	f.Close()

	wf, err := s.fs.OpenFile("/home/A Project/oob/test.txt", os.O_CREATE|os.O_RDWR, 0700)
	c.Assert(err, check.IsNil)
	_, err = wf.Write([]byte("hello oob\n"))
	c.Check(err, check.IsNil)
	err = wf.Close()
	c.Check(err, check.IsNil)

	// Delete test.txt behind s.fs's back by updating the
	// collection record with the old (empty) ManifestText.
	err = s.client.RequestAndDecode(nil, "PATCH", "arvados/v1/collections/"+oob.UUID, s.client.UpdateBody(&oob), nil)
	c.Assert(err, check.IsNil)

	err = project.Sync()
	c.Check(err, check.IsNil)
	_, err = s.fs.Open("/home/A Project/oob/test.txt")
	c.Check(err, check.NotNil)
	_, err = s.fs.Open("/home/A Project/oob")
	c.Check(err, check.IsNil)

	err = s.client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+oob.UUID, nil, nil)
	c.Assert(err, check.IsNil)

	err = project.Sync()
	c.Check(err, check.IsNil)
	_, err = s.fs.Open("/home/A Project/oob")
	c.Check(err, check.NotNil)
}
