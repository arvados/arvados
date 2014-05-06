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
	"regexp"
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

// The Keep VolumeManager maintains a list of available volumes.
var KeepVM VolumeManager

// enforce_permissions controls whether permission signatures
// should be enforced (affecting GET and DELETE requests)
var enforce_permissions bool

// permission_ttl is the time duration (in seconds) for which
// new permission signatures (returned by PUT requests) will be
// valid.
var permission_ttl int

// data_manager_token represents the API token used by the
// Data Manager, and is required on certain privileged operations.
var data_manager_token string

// ==========
// Error types.
//
type KeepError struct {
	HTTPCode int
	ErrMsg   string
}

var (
	CollisionError  = &KeepError{400, "Collision"}
	MD5Error        = &KeepError{401, "MD5 Failure"}
	PermissionError = &KeepError{401, "Permission denied"}
	CorruptError    = &KeepError{402, "Corruption"}
	ExpiredError    = &KeepError{403, "Expired permission signature"}
	NotFoundError   = &KeepError{404, "Not Found"}
	GenericError    = &KeepError{500, "Fail"}
	FullError       = &KeepError{503, "Full"}
	TooLongError    = &KeepError{504, "Too Long"}
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

	var data_manager_token, listen, permission_key, volumearg string
	var serialize_io bool
	flag.StringVar(
		&data_manager_token,
		"data-manager-token",
		"",
		"API token used by the Data Manager. All DELETE requests or unqualified GET /index requests must carry this token.")
	flag.BoolVar(
		&enforce_permissions,
		"enforce-permissions",
		false,
		"Enforce permission signatures on requests.")
	flag.StringVar(
		&listen,
		"listen",
		DEFAULT_ADDR,
		"interface on which to listen for requests, in the format ipaddr:port. e.g. -listen=10.0.1.24:8000. Use -listen=:port to listen on all network interfaces.")
	flag.StringVar(
		&permission_key,
		"permission-key",
		"",
		"Secret key to use for generating and verifying permission signatures.")
	flag.IntVar(
		&permission_ttl,
		"permission-ttl",
		300,
		"Expiration time (in seconds) for newly generated permission signatures.")
	flag.BoolVar(
		&serialize_io,
		"serialize",
		false,
		"If set, all read and write operations on local Keep volumes will be serialized.")
	flag.StringVar(
		&volumearg,
		"volumes",
		"",
		"Comma-separated list of directories to use for Keep volumes, e.g. -volumes=/var/keep1,/var/keep2. If empty or not supplied, Keep will scan mounted filesystems for volumes with a /keep top-level directory.")
	flag.Parse()

	// Look for local keep volumes.
	var keepvols []string
	if volumearg == "" {
		// TODO(twp): decide whether this is desirable default behavior.
		// In production we may want to require the admin to specify
		// Keep volumes explicitly.
		keepvols = FindKeepVolumes()
	} else {
		keepvols = strings.Split(volumearg, ",")
	}

	// Check that the specified volumes actually exist.
	var goodvols []Volume = nil
	for _, v := range keepvols {
		if _, err := os.Stat(v); err == nil {
			log.Println("adding Keep volume:", v)
			newvol := MakeUnixVolume(v, serialize_io)
			goodvols = append(goodvols, &newvol)
		} else {
			log.Printf("bad Keep volume: %s\n", err)
		}
	}

	if len(goodvols) == 0 {
		log.Fatal("could not find any keep volumes")
	}

	// Initialize permission key.
	if permission_key != "" {
		PermissionSecret = []byte(permission_key)
	}

	// If --enforce-permissions is true, we must have a permission key to continue.
	if enforce_permissions && PermissionSecret == nil {
		log.Fatal("--enforce-permissions requires a permission key")
	}

	// Start a round-robin VolumeManager with the volumes we have found.
	KeepVM = MakeRRVolumeManager(goodvols)

	// Tell the built-in HTTP server to direct all requests to the REST
	// router.
	http.Handle("/", NewRESTRouter())

	// Start listening for requests.
	http.ListenAndServe(listen, nil)
}

