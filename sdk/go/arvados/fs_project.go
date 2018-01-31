// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"log"
	"os"
	"sync"
)

// projectnode exposes an Arvados project as a filesystem directory.
type projectnode struct {
	inode
	uuid      string
	setupOnce sync.Once
	err       error
}

func (pn *projectnode) setup() {
	fs := pn.FS().(*siteFileSystem)
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
			// TODO: retry on next access, instead of returning the same error forever
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
			pn.inode.Child(coll.Name, func(inode) inode {
				return deferredCollectionFS(fs, pn, coll)
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
			// TODO: retry on next access, instead of returning the same error forever
			return
		}
		if len(resp.Items) == 0 {
			break
		}
		for _, group := range resp.Items {
			if group.Name == "" || group.Name == "." || group.Name == ".." {
				continue
			}
			pn.inode.Child(group.Name, func(inode) inode {
				return fs.newProjectNode(pn, group.Name, group.UUID)
			})
		}
		params.Filters = append(filters, Filter{"uuid", ">", resp.Items[len(resp.Items)-1].UUID})
	}
}

func (pn *projectnode) Readdir() []os.FileInfo {
	pn.setupOnce.Do(pn.setup)
	return pn.inode.Readdir()
}

func (pn *projectnode) Child(name string, replace func(inode) inode) inode {
	pn.setupOnce.Do(pn.setup)
	if pn.err != nil {
		log.Printf("BUG: not propagating error setting up %T %v: %s", pn, pn, pn.err)
		// TODO: propagate error, instead of just being empty
		return nil
	}
	if replace == nil {
		// lookup
		return pn.inode.Child(name, nil)
	}
	return pn.inode.Child(name, func(existing inode) inode {
		if repl := replace(existing); repl == nil {
			// delete
			// (TODO)
			return pn.Child(name, nil) // not implemented
		} else if repl.FileInfo().IsDir() {
			// mkdir
			// TODO: repl.SetParent(pn, name), etc.
			return pn.Child(name, nil) // not implemented
		} else {
			// create file
			// TODO: repl.SetParent(pn, name), etc.
			return pn.Child(name, nil) // not implemented
		}
	})
}
