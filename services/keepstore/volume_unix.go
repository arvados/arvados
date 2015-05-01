// A UnixVolume is a Volume backed by a locally mounted disk.
//
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// IORequests are encapsulated Get or Put requests.  They are used to
// implement serialized I/O (i.e. only one read/write operation per
// volume). When running in serialized mode, the Keep front end sends
// IORequests on a channel to an IORunner, which handles them one at a
// time and returns an IOResponse.
//
type IOMethod int

const (
	KeepGet IOMethod = iota
	KeepPut
)

type IORequest struct {
	method IOMethod
	loc    string
	data   []byte
	reply  chan *IOResponse
}

type IOResponse struct {
	data []byte
	err  error
}

// A UnixVolume has the following properties:
//
//   root
//       the path to the volume's root directory
//   queue
//       A channel of IORequests. If non-nil, all I/O requests for
//       this volume should be queued on this channel; the result
//       will be delivered on the IOResponse channel supplied in the
//       request.
//
type UnixVolume struct {
	root     string // path to this volume
	queue    chan *IORequest
	readonly bool
}

func (v *UnixVolume) IOHandler() {
	for req := range v.queue {
		var result IOResponse
		switch req.method {
		case KeepGet:
			result.data, result.err = v.Read(req.loc)
		case KeepPut:
			result.err = v.Write(req.loc, req.data)
		}
		req.reply <- &result
	}
}

func MakeUnixVolume(root string, serialize bool, readonly bool) *UnixVolume {
	v := &UnixVolume{
		root: root,
		queue: nil,
		readonly: readonly,
	}
	if serialize {
		v.queue =make(chan *IORequest)
		go v.IOHandler()
	}
	return v
}

func (v *UnixVolume) Get(loc string) ([]byte, error) {
	if v.queue == nil {
		return v.Read(loc)
	}
	reply := make(chan *IOResponse)
	v.queue <- &IORequest{KeepGet, loc, nil, reply}
	response := <-reply
	return response.data, response.err
}

func (v *UnixVolume) Put(loc string, block []byte) error {
	if v.readonly {
		return MethodDisabledError
	}
	if v.queue == nil {
		return v.Write(loc, block)
	}
	reply := make(chan *IOResponse)
	v.queue <- &IORequest{KeepPut, loc, block, reply}
	response := <-reply
	return response.err
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

// Read retrieves a block identified by the locator string "loc", and
// returns its contents as a byte slice.
//
// If the block could not be opened or read, Read returns a nil slice
// and the os.Error that was generated.
//
// If the block is present but its content hash does not match loc,
// Read returns the block and a CorruptError.  It is the caller's
// responsibility to decide what (if anything) to do with the
// corrupted data block.
//
func (v *UnixVolume) Read(loc string) ([]byte, error) {
	buf, err := ioutil.ReadFile(v.blockPath(loc))
	return buf, err
}

// Write stores a block of data identified by the locator string
// "loc".  It returns nil on success.  If the volume is full, it
// returns a FullError.  If the write fails due to some other error,
// that error is returned.
//
func (v *UnixVolume) Write(loc string, block []byte) error {
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

	if _, err := tmpfile.Write(block); err != nil {
		log.Printf("%s: writing to %s: %s\n", v, bpath, err)
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
// current state.
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

// Index returns a list of blocks found on this volume which begin with
// the specified prefix. If the prefix is an empty string, Index returns
// a complete list of blocks.
//
// The return value is a multiline string (separated by
// newlines). Each line is in the format
//
//     locator+size modification-time
//
// e.g.:
//
//     e4df392f86be161ca6ed3773a962b8f3+67108864 1388894303
//     e4d41e6fd68460e0e3fc18cc746959d2+67108864 1377796043
//     e4de7a2810f5554cd39b36d8ddb132ff+67108864 1388701136
//
func (v *UnixVolume) Index(prefix string) (output string) {
	filepath.Walk(v.root,
		func(path string, info os.FileInfo, err error) error {
			// This WalkFunc inspects each path in the volume
			// and prints an index line for all files that begin
			// with prefix.
			if err != nil {
				log.Printf("IndexHandler: %s: walking to %s: %s",
					v, path, err)
				return nil
			}
			locator := filepath.Base(path)
			// Skip directories that do not match prefix.
			// We know there is nothing interesting inside.
			if info.IsDir() &&
				!strings.HasPrefix(locator, prefix) &&
				!strings.HasPrefix(prefix, locator) {
				return filepath.SkipDir
			}
			// Skip any file that is not apparently a locator, e.g. .meta files
			if !IsValidLocator(locator) {
				return nil
			}
			// Print filenames beginning with prefix
			if !info.IsDir() && strings.HasPrefix(locator, prefix) {
				output = output + fmt.Sprintf(
					"%s+%d %d\n", locator, info.Size(), info.ModTime().Unix())
			}
			return nil
		})

	return
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
// MIN_FREE_KILOBYTES.
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
		isFull = avail < MIN_FREE_KILOBYTES
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
