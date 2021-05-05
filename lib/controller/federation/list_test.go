// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sort"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&CollectionListSuite{})

type collectionLister struct {
	arvadostest.APIStub
	ItemsToReturn []arvados.Collection
	MaxPageSize   int
}

func (cl *collectionLister) matchFilters(c arvados.Collection, filters []arvados.Filter) bool {
nextfilter:
	for _, f := range filters {
		if f.Attr == "uuid" && f.Operator == "=" {
			s, ok := f.Operand.(string)
			if ok && s == c.UUID {
				continue nextfilter
			}
		} else if f.Attr == "uuid" && f.Operator == "in" {
			if operand, ok := f.Operand.([]string); ok {
				for _, s := range operand {
					if s == c.UUID {
						continue nextfilter
					}
				}
			} else if operand, ok := f.Operand.([]interface{}); ok {
				for _, s := range operand {
					if s, ok := s.(string); ok && s == c.UUID {
						continue nextfilter
					}
				}
			}
		}
		return false
	}
	return true
}

func (cl *collectionLister) CollectionList(ctx context.Context, options arvados.ListOptions) (resp arvados.CollectionList, _ error) {
	cl.APIStub.CollectionList(ctx, options)
	for _, c := range cl.ItemsToReturn {
		if cl.MaxPageSize > 0 && len(resp.Items) >= cl.MaxPageSize {
			break
		}
		if options.Limit >= 0 && int64(len(resp.Items)) >= options.Limit {
			break
		}
		if cl.matchFilters(c, options.Filters) {
			if reflect.DeepEqual(options.Select, []string{"uuid", "name"}) {
				c = arvados.Collection{UUID: c.UUID, Name: c.Name}
			} else if reflect.DeepEqual(options.Select, []string{"name"}) {
				c = arvados.Collection{Name: c.Name}
			} else if len(options.Select) > 0 {
				panic(fmt.Sprintf("not implemented: options=%#v", options))
			}
			resp.Items = append(resp.Items, c)
		}
	}
	return
}

type CollectionListSuite struct {
	FederationSuite
	ids      []string   // aaaaa, bbbbb, ccccc
	uuids    [][]string // [[aa-*, aa-*, aa-*], [bb-*, bb-*, ...], ...]
	backends []*collectionLister
}

func (s *CollectionListSuite) SetUpTest(c *check.C) {
	s.FederationSuite.SetUpTest(c)

	s.ids = nil
	s.uuids = nil
	s.backends = nil
	for i, id := range []string{"aaaaa", "bbbbb", "ccccc"} {
		cl := &collectionLister{}
		s.ids = append(s.ids, id)
		s.uuids = append(s.uuids, nil)
		for j := 0; j < 5; j++ {
			uuid := fmt.Sprintf("%s-4zz18-%s%010d", id, id, j)
			s.uuids[i] = append(s.uuids[i], uuid)
			cl.ItemsToReturn = append(cl.ItemsToReturn, arvados.Collection{
				UUID: uuid,
			})
		}
		s.backends = append(s.backends, cl)
		if i == 0 {
			s.fed.local = cl
		} else if i%1 == 0 {
			// call some backends directly via API
			s.addDirectRemote(c, id, cl)
		} else {
			// call some backends through rpc->router->API
			// to ensure nothing is lost in translation
			s.addHTTPRemote(c, id, cl)
		}
	}
}

type listTrial struct {
	count        string
	limit        int64
	offset       int64
	order        []string
	filters      []arvados.Filter
	selectfields []string
	expectUUIDs  []string
	expectCalls  []int // number of API calls to backends
	expectStatus int
}

