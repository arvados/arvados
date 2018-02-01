// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

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

func (s *SiteFSSuite) TestHomeProject(c *check.C) {
	f, err := s.fs.Open("/home")
	c.Assert(err, check.IsNil)
	fis, err := f.Readdir(-1)
	c.Check(len(fis), check.Not(check.Equals), 0)

	ok := false
	for _, fi := range fis {
		c.Check(fi.Name(), check.Not(check.Equals), "")
		if fi.Name() == "Unrestricted public data" {
			ok = true
		}
	}
	c.Check(ok, check.Equals, true)

	f, err = s.fs.Open("/home/Unrestricted public data/..")
	c.Assert(err, check.IsNil)
	fi, err := f.Stat()
	c.Check(err, check.IsNil)
	c.Check(fi.IsDir(), check.Equals, true)
	c.Check(fi.Name(), check.Equals, "home")

	f, err = s.fs.Open("/home/Unrestricted public data/Subproject in anonymous accessible project")
	c.Check(err, check.IsNil)
	fi, err = f.Stat()
	c.Check(err, check.IsNil)
	c.Check(fi.IsDir(), check.Equals, true)

	for _, nx := range []string{
		"/home/A Project",
		"/home/A Project/does not exist",
		"/home/Unrestricted public data/does not exist",
	} {
		c.Log(nx)
		f, err = s.fs.Open(nx)
		c.Check(err, check.NotNil)
		c.Check(os.IsNotExist(err), check.Equals, true)
	}
}
