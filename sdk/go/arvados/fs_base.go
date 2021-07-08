// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

var (
	ErrReadOnlyFile      = errors.New("read-only file")
	ErrNegativeOffset    = errors.New("cannot seek to negative offset")
	ErrFileExists        = errors.New("file exists")
	ErrInvalidOperation  = errors.New("invalid operation")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrDirectoryNotEmpty = errors.New("directory not empty")
	ErrWriteOnlyMode     = errors.New("file is O_WRONLY")
	ErrSyncNotSupported  = errors.New("O_SYNC flag is not supported")
	ErrIsDirectory       = errors.New("cannot rename file to overwrite existing directory")
	ErrNotADirectory     = errors.New("not a directory")
	ErrPermission        = os.ErrPermission
	DebugLocksPanicMode  = false
)

type syncer interface {
	Sync() error
}

func debugPanicIfNotLocked(l sync.Locker, writing bool) {
	if !DebugLocksPanicMode {
		return
	}
	race := false
	if rl, ok := l.(interface {
		RLock()
		RUnlock()
	}); ok && writing {
		go func() {
			// Fail if we can grab the read lock during an
			// operation that purportedly has write lock.
			rl.RLock()
			race = true
			rl.RUnlock()
		}()
	} else {
		go func() {
			l.Lock()
			race = true
			l.Unlock()
		}()
	}
	time.Sleep(100)
	if race {
		panic("bug: caller-must-have-lock func called, but nobody has lock")
	}
}

