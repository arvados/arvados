// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"os"
	"sync"
)

type nullnode struct{}

func (*nullnode) Mkdir(string, os.FileMode) error {
	return ErrInvalidOperation
}

func (*nullnode) Read([]byte, filenodePtr) (int, filenodePtr, error) {
	return 0, filenodePtr{}, ErrInvalidOperation
}

func (*nullnode) Write([]byte, filenodePtr) (int, filenodePtr, error) {
	return 0, filenodePtr{}, ErrInvalidOperation
}

func (*nullnode) Truncate(int64) error {
	return ErrInvalidOperation
}

func (*nullnode) FileInfo() os.FileInfo {
	return fileinfo{}
}

func (*nullnode) IsDir() bool {
	return false
}

func (*nullnode) Readdir() []os.FileInfo {
	return nil
}

func (*nullnode) Child(name string, replace func(inode) inode) inode {
	return nil
}

type treenode struct {
	parent   inode
	inodes   map[string]inode
	fileinfo fileinfo
	sync.RWMutex
	nullnode
}

func (n *treenode) Parent() inode {
	n.RLock()
	defer n.RUnlock()
	return n.parent
}

func (n *treenode) IsDir() bool {
	return true
}

func (n *treenode) Child(name string, replace func(inode) inode) (child inode) {
	// TODO: special treatment for "", ".", ".."
	child = n.inodes[name]
	if replace != nil {
		child = replace(child)
		if child == nil {
			delete(n.inodes, name)
		} else {
			n.inodes[name] = child
		}
	}
	return
}

func (n *treenode) Size() int64 {
	return n.FileInfo().Size()
}

func (n *treenode) FileInfo() os.FileInfo {
	n.Lock()
	defer n.Unlock()
	n.fileinfo.size = int64(len(n.inodes))
	return n.fileinfo
}

func (n *treenode) Readdir() (fi []os.FileInfo) {
	n.RLock()
	defer n.RUnlock()
	fi = make([]os.FileInfo, 0, len(n.inodes))
	for _, inode := range n.inodes {
		fi = append(fi, inode.FileInfo())
	}
	return
}
