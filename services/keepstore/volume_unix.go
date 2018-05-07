// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type unixVolumeAdder struct {
	*Config
}

// String implements flag.Value
func (s *unixVolumeAdder) String() string {
	return "-"
}

func (vs *unixVolumeAdder) Set(path string) error {
	if dirs := strings.Split(path, ","); len(dirs) > 1 {
		log.Print("DEPRECATED: using comma-separated volume list.")
		for _, dir := range dirs {
			if err := vs.Set(dir); err != nil {
				return err
			}
		}
		return nil
	}
	vs.Config.Volumes = append(vs.Config.Volumes, &UnixVolume{
		Root:      path,
		ReadOnly:  deprecated.flagReadonly,
		Serialize: deprecated.flagSerializeIO,
	})
	return nil
}

func init() {
	VolumeTypes = append(VolumeTypes, func() VolumeWithExamples { return &UnixVolume{} })

	flag.Var(&unixVolumeAdder{theConfig}, "volumes", "see Volumes configuration")
	flag.Var(&unixVolumeAdder{theConfig}, "volume", "see Volumes configuration")
}

// Discover adds a UnixVolume for every directory named "keep" that is
// located at the top level of a device- or tmpfs-backed mount point
// other than "/". It returns the number of volumes added.
func (vs *unixVolumeAdder) Discover() int {
	added := 0
	f, err := os.Open(ProcMounts)
	if err != nil {
		log.Fatalf("opening %s: %s", ProcMounts, err)
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		args := strings.Fields(scanner.Text())
		if err := scanner.Err(); err != nil {
			log.Fatalf("reading %s: %s", ProcMounts, err)
		}
		dev, mount := args[0], args[1]
		if mount == "/" {
			continue
		}
		if dev != "tmpfs" && !strings.HasPrefix(dev, "/dev/") {
			continue
		}
		keepdir := mount + "/keep"
		if st, err := os.Stat(keepdir); err != nil || !st.IsDir() {
			continue
		}
		// Set the -readonly flag (but only for this volume)
		// if the filesystem is mounted readonly.
		flagReadonlyWas := deprecated.flagReadonly
		for _, fsopt := range strings.Split(args[3], ",") {
			if fsopt == "ro" {
				deprecated.flagReadonly = true
				break
			}
			if fsopt == "rw" {
				break
			}
		}
		if err := vs.Set(keepdir); err != nil {
			log.Printf("adding %q: %s", keepdir, err)
		} else {
			added++
		}
		deprecated.flagReadonly = flagReadonlyWas
	}
	return added
}

// A UnixVolume stores and retrieves blocks in a local directory.
type UnixVolume struct {
	Root                 string // path to the volume's root directory
	ReadOnly             bool
	Serialize            bool
	DirectoryReplication int
	StorageClasses       []string

	// something to lock during IO, typically a sync.Mutex (or nil
	// to skip locking)
	locker sync.Locker

	os osWithStats
}

