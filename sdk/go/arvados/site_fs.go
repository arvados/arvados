// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"time"
)

func (c *Client) SiteFileSystem(kc keepClient) FileSystem {
	root := &treenode{
		fileinfo: fileinfo{
			name:    "/",
			mode:    os.ModeDir | 0755,
			modTime: time.Now(),
		},
		inodes: make(map[string]inode),
	}
	root.parent = root
	root.Child("by_id", func(inode) inode {
		return &vdirnode{
			treenode: treenode{
				parent: root,
				inodes: make(map[string]inode),
				fileinfo: fileinfo{
					name:    "by_id",
					modTime: time.Now(),
					mode:    0755 | os.ModeDir,
				},
			},
			create: func(name string) inode {
				return newEntByID(c, kc, name)
			},
		}
	})
	return &fileSystem{inode: root}
}

func newEntByID(c *Client, kc keepClient, id string) inode {
	var coll Collection
	err := c.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+id, nil, nil)
	if err != nil {
		return nil
	}
	fs, err := coll.FileSystem(c, kc)
	fs.(*collectionFileSystem).inode.(*dirnode).fileinfo.name = id
	if err != nil {
		return nil
	}
	return fs
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
