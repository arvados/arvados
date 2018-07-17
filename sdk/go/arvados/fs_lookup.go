// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"sync"
	"time"
)

// lookupnode is a caching tree node that is initially empty and calls
// loadOne and loadAll to load/update child nodes as needed.
//
// See (*customFileSystem)MountUsers for example usage.
type lookupnode struct {
	inode
	loadOne func(parent inode, name string) (inode, error)
	loadAll func(parent inode) ([]inode, error)
	stale   func(time.Time) bool

	// internal fields
	staleLock sync.Mutex
	staleAll  time.Time
	staleOne  map[string]time.Time
}

func (ln *lookupnode) Readdir() ([]os.FileInfo, error) {
	ln.staleLock.Lock()
	defer ln.staleLock.Unlock()
	checkTime := time.Now()
	if ln.stale(ln.staleAll) {
		all, err := ln.loadAll(ln)
		if err != nil {
			return nil, err
		}
		for _, child := range all {
			_, err = ln.inode.Child(child.FileInfo().Name(), func(inode) (inode, error) {
				return child, nil
			})
			if err != nil {
				return nil, err
			}
		}
		ln.staleAll = checkTime
		// No value in ln.staleOne can make a difference to an
		// "entry is stale?" test now, because no value is
		// newer than ln.staleAll. Reclaim memory.
		ln.staleOne = nil
	}
	return ln.inode.Readdir()
}

func (ln *lookupnode) Child(name string, replace func(inode) (inode, error)) (inode, error) {
	ln.staleLock.Lock()
	defer ln.staleLock.Unlock()
	checkTime := time.Now()
	if ln.stale(ln.staleAll) && ln.stale(ln.staleOne[name]) {
		_, err := ln.inode.Child(name, func(inode) (inode, error) {
			return ln.loadOne(ln, name)
		})
		if err != nil {
			return nil, err
		}
		if ln.staleOne == nil {
			ln.staleOne = map[string]time.Time{name: checkTime}
		} else {
			ln.staleOne[name] = checkTime
		}
	}
	return ln.inode.Child(name, replace)
}
