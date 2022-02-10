// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"strings"
	"sync"
	"time"
)

type CustomFileSystem interface {
	FileSystem
	MountByID(mount string)
	MountProject(mount, uuid string)
	MountUsers(mount string)
	ForwardSlashNameSubstitution(string)
}

type customFileSystem struct {
	fileSystem
	root *vdirnode
	thr  *throttle

	staleThreshold time.Time
	staleLock      sync.Mutex

	forwardSlashNameSubstitution string
}

func (c *Client) CustomFileSystem(kc keepClient) CustomFileSystem {
	root := &vdirnode{}
	fs := &customFileSystem{
		root: root,
		fileSystem: fileSystem{
			fsBackend: keepBackend{apiClient: c, keepClient: kc},
			root:      root,
			thr:       newThrottle(concurrentWriters),
		},
	}
	root.treenode = treenode{
		fs:     fs,
		parent: root,
		fileinfo: fileinfo{
			name:    "/",
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
		},
		inodes: make(map[string]inode),
	}
	return fs
}

func (fs *customFileSystem) MountByID(mount string) {
	fs.root.treenode.Lock()
	defer fs.root.treenode.Unlock()
	fs.root.treenode.Child(mount, func(inode) (inode, error) {
		return &vdirnode{
			treenode: treenode{
				fs:     fs,
				parent: fs.root,
				inodes: make(map[string]inode),
				fileinfo: fileinfo{
					name:    mount,
					modTime: time.Now(),
					mode:    0755 | os.ModeDir,
				},
			},
			create: fs.mountByID,
		}, nil
	})
}

func (fs *customFileSystem) MountProject(mount, uuid string) {
	fs.root.treenode.Lock()
	defer fs.root.treenode.Unlock()
	fs.root.treenode.Child(mount, func(inode) (inode, error) {
		return fs.newProjectNode(fs.root, mount, uuid), nil
	})
}

func (fs *customFileSystem) MountUsers(mount string) {
	fs.root.treenode.Lock()
	defer fs.root.treenode.Unlock()
	fs.root.treenode.Child(mount, func(inode) (inode, error) {
		return &lookupnode{
			stale:   fs.Stale,
			loadOne: fs.usersLoadOne,
			loadAll: fs.usersLoadAll,
			treenode: treenode{
				fs:     fs,
				parent: fs.root,
				inodes: make(map[string]inode),
				fileinfo: fileinfo{
					name:    mount,
					modTime: time.Now(),
					mode:    0755 | os.ModeDir,
				},
			},
		}, nil
	})
}

func (fs *customFileSystem) ForwardSlashNameSubstitution(repl string) {
	fs.forwardSlashNameSubstitution = repl
}

// SiteFileSystem returns a FileSystem that maps collections and other
// Arvados objects onto a filesystem layout.
//
// This is experimental: the filesystem layout is not stable, and
// there are significant known bugs and shortcomings. For example,
// writes are not persisted until Sync() is called.
func (c *Client) SiteFileSystem(kc keepClient) CustomFileSystem {
	fs := c.CustomFileSystem(kc)
	fs.MountByID("by_id")
	fs.MountUsers("users")
	return fs
}

func (fs *customFileSystem) Sync() error {
	return fs.root.Sync()
}

// Stale returns true if information obtained at time t should be
// considered stale.
func (fs *customFileSystem) Stale(t time.Time) bool {
	fs.staleLock.Lock()
	defer fs.staleLock.Unlock()
	return !fs.staleThreshold.Before(t)
}

func (fs *customFileSystem) newNode(name string, perm os.FileMode, modTime time.Time) (node inode, err error) {
	return nil, ErrInvalidOperation
}

func (fs *customFileSystem) mountByID(parent inode, id string) inode {
	if strings.Contains(id, "-4zz18-") || pdhRegexp.MatchString(id) {
		return fs.mountCollection(parent, id)
	} else if strings.Contains(id, "-j7d0g-") {
		return fs.newProjectNode(fs.root, id, id)
	} else {
		return nil
	}
}

func (fs *customFileSystem) mountCollection(parent inode, id string) inode {
	var coll Collection
	err := fs.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+id, nil, nil)
	if err != nil {
		return nil
	}
	newfs, err := coll.FileSystem(fs, fs)
	if err != nil {
		return nil
	}
	cfs := newfs.(*collectionFileSystem)
	cfs.SetParent(parent, id)
	return cfs
}

func (fs *customFileSystem) newProjectNode(root inode, name, uuid string) inode {
	return &lookupnode{
		stale:   fs.Stale,
		loadOne: func(parent inode, name string) (inode, error) { return fs.projectsLoadOne(parent, uuid, name) },
		loadAll: func(parent inode) ([]inode, error) { return fs.projectsLoadAll(parent, uuid) },
		treenode: treenode{
			fs:     fs,
			parent: root,
			inodes: make(map[string]inode),
			fileinfo: fileinfo{
				name:    name,
				modTime: time.Now(),
				mode:    0755 | os.ModeDir,
			},
		},
	}
}

// vdirnode wraps an inode by rejecting (with ErrInvalidOperation)
// calls that add/replace children directly, instead calling a
// create() func when a non-existing child is looked up.
//
// create() can return either a new node, which will be added to the
// treenode, or nil for ENOENT.
type vdirnode struct {
	treenode
	create func(parent inode, name string) inode
}

func (vn *vdirnode) Child(name string, replace func(inode) (inode, error)) (inode, error) {
	return vn.treenode.Child(name, func(existing inode) (inode, error) {
		if existing == nil && vn.create != nil {
			existing = vn.create(vn, name)
			if existing != nil {
				existing.SetParent(vn, name)
				vn.treenode.fileinfo.modTime = time.Now()
			}
		}
		if replace == nil {
			return existing, nil
		} else if tryRepl, err := replace(existing); err != nil {
			return existing, err
		} else if tryRepl != existing {
			return existing, ErrInvalidOperation
		} else {
			return existing, nil
		}
	})
}
