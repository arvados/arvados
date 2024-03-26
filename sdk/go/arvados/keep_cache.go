// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

type KeepGateway interface {
	ReadAt(locator string, dst []byte, offset int) (int, error)
	BlockRead(ctx context.Context, opts BlockReadOptions) (int, error)
	BlockWrite(ctx context.Context, opts BlockWriteOptions) (BlockWriteResponse, error)
	LocalLocator(locator string) (string, error)
}

// DiskCache wraps KeepGateway, adding a disk-based cache layer.
//
// A DiskCache is automatically incorporated into the backend stack of
// each keepclient.KeepClient. Most programs do not need to use
// DiskCache directly.
type DiskCache struct {
	KeepGateway
	Dir     string
	MaxSize ByteSizeOrPercent
	Logger  logrus.FieldLogger

	*sharedCache
	setupOnce sync.Once
}

var (
	sharedCachesLock sync.Mutex
	sharedCaches     = map[string]*sharedCache{}
)

// sharedCache has fields that coordinate the cache usage in a single
// cache directory; it can be shared by multiple DiskCaches.
//
// This serves to share a single pool of held-open filehandles, a
// single tidying goroutine, etc., even when the program (like
// keep-web) uses multiple KeepGateway stacks that use different auth
// tokens, etc.
type sharedCache struct {
	dir     string
	maxSize ByteSizeOrPercent

	tidying        int32 // see tidy()
	defaultMaxSize int64

	// The "heldopen" fields are used to open cache files for
	// reading, and leave them open for future/concurrent ReadAt
	// operations. See quickReadAt.
	heldopen     map[string]*openFileEnt
	heldopenMax  int
	heldopenLock sync.Mutex

	// The "writing" fields allow multiple concurrent/sequential
	// ReadAt calls to be notified as a single
	// read-block-from-backend-into-cache goroutine fills the
	// cache file.
	writing     map[string]*writeprogress
	writingCond *sync.Cond
	writingLock sync.Mutex

	sizeMeasured    int64 // actual size on disk after last tidy(); zero if not measured yet
	sizeEstimated   int64 // last measured size, plus files we have written since
	lastFileCount   int64 // number of files on disk at last count
	writesSinceTidy int64 // number of files written since last tidy()
}

type writeprogress struct {
	cond    *sync.Cond     // broadcast whenever size or done changes
	done    bool           // size and err have their final values
	size    int            // bytes copied into cache file so far
	err     error          // error encountered while copying from backend to cache
	sharedf *os.File       // readable filehandle, usable if done && err==nil
	readers sync.WaitGroup // goroutines that haven't finished reading from f yet
}

type openFileEnt struct {
	sync.RWMutex
	f   *os.File
	err error // if err is non-nil, f should not be used.
}

const (
	cacheFileSuffix = ".keepcacheblock"
	tmpFileSuffix   = ".tmp"
)

func (cache *DiskCache) setup() {
	sharedCachesLock.Lock()
	defer sharedCachesLock.Unlock()
	dir := cache.Dir
	if sharedCaches[dir] == nil {
		sharedCaches[dir] = &sharedCache{dir: dir, maxSize: cache.MaxSize}
	}
	cache.sharedCache = sharedCaches[dir]
}

func (cache *DiskCache) cacheFile(locator string) string {
	hash := locator
	if i := strings.Index(hash, "+"); i > 0 {
		hash = hash[:i]
	}
	return filepath.Join(cache.dir, hash[:3], hash+cacheFileSuffix)
}

// Open a cache file, creating the parent dir if necessary.
func (cache *DiskCache) openFile(name string, flags int) (*os.File, error) {
	f, err := os.OpenFile(name, flags, 0600)
	if os.IsNotExist(err) {
		// Create the parent dir and try again. (We could have
		// checked/created the parent dir before, but that
		// would be less efficient in the much more common
		// situation where it already exists.)
		parent, _ := filepath.Split(name)
		os.Mkdir(parent, 0700)
		f, err = os.OpenFile(name, flags, 0600)
	}
	return f, err
}

