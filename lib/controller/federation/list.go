// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

//go:generate go run generate.go

// CollectionList is used as a template to auto-generate List()
// methods for other types; see generate.go.

func (conn *Conn) CollectionList(ctx context.Context, options arvados.ListOptions) (arvados.CollectionList, error) {
	var mtx sync.Mutex
	var merged arvados.CollectionList
	err := conn.splitListRequest(ctx, options, func(ctx context.Context, _ string, backend arvados.API, options arvados.ListOptions) ([]string, error) {
		cl, err := backend.CollectionList(ctx, options)
		if err != nil {
			return nil, err
		}
		mtx.Lock()
		defer mtx.Unlock()
		if len(merged.Items) == 0 {
			merged = cl
		} else {
			merged.Items = append(merged.Items, cl.Items...)
		}
		uuids := make([]string, 0, len(cl.Items))
		for _, item := range cl.Items {
			uuids = append(uuids, item.UUID)
		}
		return uuids, nil
	})
	sort.Slice(merged.Items, func(i, j int) bool { return merged.Items[i].UUID < merged.Items[j].UUID })
	return merged, err
}

// Call fn on one or more local/remote backends if opts indicates a
// federation-wide list query, i.e.:
//
// * There is at least one filter of the form
//   ["uuid","in",[a,b,c,...]] or ["uuid","=",a]
//
// * One or more of the supplied UUIDs (a,b,c,...) has a non-local
//   prefix.
//
// * There are no other filters
//
// (If opts doesn't indicate a federation-wide list query, fn is just
// called once with the local backend.)
//
// fn is called more than once only if the query meets the following
// restrictions:
//
// * Count=="none"
//
// * Limit<0
//
// * len(Order)==0
//
// * Each filter must be either "uuid = ..." or "uuid in [...]".
//
// * The maximum possible response size (total number of objects that
//   could potentially be matched by all of the specified filters)
//   exceeds the local cluster's response page size limit.
//
// If the query involves multiple backends but doesn't meet these
// restrictions, an error is returned without calling fn.
//
// Thus, the caller can assume that either:
//
// * splitListRequest() returns an error, or
//
// * fn is called exactly once, or
//
// * fn is called more than once, with options that satisfy the above
//   restrictions.
//
// Each call to fn indicates a single (local or remote) backend and a
// corresponding options argument suitable for sending to that
// backend.
func (conn *Conn) splitListRequest(ctx context.Context, opts arvados.ListOptions, fn func(context.Context, string, arvados.API, arvados.ListOptions) ([]string, error)) error {
	cannotSplit := false
	var matchAllFilters map[string]bool
	for _, f := range opts.Filters {
		matchThisFilter := map[string]bool{}
		if f.Attr != "uuid" {
			cannotSplit = true
			continue
		}
		if f.Operator == "=" {
			if uuid, ok := f.Operand.(string); ok {
				matchThisFilter[uuid] = true
			} else {
				return httpErrorf(http.StatusBadRequest, "invalid operand type %T for filter %q", f.Operand, f)
			}
		} else if f.Operator == "in" {
			if operand, ok := f.Operand.([]interface{}); ok {
				// skip any elements that aren't
				// strings (thus can't match a UUID,
				// thus can't affect the response).
				for _, v := range operand {
					if uuid, ok := v.(string); ok {
						matchThisFilter[uuid] = true
					}
				}
			} else if strings, ok := f.Operand.([]string); ok {
				for _, uuid := range strings {
					matchThisFilter[uuid] = true
				}
			} else {
				return httpErrorf(http.StatusBadRequest, "invalid operand type %T in filter %q", f.Operand, f)
			}
		} else {
			cannotSplit = true
			continue
		}

		if matchAllFilters == nil {
			matchAllFilters = matchThisFilter
		} else {
			// Reduce matchAllFilters to the intersection
			// of matchAllFilters ∩ matchThisFilter.
			for uuid := range matchAllFilters {
				if !matchThisFilter[uuid] {
					delete(matchAllFilters, uuid)
				}
			}
		}
	}

	if matchAllFilters == nil {
		// Not filtering by UUID at all; just query the local
		// cluster.
		_, err := fn(ctx, conn.cluster.ClusterID, conn.local, opts)
		return err
	}

	// Collate UUIDs in matchAllFilters by remote cluster ID --
	// e.g., todoByRemote["aaaaa"]["aaaaa-4zz18-000000000000000"]
	// will be true -- and count the total number of UUIDs we're
	// filtering on, so we can compare it to our max page size
	// limit.
	nUUIDs := 0
	todoByRemote := map[string]map[string]bool{}
	for uuid := range matchAllFilters {
		if len(uuid) != 27 {
			// Cannot match anything, just drop it
		} else {
			if todoByRemote[uuid[:5]] == nil {
				todoByRemote[uuid[:5]] = map[string]bool{}
			}
			todoByRemote[uuid[:5]][uuid] = true
			nUUIDs++
		}
	}

	if len(todoByRemote) > 1 {
		if cannotSplit {
			return httpErrorf(http.StatusBadRequest, "cannot execute federated list query: each filter must be either 'uuid = ...' or 'uuid in [...]'")
		}
		if opts.Count != "none" {
			return httpErrorf(http.StatusBadRequest, "cannot execute federated list query unless count==\"none\"")
		}
		if opts.Limit >= 0 || opts.Offset != 0 || len(opts.Order) > 0 {
			return httpErrorf(http.StatusBadRequest, "cannot execute federated list query with limit, offset, or order parameter")
		}
		if max := conn.cluster.API.MaxItemsPerResponse; nUUIDs > max {
			return httpErrorf(http.StatusBadRequest, "cannot execute federated list query because number of UUIDs (%d) exceeds page size limit %d", nUUIDs, max)
		}
		selectingUUID := false
		for _, attr := range opts.Select {
			if attr == "uuid" {
				selectingUUID = true
			}
		}
		if opts.Select != nil && !selectingUUID {
			return httpErrorf(http.StatusBadRequest, "cannot execute federated list query with a select parameter that does not include uuid")
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs := make(chan error, len(todoByRemote))
	for clusterID, todo := range todoByRemote {
		go func(clusterID string, todo map[string]bool) {
			// This goroutine sends exactly one value to
			// errs.
			batch := make([]string, 0, len(todo))
			for uuid := range todo {
				batch = append(batch, uuid)
			}

			var backend arvados.API
			if clusterID == conn.cluster.ClusterID {
				backend = conn.local
			} else if backend = conn.remotes[clusterID]; backend == nil {
				errs <- httpErrorf(http.StatusNotFound, "cannot execute federated list query: no proxy available for cluster %q", clusterID)
				return
			}
			remoteOpts := opts
			for len(todo) > 0 {
				if len(batch) > len(todo) {
					// Reduce batch to just the todo's
					batch = batch[:0]
					for uuid := range todo {
						batch = append(batch, uuid)
					}
				}
				remoteOpts.Filters = []arvados.Filter{{"uuid", "in", batch}}

				done, err := fn(ctx, clusterID, backend, remoteOpts)
				if err != nil {
					errs <- err
					return
				}
				progress := false
				for _, uuid := range done {
					if _, ok := todo[uuid]; ok {
						progress = true
						delete(todo, uuid)
					}
				}
				if !progress {
					errs <- httpErrorf(http.StatusBadGateway, "cannot make progress in federated list query: cluster %q returned none of the requested UUIDs", clusterID)
					return
				}
			}
			errs <- nil
		}(clusterID, todo)
	}

	// Wait for all goroutines to return, then return the first
	// non-nil error, if any.
	var firstErr error
	for range todoByRemote {
		if err := <-errs; err != nil && firstErr == nil {
			firstErr = err
			// Signal to any remaining fn() calls that
			// further effort is futile.
			cancel()
		}
	}
	return firstErr
}

func httpErrorf(code int, format string, args ...interface{}) error {
	return httpserver.ErrorWithStatus(fmt.Errorf(format, args...), code)
}
