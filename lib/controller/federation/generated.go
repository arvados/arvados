// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package federation

import (
	"context"
	"sort"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

//
// -- this file is auto-generated -- do not edit -- edit list.go and run "go generate" instead --
//

func (conn *Conn) ContainerList(ctx context.Context, options arvados.ListOptions) (arvados.ContainerList, error) {
	var mtx sync.Mutex
	var merged arvados.ContainerList
	err := conn.splitListRequest(ctx, options, func(ctx context.Context, _ string, backend arvados.API, options arvados.ListOptions) ([]string, error) {
		cl, err := backend.ContainerList(ctx, options)
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

func (conn *Conn) SpecimenList(ctx context.Context, options arvados.ListOptions) (arvados.SpecimenList, error) {
	var mtx sync.Mutex
	var merged arvados.SpecimenList
	err := conn.splitListRequest(ctx, options, func(ctx context.Context, _ string, backend arvados.API, options arvados.ListOptions) ([]string, error) {
		cl, err := backend.SpecimenList(ctx, options)
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
