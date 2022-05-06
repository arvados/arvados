// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	check "gopkg.in/check.v1"
)

func (s *IntegrationSuite) TestRanges(c *check.C) {
	blocksize := 1000000
	var uuid string
	{
		testdata := make([]byte, blocksize)
		for i := 0; i < blocksize; i++ {
			testdata[i] = byte(' ')
		}
		copy(testdata[1:4], []byte("foo"))
		arv, err := arvadosclient.MakeArvadosClient()
		c.Assert(err, check.Equals, nil)
		arv.ApiToken = arvadostest.ActiveToken
		kc, err := keepclient.MakeKeepClient(arv)
		c.Assert(err, check.Equals, nil)
		loc, _, err := kc.PutB(testdata[:])
		c.Assert(err, check.Equals, nil)
		loc2, _, err := kc.PutB([]byte{'Z'})
		c.Assert(err, check.Equals, nil)

		mtext := fmt.Sprintf(". %s %s %s %s %s 1:%d:testdata.bin 0:1:space.txt\n", loc, loc, loc, loc, loc2, blocksize*4)
		coll := map[string]interface{}{}
		err = arv.Create("collections",
			map[string]interface{}{
				"collection": map[string]interface{}{
					"name":          "test data for keep-web TestRanges",
					"manifest_text": mtext,
				},
			}, &coll)
		c.Assert(err, check.Equals, nil)
		uuid = coll["uuid"].(string)
		defer arv.Delete("collections", uuid, nil, nil)
	}

	url := mustParseURL("http://" + uuid + ".collections.example.com/testdata.bin")
	for _, trial := range []struct {
		header     string
		expectObey bool
		expectBody string
	}{
		{"0-2", true, "foo"},
		{"-2", true, " Z"},
		{"1-4", true, "oo  "},
		{"z-y", false, ""},
		{"1000000-1000003", true, "foo "},
		{"999999-1000003", true, " foo "},
		{"2000000-2000003", true, "foo "},
		{"1999999-2000002", true, " foo"},
		{"3999998-3999999", true, " Z"},
		{"3999998-4000004", true, " Z"},
		{"3999998-", true, " Z"},
	} {
		c.Logf("trial: %#v", trial)
		resp := httptest.NewRecorder()
		req := &http.Request{
			Method:     "GET",
			URL:        url,
			Host:       url.Host,
			RequestURI: url.RequestURI(),
			Header: http.Header{
				"Authorization": {"OAuth2 " + arvadostest.ActiveToken},
				"Range":         {"bytes=" + trial.header},
			},
		}
		s.handler.ServeHTTP(resp, req)
		if trial.expectObey {
			c.Check(resp.Code, check.Equals, http.StatusPartialContent)
			c.Check(resp.Body.Len(), check.Equals, len(trial.expectBody))
			if resp.Body.Len() > 1000 {
				c.Check(resp.Body.String()[:1000]+"[...]", check.Equals, trial.expectBody)
			} else {
				c.Check(resp.Body.String(), check.Equals, trial.expectBody)
			}
		} else {
			c.Check(resp.Code, check.Equals, http.StatusRequestedRangeNotSatisfiable)
		}
	}
}
