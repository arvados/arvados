package main

import (
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
	for i := 0; i < 5; i++ {
		coll, err := cache.Get(arv, arvadostest.FooCollection, false)
		c.Check(err, check.Equals, nil)
		c.Assert(coll, check.NotNil)
		c.Check(coll["portable_data_hash"], check.Equals, arvadostest.FooPdh)
		c.Check(coll["manifest_text"].(string)[:2], check.Equals, ". ")
	}
	c.Check(cache.Stats().Requests, check.Equals, uint64(5))
	c.Check(cache.Stats().CollectionHits, check.Equals, uint64(4))
	c.Check(cache.Stats().PermissionHits, check.Equals, uint64(4))
	c.Check(cache.Stats().PDHHits, check.Equals, uint64(4))
	c.Check(cache.Stats().APICalls, check.Equals, uint64(1))

	// Hit the same collection 2 more times, this time requesting
	// it by PDH and using a different token. The first req should
	// miss the permission cache. Both reqs should hit the
	// Collection cache and skip the API lookup.
	arv.ApiToken = arvadostest.ActiveToken
	for i := 0; i < 2; i++ {
		coll, err := cache.Get(arv, arvadostest.FooPdh, false)
		c.Check(err, check.Equals, nil)
		c.Assert(coll, check.NotNil)
		c.Check(coll["portable_data_hash"], check.Equals, arvadostest.FooPdh)
		c.Check(coll["manifest_text"].(string)[:2], check.Equals, ". ")
	}
	c.Check(cache.Stats().Requests, check.Equals, uint64(7))
	c.Check(cache.Stats().CollectionHits, check.Equals, uint64(6))
	c.Check(cache.Stats().PermissionHits, check.Equals, uint64(5))
	c.Check(cache.Stats().PDHHits, check.Equals, uint64(4))
	c.Check(cache.Stats().APICalls, check.Equals, uint64(2))
}
