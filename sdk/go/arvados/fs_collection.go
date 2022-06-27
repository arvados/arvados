// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	maxBlockSize      = 1 << 26
	concurrentWriters = 4 // max goroutines writing to Keep in background and during flush()
)

// A CollectionFileSystem is a FileSystem that can be serialized as a
// manifest and stored as a collection.
type CollectionFileSystem interface {
	FileSystem

	// Flush all file data to Keep and return a snapshot of the
	// filesystem suitable for saving as (Collection)ManifestText.
	// Prefix (normally ".") is a top level directory, effectively
	// prepended to all paths in the returned manifest.
	MarshalManifest(prefix string) (string, error)

	// Total data bytes in all files.
	Size() int64
}

type collectionFileSystem struct {
	fileSystem
	uuid           string
	savedPDH       atomic.Value
	replicas       int
	storageClasses []string
	// guessSignatureTTL tracks a lower bound for the server's
	// configured BlobSigningTTL. The guess is initially zero, and
	// increases when we come across a signature with an expiry
	// time further in the future than the previous guess.
	//
	// When the guessed TTL is much smaller than the real TTL,
	// preemptive signature refresh is delayed or missed entirely,
	// which is OK.
	guessSignatureTTL time.Duration
	holdCheckChanges  time.Time
	lockCheckChanges  sync.Mutex
}