// DeviceID returns a globally unique ID for the volume's root
// directory, consisting of the filesystem's UUID and the path from
// filesystem root to storage directory, joined by "/". For example,
// the DeviceID for a local directory "/mnt/xvda1/keep" might be
// "fa0b6166-3b55-4994-bd3f-92f4e00a1bb0/keep".
func (v *UnixVolume) DeviceID() string {
	giveup := func(f string, args ...interface{}) string {
		log.Printf(f+"; using blank DeviceID for volume %s", append(args, v)...)
		return ""
	}
	buf, err := exec.Command("findmnt", "--noheadings", "--target", v.Root).CombinedOutput()
	if err != nil {
		return giveup("findmnt: %s (%q)", err, buf)
	}
	findmnt := strings.Fields(string(buf))
	if len(findmnt) < 2 {
		return giveup("could not parse findmnt output: %q", buf)
	}
	fsRoot, dev := findmnt[0], findmnt[1]

	absRoot, err := filepath.Abs(v.Root)
	if err != nil {
		return giveup("resolving relative path %q: %s", v.Root, err)
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return giveup("resolving symlinks in %q: %s", absRoot, err)
	}

	// Find path from filesystem root to realRoot
	var fsPath string
	if strings.HasPrefix(realRoot, fsRoot+"/") {
		fsPath = realRoot[len(fsRoot):]
	} else if fsRoot == "/" {
		fsPath = realRoot
	} else if fsRoot == realRoot {
		fsPath = ""
	} else {
		return giveup("findmnt reports mount point %q which is not a prefix of volume root %q", fsRoot, realRoot)
	}

	if !strings.HasPrefix(dev, "/") {
		return giveup("mount %q device %q is not a path", fsRoot, dev)
	}

	fi, err := os.Stat(dev)
	if err != nil {
		return giveup("stat %q: %s\n", dev, err)
	}
	ino := fi.Sys().(*syscall.Stat_t).Ino

	// Find a symlink in /dev/disk/by-uuid/ whose target is (i.e.,
	// has the same inode as) the mounted device
	udir := "/dev/disk/by-uuid"
	d, err := os.Open(udir)
	if err != nil {
		return giveup("opening %q: %s", udir, err)
	}
	uuids, err := d.Readdirnames(0)
	if err != nil {
		return giveup("reading %q: %s", udir, err)
	}
	for _, uuid := range uuids {
		link := filepath.Join(udir, uuid)
		fi, err = os.Stat(link)
		if err != nil {
			log.Printf("error: stat %q: %s", link, err)
			continue
		}
		if fi.Sys().(*syscall.Stat_t).Ino == ino {
			return uuid + fsPath
		}
	}
	return giveup("could not find entry in %q matching %q", udir, dev)
}

// Examples implements VolumeWithExamples.
func (*UnixVolume) Examples() []Volume {
	return []Volume{
		&UnixVolume{
			Root:                 "/mnt/local-disk",
			Serialize:            true,
			DirectoryReplication: 1,
		},
		&UnixVolume{
			Root:                 "/mnt/network-disk",
			Serialize:            false,
			DirectoryReplication: 2,
		},
	}
}

// Type implements Volume
func (v *UnixVolume) Type() string {
	return "Directory"
}

// Start implements Volume
func (v *UnixVolume) Start() error {
	if v.Serialize {
		v.locker = &sync.Mutex{}
	}
	if !strings.HasPrefix(v.Root, "/") {
		return fmt.Errorf("volume root does not start with '/': %q", v.Root)
	}
	if v.DirectoryReplication == 0 {
		v.DirectoryReplication = 1
	}
	_, err := v.os.Stat(v.Root)
	return err
}

