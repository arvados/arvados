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

	rows, err := db.QueryxContext(ctx, `SELECT uuid, manifest_text, modified_at, portable_data_hash, replication_desired, storage_classes_desired, is_trashed FROM collections`)
	if err != nil {
		return err
	}
	callCount := 0
	for rows.Next() {
		var coll arvados.Collection
		var classesDesired []byte
		err = rows.Scan(&coll.UUID, &coll.ManifestText, &coll.ModifiedAt, &coll.PortableDataHash, &coll.ReplicationDesired, &classesDesired, &coll.IsTrashed)
		if err != nil {
			rows.Close()
			return err
		}
		err = json.Unmarshal(classesDesired, &coll.StorageClassesDesired)
		if err != nil {
			rows.Close()
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
		progress(callCount, expectCount)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
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

	var err error
	collQ := make(chan arvados.Collection, cluster.Collections.BalanceCollectionBuffers)
	go func() {
		defer close(collQ)
		err = EachCollection(ctx, bal.DB, c, func(coll arvados.Collection) error {
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
				classes, err := json.Marshal(coll.StorageClassesDesired)
				if err != nil {
					bal.logf("BUG? json.Marshal(%v) failed: %s", classes, err)
					continue
				}
				_, err = bal.DB.ExecContext(ctx, `update collections set
					replication_confirmed=$1,
					replication_confirmed_at=$2,
					storage_classes_confirmed=$3,
					storage_classes_confirmed_at=$2
					where uuid=$4`,
					repl, thresholdStr, classes, coll.UUID)
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
