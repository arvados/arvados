// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/check.v1"
)

func (s *IntegrationSuite) checkCacheMetrics(c *check.C, regs ...string) {
	s.handler.Cache.updateGauges()
	reg := s.handler.Cache.registry
	mfs, err := reg.Gather()
	c.Check(err, check.IsNil)
	buf := &bytes.Buffer{}
	enc := expfmt.NewEncoder(buf, expfmt.FmtText)
	for _, mf := range mfs {
		c.Check(enc.Encode(mf), check.IsNil)
	}
	mm := buf.String()
	// Remove comments to make the "value vs. regexp" failure
	// output easier to read.
	mm = regexp.MustCompile(`(?m)^#.*\n`).ReplaceAllString(mm, "")
	for _, reg := range regs {
		c.Check(mm, check.Matches, `(?ms).*keepweb_sessions_`+reg+`\n.*`)
	}
}

func (s *IntegrationSuite) TestCache(c *check.C) {
	// Hit the same collection 5 times using the same token. Only
	// the first req should cause an API call; the next 4 should
	// hit all caches.
	u := mustParseURL("http://" + arvadostest.FooCollection + ".keep-web.example/foo")
	req := &http.Request{
		Method:     "GET",
		Host:       u.Host,
		URL:        u,
		RequestURI: u.RequestURI(),
		Header: http.Header{
			"Authorization": {"Bearer " + arvadostest.ActiveToken},
		},
	}
	for i := 0; i < 5; i++ {
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusOK)
	}
	s.checkCacheMetrics(c,
		"hits 4",
		"misses 1",
		"active 1")

	// Hit a shared collection 3 times using PDH, using a
	// different token.
	u2 := mustParseURL("http://" + strings.Replace(arvadostest.BarFileCollectionPDH, "+", "-", 1) + ".keep-web.example/bar")
	req2 := &http.Request{
		Method:     "GET",
		Host:       u2.Host,
		URL:        u2,
		RequestURI: u2.RequestURI(),
		Header: http.Header{
			"Authorization": {"Bearer " + arvadostest.SpectatorToken},
		},
	}
	for i := 0; i < 3; i++ {
		resp2 := httptest.NewRecorder()
		s.handler.ServeHTTP(resp2, req2)
		c.Check(resp2.Code, check.Equals, http.StatusOK)
	}
	s.checkCacheMetrics(c,
		"hits 6",
		"misses 2",
		"active 2")

	// Alternating between two collections/tokens N times should
	// use the existing sessions.
	for i := 0; i < 7; i++ {
		resp := httptest.NewRecorder()
		s.handler.ServeHTTP(resp, req)
		c.Check(resp.Code, check.Equals, http.StatusOK)

		resp2 := httptest.NewRecorder()
		s.handler.ServeHTTP(resp2, req2)
		c.Check(resp2.Code, check.Equals, http.StatusOK)
	}
	s.checkCacheMetrics(c,
		"hits 20",
		"misses 2",
		"active 2")
}

func (s *IntegrationSuite) TestForceReloadPDH(c *check.C) {
	filename := strings.Replace(time.Now().Format(time.RFC3339Nano), ":", ".", -1)
	manifest := ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:" + filename + "\n"
	pdh := arvados.PortableDataHash(manifest)
	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveToken

	_, resp := s.do("GET", "http://"+strings.Replace(pdh, "+", "-", 1)+".keep-web.example/"+filename, arvadostest.ActiveToken, nil)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)

	var coll arvados.Collection
	err := client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": manifest,
		},
	})
	c.Assert(err, check.IsNil)
	defer client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+coll.UUID, nil, nil)
	c.Assert(coll.PortableDataHash, check.Equals, pdh)

	_, resp = s.do("GET", "http://"+strings.Replace(pdh, "+", "-", 1)+".keep-web.example/"+filename, "", http.Header{
		"Authorization": {"Bearer " + arvadostest.ActiveToken},
		"Cache-Control": {"must-revalidate"},
	})
	c.Check(resp.Code, check.Equals, http.StatusOK)

	_, resp = s.do("GET", "http://"+strings.Replace(pdh, "+", "-", 1)+".keep-web.example/missingfile", "", http.Header{
		"Authorization": {"Bearer " + arvadostest.ActiveToken},
		"Cache-Control": {"must-revalidate"},
	})
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
}

func (s *IntegrationSuite) TestForceReloadUUID(c *check.C) {
	client := arvados.NewClientFromEnv()
	client.AuthToken = arvadostest.ActiveToken
	var coll arvados.Collection
	err := client.RequestAndDecode(&coll, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:oldfile\n",
		},
	})
	c.Assert(err, check.IsNil)
	defer client.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+coll.UUID, nil, nil)

	_, resp := s.do("GET", "http://"+coll.UUID+".keep-web.example/newfile", arvadostest.ActiveToken, nil)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	_, resp = s.do("GET", "http://"+coll.UUID+".keep-web.example/oldfile", arvadostest.ActiveToken, nil)
	c.Check(resp.Code, check.Equals, http.StatusOK)
	_, resp = s.do("GET", "http://"+coll.UUID+".keep-web.example/newfile", arvadostest.ActiveToken, nil)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	err = client.RequestAndDecode(&coll, "PATCH", "arvados/v1/collections/"+coll.UUID, nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:oldfile 0:0:newfile\n",
		},
	})
	c.Assert(err, check.IsNil)
	_, resp = s.do("GET", "http://"+coll.UUID+".keep-web.example/newfile", arvadostest.ActiveToken, nil)
	c.Check(resp.Code, check.Equals, http.StatusNotFound)
	_, resp = s.do("GET", "http://"+coll.UUID+".keep-web.example/newfile", "", http.Header{
		"Authorization": {"Bearer " + arvadostest.ActiveToken},
		"Cache-Control": {"must-revalidate"},
	})
	c.Check(resp.Code, check.Equals, http.StatusOK)
}
