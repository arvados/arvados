package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// A UnixVolume stores and retrieves blocks in a local directory.
type UnixVolume struct {
	// path to the volume's root directory
	root string
	// something to lock during IO, typically a sync.Mutex (or nil
	// to skip locking)
	locker   sync.Locker
	readonly bool
}

func (v *UnixVolume) Touch(loc string) error {
	if v.readonly {
		return MethodDisabledError
	}
	p := v.blockPath(loc)
	f, err := os.OpenFile(p, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if v.locker != nil {
		v.locker.Lock()
		defer v.locker.Unlock()
	}
	if e := lockfile(f); e != nil {
		return e
	}
	defer unlockfile(f)
	now := time.Now().Unix()
	utime := syscall.Utimbuf{now, now}
	return syscall.Utime(p, &utime)
}

func (v *UnixVolume) Mtime(loc string) (time.Time, error) {
	p := v.blockPath(loc)
	if fi, err := os.Stat(p); err != nil {
		return time.Time{}, err
	} else {
		return fi.ModTime(), nil
	}
}

// Lock the locker (if one is in use), open the file for reading, and
// call the given function if and when the file is ready to read.
func (v *UnixVolume) getFunc(path string, fn func(io.Reader) error) error {
	if v.locker != nil {
		v.locker.Lock()
		defer v.locker.Unlock()
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return fn(f)
}

// stat is os.Stat() with some extra sanity checks.
func (v *UnixVolume) stat(path string) (os.FileInfo, error) {
	stat, err := os.Stat(path)
	if err == nil {
		if stat.Size() < 0 {
			err = os.ErrInvalid
		} else if stat.Size() > BlockSize {
			err = TooLongError
		}
	}
	return stat, err
}

// Get retrieves a block identified by the locator string "loc", and
// returns its contents as a byte slice.
//
// Get returns a nil buffer IFF it returns a non-nil error.
func (v *UnixVolume) Get(loc string) ([]byte, error) {
	path := v.blockPath(loc)
	stat, err := v.stat(path)
	if err != nil {
		return nil, err
	}
	buf := bufs.Get(int(stat.Size()))
	err = v.getFunc(path, func(rdr io.Reader) error {
		_, err = io.ReadFull(rdr, buf)
		return err
	})
	if err != nil {
		bufs.Put(buf)
		return nil, err
	}
	return buf, nil
}

// Compare returns nil if Get(loc) would return the same content as
// expect. It is functionally equivalent to Get() followed by
// bytes.Compare(), but uses less memory.
func (v *UnixVolume) Compare(loc string, expect []byte) error {
	path := v.blockPath(loc)
	stat, err := v.stat(path)
	if err != nil {
		return err
	}
	bufLen := 1 << 20
	if int64(bufLen) > stat.Size() {
		bufLen = int(stat.Size())
	}
	cmp := expect
	buf := make([]byte, bufLen)
	return v.getFunc(path, func(rdr io.Reader) error {
		// Loop invariants: all data read so far matched what
		// we expected, and the first N bytes of cmp are
		// expected to equal the next N bytes read from
		// reader.
		for {
			n, err := rdr.Read(buf)
			if n > len(cmp) || bytes.Compare(cmp[:n], buf[:n]) != 0 {
				return collisionOrCorrupt(loc[:32], expect[:len(expect)-len(cmp)], buf[:n], rdr)
			}
			cmp = cmp[n:]
			if err == io.EOF {
				if len(cmp) != 0 {
					return collisionOrCorrupt(loc[:32], expect[:len(expect)-len(cmp)], nil, nil)
				}
				return nil
			} else if err != nil {
				return err
			}
		}
	})
}

// Put stores a block of data identified by the locator string
// "loc".  It returns nil on success.  If the volume is full, it
// returns a FullError.  If the write fails due to some other error,
// that error is returned.
func (v *UnixVolume) Put(loc string, block []byte) error {
	if v.readonly {
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

	tmpfile, tmperr := ioutil.TempFile(bdir, "tmp"+loc)
	if tmperr != nil {
		log.Printf("ioutil.TempFile(%s, tmp%s): %s", bdir, loc, tmperr)
		return tmperr
	}
	bpath := v.blockPath(loc)

	if v.locker != nil {
		v.locker.Lock()
		defer v.locker.Unlock()
	}
	if _, err := tmpfile.Write(block); err != nil {
		log.Printf("%s: writing to %s: %s\n", v, bpath, err)
		tmpfile.Close()
		os.Remove(tmpfile.Name())
		return err
	}
	if err := tmpfile.Close(); err != nil {
		log.Printf("closing %s: %s\n", tmpfile.Name(), err)
		os.Remove(tmpfile.Name())
		return err
	}
	if err := os.Rename(tmpfile.Name(), bpath); err != nil {
		log.Printf("rename %s %s: %s\n", tmpfile.Name(), bpath, err)
		os.Remove(tmpfile.Name())
		return err
	}
	return nil
}

// Status returns a VolumeStatus struct describing the volume's
// current state, or nil if an error occurs.
//
func (v *UnixVolume) Status() *VolumeStatus {
	var fs syscall.Statfs_t
	var devnum uint64

	if fi, err := os.Stat(v.root); err == nil {
		devnum = fi.Sys().(*syscall.Stat_t).Dev
	} else {
		log.Printf("%s: os.Stat: %s\n", v, err)
		return nil
	}

	err := syscall.Statfs(v.root, &fs)
	if err != nil {
		log.Printf("%s: statfs: %s\n", v, err)
		return nil
	}
	// These calculations match the way df calculates disk usage:
	// "free" space is measured by fs.Bavail, but "used" space
	// uses fs.Blocks - fs.Bfree.
	free := fs.Bavail * uint64(fs.Bsize)
	used := (fs.Blocks - fs.Bfree) * uint64(fs.Bsize)
	return &VolumeStatus{v.root, devnum, free, used}
}

var blockDirRe = regexp.MustCompile(`^[0-9a-f]+$`)

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
	var lastErr error = nil
	rootdir, err := os.Open(v.root)
	if err != nil {
		return err
	}
	defer rootdir.Close()
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
		blockdirpath := filepath.Join(v.root, names[0])
		blockdir, err := os.Open(blockdirpath)
		if err != nil {
			log.Print("Error reading ", blockdirpath, ": ", err)
			lastErr = err
			continue
		}
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
			_, err = fmt.Fprint(w,
				name,
				"+", fileInfo[0].Size(),
				" ", fileInfo[0].ModTime().Unix(),
				"\n")
		}
		blockdir.Close()
	}
}

