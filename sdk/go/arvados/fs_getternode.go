// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"os"
	"time"
)

// A getternode is a read-only character device that returns whatever
// data is returned by the supplied function.
type getternode struct {
	Getter func() ([]byte, error)

	treenode
	data *bytes.Reader
}

func (*getternode) IsDir() bool {
	return false
}

func (*getternode) Child(string, func(inode) (inode, error)) (inode, error) {
	return nil, ErrInvalidOperation
}

func (gn *getternode) get() error {
	if gn.data != nil {
		return nil
	}
	data, err := gn.Getter()
	if err != nil {
		return err
	}
	gn.data = bytes.NewReader(data)
	return nil
}

func (gn *getternode) Size() int64 {
	return gn.FileInfo().Size()
}

func (gn *getternode) FileInfo() os.FileInfo {
	gn.Lock()
	defer gn.Unlock()
	var size int64
	if gn.get() == nil {
		size = gn.data.Size()
	}
	return fileinfo{
		modTime: time.Now(),
		mode:    0444,
		size:    size,
	}
}

func (gn *getternode) Read(p []byte, ptr filenodePtr) (int, filenodePtr, error) {
	if err := gn.get(); err != nil {
		return 0, ptr, err
	}
	n, err := gn.data.ReadAt(p, ptr.off)
	return n, filenodePtr{off: ptr.off + int64(n)}, err
}