// A File is an *os.File-like interface for reading and writing files
// in a FileSystem.
type File interface {
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
	Size() int64
	Readdir(int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
	Truncate(int64) error
	Sync() error
}

// A FileSystem is an http.Filesystem plus Stat() and support for
// opening writable files. All methods are safe to call from multiple
// goroutines.
type FileSystem interface {
	http.FileSystem
	fsBackend

	rootnode() inode

	// filesystem-wide lock: used by Rename() to prevent deadlock
	// while locking multiple inodes.
	locker() sync.Locker

	// throttle for limiting concurrent background writers
	throttle() *throttle

	// create a new node with nil parent.
	newNode(name string, perm os.FileMode, modTime time.Time) (node inode, err error)

	// analogous to os.Stat()
	Stat(name string) (os.FileInfo, error)

	// analogous to os.Create(): create/truncate a file and open it O_RDWR.
	Create(name string) (File, error)

	// Like os.OpenFile(): create or open a file or directory.
	//
	// If flag&os.O_EXCL==0, it opens an existing file or
	// directory if one exists. If flag&os.O_CREATE!=0, it creates
	// a new empty file or directory if one does not already
	// exist.
	//
	// When creating a new item, perm&os.ModeDir determines
	// whether it is a file or a directory.
	//
	// A file can be opened multiple times and used concurrently
	// from multiple goroutines. However, each File object should
	// be used by only one goroutine at a time.
	OpenFile(name string, flag int, perm os.FileMode) (File, error)

	Mkdir(name string, perm os.FileMode) error
	Remove(name string) error
	RemoveAll(name string) error
	Rename(oldname, newname string) error

	// Write buffered data from memory to storage, returning when
	// all updates have been saved to persistent storage.
	Sync() error

	// Write buffered data from memory to storage, but don't wait
	// for all writes to finish before returning. If shortBlocks
	// is true, flush everything; otherwise, if there's less than
	// a full block of buffered data at the end of a stream, leave
	// it buffered in memory in case more data can be appended. If
	// path is "", flush all dirs/streams; otherwise, flush only
	// the specified dir/stream.
	Flush(path string, shortBlocks bool) error

	// Estimate current memory usage.
	MemorySize() int64
}

type inode interface {
	SetParent(parent inode, name string)
	Parent() inode
	FS() FileSystem
	Read([]byte, filenodePtr) (int, filenodePtr, error)
	Write([]byte, filenodePtr) (int, filenodePtr, error)
	Truncate(int64) error
	IsDir() bool
	Readdir() ([]os.FileInfo, error)
	Size() int64
	FileInfo() os.FileInfo

	// Child() performs lookups and updates of named child nodes.
	//
	// (The term "child" here is used strictly. This means name is
	// not "." or "..", and name does not contain "/".)
	//
	// If replace is non-nil, Child calls replace(x) where x is
	// the current child inode with the given name. If possible,
	// the child inode is replaced with the one returned by
	// replace().
	//
	// If replace(x) returns an inode (besides x or nil) that is
	// subsequently returned by Child(), then Child()'s caller
	// must ensure the new child's name and parent are set/updated
	// to Child()'s name argument and its receiver respectively.
	// This is not necessarily done before replace(x) returns, but
	// it must be done before Child()'s caller releases the
	// parent's lock.
	//
	// Nil represents "no child". replace(nil) signifies that no
	// child with this name exists yet. If replace() returns nil,
	// the existing child should be deleted if possible.
	//
	// An implementation of Child() is permitted to ignore
	// replace() or its return value. For example, a regular file
	// inode does not have children, so Child() always returns
	// nil.
	//
	// Child() returns the child, if any, with the given name: if
	// a child was added or changed, the new child is returned.
	//
	// Caller must have lock (or rlock if replace is nil).
	Child(name string, replace func(inode) (inode, error)) (inode, error)

	sync.Locker
	RLock()
	RUnlock()
	MemorySize() int64
}

type fileinfo struct {
	name    string
	mode    os.FileMode
	size    int64
	modTime time.Time
}

// Name implements os.FileInfo.
func (fi fileinfo) Name() string {
	return fi.name
}

// ModTime implements os.FileInfo.
func (fi fileinfo) ModTime() time.Time {
	return fi.modTime
}

// Mode implements os.FileInfo.
func (fi fileinfo) Mode() os.FileMode {
	return fi.mode
}

// IsDir implements os.FileInfo.
func (fi fileinfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Size implements os.FileInfo.
func (fi fileinfo) Size() int64 {
	return fi.size
}

// Sys implements os.FileInfo.
func (fi fileinfo) Sys() interface{} {
	return nil
}

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

func (*nullnode) Readdir() ([]os.FileInfo, error) {
	return nil, ErrInvalidOperation
}

func (*nullnode) Child(name string, replace func(inode) (inode, error)) (inode, error) {
	return nil, ErrNotADirectory
}

func (*nullnode) MemorySize() int64 {
	// Types that embed nullnode should report their own size, but
	// if they don't, we at least report a non-zero size to ensure
	// a large tree doesn't get reported as 0 bytes.
	return 64
}

type treenode struct {
	fs       FileSystem
	parent   inode
	inodes   map[string]inode
	fileinfo fileinfo
	sync.RWMutex
	nullnode
}

func (n *treenode) FS() FileSystem {
	return n.fs
}

func (n *treenode) SetParent(p inode, name string) {
	n.Lock()
	defer n.Unlock()
	n.parent = p
	n.fileinfo.name = name
}

func (n *treenode) Parent() inode {
	n.RLock()
	defer n.RUnlock()
	return n.parent
}

func (n *treenode) IsDir() bool {
	return true
}

func (n *treenode) Child(name string, replace func(inode) (inode, error)) (child inode, err error) {
	debugPanicIfNotLocked(n, false)
	child = n.inodes[name]
	if name == "" || name == "." || name == ".." {
		err = ErrInvalidArgument
		return
	}
	if replace == nil {
		return
	}
	newchild, err := replace(child)
	if err != nil {
		return
	}
	if newchild == nil {
		debugPanicIfNotLocked(n, true)
		delete(n.inodes, name)
	} else if newchild != child {
		debugPanicIfNotLocked(n, true)
		n.inodes[name] = newchild
		n.fileinfo.modTime = time.Now()
		child = newchild
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

func (n *treenode) Readdir() (fi []os.FileInfo, err error) {
	n.RLock()
	defer n.RUnlock()
	fi = make([]os.FileInfo, 0, len(n.inodes))
	for _, inode := range n.inodes {
		fi = append(fi, inode.FileInfo())
	}
	return
}

func (n *treenode) Sync() error {
	n.RLock()
	defer n.RUnlock()
	for _, inode := range n.inodes {
		syncer, ok := inode.(syncer)
		if !ok {
			return ErrInvalidOperation
		}
		err := syncer.Sync()
		if err != nil {
			return err
		}
	}
	return nil
}

func (n *treenode) MemorySize() (size int64) {
	n.RLock()
	defer n.RUnlock()
	debugPanicIfNotLocked(n, false)
	for _, inode := range n.inodes {
		size += inode.MemorySize()
	}
	return
}

type fileSystem struct {
	root inode
	fsBackend
	mutex sync.Mutex
	thr   *throttle
}

func (fs *fileSystem) rootnode() inode {
	return fs.root
}

func (fs *fileSystem) throttle() *throttle {
	return fs.thr
}

func (fs *fileSystem) locker() sync.Locker {
	return &fs.mutex
}

// OpenFile is analogous to os.OpenFile().
func (fs *fileSystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return fs.openFile(name, flag, perm)
}

func (fs *fileSystem) openFile(name string, flag int, perm os.FileMode) (*filehandle, error) {
	if flag&os.O_SYNC != 0 {
		return nil, ErrSyncNotSupported
	}
	dirname, name := path.Split(name)
	parent, err := rlookup(fs.root, dirname)
	if err != nil {
		return nil, err
	}
	var readable, writable bool
	switch flag & (os.O_RDWR | os.O_RDONLY | os.O_WRONLY) {
	case os.O_RDWR:
		readable = true
		writable = true
	case os.O_RDONLY:
		readable = true
	case os.O_WRONLY:
		writable = true
	default:
		return nil, fmt.Errorf("invalid flags 0x%x", flag)
	}
	if !writable && parent.IsDir() {
		// A directory can be opened via "foo/", "foo/.", or
		// "foo/..".
		switch name {
		case ".", "":
			return &filehandle{inode: parent}, nil
		case "..":
			return &filehandle{inode: parent.Parent()}, nil
		}
	}
	createMode := flag&os.O_CREATE != 0
	// We always need to take Lock() here, not just RLock(). Even
	// if we know we won't be creating a file, parent might be a
	// lookupnode, which sometimes populates its inodes map during
	// a Child() call.
	parent.Lock()
	defer parent.Unlock()
	n, err := parent.Child(name, nil)
	if err != nil {
		return nil, err
	} else if n == nil {
		if !createMode {
			return nil, os.ErrNotExist
		}
		n, err = parent.Child(name, func(inode) (repl inode, err error) {
			repl, err = parent.FS().newNode(name, perm|0755, time.Now())
			if err != nil {
				return
			}
			repl.SetParent(parent, name)
			return
		})
		if err != nil {
			return nil, err
		} else if n == nil {
			// Parent rejected new child, but returned no error
			return nil, ErrInvalidArgument
		}
	} else if flag&os.O_EXCL != 0 {
		return nil, ErrFileExists
	} else if flag&os.O_TRUNC != 0 {
		if !writable {
			return nil, fmt.Errorf("invalid flag O_TRUNC in read-only mode")
		} else if n.IsDir() {
			return nil, fmt.Errorf("invalid flag O_TRUNC when opening directory")
		} else if err := n.Truncate(0); err != nil {
			return nil, err
		}
	}
	return &filehandle{
		inode:    n,
		append:   flag&os.O_APPEND != 0,
		readable: readable,
		writable: writable,
	}, nil
}

func (fs *fileSystem) Open(name string) (http.File, error) {
	return fs.OpenFile(name, os.O_RDONLY, 0)
}

func (fs *fileSystem) Create(name string) (File, error) {
	return fs.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0)
}

func (fs *fileSystem) Mkdir(name string, perm os.FileMode) error {
	dirname, name := path.Split(name)
	n, err := rlookup(fs.root, dirname)
	if err != nil {
		return err
	}
	n.Lock()
	defer n.Unlock()
	if child, err := n.Child(name, nil); err != nil {
		return err
	} else if child != nil {
		return os.ErrExist
	}

	_, err = n.Child(name, func(inode) (repl inode, err error) {
		repl, err = n.FS().newNode(name, perm|os.ModeDir, time.Now())
		if err != nil {
			return
		}
		repl.SetParent(n, name)
		return
	})
	return err
}

func (fs *fileSystem) Stat(name string) (os.FileInfo, error) {
	node, err := rlookup(fs.root, name)
	if err != nil {
		return nil, err
	}
	return node.FileInfo(), nil
}

func (fs *fileSystem) Rename(oldname, newname string) error {
	olddir, oldname := path.Split(oldname)
	if oldname == "" || oldname == "." || oldname == ".." {
		return ErrInvalidArgument
	}
	olddirf, err := fs.openFile(olddir+".", os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("%q: %s", olddir, err)
	}
	defer olddirf.Close()

	newdir, newname := path.Split(newname)
	if newname == "." || newname == ".." {
		return ErrInvalidArgument
	} else if newname == "" {
		// Rename("a/b", "c/") means Rename("a/b", "c/b")
		newname = oldname
	}
	newdirf, err := fs.openFile(newdir+".", os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("%q: %s", newdir, err)
	}
	defer newdirf.Close()

	// TODO: If the nearest common ancestor ("nca") of olddirf and
	// newdirf is on a different filesystem than fs, we should
	// call nca.FS().Rename() instead of proceeding. Until then
	// it's awkward for filesystems to implement their own Rename
	// methods effectively: the only one that runs is the one on
	// the root FileSystem exposed to the caller (webdav, fuse,
	// etc).

	// When acquiring locks on multiple inodes, avoid deadlock by
	// locking the entire containing filesystem first.
	cfs := olddirf.inode.FS()
	cfs.locker().Lock()
	defer cfs.locker().Unlock()

	if cfs != newdirf.inode.FS() {
		// Moving inodes across filesystems is not (yet)
		// supported. Locking inodes from different
		// filesystems could deadlock, so we must error out
		// now.
		return ErrInvalidArgument
	}

	// To ensure we can test reliably whether we're about to move
	// a directory into itself, lock all potential common
	// ancestors of olddir and newdir.
	needLock := []sync.Locker{}
	for _, node := range []inode{olddirf.inode, newdirf.inode} {
		needLock = append(needLock, node)
		for node.Parent() != node && node.Parent().FS() == node.FS() {
			node = node.Parent()
			needLock = append(needLock, node)
		}
	}
	locked := map[sync.Locker]bool{}
	for i := len(needLock) - 1; i >= 0; i-- {
		if n := needLock[i]; !locked[n] {
			n.Lock()
			defer n.Unlock()
			locked[n] = true
		}
	}

	_, err = olddirf.inode.Child(oldname, func(oldinode inode) (inode, error) {
		if oldinode == nil {
			return oldinode, os.ErrNotExist
		}
		if locked[oldinode] {
			// oldinode cannot become a descendant of itself.
			return oldinode, ErrInvalidArgument
		}
		if oldinode.FS() != cfs && newdirf.inode != olddirf.inode {
			// moving a mount point to a different parent
			// is not (yet) supported.
			return oldinode, ErrInvalidArgument
		}
		accepted, err := newdirf.inode.Child(newname, func(existing inode) (inode, error) {
			if existing != nil && existing.IsDir() {
				return existing, ErrIsDirectory
			}
			return oldinode, nil
		})
		if err != nil {
			// Leave oldinode in olddir.
			return oldinode, err
		}
		accepted.SetParent(newdirf.inode, newname)
		return nil, nil
	})
	return err
}

func (fs *fileSystem) Remove(name string) error {
	return fs.remove(strings.TrimRight(name, "/"), false)
}

func (fs *fileSystem) RemoveAll(name string) error {
	err := fs.remove(strings.TrimRight(name, "/"), true)
	if os.IsNotExist(err) {
		// "If the path does not exist, RemoveAll returns
		// nil." (see "os" pkg)
		err = nil
	}
	return err
}

func (fs *fileSystem) remove(name string, recursive bool) error {
	dirname, name := path.Split(name)
	if name == "" || name == "." || name == ".." {
		return ErrInvalidArgument
	}
	dir, err := rlookup(fs.root, dirname)
	if err != nil {
		return err
	}
	dir.Lock()
	defer dir.Unlock()
	_, err = dir.Child(name, func(node inode) (inode, error) {
		if node == nil {
			return nil, os.ErrNotExist
		}
		if !recursive && node.IsDir() && node.Size() > 0 {
			return node, ErrDirectoryNotEmpty
		}
		return nil, nil
	})
	return err
}

func (fs *fileSystem) Sync() error {
	if syncer, ok := fs.root.(syncer); ok {
		return syncer.Sync()
	}
	return ErrInvalidOperation
}

func (fs *fileSystem) Flush(string, bool) error {
	log.Printf("TODO: flush fileSystem")
	return ErrInvalidOperation
}

func (fs *fileSystem) MemorySize() int64 {
	return fs.root.MemorySize()
}

// rlookup (recursive lookup) returns the inode for the file/directory
// with the given name (which may contain "/" separators). If no such
// file/directory exists, the returned node is nil.
func rlookup(start inode, path string) (node inode, err error) {
	node = start
	for _, name := range strings.Split(path, "/") {
		if node.IsDir() {
			if name == "." || name == "" {
				continue
			}
			if name == ".." {
				node = node.Parent()
				continue
			}
		}
		node, err = func() (inode, error) {
			node.RLock()
			defer node.RUnlock()
			return node.Child(name, nil)
		}()
		if node == nil || err != nil {
			break
		}
	}
	if node == nil && err == nil {
		err = os.ErrNotExist
	}
	return
}

func permittedName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.Contains(name, "/")
}
