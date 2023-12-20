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
type DiskCache struct {
	KeepGateway
	Dir     string
	MaxSize int64
	Logger  logrus.FieldLogger

	tidying        int32 // see tidy()
	tidyHoldUntil  time.Time
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
}

type writeprogress struct {
	cond *sync.Cond // broadcast whenever size or done changes
	done bool       // size and err have their final values
	size int        // bytes copied into cache file so far
	err  error      // error encountered while copying from backend to cache
}

type openFileEnt struct {
	sync.RWMutex
	f   *os.File
	err error // if err is non-nil, f should not be used.
}

const (
	cacheFileSuffix  = ".keepcacheblock"
	tmpFileSuffix    = ".tmp"
	tidyHoldDuration = 10 * time.Second
)

func (cache *DiskCache) cacheFile(locator string) string {
	hash := locator
	if i := strings.Index(hash, "+"); i > 0 {
		hash = hash[:i]
	}
	return filepath.Join(cache.Dir, hash[:3], hash+cacheFileSuffix)
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
	cache.gotidy()
	unique := fmt.Sprintf("%x.%p%s", os.Getpid(), &opts, tmpFileSuffix)
	tmpfilename := filepath.Join(cache.Dir, "tmp", unique)
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
		defer tmpfile.Close()
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
	cache.gotidy()
	cachefilename := cache.cacheFile(locator)
	if n, err := cache.quickReadAt(cachefilename, dst, offset); err == nil {
		return n, err
	}
	readf, err := cache.openFile(cachefilename, os.O_CREATE|os.O_RDONLY)
	if err != nil {
		return 0, fmt.Errorf("ReadAt: %w", err)
	}
	defer readf.Close()

	err = syscall.Flock(int(readf.Fd()), syscall.LOCK_SH)
	if err != nil {
		return 0, fmt.Errorf("flock(%s, lock_sh) failed: %w", cachefilename, err)
	}

	cache.writingLock.Lock()
	progress := cache.writing[cachefilename]
	if progress != nil {
		cache.writingLock.Unlock()
	} else {
		progress = &writeprogress{}
		progress.cond = sync.NewCond(&sync.Mutex{})
		if cache.writing == nil {
			cache.writing = map[string]*writeprogress{}
		}
		cache.writing[cachefilename] = progress
		cache.writingLock.Unlock()

		// Start a goroutine to copy from backend to f. As
		// data arrives, wake up any waiting loops (see below)
		// so ReadAt() requests for partial data can return as
		// soon as the relevant bytes have been copied.
		go func() {
			var size int
			var writef *os.File
			var err error
			defer func() {
				closeErr := writef.Close()
				if err == nil {
					err = closeErr
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
			}()
			writef, err = cache.openFile(cachefilename, os.O_WRONLY)
			if err != nil {
				err = fmt.Errorf("ReadAt: %w", err)
				return
			}
			err = syscall.Flock(int(writef.Fd()), syscall.LOCK_SH)
			if err != nil {
				err = fmt.Errorf("flock(%s, lock_sh) failed: %w", cachefilename, err)
				return
			}
			size, err = cache.KeepGateway.BlockRead(context.Background(), BlockReadOptions{
				Locator: locator,
				WriteTo: funcwriter(func(p []byte) (int, error) {
					n, err := writef.Write(p)
					if n > 0 {
						progress.cond.L.Lock()
						progress.size += n
						progress.cond.L.Unlock()
						progress.cond.Broadcast()
					}
					return n, err
				})})
		}()
	}
	progress.cond.L.Lock()
	for !progress.done && progress.size < len(dst)+offset {
		progress.cond.Wait()
	}
	ok := progress.size >= len(dst)+offset
	err = progress.err
	progress.cond.L.Unlock()

	if !ok && err != nil {
		// If the copy-from-backend goroutine encountered an
		// error before copying enough bytes to satisfy our
		// request, we return that error.
		return 0, err
	} else {
		// Regardless of whether the copy-from-backend
		// goroutine succeeded, or failed after copying the
		// bytes we need, the only errors we need to report
		// are errors reading from the cache file.
		return readf.ReadAt(dst, int64(offset))
	}
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
			cache.heldopenMax = 256
		} else if lim.Cur > 40000 {
			cache.heldopenMax = 10000
		} else {
			cache.heldopenMax = int(lim.Cur / 4)
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
		// Open and flock the file, then call wg.Done() to
		// unblock any other goroutines that are waiting in
		// the !isnew case above.
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
		if int(blocksize)-offset > len(buf) {
			buf = buf[:int(blocksize)-offset]
		}
		nr, err := cache.ReadAt(opts.Locator, buf, offset)
		if nr > 0 {
			nw, err := opts.WriteTo.Write(buf)
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
	// Return quickly if another tidy goroutine is running in this process.
	n := atomic.AddInt32(&cache.tidying, 1)
	if n != 1 || time.Now().Before(cache.tidyHoldUntil) {
		atomic.AddInt32(&cache.tidying, -1)
		return
	}
	go func() {
		cache.tidy()
		cache.tidyHoldUntil = time.Now().Add(tidyHoldDuration)
		atomic.AddInt32(&cache.tidying, -1)
	}()
}

// Delete cache files as needed to control disk usage.
func (cache *DiskCache) tidy() {
	maxsize := cache.MaxSize
	if maxsize < 1 {
		if maxsize = atomic.LoadInt64(&cache.defaultMaxSize); maxsize == 0 {
			var stat unix.Statfs_t
			if nil == unix.Statfs(cache.Dir, &stat) {
				maxsize = int64(stat.Bavail) * stat.Bsize / 10
			}
			atomic.StoreInt64(&cache.defaultMaxSize, maxsize)
		}
	}

	// Bail if a tidy goroutine is running in a different process.
	lockfile, err := cache.openFile(filepath.Join(cache.Dir, "tmp", "tidy.lock"), os.O_CREATE|os.O_WRONLY)
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
	filepath.Walk(cache.Dir, func(path string, info fs.FileInfo, err error) error {
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
	// GiB} as the defaultSize for subsequent tidy() operations.
	if maxsize == 0 {
		if totalsize < 1<<30 {
			atomic.StoreInt64(&cache.defaultMaxSize, 1<<30)
		} else {
			atomic.StoreInt64(&cache.defaultMaxSize, totalsize)
		}
		cache.debugf("found initial size %d, setting defaultMaxSize %d", totalsize, cache.defaultMaxSize)
		return
	}

	if totalsize <= maxsize {
		return
	}

	// Delete oldest entries until totalsize < maxsize.
	sort.Slice(ents, func(i, j int) bool {
		return ents[i].atime.Before(ents[j].atime)
	})
	deleted := 0
	for _, ent := range ents {
		os.Remove(ent.path)
		go cache.deleteHeldopen(ent.path, nil)
		deleted++
		totalsize -= ent.size
		if totalsize <= maxsize {
			break
		}
	}

	if cache.Logger != nil {
		cache.Logger.WithFields(logrus.Fields{
			"deleted":   deleted,
			"totalsize": totalsize,
		}).Debugf("DiskCache: remaining cache usage after deleting")
	}
}
