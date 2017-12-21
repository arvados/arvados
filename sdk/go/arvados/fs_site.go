// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"time"
)

type siteFileSystem struct {
	fileSystem
}

// SiteFileSystem returns a FileSystem that maps collections and other
// Arvados objects onto a filesystem layout.
//
// This is experimental: the filesystem layout is not stable, and
// there are significant known bugs and shortcomings. For example,
// writes are not persisted until Sync() is called.
func (c *Client) SiteFileSystem(kc keepClient) FileSystem {
	fs := &siteFileSystem{
		fileSystem: fileSystem{
			fsBackend: keepBackend{apiClient: c, keepClient: kc},
		},
	}
	root := &treenode{
		fs: fs,
		fileinfo: fileinfo{
			name:    "/",
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
		},
		inodes: make(map[string]inode),
	}
	root.parent = root
	root.Child("by_id", func(inode) inode {
		var vn inode
		vn = &vdirnode{
			inode: &treenode{
				fs:     fs,
				parent: root,
				inodes: make(map[string]inode),
				fileinfo: fileinfo{
					name:    "by_id",
					modTime: time.Now(),
					mode:    0755 | os.ModeDir,
				},
			},
			create: fs.mountCollection,
		}
		return vn
	})
	fs.root = root
	return fs
}

func (fs *siteFileSystem) newNode(name string, perm os.FileMode, modTime time.Time) (node inode, err error) {
	return nil, ErrInvalidOperation
}

func (fs *siteFileSystem) mountCollection(parent inode, id string) inode {
	var coll Collection
	err := fs.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+id, nil, nil)
	if err != nil {
		return nil
	}
	cfs, err := coll.FileSystem(fs, fs)
	if err != nil {
		return nil
	}
	root := cfs.rootnode()
	root.SetParent(parent)
	root.(*dirnode).fileinfo.name = id
	return root
}

// vdirnode wraps an inode by ignoring any requests to add/replace
// children, and calling a create() func when a non-existing child is
// looked up.
//
// create() can return either a new node, which will be added to the
// treenode, or nil for ENOENT.
type vdirnode struct {
	inode
	create func(parent inode, name string) inode
}

func (vn *vdirnode) Child(name string, _ func(inode) inode) inode {
	return vn.inode.Child(name, func(existing inode) inode {
		if existing != nil {
			return existing
		} else {
			n := vn.create(vn, name)
			if n != nil {
				n.SetParent(vn)
			}
			return n
		}
	})
}
