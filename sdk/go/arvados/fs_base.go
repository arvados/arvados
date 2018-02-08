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
	ErrPermission        = os.ErrPermission
)

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
	Sync() error
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
	Child(name string, replace func(inode) inode) inode

	sync.Locker
	RLock()
	RUnlock()
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

func (*nullnode) Child(name string, replace func(inode) inode) inode {
	return nil
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

func (n *treenode) Child(name string, replace func(inode) inode) (child inode) {
	// TODO: special treatment for "", ".", ".."
	child = n.inodes[name]
	if replace != nil {
		newchild := replace(child)
		if newchild == nil {
			delete(n.inodes, name)
		} else if newchild != child {
			n.inodes[name] = newchild
			n.fileinfo.modTime = time.Now()
			child = newchild
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

func (n *treenode) Readdir() (fi []os.FileInfo, err error) {
	n.RLock()
	defer n.RUnlock()
	fi = make([]os.FileInfo, 0, len(n.inodes))
	for _, inode := range n.inodes {
		fi = append(fi, inode.FileInfo())
	}
	return
}

type fileSystem struct {
	root inode
	fsBackend
	mutex sync.Mutex
}

func (fs *fileSystem) rootnode() inode {
	return fs.root
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
	parent := rlookup(fs.root, dirname)
	if parent == nil {
		return nil, os.ErrNotExist
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
	if createMode {
		parent.Lock()
		defer parent.Unlock()
	} else {
		parent.RLock()
		defer parent.RUnlock()
	}
	n := parent.Child(name, nil)
	if n == nil {
		if !createMode {
			return nil, os.ErrNotExist
		}
		var err error
		n = parent.Child(name, func(inode) inode {
			n, err = parent.FS().newNode(name, perm|0755, time.Now())
			n.SetParent(parent, name)
			return n
		})
		if err != nil {
			return nil, err
		} else if n == nil {
			// parent rejected new child
			return nil, ErrInvalidOperation
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

func (fs *fileSystem) Mkdir(name string, perm os.FileMode) (err error) {
	dirname, name := path.Split(name)
	n := rlookup(fs.root, dirname)
	if n == nil {
		return os.ErrNotExist
	}
	n.Lock()
	defer n.Unlock()
	if n.Child(name, nil) != nil {
		return os.ErrExist
	}
	child := n.Child(name, func(inode) (child inode) {
		child, err = n.FS().newNode(name, perm|os.ModeDir, time.Now())
		child.SetParent(n, name)
		return
	})
	if err != nil {
		return err
	} else if child == nil {
		return ErrInvalidArgument
	}
	return nil
}

func (fs *fileSystem) Stat(name string) (fi os.FileInfo, err error) {
	node := rlookup(fs.root, name)
	if node == nil {
		err = os.ErrNotExist
	} else {
		fi = node.FileInfo()
	}
	return
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
	// the root filesystem exposed to the caller (webdav, fuse,
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

	// Return ErrInvalidOperation if olddirf.inode doesn't even
	// bother calling our "remove oldname entry" replacer func.
	err = ErrInvalidArgument
	olddirf.inode.Child(oldname, func(oldinode inode) inode {
		err = nil
		if oldinode == nil {
			err = os.ErrNotExist
			return nil
		}
		if locked[oldinode] {
			// oldinode cannot become a descendant of itself.
			err = ErrInvalidArgument
			return oldinode
		}
		if oldinode.FS() != cfs && newdirf.inode != olddirf.inode {
			// moving a mount point to a different parent
			// is not (yet) supported.
			err = ErrInvalidArgument
			return oldinode
		}
		accepted := newdirf.inode.Child(newname, func(existing inode) inode {
			if existing != nil && existing.IsDir() {
				err = ErrIsDirectory
				return existing
			}
			return oldinode
		})
		if accepted != oldinode {
			if err == nil {
				// newdirf didn't accept oldinode.
				err = ErrInvalidArgument
			}
			// Leave oldinode in olddir.
			return oldinode
		}
		accepted.SetParent(newdirf.inode, newname)
		return nil
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

func (fs *fileSystem) remove(name string, recursive bool) (err error) {
	dirname, name := path.Split(name)
	if name == "" || name == "." || name == ".." {
		return ErrInvalidArgument
	}
	dir := rlookup(fs.root, dirname)
	if dir == nil {
		return os.ErrNotExist
	}
	dir.Lock()
	defer dir.Unlock()
	dir.Child(name, func(node inode) inode {
		if node == nil {
			err = os.ErrNotExist
			return nil
		}
		if !recursive && node.IsDir() && node.Size() > 0 {
			err = ErrDirectoryNotEmpty
			return node
		}
		return nil
	})
	return err
}

func (fs *fileSystem) Sync() error {
	log.Printf("TODO: sync fileSystem")
	return ErrInvalidOperation
}

// rlookup (recursive lookup) returns the inode for the file/directory
// with the given name (which may contain "/" separators). If no such
// file/directory exists, the returned node is nil.
func rlookup(start inode, path string) (node inode) {
	node = start
	for _, name := range strings.Split(path, "/") {
		if node == nil {
			break
		}
		if node.IsDir() {
			if name == "." || name == "" {
				continue
			}
			if name == ".." {
				node = node.Parent()
				continue
			}
		}
		node = func() inode {
			node.RLock()
			defer node.RUnlock()
			return node.Child(name, nil)
		}()
	}
	return
}
