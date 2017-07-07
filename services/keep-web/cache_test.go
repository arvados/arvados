// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/arvadostest"
	"gopkg.in/check.v1"
)

func (s *UnitSuite) TestCache(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)

	cache := DefaultConfig().Cache

	// Hit the same collection 5 times using the same token. Only
	// the first req should cause an API call; the next 4 should
	// hit all caches.
	arv.ApiToken = arvadostest.AdminToken
	var coll *arvados.Collection
	for i := 0; i < 5; i++ {
		coll, err = cache.Get(arv, arvadostest.FooCollection, false)
		c.Check(err, check.Equals, nil)
		c.Assert(coll, check.NotNil)
		c.Check(coll.PortableDataHash, check.Equals, arvadostest.FooPdh)
		c.Check(coll.ManifestText[:2], check.Equals, ". ")
	}
	c.Check(cache.Stats().Requests, check.Equals, uint64(5))
	c.Check(cache.Stats().CollectionHits, check.Equals, uint64(4))
	c.Check(cache.Stats().PermissionHits, check.Equals, uint64(4))
	c.Check(cache.Stats().PDHHits, check.Equals, uint64(4))
	c.Check(cache.Stats().APICalls, check.Equals, uint64(1))

	// Hit the same collection 2 more times, this time requesting
	// it by PDH and using a different token. The first req should
	// miss the permission cache and fetch the new manifest; the
	// second should hit the Collection cache and skip the API
	// lookup.
	arv.ApiToken = arvadostest.ActiveToken

	coll2, err := cache.Get(arv, arvadostest.FooPdh, false)
	c.Check(err, check.Equals, nil)
	c.Assert(coll2, check.NotNil)
	c.Check(coll2.PortableDataHash, check.Equals, arvadostest.FooPdh)
	c.Check(coll2.ManifestText[:2], check.Equals, ". ")
	c.Check(coll2.ManifestText, check.Not(check.Equals), coll.ManifestText)

	c.Check(cache.Stats().Requests, check.Equals, uint64(5+1))
	c.Check(cache.Stats().CollectionHits, check.Equals, uint64(4+0))
	c.Check(cache.Stats().PermissionHits, check.Equals, uint64(4+0))
	c.Check(cache.Stats().PDHHits, check.Equals, uint64(4+0))
	c.Check(cache.Stats().APICalls, check.Equals, uint64(1+1))

	coll2, err = cache.Get(arv, arvadostest.FooPdh, false)
	c.Check(err, check.Equals, nil)
	c.Assert(coll2, check.NotNil)
	c.Check(coll2.PortableDataHash, check.Equals, arvadostest.FooPdh)
	c.Check(coll2.ManifestText[:2], check.Equals, ". ")

	c.Check(cache.Stats().Requests, check.Equals, uint64(5+2))
	c.Check(cache.Stats().CollectionHits, check.Equals, uint64(4+1))
	c.Check(cache.Stats().PermissionHits, check.Equals, uint64(4+1))
	c.Check(cache.Stats().PDHHits, check.Equals, uint64(4+0))
	c.Check(cache.Stats().APICalls, check.Equals, uint64(1+1))

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
	c.Check(cache.Stats().Requests, check.Equals, uint64(5+2+20))
	c.Check(cache.Stats().CollectionHits, check.Equals, uint64(4+1+18))
	c.Check(cache.Stats().PermissionHits, check.Equals, uint64(4+1+18))
	c.Check(cache.Stats().PDHHits, check.Equals, uint64(4+0+18))
	c.Check(cache.Stats().APICalls, check.Equals, uint64(1+1+2))
}

func (s *UnitSuite) TestCacheForceReloadByPDH(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)

	cache := DefaultConfig().Cache

	for _, forceReload := range []bool{false, true, false, true} {
		_, err := cache.Get(arv, arvadostest.FooPdh, forceReload)
		c.Check(err, check.Equals, nil)
	}

	c.Check(cache.Stats().Requests, check.Equals, uint64(4))
	c.Check(cache.Stats().CollectionHits, check.Equals, uint64(3))
	c.Check(cache.Stats().PermissionHits, check.Equals, uint64(1))
	c.Check(cache.Stats().PDHHits, check.Equals, uint64(0))
	c.Check(cache.Stats().APICalls, check.Equals, uint64(3))
}

func (s *UnitSuite) TestCacheForceReloadByUUID(c *check.C) {
	arv, err := arvadosclient.MakeArvadosClient()
	c.Assert(err, check.Equals, nil)

	cache := DefaultConfig().Cache

	for _, forceReload := range []bool{false, true, false, true} {
		_, err := cache.Get(arv, arvadostest.FooCollection, forceReload)
		c.Check(err, check.Equals, nil)
	}

	c.Check(cache.Stats().Requests, check.Equals, uint64(4))
	c.Check(cache.Stats().CollectionHits, check.Equals, uint64(3))
	c.Check(cache.Stats().PermissionHits, check.Equals, uint64(1))
	c.Check(cache.Stats().PDHHits, check.Equals, uint64(3))
	c.Check(cache.Stats().APICalls, check.Equals, uint64(3))
}
