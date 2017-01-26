package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
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
		copy(testdata[:3], []byte("foo"))
		arv, err := arvadosclient.MakeArvadosClient()
		c.Assert(err, check.Equals, nil)
		arv.ApiToken = arvadostest.ActiveToken
		kc, err := keepclient.MakeKeepClient(arv)
		c.Assert(err, check.Equals, nil)
		loc, _, err := kc.PutB(testdata[:])
		c.Assert(err, check.Equals, nil)

		mtext := "."
		for i := 0; i < 4; i++ {
			mtext = mtext + " " + loc
		}
		mtext = mtext + fmt.Sprintf(" 0:%d:testdata.bin\n", blocksize*4)
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
		{"1-4", true, "oo  "},
		{"z-y", false, ""},
		{"1000000-1000003", true, "foo "},
		{"999999-1000003", true, " foo "},
		{"2000000-2000003", true, "foo "},
		{"1999999-2000002", true, " foo"},
		{"3999998-3999999", true, "  "},
		{"3999998-4000004", true, "  "},
		{"3999998-", true, "  "},
	} {
		c.Logf("%+v", trial)
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
		s.testServer.Handler.ServeHTTP(resp, req)
		if trial.expectObey {
			c.Check(resp.Code, check.Equals, http.StatusPartialContent)
			c.Check(resp.Body.Len(), check.Equals, len(trial.expectBody))
			c.Check(resp.Body.String()[:len(trial.expectBody)], check.Equals, trial.expectBody)
		} else {
			c.Check(resp.Code, check.Equals, http.StatusOK)
			c.Check(resp.Body.Len(), check.Equals, blocksize*4)
		}
	}
}
