// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
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
func EachCollection(ctx context.Context, db *sqlx.DB, c *arvados.Client, f func(arvados.Collection) error, progress func(done, total int)) error {
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
	var newestModifiedAt time.Time

	rows, err := db.QueryxContext(ctx, `SELECT
		uuid, manifest_text, modified_at, portable_data_hash,
		replication_desired, replication_confirmed, replication_confirmed_at,
		storage_classes_desired, storage_classes_confirmed, storage_classes_confirmed_at,
		is_trashed
		FROM collections`)
	if err != nil {
		return err
	}
	defer rows.Close()
	progressTicker := time.NewTicker(10 * time.Second)
	defer progressTicker.Stop()
	callCount := 0
	for rows.Next() {
		var coll arvados.Collection
		var classesDesired, classesConfirmed []byte
		err = rows.Scan(&coll.UUID, &coll.ManifestText, &coll.ModifiedAt, &coll.PortableDataHash,
			&coll.ReplicationDesired, &coll.ReplicationConfirmed, &coll.ReplicationConfirmedAt,
			&classesDesired, &classesConfirmed, &coll.StorageClassesConfirmedAt,
			&coll.IsTrashed)
		if err != nil {
			return err
		}

		err = json.Unmarshal(classesDesired, &coll.StorageClassesDesired)
		if err != nil && len(classesDesired) > 0 {
			return err
		}
		err = json.Unmarshal(classesConfirmed, &coll.StorageClassesConfirmed)
		if err != nil && len(classesConfirmed) > 0 {
			return err
		}
		if newestModifiedAt.IsZero() || newestModifiedAt.Before(coll.ModifiedAt) {
			newestModifiedAt = coll.ModifiedAt
		}
		callCount++
		err = f(coll)
		if err != nil {
			return err
		}
		select {
		case <-progressTicker.C:
			progress(callCount, expectCount)
		default:
		}
	}
	progress(callCount, expectCount)
	err = rows.Close()
	if err != nil {
		return err
	}
	if checkCount, err := countCollections(c, arvados.ResourceListParams{
		Filters: []arvados.Filter{{
			Attr:     "modified_at",
			Operator: "<=",
			Operand:  newestModifiedAt}},
		IncludeTrash:       true,
		IncludeOldVersions: true,
	}); err != nil {
		return err
	} else if callCount < checkCount {
		return fmt.Errorf("Retrieved %d collections with modtime <= T=%q, but server now reports there are %d collections with modtime <= T", callCount, newestModifiedAt, checkCount)
	}

	return nil
}

func (bal *Balancer) updateCollections(ctx context.Context, c *arvados.Client, cluster *arvados.Cluster) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer bal.time("update_collections", "wall clock time to update collections")()
	threshold := time.Now()
	thresholdStr := threshold.Format(time.RFC3339Nano)

	updated := int64(0)

	errs := make(chan error, 1)
	collQ := make(chan arvados.Collection, cluster.Collections.BalanceCollectionBuffers)
	go func() {
		defer close(collQ)
		err := EachCollection(ctx, bal.DB, c, func(coll arvados.Collection) error {
			if atomic.LoadInt64(&updated) >= int64(cluster.Collections.BalanceUpdateLimit) {
				bal.logf("reached BalanceUpdateLimit (%d)", cluster.Collections.BalanceUpdateLimit)
				cancel()
				return context.Canceled
			}
			collQ <- coll
			return nil
		}, func(done, total int) {
			bal.logf("update collections: %d/%d (%d updated @ %.01f updates/s)", done, total, atomic.LoadInt64(&updated), float64(atomic.LoadInt64(&updated))/time.Since(threshold).Seconds())
		})
		if err != nil && err != context.Canceled {
			select {
			case errs <- err:
			default:
			}
		}
	}()

	var wg sync.WaitGroup

	// Use about 1 goroutine per 2 CPUs. Based on experiments with
	// a 2-core host, using more concurrent database
	// calls/transactions makes this process slower, not faster.
	for i := 0; i < runtime.NumCPU()+1/2; i++ {
		wg.Add(1)
		goSendErr(errs, func() error {
			defer wg.Done()
			tx, err := bal.DB.Beginx()
			if err != nil {
				return err
			}
			txPending := 0
			flush := func(final bool) error {
				err := tx.Commit()
				if err != nil && ctx.Err() == nil {
					tx.Rollback()
					return err
				}
				txPending = 0
				if final {
					return nil
				}
				tx, err = bal.DB.Beginx()
				return err
			}
			txBatch := 100
			for coll := range collQ {
				if ctx.Err() != nil || len(errs) > 0 {
					continue
				}
				blkids, err := coll.SizedDigests()
				if err != nil {
					bal.logf("%s: %s", coll.UUID, err)
					continue
				}
				repl := bal.BlockStateMap.GetConfirmedReplication(blkids, coll.StorageClassesDesired)

				desired := bal.DefaultReplication
				if coll.ReplicationDesired != nil {
					desired = *coll.ReplicationDesired
				}
				if repl > desired {
					// If actual>desired, confirm
					// the desired number rather
					// than actual to avoid
					// flapping updates when
					// replication increases
					// temporarily.
					repl = desired
				}
				classes, err := json.Marshal(coll.StorageClassesDesired)
				if err != nil {
					bal.logf("BUG? json.Marshal(%v) failed: %s", classes, err)
					continue
				}
				needUpdate := coll.ReplicationConfirmed == nil || *coll.ReplicationConfirmed != repl || len(coll.StorageClassesConfirmed) != len(coll.StorageClassesDesired)
				for i := range coll.StorageClassesDesired {
					if !needUpdate && coll.StorageClassesDesired[i] != coll.StorageClassesConfirmed[i] {
						needUpdate = true
					}
				}
				if !needUpdate {
					continue
				}
				_, err = tx.ExecContext(ctx, `update collections set
					replication_confirmed=$1,
					replication_confirmed_at=$2,
					storage_classes_confirmed=$3,
					storage_classes_confirmed_at=$2
					where uuid=$4`,
					repl, thresholdStr, classes, coll.UUID)
				if err != nil {
					if ctx.Err() == nil {
						bal.logf("%s: update failed: %s", coll.UUID, err)
					}
					continue
				}
				atomic.AddInt64(&updated, 1)
				if txPending++; txPending >= txBatch {
					err = flush(false)
					if err != nil {
						return err
					}
				}
			}
			return flush(true)
		})
	}
	wg.Wait()
	bal.logf("updated %d collections", updated)
	if len(errs) > 0 {
		return fmt.Errorf("error updating collections: %s", <-errs)
	}
	return nil
}

// Call f in a new goroutine. If it returns a non-nil error, send the
// error to the errs channel (unless the channel is already full with
// another error).
func goSendErr(errs chan<- error, f func() error) {
	go func() {
		err := f()
		if err != nil {
			select {
			case errs <- err:
			default:
			}
		}
	}()
}
