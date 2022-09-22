// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
)

func (fs *customFileSystem) usersLoadOne(parent inode, name string) (inode, error) {
	var resp UserList
	err := fs.RequestAndDecode(&resp, "GET", "arvados/v1/users", nil, ResourceListParams{
		Count:   "none",
		Filters: []Filter{{"username", "=", name}},
	})
	if err != nil {
		return nil, err
	} else if len(resp.Items) == 0 {
		return nil, os.ErrNotExist
	}
	user := resp.Items[0]
	return fs.newProjectDir(parent, user.Username, user.UUID, nil), nil
}

func (fs *customFileSystem) usersLoadAll(parent inode) ([]inode, error) {
	params := ResourceListParams{
		Count: "none",
		Order: "uuid",
	}
	var inodes []inode
	for {
		var resp UserList
		err := fs.RequestAndDecode(&resp, "GET", "arvados/v1/users", nil, params)
		if err != nil {
			return nil, err
		} else if len(resp.Items) == 0 {
			return inodes, nil
		}
		for _, user := range resp.Items {
			if user.Username == "" {
				continue
			}
			inodes = append(inodes, fs.newProjectDir(parent, user.Username, user.UUID, nil))
		}
		params.Filters = []Filter{{"uuid", ">", resp.Items[len(resp.Items)-1].UUID}}
	}
}
