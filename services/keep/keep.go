package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ======================
// Configuration settings
//
// TODO(twp): make all of these configurable via command line flags
// and/or configuration file settings.

// Default TCP address on which to listen for requests.
const DEFAULT_ADDR = ":25107"

// A Keep "block" is 64MB.
const BLOCKSIZE = 64 * 1024 * 1024

// A Keep volume must have at least MIN_FREE_KILOBYTES available
// in order to permit writes.
const MIN_FREE_KILOBYTES = BLOCKSIZE / 1024

var PROC_MOUNTS = "/proc/mounts"

var KeepVolumes []string

// ==========
// Error types.
//
type KeepError struct {
	HTTPCode int
	ErrMsg   string
}

var (
	CollisionError = &KeepError{400, "Collision"}
	MD5Error       = &KeepError{401, "MD5 Failure"}
	CorruptError   = &KeepError{402, "Corruption"}
	NotFoundError  = &KeepError{404, "Not Found"}
	GenericError   = &KeepError{500, "Fail"}
	FullError      = &KeepError{503, "Full"}
	TooLongError   = &KeepError{504, "Too Long"}
)

func (e *KeepError) Error() string {
	return e.ErrMsg
}

// This error is returned by ReadAtMost if the available
// data exceeds BLOCKSIZE bytes.
var ReadErrorTooLong = errors.New("Too long")

func main() {
	// Parse command-line flags:
	//
	// -listen=ipaddr:port
	//    Interface on which to listen for requests. Use :port without
	//    an ipaddr to listen on all network interfaces.
	//    Examples:
	//      -listen=127.0.0.1:4949
	//      -listen=10.0.1.24:8000
	//      -listen=:25107 (to listen to port 25107 on all interfaces)
	//
	// -volumes
	//    A comma-separated list of directories to use as Keep volumes.
	//    Example:
	//      -volumes=/var/keep01,/var/keep02,/var/keep03/subdir
	//
	//    If -volumes is empty or is not present, Keep will select volumes
	//    by looking at currently mounted filesystems for /keep top-level
	//    directories.

	var listen, keepvols string
	flag.StringVar(&listen, "listen", DEFAULT_ADDR,
		"interface on which to listen for requests, in the format ipaddr:port. e.g. -listen=10.0.1.24:8000. Use -listen=:port to listen on all network interfaces.")
	flag.StringVar(&keepvols, "volumes", "",
		"Comma-separated list of directories to use for Keep volumes, e.g. -volumes=/var/keep1,/var/keep2. If empty or not supplied, Keep will scan mounted filesystems for volumes with a /keep top-level directory.")
	flag.Parse()

	// Look for local keep volumes.
	if keepvols == "" {
		// TODO(twp): decide whether this is desirable default behavior.
		// In production we may want to require the admin to specify
		// Keep volumes explicitly.
		KeepVolumes = FindKeepVolumes()
	} else {
		KeepVolumes = strings.Split(keepvols, ",")
	}

	if len(KeepVolumes) == 0 {
		log.Fatal("could not find any keep volumes")
	}
	for _, v := range KeepVolumes {
		log.Println("keep volume:", v)
	}

	// Set up REST handlers.
	//
	// Start with a router that will route each URL path to an
	// appropriate handler.
	//
	rest := mux.NewRouter()
	rest.HandleFunc(`/{hash:[0-9a-f]{32}}`, GetBlockHandler).Methods("GET", "HEAD")
	rest.HandleFunc(`/{hash:[0-9a-f]{32}}`, PutBlockHandler).Methods("PUT")
	rest.HandleFunc(`/index`, IndexHandler).Methods("GET", "HEAD")
	rest.HandleFunc(`/index/{prefix:[0-9a-f]{0,32}}`, IndexHandler).Methods("GET", "HEAD")
	rest.HandleFunc(`/status.json`, StatusHandler).Methods("GET", "HEAD")

	// Tell the built-in HTTP server to direct all requests to the REST
	// router.
	http.Handle("/", rest)

	// Start listening for requests.
	http.ListenAndServe(listen, nil)
}

