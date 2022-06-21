// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"time"
)

// lookupnode is a caching tree node that is initially empty and calls
// loadOne and loadAll to load/update child nodes as needed.
//
// See (*customFileSystem)MountUsers for example usage.
type lookupnode struct {
	treenode
	loadOne func(parent inode, name string) (inode, error)
	loadAll func(parent inode) ([]inode, error)
	stale   func(time.Time) bool

	// internal fields
	staleAll time.Time
	staleOne map[string]time.Time
}

// Sync flushes pending writes for loaded children and, if successful,
// triggers a reload on next lookup.
func (ln *lookupnode) Sync() error {
	err := ln.treenode.Sync()
	if err != nil {
		return err
	}
	ln.Lock()
	ln.staleAll = time.Time{}
	ln.staleOne = nil
	ln.Unlock()
	return nil
}

func (ln *lookupnode) Readdir() ([]os.FileInfo, error) {
	ln.Lock()
	checkTime := time.Now()
	if ln.stale(ln.staleAll) {
		all, err := ln.loadAll(ln)
		if err != nil {
			ln.Unlock()
			return nil, err
		}
		for _, child := range all {
			_, err = ln.treenode.Child(child.FileInfo().Name(), func(inode) (inode, error) {
				return child, nil
			})
			if err != nil {
				ln.Unlock()
				return nil, err
			}
		}
		ln.staleAll = checkTime
		// No value in ln.staleOne can make a difference to an
		// "entry is stale?" test now, because no value is
		// newer than ln.staleAll. Reclaim memory.
		ln.staleOne = nil
	}
	ln.Unlock()
	return ln.treenode.Readdir()
}

// Child rejects (with ErrInvalidOperation) calls to add/replace
// children, instead calling loadOne when a non-existing child is
// looked up.
func (ln *lookupnode) Child(name string, replace func(inode) (inode, error)) (inode, error) {
	checkTime := time.Now()
	var existing inode
	var err error
	if ln.stale(ln.staleAll) && ln.stale(ln.staleOne[name]) {
		existing, err = ln.treenode.Child(name, func(inode) (inode, error) {
			return ln.loadOne(ln, name)
		})
		if err == nil && existing != nil {
			if ln.staleOne == nil {
				ln.staleOne = map[string]time.Time{name: checkTime}
			} else {
				ln.staleOne[name] = checkTime
			}
		}
	} else {
		existing, err = ln.treenode.Child(name, nil)
		if err != nil && !os.IsNotExist(err) {
			return existing, err
		}
	}
	if replace != nil {
		// Let the callback try to delete or replace the
		// existing node; if it does, return
		// ErrInvalidOperation.
		if tryRepl, err := replace(existing); err != nil {
			// Propagate error from callback
			return existing, err
		} else if tryRepl != existing {
			return existing, ErrInvalidOperation
		}
	}
	// Return original error from ln.treenode.Child() (it might be
	// ErrNotExist).
	return existing, err
}
