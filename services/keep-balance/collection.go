// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/jmoiron/sqlx"
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
func EachCollection(ctx context.Context, c *arvados.Client, pageSize int, f func(arvados.Collection) error, progress func(done, total int)) error {
	if progress == nil {
		progress = func(_, _ int) {}
	}

	expectCount, err := countCollections(c, arvados.ResourceListParams{
		IncludeTrash:       true,
		IncludeOldVersions: true,
	})
	if err != nil {
		return err
	}

	// Note the obvious way to get all collections (sorting by
	// UUID) would be much easier, but would lose data: If a
	// client were to move files from collection with uuid="zzz"
	// to a collection with uuid="aaa" around the time when we
	// were fetching the "mmm" page, we would never see those
	// files' block IDs at all -- even if the client is careful to
	// save "aaa" before saving "zzz".
	//
	// Instead, we get pages in modified_at order. Collections
	// that are modified during the run will be re-fetched in a
	// subsequent page.

	limit := pageSize
	if limit <= 0 {
		// Use the maximum page size the server allows
		limit = 1<<31 - 1
	}
	params := arvados.ResourceListParams{
		Limit:              &limit,
		Order:              "modified_at, uuid",
		Count:              "none",
		Select:             []string{"uuid", "unsigned_manifest_text", "modified_at", "portable_data_hash", "replication_desired", "storage_classes_desired", "is_trashed"},
		IncludeTrash:       true,
		IncludeOldVersions: true,
	}
	var last arvados.Collection
	var filterTime time.Time
	callCount := 0
	gettingExactTimestamp := false
	for {
		progress(callCount, expectCount)
		var page arvados.CollectionList
		err := c.RequestAndDecodeContext(ctx, &page, "GET", "arvados/v1/collections", nil, params)
		if err != nil {
			return err
		}
		for _, coll := range page.Items {
			if last.ModifiedAt == coll.ModifiedAt && last.UUID >= coll.UUID {
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
		} else if last.ModifiedAt.IsZero() {
			return fmt.Errorf("BUG: Last collection on the page (%s) has no modified_at timestamp; cannot make progress", last.UUID)
		} else if len(page.Items) > 0 && last.ModifiedAt == filterTime {
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
			filterTime = last.ModifiedAt
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
		IncludeTrash:       true,
		IncludeOldVersions: true,
	}); err != nil {
		return err
	} else if callCount < checkCount {
		return fmt.Errorf("Retrieved %d collections with modtime <= T=%q, but server now reports there are %d collections with modtime <= T", callCount, filterTime, checkCount)
	}

	return nil
}

func (bal *Balancer) updateCollections(ctx context.Context, c *arvados.Client, cluster *arvados.Cluster) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer bal.time("update_collections", "wall clock time to update collections")()
	threshold := time.Now()
	thresholdStr := threshold.Format(time.RFC3339Nano)

	var err error
	collQ := make(chan arvados.Collection, cluster.Collections.BalanceCollectionBuffers)
	go func() {
		defer close(collQ)
		err = EachCollection(ctx, c, cluster.Collections.BalanceCollectionBatch, func(coll arvados.Collection) error {
			if coll.ModifiedAt.After(threshold) {
				return io.EOF
			}
			if coll.IsTrashed {
				return nil
			}
			collQ <- coll
			return nil
		}, func(done, total int) {
			bal.logf("update collections: %d/%d", done, total)
		})
		if err == io.EOF {
			err = nil
		} else if err != nil {
			bal.logf("error updating collections: %s", err)
		}
	}()

	db, err := bal.db(cluster)
	if err != nil {
		return err
	}

	var updated int64
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for coll := range collQ {
				blkids, err := coll.SizedDigests()
				if err != nil {
					bal.logf("%s: %s", coll.UUID, err)
					continue
				}
				repl := bal.BlockStateMap.GetConfirmedReplication(blkids, coll.StorageClassesDesired)
				tx, err := db.Beginx()
				if err != nil {
					bal.logf("error opening transaction: %s", coll.UUID, err)
					cancel()
					continue
				}
				classes, _ := json.Marshal(coll.StorageClassesDesired)
				_, err = tx.ExecContext(ctx, `update collections set
					replication_confirmed=$1,
					replication_confirmed_at=$2,
					storage_classes_confirmed=$3,
					storage_classes_confirmed_at=$2
					where uuid=$4`,
					repl, thresholdStr, classes, coll.UUID)
				if err != nil {
					tx.Rollback()
				} else {
					err = tx.Commit()
				}
				if err != nil {
					bal.logf("%s: update failed: %s", coll.UUID, err)
					continue
				}
				atomic.AddInt64(&updated, 1)
			}
		}()
	}
	wg.Wait()
	bal.logf("updated %d collections", updated)
	return err
}

func (bal *Balancer) db(cluster *arvados.Cluster) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", cluster.PostgreSQL.Connection.String())
	if err != nil {
		return nil, err
	}
	if p := cluster.PostgreSQL.ConnectionPool; p > 0 {
		db.SetMaxOpenConns(p)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("postgresql connect succeeded but ping failed: %s", err)
	}
	return db, nil
}