// Touch sets the timestamp for the given locator to the current time
func (v *UnixVolume) Touch(loc string) error {
	if v.ReadOnly {
		return MethodDisabledError
	}
	p := v.blockPath(loc)
	f, err := v.os.OpenFile(p, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := v.lock(context.TODO()); err != nil {
		return err
	}
	defer v.unlock()
	if e := v.lockfile(f); e != nil {
		return e
	}
	defer v.unlockfile(f)
	ts := syscall.NsecToTimespec(time.Now().UnixNano())
	v.os.stats.Tick(&v.os.stats.UtimesOps)
	err = syscall.UtimesNano(p, []syscall.Timespec{ts, ts})
	v.os.stats.TickErr(err)
	return err
}

// Mtime returns the stored timestamp for the given locator.
func (v *UnixVolume) Mtime(loc string) (time.Time, error) {
	p := v.blockPath(loc)
	fi, err := v.os.Stat(p)
	if err != nil {
		return time.Time{}, err
	}
	return fi.ModTime(), nil
}

// Lock the locker (if one is in use), open the file for reading, and
// call the given function if and when the file is ready to read.
func (v *UnixVolume) getFunc(ctx context.Context, path string, fn func(io.Reader) error) error {
	if err := v.lock(ctx); err != nil {
		return err
	}
	defer v.unlock()
	f, err := v.os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(NewCountingReader(ioutil.NopCloser(f), v.os.stats.TickInBytes))
}

// stat is os.Stat() with some extra sanity checks.
func (v *UnixVolume) stat(path string) (os.FileInfo, error) {
	stat, err := v.os.Stat(path)
	if err == nil {
		if stat.Size() < 0 {
			err = os.ErrInvalid
		} else if stat.Size() > BlockSize {
			err = TooLongError
		}
	}
	return stat, err
}

// Get retrieves a block, copies it to the given slice, and returns
// the number of bytes copied.
func (v *UnixVolume) Get(ctx context.Context, loc string, buf []byte) (int, error) {
	return getWithPipe(ctx, loc, buf, v)
}

// ReadBlock implements BlockReader.
func (v *UnixVolume) ReadBlock(ctx context.Context, loc string, w io.Writer) error {
	path := v.blockPath(loc)
	stat, err := v.stat(path)
	if err != nil {
		return v.translateError(err)
	}
	return v.getFunc(ctx, path, func(rdr io.Reader) error {
		n, err := io.Copy(w, rdr)
		if err == nil && n != stat.Size() {
			err = io.ErrUnexpectedEOF
		}
		return err
	})
}

// Compare returns nil if Get(loc) would return the same content as
// expect. It is functionally equivalent to Get() followed by
// bytes.Compare(), but uses less memory.
func (v *UnixVolume) Compare(ctx context.Context, loc string, expect []byte) error {
	path := v.blockPath(loc)
	if _, err := v.stat(path); err != nil {
		return v.translateError(err)
	}
	return v.getFunc(ctx, path, func(rdr io.Reader) error {
		return compareReaderWithBuf(ctx, rdr, expect, loc[:32])
	})
}

// Put stores a block of data identified by the locator string
// "loc".  It returns nil on success.  If the volume is full, it
// returns a FullError.  If the write fails due to some other error,
// that error is returned.
func (v *UnixVolume) Put(ctx context.Context, loc string, block []byte) error {
	return putWithPipe(ctx, loc, block, v)
}

// ReadBlock implements BlockWriter.
func (v *UnixVolume) WriteBlock(ctx context.Context, loc string, rdr io.Reader) error {
	if v.ReadOnly {
		return MethodDisabledError
	}
	if v.IsFull() {
		return FullError
	}
	bdir := v.blockDir(loc)
	if err := os.MkdirAll(bdir, 0755); err != nil {
		log.Printf("%s: could not create directory %s: %s",
			loc, bdir, err)
		return err
	}

	tmpfile, tmperr := v.os.TempFile(bdir, "tmp"+loc)
	if tmperr != nil {
		log.Printf("ioutil.TempFile(%s, tmp%s): %s", bdir, loc, tmperr)
		return tmperr
	}

	bpath := v.blockPath(loc)

	if err := v.lock(ctx); err != nil {
		return err
	}
	defer v.unlock()
	n, err := io.Copy(tmpfile, rdr)
	v.os.stats.TickOutBytes(uint64(n))
	if err != nil {
		log.Printf("%s: writing to %s: %s\n", v, bpath, err)
		tmpfile.Close()
		v.os.Remove(tmpfile.Name())
		return err
	}
	if err := tmpfile.Close(); err != nil {
		log.Printf("closing %s: %s\n", tmpfile.Name(), err)
		v.os.Remove(tmpfile.Name())
		return err
	}
	if err := v.os.Rename(tmpfile.Name(), bpath); err != nil {
		log.Printf("rename %s %s: %s\n", tmpfile.Name(), bpath, err)
		return v.os.Remove(tmpfile.Name())
	}
	return nil
}

// Status returns a VolumeStatus struct describing the volume's
// current state, or nil if an error occurs.
//
func (v *UnixVolume) Status() *VolumeStatus {
	fi, err := v.os.Stat(v.Root)
	if err != nil {
		log.Printf("%s: os.Stat: %s\n", v, err)
		return nil
	}
	devnum := fi.Sys().(*syscall.Stat_t).Dev

	var fs syscall.Statfs_t
	if err := syscall.Statfs(v.Root, &fs); err != nil {
		log.Printf("%s: statfs: %s\n", v, err)
		return nil
	}
	// These calculations match the way df calculates disk usage:
	// "free" space is measured by fs.Bavail, but "used" space
	// uses fs.Blocks - fs.Bfree.
	free := fs.Bavail * uint64(fs.Bsize)
	used := (fs.Blocks - fs.Bfree) * uint64(fs.Bsize)
	return &VolumeStatus{
		MountPoint: v.Root,
		DeviceNum:  devnum,
		BytesFree:  free,
		BytesUsed:  used,
	}
}

var blockDirRe = regexp.MustCompile(`^[0-9a-f]+$`)
var blockFileRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// IndexTo writes (to the given Writer) a list of blocks found on this
// volume which begin with the specified prefix. If the prefix is an
// empty string, IndexTo writes a complete list of blocks.
//
// Each block is given in the format
//
//     locator+size modification-time {newline}
//
// e.g.:
//
//     e4df392f86be161ca6ed3773a962b8f3+67108864 1388894303
//     e4d41e6fd68460e0e3fc18cc746959d2+67108864 1377796043
//     e4de7a2810f5554cd39b36d8ddb132ff+67108864 1388701136
//
func (v *UnixVolume) IndexTo(prefix string, w io.Writer) error {
	var lastErr error
	rootdir, err := v.os.Open(v.Root)
	if err != nil {
		return err
	}
	defer rootdir.Close()
	v.os.stats.Tick(&v.os.stats.ReaddirOps)
	for {
		names, err := rootdir.Readdirnames(1)
		if err == io.EOF {
			return lastErr
		} else if err != nil {
			return err
		}
		if !strings.HasPrefix(names[0], prefix) && !strings.HasPrefix(prefix, names[0]) {
			// prefix excludes all blocks stored in this dir
			continue
		}
		if !blockDirRe.MatchString(names[0]) {
			continue
		}
		blockdirpath := filepath.Join(v.Root, names[0])
		blockdir, err := v.os.Open(blockdirpath)
		if err != nil {
			log.Print("Error reading ", blockdirpath, ": ", err)
			lastErr = err
			continue
		}
		v.os.stats.Tick(&v.os.stats.ReaddirOps)
		for {
			fileInfo, err := blockdir.Readdir(1)
			if err == io.EOF {
				break
			} else if err != nil {
				log.Print("Error reading ", blockdirpath, ": ", err)
				lastErr = err
				break
			}
			name := fileInfo[0].Name()
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			if !blockFileRe.MatchString(name) {
				continue
			}
			_, err = fmt.Fprint(w,
				name,
				"+", fileInfo[0].Size(),
				" ", fileInfo[0].ModTime().UnixNano(),
				"\n")
			if err != nil {
				log.Print("Error writing : ", err)
				lastErr = err
				break
			}
		}
		blockdir.Close()
	}
}

// Trash trashes the block data from the unix storage
// If TrashLifetime == 0, the block is deleted
// Else, the block is renamed as path/{loc}.trash.{deadline},
// where deadline = now + TrashLifetime
func (v *UnixVolume) Trash(loc string) error {
	// Touch() must be called before calling Write() on a block.  Touch()
	// also uses lockfile().  This avoids a race condition between Write()
	// and Trash() because either (a) the file will be trashed and Touch()
	// will signal to the caller that the file is not present (and needs to
	// be re-written), or (b) Touch() will update the file's timestamp and
	// Trash() will read the correct up-to-date timestamp and choose not to
	// trash the file.

	if v.ReadOnly {
		return MethodDisabledError
	}
	if err := v.lock(context.TODO()); err != nil {
		return err
	}
	defer v.unlock()
	p := v.blockPath(loc)
	f, err := v.os.OpenFile(p, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if e := v.lockfile(f); e != nil {
		return e
	}
	defer v.unlockfile(f)

	// If the block has been PUT in the last blobSignatureTTL
	// seconds, return success without removing the block. This
	// protects data from garbage collection until it is no longer
	// possible for clients to retrieve the unreferenced blocks
	// anyway (because the permission signatures have expired).
	if fi, err := v.os.Stat(p); err != nil {
		return err
	} else if time.Since(fi.ModTime()) < time.Duration(theConfig.BlobSignatureTTL) {
		return nil
	}

	if theConfig.TrashLifetime == 0 {
		return v.os.Remove(p)
	}
	return v.os.Rename(p, fmt.Sprintf("%v.trash.%d", p, time.Now().Add(theConfig.TrashLifetime.Duration()).Unix()))
}

// Untrash moves block from trash back into store
// Look for path/{loc}.trash.{deadline} in storage,
// and rename the first such file as path/{loc}
func (v *UnixVolume) Untrash(loc string) (err error) {
	if v.ReadOnly {
		return MethodDisabledError
	}

	v.os.stats.Tick(&v.os.stats.ReaddirOps)
	files, err := ioutil.ReadDir(v.blockDir(loc))
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return os.ErrNotExist
	}

	foundTrash := false
	prefix := fmt.Sprintf("%v.trash.", loc)
	for _, f := range files {
		if strings.HasPrefix(f.Name(), prefix) {
			foundTrash = true
			err = v.os.Rename(v.blockPath(f.Name()), v.blockPath(loc))
			if err == nil {
				break
			}
		}
	}

	if foundTrash == false {
		return os.ErrNotExist
	}

	return
}

// blockDir returns the fully qualified directory name for the directory
// where loc is (or would be) stored on this volume.
func (v *UnixVolume) blockDir(loc string) string {
	return filepath.Join(v.Root, loc[0:3])
}

// blockPath returns the fully qualified pathname for the path to loc
// on this volume.
func (v *UnixVolume) blockPath(loc string) string {
	return filepath.Join(v.blockDir(loc), loc)
}

// IsFull returns true if the free space on the volume is less than
// MinFreeKilobytes.
//
func (v *UnixVolume) IsFull() (isFull bool) {
	fullSymlink := v.Root + "/full"

	// Check if the volume has been marked as full in the last hour.
	if link, err := os.Readlink(fullSymlink); err == nil {
		if ts, err := strconv.Atoi(link); err == nil {
			fulltime := time.Unix(int64(ts), 0)
			if time.Since(fulltime).Hours() < 1.0 {
				return true
			}
		}
	}

	if avail, err := v.FreeDiskSpace(); err == nil {
		isFull = avail < MinFreeKilobytes
	} else {
		log.Printf("%s: FreeDiskSpace: %s\n", v, err)
		isFull = false
	}

	// If the volume is full, timestamp it.
	if isFull {
		now := fmt.Sprintf("%d", time.Now().Unix())
		os.Symlink(now, fullSymlink)
	}
	return
}

// FreeDiskSpace returns the number of unused 1k blocks available on
// the volume.
//
func (v *UnixVolume) FreeDiskSpace() (free uint64, err error) {
	var fs syscall.Statfs_t
	err = syscall.Statfs(v.Root, &fs)
	if err == nil {
		// Statfs output is not guaranteed to measure free
		// space in terms of 1K blocks.
		free = fs.Bavail * uint64(fs.Bsize) / 1024
	}
	return
}

func (v *UnixVolume) String() string {
	return fmt.Sprintf("[UnixVolume %s]", v.Root)
}

// Writable returns false if all future Put, Mtime, and Delete calls
// are expected to fail.
func (v *UnixVolume) Writable() bool {
	return !v.ReadOnly
}

// Replication returns the number of replicas promised by the
// underlying device (as specified in configuration).
func (v *UnixVolume) Replication() int {
	return v.DirectoryReplication
}

// GetStorageClasses implements Volume
func (v *UnixVolume) GetStorageClasses() []string {
	return v.StorageClasses
}

// InternalStats returns I/O and filesystem ops counters.
func (v *UnixVolume) InternalStats() interface{} {
	return &v.os.stats
}

// lock acquires the serialize lock, if one is in use. If ctx is done
// before the lock is acquired, lock returns ctx.Err() instead of
// acquiring the lock.
func (v *UnixVolume) lock(ctx context.Context) error {
	if v.locker == nil {
		return nil
	}
	locked := make(chan struct{})
	go func() {
		v.locker.Lock()
		close(locked)
	}()
	select {
	case <-ctx.Done():
		go func() {
			<-locked
			v.locker.Unlock()
		}()
		return ctx.Err()
	case <-locked:
		return nil
	}
}

// unlock releases the serialize lock, if one is in use.
func (v *UnixVolume) unlock() {
	if v.locker == nil {
		return
	}
	v.locker.Unlock()
}

// lockfile and unlockfile use flock(2) to manage kernel file locks.
func (v *UnixVolume) lockfile(f *os.File) error {
	v.os.stats.Tick(&v.os.stats.FlockOps)
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
	v.os.stats.TickErr(err)
	return err
}

func (v *UnixVolume) unlockfile(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	v.os.stats.TickErr(err)
	return err
}

// Where appropriate, translate a more specific filesystem error to an
// error recognized by handlers, like os.ErrNotExist.
func (v *UnixVolume) translateError(err error) error {
	switch err.(type) {
	case *os.PathError:
		// stat() returns a PathError if the parent directory
		// (not just the file itself) is missing
		return os.ErrNotExist
	default:
		return err
	}
}

var unixTrashLocRegexp = regexp.MustCompile(`/([0-9a-f]{32})\.trash\.(\d+)$`)

// EmptyTrash walks hierarchy looking for {hash}.trash.*
// and deletes those with deadline < now.
func (v *UnixVolume) EmptyTrash() {
	var bytesDeleted, bytesInTrash int64
	var blocksDeleted, blocksInTrash int64

	doFile := func(path string, info os.FileInfo) {
		if info.Mode().IsDir() {
			return
		}
		matches := unixTrashLocRegexp.FindStringSubmatch(path)
		if len(matches) != 3 {
			return
		}
		deadline, err := strconv.ParseInt(matches[2], 10, 64)
		if err != nil {
			log.Printf("EmptyTrash: %v: ParseInt(%v): %v", path, matches[2], err)
			return
		}
		atomic.AddInt64(&bytesInTrash, info.Size())
		atomic.AddInt64(&blocksInTrash, 1)
		if deadline > time.Now().Unix() {
			return
		}
		err = v.os.Remove(path)
		if err != nil {
			log.Printf("EmptyTrash: Remove %v: %v", path, err)
			return
		}
		atomic.AddInt64(&bytesDeleted, info.Size())
		atomic.AddInt64(&blocksDeleted, 1)
	}

	type dirent struct {
		path string
		info os.FileInfo
	}
	var wg sync.WaitGroup
	todo := make(chan dirent, theConfig.EmptyTrashWorkers)
	for i := 0; i < 1 || i < theConfig.EmptyTrashWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for e := range todo {
				doFile(e.path, e.info)
			}
		}()
	}

	err := filepath.Walk(v.Root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("EmptyTrash: filepath.Walk: %v: %v", path, err)
			return nil
		}
		todo <- dirent{path, info}
		return nil
	})
	close(todo)
	wg.Wait()

	if err != nil {
		log.Printf("EmptyTrash error for %v: %v", v.String(), err)
	}

	log.Printf("EmptyTrash stats for %v: Deleted %v bytes in %v blocks. Remaining in trash: %v bytes in %v blocks.", v.String(), bytesDeleted, blocksDeleted, bytesInTrash-bytesDeleted, blocksInTrash-blocksDeleted)
}

