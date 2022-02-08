// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package mount

import (
	"errors"
	"io"
	"log"
	"os"
	"runtime/debug"
	"sync"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/arvados/cgofuse/fuse"
)

// sharedFile wraps arvados.File with a sync.Mutex, so fuse can safely
// use a single filehandle concurrently on behalf of multiple
// threads/processes.
type sharedFile struct {
	arvados.File
	sync.Mutex
}

// keepFS implements cgofuse's FileSystemInterface.
type keepFS struct {
	fuse.FileSystemBase
	Client     *arvados.Client
	KeepClient *keepclient.KeepClient
	ReadOnly   bool
	Uid        int
	Gid        int

	root   arvados.CustomFileSystem
	open   map[uint64]*sharedFile
	lastFH uint64
	sync.RWMutex

	// If non-nil, this channel will be closed by Init() to notify
	// other goroutines that the mount is ready.
	ready chan struct{}
}

var (
	invalidFH = ^uint64(0)
)

// newFH wraps f in a sharedFile, adds it to fs's lookup table using a
// new handle number, and returns the handle number.
func (fs *keepFS) newFH(f arvados.File) uint64 {
	fs.Lock()
	defer fs.Unlock()
	if fs.open == nil {
		fs.open = make(map[uint64]*sharedFile)
	}
	fs.lastFH++
	fh := fs.lastFH
	fs.open[fh] = &sharedFile{File: f}
	return fh
}

func (fs *keepFS) lookupFH(fh uint64) *sharedFile {
	fs.RLock()
	defer fs.RUnlock()
	return fs.open[fh]
}

func (fs *keepFS) Init() {
	defer fs.debugPanics()
	fs.root = fs.Client.SiteFileSystem(fs.KeepClient)
	fs.root.MountProject("home", "")
	if fs.ready != nil {
		close(fs.ready)
	}
}

func (fs *keepFS) Create(path string, flags int, mode uint32) (errc int, fh uint64) {
	defer fs.debugPanics()
	if fs.ReadOnly {
		return -fuse.EROFS, invalidFH
	}
	f, err := fs.root.OpenFile(path, flags|os.O_CREATE, os.FileMode(mode))
	if err == os.ErrExist {
		return -fuse.EEXIST, invalidFH
	} else if err != nil {
		return -fuse.EINVAL, invalidFH
	}
	return 0, fs.newFH(f)
}

func (fs *keepFS) Open(path string, flags int) (errc int, fh uint64) {
	defer fs.debugPanics()
	if fs.ReadOnly && flags&(os.O_RDWR|os.O_WRONLY|os.O_CREATE) != 0 {
		return -fuse.EROFS, invalidFH
	}
	f, err := fs.root.OpenFile(path, flags, 0)
	if err != nil {
		return -fuse.ENOENT, invalidFH
	} else if fi, err := f.Stat(); err != nil {
		return -fuse.EIO, invalidFH
	} else if fi.IsDir() {
		f.Close()
		return -fuse.EISDIR, invalidFH
	}
	return 0, fs.newFH(f)
}

func (fs *keepFS) Utimens(path string, tmsp []fuse.Timespec) int {
	defer fs.debugPanics()
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	f, err := fs.root.OpenFile(path, 0, 0)
	if err != nil {
		return fs.errCode(err)
	}
	f.Close()
	return 0
}

func (fs *keepFS) errCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, os.ErrNotExist) {
		return -fuse.ENOENT
	}
	if errors.Is(err, os.ErrExist) {
		return -fuse.EEXIST
	}
	if errors.Is(err, arvados.ErrInvalidArgument) {
		return -fuse.EINVAL
	}
	if errors.Is(err, arvados.ErrInvalidOperation) {
		return -fuse.ENOSYS
	}
	if errors.Is(err, arvados.ErrDirectoryNotEmpty) {
		return -fuse.ENOTEMPTY
	}
	return -fuse.EIO
}

func (fs *keepFS) Mkdir(path string, mode uint32) int {
	defer fs.debugPanics()
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	f, err := fs.root.OpenFile(path, os.O_CREATE|os.O_EXCL, os.FileMode(mode)|os.ModeDir)
	if err != nil {
		return fs.errCode(err)
	}
	f.Close()
	return 0
}

func (fs *keepFS) Opendir(path string) (errc int, fh uint64) {
	defer fs.debugPanics()
	f, err := fs.root.OpenFile(path, 0, 0)
	if err != nil {
		return fs.errCode(err), invalidFH
	} else if fi, err := f.Stat(); err != nil {
		return fs.errCode(err), invalidFH
	} else if !fi.IsDir() {
		f.Close()
		return -fuse.ENOTDIR, invalidFH
	}
	return 0, fs.newFH(f)
}

func (fs *keepFS) Releasedir(path string, fh uint64) (errc int) {
	defer fs.debugPanics()
	return fs.Release(path, fh)
}

func (fs *keepFS) Rmdir(path string) int {
	defer fs.debugPanics()
	return fs.errCode(fs.root.Remove(path))
}

func (fs *keepFS) Release(path string, fh uint64) (errc int) {
	defer fs.debugPanics()
	fs.Lock()
	defer fs.Unlock()
	defer delete(fs.open, fh)
	if f := fs.open[fh]; f != nil {
		err := f.Close()
		if err != nil {
			return -fuse.EIO
		}
	}
	return 0
}

func (fs *keepFS) Rename(oldname, newname string) (errc int) {
	defer fs.debugPanics()
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	return fs.errCode(fs.root.Rename(oldname, newname))
}

func (fs *keepFS) Unlink(path string) (errc int) {
	defer fs.debugPanics()
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	return fs.errCode(fs.root.Remove(path))
}

