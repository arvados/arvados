package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Default TCP address on which to listen for requests.
const DEFAULT_ADDR = ":25107"

// A Keep "block" is 64MB.
const BLOCKSIZE = 64 * 1024 * 1024

// A Keep volume must have at least MIN_FREE_KILOBYTES available
// in order to permit writes.
const MIN_FREE_KILOBYTES = BLOCKSIZE / 1024

var PROC_MOUNTS = "/proc/mounts"

var KeepVolumes []string

type KeepError struct {
	HTTPCode int
	Err      error
}

const (
	ErrCollision = 400
	ErrMD5Fail   = 401
	ErrCorrupt   = 402
	ErrNotFound  = 404
	ErrOther     = 500
	ErrFull      = 503
)

func (e *KeepError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.HTTPCode, e.Err.Error())
}

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
		"interface on which to listen for requests")
	flag.StringVar(&keepvols, "volumes", "",
		"comma-separated list of directories to use for Keep volumes")
	flag.Parse()

	// Look for local keep volumes.
	if keepvols == "" {
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
	rest.HandleFunc("/{hash:[0-9a-f]{32}}", GetBlockHandler).Methods("GET")
	rest.HandleFunc("/{hash:[0-9a-f]{32}}", PutBlockHandler).Methods("PUT")

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
	// TODO(twp): decide what to do when the input stream contains
	// more than BLOCKSIZE bytes.
	//
	buf := make([]byte, BLOCKSIZE)
	if nread, err := req.Body.Read(buf); err == nil {
		if err := PutBlock(buf[:nread], hash); err == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			ke := err.(*KeepError)
			http.Error(w, ke.Error(), ke.HTTPCode)
		}
	} else {
		log.Println("error reading request: ", err)
		http.Error(w, err.Error(), 500)
	}
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
			return buf, &KeepError{ErrCorrupt, errors.New("Corrupt")}
		}

		// Success!
		return buf[:nread], nil
	}

	log.Printf("%s: not found on any volumes, giving up\n", hash)
	return buf, &KeepError{ErrNotFound, errors.New("not found: " + hash)}
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
		return &KeepError{ErrMD5Fail, errors.New("MD5Fail")}
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
			return &KeepError{ErrCollision, errors.New("Collision")}
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
		return &KeepError{ErrFull, errors.New("Full")}
	} else {
		log.Printf("all Keep volumes failed")
		return &KeepError{ErrOther, errors.New("Fail")}
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