type unixStats struct {
	statsTicker
	OpenOps    uint64
	StatOps    uint64
	FlockOps   uint64
	UtimesOps  uint64
	CreateOps  uint64
	RenameOps  uint64
	UnlinkOps  uint64
	ReaddirOps uint64
}

func (s *unixStats) TickErr(err error) {
	if err == nil {
		return
	}
	s.statsTicker.TickErr(err, fmt.Sprintf("%T", err))
}

type osWithStats struct {
	stats unixStats
}

func (o *osWithStats) Open(name string) (*os.File, error) {
	o.stats.Tick(&o.stats.OpenOps)
	f, err := os.Open(name)
	o.stats.TickErr(err)
	return f, err
}

func (o *osWithStats) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	o.stats.Tick(&o.stats.OpenOps)
	f, err := os.OpenFile(name, flag, perm)
	o.stats.TickErr(err)
	return f, err
}

func (o *osWithStats) Remove(path string) error {
	o.stats.Tick(&o.stats.UnlinkOps)
	err := os.Remove(path)
	o.stats.TickErr(err)
	return err
}

func (o *osWithStats) Rename(a, b string) error {
	o.stats.Tick(&o.stats.RenameOps)
	err := os.Rename(a, b)
	o.stats.TickErr(err)
	return err
}

func (o *osWithStats) Stat(path string) (os.FileInfo, error) {
	o.stats.Tick(&o.stats.StatOps)
	fi, err := os.Stat(path)
	o.stats.TickErr(err)
	return fi, err
}

func (o *osWithStats) TempFile(dir, base string) (*os.File, error) {
	o.stats.Tick(&o.stats.CreateOps)
	f, err := ioutil.TempFile(dir, base)
	o.stats.TickErr(err)
	return f, err
}
