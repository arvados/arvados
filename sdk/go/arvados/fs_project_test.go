// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
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

	checkOpen := func(path string, exists bool) {
		f, err := s.fs.Open(path)
		if exists {
			if c.Check(err, check.IsNil) {
				c.Check(f.Close(), check.IsNil)
			}
		} else {
			c.Check(err, check.Equals, os.ErrNotExist)
		}
	}

	checkDirContains := func(parent, child string, exists bool) {
		f, err := s.fs.Open(parent)
		if !c.Check(err, check.IsNil) {
			return
		}
		ents, err := f.Readdir(-1)
		if !c.Check(err, check.IsNil) {
			return
		}
		for _, ent := range ents {
			if !exists {
				c.Check(ent.Name(), check.Not(check.Equals), child)
				if child == "" {
					// no children are expected
					c.Errorf("child %q found in parent %q", child, parent)
				}
			} else if ent.Name() == child {
				return
			}
		}
		if exists {
			c.Errorf("child %q not found in parent %q", child, parent)
		}
	}

	checkOpen("/users/active/This filter group/baz_file", true)
	checkOpen("/users/active/This filter group/A Subproject", true)
	checkOpen("/users/active/This filter group/A Project", false)
	s.fs.MountProject("fg", fixtureThisFilterGroupUUID)
	checkOpen("/fg/baz_file", true)
	checkOpen("/fg/A Subproject", true)
	checkOpen("/fg/A Project", false)
	s.fs.MountProject("home", "")
	checkOpen("/home/A filter group with an is_a collection filter/baz_file", true)
	checkOpen("/home/A filter group with an is_a collection filter/baz_file/baz", true)
	checkOpen("/home/A filter group with an is_a collection filter/A Subproject", false)
	checkOpen("/home/A filter group with an is_a collection filter/A Project", false)

	// An empty filter means everything that is visible should be returned.
	checkOpen("/users/active/A filter group without filters/baz_file", true)
	checkOpen("/users/active/A filter group without filters/A Subproject", true)
	checkOpen("/users/active/A filter group without filters/A Project", true)
	s.fs.MountProject("fg2", fixtureAFilterGroupTwoUUID)
	checkOpen("/fg2/baz_file", true)
	checkOpen("/fg2/A Subproject", true)
	checkOpen("/fg2/A Project", true)

	// If a filter group matches itself or one of its ancestors,
	// the matched item appears as an empty directory.
	checkDirContains("/users/active/A filter group without filters", "A filter group without filters", true)
	checkOpen("/users/active/A filter group without filters/A filter group without filters", true)
	checkOpen("/users/active/A filter group without filters/A filter group without filters/baz_file", false)
	checkDirContains("/users/active/A filter group without filters/A filter group without filters", "", false)

	// An 'is_a' 'arvados#collection' filter means only collections should be returned.
	checkOpen("/users/active/A filter group with an is_a collection filter/baz_file", true)
	checkOpen("/users/active/A filter group with an is_a collection filter/baz_file/baz", true)
	checkOpen("/users/active/A filter group with an is_a collection filter/A Subproject", false)
	checkOpen("/users/active/A filter group with an is_a collection filter/A Project", false)
	s.fs.MountProject("fg3", fixtureAFilterGroupThreeUUID)
	checkOpen("/fg3/baz_file", true)
	checkOpen("/fg3/baz_file/baz", true)
	checkOpen("/fg3/A Subproject", false)

	// An 'exists' 'arvados#collection' filter means only collections with certain properties should be returned.
	s.fs.MountProject("fg4", fixtureAFilterGroupFourUUID)
	checkOpen("/fg4/collection with list property with odd values", true)
	checkOpen("/fg4/collection with list property with even values", true)
	checkOpen("/fg4/baz_file", false)

	// A 'contains' 'arvados#collection' filter means only collections with certain properties should be returned.
	s.fs.MountProject("fg5", fixtureAFilterGroupFiveUUID)
	checkOpen("/fg5/collection with list property with odd values", true)
	checkOpen("/fg5/collection with list property with string value", true)
	checkOpen("/fg5/collection with prop2 5", false)
	checkOpen("/fg5/collection with list property with even values", false)
}

func (s *SiteFSSuite) TestCurrentUserHome(c *check.C) {
	s.fs.MountProject("home", "")
	s.testHomeProject(c, "/home", "home")
}

func (s *SiteFSSuite) TestUsersDir(c *check.C) {
	// /users/active is a hardlink to a dir whose name is the UUID
	// of the active user
	s.testHomeProject(c, "/users/active", fixtureActiveUserUUID)
}

func (s *SiteFSSuite) testHomeProject(c *check.C, path, expectRealName string) {
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
	c.Check(fi.Name(), check.Equals, expectRealName)

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
	f, err = s.fs.Open("/home/A Project/oob/test.txt")
	if c.Check(err, check.IsNil) {
		f.Close()
	}

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

	// Sync again to reload collection.
	err = project.Sync()
	c.Check(err, check.IsNil)

	// Check test.txt deletion is reflected in fs.
	_, err = s.fs.Open("/home/A Project/oob/test.txt")
	c.Check(err, check.NotNil)
	f, err = s.fs.Open("/home/A Project/oob")
	if c.Check(err, check.IsNil) {
		f.Close()
	}

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
	c.Check(err, ErrorIs, ErrInvalidOperation)

	err = s.fs.Mkdir("/home/A Project/newdirname", 0)
	c.Check(err, ErrorIs, ErrInvalidOperation)

	err = s.fs.Mkdir("/by_id/newdirname", 0)
	c.Check(err, ErrorIs, ErrInvalidOperation)

	err = s.fs.Mkdir("/by_id/"+fixtureAProjectUUID+"/newdirname", 0)
	c.Check(err, ErrorIs, ErrInvalidOperation)

	_, err = s.fs.OpenFile("/home/A Project", 0, 0)
	c.Check(err, check.IsNil)
}

type errorIsChecker struct {
	*check.CheckerInfo
}

var ErrorIs check.Checker = errorIsChecker{
	&check.CheckerInfo{Name: "ErrorIs", Params: []string{"value", "target"}},
}

func (checker errorIsChecker) Check(params []interface{}, names []string) (result bool, errStr string) {
	err, ok := params[0].(error)
	if !ok {
		return false, ""
	}
	target, ok := params[1].(error)
	if !ok {
		return false, ""
	}
	return errors.Is(err, target), ""
}
