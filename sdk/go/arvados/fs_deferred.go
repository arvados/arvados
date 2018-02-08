// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"log"
	"os"
	"sync"
	"time"
)

func deferredCollectionFS(fs FileSystem, parent inode, coll Collection) inode {
	var modTime time.Time
	if coll.ModifiedAt != nil {
		modTime = *coll.ModifiedAt
	} else {
		modTime = time.Now()
	}
	placeholder := &treenode{
		fs:     fs,
		parent: parent,
		inodes: nil,
		fileinfo: fileinfo{
			name:    coll.Name,
			modTime: modTime,
			mode:    0755 | os.ModeDir,
		},
	}
	return &deferrednode{wrapped: placeholder, create: func() inode {
		err := fs.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+coll.UUID, nil, nil)
		if err != nil {
			log.Printf("BUG: unhandled error: %s", err)
			return placeholder
		}
		cfs, err := coll.FileSystem(fs, fs)
		if err != nil {
			log.Printf("BUG: unhandled error: %s", err)
			return placeholder
		}
		root := cfs.rootnode()
		root.SetParent(parent, coll.Name)
		return root
	}}
}

// A deferrednode wraps an inode that's expensive to build. Initially,
// it responds to basic directory functions by proxying to the given
// placeholder. If a caller uses a read/write/lock operation,
// deferrednode calls the create() func to create the real inode, and
// proxies to the real inode from then on.
//
// In practice, this means a deferrednode's parent's directory listing
// can be generated using only the placeholder, instead of waiting for
// create().
type deferrednode struct {
	wrapped inode
	create  func() inode
	mtx     sync.Mutex
	created bool
}

func (dn *deferrednode) realinode() inode {
	dn.mtx.Lock()
	defer dn.mtx.Unlock()
	if !dn.created {
		dn.wrapped = dn.create()
		dn.created = true
	}
	return dn.wrapped
}

func (dn *deferrednode) currentinode() inode {
	dn.mtx.Lock()
	defer dn.mtx.Unlock()
	return dn.wrapped
}

func (dn *deferrednode) Read(p []byte, pos filenodePtr) (int, filenodePtr, error) {
	return dn.realinode().Read(p, pos)
}

func (dn *deferrednode) Write(p []byte, pos filenodePtr) (int, filenodePtr, error) {
	return dn.realinode().Write(p, pos)
}

func (dn *deferrednode) Child(name string, replace func(inode) (inode, error)) (inode, error) {
	return dn.realinode().Child(name, replace)
}

func (dn *deferrednode) Truncate(size int64) error       { return dn.realinode().Truncate(size) }
func (dn *deferrednode) SetParent(p inode, name string)  { dn.realinode().SetParent(p, name) }
func (dn *deferrednode) IsDir() bool                     { return dn.currentinode().IsDir() }
func (dn *deferrednode) Readdir() ([]os.FileInfo, error) { return dn.realinode().Readdir() }
func (dn *deferrednode) Size() int64                     { return dn.currentinode().Size() }
func (dn *deferrednode) FileInfo() os.FileInfo           { return dn.currentinode().FileInfo() }
func (dn *deferrednode) Lock()                           { dn.realinode().Lock() }
func (dn *deferrednode) Unlock()                         { dn.realinode().Unlock() }
func (dn *deferrednode) RLock()                          { dn.realinode().RLock() }
func (dn *deferrednode) RUnlock()                        { dn.realinode().RUnlock() }
func (dn *deferrednode) FS() FileSystem                  { return dn.currentinode().FS() }
func (dn *deferrednode) Parent() inode                   { return dn.currentinode().Parent() }