// FindKeepVolumes
//     Returns a list of Keep volumes mounted on this system.
//
//     A Keep volume is a normal or tmpfs volume with a /keep
//     directory at the top level of the mount point.
//
func FindKeepVolumes() []string {
	vols := make([]string, 0)

	if f, err := os.Open(PROC_MOUNTS); err != nil {
		log.Fatalf("opening %s: %s\n", PROC_MOUNTS, err)
	} else {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			args := strings.Fields(scanner.Text())
			dev, mount := args[0], args[1]
			if (dev == "tmpfs" || strings.HasPrefix(dev, "/dev/")) && mount != "/" {
				keep := mount + "/keep"
				if st, err := os.Stat(keep); err == nil && st.IsDir() {
					vols = append(vols, keep)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
	return vols
}

func GetBlockHandler(w http.ResponseWriter, req *http.Request) {
	hash := mux.Vars(req)["hash"]

	block, err := GetBlock(hash)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}

	_, err = w.Write(block)
	if err != nil {
		log.Printf("GetBlockHandler: writing response: %s", err)
	}

	return
}

func PutBlockHandler(w http.ResponseWriter, req *http.Request) {
	hash := mux.Vars(req)["hash"]

	// Read the block data to be stored.
	// If the request exceeds BLOCKSIZE bytes, issue a HTTP 500 error.
	//
	// Note: because req.Body is a buffered Reader, each Read() call will
	// collect only the data in the network buffer (typically 16384 bytes),
	// even if it is passed a much larger slice.
	//
	// Instead, call ReadAtMost to read data from the socket
	// repeatedly until either EOF or BLOCKSIZE bytes have been read.
	//
	if buf, err := ReadAtMost(req.Body, BLOCKSIZE); err == nil {
		if err := PutBlock(buf, hash); err == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			ke := err.(*KeepError)
			http.Error(w, ke.Error(), ke.HTTPCode)
		}
	} else {
		log.Println("error reading request: ", err)
		errmsg := err.Error()
		if err == ReadErrorTooLong {
			// Use a more descriptive error message that includes
			// the maximum request size.
			errmsg = fmt.Sprintf("Max request size %d bytes", BLOCKSIZE)
		}
		http.Error(w, errmsg, 500)
	}
}

func IndexHandler(w http.ResponseWriter, req *http.Request) {
	prefix := mux.Vars(req)["prefix"]

	index := IndexLocators(prefix)
	w.Write([]byte(index))
}

// StatusHandler
//     Responds to /status.json requests with the current node status,
//     described in a JSON structure.
//
//     The data given in a status.json response includes:
//        time - the time the status was last updated
//        df   - the output of the most recent `df --block-size=1k`
//        disk_devices - list of disk device files (i.e. /dev/(s|xv|h)d)
//        dirs - an object describing Keep volumes, keyed on Keep volume dirs,
//          each value is an object describing the status of that volume
//            * status ("full [timestamp]" or "ok [timestamp]")
//            * last_error
//            * last_error_time
//
type VolumeStatus struct {
	Space       string
	LastErr     string
	LastErrTime time.Time
}

type NodeStatus struct {
	LastUpdate time.Time
	DfOutput   string
	DiskDev    []string
	Volumes    map[string]VolumeStatus
}

func StatusHandler(w http.ResponseWriter, req *http.Request) {
	st := new(NodeStatus)
	st.LastUpdate = time.Now()

	// Get a list of disk devices on this system.
	st.DiskDev = make([]string, 1)
	if devdir, err := os.Open("/dev"); err != nil {
		log.Printf("StatusHandler: opening /dev: %s\n", err)
	} else {
		devs, err := devdir.Readdirnames(0)
		if err == nil {
			for _, d := range devs {
				if strings.HasPrefix(d, "sd") ||
					strings.HasPrefix(d, "hd") ||
					strings.HasPrefix(d, "xvd") {
					st.DiskDev = append(st.DiskDev, d)
				}
			}
		} else {
			log.Printf("Readdirnames: %s", err)
		}
	}

	st.Volumes = make(map[string]VolumeStatus)
	for _, vol := range KeepVolumes {
		st.Volumes[vol] = GetVolumeStatus(vol)
	}

	if jstat, err := json.Marshal(st); err == nil {
		w.Write(jstat)
	} else {
		log.Printf("json.Marshal: %s\n", err)
		log.Printf("NodeStatus = %v\n", st)
		http.Error(w, err.Error(), 500)
	}
}

// GetVolumeStatus
//     Returns a VolumeStatus describing the requested volume.
func GetVolumeStatus(volume string) VolumeStatus {
	var isfull, lasterr string
	var lasterrtime time.Time

	if IsFull(volume) {
		isfull = fmt.Sprintf("full %d", time.Now().Unix())
	} else {
		isfull = fmt.Sprintf("ok %d", time.Now().Unix())
	}

	// Not implemented yet
	lasterr = ""
	lasterrtime = time.Unix(0, 0)

	return VolumeStatus{isfull, lasterr, lasterrtime}
}

// IndexLocators
//     Returns a string containing a list of locator ids found on this
//     Keep server.  If {prefix} is given, return only those locator
//     ids that begin with the given prefix string.
//
//     The return string consists of a sequence of newline-separated
//     strings in the format
//
//         locator+size modification-time
//
//     e.g.:
//
//         e4df392f86be161ca6ed3773a962b8f3+67108864 1388894303
//         e4d41e6fd68460e0e3fc18cc746959d2+67108864 1377796043
//         e4de7a2810f5554cd39b36d8ddb132ff+67108864 1388701136
//
func IndexLocators(prefix string) string {
	var output string
	for _, vol := range KeepVolumes {
		filepath.Walk(vol,
			func(path string, info os.FileInfo, err error) error {
				// This WalkFunc inspects each path in the volume
				// and prints an index line for all files that begin
				// with prefix.
				if err != nil {
					log.Printf("IndexHandler: %s: walking to %s: %s",
						vol, path, err)
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
				// Print filenames beginning with prefix
				if !info.IsDir() && strings.HasPrefix(locator, prefix) {
					output = output + fmt.Sprintf(
						"%s+%d %d\n", locator, info.Size(), info.ModTime().Unix())
				}
				return nil
			})
	}

	return output
}

func GetBlock(hash string) ([]byte, error) {
	var buf = make([]byte, BLOCKSIZE)

	// Attempt to read the requested hash from a keep volume.
	for _, vol := range KeepVolumes {
		var f *os.File
		var err error
		var nread int

		blockFilename := fmt.Sprintf("%s/%s/%s", vol, hash[0:3], hash)

		f, err = os.Open(blockFilename)
		if err != nil {
			if !os.IsNotExist(err) {
				// A block is stored on only one Keep disk,
				// so os.IsNotExist is expected.  Report any other errors.
				log.Printf("%s: opening %s: %s\n", vol, blockFilename, err)
			}
			continue
		}

		nread, err = f.Read(buf)
		if err != nil {
			log.Printf("%s: reading %s: %s\n", vol, blockFilename, err)
			continue
		}

		// Double check the file checksum.
		//
		filehash := fmt.Sprintf("%x", md5.Sum(buf[:nread]))
		if filehash != hash {
			// TODO(twp): this condition probably represents a bad disk and
			// should raise major alarm bells for an administrator: e.g.
			// they should be sent directly to an event manager at high
			// priority or logged as urgent problems.
			//
			log.Printf("%s: checksum mismatch: %s (actual hash %s)\n",
				vol, blockFilename, filehash)
			return buf, CorruptError
		}

		// Success!
		return buf[:nread], nil
	}

	log.Printf("%s: not found on any volumes, giving up\n", hash)
	return buf, NotFoundError
}

/* PutBlock(block, hash)
   Stores the BLOCK (identified by the content id HASH) in Keep.

   The MD5 checksum of the block must be identical to the content id HASH.
   If not, an error is returned.

   PutBlock stores the BLOCK on the first Keep volume with free space.
   A failure code is returned to the user only if all volumes fail.

   On success, PutBlock returns nil.
   On failure, it returns a KeepError with one of the following codes:

   400 Collision
          A different block with the same hash already exists on this
          Keep server.
   401 MD5Fail
          The MD5 hash of the BLOCK does not match the argument HASH.
   503 Full
          There was not enough space left in any Keep volume to store
          the object.
   500 Fail
          The object could not be stored for some other reason (e.g.
          all writes failed). The text of the error message should
          provide as much detail as possible.
*/

func PutBlock(block []byte, hash string) error {
	// Check that BLOCK's checksum matches HASH.
	blockhash := fmt.Sprintf("%x", md5.Sum(block))
	if blockhash != hash {
		log.Printf("%s: MD5 checksum %s did not match request", hash, blockhash)
		return MD5Error
	}

	// If we already have a block on disk under this identifier, return
	// success (but check for MD5 collisions).
	// The only errors that GetBlock can return are ErrCorrupt and ErrNotFound.
	// In either case, we want to write our new (good) block to disk, so there is
	// nothing special to do if err != nil.
	if oldblock, err := GetBlock(hash); err == nil {
		if bytes.Compare(block, oldblock) == 0 {
			return nil
		} else {
			return CollisionError
		}
	}

	// Store the block on the first available Keep volume.
	allFull := true
	for _, vol := range KeepVolumes {
		if IsFull(vol) {
			continue
		}
		allFull = false
		blockDir := fmt.Sprintf("%s/%s", vol, hash[0:3])
		if err := os.MkdirAll(blockDir, 0755); err != nil {
			log.Printf("%s: could not create directory %s: %s",
				hash, blockDir, err)
			continue
		}

		tmpfile, tmperr := ioutil.TempFile(blockDir, "tmp"+hash)
		if tmperr != nil {
			log.Printf("ioutil.TempFile(%s, tmp%s): %s", blockDir, hash, tmperr)
			continue
		}
		blockFilename := fmt.Sprintf("%s/%s", blockDir, hash)

		if _, err := tmpfile.Write(block); err != nil {
			log.Printf("%s: writing to %s: %s\n", vol, blockFilename, err)
			continue
		}
		if err := tmpfile.Close(); err != nil {
			log.Printf("closing %s: %s\n", tmpfile.Name(), err)
			os.Remove(tmpfile.Name())
			continue
		}
		if err := os.Rename(tmpfile.Name(), blockFilename); err != nil {
			log.Printf("rename %s %s: %s\n", tmpfile.Name(), blockFilename, err)
			os.Remove(tmpfile.Name())
			continue
		}
		return nil
	}

	if allFull {
		log.Printf("all Keep volumes full")
		return FullError
	} else {
		log.Printf("all Keep volumes failed")
		return GenericError
	}
}

func IsFull(volume string) (isFull bool) {
	fullSymlink := volume + "/full"

	// Check if the volume has been marked as full in the last hour.
	if link, err := os.Readlink(fullSymlink); err == nil {
		if ts, err := strconv.Atoi(link); err == nil {
			fulltime := time.Unix(int64(ts), 0)
			if time.Since(fulltime).Hours() < 1.0 {
				return true
			}
		}
	}

	if avail, err := FreeDiskSpace(volume); err == nil {
		isFull = avail < MIN_FREE_KILOBYTES
	} else {
		log.Printf("%s: FreeDiskSpace: %s\n", volume, err)
		isFull = false
	}

	// If the volume is full, timestamp it.
	if isFull {
		now := fmt.Sprintf("%d", time.Now().Unix())
		os.Symlink(now, fullSymlink)
	}
	return
}

// FreeDiskSpace(volume)
//     Returns the amount of available disk space on VOLUME,
//     as a number of 1k blocks.
//
func FreeDiskSpace(volume string) (free uint64, err error) {
	var fs syscall.Statfs_t
	err = syscall.Statfs(volume, &fs)
	if err == nil {
		// Statfs output is not guaranteed to measure free
		// space in terms of 1K blocks.
		free = fs.Bavail * uint64(fs.Bsize) / 1024
	}
	return
}

// ReadAtMost
//     Reads bytes repeatedly from an io.Reader until either
//     encountering EOF, or the maxbytes byte limit has been reached.
//     Returns a byte slice of the bytes that were read.
//
//     If the reader contains more than maxbytes, returns a nil slice
//     and an error.
//
func ReadAtMost(r io.Reader, maxbytes int) ([]byte, error) {
	// Attempt to read one more byte than maxbytes.
	lr := io.LimitReader(r, int64(maxbytes+1))
	buf, err := ioutil.ReadAll(lr)
	if len(buf) > maxbytes {
		return nil, ReadErrorTooLong
	}
	return buf, err
}
