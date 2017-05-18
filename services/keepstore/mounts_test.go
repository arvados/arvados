package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&MountsSuite{})

type MountsSuite struct {
	vm  VolumeManager
	rtr http.Handler
}

func (s *MountsSuite) SetUpTest(c *check.C) {
	s.vm = MakeTestVolumeManager(2)
	KeepVM = s.vm
	s.rtr = MakeRESTRouter()
	theConfig.systemAuthToken = arvadostest.DataManagerToken
}

func (s *MountsSuite) TearDownTest(c *check.C) {
	s.vm.Close()
	KeepVM = nil
	theConfig = DefaultConfig()
	theConfig.Start()
}

func (s *MountsSuite) TestMounts(c *check.C) {
	vols := s.vm.AllWritable()
	vols[0].Put(context.Background(), TestHash, TestBlock)
	vols[1].Put(context.Background(), TestHash2, TestBlock2)

	resp := s.call("GET", "/mounts", "")
	c.Check(resp.Code, check.Equals, http.StatusOK)
	var mntList []struct {
		UUID        string
		DeviceID    string
		ReadOnly    bool
		Replication int
		Tier        int
	}
	err := json.Unmarshal(resp.Body.Bytes(), &mntList)
	c.Assert(err, check.IsNil)
	c.Assert(len(mntList), check.Equals, 2)
	for _, m := range mntList {
		c.Check(len(m.UUID), check.Equals, 27)
		c.Check(m.UUID[:12], check.Equals, "zzzzz-ivpuk-")
		c.Check(m.DeviceID, check.Equals, "mock-device-id")
		c.Check(m.ReadOnly, check.Equals, false)
		c.Check(m.Replication, check.Equals, 1)
		c.Check(m.Tier, check.Equals, 1)
	}
	c.Check(mntList[0].UUID, check.Not(check.Equals), mntList[1].UUID)

	// Bad auth
	for _, tok := range []string{"", "xyzzy"} {
		resp = s.call("GET", "/mounts/"+mntList[1].UUID+"/blocks", tok)
		c.Check(resp.Code, check.Equals, http.StatusUnauthorized)
		c.Check(resp.Body.String(), check.Equals, "Unauthorized\n")
	}

	tok := arvadostest.DataManagerToken

	// Nonexistent mount UUID
	resp = s.call("GET", "/mounts/X/blocks", tok)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	c.Check(resp.Body.String(), check.Equals, "mount not found\n")

	// Complete index of first mount
	resp = s.call("GET", "/mounts/"+mntList[0].UUID+"/blocks", tok)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, TestHash+`\+[0-9]+ [0-9]+\n\n`)

	// Partial index of first mount (one block matches prefix)
	resp = s.call("GET", "/mounts/"+mntList[0].UUID+"/blocks?prefix="+TestHash[:2], tok)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, TestHash+`\+[0-9]+ [0-9]+\n\n`)

	// Complete index of second mount (note trailing slash)
	resp = s.call("GET", "/mounts/"+mntList[1].UUID+"/blocks/", tok)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Matches, TestHash2+`\+[0-9]+ [0-9]+\n\n`)

	// Partial index of second mount (no blocks match prefix)
	resp = s.call("GET", "/mounts/"+mntList[1].UUID+"/blocks/?prefix="+TestHash[:2], tok)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	c.Check(resp.Body.String(), check.Equals, "\n")
}

func (s *MountsSuite) call(method, path, tok string) *httptest.ResponseRecorder {
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	if tok != "" {
		req.Header.Set("Authorization", "OAuth2 "+tok)
	}
	s.rtr.ServeHTTP(resp, req)
	return resp
}
