// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"crypto/rand"
	"errors"
	"fmt"
	prand "math/rand"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"

	"golang.org/x/net/context"
	"golang.org/x/net/webdav"
)

var (
	lockPrefix     string = uuid()
	nextLockSuffix int64  = prand.Int63()
	errReadOnly           = errors.New("read-only filesystem")
)

// webdavFS implements a read-only webdav.FileSystem by wrapping an
// arvados.CollectionFilesystem.
type webdavFS struct {
	collfs arvados.CollectionFileSystem
}

var _ webdav.FileSystem = &webdavFS{}

func (fs *webdavFS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	return errReadOnly
}

func (fs *webdavFS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (webdav.File, error) {
	fi, err := fs.collfs.Stat(name)
	if err != nil {
		return nil, err
	}
	return &webdavFile{collfs: fs.collfs, fileInfo: fi, name: name}, nil
}

func (fs *webdavFS) RemoveAll(ctx context.Context, name string) error {
	return errReadOnly
}

func (fs *webdavFS) Rename(ctx context.Context, oldName, newName string) error {
	return errReadOnly
}

func (fs *webdavFS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return fs.collfs.Stat(name)
}

// webdavFile implements a read-only webdav.File by wrapping
// http.File.
//
// The http.File is opened from an arvados.CollectionFileSystem, but
// not until Seek, Read, or Readdir is called. This deferred-open
// strategy makes webdav's OpenFile-Stat-Close cycle fast even though
// the collfs's Open method is slow. This is relevant because webdav
// does OpenFile-Stat-Close on each file when preparing directory
// listings.
//
// Writes to a webdavFile always fail.
type webdavFile struct {
	// fields populated by (*webdavFS).OpenFile()
	collfs   http.FileSystem
	fileInfo os.FileInfo
	name     string

	// internal fields
	file     http.File
	loadOnce sync.Once
	err      error
}

func (f *webdavFile) load() {
	f.file, f.err = f.collfs.Open(f.name)
}

func (f *webdavFile) Write([]byte) (int, error) {
	return 0, errReadOnly
}

func (f *webdavFile) Seek(offset int64, whence int) (int64, error) {
	f.loadOnce.Do(f.load)
	if f.err != nil {
		return 0, f.err
	}
	return f.file.Seek(offset, whence)
}

func (f *webdavFile) Read(buf []byte) (int, error) {
	f.loadOnce.Do(f.load)
	if f.err != nil {
		return 0, f.err
	}
	return f.file.Read(buf)
}

func (f *webdavFile) Close() error {
	if f.file == nil {
		// We never called load(), or load() failed
		return f.err
	}
	return f.file.Close()
}

func (f *webdavFile) Readdir(n int) ([]os.FileInfo, error) {
	f.loadOnce.Do(f.load)
	if f.err != nil {
		return nil, f.err
	}
	return f.file.Readdir(n)
}

func (f *webdavFile) Stat() (os.FileInfo, error) {
	return f.fileInfo, nil
}

// noLockSystem implements webdav.LockSystem by returning success for
// every possible locking operation, even though it has no side
// effects such as actually locking anything. This works for a
// read-only webdav filesystem because webdav locks only apply to
// writes.
//
// This is more suitable than webdav.NewMemLS() for two reasons:
// First, it allows keep-web to use one locker for all collections
// even though coll1.vhost/foo and coll2.vhost/foo have the same path
// but represent different resources. Additionally, it returns valid
// tokens (rfc2518 specifies that tokens are represented as URIs and
// are unique across all resources for all time), which might improve
// client compatibility.
//
// However, it does also permit impossible operations, like acquiring
// conflicting locks and releasing non-existent locks.  This might
// confuse some clients if they try to probe for correctness.
//
// Currently this is a moot point: the LOCK and UNLOCK methods are not
// accepted by keep-web, so it suffices to implement the
// webdav.LockSystem interface.
type noLockSystem struct{}

func (*noLockSystem) Confirm(time.Time, string, string, ...webdav.Condition) (func(), error) {
	return noop, nil
}

func (*noLockSystem) Create(now time.Time, details webdav.LockDetails) (token string, err error) {
	return fmt.Sprintf("opaquelocktoken:%s-%x", lockPrefix, atomic.AddInt64(&nextLockSuffix, 1)), nil
}

func (*noLockSystem) Refresh(now time.Time, token string, duration time.Duration) (webdav.LockDetails, error) {
	return webdav.LockDetails{}, nil
}

func (*noLockSystem) Unlock(now time.Time, token string) error {
	return nil
}

func noop() {}

// Return a version 1 variant 4 UUID, meaning all bits are random
// except the ones indicating the version and variant.
func uuid() string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic(err)
	}
	// variant 1: N=10xx
	data[8] = data[8]&0x3f | 0x80
	// version 4: M=0100
	data[6] = data[6]&0x0f | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", data[0:4], data[4:6], data[6:8], data[8:10], data[10:])
}
