// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrReadOnlyFile     = errors.New("read-only file")
	ErrNegativeOffset   = errors.New("cannot seek to negative offset")
	ErrFileExists       = errors.New("file exists")
	ErrInvalidOperation = errors.New("invalid operation")
	ErrPermission       = os.ErrPermission
)

const maxBlockSize = 1 << 26

type File interface {
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
	Size() int64
	Readdir(int) ([]os.FileInfo, error)
	Stat() (os.FileInfo, error)
}

type keepClient interface {
	ReadAt(locator string, p []byte, off int64) (int, error)
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

func (fi fileinfo) Stat() os.FileInfo {
	return fi
}

// A CollectionFileSystem is an http.Filesystem plus Stat() and
// support for opening writable files.
type CollectionFileSystem interface {
	http.FileSystem
	Stat(name string) (os.FileInfo, error)
	Create(name string) (File, error)
	OpenFile(name string, flag int, perm os.FileMode) (File, error)
}

type fileSystem struct {
	dirnode
}

func (fs *fileSystem) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
	return fs.dirnode.OpenFile(path.Clean(name), flag, perm)
}

func (fs *fileSystem) Open(name string) (http.File, error) {
	return fs.dirnode.OpenFile(path.Clean(name), os.O_RDONLY, 0)
}

func (fs *fileSystem) Create(name string) (File, error) {
	return fs.dirnode.OpenFile(path.Clean(name), os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0)
}

func (fs *fileSystem) Stat(name string) (os.FileInfo, error) {
	f, err := fs.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}

type inode interface {
	os.FileInfo
	OpenFile(string, int, os.FileMode) (*file, error)
	Parent() inode
	Read([]byte, filenodePtr) (int, filenodePtr, error)
	Write([]byte, filenodePtr) (int, filenodePtr, error)
	Readdir() []os.FileInfo
	Stat() os.FileInfo
	sync.Locker
	RLock()
	RUnlock()
}

// filenode implements inode.
type filenode struct {
	fileinfo
	parent   *dirnode
	extents  []extent
	repacked int64 // number of times anything in []extents has changed len
	sync.RWMutex
}

// filenodePtr is an offset into a file that is (usually) efficient to
// seek to. Specifically, if filenode.repacked==filenodePtr.repacked
// then filenode.extents[filenodePtr.extentIdx][filenodePtr.extentOff]
// corresponds to file offset filenodePtr.off. Otherwise, it is
// necessary to reexamine len(filenode.extents[0]) etc. to find the
// correct extent and offset.
type filenodePtr struct {
	off       int64
	extentIdx int
	extentOff int
	repacked  int64
}

// seek returns a ptr that is consistent with both startPtr.off and
// the current state of fn. The caller must already hold fn.RLock() or
// fn.Lock().
//
// If startPtr points beyond the end of the file, ptr will point to
// exactly the end of the file.
//
// After seeking:
//
//     ptr.extentIdx == len(filenode.extents) // i.e., at EOF
//     ||
//     filenode.extents[ptr.extentIdx].Len() >= ptr.extentOff
func (fn *filenode) seek(startPtr filenodePtr) (ptr filenodePtr) {
	ptr = startPtr
	if ptr.off < 0 {
		// meaningless anyway
		return
	} else if ptr.off >= fn.fileinfo.size {
		ptr.off = fn.fileinfo.size
		ptr.extentIdx = len(fn.extents)
		ptr.extentOff = 0
		ptr.repacked = fn.repacked
		return
	} else if ptr.repacked == fn.repacked {
		// extentIdx and extentOff accurately reflect ptr.off,
		// but might have fallen off the end of an extent
		if int64(ptr.extentOff) >= fn.extents[ptr.extentIdx].Len() {
			ptr.extentIdx++
			ptr.extentOff = 0
		}
		return
	}
	defer func() {
		ptr.repacked = fn.repacked
	}()
	if ptr.off >= fn.fileinfo.size {
		ptr.extentIdx, ptr.extentOff = len(fn.extents), 0
		return
	}
	// Recompute extentIdx and extentOff.  We have already
	// established fn.fileinfo.size > ptr.off >= 0, so we don't
	// have to deal with edge cases here.
	var off int64
	for ptr.extentIdx, ptr.extentOff = 0, 0; off < ptr.off; ptr.extentIdx++ {
		// This would panic (index out of range) if
		// fn.fileinfo.size were larger than
		// sum(fn.extents[i].Len()) -- but that can't happen
		// because we have ensured fn.fileinfo.size is always
		// accurate.
		extLen := fn.extents[ptr.extentIdx].Len()
		if off+extLen > ptr.off {
			ptr.extentOff = int(ptr.off - off)
			break
		}
		off += extLen
	}
	return
}