// Rename a file, creating the new path's parent dir if necessary.
func (cache *DiskCache) rename(old, new string) error {
	if nil == os.Rename(old, new) {
		return nil
	}
	parent, _ := filepath.Split(new)
	os.Mkdir(parent, 0700)
	return os.Rename(old, new)
}

func (cache *DiskCache) debugf(format string, args ...interface{}) {
	logger := cache.Logger
	if logger == nil {
		return
	}
	logger.Debugf(format, args...)
}

// BlockWrite writes through to the wrapped KeepGateway, and (if
// possible) retains a copy of the written block in the cache.
func (cache *DiskCache) BlockWrite(ctx context.Context, opts BlockWriteOptions) (BlockWriteResponse, error) {
	cache.setupOnce.Do(cache.setup)
	unique := fmt.Sprintf("%x.%p%s", os.Getpid(), &opts, tmpFileSuffix)
	tmpfilename := filepath.Join(cache.dir, "tmp", unique)
	tmpfile, err := cache.openFile(tmpfilename, os.O_CREATE|os.O_EXCL|os.O_RDWR)
	if err != nil {
		cache.debugf("BlockWrite: open(%s) failed: %s", tmpfilename, err)
		return cache.KeepGateway.BlockWrite(ctx, opts)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	copyerr := make(chan error, 1)

	// Start a goroutine to copy the caller's source data to
	// tmpfile, a hash checker, and (via pipe) the wrapped
	// KeepGateway.
	pipereader, pipewriter := io.Pipe()
	defer pipereader.Close()
	go func() {
		// Note this is a double-close (which is a no-op) in
		// the happy path.
		defer tmpfile.Close()
		// Note this is a no-op in the happy path (the
		// uniquely named tmpfilename will have been renamed).
		defer os.Remove(tmpfilename)
		defer pipewriter.Close()

		// Copy from opts.Data or opts.Reader, depending on
		// which was provided.
		var src io.Reader
		if opts.Data != nil {
			src = bytes.NewReader(opts.Data)
		} else {
			src = opts.Reader
		}

		hashcheck := md5.New()
		n, err := io.Copy(io.MultiWriter(tmpfile, pipewriter, hashcheck), src)
		if err != nil {
			copyerr <- err
			cancel()
			return
		} else if opts.DataSize > 0 && opts.DataSize != int(n) {
			copyerr <- fmt.Errorf("block size %d did not match provided size %d", n, opts.DataSize)
			cancel()
			return
		}
		err = tmpfile.Close()
		if err != nil {
			// Don't rename tmpfile into place, but allow
			// the BlockWrite call to succeed if nothing
			// else goes wrong.
			return
		}
		hash := fmt.Sprintf("%x", hashcheck.Sum(nil))
		if opts.Hash != "" && opts.Hash != hash {
			// Even if the wrapped KeepGateway doesn't
			// notice a problem, this should count as an
			// error.
			copyerr <- fmt.Errorf("block hash %s did not match provided hash %s", hash, opts.Hash)
			cancel()
			return
		}
		cachefilename := cache.cacheFile(hash)
		err = cache.rename(tmpfilename, cachefilename)
		if err != nil {
			cache.debugf("BlockWrite: rename(%s, %s) failed: %s", tmpfilename, cachefilename, err)
		}
		atomic.AddInt64(&cache.sizeEstimated, int64(n))
		cache.gotidy()
	}()

	// Write through to the wrapped KeepGateway from the pipe,
	// instead of the original reader.
	newopts := opts
	if newopts.DataSize == 0 {
		newopts.DataSize = len(newopts.Data)
	}
	newopts.Reader = pipereader
	newopts.Data = nil

	resp, err := cache.KeepGateway.BlockWrite(ctx, newopts)
	if len(copyerr) > 0 {
		// If the copy-to-pipe goroutine failed, that error
		// will be more helpful than the resulting "context
		// canceled" or "read [from pipereader] failed" error
		// seen by the wrapped KeepGateway.
		//
		// If the wrapped KeepGateway encounters an error
		// before all the data is copied into the pipe, it
		// stops reading from the pipe, which causes the
		// io.Copy() in the goroutine to block until our
		// deferred pipereader.Close() call runs. In that case
		// len(copyerr)==0 here, so the wrapped KeepGateway
		// error is the one we return to our caller.
		err = <-copyerr
	}
	return resp, err
}

type funcwriter func([]byte) (int, error)

func (fw funcwriter) Write(p []byte) (int, error) {
	return fw(p)
}

// ReadAt reads the entire block from the wrapped KeepGateway into the
// cache if needed, and copies the requested portion into the provided
// slice.
//
// ReadAt returns as soon as the requested portion is available in the
// cache. The remainder of the block may continue to be copied into
// the cache in the background.
func (cache *DiskCache) ReadAt(locator string, dst []byte, offset int) (int, error) {
	cache.setupOnce.Do(cache.setup)
	cachefilename := cache.cacheFile(locator)
	if n, err := cache.quickReadAt(cachefilename, dst, offset); err == nil {
		return n, nil
	}

	cache.writingLock.Lock()
	progress := cache.writing[cachefilename]
	if progress == nil {
		// Nobody else is fetching from backend, so we'll add
		// a new entry to cache.writing, fetch in a separate
		// goroutine.
		progress = &writeprogress{}
		progress.cond = sync.NewCond(&sync.Mutex{})
		if cache.writing == nil {
			cache.writing = map[string]*writeprogress{}
		}
		cache.writing[cachefilename] = progress

		// Start a goroutine to copy from backend to f. As
		// data arrives, wake up any waiting loops (see below)
		// so ReadAt() requests for partial data can return as
		// soon as the relevant bytes have been copied.
		go func() {
			var size int
			var err error
			defer func() {
				if err == nil && progress.sharedf != nil {
					err = progress.sharedf.Sync()
				}
				progress.cond.L.Lock()
				progress.err = err
				progress.done = true
				progress.size = size
				progress.cond.L.Unlock()
				progress.cond.Broadcast()
				cache.writingLock.Lock()
				delete(cache.writing, cachefilename)
				cache.writingLock.Unlock()

				// Wait for other goroutines to wake
				// up, notice we're done, and use our
				// sharedf to read their data, before
				// we close sharedf.
				//
				// Nobody can join the WaitGroup after
				// the progress entry is deleted from
				// cache.writing above. Therefore,
				// this Wait ensures nobody else is
				// accessing progress, and we don't
				// need to lock anything.
				progress.readers.Wait()
				progress.sharedf.Close()
			}()
			progress.sharedf, err = cache.openFile(cachefilename, os.O_CREATE|os.O_RDWR)
			if err != nil {
				err = fmt.Errorf("ReadAt: %w", err)
				return
			}
			err = syscall.Flock(int(progress.sharedf.Fd()), syscall.LOCK_SH)
			if err != nil {
				err = fmt.Errorf("flock(%s, lock_sh) failed: %w", cachefilename, err)
				return
			}
			size, err = cache.KeepGateway.BlockRead(context.Background(), BlockReadOptions{
				Locator: locator,
				WriteTo: funcwriter(func(p []byte) (int, error) {
					n, err := progress.sharedf.Write(p)
					if n > 0 {
						progress.cond.L.Lock()
						progress.size += n
						progress.cond.L.Unlock()
						progress.cond.Broadcast()
					}
					return n, err
				})})
			atomic.AddInt64(&cache.sizeEstimated, int64(size))
			cache.gotidy()
		}()
	}
	// We add ourselves to the readers WaitGroup so the
	// fetch-from-backend goroutine doesn't close the shared
	// filehandle before we read the data we need from it.
	progress.readers.Add(1)
	defer progress.readers.Done()
	cache.writingLock.Unlock()

	progress.cond.L.Lock()
	for !progress.done && progress.size < len(dst)+offset {
		progress.cond.Wait()
	}
	sharedf := progress.sharedf
	err := progress.err
	progress.cond.L.Unlock()

	if err != nil {
		// If the copy-from-backend goroutine encountered an
		// error, we return that error. (Even if we read the
		// desired number of bytes, the error might be
		// something like BadChecksum so we should not ignore
		// it.)
		return 0, err
	}
	if len(dst) == 0 {
		// It's possible that sharedf==nil here (the writer
		// goroutine might not have done anything at all yet)
		// and we don't need it anyway because no bytes are
		// being read. Reading zero bytes seems pointless, but
		// if someone does it, we might as well return
		// suitable values, rather than risk a crash by
		// calling sharedf.ReadAt() when sharedf is nil.
		return 0, nil
	}
	return sharedf.ReadAt(dst, int64(offset))
}

var quickReadAtLostRace = errors.New("quickReadAt: lost race")

// Remove the cache entry for the indicated cachefilename if it
// matches expect (quickReadAt() usage), or if expect is nil (tidy()
// usage).
//
// If expect is non-nil, close expect's filehandle.
//
// If expect is nil and a different cache entry is deleted, close its
// filehandle.
func (cache *DiskCache) deleteHeldopen(cachefilename string, expect *openFileEnt) {
	needclose := expect

	cache.heldopenLock.Lock()
	found := cache.heldopen[cachefilename]
	if found != nil && (expect == nil || expect == found) {
		delete(cache.heldopen, cachefilename)
		needclose = found
	}
	cache.heldopenLock.Unlock()

	if needclose != nil {
		needclose.Lock()
		defer needclose.Unlock()
		if needclose.f != nil {
			needclose.f.Close()
			needclose.f = nil
		}
	}
}

// quickReadAt attempts to use a cached-filehandle approach to read
// from the indicated file. The expectation is that the caller
// (ReadAt) will try a more robust approach when this fails, so
// quickReadAt doesn't try especially hard to ensure success in
// races. In particular, when there are concurrent calls, and one
// fails, that can cause others to fail too.
func (cache *DiskCache) quickReadAt(cachefilename string, dst []byte, offset int) (int, error) {
	isnew := false
	cache.heldopenLock.Lock()
	if cache.heldopenMax == 0 {
		// Choose a reasonable limit on open cache files based
		// on RLIMIT_NOFILE. Note Go automatically raises
		// softlimit to hardlimit, so it's typically 1048576,
		// not 1024.
		lim := syscall.Rlimit{}
		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &lim)
		if err != nil {
			cache.heldopenMax = 100
		} else if lim.Cur > 400000 {
			cache.heldopenMax = 10000
		} else {
			cache.heldopenMax = int(lim.Cur / 40)
		}
	}
	heldopen := cache.heldopen[cachefilename]
	if heldopen == nil {
		isnew = true
		heldopen = &openFileEnt{}
		if cache.heldopen == nil {
			cache.heldopen = make(map[string]*openFileEnt, cache.heldopenMax)
		} else if len(cache.heldopen) > cache.heldopenMax {
			// Rather than go to the trouble of tracking
			// last access time, just close all files, and
			// open again as needed. Even in the worst
			// pathological case, this causes one extra
			// open+close per read, which is not
			// especially bad (see benchmarks).
			go func(m map[string]*openFileEnt) {
				for _, heldopen := range m {
					heldopen.Lock()
					defer heldopen.Unlock()
					if heldopen.f != nil {
						heldopen.f.Close()
						heldopen.f = nil
					}
				}
			}(cache.heldopen)
			cache.heldopen = nil
		}
		cache.heldopen[cachefilename] = heldopen
		heldopen.Lock()
	}
	cache.heldopenLock.Unlock()

	if isnew {
		// Open and flock the file, save the filehandle (or
		// error) in heldopen.f, and release the write lock so
		// other goroutines waiting at heldopen.RLock() below
		// can use the shared filehandle (or shared error).
		f, err := os.Open(cachefilename)
		if err == nil {
			err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH)
			if err == nil {
				heldopen.f = f
			} else {
				f.Close()
			}
		}
		if err != nil {
			heldopen.err = err
			go cache.deleteHeldopen(cachefilename, heldopen)
		}
		heldopen.Unlock()
	}
	// Acquire read lock to ensure (1) initialization is complete,
	// if it's done by a different goroutine, and (2) any "delete
	// old/unused entries" waits for our read to finish before
	// closing the file.
	heldopen.RLock()
	defer heldopen.RUnlock()
	if heldopen.err != nil {
		// Other goroutine encountered an error during setup
		return 0, heldopen.err
	} else if heldopen.f == nil {
		// Other goroutine closed the file before we got RLock
		return 0, quickReadAtLostRace
	}

	// If another goroutine is currently writing the file, wait
	// for it to catch up to the end of the range we need.
	cache.writingLock.Lock()
	progress := cache.writing[cachefilename]
	cache.writingLock.Unlock()
	if progress != nil {
		progress.cond.L.Lock()
		for !progress.done && progress.size < len(dst)+offset {
			progress.cond.Wait()
		}
		progress.cond.L.Unlock()
		// If size<needed && progress.err!=nil here, we'll end
		// up reporting a less helpful "EOF reading from cache
		// file" below, instead of the actual error fetching
		// from upstream to cache file.  This is OK though,
		// because our caller (ReadAt) doesn't even report our
		// error, it just retries.
	}

	n, err := heldopen.f.ReadAt(dst, int64(offset))
	if err != nil {
		// wait for any concurrent users to finish, then
		// delete this cache entry in case reopening the
		// backing file helps.
		go cache.deleteHeldopen(cachefilename, heldopen)
	}
	return n, err
}

