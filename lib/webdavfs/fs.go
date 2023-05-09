// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

// Package webdavfs adds special behaviors to an arvados.FileSystem so
// it's suitable to use with a webdav server.
package webdavfs

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	prand "math/rand"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"golang.org/x/net/webdav"
)

var (
	lockPrefix     string = uuid()
	nextLockSuffix int64  = prand.Int63()
	ErrReadOnly           = errors.New("read-only filesystem")
)

// FS implements a webdav.FileSystem by wrapping an
// arvados.CollectionFilesystem.
type FS struct {
	FileSystem arvados.FileSystem
	// Prefix works like fs.Sub: Stat(name) calls
	// Stat(prefix+name) in the wrapped filesystem.
	Prefix string
	// If Writing is false, all write operations return errors.
	// (Opening a file for writing succeeds -- otherwise webdav
	// would return 404 -- but writing to it fails.)
	Writing bool
	// webdav PROPFIND reads the first few bytes of each file
	// whose filename extension isn't recognized, which is
	// prohibitively expensive: we end up fetching multiple 64MiB
	// blocks. Avoid this by returning EOF on all reads when
	// handling a PROPFIND.
	AlwaysReadEOF bool
}

func (fs *FS) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if !fs.Writing {
		return ErrReadOnly
	}
	name = strings.TrimRight(name, "/")
	return fs.FileSystem.Mkdir(fs.Prefix+name, 0755)
}

func (fs *FS) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (f webdav.File, err error) {
	writing := flag&(os.O_WRONLY|os.O_RDWR|os.O_TRUNC) != 0
	f, err = fs.FileSystem.OpenFile(fs.Prefix+name, flag, perm)
	if !fs.Writing {
		// webdav module returns 404 on all OpenFile errors,
		// but returns 405 Method Not Allowed if OpenFile()
		// succeeds but Write() or Close() fails. We'd rather
		// have 405. writeFailer ensures Close() fails if the
		// file is opened for writing *or* Write() is called.
		var err error
		if writing {
			err = ErrReadOnly
		}
		f = writeFailer{File: f, err: err}
	}
	if fs.AlwaysReadEOF {
		f = readEOF{File: f}
	}
	return
}

func (fs *FS) RemoveAll(ctx context.Context, name string) error {
	return fs.FileSystem.RemoveAll(fs.Prefix + name)
}

func (fs *FS) Rename(ctx context.Context, oldName, newName string) error {
	if !fs.Writing {
		return ErrReadOnly
	}
	if strings.HasSuffix(oldName, "/") {
		// WebDAV "MOVE foo/ bar/" means rename foo to bar.
		oldName = oldName[:len(oldName)-1]
		newName = strings.TrimSuffix(newName, "/")
	}
	return fs.FileSystem.Rename(fs.Prefix+oldName, fs.Prefix+newName)
}

func (fs *FS) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	return fs.FileSystem.Stat(fs.Prefix + name)
}

type writeFailer struct {
	webdav.File
	err error
}

func (wf writeFailer) Write([]byte) (int, error) {
	wf.err = ErrReadOnly
	return 0, wf.err
}

func (wf writeFailer) Close() error {
	err := wf.File.Close()
	if err != nil {
		wf.err = err
	}
	return wf.err
}

type readEOF struct {
	webdav.File
}

func (readEOF) Read(p []byte) (int, error) {
	return 0, io.EOF
}

// NoLockSystem implements webdav.LockSystem by returning success for
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
var NoLockSystem = noLockSystem{}

type noLockSystem struct{}

func (noLockSystem) Confirm(time.Time, string, string, ...webdav.Condition) (func(), error) {
	return noop, nil
}

func (noLockSystem) Create(now time.Time, details webdav.LockDetails) (token string, err error) {
	return fmt.Sprintf("opaquelocktoken:%s-%x", lockPrefix, atomic.AddInt64(&nextLockSuffix, 1)), nil
}

func (noLockSystem) Refresh(now time.Time, token string, duration time.Duration) (webdav.LockDetails, error) {
	return webdav.LockDetails{}, nil
}

func (noLockSystem) Unlock(now time.Time, token string) error {
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
