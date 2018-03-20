// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
)

// usersnode is a virtual directory with an entry for each visible
// Arvados username, each showing the respective user's "home
// projects".
type usersnode struct {
	inode
	staleChecker
	err error
}

func (un *usersnode) load() {
	fs := un.FS().(*customFileSystem)

	params := ResourceListParams{
		Order: "uuid",
	}
	for {
		var resp UserList
		un.err = fs.RequestAndDecode(&resp, "GET", "arvados/v1/users", nil, params)
		if un.err != nil {
			return
		}
		if len(resp.Items) == 0 {
			break
		}
		for _, user := range resp.Items {
			if user.Username == "" {
				continue
			}
			un.inode.Child(user.Username, func(inode) (inode, error) {
				return fs.newProjectNode(un, user.Username, user.UUID), nil
			})
		}
		params.Filters = []Filter{{"uuid", ">", resp.Items[len(resp.Items)-1].UUID}}
	}
	un.err = nil
}

func (un *usersnode) Readdir() ([]os.FileInfo, error) {
	un.staleChecker.DoIfStale(un.load, un.FS().(*customFileSystem).Stale)
	if un.err != nil {
		return nil, un.err
	}
	return un.inode.Readdir()
}

func (un *usersnode) Child(name string, _ func(inode) (inode, error)) (inode, error) {
	un.staleChecker.DoIfStale(un.load, un.FS().(*customFileSystem).Stale)
	if un.err != nil {
		return nil, un.err
	}
	return un.inode.Child(name, nil)
}