// BlockRead reads an entire block using a 128 KiB buffer.
func (cache *DiskCache) BlockRead(ctx context.Context, opts BlockReadOptions) (int, error) {
	cache.setupOnce.Do(cache.setup)
	i := strings.Index(opts.Locator, "+")
	if i < 0 || i >= len(opts.Locator) {
		return 0, errors.New("invalid block locator: no size hint")
	}
	sizestr := opts.Locator[i+1:]
	i = strings.Index(sizestr, "+")
	if i > 0 {
		sizestr = sizestr[:i]
	}
	blocksize, err := strconv.ParseInt(sizestr, 10, 32)
	if err != nil || blocksize < 0 {
		return 0, errors.New("invalid block locator: invalid size hint")
	}

	offset := 0
	buf := make([]byte, 131072)
	for offset < int(blocksize) {
		if ctx.Err() != nil {
			return offset, ctx.Err()
		}
		if int(blocksize)-offset < len(buf) {
			buf = buf[:int(blocksize)-offset]
		}
		nr, err := cache.ReadAt(opts.Locator, buf, offset)
		if nr > 0 {
			nw, err := opts.WriteTo.Write(buf[:nr])
			if err != nil {
				return offset + nw, err
			}
		}
		offset += nr
		if err != nil {
			return offset, err
		}
	}
	return offset, nil
}