// FileSystem returns a CollectionFileSystem for the collection.
func (c *Collection) FileSystem(client apiClient, kc keepClient) (CollectionFileSystem, error) {
	modTime := c.ModifiedAt
	if modTime.IsZero() {
		modTime = time.Now()
	}
	fs := &collectionFileSystem{
		uuid:           c.UUID,
		storageClasses: c.StorageClassesDesired,
		fileSystem: fileSystem{
			fsBackend: keepBackend{apiClient: client, keepClient: kc},
			thr:       newThrottle(concurrentWriters),
		},
	}
	fs.savedPDH.Store(c.PortableDataHash)
	if r := c.ReplicationDesired; r != nil {
		fs.replicas = *r
	}
	root := &dirnode{
		fs: fs,
		treenode: treenode{
			fileinfo: fileinfo{
				name:    ".",
				mode:    os.ModeDir | 0755,
				modTime: modTime,
				sys:     func() interface{} { return c },
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

// caller must have lock (or guarantee no concurrent accesses somehow)
func eachNode(n inode, ffunc func(*filenode), dfunc func(*dirnode)) {
	switch n := n.(type) {
	case *filenode:
		if ffunc != nil {
			ffunc(n)
		}
	case *dirnode:
		if dfunc != nil {
			dfunc(n)
		}
		for _, n := range n.inodes {
			eachNode(n, ffunc, dfunc)
		}
	}
}

// caller must have lock (or guarantee no concurrent accesses somehow)
func backdateTree(n inode, modTime time.Time) {
	eachNode(n, func(fn *filenode) {
		fn.fileinfo.modTime = modTime
	}, func(dn *dirnode) {
		dn.fileinfo.modTime = modTime
	})
}

// Approximate portion of signature TTL remaining, usually between 0
// and 1, or negative if some signatures have expired.
func (fs *collectionFileSystem) signatureTimeLeft() (float64, time.Duration) {
	var (
		now      = time.Now()
		earliest = now.Add(time.Hour * 24 * 7 * 365)
		latest   time.Time
	)
	fs.fileSystem.root.RLock()
	eachNode(fs.root, func(fn *filenode) {
		fn.Lock()
		defer fn.Unlock()
		for _, seg := range fn.segments {
			seg, ok := seg.(storedSegment)
			if !ok {
				continue
			}
			expiryTime, err := signatureExpiryTime(seg.locator)
			if err != nil {
				continue
			}
			if expiryTime.Before(earliest) {
				earliest = expiryTime
			}
			if expiryTime.After(latest) {
				latest = expiryTime
			}
		}
	}, nil)
	fs.fileSystem.root.RUnlock()

	if latest.IsZero() {
		// No signatures == 100% of TTL remaining.
		return 1, 1
	}

	ttl := latest.Sub(now)
	fs.fileSystem.root.Lock()
	{
		if ttl > fs.guessSignatureTTL {
			// ttl is closer to the real TTL than
			// guessSignatureTTL.
			fs.guessSignatureTTL = ttl
		} else {
			// Use the previous best guess to compute the
			// portion remaining (below, after unlocking
			// mutex).
			ttl = fs.guessSignatureTTL
		}
	}
	fs.fileSystem.root.Unlock()

	return earliest.Sub(now).Seconds() / ttl.Seconds(), ttl
}

func (fs *collectionFileSystem) updateSignatures(newmanifest string) {
	newLoc := map[string]string{}
	for _, tok := range regexp.MustCompile(`\S+`).FindAllString(newmanifest, -1) {
		if mBlkRe.MatchString(tok) {
			newLoc[stripAllHints(tok)] = tok
		}
	}
	fs.fileSystem.root.Lock()
	defer fs.fileSystem.root.Unlock()
	eachNode(fs.root, func(fn *filenode) {
		fn.Lock()
		defer fn.Unlock()
		for idx, seg := range fn.segments {
			seg, ok := seg.(storedSegment)
			if !ok {
				continue
			}
			loc, ok := newLoc[stripAllHints(seg.locator)]
			if !ok {
				continue
			}
			seg.locator = loc
			fn.segments[idx] = seg
		}
	}, nil)
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
	}
	return &filenode{
		fs: fs,
		fileinfo: fileinfo{
			name:    name,
			mode:    perm & ^os.ModeDir,
			modTime: modTime,
		},
	}, nil
}

func (fs *collectionFileSystem) Child(name string, replace func(inode) (inode, error)) (inode, error) {
	return fs.rootnode().Child(name, replace)
}

func (fs *collectionFileSystem) FS() FileSystem {
	return fs
}

func (fs *collectionFileSystem) FileInfo() os.FileInfo {
	return fs.rootnode().FileInfo()
}

func (fs *collectionFileSystem) IsDir() bool {
	return true
}

func (fs *collectionFileSystem) Lock() {
	fs.rootnode().Lock()
}

func (fs *collectionFileSystem) Unlock() {
	fs.rootnode().Unlock()
}

func (fs *collectionFileSystem) RLock() {
	fs.rootnode().RLock()
}

func (fs *collectionFileSystem) RUnlock() {
	fs.rootnode().RUnlock()
}

func (fs *collectionFileSystem) Parent() inode {
	return fs.rootnode().Parent()
}

func (fs *collectionFileSystem) Read(_ []byte, ptr filenodePtr) (int, filenodePtr, error) {
	return 0, ptr, ErrInvalidOperation
}

func (fs *collectionFileSystem) Write(_ []byte, ptr filenodePtr) (int, filenodePtr, error) {
	return 0, ptr, ErrInvalidOperation
}

func (fs *collectionFileSystem) Readdir() ([]os.FileInfo, error) {
	return fs.rootnode().Readdir()
}

func (fs *collectionFileSystem) SetParent(parent inode, name string) {
	fs.rootnode().SetParent(parent, name)
}

func (fs *collectionFileSystem) Truncate(int64) error {
	return ErrInvalidOperation
}

// Check for and incorporate upstream changes -- unless that has
// already been done recently, in which case this func is a no-op.
func (fs *collectionFileSystem) checkChangesOnServer() error {
	if fs.uuid == "" && fs.savedPDH.Load() == "" {
		return nil
	}

	// First try UUID if any, then last known PDH. Stop if all
	// signatures are new enough.
	checkingAll := false
	for _, id := range []string{fs.uuid, fs.savedPDH.Load().(string)} {
		if id == "" {
			continue
		}

		fs.lockCheckChanges.Lock()
		if !checkingAll && fs.holdCheckChanges.After(time.Now()) {
			fs.lockCheckChanges.Unlock()
			return nil
		}
		remain, ttl := fs.signatureTimeLeft()
		if remain > 0.01 && !checkingAll {
			fs.holdCheckChanges = time.Now().Add(ttl / 100)
		}
		fs.lockCheckChanges.Unlock()

		if remain >= 0.5 {
			break
		}
		checkingAll = true
		var coll Collection
		err := fs.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+id, nil, map[string]interface{}{"select": []string{"portable_data_hash", "manifest_text"}})
		if err != nil {
			continue
		}
		fs.updateSignatures(coll.ManifestText)
	}
	return nil
}

// Refresh signature on a single locator, if necessary. Assume caller
// has lock. If an update is needed, and there are any storedSegments
// whose signatures can be updated, start a background task to update
// them asynchronously when the caller releases locks.
func (fs *collectionFileSystem) refreshSignature(locator string) string {
	exp, err := signatureExpiryTime(locator)
	if err != nil || exp.Sub(time.Now()) > time.Minute {
		// Synchronous update is not needed. Start an
		// asynchronous update if needed.
		go fs.checkChangesOnServer()
		return locator
	}
	var manifests string
	for _, id := range []string{fs.uuid, fs.savedPDH.Load().(string)} {
		if id == "" {
			continue
		}
		var coll Collection
		err := fs.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+id, nil, map[string]interface{}{"select": []string{"portable_data_hash", "manifest_text"}})
		if err != nil {
			continue
		}
		manifests += coll.ManifestText
	}
	hash := stripAllHints(locator)
	for _, tok := range regexp.MustCompile(`\S+`).FindAllString(manifests, -1) {
		if mBlkRe.MatchString(tok) {
			if stripAllHints(tok) == hash {
				locator = tok
				break
			}
		}
	}
	go fs.updateSignatures(manifests)
	return locator
}

func (fs *collectionFileSystem) Sync() error {
	err := fs.checkChangesOnServer()
	if err != nil {
		return err
	}
	if fs.uuid == "" {
		return nil
	}
	txt, err := fs.MarshalManifest(".")
	if err != nil {
		return fmt.Errorf("sync failed: %s", err)
	}
	if PortableDataHash(txt) == fs.savedPDH.Load() {
		// No local changes since last save or initial load.
		return nil
	}
	coll := Collection{
		UUID:         fs.uuid,
		ManifestText: txt,
	}

	selectFields := []string{"uuid", "portable_data_hash"}
	fs.lockCheckChanges.Lock()
	remain, _ := fs.signatureTimeLeft()
	fs.lockCheckChanges.Unlock()
	if remain < 0.5 {
		selectFields = append(selectFields, "manifest_text")
	}

	err = fs.RequestAndDecode(&coll, "PUT", "arvados/v1/collections/"+fs.uuid, nil, map[string]interface{}{
		"collection": map[string]string{
			"manifest_text": coll.ManifestText,
		},
		"select": selectFields,
	})
	if err != nil {
		return fmt.Errorf("sync failed: update %s: %s", fs.uuid, err)
	}
	fs.updateSignatures(coll.ManifestText)
	fs.savedPDH.Store(coll.PortableDataHash)
	return nil
}

func (fs *collectionFileSystem) Flush(path string, shortBlocks bool) error {
	node, err := rlookup(fs.fileSystem.root, path)
	if err != nil {
		return err
	}
	dn, ok := node.(*dirnode)
	if !ok {
		return ErrNotADirectory
	}
	dn.Lock()
	defer dn.Unlock()
	names := dn.sortedNames()
	if path != "" {
		// Caller only wants to flush the specified dir,
		// non-recursively.  Drop subdirs from the list of
		// names.
		var filenames []string
		for _, name := range names {
			if _, ok := dn.inodes[name].(*filenode); ok {
				filenames = append(filenames, name)
			}
		}
		names = filenames
	}
	for _, name := range names {
		child := dn.inodes[name]
		child.Lock()
		defer child.Unlock()
	}
	return dn.flush(context.TODO(), names, flushOpts{sync: false, shortBlocks: shortBlocks})
}

func (fs *collectionFileSystem) MemorySize() int64 {
	fs.fileSystem.root.Lock()
	defer fs.fileSystem.root.Unlock()
	return fs.fileSystem.root.(*dirnode).MemorySize()
}

func (fs *collectionFileSystem) MarshalManifest(prefix string) (string, error) {
	fs.fileSystem.root.Lock()
	defer fs.fileSystem.root.Unlock()
	return fs.fileSystem.root.(*dirnode).marshalManifest(context.TODO(), prefix)
}

func (fs *collectionFileSystem) Size() int64 {
	return fs.fileSystem.root.(*dirnode).TreeSize()
}

func (fs *collectionFileSystem) Snapshot() (inode, error) {
	return fs.fileSystem.root.Snapshot()
}

func (fs *collectionFileSystem) Splice(r inode) error {
	return fs.fileSystem.root.Splice(r)
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
	fs       *collectionFileSystem
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
	if ss, ok := fn.segments[ptr.segmentIdx].(storedSegment); ok {
		ss.locator = fn.fs.refreshSignature(ss.locator)
		fn.segments[ptr.segmentIdx] = ss
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
	// TODO: share code with (*dirnode)flush()
	// TODO: pack/flush small blocks too, when fragmented
	for idx, seg := range fn.segments {
		seg, ok := seg.(*memSegment)
		if !ok || seg.Len() < maxBlockSize || seg.flushing != nil {
			continue
		}
		// Setting seg.flushing guarantees seg.buf will not be
		// modified in place: WriteAt and Truncate will
		// allocate a new buf instead, if necessary.
		idx, buf := idx, seg.buf
		done := make(chan struct{})
		seg.flushing = done
		// If lots of background writes are already in
		// progress, block here until one finishes, rather
		// than pile up an unlimited number of buffered writes
		// and network flush operations.
		fn.fs.throttle().Acquire()
		go func() {
			defer close(done)
			resp, err := fn.FS().BlockWrite(context.Background(), BlockWriteOptions{
				Data:           buf,
				Replicas:       fn.fs.replicas,
				StorageClasses: fn.fs.storageClasses,
			})
			fn.fs.throttle().Release()
			fn.Lock()
			defer fn.Unlock()
			if seg.flushing != done {
				// A new seg.buf has been allocated.
				return
			}
			if err != nil {
				// TODO: stall (or return errors from)
				// subsequent writes until flushing
				// starts to succeed.
				return
			}
			if len(fn.segments) <= idx || fn.segments[idx] != seg || len(seg.buf) != len(buf) {
				// Segment has been dropped/moved/resized.
				return
			}
			fn.memsize -= int64(len(buf))
			fn.segments[idx] = storedSegment{
				kc:      fn.FS(),
				locator: resp.Locator,
				size:    len(buf),
				offset:  0,
				length:  len(buf),
			}
		}()
	}
}

// Block until all pending pruneMemSegments/flush work is
// finished. Caller must NOT have lock.
func (fn *filenode) waitPrune() {
	var pending []<-chan struct{}
	fn.Lock()
	for _, seg := range fn.segments {
		if seg, ok := seg.(*memSegment); ok && seg.flushing != nil {
			pending = append(pending, seg.flushing)
		}
	}
	fn.Unlock()
	for _, p := range pending {
		<-p
	}
}

func (fn *filenode) Snapshot() (inode, error) {
	fn.RLock()
	defer fn.RUnlock()
	segments := make([]segment, 0, len(fn.segments))
	for _, seg := range fn.segments {
		segments = append(segments, seg.Slice(0, seg.Len()))
	}
	return &filenode{
		fileinfo: fn.fileinfo,
		segments: segments,
	}, nil
}

func (fn *filenode) Splice(repl inode) error {
	repl, err := repl.Snapshot()
	if err != nil {
		return err
	}
	fn.parent.Lock()
	defer fn.parent.Unlock()
	fn.Lock()
	defer fn.Unlock()
	_, err = fn.parent.Child(fn.fileinfo.name, func(inode) (inode, error) { return repl, nil })
	if err != nil {
		return err
	}
	switch repl := repl.(type) {
	case *dirnode:
		repl.parent = fn.parent
		repl.fileinfo.name = fn.fileinfo.name
		repl.setTreeFS(fn.fs)
	case *filenode:
		repl.parent = fn.parent
		repl.fileinfo.name = fn.fileinfo.name
		repl.fs = fn.fs
	default:
		return fmt.Errorf("cannot splice snapshot containing %T: %w", repl, ErrInvalidArgument)
	}
	return nil
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
			coll.UUID = dn.fs.uuid
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

type fnSegmentRef struct {
	fn  *filenode
	idx int
}

// commitBlock concatenates the data from the given filenode segments
// (which must be *memSegments), writes the data out to Keep as a
// single block, and replaces the filenodes' *memSegments with
// storedSegments that reference the relevant portions of the new
// block.
//
// bufsize is the total data size in refs. It is used to preallocate
// the correct amount of memory when len(refs)>1.
//
// If sync is false, commitBlock returns right away, after starting a
// goroutine to do the writes, reacquire the filenodes' locks, and
// swap out the *memSegments. Some filenodes' segments might get
// modified/rearranged in the meantime, in which case commitBlock
// won't replace them.
//
// Caller must have write lock.
func (dn *dirnode) commitBlock(ctx context.Context, refs []fnSegmentRef, bufsize int, sync bool) error {
	if len(refs) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	done := make(chan struct{})
	var block []byte
	segs := make([]*memSegment, 0, len(refs))
	offsets := make([]int, 0, len(refs)) // location of segment's data within block
	for _, ref := range refs {
		seg := ref.fn.segments[ref.idx].(*memSegment)
		if !sync && seg.flushingUnfinished() {
			// Let the other flushing goroutine finish. If
			// it fails, we'll try again next time.
			close(done)
			return nil
		}
		// In sync mode, we proceed regardless of
		// whether another flush is in progress: It
		// can't finish before we do, because we hold
		// fn's lock until we finish our own writes.
		seg.flushing = done
		offsets = append(offsets, len(block))
		if len(refs) == 1 {
			block = seg.buf
		} else if block == nil {
			block = append(make([]byte, 0, bufsize), seg.buf...)
		} else {
			block = append(block, seg.buf...)
		}
		segs = append(segs, seg)
	}
	blocksize := len(block)
	dn.fs.throttle().Acquire()
	errs := make(chan error, 1)
	go func() {
		defer close(done)
		defer close(errs)
		resp, err := dn.fs.BlockWrite(context.Background(), BlockWriteOptions{
			Data:           block,
			Replicas:       dn.fs.replicas,
			StorageClasses: dn.fs.storageClasses,
		})
		dn.fs.throttle().Release()
		if err != nil {
			errs <- err
			return
		}
		for idx, ref := range refs {
			if !sync {
				ref.fn.Lock()
				// In async mode, fn's lock was
				// released while we were waiting for
				// PutB(); lots of things might have
				// changed.
				if len(ref.fn.segments) <= ref.idx {
					// file segments have
					// rearranged or changed in
					// some way
					ref.fn.Unlock()
					continue
				} else if seg, ok := ref.fn.segments[ref.idx].(*memSegment); !ok || seg != segs[idx] {
					// segment has been replaced
					ref.fn.Unlock()
					continue
				} else if seg.flushing != done {
					// seg.buf has been replaced
					ref.fn.Unlock()
					continue
				}
			}
			data := ref.fn.segments[ref.idx].(*memSegment).buf
			ref.fn.segments[ref.idx] = storedSegment{
				kc:      dn.fs,
				locator: resp.Locator,
				size:    blocksize,
				offset:  offsets[idx],
				length:  len(data),
			}
			// atomic is needed here despite caller having
			// lock: caller might be running concurrent
			// commitBlock() goroutines using the same
			// lock, writing different segments from the
			// same file.
			atomic.AddInt64(&ref.fn.memsize, -int64(len(data)))
			if !sync {
				ref.fn.Unlock()
			}
		}
	}()
	if sync {
		return <-errs
	}
	return nil
}

type flushOpts struct {
	sync        bool
	shortBlocks bool
}

// flush in-memory data and remote-cluster block references (for the
// children with the given names, which must be children of dn) to
// local-cluster persistent storage.
//
// Caller must have write lock on dn and the named children.
//
// If any children are dirs, they will be flushed recursively.
func (dn *dirnode) flush(ctx context.Context, names []string, opts flushOpts) error {
	cg := newContextGroup(ctx)
	defer cg.Cancel()

	goCommit := func(refs []fnSegmentRef, bufsize int) {
		cg.Go(func() error {
			return dn.commitBlock(cg.Context(), refs, bufsize, opts.sync)
		})
	}

	var pending []fnSegmentRef
	var pendingLen int = 0
	localLocator := map[string]string{}
	for _, name := range names {
		switch node := dn.inodes[name].(type) {
		case *dirnode:
			grandchildNames := node.sortedNames()
			for _, grandchildName := range grandchildNames {
				grandchild := node.inodes[grandchildName]
				grandchild.Lock()
				defer grandchild.Unlock()
			}
			cg.Go(func() error { return node.flush(cg.Context(), grandchildNames, opts) })
		case *filenode:
			for idx, seg := range node.segments {
				switch seg := seg.(type) {
				case storedSegment:
					loc, ok := localLocator[seg.locator]
					if !ok {
						var err error
						loc, err = dn.fs.LocalLocator(seg.locator)
						if err != nil {
							return err
						}
						localLocator[seg.locator] = loc
					}
					seg.locator = loc
					node.segments[idx] = seg
				case *memSegment:
					if seg.Len() > maxBlockSize/2 {
						goCommit([]fnSegmentRef{{node, idx}}, seg.Len())
						continue
					}
					if pendingLen+seg.Len() > maxBlockSize {
						goCommit(pending, pendingLen)
						pending = nil
						pendingLen = 0
					}
					pending = append(pending, fnSegmentRef{node, idx})
					pendingLen += seg.Len()
				default:
					panic(fmt.Sprintf("can't sync segment type %T", seg))
				}
			}
		}
	}
	if opts.shortBlocks {
		goCommit(pending, pendingLen)
	}
	return cg.Wait()
}

// caller must have write lock.
func (dn *dirnode) MemorySize() (size int64) {
	for _, name := range dn.sortedNames() {
		node := dn.inodes[name]
		node.Lock()
		defer node.Unlock()
		switch node := node.(type) {
		case *dirnode:
			size += node.MemorySize()
		case *filenode:
			size += 64
			for _, seg := range node.segments {
				switch seg := seg.(type) {
				case *memSegment:
					size += int64(seg.Len())
				}
				size += 64
			}
		}
	}
	return 64 + size
}

// caller must have write lock.
func (dn *dirnode) sortedNames() []string {
	names := make([]string, 0, len(dn.inodes))
	for name := range dn.inodes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// caller must have write lock.
func (dn *dirnode) marshalManifest(ctx context.Context, prefix string) (string, error) {
	cg := newContextGroup(ctx)
	defer cg.Cancel()

	if len(dn.inodes) == 0 {
		if prefix == "." {
			return "", nil
		}
		// Express the existence of an empty directory by
		// adding an empty file named `\056`, which (unlike
		// the more obvious spelling `.`) is accepted by the
		// API's manifest validator.
		return manifestEscape(prefix) + " d41d8cd98f00b204e9800998ecf8427e+0 0:0:\\056\n", nil
	}

	names := dn.sortedNames()

	// Wait for children to finish any pending write operations
	// before locking them.
	for _, name := range names {
		node := dn.inodes[name]
		if fn, ok := node.(*filenode); ok {
			fn.waitPrune()
		}
	}

	var dirnames []string
	var filenames []string
	for _, name := range names {
		node := dn.inodes[name]
		node.Lock()
		defer node.Unlock()
		switch node := node.(type) {
		case *dirnode:
			dirnames = append(dirnames, name)
		case *filenode:
			filenames = append(filenames, name)
		default:
			panic(fmt.Sprintf("can't marshal inode type %T", node))
		}
	}

	subdirs := make([]string, len(dirnames))
	rootdir := ""
	for i, name := range dirnames {
		i, name := i, name
		cg.Go(func() error {
			txt, err := dn.inodes[name].(*dirnode).marshalManifest(cg.Context(), prefix+"/"+name)
			subdirs[i] = txt
			return err
		})
	}

	cg.Go(func() error {
		var streamLen int64
		type filepart struct {
			name   string
			offset int64
			length int64
		}

		var fileparts []filepart
		var blocks []string
		if err := dn.flush(cg.Context(), filenames, flushOpts{sync: true, shortBlocks: true}); err != nil {
			return err
		}
		for _, name := range filenames {
			node := dn.inodes[name].(*filenode)
			if len(node.segments) == 0 {
				fileparts = append(fileparts, filepart{name: name})
				continue
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
					// calling flush(sync=true).
					panic(fmt.Sprintf("can't marshal segment type %T", seg))
				}
			}
		}
		var filetokens []string
		for _, s := range fileparts {
			filetokens = append(filetokens, fmt.Sprintf("%d:%d:%s", s.offset, s.length, manifestEscape(s.name)))
		}
		if len(filetokens) == 0 {
			return nil
		} else if len(blocks) == 0 {
			blocks = []string{"d41d8cd98f00b204e9800998ecf8427e+0"}
		}
		rootdir = manifestEscape(prefix) + " " + strings.Join(blocks, " ") + " " + strings.Join(filetokens, " ") + "\n"
		return nil
	})
	err := cg.Wait()
	return rootdir + strings.Join(subdirs, ""), err
}

func (dn *dirnode) loadManifest(txt string) error {
	streams := bytes.Split([]byte(txt), []byte{'\n'})
	if len(streams[len(streams)-1]) != 0 {
		return fmt.Errorf("line %d: no trailing newline", len(streams))
	}
	streams = streams[:len(streams)-1]
	segments := []storedSegment{}
	// To reduce allocs, we reuse a single "pathparts" slice
	// (pre-split on "/" separators) for the duration of this
	// func.
	var pathparts []string
	// To reduce allocs, we reuse a single "toks" slice of 3 byte
	// slices.
	var toks = make([][]byte, 3)
	// Similar to bytes.SplitN(token, []byte{c}, 3), but splits
	// into the toks slice rather than allocating a new one, and
	// returns the number of toks (1, 2, or 3).
	splitToToks := func(src []byte, c rune) int {
		c1 := bytes.IndexRune(src, c)
		if c1 < 0 {
			toks[0] = src
			return 1
		}
		toks[0], src = src[:c1], src[c1+1:]
		c2 := bytes.IndexRune(src, c)
		if c2 < 0 {
			toks[1] = src
			return 2
		}
		toks[1], toks[2] = src[:c2], src[c2+1:]
		return 3
	}
	for i, stream := range streams {
		lineno := i + 1
		var anyFileTokens bool
		var pos int64
		var segIdx int
		segments = segments[:0]
		pathparts = nil
		streamparts := 0
		for i, token := range bytes.Split(stream, []byte{' '}) {
			if i == 0 {
				pathparts = strings.Split(manifestUnescape(string(token)), "/")
				streamparts = len(pathparts)
				continue
			}
			if !bytes.ContainsRune(token, ':') {
				if anyFileTokens {
					return fmt.Errorf("line %d: bad file segment %q", lineno, token)
				}
				if splitToToks(token, '+') < 2 {
					return fmt.Errorf("line %d: bad locator %q", lineno, token)
				}
				length, err := strconv.ParseInt(string(toks[1]), 10, 32)
				if err != nil || length < 0 {
					return fmt.Errorf("line %d: bad locator %q", lineno, token)
				}
				segments = append(segments, storedSegment{
					locator: string(token),
					size:    int(length),
					offset:  0,
					length:  int(length),
				})
				continue
			} else if len(segments) == 0 {
				return fmt.Errorf("line %d: bad locator %q", lineno, token)
			}
			if splitToToks(token, ':') != 3 {
				return fmt.Errorf("line %d: bad file segment %q", lineno, token)
			}
			anyFileTokens = true

			offset, err := strconv.ParseInt(string(toks[0]), 10, 64)
			if err != nil || offset < 0 {
				return fmt.Errorf("line %d: bad file segment %q", lineno, token)
			}
			length, err := strconv.ParseInt(string(toks[1]), 10, 64)
			if err != nil || length < 0 {
				return fmt.Errorf("line %d: bad file segment %q", lineno, token)
			}
			if !bytes.ContainsAny(toks[2], `\/`) {
				// optimization for a common case
				pathparts = append(pathparts[:streamparts], string(toks[2]))
			} else {
				pathparts = append(pathparts[:streamparts], strings.Split(manifestUnescape(string(toks[2])), "/")...)
			}
			fnode, err := dn.createFileAndParents(pathparts)
			if fnode == nil && err == nil && length == 0 {
				// Special case: an empty file used as
				// a marker to preserve an otherwise
				// empty directory in a manifest.
				continue
			}
			if err != nil || (fnode == nil && length != 0) {
				return fmt.Errorf("line %d: cannot use name %q with length %d: %s", lineno, toks[2], length, err)
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
			for ; segIdx < len(segments); segIdx++ {
				seg := segments[segIdx]
				next := pos + int64(seg.Len())
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
		} else if streamparts == 0 {
			return fmt.Errorf("line %d: no stream name", lineno)
		}
	}
	return nil
}

// only safe to call from loadManifest -- no locking.
//
// If path is a "parent directory exists" marker (the last path
// component is "."), the returned values are both nil.
//
// Newly added nodes have modtime==0. Caller is responsible for fixing
// them with backdateTree.
func (dn *dirnode) createFileAndParents(names []string) (fn *filenode, err error) {
	var node inode = dn
	basename := names[len(names)-1]
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
		node.Lock()
		unlock := node.Unlock
		node, err = node.Child(name, func(child inode) (inode, error) {
			if child == nil {
				// note modtime will be fixed later in backdateTree()
				child, err := node.FS().newNode(name, 0755|os.ModeDir, time.Time{})
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
		unlock()
		if err != nil {
			return
		}
	}
	if basename == "." {
		return
	} else if !permittedName(basename) {
		err = fmt.Errorf("invalid file part %q in path %q", basename, names)
		return
	}
	node.Lock()
	defer node.Unlock()
	_, err = node.Child(basename, func(child inode) (inode, error) {
		switch child := child.(type) {
		case nil:
			child, err = node.FS().newNode(basename, 0755, time.Time{})
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

func (dn *dirnode) TreeSize() (bytes int64) {
	dn.RLock()
	defer dn.RUnlock()
	for _, i := range dn.inodes {
		switch i := i.(type) {
		case *filenode:
			bytes += i.Size()
		case *dirnode:
			bytes += i.TreeSize()
		}
	}
	return
}

func (dn *dirnode) Snapshot() (inode, error) {
	return dn.snapshot()
}

func (dn *dirnode) snapshot() (*dirnode, error) {
	dn.RLock()
	defer dn.RUnlock()
	snap := &dirnode{
		treenode: treenode{
			inodes:   make(map[string]inode, len(dn.inodes)),
			fileinfo: dn.fileinfo,
		},
	}
	for name, child := range dn.inodes {
		dupchild, err := child.Snapshot()
		if err != nil {
			return nil, err
		}
		snap.inodes[name] = dupchild
		dupchild.SetParent(snap, name)
	}
	return snap, nil
}

func (dn *dirnode) Splice(repl inode) error {
	repl, err := repl.Snapshot()
	if err != nil {
		return fmt.Errorf("cannot copy snapshot: %w", err)
	}
	switch repl := repl.(type) {
	default:
		return fmt.Errorf("cannot splice snapshot containing %T: %w", repl, ErrInvalidArgument)
	case *dirnode:
		dn.Lock()
		defer dn.Unlock()
		dn.inodes = repl.inodes
		dn.setTreeFS(dn.fs)
	case *filenode:
		dn.parent.Lock()
		defer dn.parent.Unlock()
		removing, err := dn.parent.Child(dn.fileinfo.name, nil)
		if err != nil {
			return fmt.Errorf("cannot use Splice to replace a top-level directory with a file: %w", ErrInvalidOperation)
		} else if removing != dn {
			// If ../thisdirname is not this dirnode, it
			// must be an inode that wraps a dirnode, like
			// a collectionFileSystem or deferrednode.
			if deferred, ok := removing.(*deferrednode); ok {
				// More useful to report the type of
				// the wrapped node rather than just
				// *deferrednode. (We know the real
				// inode is already loaded because dn
				// is inside it.)
				removing = deferred.realinode()
			}
			return fmt.Errorf("cannot use Splice to attach a file at top level of %T: %w", removing, ErrInvalidOperation)
		}
		dn.Lock()
		defer dn.Unlock()
		_, err = dn.parent.Child(dn.fileinfo.name, func(inode) (inode, error) { return repl, nil })
		if err != nil {
			return fmt.Errorf("error replacing filenode: dn.parent.Child(): %w", err)
		}
		repl.fs = dn.fs
	}
	return nil
}

func (dn *dirnode) setTreeFS(fs *collectionFileSystem) {
	dn.fs = fs
	for _, child := range dn.inodes {
		switch child := child.(type) {
		case *dirnode:
			child.setTreeFS(fs)
		case *filenode:
			child.fs = fs
		}
	}
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
	// If flushing is not nil and not ready/closed, then a) buf is
	// being shared by a pruneMemSegments goroutine, and must be
	// copied on write; and b) the flushing channel will close
	// when the goroutine finishes, whether it succeeds or not.
	flushing <-chan struct{}
}

func (me *memSegment) flushingUnfinished() bool {
	if me.flushing == nil {
		return false
	}
	select {
	case <-me.flushing:
		me.flushing = nil
		return false
	default:
		return true
	}
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
	if n > cap(me.buf) || (me.flushing != nil && n > len(me.buf)) {
		newsize := 1024
		for newsize < n {
			newsize = newsize << 2
		}
		newbuf := make([]byte, n, newsize)
		copy(newbuf, me.buf)
		me.buf, me.flushing = newbuf, nil
	} else {
		// reclaim existing capacity, and zero reclaimed part
		oldlen := len(me.buf)
		me.buf = me.buf[:n]
		for i := oldlen; i < n; i++ {
			me.buf[i] = 0
		}
	}
}

func (me *memSegment) WriteAt(p []byte, off int) {
	if off+len(p) > len(me.buf) {
		panic("overflowed segment")
	}
	if me.flushing != nil {
		me.buf, me.flushing = append([]byte(nil), me.buf...), nil
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
