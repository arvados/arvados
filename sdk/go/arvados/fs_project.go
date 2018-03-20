// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"sync"
	"time"
)

type staleChecker struct {
	mtx  sync.Mutex
	last time.Time
}

func (sc *staleChecker) DoIfStale(fn func(), staleFunc func(time.Time) bool) {
	sc.mtx.Lock()
	defer sc.mtx.Unlock()
	if !staleFunc(sc.last) {
		return
	}
	sc.last = time.Now()
	fn()
}

// projectnode exposes an Arvados project as a filesystem directory.
type projectnode struct {
	inode
	staleChecker
	uuid string
	err  error
}

func (pn *projectnode) load() {
	fs := pn.FS().(*customFileSystem)

	if pn.uuid == "" {
		var resp User
		pn.err = fs.RequestAndDecode(&resp, "GET", "arvados/v1/users/current", nil, nil)
		if pn.err != nil {
			return
		}
		pn.uuid = resp.UUID
	}
	filters := []Filter{{"owner_uuid", "=", pn.uuid}}
	params := ResourceListParams{
		Filters: filters,
		Order:   "uuid",
	}
	for {
		var resp CollectionList
		pn.err = fs.RequestAndDecode(&resp, "GET", "arvados/v1/collections", nil, params)
		if pn.err != nil {
			return
		}
		if len(resp.Items) == 0 {
			break
		}
		for _, i := range resp.Items {
			coll := i
			if coll.Name == "" {
				continue
			}
			pn.inode.Child(coll.Name, func(inode) (inode, error) {
				return deferredCollectionFS(fs, pn, coll), nil
			})
		}
		params.Filters = append(filters, Filter{"uuid", ">", resp.Items[len(resp.Items)-1].UUID})
	}

	filters = append(filters, Filter{"group_class", "=", "project"})
	params.Filters = filters
	for {
		var resp GroupList
		pn.err = fs.RequestAndDecode(&resp, "GET", "arvados/v1/groups", nil, params)
		if pn.err != nil {
			return
		}
		if len(resp.Items) == 0 {
			break
		}
		for _, group := range resp.Items {
			if group.Name == "" || group.Name == "." || group.Name == ".." {
				continue
			}
			pn.inode.Child(group.Name, func(inode) (inode, error) {
				return fs.newProjectNode(pn, group.Name, group.UUID), nil
			})
		}
		params.Filters = append(filters, Filter{"uuid", ">", resp.Items[len(resp.Items)-1].UUID})
	}
	pn.err = nil
}

func (pn *projectnode) Readdir() ([]os.FileInfo, error) {
	pn.staleChecker.DoIfStale(pn.load, pn.FS().(*customFileSystem).Stale)
	if pn.err != nil {
		return nil, pn.err
	}
	return pn.inode.Readdir()
}

func (pn *projectnode) Child(name string, replace func(inode) (inode, error)) (inode, error) {
	pn.staleChecker.DoIfStale(pn.load, pn.FS().(*customFileSystem).Stale)
	if pn.err != nil {
		return nil, pn.err
	}
	if replace == nil {
		// lookup
		return pn.inode.Child(name, nil)
	}
	return pn.inode.Child(name, func(existing inode) (inode, error) {
		if repl, err := replace(existing); err != nil {
			return existing, err
		} else if repl == nil {
			if existing == nil {
				return nil, nil
			}
			// rmdir
			// (TODO)
			return existing, ErrInvalidArgument
		} else if existing != nil {
			// clobber
			return existing, ErrInvalidArgument
		} else if repl.FileInfo().IsDir() {
			// mkdir
			// TODO: repl.SetParent(pn, name), etc.
			return existing, ErrInvalidArgument
		} else {
			// create file
			return existing, ErrInvalidArgument
		}
	})
}
