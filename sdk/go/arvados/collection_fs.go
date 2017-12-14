// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
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

	maxBlockSize = 1 << 26
)

// A File is an *os.File-like interface for reading and writing files
// in a CollectionFileSystem.
type File interface {
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
	Size() int64
	Readdir(int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
	Truncate(int64) error
}

type keepClient interface {
	ReadAt(locator string, p []byte, off int) (int, error)
	PutB(p []byte) (string, int, error)
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

// A FileSystem is an http.Filesystem plus Stat() and support for
// opening writable files. All methods are safe to call from multiple
// goroutines.
type FileSystem interface {
	http.FileSystem

	inode

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
}

// A CollectionFileSystem is a FileSystem that can be serialized as a
// manifest and stored as a collection.
type CollectionFileSystem interface {
	FileSystem

	// Flush all file data to Keep and return a snapshot of the
	// filesystem suitable for saving as (Collection)ManifestText.
	// Prefix (normally ".") is a top level directory, effectively
	// prepended to all paths in the returned manifest.
	MarshalManifest(prefix string) (string, error)
}

type fileSystem struct {
	inode
}

type collectionFileSystem struct {
	fileSystem
}

func (fs collectionFileSystem) MarshalManifest(prefix string) (string, error) {
	fs.fileSystem.inode.Lock()
	defer fs.fileSystem.inode.Unlock()
	return fs.fileSystem.inode.(*dirnode).marshalManifest(prefix)
}

// OpenFile is analogous to os.OpenFile().
func (fs *fileSystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return fs.openFile(name, flag, perm)
}

func (fs *fileSystem) openFile(name string, flag int, perm os.FileMode) (*filehandle, error) {
	var dn inode = fs.inode
	if flag&os.O_SYNC != 0 {
		return nil, ErrSyncNotSupported
	}
	dirname, name := path.Split(name)
	parent := rlookup(dn, dirname)
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
			var dn *dirnode
			switch parent := parent.(type) {
			case *dirnode:
				dn = parent
			case *collectionFileSystem:
				dn = parent.inode.(*dirnode)
			default:
				err = ErrInvalidArgument
				return nil
			}
			if perm.IsDir() {
				n, err = dn.newDirnode(dn, name, perm|0755, time.Now())
			} else {
				n, err = dn.newFilenode(dn, name, perm|0755, time.Now())
			}
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
		} else if fn, ok := n.(*filenode); !ok {
			return nil, fmt.Errorf("invalid flag O_TRUNC when opening directory")
		} else {
			fn.Truncate(0)
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
	n := rlookup(fs.inode, dirname)
	if n == nil {
		return os.ErrNotExist
	}
	n.Lock()
	defer n.Unlock()
	if n.Child(name, nil) != nil {
		return os.ErrExist
	}
	dn, ok := n.(*dirnode)
	if !ok {
		return ErrInvalidArgument
	}
	child := n.Child(name, func(inode) (child inode) {
		child, err = dn.newDirnode(dn, name, perm, time.Now())
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
	node := rlookup(fs.inode, name)
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

	// When acquiring locks on multiple nodes, all common
	// ancestors must be locked first in order to avoid
	// deadlock. This is assured by locking the path from root to
	// newdir, then locking the path from root to olddir, skipping
	// any already-locked nodes.
	needLock := []sync.Locker{}
	for _, f := range []*filehandle{olddirf, newdirf} {
		node := f.inode
		needLock = append(needLock, node)
		for node.Parent() != node {
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

	if _, ok := newdirf.inode.(*dirnode); !ok {
		return ErrInvalidOperation
	}

	err = nil
	olddirf.inode.Child(oldname, func(oldinode inode) inode {
		if oldinode == nil {
			err = os.ErrNotExist
			return nil
		}
		newdirf.inode.Child(newname, func(existing inode) inode {
			if existing != nil && existing.IsDir() {
				err = ErrIsDirectory
				return existing
			}
			return oldinode
		})
		if err != nil {
			return oldinode
		}
		switch n := oldinode.(type) {
		case *dirnode:
			n.parent = newdirf.inode
		case *filenode:
			n.parent = newdirf.inode.(*dirnode)
		default:
			panic(fmt.Sprintf("bad inode type %T", n))
		}
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
	dir := rlookup(fs, dirname)
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

type inode interface {
	Parent() inode
	Read([]byte, filenodePtr) (int, filenodePtr, error)
	Write([]byte, filenodePtr) (int, filenodePtr, error)
	Truncate(int64) error
	IsDir() bool
	Readdir() []os.FileInfo
	Size() int64
	FileInfo() os.FileInfo
	// Caller must have lock (or rlock if func is nil)
	Child(string, func(inode) inode) inode
	sync.Locker
	RLock()
	RUnlock()
}

// filenode implements inode.
type filenode struct {
	fileinfo fileinfo
	parent   *dirnode
	segments []segment
	// number of times `segments` has changed in a
	// way that might invalidate a filenodePtr
	repacked int64
	memsize  int64 // bytes in memSegments
	sync.RWMutex
	nullnode
}

// filenodePtr is an offset into a file that is (usually) efficient to
// seek to. Specifically, if filenode.repacked==filenodePtr.repacked
// then
// filenode.segments[filenodePtr.segmentIdx][filenodePtr.segmentOff]
// corresponds to file offset filenodePtr.off. Otherwise, it is
// necessary to reexamine len(filenode.segments[0]) etc. to find the
// correct segment and offset.
type filenodePtr struct {
	off        int64
	segmentIdx int
	segmentOff int
	repacked   int64
}

// seek returns a ptr that is consistent with both startPtr.off and
// the current state of fn. The caller must already hold fn.RLock() or
// fn.Lock().
//
// If startPtr is beyond EOF, ptr.segment* will indicate precisely
// EOF.
//
// After seeking:
//
//     ptr.segmentIdx == len(filenode.segments) // i.e., at EOF
//     ||
//     filenode.segments[ptr.segmentIdx].Len() > ptr.segmentOff
func (fn *filenode) seek(startPtr filenodePtr) (ptr filenodePtr) {
	ptr = startPtr
	if ptr.off < 0 {
		// meaningless anyway
		return
	} else if ptr.off >= fn.fileinfo.size {
		ptr.segmentIdx = len(fn.segments)
		ptr.segmentOff = 0
		ptr.repacked = fn.repacked
		return
	} else if ptr.repacked == fn.repacked {
		// segmentIdx and segmentOff accurately reflect
		// ptr.off, but might have fallen off the end of a
		// segment
		if ptr.segmentOff >= fn.segments[ptr.segmentIdx].Len() {
			ptr.segmentIdx++
			ptr.segmentOff = 0
		}
		return
	}
	defer func() {
		ptr.repacked = fn.repacked
	}()
	if ptr.off >= fn.fileinfo.size {
		ptr.segmentIdx, ptr.segmentOff = len(fn.segments), 0
		return
	}
	// Recompute segmentIdx and segmentOff.  We have already
	// established fn.fileinfo.size > ptr.off >= 0, so we don't
	// have to deal with edge cases here.
	var off int64
	for ptr.segmentIdx, ptr.segmentOff = 0, 0; off < ptr.off; ptr.segmentIdx++ {
		// This would panic (index out of range) if
		// fn.fileinfo.size were larger than
		// sum(fn.segments[i].Len()) -- but that can't happen
		// because we have ensured fn.fileinfo.size is always
		// accurate.
		segLen := int64(fn.segments[ptr.segmentIdx].Len())
		if off+segLen > ptr.off {
			ptr.segmentOff = int(ptr.off - off)
			break
		}
		off += segLen
	}
	return
}

// caller must have lock
func (fn *filenode) appendSegment(e segment) {
	fn.segments = append(fn.segments, e)
	fn.fileinfo.size += int64(e.Len())
}

func (fn *filenode) Parent() inode {
	fn.RLock()
	defer fn.RUnlock()
	return fn.parent
}

// Read reads file data from a single segment, starting at startPtr,
// into p. startPtr is assumed not to be up-to-date. Caller must have
// RLock or Lock.
func (fn *filenode) Read(p []byte, startPtr filenodePtr) (n int, ptr filenodePtr, err error) {
	ptr = fn.seek(startPtr)
	if ptr.off < 0 {
		err = ErrNegativeOffset
		return
	}
	if ptr.segmentIdx >= len(fn.segments) {
		err = io.EOF
		return
	}
	n, err = fn.segments[ptr.segmentIdx].ReadAt(p, int64(ptr.segmentOff))
	if n > 0 {
		ptr.off += int64(n)
		ptr.segmentOff += n
		if ptr.segmentOff == fn.segments[ptr.segmentIdx].Len() {
			ptr.segmentIdx++
			ptr.segmentOff = 0
			if ptr.segmentIdx < len(fn.segments) && err == io.EOF {
				err = nil
			}
		}
	}
	return
}

func (fn *filenode) Size() int64 {
	fn.RLock()
	defer fn.RUnlock()
	return fn.fileinfo.Size()
}

func (fn *filenode) FileInfo() os.FileInfo {
	fn.RLock()
	defer fn.RUnlock()
	return fn.fileinfo
}

func (fn *filenode) Truncate(size int64) error {
	fn.Lock()
	defer fn.Unlock()
	return fn.truncate(size)
}

func (fn *filenode) truncate(size int64) error {
	if size == fn.fileinfo.size {
		return nil
	}
	fn.repacked++
	if size < fn.fileinfo.size {
		ptr := fn.seek(filenodePtr{off: size})
		for i := ptr.segmentIdx; i < len(fn.segments); i++ {
			if seg, ok := fn.segments[i].(*memSegment); ok {
				fn.memsize -= int64(seg.Len())
			}
		}
		if ptr.segmentOff == 0 {
			fn.segments = fn.segments[:ptr.segmentIdx]
		} else {
			fn.segments = fn.segments[:ptr.segmentIdx+1]
			switch seg := fn.segments[ptr.segmentIdx].(type) {
			case *memSegment:
				seg.Truncate(ptr.segmentOff)
				fn.memsize += int64(seg.Len())
			default:
				fn.segments[ptr.segmentIdx] = seg.Slice(0, ptr.segmentOff)
			}
		}
		fn.fileinfo.size = size
		return nil
	}
	for size > fn.fileinfo.size {
		grow := size - fn.fileinfo.size
		var seg *memSegment
		var ok bool
		if len(fn.segments) == 0 {
			seg = &memSegment{}
			fn.segments = append(fn.segments, seg)
		} else if seg, ok = fn.segments[len(fn.segments)-1].(*memSegment); !ok || seg.Len() >= maxBlockSize {
			seg = &memSegment{}
			fn.segments = append(fn.segments, seg)
		}
		if maxgrow := int64(maxBlockSize - seg.Len()); maxgrow < grow {
			grow = maxgrow
		}
		seg.Truncate(seg.Len() + int(grow))
		fn.fileinfo.size += grow
		fn.memsize += grow
	}
	return nil
}

// Write writes data from p to the file, starting at startPtr,
// extending the file size if necessary. Caller must have Lock.
func (fn *filenode) Write(p []byte, startPtr filenodePtr) (n int, ptr filenodePtr, err error) {
	if startPtr.off > fn.fileinfo.size {
		if err = fn.truncate(startPtr.off); err != nil {
			return 0, startPtr, err
		}
	}
	ptr = fn.seek(startPtr)
	if ptr.off < 0 {
		err = ErrNegativeOffset
		return
	}
	for len(p) > 0 && err == nil {
		cando := p
		if len(cando) > maxBlockSize {
			cando = cando[:maxBlockSize]
		}
		// Rearrange/grow fn.segments (and shrink cando if
		// needed) such that cando can be copied to
		// fn.segments[ptr.segmentIdx] at offset
		// ptr.segmentOff.
		cur := ptr.segmentIdx
		prev := ptr.segmentIdx - 1
		var curWritable bool
		if cur < len(fn.segments) {
			_, curWritable = fn.segments[cur].(*memSegment)
		}
		var prevAppendable bool
		if prev >= 0 && fn.segments[prev].Len() < maxBlockSize {
			_, prevAppendable = fn.segments[prev].(*memSegment)
		}
		if ptr.segmentOff > 0 && !curWritable {
			// Split a non-writable block.
			if max := fn.segments[cur].Len() - ptr.segmentOff; max <= len(cando) {
				// Truncate cur, and insert a new
				// segment after it.
				cando = cando[:max]
				fn.segments = append(fn.segments, nil)
				copy(fn.segments[cur+1:], fn.segments[cur:])
			} else {
				// Split cur into two copies, truncate
				// the one on the left, shift the one
				// on the right, and insert a new
				// segment between them.
				fn.segments = append(fn.segments, nil, nil)
				copy(fn.segments[cur+2:], fn.segments[cur:])
				fn.segments[cur+2] = fn.segments[cur+2].Slice(ptr.segmentOff+len(cando), -1)
			}
			cur++
			prev++
			seg := &memSegment{}
			seg.Truncate(len(cando))
			fn.memsize += int64(len(cando))
			fn.segments[cur] = seg
			fn.segments[prev] = fn.segments[prev].Slice(0, ptr.segmentOff)
			ptr.segmentIdx++
			ptr.segmentOff = 0
			fn.repacked++
			ptr.repacked++
		} else if curWritable {
			if fit := int(fn.segments[cur].Len()) - ptr.segmentOff; fit < len(cando) {
				cando = cando[:fit]
			}
		} else {
			if prevAppendable {
				// Shrink cando if needed to fit in
				// prev segment.
				if cangrow := maxBlockSize - fn.segments[prev].Len(); cangrow < len(cando) {
					cando = cando[:cangrow]
				}
			}

			if cur == len(fn.segments) {
				// ptr is at EOF, filesize is changing.
				fn.fileinfo.size += int64(len(cando))
			} else if el := fn.segments[cur].Len(); el <= len(cando) {
				// cando is long enough that we won't
				// need cur any more. shrink cando to
				// be exactly as long as cur
				// (otherwise we'd accidentally shift
				// the effective position of all
				// segments after cur).
				cando = cando[:el]
				copy(fn.segments[cur:], fn.segments[cur+1:])
				fn.segments = fn.segments[:len(fn.segments)-1]
			} else {
				// shrink cur by the same #bytes we're growing prev
				fn.segments[cur] = fn.segments[cur].Slice(len(cando), -1)
			}

			if prevAppendable {
				// Grow prev.
				ptr.segmentIdx--
				ptr.segmentOff = fn.segments[prev].Len()
				fn.segments[prev].(*memSegment).Truncate(ptr.segmentOff + len(cando))
				fn.memsize += int64(len(cando))
				ptr.repacked++
				fn.repacked++
			} else {
				// Insert a segment between prev and
				// cur, and advance prev/cur.
				fn.segments = append(fn.segments, nil)
				if cur < len(fn.segments) {
					copy(fn.segments[cur+1:], fn.segments[cur:])
					ptr.repacked++
					fn.repacked++
				} else {
					// appending a new segment does
					// not invalidate any ptrs
				}
				seg := &memSegment{}
				seg.Truncate(len(cando))
				fn.memsize += int64(len(cando))
				fn.segments[cur] = seg
				cur++
				prev++
			}
		}

		// Finally we can copy bytes from cando to the current segment.
		fn.segments[ptr.segmentIdx].(*memSegment).WriteAt(cando, ptr.segmentOff)
		n += len(cando)
		p = p[len(cando):]

		ptr.off += int64(len(cando))
		ptr.segmentOff += len(cando)
		if ptr.segmentOff >= maxBlockSize {
			fn.pruneMemSegments()
		}
		if fn.segments[ptr.segmentIdx].Len() == ptr.segmentOff {
			ptr.segmentOff = 0
			ptr.segmentIdx++
		}

		fn.fileinfo.modTime = time.Now()
	}
	return
}

// Write some data out to disk to reduce memory use. Caller must have
// write lock.
func (fn *filenode) pruneMemSegments() {
	// TODO: async (don't hold Lock() while waiting for Keep)
	// TODO: share code with (*dirnode)sync()
	// TODO: pack/flush small blocks too, when fragmented
	for idx, seg := range fn.segments {
		seg, ok := seg.(*memSegment)
		if !ok || seg.Len() < maxBlockSize {
			continue
		}
		locator, _, err := fn.parent.kc.PutB(seg.buf)
		if err != nil {
			// TODO: stall (or return errors from)
			// subsequent writes until flushing
			// starts to succeed
			continue
		}
		fn.memsize -= int64(seg.Len())
		fn.segments[idx] = storedSegment{
			kc:      fn.parent.kc,
			locator: locator,
			size:    seg.Len(),
			offset:  0,
			length:  seg.Len(),
		}
	}
}

// FileSystem returns a CollectionFileSystem for the collection.
func (c *Collection) FileSystem(client *Client, kc keepClient) (CollectionFileSystem, error) {
	var modTime time.Time
	if c.ModifiedAt == nil {
		modTime = time.Now()
	} else {
		modTime = *c.ModifiedAt
	}
	dn := &dirnode{
		client: client,
		kc:     kc,
		treenode: treenode{
			fileinfo: fileinfo{
				name:    ".",
				mode:    os.ModeDir | 0755,
				modTime: modTime,
			},
			parent: nil,
			inodes: make(map[string]inode),
		},
	}
	dn.parent = dn
	fs := &collectionFileSystem{fileSystem: fileSystem{inode: dn}}
	if err := dn.loadManifest(c.ManifestText); err != nil {
		return nil, err
	}
	return fs, nil
}

type filehandle struct {
	inode
	ptr        filenodePtr
	append     bool
	readable   bool
	writable   bool
	unreaddirs []os.FileInfo
}

func (f *filehandle) Read(p []byte) (n int, err error) {
	if !f.readable {
		return 0, ErrWriteOnlyMode
	}
	f.inode.RLock()
	defer f.inode.RUnlock()
	n, f.ptr, err = f.inode.Read(p, f.ptr)
	return
}

func (f *filehandle) Seek(off int64, whence int) (pos int64, err error) {
	size := f.inode.Size()
	ptr := f.ptr
	switch whence {
	case io.SeekStart:
		ptr.off = off
	case io.SeekCurrent:
		ptr.off += off
	case io.SeekEnd:
		ptr.off = size + off
	}
	if ptr.off < 0 {
		return f.ptr.off, ErrNegativeOffset
	}
	if ptr.off != f.ptr.off {
		f.ptr = ptr
		// force filenode to recompute f.ptr fields on next
		// use
		f.ptr.repacked = -1
	}
	return f.ptr.off, nil
}

func (f *filehandle) Truncate(size int64) error {
	return f.inode.Truncate(size)
}

func (f *filehandle) Write(p []byte) (n int, err error) {
	if !f.writable {
		return 0, ErrReadOnlyFile
	}
	f.inode.Lock()
	defer f.inode.Unlock()
	if fn, ok := f.inode.(*filenode); ok && f.append {
		f.ptr = filenodePtr{
			off:        fn.fileinfo.size,
			segmentIdx: len(fn.segments),
			segmentOff: 0,
			repacked:   fn.repacked,
		}
	}
	n, f.ptr, err = f.inode.Write(p, f.ptr)
	return
}

func (f *filehandle) Readdir(count int) ([]os.FileInfo, error) {
	if !f.inode.IsDir() {
		return nil, ErrInvalidOperation
	}
	if count <= 0 {
		return f.inode.Readdir(), nil
	}
	if f.unreaddirs == nil {
		f.unreaddirs = f.inode.Readdir()
	}
	if len(f.unreaddirs) == 0 {
		return nil, io.EOF
	}
	if count > len(f.unreaddirs) {
		count = len(f.unreaddirs)
	}
	ret := f.unreaddirs[:count]
	f.unreaddirs = f.unreaddirs[count:]
	return ret, nil
}

func (f *filehandle) Stat() (os.FileInfo, error) {
	return f.inode.FileInfo(), nil
}

func (f *filehandle) Close() error {
	return nil
}

type dirnode struct {
	treenode
	client *Client
	kc     keepClient
}

// sync flushes in-memory data (for all files in the tree rooted at
// dn) to persistent storage. Caller must hold dn.Lock().
func (dn *dirnode) sync() error {
	type shortBlock struct {
		fn  *filenode
		idx int
	}
	var pending []shortBlock
	var pendingLen int

	flush := func(sbs []shortBlock) error {
		if len(sbs) == 0 {
			return nil
		}
		block := make([]byte, 0, maxBlockSize)
		for _, sb := range sbs {
			block = append(block, sb.fn.segments[sb.idx].(*memSegment).buf...)
		}
		locator, _, err := dn.kc.PutB(block)
		if err != nil {
			return err
		}
		off := 0
		for _, sb := range sbs {
			data := sb.fn.segments[sb.idx].(*memSegment).buf
			sb.fn.segments[sb.idx] = storedSegment{
				kc:      dn.kc,
				locator: locator,
				size:    len(block),
				offset:  off,
				length:  len(data),
			}
			off += len(data)
			sb.fn.memsize -= int64(len(data))
		}
		return nil
	}

	names := make([]string, 0, len(dn.inodes))
	for name := range dn.inodes {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fn, ok := dn.inodes[name].(*filenode)
		if !ok {
			continue
		}
		fn.Lock()
		defer fn.Unlock()
		for idx, seg := range fn.segments {
			seg, ok := seg.(*memSegment)
			if !ok {
				continue
			}
			if seg.Len() > maxBlockSize/2 {
				if err := flush([]shortBlock{{fn, idx}}); err != nil {
					return err
				}
				continue
			}
			if pendingLen+seg.Len() > maxBlockSize {
				if err := flush(pending); err != nil {
					return err
				}
				pending = nil
				pendingLen = 0
			}
			pending = append(pending, shortBlock{fn, idx})
			pendingLen += seg.Len()
		}
	}
	return flush(pending)
}

// caller must have read lock.
func (dn *dirnode) marshalManifest(prefix string) (string, error) {
	var streamLen int64
	type filepart struct {
		name   string
		offset int64
		length int64
	}
	var fileparts []filepart
	var subdirs string
	var blocks []string

	if err := dn.sync(); err != nil {
		return "", err
	}

	names := make([]string, 0, len(dn.inodes))
	for name, node := range dn.inodes {
		names = append(names, name)
		node.Lock()
		defer node.Unlock()
	}
	sort.Strings(names)

	for _, name := range names {
		switch node := dn.inodes[name].(type) {
		case *dirnode:
			subdir, err := node.marshalManifest(prefix + "/" + name)
			if err != nil {
				return "", err
			}
			subdirs = subdirs + subdir
		case *filenode:
			if len(node.segments) == 0 {
				fileparts = append(fileparts, filepart{name: name})
				break
			}
			for _, seg := range node.segments {
				switch seg := seg.(type) {
				case storedSegment:
					if len(blocks) > 0 && blocks[len(blocks)-1] == seg.locator {
						streamLen -= int64(seg.size)
					} else {
						blocks = append(blocks, seg.locator)
					}
					next := filepart{
						name:   name,
						offset: streamLen + int64(seg.offset),
						length: int64(seg.length),
					}
					if prev := len(fileparts) - 1; prev >= 0 &&
						fileparts[prev].name == name &&
						fileparts[prev].offset+fileparts[prev].length == next.offset {
						fileparts[prev].length += next.length
					} else {
						fileparts = append(fileparts, next)
					}
					streamLen += int64(seg.size)
				default:
					// This can't happen: we
					// haven't unlocked since
					// calling sync().
					panic(fmt.Sprintf("can't marshal segment type %T", seg))
				}
			}
		default:
			panic(fmt.Sprintf("can't marshal inode type %T", node))
		}
	}
	var filetokens []string
	for _, s := range fileparts {
		filetokens = append(filetokens, fmt.Sprintf("%d:%d:%s", s.offset, s.length, manifestEscape(s.name)))
	}
	if len(filetokens) == 0 {
		return subdirs, nil
	} else if len(blocks) == 0 {
		blocks = []string{"d41d8cd98f00b204e9800998ecf8427e+0"}
	}
	return manifestEscape(prefix) + " " + strings.Join(blocks, " ") + " " + strings.Join(filetokens, " ") + "\n" + subdirs, nil
}

func (dn *dirnode) loadManifest(txt string) error {
	var dirname string
	streams := strings.Split(txt, "\n")
	if streams[len(streams)-1] != "" {
		return fmt.Errorf("line %d: no trailing newline", len(streams))
	}
	streams = streams[:len(streams)-1]
	segments := []storedSegment{}
	for i, stream := range streams {
		lineno := i + 1
		var anyFileTokens bool
		var pos int64
		var segIdx int
		segments = segments[:0]
		for i, token := range strings.Split(stream, " ") {
			if i == 0 {
				dirname = manifestUnescape(token)
				continue
			}
			if !strings.Contains(token, ":") {
				if anyFileTokens {
					return fmt.Errorf("line %d: bad file segment %q", lineno, token)
				}
				toks := strings.SplitN(token, "+", 3)
				if len(toks) < 2 {
					return fmt.Errorf("line %d: bad locator %q", lineno, token)
				}
				length, err := strconv.ParseInt(toks[1], 10, 32)
				if err != nil || length < 0 {
					return fmt.Errorf("line %d: bad locator %q", lineno, token)
				}
				segments = append(segments, storedSegment{
					locator: token,
					size:    int(length),
					offset:  0,
					length:  int(length),
				})
				continue
			} else if len(segments) == 0 {
				return fmt.Errorf("line %d: bad locator %q", lineno, token)
			}

			toks := strings.SplitN(token, ":", 3)
			if len(toks) != 3 {
				return fmt.Errorf("line %d: bad file segment %q", lineno, token)
			}
			anyFileTokens = true

			offset, err := strconv.ParseInt(toks[0], 10, 64)
			if err != nil || offset < 0 {
				return fmt.Errorf("line %d: bad file segment %q", lineno, token)
			}
			length, err := strconv.ParseInt(toks[1], 10, 64)
			if err != nil || length < 0 {
				return fmt.Errorf("line %d: bad file segment %q", lineno, token)
			}
			name := dirname + "/" + manifestUnescape(toks[2])
			fnode, err := dn.createFileAndParents(name)
			if err != nil {
				return fmt.Errorf("line %d: cannot use path %q: %s", lineno, name, err)
			}
			// Map the stream offset/range coordinates to
			// block/offset/range coordinates and add
			// corresponding storedSegments to the filenode
			if pos > offset {
				// Can't continue where we left off.
				// TODO: binary search instead of
				// rewinding all the way (but this
				// situation might be rare anyway)
				segIdx, pos = 0, 0
			}
			for next := int64(0); segIdx < len(segments); segIdx++ {
				seg := segments[segIdx]
				next = pos + int64(seg.Len())
				if next <= offset || seg.Len() == 0 {
					pos = next
					continue
				}
				if pos >= offset+length {
					break
				}
				var blkOff int
				if pos < offset {
					blkOff = int(offset - pos)
				}
				blkLen := seg.Len() - blkOff
				if pos+int64(blkOff+blkLen) > offset+length {
					blkLen = int(offset + length - pos - int64(blkOff))
				}
				fnode.appendSegment(storedSegment{
					kc:      dn.kc,
					locator: seg.locator,
					size:    seg.size,
					offset:  blkOff,
					length:  blkLen,
				})
				if next > offset+length {
					break
				} else {
					pos = next
				}
			}
			if segIdx == len(segments) && pos < offset+length {
				return fmt.Errorf("line %d: invalid segment in %d-byte stream: %q", lineno, pos, token)
			}
		}
		if !anyFileTokens {
			return fmt.Errorf("line %d: no file segments", lineno)
		} else if len(segments) == 0 {
			return fmt.Errorf("line %d: no locators", lineno)
		} else if dirname == "" {
			return fmt.Errorf("line %d: no stream name", lineno)
		}
	}
	return nil
}

// only safe to call from loadManifest -- no locking
func (dn *dirnode) createFileAndParents(path string) (fn *filenode, err error) {
	node := dn
	names := strings.Split(path, "/")
	basename := names[len(names)-1]
	if basename == "" || basename == "." || basename == ".." {
		err = fmt.Errorf("invalid filename")
		return
	}
	for _, name := range names[:len(names)-1] {
		switch name {
		case "", ".":
			continue
		case "..":
			if node == dn {
				// can't be sure parent will be a *dirnode
				return nil, ErrInvalidArgument
			}
			node = node.Parent().(*dirnode)
			continue
		}
		node.Child(name, func(child inode) inode {
			switch child.(type) {
			case nil:
				node, err = dn.newDirnode(node, name, 0755|os.ModeDir, node.Parent().FileInfo().ModTime())
				child = node
			case *dirnode:
				node = child.(*dirnode)
			case *filenode:
				err = ErrFileExists
			default:
				err = ErrInvalidOperation
			}
			return child
		})
		if err != nil {
			return
		}
	}
	node.Child(basename, func(child inode) inode {
		switch child := child.(type) {
		case nil:
			fn, err = dn.newFilenode(node, basename, 0755, node.FileInfo().ModTime())
			return fn
		case *filenode:
			fn = child
			return child
		case *dirnode:
			err = ErrIsDirectory
			return child
		default:
			err = ErrInvalidOperation
			return child
		}
	})
	return
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

// Caller must have lock, and must have already ensured
// Children(name,nil) is nil.
func (dn *dirnode) newDirnode(parent *dirnode, name string, perm os.FileMode, modTime time.Time) (node *dirnode, err error) {
	if name == "" || name == "." || name == ".." {
		return nil, ErrInvalidArgument
	}
	return &dirnode{
		client: dn.client,
		kc:     dn.kc,
		treenode: treenode{
			parent: parent,
			fileinfo: fileinfo{
				name:    name,
				mode:    perm | os.ModeDir,
				modTime: modTime,
			},
			inodes: make(map[string]inode),
		},
	}, nil
}

func (dn *dirnode) newFilenode(parent *dirnode, name string, perm os.FileMode, modTime time.Time) (node *filenode, err error) {
	if name == "" || name == "." || name == ".." {
		return nil, ErrInvalidArgument
	}
	return &filenode{
		parent: parent,
		fileinfo: fileinfo{
			name:    name,
			mode:    perm & ^os.ModeDir,
			modTime: modTime,
		},
	}, nil
}

type segment interface {
	io.ReaderAt
	Len() int
	// Return a new segment with a subsection of the data from this
	// one. length<0 means length=Len()-off.
	Slice(off int, length int) segment
}

type memSegment struct {
	buf []byte
}

func (me *memSegment) Len() int {
	return len(me.buf)
}

func (me *memSegment) Slice(off, length int) segment {
	if length < 0 {
		length = len(me.buf) - off
	}
	buf := make([]byte, length)
	copy(buf, me.buf[off:])
	return &memSegment{buf: buf}
}

func (me *memSegment) Truncate(n int) {
	if n > cap(me.buf) {
		newsize := 1024
		for newsize < n {
			newsize = newsize << 2
		}
		newbuf := make([]byte, n, newsize)
		copy(newbuf, me.buf)
		me.buf = newbuf
	} else {
		// Zero unused part when shrinking, in case we grow
		// and start using it again later.
		for i := n; i < len(me.buf); i++ {
			me.buf[i] = 0
		}
	}
	me.buf = me.buf[:n]
}

func (me *memSegment) WriteAt(p []byte, off int) {
	if off+len(p) > len(me.buf) {
		panic("overflowed segment")
	}
	copy(me.buf[off:], p)
}

func (me *memSegment) ReadAt(p []byte, off int64) (n int, err error) {
	if off > int64(me.Len()) {
		err = io.EOF
		return
	}
	n = copy(p, me.buf[int(off):])
	if n < len(p) {
		err = io.EOF
	}
	return
}

type storedSegment struct {
	kc      keepClient
	locator string
	size    int // size of stored block (also encoded in locator)
	offset  int // position of segment within the stored block
	length  int // bytes in this segment (offset + length <= size)
}

func (se storedSegment) Len() int {
	return se.length
}

func (se storedSegment) Slice(n, size int) segment {
	se.offset += n
	se.length -= n
	if size >= 0 && se.length > size {
		se.length = size
	}
	return se
}

func (se storedSegment) ReadAt(p []byte, off int64) (n int, err error) {
	if off > int64(se.length) {
		return 0, io.EOF
	}
	maxlen := se.length - int(off)
	if len(p) > maxlen {
		p = p[:maxlen]
		n, err = se.kc.ReadAt(se.locator, p, int(off)+se.offset)
		if err == nil {
			err = io.EOF
		}
		return
	}
	return se.kc.ReadAt(se.locator, p, int(off)+se.offset)
}

func canonicalName(name string) string {
	name = path.Clean("/" + name)
	if name == "/" || name == "./" {
		name = "."
	} else if strings.HasPrefix(name, "/") {
		name = "." + name
	}
	return name
}

var manifestEscapeSeq = regexp.MustCompile(`\\([0-7]{3}|\\)`)

func manifestUnescapeFunc(seq string) string {
	if seq == `\\` {
		return `\`
	}
	i, err := strconv.ParseUint(seq[1:], 8, 8)
	if err != nil {
		// Invalid escape sequence: can't unescape.
		return seq
	}
	return string([]byte{byte(i)})
}

func manifestUnescape(s string) string {
	return manifestEscapeSeq.ReplaceAllStringFunc(s, manifestUnescapeFunc)
}

var manifestEscapedChar = regexp.MustCompile(`[\000-\040:\s\\]`)

func manifestEscapeFunc(seq string) string {
	return fmt.Sprintf("\\%03o", byte(seq[0]))
}

func manifestEscape(s string) string {
	return manifestEscapedChar.ReplaceAllStringFunc(s, manifestEscapeFunc)
}