func (fs *keepFS) Truncate(path string, size int64, fh uint64) (errc int) {
	defer fs.debugPanics()
	if fs.ReadOnly {
		return -fuse.EROFS
	}

	// Sometimes fh is a valid filehandle and we don't need to
	// waste a name lookup.
	if f := fs.lookupFH(fh); f != nil {
		return fs.errCode(f.Truncate(size))
	}

	// Other times, fh is invalid and we need to lookup path.
	f, err := fs.root.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return fs.errCode(err)
	}
	defer f.Close()
	return fs.errCode(f.Truncate(size))
}

func (fs *keepFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) (errc int) {
	defer fs.debugPanics()
	var fi os.FileInfo
	var err error
	if f := fs.lookupFH(fh); f != nil {
		// Valid filehandle -- ignore path.
		fi, err = f.Stat()
	} else {
		// Invalid filehandle -- lookup path.
		fi, err = fs.root.Stat(path)
	}
	if err != nil {
		return fs.errCode(err)
	}
	fs.fillStat(stat, fi)
	return 0
}

func (fs *keepFS) Chmod(path string, mode uint32) (errc int) {
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	if fi, err := fs.root.Stat(path); err != nil {
		return fs.errCode(err)
	} else if mode & ^uint32(fuse.S_IFREG|fuse.S_IFDIR|0777) != 0 {
		// Refuse to set mode bits other than
		// regfile/dir/perms
		return -fuse.ENOSYS
	} else if (fi.Mode()&os.ModeDir != 0) != (mode&fuse.S_IFDIR != 0) {
		// Refuse to transform a regular file to a dir, or
		// vice versa
		return -fuse.ENOSYS
	}
	// As long as the change isn't nonsense, chmod is a no-op,
	// because we don't save permission bits.
	return 0
}

func (fs *keepFS) fillStat(stat *fuse.Stat_t, fi os.FileInfo) {
	defer fs.debugPanics()
	var m uint32
	if fi.IsDir() {
		m = m | fuse.S_IFDIR
	} else {
		m = m | fuse.S_IFREG
	}
	m = m | uint32(fi.Mode()&os.ModePerm)
	stat.Mode = m
	stat.Nlink = 1
	stat.Size = fi.Size()
	t := fuse.NewTimespec(fi.ModTime())
	stat.Mtim = t
	stat.Ctim = t
	stat.Atim = t
	stat.Birthtim = t
	stat.Blksize = 1024
	stat.Blocks = (stat.Size + stat.Blksize - 1) / stat.Blksize
	if fs.Uid > 0 && int64(fs.Uid) < 1<<31 {
		stat.Uid = uint32(fs.Uid)
	}
	if fs.Gid > 0 && int64(fs.Gid) < 1<<31 {
		stat.Gid = uint32(fs.Gid)
	}
}

func (fs *keepFS) Write(path string, buf []byte, ofst int64, fh uint64) (n int) {
	defer fs.debugPanics()
	if fs.ReadOnly {
		return -fuse.EROFS
	}
	f := fs.lookupFH(fh)
	if f == nil {
		return -fuse.EBADF
	}
	f.Lock()
	defer f.Unlock()
	if _, err := f.Seek(ofst, io.SeekStart); err != nil {
		return fs.errCode(err)
	}
	n, err := f.Write(buf)
	if err != nil {
		log.Printf("error writing %q: %s", path, err)
		return fs.errCode(err)
	}
	return n
}

func (fs *keepFS) Read(path string, buf []byte, ofst int64, fh uint64) (n int) {
	defer fs.debugPanics()
	f := fs.lookupFH(fh)
	if f == nil {
		return -fuse.EBADF
	}
	f.Lock()
	defer f.Unlock()
	if _, err := f.Seek(ofst, io.SeekStart); err != nil {
		return fs.errCode(err)
	}
	n, err := f.Read(buf)
	for err == nil && n < len(buf) {
		// f is an io.Reader ("If some data is available but
		// not len(p) bytes, Read conventionally returns what
		// is available instead of waiting for more") -- but
		// our caller requires us to either fill buf or reach
		// EOF.
		done := n
		n, err = f.Read(buf[done:])
		n += done
	}
	if err != nil && err != io.EOF {
		log.Printf("error reading %q: %s", path, err)
		return fs.errCode(err)
	}
	return n
}

func (fs *keepFS) Readdir(path string,
	fill func(name string, stat *fuse.Stat_t, ofst int64) bool,
	ofst int64,
	fh uint64) (errc int) {
	defer fs.debugPanics()
	f := fs.lookupFH(fh)
	if f == nil {
		return -fuse.EBADF
	}
	fill(".", nil, 0)
	fill("..", nil, 0)
	var stat fuse.Stat_t
	fis, err := f.Readdir(-1)
	if err != nil {
		return fs.errCode(err)
	}
	for _, fi := range fis {
		fs.fillStat(&stat, fi)
		fill(fi.Name(), &stat, 0)
	}
	return 0
}

func (fs *keepFS) Fsync(path string, datasync bool, fh uint64) int {
	defer fs.debugPanics()
	f := fs.lookupFH(fh)
	if f == nil {
		return -fuse.EBADF
	}
	return fs.errCode(f.Sync())
}

func (fs *keepFS) Fsyncdir(path string, datasync bool, fh uint64) int {
	return fs.Fsync(path, datasync, fh)
}

// debugPanics (when deferred by keepFS handlers) prints an error and
// stack trace on stderr when a handler crashes. (Without this,
// cgofuse recovers from panics silently and returns EIO.)
func (fs *keepFS) debugPanics() {
	if err := recover(); err != nil {
		log.Printf("(%T) %v", err, err)
		debug.PrintStack()
		panic(err)
	}
}
