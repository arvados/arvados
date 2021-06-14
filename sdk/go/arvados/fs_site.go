// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"fmt"
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
	fs.root.treenode.Child(mount, func(inode) (inode, error) {
		return fs.newProjectNode(fs.root, mount, uuid), nil
	})
}

func (fs *customFileSystem) MountUsers(mount string) {
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
	return nil, ErrInvalidArgument
}

func (fs *customFileSystem) mountByID(parent inode, id string) inode {
	if strings.Contains(id, "-4zz18-") || pdhRegexp.MatchString(id) {
		return fs.mountCollection(parent, id)
	} else if strings.Contains(id, "-j7d0g-") {
		return fs.newProjectNode(fs.root, id, id)
	} else if strings.Contains(id, "-dz642-") {
		return fs.mountContainer(fs.root, id)
	} else if strings.Contains(id, "-xvhdp-") {
		return fs.mountContainerRequest(fs.root, id)
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

func (fs *customFileSystem) mountContainerRequest(parent inode, id string) inode {
	return &lookupnode{
		stale: fs.Stale,
		loadAll: func(parent inode) ([]inode, error) {
			var all []inode
			var cr ContainerRequest
			err := fs.RequestAndDecode(&cr, "GET", "arvados/v1/container_requests/"+id, nil, nil)
			if err != nil {
				return nil, err
			}
			jsondata, _ := json.MarshalIndent(cr, "", "  ")
			jsondata = append(jsondata, '\n')
			if cr.LogUUID != "" {
				all = append(all, &getternode{Getter: func() ([]byte, error) { return []byte("../" + cr.LogUUID), nil }, treenode: treenode{fileinfo: fileinfo{name: "log", mode: os.ModeSymlink}}})
			}
			if cr.OutputUUID != "" {
				all = append(all, &getternode{Getter: func() ([]byte, error) { return []byte("../" + cr.OutputUUID), nil }, treenode: treenode{fileinfo: fileinfo{name: "output", mode: os.ModeSymlink}}})
			}
			all = append(all,
				&getternode{Getter: func() ([]byte, error) { return jsondata, nil }, treenode: treenode{fileinfo: fileinfo{name: "json"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte(cr.UUID + "\n"), nil }, treenode: treenode{fileinfo: fileinfo{name: "uuid"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte(cr.State + "\n"), nil }, treenode: treenode{fileinfo: fileinfo{name: "state"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte(cr.ContainerImage + "\n"), nil }, treenode: treenode{fileinfo: fileinfo{name: "container_image"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte("../" + cr.ContainerUUID), nil }, treenode: treenode{fileinfo: fileinfo{name: "container", mode: os.ModeSymlink}}},
			)
			if cr.RequestingContainerUUID != "" {
				all = append(all, &getternode{Getter: func() ([]byte, error) { return []byte("../" + cr.RequestingContainerUUID), nil }, treenode: treenode{fileinfo: fileinfo{name: "requesting_container", mode: os.ModeSymlink}}})
			}
			return all, nil
		},
		treenode: treenode{
			fs:     fs,
			parent: parent,
			inodes: make(map[string]inode),
			fileinfo: fileinfo{
				name:    id,
				modTime: time.Now(),
				mode:    0755 | os.ModeDir,
			},
		},
	}
}

func (fs *customFileSystem) mountContainer(parent inode, id string) inode {
	node := &lookupnode{
		stale: fs.Stale,
		loadAll: func(parent inode) ([]inode, error) {
			var all []inode
			var c Container
			err := fs.RequestAndDecode(&c, "GET", "arvados/v1/containers/"+id, nil, nil)
			if err != nil {
				return nil, err
			}
			jsondata, _ := json.MarshalIndent(c, "", "  ")
			jsondata = append(jsondata, '\n')
			if c.Log != "" {
				all = append(all, &getternode{Getter: func() ([]byte, error) { return []byte("../" + c.Log), nil }, treenode: treenode{fileinfo: fileinfo{name: "log", mode: os.ModeSymlink}}})
			}
			if c.Output != "" {
				all = append(all, &getternode{Getter: func() ([]byte, error) { return []byte("../" + c.Output), nil }, treenode: treenode{fileinfo: fileinfo{name: "output", mode: os.ModeSymlink}}})
			}
			all = append(all,
				&getternode{Getter: func() ([]byte, error) { return jsondata, nil }, treenode: treenode{fileinfo: fileinfo{name: "json"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte(c.UUID + "\n"), nil }, treenode: treenode{fileinfo: fileinfo{name: "uuid"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte(c.State + "\n"), nil }, treenode: treenode{fileinfo: fileinfo{name: "state"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte("../" + c.ContainerImage), nil }, treenode: treenode{fileinfo: fileinfo{name: "container_image", mode: os.ModeSymlink}}},
				&getternode{Getter: func() ([]byte, error) { return []byte(c.GatewayAddress + "\n"), nil }, treenode: treenode{fileinfo: fileinfo{name: "gateway_address"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte(fmt.Sprintf("%d\n", c.ExitCode)), nil }, treenode: treenode{fileinfo: fileinfo{name: "exit_code"}}},
				&getternode{Getter: func() ([]byte, error) { return []byte(fmt.Sprintf("%v\n", c.InteractiveSessionStarted)), nil }, treenode: treenode{fileinfo: fileinfo{name: "interactive_session_started"}}},
			)
			return all, nil
		},
		treenode: treenode{
			fs:     fs,
			parent: parent,
			inodes: make(map[string]inode),
			fileinfo: fileinfo{
				name:    id,
				modTime: time.Now(),
				mode:    0755 | os.ModeDir,
			},
		},
	}
	return node
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

// vdirnode wraps an inode by rejecting (with ErrInvalidArgument)
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
			return existing, ErrInvalidArgument
		} else {
			return existing, nil
		}
	})
}
