// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bytes"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"gopkg.in/check.v1"
)

func (s *UnitSuite) checkCacheMetrics(c *check.C, reg *prometheus.Registry, regs ...string) {
	mfs, err := reg.Gather()
	c.Check(err, check.IsNil)
	buf := &bytes.Buffer{}
	enc := expfmt.NewEncoder(buf, expfmt.FmtText)
	for _, mf := range mfs {
		c.Check(enc.Encode(mf), check.IsNil)
	}
	mm := buf.String()
	for _, reg := range regs {
		c.Check(mm, check.Matches, `(?ms).*collectioncache_`+reg+`\n.*`)
	}
}

func (s *UnitSuite) TestCache(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)

	cache := newConfig(s.Config).Cache
	cache.registry = prometheus.NewRegistry()

	// Hit the same collection 5 times using the same token. Only
	// the first req should cause an API call; the next 4 should
	// hit all caches.
	arv.ApiToken = arvadostest.AdminToken
	var coll *arvados.Collection
	for i := 0; i < 5; i++ {
		coll, err = cache.Get(arv, arvadostest.FooCollection, false)
		c.Check(err, check.Equals, nil)
		c.Assert(coll, check.NotNil)
		c.Check(coll.PortableDataHash, check.Equals, arvadostest.FooCollectionPDH)
		c.Check(coll.ManifestText[:2], check.Equals, ". ")
	}
	s.checkCacheMetrics(c, cache.registry,
		"requests 5",
		"hits 4",
		"pdh_hits 4",
		"api_calls 1")

	// Hit the same collection 2 more times, this time requesting
	// it by PDH and using a different token. The first req should
	// miss the permission cache and fetch the new manifest; the
	// second should hit the Collection cache and skip the API
	// lookup.
	arv.ApiToken = arvadostest.ActiveToken

	coll2, err := cache.Get(arv, arvadostest.FooCollectionPDH, false)
	c.Check(err, check.Equals, nil)
	c.Assert(coll2, check.NotNil)
	c.Check(coll2.PortableDataHash, check.Equals, arvadostest.FooCollectionPDH)
	c.Check(coll2.ManifestText[:2], check.Equals, ". ")
	c.Check(coll2.ManifestText, check.Not(check.Equals), coll.ManifestText)

	s.checkCacheMetrics(c, cache.registry,
		"requests 6",
		"hits 4",
		"pdh_hits 4",
		"api_calls 2")

	coll2, err = cache.Get(arv, arvadostest.FooCollectionPDH, false)
	c.Check(err, check.Equals, nil)
	c.Assert(coll2, check.NotNil)
	c.Check(coll2.PortableDataHash, check.Equals, arvadostest.FooCollectionPDH)
	c.Check(coll2.ManifestText[:2], check.Equals, ". ")

	s.checkCacheMetrics(c, cache.registry,
		"requests 7",
		"hits 5",
		"pdh_hits 4",
		"api_calls 2")

	// Alternating between two collections N times should produce
	// only 2 more API calls.
	arv.ApiToken = arvadostest.AdminToken
	for i := 0; i < 20; i++ {
		var target string
		if i%2 == 0 {
			target = arvadostest.HelloWorldCollection
		} else {
			target = arvadostest.FooBarDirCollection
		}
		_, err := cache.Get(arv, target, false)
		c.Check(err, check.Equals, nil)
	}
	s.checkCacheMetrics(c, cache.registry,
		"requests 27",
		"hits 23",
		"pdh_hits 22",
		"api_calls 4")
}

func (s *UnitSuite) TestCacheForceReloadByPDH(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)

	cache := newConfig(s.Config).Cache
	cache.registry = prometheus.NewRegistry()

	for _, forceReload := range []bool{false, true, false, true} {
		_, err := cache.Get(arv, arvadostest.FooCollectionPDH, forceReload)
		c.Check(err, check.Equals, nil)
	}

	s.checkCacheMetrics(c, cache.registry,
		"requests 4",
		"hits 3",
		"pdh_hits 0",
		"api_calls 1")
}

func (s *UnitSuite) TestCacheForceReloadByUUID(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)

	cache := newConfig(s.Config).Cache
	cache.registry = prometheus.NewRegistry()

	for _, forceReload := range []bool{false, true, false, true} {
		_, err := cache.Get(arv, arvadostest.FooCollection, forceReload)
		c.Check(err, check.Equals, nil)
	}

	s.checkCacheMetrics(c, cache.registry,
		"requests 4",
		"hits 3",
		"pdh_hits 3",
		"api_calls 3")
}
