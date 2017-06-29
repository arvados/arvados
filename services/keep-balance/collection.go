// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

func countCollections(c *arvados.Client, params arvados.ResourceListParams) (int, error) {
	var page arvados.CollectionList
	var zero int
	params.Limit = &zero
	params.Count = "exact"
	err := c.RequestAndDecode(&page, "GET", "arvados/v1/collections", nil, params)
	return page.ItemsAvailable, err
}

// EachCollection calls f once for every readable
// collection. EachCollection stops if it encounters an error, such as
// f returning a non-nil error.
//
// The progress function is called periodically with done (number of
// times f has been called) and total (number of times f is expected
// to be called).
//
// If pageSize > 0 it is used as the maximum page size in each API
// call; otherwise the maximum allowed page size is requested.
func EachCollection(c *arvados.Client, pageSize int, f func(arvados.Collection) error, progress func(done, total int)) error {
	if progress == nil {
		progress = func(_, _ int) {}
	}

	expectCount, err := countCollections(c, arvados.ResourceListParams{
		IncludeTrash: true,
	})
	if err != nil {
		return err
	}

	limit := pageSize
	if limit <= 0 {
		// Use the maximum page size the server allows
		limit = 1<<31 - 1
	}
	params := arvados.ResourceListParams{
		Limit:        &limit,
		Order:        "modified_at, uuid",
		Count:        "none",
		Select:       []string{"uuid", "unsigned_manifest_text", "modified_at", "portable_data_hash", "replication_desired"},
		IncludeTrash: true,
	}
	var last arvados.Collection
	var filterTime time.Time
	callCount := 0
	gettingExactTimestamp := false
	for {
		progress(callCount, expectCount)
		var page arvados.CollectionList
		err := c.RequestAndDecode(&page, "GET", "arvados/v1/collections", nil, params)
		if err != nil {
			return err
		}
		for _, coll := range page.Items {
			if last.ModifiedAt != nil && *last.ModifiedAt == *coll.ModifiedAt && last.UUID >= coll.UUID {
				continue
			}
			callCount++
			err = f(coll)
			if err != nil {
				return err
			}
			last = coll
		}
		if len(page.Items) == 0 && !gettingExactTimestamp {
			break
		} else if last.ModifiedAt == nil {
			return fmt.Errorf("BUG: Last collection on the page (%s) has no modified_at timestamp; cannot make progress", last.UUID)
		} else if len(page.Items) > 0 && *last.ModifiedAt == filterTime {
			// If we requested time>=X and never got a
			// time>X then we might not have received all
			// items with time==X yet. Switch to
			// gettingExactTimestamp mode (if we're not
			// there already), advancing our UUID
			// threshold with each request, until we get
			// an empty page.
			gettingExactTimestamp = true
			params.Filters = []arvados.Filter{{
				Attr:     "modified_at",
				Operator: "=",
				Operand:  filterTime,
			}, {
				Attr:     "uuid",
				Operator: ">",
				Operand:  last.UUID,
			}}
		} else if gettingExactTimestamp {
			// This must be an empty page (in this mode,
			// an unequal timestamp is impossible) so we
			// can start getting pages of newer
			// collections.
			gettingExactTimestamp = false
			params.Filters = []arvados.Filter{{
				Attr:     "modified_at",
				Operator: ">",
				Operand:  filterTime,
			}}
		} else {
			// In the normal case, we know we have seen
			// all collections with modtime<filterTime,
			// but we might not have seen all that have
			// modtime=filterTime. Hence we use >= instead
			// of > and skip the obvious overlapping item,
			// i.e., the last item on the previous
			// page. In some edge cases this can return
			// collections we have already seen, but
			// avoiding that would add overhead in the
			// overwhelmingly common cases, so we don't
			// bother.
			filterTime = *last.ModifiedAt
			params.Filters = []arvados.Filter{{
				Attr:     "modified_at",
				Operator: ">=",
				Operand:  filterTime,
			}, {
				Attr:     "uuid",
				Operator: "!=",
				Operand:  last.UUID,
			}}
		}
	}
	progress(callCount, expectCount)

	if checkCount, err := countCollections(c, arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "modified_at",
			Operator: "<=",
			Operand:  filterTime}},
		IncludeTrash: true,
	}); err != nil {
		return err
	} else if callCount < checkCount {
		return fmt.Errorf("Retrieved %d collections with modtime <= T=%q, but server now reports there are %d collections with modtime <= T", callCount, filterTime, checkCount)
	}

	return nil
}
