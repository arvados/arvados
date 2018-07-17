// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var maxBlockSize = 1 << 26

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

type collectionFileSystem struct {
	fileSystem
	uuid string
}

// FileSystem returns a CollectionFileSystem for the collection.
func (c *Collection) FileSystem(client apiClient, kc keepClient) (CollectionFileSystem, error) {
	var modTime time.Time
	if c.ModifiedAt == nil {
		modTime = time.Now()
	} else {
		modTime = *c.ModifiedAt
	}
	fs := &collectionFileSystem{
		uuid: c.UUID,
		fileSystem: fileSystem{
			fsBackend: keepBackend{apiClient: client, keepClient: kc},
		},
	}
	root := &dirnode{
		fs: fs,
		treenode: treenode{
			fileinfo: fileinfo{
				name:    ".",
				mode:    os.ModeDir | 0755,
				modTime: modTime,
			},
			inodes: make(map[string]inode),
		},
	}
	root.SetParent(root, ".")
	if err := root.loadManifest(c.ManifestText); err != nil {
		return nil, err
	}
	backdateTree(root, modTime)
	fs.root = root
	return fs, nil
}

func backdateTree(n inode, modTime time.Time) {
	switch n := n.(type) {
	case *filenode:
		n.fileinfo.modTime = modTime
	case *dirnode:
		n.fileinfo.modTime = modTime
		for _, n := range n.inodes {
			backdateTree(n, modTime)
		}
	}
}

func (fs *collectionFileSystem) newNode(name string, perm os.FileMode, modTime time.Time) (node inode, err error) {
	if name == "" || name == "." || name == ".." {
		return nil, ErrInvalidArgument
	}
	if perm.IsDir() {
		return &dirnode{
			fs: fs,
			treenode: treenode{
				fileinfo: fileinfo{
					name:    name,
					mode:    perm | os.ModeDir,
					modTime: modTime,
				},
				inodes: make(map[string]inode),
			},
		}, nil
	} else {
		return &filenode{
			fs: fs,
			fileinfo: fileinfo{
				name:    name,
				mode:    perm & ^os.ModeDir,
				modTime: modTime,
			},
		}, nil
	}
}

func (fs *collectionFileSystem) Sync() error {
	log.Printf("cfs.Sync()")
	if fs.uuid == "" {
		return nil
	}
	txt, err := fs.MarshalManifest(".")
	if err != nil {
		log.Printf("WARNING: (collectionFileSystem)Sync() failed: %s", err)
		return err
	}
	coll := &Collection{
		UUID:         fs.uuid,
		ManifestText: txt,
	}
	err = fs.RequestAndDecode(nil, "PUT", "arvados/v1/collections/"+fs.uuid, fs.UpdateBody(coll), map[string]interface{}{"select": []string{"uuid"}})
	if err != nil {
		log.Printf("WARNING: (collectionFileSystem)Sync() failed: %s", err)
	}
	return err
}

func (fs *collectionFileSystem) MarshalManifest(prefix string) (string, error) {
	fs.fileSystem.root.Lock()
	defer fs.fileSystem.root.Unlock()
	return fs.fileSystem.root.(*dirnode).marshalManifest(prefix)
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

// filenode implements inode.
type filenode struct {
	parent   inode
	fs       FileSystem
	fileinfo fileinfo
	segments []segment
	// number of times `segments` has changed in a
	// way that might invalidate a filenodePtr
	repacked int64
	memsize  int64 // bytes in memSegments
	sync.RWMutex
	nullnode
}

// caller must have lock
func (fn *filenode) appendSegment(e segment) {
	fn.segments = append(fn.segments, e)
	fn.fileinfo.size += int64(e.Len())
}

func (fn *filenode) SetParent(p inode, name string) {
	fn.Lock()
	defer fn.Unlock()
	fn.parent = p
	fn.fileinfo.name = name
}

func (fn *filenode) Parent() inode {
	fn.RLock()
	defer fn.RUnlock()
	return fn.parent
}

func (fn *filenode) FS() FileSystem {
	return fn.fs
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
		locator, _, err := fn.FS().PutB(seg.buf)
		if err != nil {
			// TODO: stall (or return errors from)
			// subsequent writes until flushing
			// starts to succeed
			continue
		}
		fn.memsize -= int64(seg.Len())
		fn.segments[idx] = storedSegment{
			kc:      fn.FS(),
			locator: locator,
			size:    seg.Len(),
			offset:  0,
			length:  seg.Len(),
		}
	}
}

type dirnode struct {
	fs *collectionFileSystem
	treenode
}

func (dn *dirnode) FS() FileSystem {
	return dn.fs
}

func (dn *dirnode) Child(name string, replace func(inode) (inode, error)) (inode, error) {
	if dn == dn.fs.rootnode() && name == ".arvados#collection" {
		gn := &getternode{Getter: func() ([]byte, error) {
			var coll Collection
			var err error
			coll.ManifestText, err = dn.fs.MarshalManifest(".")
			if err != nil {
				return nil, err
			}
			data, err := json.Marshal(&coll)
			if err == nil {
				data = append(data, '\n')
			}
			return data, err
		}}
		gn.SetParent(dn, name)
		return gn, nil
	}
	return dn.treenode.Child(name, replace)
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
		locator, _, err := dn.fs.PutB(block)
		if err != nil {
			return err
		}
		off := 0
		for _, sb := range sbs {
			data := sb.fn.segments[sb.idx].(*memSegment).buf
			sb.fn.segments[sb.idx] = storedSegment{
				kc:      dn.fs,
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
					kc:      dn.fs,
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
	var node inode = dn
	names := strings.Split(path, "/")
	basename := names[len(names)-1]
	if !permittedName(basename) {
		err = fmt.Errorf("invalid file part %q in path %q", basename, path)
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
			node = node.Parent()
			continue
		}
		node, err = node.Child(name, func(child inode) (inode, error) {
			if child == nil {
				child, err := node.FS().newNode(name, 0755|os.ModeDir, node.Parent().FileInfo().ModTime())
				if err != nil {
					return nil, err
				}
				child.SetParent(node, name)
				return child, nil
			} else if !child.IsDir() {
				return child, ErrFileExists
			} else {
				return child, nil
			}
		})
		if err != nil {
			return
		}
	}
	_, err = node.Child(basename, func(child inode) (inode, error) {
		switch child := child.(type) {
		case nil:
			child, err = node.FS().newNode(basename, 0755, node.FileInfo().ModTime())
			if err != nil {
				return nil, err
			}
			child.SetParent(node, basename)
			fn = child.(*filenode)
			return child, nil
		case *filenode:
			fn = child
			return child, nil
		case *dirnode:
			return child, ErrIsDirectory
		default:
			return child, ErrInvalidArgument
		}
	})
	return
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
	kc      fsBackend
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
