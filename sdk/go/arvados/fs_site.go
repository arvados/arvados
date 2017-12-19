// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"time"
)

// SiteFileSystem returns a FileSystem that maps collections and other
// Arvados objects onto a filesystem layout.
//
// This is experimental: the filesystem layout is not stable, and
// there are significant known bugs and shortcomings. For example,
// although the FileSystem allows files to be added and modified in
// collections, these changes are not persistent or visible to other
// Arvados clients.
func (c *Client) SiteFileSystem(kc keepClient) FileSystem {
	fs := &fileSystem{
		fsBackend: keepBackend{apiClient: c, keepClient: kc},
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
			treenode: treenode{
				fs:     fs,
				parent: root,
				inodes: make(map[string]inode),
				fileinfo: fileinfo{
					name:    "by_id",
					modTime: time.Now(),
					mode:    0755 | os.ModeDir,
				},
			},
			create: func(name string) inode {
				return newEntByID(vn, name)
			},
		}
		return vn
	})
	fs.root = root
	return fs
}

func newEntByID(parent inode, id string) inode {
	var coll Collection
	err := parent.FS().RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+id, nil, nil)
	if err != nil {
		return nil
	}
	fs, err := coll.FileSystem(parent.FS(), parent.FS())
	if err != nil {
		return nil
	}
	root := fs.(*collectionFileSystem).root.(*dirnode)
	root.fileinfo.name = id
	root.parent = parent
	return root
}

type vdirnode struct {
	treenode
	create func(string) inode
}

func (vn *vdirnode) Child(name string, _ func(inode) inode) inode {
	return vn.treenode.Child(name, func(existing inode) inode {
		if existing != nil {
			return existing
		} else {
			return vn.create(name)
		}
	})
}