func (v *UnixVolume) Delete(loc string) error {
	// Touch() must be called before calling Write() on a block.  Touch()
	// also uses lockfile().  This avoids a race condition between Write()
	// and Delete() because either (a) the file will be deleted and Touch()
	// will signal to the caller that the file is not present (and needs to
	// be re-written), or (b) Touch() will update the file's timestamp and
	// Delete() will read the correct up-to-date timestamp and choose not to
	// delete the file.

	if v.readonly {
		return MethodDisabledError
	}
	if v.locker != nil {
		v.locker.Lock()
		defer v.locker.Unlock()
	}
	p := v.blockPath(loc)
	f, err := os.OpenFile(p, os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if e := lockfile(f); e != nil {
		return e
	}
	defer unlockfile(f)

	// If the block has been PUT in the last blob_signature_ttl
	// seconds, return success without removing the block. This
	// protects data from garbage collection until it is no longer
	// possible for clients to retrieve the unreferenced blocks
	// anyway (because the permission signatures have expired).
	if fi, err := os.Stat(p); err != nil {
		return err
	} else {
		if time.Since(fi.ModTime()) < blob_signature_ttl {
			return nil
		}
	}
	return os.Remove(p)
}

// blockDir returns the fully qualified directory name for the directory
// where loc is (or would be) stored on this volume.
func (v *UnixVolume) blockDir(loc string) string {
	return filepath.Join(v.root, loc[0:3])
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
	fullSymlink := v.root + "/full"

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
	err = syscall.Statfs(v.root, &fs)
	if err == nil {
		// Statfs output is not guaranteed to measure free
		// space in terms of 1K blocks.
		free = fs.Bavail * uint64(fs.Bsize) / 1024
	}
	return
}

func (v *UnixVolume) String() string {
	return fmt.Sprintf("[UnixVolume %s]", v.root)
}

func (v *UnixVolume) Writable() bool {
	return !v.readonly
}

// lockfile and unlockfile use flock(2) to manage kernel file locks.
func lockfile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
}

func unlockfile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