func (s *CollectionListSuite) TestCollectionListNoUUIDFilters(c *check.C) {
	s.test(c, listTrial{
		count:       "none",
		limit:       1,
		expectUUIDs: []string{s.uuids[0][0]},
		expectCalls: []int{1, 0, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListOneLocal(c *check.C) {
	s.test(c, listTrial{
		count:       "none",
		limit:       -1,
		filters:     []arvados.Filter{{"uuid", "=", s.uuids[0][0]}},
		expectUUIDs: []string{s.uuids[0][0]},
		expectCalls: []int{1, 0, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListOneRemote(c *check.C) {
	s.test(c, listTrial{
		count:       "none",
		limit:       -1,
		filters:     []arvados.Filter{{"uuid", "=", s.uuids[1][0]}},
		expectUUIDs: []string{s.uuids[1][0]},
		expectCalls: []int{0, 1, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListOneLocalDeselectingUUID(c *check.C) {
	s.test(c, listTrial{
		count:        "none",
		limit:        -1,
		filters:      []arvados.Filter{{"uuid", "=", s.uuids[0][0]}},
		selectfields: []string{"name"},
		expectUUIDs:  []string{""}, // select=name is honored
		expectCalls:  []int{1, 0, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListOneLocalUsingInOperator(c *check.C) {
	s.test(c, listTrial{
		count:       "none",
		limit:       -1,
		filters:     []arvados.Filter{{"uuid", "in", []string{s.uuids[0][0]}}},
		expectUUIDs: []string{s.uuids[0][0]},
		expectCalls: []int{1, 0, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListOneRemoteUsingInOperator(c *check.C) {
	s.test(c, listTrial{
		count:       "none",
		limit:       -1,
		filters:     []arvados.Filter{{"uuid", "in", []string{s.uuids[1][1]}}},
		expectUUIDs: []string{s.uuids[1][1]},
		expectCalls: []int{0, 1, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListOneRemoteDeselectingUUID(c *check.C) {
	s.test(c, listTrial{
		count:        "none",
		limit:        -1,
		filters:      []arvados.Filter{{"uuid", "=", s.uuids[1][0]}},
		selectfields: []string{"name"},
		expectUUIDs:  []string{s.uuids[1][0]}, // uuid is returned, despite not being selected
		expectCalls:  []int{0, 1, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListOneLocalOneRemote(c *check.C) {
	s.test(c, listTrial{
		count:       "none",
		limit:       -1,
		filters:     []arvados.Filter{{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][0]}}},
		expectUUIDs: []string{s.uuids[0][0], s.uuids[1][0]},
		expectCalls: []int{1, 1, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListOneLocalOneRemoteDeselectingUUID(c *check.C) {
	s.test(c, listTrial{
		count:        "none",
		limit:        -1,
		filters:      []arvados.Filter{{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][0]}}},
		selectfields: []string{"name"},
		expectUUIDs:  []string{s.uuids[0][0], s.uuids[1][0]}, // uuid is returned, despite not being selected
		expectCalls:  []int{1, 1, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListTwoRemotes(c *check.C) {
	s.test(c, listTrial{
		count:       "none",
		limit:       -1,
		filters:     []arvados.Filter{{"uuid", "in", []string{s.uuids[2][0], s.uuids[1][0]}}},
		expectUUIDs: []string{s.uuids[1][0], s.uuids[2][0]},
		expectCalls: []int{0, 1, 1},
	})
}

func (s *CollectionListSuite) TestCollectionListSatisfyAllFilters(c *check.C) {
	s.cluster.API.MaxItemsPerResponse = 2
	s.test(c, listTrial{
		count: "none",
		limit: -1,
		filters: []arvados.Filter{
			{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][1], s.uuids[2][0], s.uuids[2][1], s.uuids[2][2]}},
			{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][2], s.uuids[2][1]}},
		},
		expectUUIDs: []string{s.uuids[0][0], s.uuids[2][1]},
		expectCalls: []int{1, 0, 1},
	})
}

func (s *CollectionListSuite) TestCollectionListEmptySet(c *check.C) {
	s.test(c, listTrial{
		count:       "none",
		limit:       -1,
		filters:     []arvados.Filter{{"uuid", "in", []string{}}},
		expectUUIDs: []string{},
		expectCalls: []int{0, 0, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListUnmatchableUUID(c *check.C) {
	s.test(c, listTrial{
		count: "none",
		limit: -1,
		filters: []arvados.Filter{
			{"uuid", "in", []string{s.uuids[0][0], "abcdefg"}},
			{"uuid", "in", []string{s.uuids[0][0], "bbbbb-4zz18-bogus"}},
			{"uuid", "in", []string{s.uuids[0][0], "bogus-4zz18-bogus"}},
		},
		expectUUIDs: []string{s.uuids[0][0]},
		expectCalls: []int{1, 0, 0},
	})
}

func (s *CollectionListSuite) TestCollectionListMultiPage(c *check.C) {
	for i := range s.backends {
		s.uuids[i] = s.uuids[i][:3]
		s.backends[i].ItemsToReturn = s.backends[i].ItemsToReturn[:3]
	}
	s.cluster.API.MaxItemsPerResponse = 9
	for _, stub := range s.backends {
		stub.MaxPageSize = 2
	}
	allUUIDs := append(append(append([]string(nil), s.uuids[0]...), s.uuids[1]...), s.uuids[2]...)
	s.test(c, listTrial{
		count:       "none",
		limit:       -1,
		filters:     []arvados.Filter{{"uuid", "in", append([]string(nil), allUUIDs...)}},
		expectUUIDs: allUUIDs,
		expectCalls: []int{2, 2, 2},
	})
}

func (s *CollectionListSuite) TestCollectionListMultiSiteExtraFilters(c *check.C) {
	// not [yet] supported
	s.test(c, listTrial{
		count: "none",
		limit: -1,
		filters: []arvados.Filter{
			{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][0]}},
			{"uuid", "is_a", "teapot"},
		},
		expectCalls:  []int{0, 0, 0},
		expectStatus: http.StatusBadRequest,
	})
}

func (s *CollectionListSuite) TestCollectionListMultiSiteWithCount(c *check.C) {
	for _, count := range []string{"", "exact"} {
		s.SetUpTest(c) // Reset backends / call counters
		s.test(c, listTrial{
			count: count,
			limit: -1,
			filters: []arvados.Filter{
				{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][0]}},
				{"uuid", "is_a", "teapot"},
			},
			expectCalls:  []int{0, 0, 0},
			expectStatus: http.StatusBadRequest,
		})
	}
}

func (s *CollectionListSuite) TestCollectionListMultiSiteWithLimit(c *check.C) {
	for _, limit := range []int64{0, 1, 2} {
		s.SetUpTest(c) // Reset backends / call counters
		s.test(c, listTrial{
			count: "none",
			limit: limit,
			filters: []arvados.Filter{
				{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][0], s.uuids[2][0]}},
				{"uuid", "is_a", "teapot"},
			},
			expectCalls:  []int{0, 0, 0},
			expectStatus: http.StatusBadRequest,
		})
	}
}

func (s *CollectionListSuite) TestCollectionListMultiSiteWithHighLimit(c *check.C) {
	uuids := []string{s.uuids[0][0], s.uuids[1][0], s.uuids[2][0]}
	for _, limit := range []int64{3, 4, 1234567890} {
		s.SetUpTest(c) // Reset backends / call counters
		s.test(c, listTrial{
			count: "none",
			limit: limit,
			filters: []arvados.Filter{
				{"uuid", "in", uuids},
			},
			expectUUIDs: uuids,
			expectCalls: []int{1, 1, 1},
		})
	}
}

func (s *CollectionListSuite) TestCollectionListMultiSiteWithOffset(c *check.C) {
	s.test(c, listTrial{
		count:  "none",
		limit:  -1,
		offset: 1,
		filters: []arvados.Filter{
			{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][0]}},
			{"uuid", "is_a", "teapot"},
		},
		expectCalls:  []int{0, 0, 0},
		expectStatus: http.StatusBadRequest,
	})
}

func (s *CollectionListSuite) TestCollectionListMultiSiteWithOrder(c *check.C) {
	s.test(c, listTrial{
		count: "none",
		limit: -1,
		order: []string{"uuid desc"},
		filters: []arvados.Filter{
			{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][0]}},
			{"uuid", "is_a", "teapot"},
		},
		expectCalls:  []int{0, 0, 0},
		expectStatus: http.StatusBadRequest,
	})
}

func (s *CollectionListSuite) TestCollectionListInvalidFilters(c *check.C) {
	s.test(c, listTrial{
		count: "none",
		limit: -1,
		filters: []arvados.Filter{
			{"uuid", "in", "teapot"},
		},
		expectCalls:  []int{0, 0, 0},
		expectStatus: http.StatusBadRequest,
	})
}

func (s *CollectionListSuite) TestCollectionListRemoteUnknown(c *check.C) {
	s.test(c, listTrial{
		count: "none",
		limit: -1,
		filters: []arvados.Filter{
			{"uuid", "in", []string{s.uuids[0][0], "bogus-4zz18-000001111122222"}},
		},
		expectStatus: http.StatusNotFound,
	})
}

func (s *CollectionListSuite) TestCollectionListRemoteError(c *check.C) {
	s.addDirectRemote(c, "bbbbb", &arvadostest.APIStub{Error: fmt.Errorf("stub backend error")})
	s.test(c, listTrial{
		count: "none",
		limit: -1,
		filters: []arvados.Filter{
			{"uuid", "in", []string{s.uuids[0][0], s.uuids[1][0]}},
		},
		expectStatus: http.StatusBadGateway,
	})
}

func (s *CollectionListSuite) test(c *check.C, trial listTrial) {
	resp, err := s.fed.CollectionList(s.ctx, arvados.ListOptions{
		Count:   trial.count,
		Limit:   trial.limit,
		Offset:  trial.offset,
		Order:   trial.order,
		Filters: trial.filters,
		Select:  trial.selectfields,
	})
	if trial.expectStatus != 0 {
		c.Assert(err, check.NotNil)
		err, ok := err.(interface{ HTTPStatus() int })
		c.Assert(ok, check.Equals, true) // err must implement interface{ HTTPStatus() int }
		c.Check(err.HTTPStatus(), check.Equals, trial.expectStatus)
		c.Logf("returned error is %#v", err)
		c.Logf("returned error string is %q", err)
	} else {
		c.Check(err, check.IsNil)
		expectItems := []arvados.Collection{}
		for _, uuid := range trial.expectUUIDs {
			expectItems = append(expectItems, arvados.Collection{UUID: uuid})
		}
		// expectItems is sorted by UUID, so sort resp.Items
		// by UUID before checking DeepEquals.
		sort.Slice(resp.Items, func(i, j int) bool { return resp.Items[i].UUID < resp.Items[j].UUID })
		c.Check(resp, check.DeepEquals, arvados.CollectionList{
			Items: expectItems,
		})
	}

	for i, stub := range s.backends {
		if i >= len(trial.expectCalls) {
			break
		}
		calls := stub.Calls(nil)
		c.Check(calls, check.HasLen, trial.expectCalls[i])
		if len(calls) == 0 {
			continue
		}
		opts := calls[0].Options.(arvados.ListOptions)
		c.Check(opts.Limit, check.Equals, trial.limit)
	}
}
