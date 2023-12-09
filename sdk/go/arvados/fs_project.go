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

// The groups content endpoint returns Collection and Group (project)
// objects. This struct lets us load the common Items fields for both
// types (UUID, Name, ModifiedAt, and Properties), and GroupClass for
// groups, into one struct.
type groupContentsResponse struct {
	Items []struct {
		UUID       string                 `json:"uuid"`
		Name       string                 `json:"name"`
		ModifiedAt time.Time              `json:"modified_at"`
		GroupClass string                 `json:"group_class"`
		Properties map[string]interface{} `json:"properties"`
	}
}

// loadOneChild loads only the named child, if it exists.
func (fs *customFileSystem) projectsLoadOne(parent inode, uuid, name string) (inode, error) {
	uuid, err := fs.defaultUUID(uuid)
	if err != nil {
		return nil, err
	}

	var resp groupContentsResponse
	for _, subst := range []string{"/", fs.forwardSlashNameSubstitution} {
		resp = groupContentsResponse{}
		err = fs.RequestAndDecode(&resp, "GET", "arvados/v1/groups/"+uuid+"/contents", nil, ResourceListParams{
			Count: "none",
			Filters: []Filter{
				{"name", "=", strings.Replace(name, subst, "/", -1)},
				{"uuid", "is_a", []string{"arvados#collection", "arvados#group"}},
				{"groups.group_class", "in", []string{"project", "filter"}},
			},
			Select: []string{"uuid", "name", "modified_at", "properties", "group_class"},
		})
		if err != nil {
			return nil, err
		}
		if len(resp.Items) > 0 || fs.forwardSlashNameSubstitution == "/" || fs.forwardSlashNameSubstitution == "" || !strings.Contains(name, fs.forwardSlashNameSubstitution) {
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
	if len(resp.Items) == 0 {
		return nil, nil
	}
	item := resp.Items[0]
	isGroup := strings.Contains(item.UUID, "-j7d0g-")
	if strings.Contains(item.UUID, "-4zz18-") {
		return fs.newDeferredCollectionDir(parent, name, item.UUID, item.ModifiedAt, item.Properties), nil
	} else if isGroup && item.GroupClass == "filter" {
		return fs.newCollectionOrProjectSymlink(parent, name, item.UUID, item.ModifiedAt, item.Properties)
	} else if isGroup && item.GroupClass == "project" {
		return &hardlink{
			inode: fs.projectSingleton(item.UUID, &Group{
				UUID:       item.UUID,
				Name:       item.Name,
				ModifiedAt: item.ModifiedAt,
				Properties: item.Properties,
			}),
			parent: parent,
			name:   item.Name,
		}, nil
	} else {
		log.Printf("group contents: unrecognized UUID in response: %q", item.UUID)
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
			filters = append(filters, Filter{"groups.group_class", "in", []string{"project", "filter"}})
		}

		params := ResourceListParams{
			Count:   "none",
			Filters: filters,
			Order:   "uuid",
			Select:  []string{"uuid", "name", "modified_at", "properties", "group_class"},
			Limit:   &pagesize,
		}

		for {
			var resp groupContentsResponse
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
				isGroup := strings.Contains(i.UUID, "-j7d0g-")
				if strings.Contains(i.UUID, "-4zz18-") {
					inodes = append(inodes, fs.newDeferredCollectionDir(parent, i.Name, i.UUID, i.ModifiedAt, i.Properties))
				} else if isGroup && i.GroupClass == "filter" {
					inode, err := fs.newCollectionOrProjectSymlink(parent, i.Name, i.UUID, i.ModifiedAt, i.Properties)
					if err != nil {
						return nil, err
					}
					if inode != nil {
						inodes = append(inodes, inode)
					}
				} else if isGroup && i.GroupClass == "project" {
					inodes = append(inodes, fs.newProjectDir(parent, i.Name, i.UUID, &Group{
						UUID:       i.UUID,
						Name:       i.Name,
						ModifiedAt: i.ModifiedAt,
						Properties: i.Properties,
					}))
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

// create a symlink to the given collection or project. If it's not
// possible to create a symlink because the filesystem does not have a
// "by_id" mount point to put the target in, return (nil, nil).
func (fs *customFileSystem) newCollectionOrProjectSymlink(parent inode, name, targetUUID string, modTime time.Time, props map[string]interface{}) (inode, error) {
	fs.root.treenode.Lock()
	byIDPath := fs.byIDPath
	fs.root.treenode.Unlock()
	if byIDPath == "" {
		return nil, nil
	}
	targetPath := []byte("/" + byIDPath + "/" + targetUUID)
	return &getternode{
		Getter: func() ([]byte, error) { return targetPath, nil },
		treenode: treenode{
			fileinfo: fileinfo{
				name:    name,
				modTime: modTime,
				mode:    os.ModeSymlink,
			},
		},
	}, nil
}

func (fs *customFileSystem) newProjectDir(parent inode, name, uuid string, proj *Group) inode {
	return &hardlink{inode: fs.projectSingleton(uuid, proj), parent: parent, name: name}
}

func (fs *customFileSystem) newDeferredCollectionDir(parent inode, name, uuid string, modTime time.Time, props map[string]interface{}) inode {
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
			sys:     func() interface{} { return &Collection{UUID: uuid, Name: name, ModifiedAt: modTime, Properties: props} },
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