func (fn *filenode) appendExtent(e extent) {
	fn.Lock()
	defer fn.Unlock()
	fn.extents = append(fn.extents, e)
	fn.fileinfo.size += e.Len()
	fn.repacked++
}

func (fn *filenode) OpenFile(string, int, os.FileMode) (*file, error) {
	return nil, os.ErrNotExist
}

func (fn *filenode) Parent() inode {
	return fn.parent
}

func (fn *filenode) Readdir() []os.FileInfo {
	return nil
}

func (fn *filenode) Read(p []byte, startPtr filenodePtr) (n int, ptr filenodePtr, err error) {
	fn.RLock()
	defer fn.RUnlock()
	ptr = fn.seek(startPtr)
	if ptr.off < 0 {
		err = ErrNegativeOffset
		return
	}
	if ptr.extentIdx >= len(fn.extents) {
		err = io.EOF
		return
	}
	n, err = fn.extents[ptr.extentIdx].ReadAt(p, int64(ptr.extentOff))
	if n > 0 {
		ptr.off += int64(n)
		ptr.extentOff += n
		if int64(ptr.extentOff) == fn.extents[ptr.extentIdx].Len() {
			ptr.extentIdx++
			ptr.extentOff = 0
			if ptr.extentIdx < len(fn.extents) && err == io.EOF {
				err = nil
			}
		}
	}
	return
}

func (fn *filenode) Write(p []byte, startPtr filenodePtr) (n int, ptr filenodePtr, err error) {
	fn.Lock()
	defer fn.Unlock()
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
		// Rearrange/grow fn.extents (and shrink cando if
		// needed) such that cando can be copied to
		// fn.extents[ptr.extentIdx] at offset ptr.extentOff.
		cur := ptr.extentIdx
		prev := ptr.extentIdx - 1
		var curWritable bool
		if cur < len(fn.extents) {
			_, curWritable = fn.extents[cur].(writableExtent)
		}
		var prevAppendable bool
		if prev >= 0 && fn.extents[prev].Len() < int64(maxBlockSize) {
			_, prevAppendable = fn.extents[prev].(writableExtent)
		}
		if ptr.extentOff > 0 {
			if !curWritable {
				// Split a non-writable block.
				if max := int(fn.extents[cur].Len()) - ptr.extentOff; max <= len(cando) {
					cando = cando[:max]
					fn.extents = append(fn.extents, nil)
					copy(fn.extents[cur+1:], fn.extents[cur:])
				} else {
					fn.extents = append(fn.extents, nil, nil)
					copy(fn.extents[cur+2:], fn.extents[cur:])
					fn.extents[cur+2] = fn.extents[cur+2].Slice(ptr.extentOff+len(cando), -1)
				}
				cur++
				prev++
				e := &memExtent{}
				e.Truncate(len(cando))
				fn.extents[cur] = e
				fn.extents[prev] = fn.extents[prev].Slice(0, ptr.extentOff)
				ptr.extentIdx++
				ptr.extentOff = 0
				fn.repacked++
				ptr.repacked++
			}
		} else if len(fn.extents) == 0 {
			// File has no extents yet.
			e := &memExtent{}
			e.Truncate(len(cando))
			fn.fileinfo.size += e.Len()
			fn.extents = append(fn.extents, e)
		} else if curWritable {
			if fit := int(fn.extents[cur].Len()) - ptr.extentOff; fit < len(cando) {
				cando = cando[:fit]
			}
		} else {
			if prevAppendable {
				// Grow prev.
				if cangrow := int(maxBlockSize - fn.extents[prev].Len()); cangrow < len(cando) {
					cando = cando[:cangrow]
				}
				ptr.extentIdx--
				ptr.extentOff = int(fn.extents[prev].Len())
				fn.extents[prev].(*memExtent).Truncate(ptr.extentOff + len(cando))
			} else {
				// Insert an extent between prev and cur. It will be the new prev.
				fn.extents = append(fn.extents, nil)
				copy(fn.extents[cur+1:], fn.extents[cur:])
				e := &memExtent{}
				e.Truncate(len(cando))
				fn.extents[cur] = e
				cur++
				prev++
			}

			if cur == len(fn.extents) {
				// There is no cur.
			} else if el := int(fn.extents[cur].Len()); el <= len(cando) {
				// Drop cur.
				cando = cando[:el]
				copy(fn.extents[cur:], fn.extents[cur+1:])
				fn.extents = fn.extents[:len(fn.extents)-1]
			} else {
				// Shrink cur.
				fn.extents[cur] = fn.extents[cur].Slice(len(cando), -1)
			}

			ptr.repacked++
			fn.repacked++
		}

		// Finally we can copy bytes from cando to the current extent.
		fn.extents[ptr.extentIdx].(writableExtent).WriteAt(cando, ptr.extentOff)
		n += len(cando)
		p = p[len(cando):]

		ptr.off += int64(len(cando))
		ptr.extentOff += len(cando)
		if fn.extents[ptr.extentIdx].Len() == int64(ptr.extentOff) {
			ptr.extentOff = 0
			ptr.extentIdx++
		}
	}
	return
}

