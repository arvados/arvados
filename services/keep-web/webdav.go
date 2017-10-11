// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"errors"
	"net/http"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
)

var errReadOnly = errors.New("read-only filesystem")

// webdavFS implements a read-only webdav.FileSystem by wrapping
// http.Filesystem.
type webdavFS struct {
	httpfs http.FileSystem
}

var _ webdav.FileSystem = &webdavFS{}

func (fs *webdavFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return errReadOnly
}

func (fs *webdavFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	f, err := fs.httpfs.Open(name)
	if err != nil {
		return nil, err
	}
	return &webdavFile{File: f}, nil
}

func (fs *webdavFS) RemoveAll(ctx context.Context, name string) error {
	return errReadOnly
}

func (fs *webdavFS) Rename(ctx context.Context, oldName, newName string) error {
	return errReadOnly
}

func (fs *webdavFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if f, err := fs.httpfs.Open(name); err != nil {
		return nil, err
	} else {
		return f.Stat()
	}
}

// webdavFile implements a read-only webdav.File by wrapping
// http.File. Writes fail.
type webdavFile struct {
	http.File
}

func (f *webdavFile) Write([]byte) (int, error) {
	return 0, errReadOnly
}
