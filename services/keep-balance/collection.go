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

	expectCount, err := countCollections(c, arvados.ResourceListParams{})
	if err != nil {
		return err
	}

	limit := pageSize
	if limit <= 0 {
		// Use the maximum page size the server allows
		limit = 1<<31 - 1
	}
	params := arvados.ResourceListParams{
		Limit:  &limit,
		Order:  "modified_at, uuid",
		Select: []string{"uuid", "unsigned_manifest_text", "modified_at", "portable_data_hash", "replication_desired"},
	}
	var last arvados.Collection
	var filterTime time.Time
	callCount := 0
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
		if last.ModifiedAt == nil || *last.ModifiedAt == filterTime {
			if page.ItemsAvailable > len(page.Items) {
				// TODO: use "mtime=X && UUID>Y"
				// filters to get all collections with
				// this timestamp, then use "mtime>X"
				// to get the next timestamp.
				return fmt.Errorf("BUG: Received an entire page with the same modified_at timestamp (%v), cannot make progress", filterTime)
			}
			break
		}
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
	progress(callCount, expectCount)

	if checkCount, err := countCollections(c, arvados.ResourceListParams{Filters: []arvados.Filter{{
		Attr:     "modified_at",
		Operator: "<=",
		Operand:  filterTime}}}); err != nil {
		return err
	} else if callCount < checkCount {
		return fmt.Errorf("Retrieved %d collections with modtime <= T=%q, but server now reports there are %d collections with modtime <= T", callCount, filterTime, checkCount)
	}

	return nil
}