// FileSystem returns a CollectionFileSystem for the collection.
func (c *Collection) FileSystem(client *Client, kc keepClient) CollectionFileSystem {
	fs := &fileSystem{dirnode: dirnode{
		cache:    &keepBlockCache{kc: kc},
		client:   client,
		kc:       kc,
		fileinfo: fileinfo{name: ".", mode: os.ModeDir | 0755},
		parent:   nil,
		inodes:   make(map[string]inode),
	}}
	fs.dirnode.parent = &fs.dirnode
	fs.dirnode.loadManifest(c.ManifestText)
	return fs
}

type file struct {
	inode
	ptr        filenodePtr
	append     bool
	writable   bool
	unreaddirs []os.FileInfo
}

func (f *file) Read(p []byte) (n int, err error) {
	n, f.ptr, err = f.inode.Read(p, f.ptr)
	return
}

func (f *file) Seek(off int64, whence int) (pos int64, err error) {
	size := f.inode.Size()
	ptr := f.ptr
	switch whence {
	case os.SEEK_SET:
		ptr.off = off
	case os.SEEK_CUR:
		ptr.off += off
	case os.SEEK_END:
		ptr.off = size + off
	}
	if ptr.off < 0 {
		return f.ptr.off, ErrNegativeOffset
	}
	if ptr.off > size {
		ptr.off = size
	}
	if ptr.off != f.ptr.off {
		f.ptr = ptr
		// force filenode to recompute f.ptr fields on next
		// use
		f.ptr.repacked = -1
	}
	return f.ptr.off, nil
}

