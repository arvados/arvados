// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"log"
	"os"
	"strings"
	"time"
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
	for _, subst := range []string{"/", fs.forwardSlashNameSubstitution} {
		contents = CollectionList{}
		err = fs.RequestAndDecode(&contents, "GET", "arvados/v1/groups/"+uuid+"/contents", nil, ResourceListParams{
			Count: "none",
			Filters: []Filter{
				{"name", "=", strings.Replace(name, subst, "/", -1)},
				{"uuid", "is_a", []string{"arvados#collection", "arvados#group"}},
				{"groups.group_class", "=", "project"},
			},
			Select: []string{"uuid", "name", "modified_at", "properties"},
		})
		if err != nil {
			return nil, err
		}
		if len(contents.Items) > 0 || fs.forwardSlashNameSubstitution == "/" || fs.forwardSlashNameSubstitution == "" || !strings.Contains(name, fs.forwardSlashNameSubstitution) {
			break
		}
		// If the requested name contains the configured "/"
		// replacement string and didn't match a
		// project/collection exactly, we'll try again with
		// "/" in its place, so a lookup of a munged name
		// works regardless of whether the directory listing
		// has been populated with escaped names.
		//
		// Note this doesn't handle items whose names contain
		// both "/" and the substitution string.
	}
	if len(contents.Items) == 0 {
		return nil, nil
	}
	coll := contents.Items[0]

	if strings.Contains(coll.UUID, "-j7d0g-") {
		// Group item was loaded into a Collection var -- but
		// we only need the Name and UUID anyway, so it's OK.
		return &hardlink{
			inode: fs.projectSingleton(coll.UUID, &Group{
				UUID:       coll.UUID,
				Name:       coll.Name,
				ModifiedAt: coll.ModifiedAt,
				Properties: coll.Properties,
			}),
			parent: parent,
			name:   coll.Name,
		}, nil
	} else if strings.Contains(coll.UUID, "-4zz18-") {
		return fs.newDeferredCollectionDir(parent, name, coll.UUID, coll.ModifiedAt), nil
	} else {
		log.Printf("group contents: unrecognized UUID in response: %q", coll.UUID)
		return nil, ErrInvalidArgument
	}
}

func (fs *customFileSystem) projectsLoadAll(parent inode, uuid string) ([]inode, error) {
	uuid, err := fs.defaultUUID(uuid)
	if err != nil {
		return nil, err
	}

	pagesize := 100000
	var inodes []inode

	// When #17424 is resolved, remove the outer loop here and use
	// []string{"arvados#collection", "arvados#group"} directly as the uuid
	// filter.
	for _, class := range []string{"arvados#collection", "arvados#group"} {
		// Note: the "filters" slice's backing array might be reused
		// by append(filters,...) below. This isn't goroutine safe,
		// but all accesses are in the same goroutine, so it's OK.
		filters := []Filter{
			{"uuid", "is_a", class},
		}
		if class == "arvados#group" {
			filters = append(filters, Filter{"group_class", "=", "project"})
		}

		params := ResourceListParams{
			Count:   "none",
			Filters: filters,
			Order:   "uuid",
			Select:  []string{"uuid", "name", "modified_at", "properties"},
			Limit:   &pagesize,
		}

		for {
			// The groups content endpoint returns
			// Collection and Group (project)
			// objects. This function only accesses the
			// UUID, Name, and ModifiedAt fields. Both
			// collections and groups have those fields,
			// so it is easier to just treat the
			// ObjectList that comes back as a
			// CollectionList.
			var resp CollectionList
			err = fs.RequestAndDecode(&resp, "GET", "arvados/v1/groups/"+uuid+"/contents", nil, params)
			if err != nil {
				return nil, err
			}
			if len(resp.Items) == 0 {
				break
			}
			for _, i := range resp.Items {
				if fs.forwardSlashNameSubstitution != "" {
					i.Name = strings.Replace(i.Name, "/", fs.forwardSlashNameSubstitution, -1)
				}
				if !permittedName(i.Name) {
					continue
				}
				if strings.Contains(i.UUID, "-j7d0g-") {
					inodes = append(inodes, fs.newProjectDir(parent, i.Name, i.UUID, &Group{
						UUID:       i.UUID,
						Name:       i.Name,
						ModifiedAt: i.ModifiedAt,
						Properties: i.Properties,
					}))
				} else if strings.Contains(i.UUID, "-4zz18-") {
					inodes = append(inodes, fs.newDeferredCollectionDir(parent, i.Name, i.UUID, i.ModifiedAt))
				} else {
					log.Printf("group contents: unrecognized UUID in response: %q", i.UUID)
					return nil, ErrInvalidArgument
				}
			}
			params.Filters = append(filters, Filter{"uuid", ">", resp.Items[len(resp.Items)-1].UUID})
		}
	}
	return inodes, nil
}

func (fs *customFileSystem) newProjectDir(parent inode, name, uuid string, proj *Group) inode {
	return &hardlink{inode: fs.projectSingleton(uuid, proj), parent: parent, name: name}
}

func (fs *customFileSystem) newDeferredCollectionDir(parent inode, name, uuid string, modTime time.Time) inode {
	if modTime.IsZero() {
		modTime = time.Now()
	}
	placeholder := &treenode{
		fs:     fs,
		parent: parent,
		inodes: nil,
		fileinfo: fileinfo{
			name:    name,
			modTime: modTime,
			mode:    0755 | os.ModeDir,
			sys:     func() interface{} { return &Collection{UUID: uuid, Name: name, ModifiedAt: modTime} },
		},
	}
	return &deferrednode{wrapped: placeholder, create: func() inode {
		node, err := fs.collectionSingleton(uuid)
		if err != nil {
			log.Printf("BUG: unhandled error: %s", err)
			return placeholder
		}
		return &hardlink{inode: node, parent: parent, name: name}
	}}
}
