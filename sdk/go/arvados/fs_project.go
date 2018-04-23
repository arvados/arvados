// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"log"
	"os"
	"strings"
)

func (fs *customFileSystem) defaultUUID(uuid string) (string, error) {
	if uuid != "" {
		return uuid, nil
	}
	var resp User
	err := fs.RequestAndDecode(&resp, "GET", "arvados/v1/users/current", nil, nil)
	if err != nil {
		return "", err
	}
	return resp.UUID, nil
}

// loadOneChild loads only the named child, if it exists.
func (fs *customFileSystem) projectsLoadOne(parent inode, uuid, name string) (inode, error) {
	uuid, err := fs.defaultUUID(uuid)
	if err != nil {
		return nil, err
	}

	var contents CollectionList
	err = fs.RequestAndDecode(&contents, "GET", "arvados/v1/groups/"+uuid+"/contents", nil, ResourceListParams{
		Count: "none",
		Filters: []Filter{
			{"name", "=", name},
			{"uuid", "is_a", []string{"arvados#collection", "arvados#group"}},
			{"groups.group_class", "=", "project"},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(contents.Items) == 0 {
		return nil, os.ErrNotExist
	}
	coll := contents.Items[0]

	if strings.Contains(coll.UUID, "-j7d0g-") {
		// Group item was loaded into a Collection var -- but
		// we only need the Name and UUID anyway, so it's OK.
		return fs.newProjectNode(parent, coll.Name, coll.UUID), nil
	} else if strings.Contains(coll.UUID, "-4zz18-") {
		return deferredCollectionFS(fs, parent, coll), nil
	} else {
		log.Printf("projectnode: unrecognized UUID in response: %q", coll.UUID)
		return nil, ErrInvalidArgument
	}
}

func (fs *customFileSystem) projectsLoadAll(parent inode, uuid string) ([]inode, error) {
	uuid, err := fs.defaultUUID(uuid)
	if err != nil {
		return nil, err
	}

	var inodes []inode

	// Note: the "filters" slice's backing array might be reused
	// by append(filters,...) below. This isn't goroutine safe,
	// but all accesses are in the same goroutine, so it's OK.
	filters := []Filter{{"owner_uuid", "=", uuid}}
	params := ResourceListParams{
		Count:   "none",
		Filters: filters,
		Order:   "uuid",
	}
	for {
		var resp CollectionList
		err = fs.RequestAndDecode(&resp, "GET", "arvados/v1/collections", nil, params)
		if err != nil {
			return nil, err
		}
		if len(resp.Items) == 0 {
			break
		}
		for _, i := range resp.Items {
			coll := i
			if !permittedName(coll.Name) {
				continue
			}
			inodes = append(inodes, deferredCollectionFS(fs, parent, coll))
		}
		params.Filters = append(filters, Filter{"uuid", ">", resp.Items[len(resp.Items)-1].UUID})
	}

	filters = append(filters, Filter{"group_class", "=", "project"})
	params.Filters = filters
	for {
		var resp GroupList
		err = fs.RequestAndDecode(&resp, "GET", "arvados/v1/groups", nil, params)
		if err != nil {
			return nil, err
		}
		if len(resp.Items) == 0 {
			break
		}
		for _, group := range resp.Items {
			if !permittedName(group.Name) {
				continue
			}
			inodes = append(inodes, fs.newProjectNode(parent, group.Name, group.UUID))
		}
		params.Filters = append(filters, Filter{"uuid", ">", resp.Items[len(resp.Items)-1].UUID})
	}
	return inodes, nil
}