func (f *file) Write(p []byte) (n int, err error) {
	if !f.writable {
		return 0, ErrReadOnlyFile
	}
	n, f.ptr, err = f.inode.Write(p, f.ptr)
	return
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
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

func (f *file) Stat() (os.FileInfo, error) {
	return f.inode, nil
}

func (f *file) Close() error {
	// FIXME: flush
	return nil
}

func (f *file) OpenFile(name string, flag int, perm os.FileMode) (*file, error) {
	return f.inode.OpenFile(name, flag, perm)
}

type dirnode struct {
	fileinfo
	parent *dirnode
	client *Client
	kc     keepClient
	cache  blockCache
	inodes map[string]inode
	sync.RWMutex
}

func (dn *dirnode) loadManifest(txt string) {
	// FIXME: faster
	var dirname string
	for _, stream := range strings.Split(txt, "\n") {
		var extents []storedExtent
		for i, token := range strings.Split(stream, " ") {
			if i == 0 {
				dirname = manifestUnescape(token)
				continue
			}
			if !strings.Contains(token, ":") {
				toks := strings.SplitN(token, "+", 3)
				if len(toks) < 2 {
					// FIXME: broken
					continue
				}
				length, err := strconv.ParseInt(toks[1], 10, 32)
				if err != nil || length < 0 {
					// FIXME: broken
					continue
				}
				extents = append(extents, storedExtent{
					locator: token,
					offset:  0,
					length:  int(length),
				})
				continue
			}
			toks := strings.Split(token, ":")
			if len(toks) != 3 {
				// FIXME: broken manifest
				continue
			}
			offset, err := strconv.ParseInt(toks[0], 10, 64)
			if err != nil || offset < 0 {
				// FIXME: broken manifest
				continue
			}
			length, err := strconv.ParseInt(toks[1], 10, 64)
			if err != nil || length < 0 {
				// FIXME: broken manifest
				continue
			}
			name := path.Clean(dirname + "/" + manifestUnescape(toks[2]))
			dn.makeParentDirs(name)
			f, err := dn.OpenFile(name, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0700)
			if err != nil {
				// FIXME: broken
				continue
			}
			if f.inode.Stat().IsDir() {
				f.Close()
				// FIXME: broken manifest
				continue
			}
			var pos int64
			for _, e := range extents {
				if pos+e.Len() < offset {
					pos += e.Len()
					continue
				}
				if pos > offset+length {
					break
				}
				var blkOff int
				if pos < offset {
					blkOff = int(offset - pos)
				}
				blkLen := int(e.Len()) - blkOff
				if pos+int64(blkOff+blkLen) > offset+length {
					blkLen = int(offset + length - pos - int64(blkOff))
				}
				f.inode.(*filenode).appendExtent(storedExtent{
					cache:   dn.cache,
					locator: e.locator,
					offset:  blkOff,
					length:  blkLen,
				})
				pos += e.Len()
			}
			f.Close()
		}
	}
}

func (dn *dirnode) makeParentDirs(name string) {
	names := strings.Split(name, "/")
	for _, name := range names[:len(names)-1] {
		dn.Lock()
		defer dn.Unlock()
		if n, ok := dn.inodes[name]; !ok {
			n := &dirnode{
				parent: dn,
				client: dn.client,
				kc:     dn.kc,
				fileinfo: fileinfo{
					name: name,
					mode: os.ModeDir | 0755,
				},
			}
			if dn.inodes == nil {
				dn.inodes = make(map[string]inode)
			}
			dn.inodes[name] = n
			dn.fileinfo.size++
			dn = n
		} else if n, ok := n.(*dirnode); ok {
			dn = n
		} else {
			// fail
			return
		}
	}
}

func (dn *dirnode) Parent() inode {
	return dn.parent
}

func (dn *dirnode) Readdir() (fi []os.FileInfo) {
	dn.RLock()
	defer dn.RUnlock()
	fi = make([]os.FileInfo, 0, len(dn.inodes))
	for _, inode := range dn.inodes {
		fi = append(fi, inode.Stat())
	}
	return
}

func (dn *dirnode) Read(p []byte, ptr filenodePtr) (int, filenodePtr, error) {
	return 0, ptr, ErrInvalidOperation
}

func (dn *dirnode) Write(p []byte, ptr filenodePtr) (int, filenodePtr, error) {
	return 0, ptr, ErrInvalidOperation
}

func (dn *dirnode) OpenFile(name string, flag int, perm os.FileMode) (*file, error) {
	name = strings.TrimSuffix(name, "/")
	if name == "." || name == "" {
		return &file{inode: dn}, nil
	}
	if dirname, name := path.Split(name); dirname != "" {
		// OpenFile("foo/bar/baz") =>
		// OpenFile("foo/bar").OpenFile("baz") (or
		// ErrNotExist, if foo/bar is a file)
		f, err := dn.OpenFile(dirname, os.O_RDONLY, 0)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if dn, ok := f.inode.(*dirnode); ok {
			return dn.OpenFile(name, flag, perm)
		} else {
			return nil, os.ErrNotExist
		}
	}
	dn.Lock()
	defer dn.Unlock()
	if name == ".." {
		return &file{inode: dn.parent}, nil
	}
	n, ok := dn.inodes[name]
	if !ok {
		if flag&os.O_CREATE == 0 {
			return nil, os.ErrNotExist
		}
		n = &filenode{
			parent: dn,
			fileinfo: fileinfo{
				name: name,
				mode: 0755,
			},
		}
		if dn.inodes == nil {
			dn.inodes = make(map[string]inode)
		}
		dn.inodes[name] = n
		dn.fileinfo.size++
	} else if flag&os.O_EXCL != 0 {
		return nil, ErrFileExists
	}
	return &file{
		inode:    n,
		append:   flag&os.O_APPEND != 0,
		writable: flag&(os.O_WRONLY|os.O_RDWR) != 0,
	}, nil
}

type extent interface {
	io.ReaderAt
	Len() int64
	Slice(int, int) extent
}

type writableExtent interface {
	extent
	WriteAt(p []byte, off int)
	Truncate(n int)
}

type memExtent struct {
	buf []byte
}

func (me *memExtent) Len() int64 {
	return int64(len(me.buf))
}

func (me *memExtent) Slice(n, size int) extent {
	if size < 0 {
		size = len(me.buf) - n
	}
	return &memExtent{buf: me.buf[n : n+size]}
}

func (me *memExtent) Truncate(n int) {
	if n > cap(me.buf) {
		newsize := 1024
		for newsize < n {
			newsize = newsize << 2
		}
		newbuf := make([]byte, n, newsize)
		copy(newbuf, me.buf)
		me.buf = newbuf
	}
	me.buf = me.buf[:n]
}

func (me *memExtent) WriteAt(p []byte, off int) {
	if off+len(p) > len(me.buf) {
		panic("overflowed extent")
	}
	copy(me.buf[off:], p)
}

func (me *memExtent) ReadAt(p []byte, off int64) (n int, err error) {
	if off > me.Len() {
		err = io.EOF
		return
	}
	n = copy(p, me.buf[int(off):])
	if n < len(p) {
		err = io.EOF
	}
	return
}

type storedExtent struct {
	cache   blockCache
	locator string
	offset  int
	length  int
}

func (se storedExtent) Len() int64 {
	return int64(se.length)
}

func (se storedExtent) Slice(n, size int) extent {
	se.offset += n
	se.length -= n
	if size >= 0 && se.length > size {
		se.length = size
	}
	return se
}

func (se storedExtent) ReadAt(p []byte, off int64) (n int, err error) {
	maxlen := int(int64(se.length) - off)
	if len(p) > maxlen {
		p = p[:maxlen]
		n, err = se.cache.ReadAt(se.locator, p, off+int64(se.offset))
		if err == nil {
			err = io.EOF
		}
		return
	}
	return se.cache.ReadAt(se.locator, p, off+int64(se.offset))
}

type blockCache interface {
	ReadAt(locator string, p []byte, off int64) (n int, err error)
}

type keepBlockCache struct {
	kc keepClient
}

var scratch = make([]byte, 2<<26)

func (kbc *keepBlockCache) ReadAt(locator string, p []byte, off int64) (int, error) {
	return kbc.kc.ReadAt(locator, p, off)
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

var manifestEscapeSeq = regexp.MustCompile(`\\([0-9]{3}|\\)`)

func manifestUnescapeSeq(seq string) string {
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
	return manifestEscapeSeq.ReplaceAllStringFunc(s, manifestUnescapeSeq)
}