// NewRESTRouter
//     Returns a mux.Router that passes GET and PUT requests to the
//     appropriate handlers.
//
func NewRESTRouter() *mux.Router {
	rest := mux.NewRouter()
	rest.HandleFunc(`/{hash:[0-9a-f]{32}}`, GetBlockHandler).Methods("GET", "HEAD")
	rest.HandleFunc(`/{hash:[0-9a-f]{32}}+A{signature:[0-9a-f]+}@{timestamp:[0-9a-f]+}`, GetBlockHandler).Methods("GET", "HEAD")
	rest.HandleFunc(`/{hash:[0-9a-f]{32}}`, PutBlockHandler).Methods("PUT")
	rest.HandleFunc(`/index`, IndexHandler).Methods("GET", "HEAD")
	rest.HandleFunc(`/index/{prefix:[0-9a-f]{0,32}}`, IndexHandler).Methods("GET", "HEAD")
	rest.HandleFunc(`/status.json`, StatusHandler).Methods("GET", "HEAD")
	return rest
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
	signature := mux.Vars(req)["signature"]
	timestamp := mux.Vars(req)["timestamp"]

	// If permission checking is in effect, verify this
	// request's permission signature.
	if enforce_permissions {
		if signature == "" || timestamp == "" {
			http.Error(w, PermissionError.Error(), PermissionError.HTTPCode)
			return
		} else if IsExpired(timestamp) {
			http.Error(w, ExpiredError.Error(), ExpiredError.HTTPCode)
			return
		} else if signature != MakePermSignature(hash, GetApiToken(req), timestamp) {
			http.Error(w, PermissionError.Error(), PermissionError.HTTPCode)
			return
		}
	}

	block, err := GetBlock(hash)
	if err != nil {
		http.Error(w, err.Error(), err.(*KeepError).HTTPCode)
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
			// Success; sign the locator and return it to the client.
			api_token := GetApiToken(req)
			expiry := time.Now().Add( // convert permission_ttl to time.Duration
				time.Duration(permission_ttl) * time.Second)
			signed_loc := SignLocator(hash, api_token, expiry)
			w.Write([]byte(signed_loc))
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

// IndexHandler
//     A HandleFunc to address /index and /index/{prefix} requests.
//
func IndexHandler(w http.ResponseWriter, req *http.Request) {
	prefix := mux.Vars(req)["prefix"]

	// Only the data manager may issue unqualified "GET /index" requests.
	if prefix == "" {
		if data_manager_token != GetApiToken(req) {
			http.Error(w, PermissionError.Error(), PermissionError.HTTPCode)
			return
		}
	}
	var index string
	for _, vol := range KeepVM.Volumes() {
		index = index + vol.Index(prefix)
	}
	w.Write([]byte(index))
}

// StatusHandler
//     Responds to /status.json requests with the current node status,
//     described in a JSON structure.
//
//     The data given in a status.json response includes:
//        volumes - a list of Keep volumes currently in use by this server
//          each volume is an object with the following fields:
//            * mount_point
//            * device_num (an integer identifying the underlying filesystem)
//            * bytes_free
//            * bytes_used
//
type VolumeStatus struct {
	MountPoint string `json:"mount_point"`
	DeviceNum  uint64 `json:"device_num"`
	BytesFree  uint64 `json:"bytes_free"`
	BytesUsed  uint64 `json:"bytes_used"`
}

type NodeStatus struct {
	Volumes []*VolumeStatus `json:"volumes"`
}

func StatusHandler(w http.ResponseWriter, req *http.Request) {
	st := GetNodeStatus()
	if jstat, err := json.Marshal(st); err == nil {
		w.Write(jstat)
	} else {
		log.Printf("json.Marshal: %s\n", err)
		log.Printf("NodeStatus = %v\n", st)
		http.Error(w, err.Error(), 500)
	}
}

// GetNodeStatus
//     Returns a NodeStatus struct describing this Keep
//     node's current status.
//
func GetNodeStatus() *NodeStatus {
	st := new(NodeStatus)

	st.Volumes = make([]*VolumeStatus, len(KeepVM.Volumes()))
	for i, vol := range KeepVM.Volumes() {
		st.Volumes[i] = vol.Status()
	}
	return st
}

// GetVolumeStatus
//     Returns a VolumeStatus describing the requested volume.
//
func GetVolumeStatus(volume string) *VolumeStatus {
	var fs syscall.Statfs_t
	var devnum uint64

	if fi, err := os.Stat(volume); err == nil {
		devnum = fi.Sys().(*syscall.Stat_t).Dev
	} else {
		log.Printf("GetVolumeStatus: os.Stat: %s\n", err)
		return nil
	}

	err := syscall.Statfs(volume, &fs)
	if err != nil {
		log.Printf("GetVolumeStatus: statfs: %s\n", err)
		return nil
	}
	// These calculations match the way df calculates disk usage:
	// "free" space is measured by fs.Bavail, but "used" space
	// uses fs.Blocks - fs.Bfree.
	free := fs.Bavail * uint64(fs.Bsize)
	used := (fs.Blocks - fs.Bfree) * uint64(fs.Bsize)
	return &VolumeStatus{volume, devnum, free, used}
}

func GetBlock(hash string) ([]byte, error) {
	// Attempt to read the requested hash from a keep volume.
	for _, vol := range KeepVM.Volumes() {
		if buf, err := vol.Get(hash); err != nil {
			// IsNotExist is an expected error and may be ignored.
			// (If all volumes report IsNotExist, we return a NotFoundError)
			// A CorruptError should be returned immediately.
			// Any other errors should be logged but we continue trying to
			// read.
			switch {
			case os.IsNotExist(err):
				continue
			default:
				log.Printf("GetBlock: reading %s: %s\n", hash, err)
			}
		} else {
			// Double check the file checksum.
			//
			filehash := fmt.Sprintf("%x", md5.Sum(buf))
			if filehash != hash {
				// TODO(twp): this condition probably represents a bad disk and
				// should raise major alarm bells for an administrator: e.g.
				// they should be sent directly to an event manager at high
				// priority or logged as urgent problems.
				//
				log.Printf("%s: checksum mismatch for request %s (actual hash %s)\n",
					vol, hash, filehash)
				return buf, CorruptError
			}
			// Success!
			return buf, nil
		}
	}

	log.Printf("%s: not found on any volumes, giving up\n", hash)
	return nil, NotFoundError
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

	// Choose a Keep volume to write to.
	// If this volume fails, try all of the volumes in order.
	vol := KeepVM.Choose()
	if err := vol.Put(hash, block); err == nil {
		return nil // success!
	} else {
		allFull := true
		for _, vol := range KeepVM.Volumes() {
			err := vol.Put(hash, block)
			if err == nil {
				return nil // success!
			}
			if err != FullError {
				// The volume is not full but the write did not succeed.
				// Report the error and continue trying.
				allFull = false
				log.Printf("%s: Write(%s): %s\n", vol, hash, err)
			}
		}

		if allFull {
			log.Printf("all Keep volumes full")
			return FullError
		} else {
			log.Printf("all Keep volumes failed")
			return GenericError
		}
	}
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

// IsValidLocator
//     Return true if the specified string is a valid Keep locator.
//     When Keep is extended to support hash types other than MD5,
//     this should be updated to cover those as well.
//
func IsValidLocator(loc string) bool {
	match, err := regexp.MatchString(`^[0-9a-f]{32}$`, loc)
	if err == nil {
		return match
	}
	log.Printf("IsValidLocator: %s\n", err)
	return false
}

// GetApiToken returns the OAuth token from the Authorization
// header of a HTTP request, or an empty string if no matching
// token is found.
func GetApiToken(req *http.Request) string {
	if auth, ok := req.Header["Authorization"]; ok {
		if strings.HasPrefix(auth[0], "OAuth ") {
			return auth[0][6:]
		}
	}
	return ""
}

// IsExpired returns true if the given Unix timestamp (expressed as a
// hexadecimal string) is in the past.
func IsExpired(timestamp_hex string) bool {
	ts, err := strconv.ParseInt(timestamp_hex, 16, 0)
	if err != nil {
		log.Printf("IsExpired: %s\n", err)
		return true
	}
	return time.Unix(ts, 0).Before(time.Now())
}
