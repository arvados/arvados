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

func (s *SiteFSSuite) TestFilterGroup(c *check.C) {
	// Make sure that a collection and group that match the filter are present,
	// and that a group that does not match the filter is not present.
	s.fs.MountProject("fg", fixtureThisFilterGroupUUID)

	_, err := s.fs.OpenFile("/fg/baz_file", 0, 0)
	c.Assert(err, check.IsNil)

	_, err = s.fs.OpenFile("/fg/A Subproject", 0, 0)
	c.Assert(err, check.IsNil)

	_, err = s.fs.OpenFile("/fg/A Project", 0, 0)
	c.Assert(err, check.Not(check.IsNil))

	// An empty filter means everything that is visible should be returned.
	s.fs.MountProject("fg2", fixtureAFilterGroupTwoUUID)

	_, err = s.fs.OpenFile("/fg2/baz_file", 0, 0)
	c.Assert(err, check.IsNil)

	_, err = s.fs.OpenFile("/fg2/A Subproject", 0, 0)
	c.Assert(err, check.IsNil)

	_, err = s.fs.OpenFile("/fg2/A Project", 0, 0)
	c.Assert(err, check.IsNil)

	// An 'is_a' 'arvados#collection' filter means only collections should be returned.
	s.fs.MountProject("fg3", fixtureAFilterGroupThreeUUID)

	_, err = s.fs.OpenFile("/fg3/baz_file", 0, 0)
	c.Assert(err, check.IsNil)

	_, err = s.fs.OpenFile("/fg3/A Subproject", 0, 0)
	c.Assert(err, check.Not(check.IsNil))

	// An 'exists' 'arvados#collection' filter means only collections with certain properties should be returned.
	s.fs.MountProject("fg4", fixtureAFilterGroupFourUUID)

	_, err = s.fs.Stat("/fg4/collection with list property with odd values")
	c.Assert(err, check.IsNil)

	_, err = s.fs.Stat("/fg4/collection with list property with even values")
	c.Assert(err, check.IsNil)

	// A 'contains' 'arvados#collection' filter means only collections with certain properties should be returned.
	s.fs.MountProject("fg5", fixtureAFilterGroupFiveUUID)

	_, err = s.fs.Stat("/fg5/collection with list property with odd values")
	c.Assert(err, check.IsNil)

	_, err = s.fs.Stat("/fg5/collection with list property with string value")
	c.Assert(err, check.IsNil)

	_, err = s.fs.Stat("/fg5/collection with prop2 5")
	c.Assert(err, check.Not(check.IsNil))

	_, err = s.fs.Stat("/fg5/collection with list property with even values")
	c.Assert(err, check.Not(check.IsNil))
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
	c.Assert(err, check.IsNil)
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
	var badCollection Collection
	err := s.client.RequestAndDecode(&badCollection, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"name":       "bad/collection",
			"owner_uuid": fixtureAProjectUUID,
		},
	})
	c.Assert(err, check.IsNil)
	defer s.client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+badCollection.UUID, nil, nil)

	var badProject Group
	err = s.client.RequestAndDecode(&badProject, "POST", "arvados/v1/groups", nil, map[string]interface{}{
		"group": map[string]string{
			"name":        "bad/project",
			"group_class": "project",
			"owner_uuid":  fixtureAProjectUUID,
		},
	})
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

	// Make a new fs (otherwise content will still be cached from
	// above) and enable "/" replacement string.
	s.fs = s.client.SiteFileSystem(s.kc)
	s.fs.ForwardSlashNameSubstitution("___")
	dir, err = s.fs.Open("/users/active/A Project/bad___collection")
	if c.Check(err, check.IsNil) {
		_, err = dir.Readdir(-1)
		c.Check(err, check.IsNil)
	}
	dir, err = s.fs.Open("/users/active/A Project/bad___project")
	if c.Check(err, check.IsNil) {
		_, err = dir.Readdir(-1)
		c.Check(err, check.IsNil)
	}
}

func (s *SiteFSSuite) TestProjectUpdatedByOther(c *check.C) {
	s.fs.MountProject("home", "")

	project, err := s.fs.OpenFile("/home/A Project", 0, 0)
	c.Assert(err, check.IsNil)

	_, err = s.fs.Open("/home/A Project/oob")
	c.Check(err, check.NotNil)

	var oob Collection
	err = s.client.RequestAndDecode(&oob, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"name":       "oob",
			"owner_uuid": fixtureAProjectUUID,
		},
	})
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

	err = project.Sync()
	c.Check(err, check.IsNil)
	_, err = s.fs.Open("/home/A Project/oob/test.txt")
	c.Check(err, check.IsNil)

	// Sync again to mark the project dir as stale, so the
	// collection gets reloaded from the controller on next
	// lookup.
	err = project.Sync()
	c.Check(err, check.IsNil)

	// Ensure collection was flushed by Sync
	var latest Collection
	err = s.client.RequestAndDecode(&latest, "GET", "arvados/v1/collections/"+oob.UUID, nil, nil)
	c.Check(err, check.IsNil)
	c.Check(latest.ManifestText, check.Matches, `.*:test.txt.*\n`)

	// Delete test.txt behind s.fs's back by updating the
	// collection record with an empty ManifestText.
	err = s.client.RequestAndDecode(nil, "PATCH", "arvados/v1/collections/"+oob.UUID, nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text":      "",
			"portable_data_hash": "d41d8cd98f00b204e9800998ecf8427e+0",
		},
	})
	c.Assert(err, check.IsNil)

	_, err = s.fs.Open("/home/A Project/oob/test.txt")
	c.Check(err, check.NotNil)
	_, err = s.fs.Open("/home/A Project/oob")
	c.Check(err, check.IsNil)

	err = s.client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+oob.UUID, nil, nil)
	c.Assert(err, check.IsNil)

	wf, err = s.fs.OpenFile("/home/A Project/oob/test.txt", os.O_CREATE|os.O_RDWR, 0700)
	c.Assert(err, check.IsNil)
	err = wf.Close()
	c.Check(err, check.IsNil)

	err = project.Sync()
	c.Check(err, check.NotNil) // can't update the deleted collection
	_, err = s.fs.Open("/home/A Project/oob")
	c.Check(err, check.IsNil) // parent dir still has old collection -- didn't reload, because Sync failed
}

func (s *SiteFSSuite) TestProjectUnsupportedOperations(c *check.C) {
	s.fs.MountByID("by_id")
	s.fs.MountProject("home", "")

	_, err := s.fs.OpenFile("/home/A Project/newfilename", os.O_CREATE|os.O_RDWR, 0)
	c.Check(err, check.ErrorMatches, "invalid argument")

	err = s.fs.Mkdir("/home/A Project/newdirname", 0)
	c.Check(err, check.ErrorMatches, "invalid argument")

	err = s.fs.Mkdir("/by_id/newdirname", 0)
	c.Check(err, check.ErrorMatches, "invalid argument")

	err = s.fs.Mkdir("/by_id/"+fixtureAProjectUUID+"/newdirname", 0)
	c.Check(err, check.ErrorMatches, "invalid argument")

	_, err = s.fs.OpenFile("/home/A Project", 0, 0)
	c.Check(err, check.IsNil)
}