// Start a tidy() goroutine, unless one is already running / recently
// finished.
func (cache *DiskCache) gotidy() {
	writes := atomic.AddInt64(&cache.writesSinceTidy, 1)
	// Skip if another tidy goroutine is running in this process.
	n := atomic.AddInt32(&cache.tidying, 1)
	if n != 1 {
		atomic.AddInt32(&cache.tidying, -1)
		return
	}
	// Skip if sizeEstimated is based on an actual measurement and
	// is below maxSize, and we haven't done very many writes
	// since last tidy (defined as 1% of number of cache files at
	// last count).
	if cache.sizeMeasured > 0 &&
		atomic.LoadInt64(&cache.sizeEstimated) < atomic.LoadInt64(&cache.defaultMaxSize) &&
		writes < cache.lastFileCount/100 {
		atomic.AddInt32(&cache.tidying, -1)
		return
	}
	go func() {
		cache.tidy()
		atomic.StoreInt64(&cache.writesSinceTidy, 0)
		atomic.AddInt32(&cache.tidying, -1)
	}()
}

// Delete cache files as needed to control disk usage.
func (cache *DiskCache) tidy() {
	maxsize := int64(cache.maxSize.ByteSize())
	if maxsize < 1 {
		maxsize = atomic.LoadInt64(&cache.defaultMaxSize)
		if maxsize == 0 {
			// defaultMaxSize not yet computed. Use 10% of
			// filesystem capacity (or different
			// percentage if indicated by cache.maxSize)
			pct := cache.maxSize.Percent()
			if pct == 0 {
				pct = 10
			}
			var stat unix.Statfs_t
			if nil == unix.Statfs(cache.dir, &stat) {
				maxsize = int64(stat.Bavail) * stat.Bsize * pct / 100
				atomic.StoreInt64(&cache.defaultMaxSize, maxsize)
			} else {
				// In this case we will set
				// defaultMaxSize below after
				// measuring current usage.
			}
		}
	}

	// Bail if a tidy goroutine is running in a different process.
	lockfile, err := cache.openFile(filepath.Join(cache.dir, "tmp", "tidy.lock"), os.O_CREATE|os.O_WRONLY)
	if err != nil {
		return
	}
	defer lockfile.Close()
	err = syscall.Flock(int(lockfile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return
	}

	type entT struct {
		path  string
		atime time.Time
		size  int64
	}
	var ents []entT
	var totalsize int64
	filepath.Walk(cache.dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			cache.debugf("tidy: skipping dir %s: %s", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, cacheFileSuffix) && !strings.HasSuffix(path, tmpFileSuffix) {
			return nil
		}
		var atime time.Time
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			// Access time is available (hopefully the
			// filesystem is not mounted with noatime)
			atime = time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
		} else {
			// If access time isn't available we fall back
			// to sorting by modification time.
			atime = info.ModTime()
		}
		ents = append(ents, entT{path, atime, info.Size()})
		totalsize += info.Size()
		return nil
	})
	if cache.Logger != nil {
		cache.Logger.WithFields(logrus.Fields{
			"totalsize": totalsize,
			"maxsize":   maxsize,
		}).Debugf("DiskCache: checked current cache usage")
	}

	// If MaxSize wasn't specified and we failed to come up with a
	// defaultSize above, use the larger of {current cache size, 1
	// GiB} as the defaultMaxSize for subsequent tidy()
	// operations.
	if maxsize == 0 {
		if totalsize < 1<<30 {
			atomic.StoreInt64(&cache.defaultMaxSize, 1<<30)
		} else {
			atomic.StoreInt64(&cache.defaultMaxSize, totalsize)
		}
		cache.debugf("found initial size %d, setting defaultMaxSize %d", totalsize, cache.defaultMaxSize)
		return
	}

	// If we're below MaxSize or there's only one block in the
	// cache, just update the usage estimate and return.
	//
	// (We never delete the last block because that would merely
	// cause the same block to get re-fetched repeatedly from the
	// backend.)
	if totalsize <= maxsize || len(ents) == 1 {
		atomic.StoreInt64(&cache.sizeMeasured, totalsize)
		atomic.StoreInt64(&cache.sizeEstimated, totalsize)
		cache.lastFileCount = int64(len(ents))
		return
	}

	// Set a new size target of maxsize minus 5%.  This makes some
	// room for sizeEstimate to grow before it triggers another
	// tidy. We don't want to walk/sort an entire large cache
	// directory each time we write a block.
	target := maxsize - (maxsize / 20)

	// Delete oldest entries until totalsize < target or we're
	// down to a single cached block.
	sort.Slice(ents, func(i, j int) bool {
		return ents[i].atime.Before(ents[j].atime)
	})
	deleted := 0
	for _, ent := range ents {
		os.Remove(ent.path)
		go cache.deleteHeldopen(ent.path, nil)
		deleted++
		totalsize -= ent.size
		if totalsize <= target || deleted == len(ents)-1 {
			break
		}
	}

	if cache.Logger != nil {
		cache.Logger.WithFields(logrus.Fields{
			"deleted":   deleted,
			"totalsize": totalsize,
		}).Debugf("DiskCache: remaining cache usage after deleting")
	}
	atomic.StoreInt64(&cache.sizeMeasured, totalsize)
	atomic.StoreInt64(&cache.sizeEstimated, totalsize)
	cache.lastFileCount = int64(len(ents) - deleted)
}
