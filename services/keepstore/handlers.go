package main

// REST handlers for Keep are implemented here.
//
// GetBlockHandler (GET /locator)
// PutBlockHandler (PUT /locator)
// IndexHandler    (GET /index, GET /index/prefix)
// StatusHandler   (GET /status.json)

import (
	"container/list"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MakeRESTRouter returns a new mux.Router that forwards all Keep
// requests to the appropriate handlers.
//
func MakeRESTRouter() *mux.Router {
	rest := mux.NewRouter()

	rest.HandleFunc(
		`/{hash:[0-9a-f]{32}}`, GetBlockHandler).Methods("GET", "HEAD")
	rest.HandleFunc(
		`/{hash:[0-9a-f]{32}}+{hints}`,
		GetBlockHandler).Methods("GET", "HEAD")

	rest.HandleFunc(`/{hash:[0-9a-f]{32}}`, PutBlockHandler).Methods("PUT")
	rest.HandleFunc(`/{hash:[0-9a-f]{32}}`, DeleteHandler).Methods("DELETE")
	// List all blocks stored here. Privileged client only.
	rest.HandleFunc(`/index`, IndexHandler).Methods("GET", "HEAD")
	// List blocks stored here whose hash has the given prefix.
	// Privileged client only.
	rest.HandleFunc(`/index/{prefix:[0-9a-f]{0,32}}`, IndexHandler).Methods("GET", "HEAD")

	// List volumes: path, device number, bytes used/avail.
	rest.HandleFunc(`/status.json`, StatusHandler).Methods("GET", "HEAD")

	// Replace the current pull queue.
	rest.HandleFunc(`/pull`, PullHandler).Methods("PUT")

	// Replace the current trash queue.
	rest.HandleFunc(`/trash`, TrashHandler).Methods("PUT")

	// Untrash moves blocks from trash back into store
	rest.HandleFunc(`/untrash/{hash:[0-9a-f]{32}}`, UntrashHandler).Methods("PUT")

	// Any request which does not match any of these routes gets
	// 400 Bad Request.
	rest.NotFoundHandler = http.HandlerFunc(BadRequestHandler)

	return rest
}

// BadRequestHandler is a HandleFunc to address bad requests.
func BadRequestHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, BadRequestError.Error(), BadRequestError.HTTPCode)
}

// GetBlockHandler is a HandleFunc to address Get block requests.
func GetBlockHandler(resp http.ResponseWriter, req *http.Request) {
	if enforcePermissions {
		locator := req.URL.Path[1:] // strip leading slash
		if err := VerifySignature(locator, GetApiToken(req)); err != nil {
			http.Error(resp, err.Error(), err.(*KeepError).HTTPCode)
			return
		}
	}

	block, err := GetBlock(mux.Vars(req)["hash"])
	if err != nil {
		// This type assertion is safe because the only errors
		// GetBlock can return are DiskHashError or NotFoundError.
		http.Error(resp, err.Error(), err.(*KeepError).HTTPCode)
		return
	}
	defer bufs.Put(block)

	resp.Header().Set("Content-Length", strconv.Itoa(len(block)))
	resp.Header().Set("Content-Type", "application/octet-stream")
	resp.Write(block)
}

// PutBlockHandler is a HandleFunc to address Put block requests.
func PutBlockHandler(resp http.ResponseWriter, req *http.Request) {
	hash := mux.Vars(req)["hash"]

	// Detect as many error conditions as possible before reading
	// the body: avoid transmitting data that will not end up
	// being written anyway.

	if req.ContentLength == -1 {
		http.Error(resp, SizeRequiredError.Error(), SizeRequiredError.HTTPCode)
		return
	}

	if req.ContentLength > BlockSize {
		http.Error(resp, TooLongError.Error(), TooLongError.HTTPCode)
		return
	}

	if len(KeepVM.AllWritable()) == 0 {
		http.Error(resp, FullError.Error(), FullError.HTTPCode)
		return
	}

	buf := bufs.Get(int(req.ContentLength))
	_, err := io.ReadFull(req.Body, buf)
	if err != nil {
		http.Error(resp, err.Error(), 500)
		bufs.Put(buf)
		return
	}

	replication, err := PutBlock(buf, hash)
	bufs.Put(buf)

	if err != nil {
		ke := err.(*KeepError)
		http.Error(resp, ke.Error(), ke.HTTPCode)
		return
	}

	// Success; add a size hint, sign the locator if possible, and
	// return it to the client.
	returnHash := fmt.Sprintf("%s+%d", hash, req.ContentLength)
	apiToken := GetApiToken(req)
	if PermissionSecret != nil && apiToken != "" {
		expiry := time.Now().Add(blobSignatureTTL)
		returnHash = SignLocator(returnHash, apiToken, expiry)
	}
	resp.Header().Set("X-Keep-Replicas-Stored", strconv.Itoa(replication))
	resp.Write([]byte(returnHash + "\n"))
}

// IndexHandler is a HandleFunc to address /index and /index/{prefix} requests.
func IndexHandler(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !IsDataManagerToken(GetApiToken(req)) {
		http.Error(resp, UnauthorizedError.Error(), UnauthorizedError.HTTPCode)
		return
	}

	prefix := mux.Vars(req)["prefix"]

	for _, vol := range KeepVM.AllReadable() {
		if err := vol.IndexTo(prefix, resp); err != nil {
			// The only errors returned by IndexTo are
			// write errors returned by resp.Write(),
			// which probably means the client has
			// disconnected and this error will never be
			// reported to the client -- but it will
			// appear in our own error log.
			http.Error(resp, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	// An empty line at EOF is the only way the client can be
	// assured the entire index was received.
	resp.Write([]byte{'\n'})
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

// PoolStatus struct
type PoolStatus struct {
	Alloc uint64 `json:"BytesAllocated"`
	Cap   int    `json:"BuffersMax"`
	Len   int    `json:"BuffersInUse"`
}

// NodeStatus struct
type NodeStatus struct {
	Volumes    []*VolumeStatus `json:"volumes"`
	BufferPool PoolStatus
	PullQueue  WorkQueueStatus
	TrashQueue WorkQueueStatus
	Memory     runtime.MemStats
}

var st NodeStatus
var stLock sync.Mutex

// StatusHandler addresses /status.json requests.
func StatusHandler(resp http.ResponseWriter, req *http.Request) {
	stLock.Lock()
	readNodeStatus(&st)
	jstat, err := json.Marshal(&st)
	stLock.Unlock()
	if err == nil {
		resp.Write(jstat)
	} else {
		log.Printf("json.Marshal: %s", err)
		log.Printf("NodeStatus = %v", &st)
		http.Error(resp, err.Error(), 500)
	}
}

// populate the given NodeStatus struct with current values.
func readNodeStatus(st *NodeStatus) {
	vols := KeepVM.AllReadable()
	if cap(st.Volumes) < len(vols) {
		st.Volumes = make([]*VolumeStatus, len(vols))
	}
	st.Volumes = st.Volumes[:0]
	for _, vol := range vols {
		if s := vol.Status(); s != nil {
			st.Volumes = append(st.Volumes, s)
		}
	}
	st.BufferPool.Alloc = bufs.Alloc()
	st.BufferPool.Cap = bufs.Cap()
	st.BufferPool.Len = bufs.Len()
	st.PullQueue = getWorkQueueStatus(pullq)
	st.TrashQueue = getWorkQueueStatus(trashq)
	runtime.ReadMemStats(&st.Memory)
}

// return a WorkQueueStatus for the given queue. If q is nil (which
// should never happen except in test suites), return a zero status
// value instead of crashing.
func getWorkQueueStatus(q *WorkQueue) WorkQueueStatus {
	if q == nil {
		// This should only happen during tests.
		return WorkQueueStatus{}
	}
	return q.Status()
}

// DeleteHandler processes DELETE requests.
//
// DELETE /{hash:[0-9a-f]{32} will delete the block with the specified hash
// from all connected volumes.
//
// Only the Data Manager, or an Arvados admin with scope "all", are
// allowed to issue DELETE requests.  If a DELETE request is not
// authenticated or is issued by a non-admin user, the server returns
// a PermissionError.
//
// Upon receiving a valid request from an authorized user,
// DeleteHandler deletes all copies of the specified block on local
// writable volumes.
//
// Response format:
//
// If the requested blocks was not found on any volume, the response
// code is HTTP 404 Not Found.
//
// Otherwise, the response code is 200 OK, with a response body
// consisting of the JSON message
//
//    {"copies_deleted":d,"copies_failed":f}
//
// where d and f are integers representing the number of blocks that
// were successfully and unsuccessfully deleted.
//
func DeleteHandler(resp http.ResponseWriter, req *http.Request) {
	hash := mux.Vars(req)["hash"]

	// Confirm that this user is an admin and has a token with unlimited scope.
	var tok = GetApiToken(req)
	if tok == "" || !CanDelete(tok) {
		http.Error(resp, PermissionError.Error(), PermissionError.HTTPCode)
		return
	}

	if neverDelete {
		http.Error(resp, MethodDisabledError.Error(), MethodDisabledError.HTTPCode)
		return
	}

	// Delete copies of this block from all available volumes.
	// Report how many blocks were successfully deleted, and how
	// many were found on writable volumes but not deleted.
	var result struct {
		Deleted int `json:"copies_deleted"`
		Failed  int `json:"copies_failed"`
	}
	for _, vol := range KeepVM.AllWritable() {
		if err := vol.Trash(hash); err == nil {
			result.Deleted++
		} else if os.IsNotExist(err) {
			continue
		} else {
			result.Failed++
			log.Println("DeleteHandler:", err)
		}
	}

	var st int

	if result.Deleted == 0 && result.Failed == 0 {
		st = http.StatusNotFound
	} else {
		st = http.StatusOK
	}

	resp.WriteHeader(st)

	if st == http.StatusOK {
		if body, err := json.Marshal(result); err == nil {
			resp.Write(body)
		} else {
			log.Printf("json.Marshal: %s (result = %v)", err, result)
			http.Error(resp, err.Error(), 500)
		}
	}
}

/* PullHandler processes "PUT /pull" requests for the data manager.
   The request body is a JSON message containing a list of pull
   requests in the following format:

   [
      {
         "locator":"e4d909c290d0fb1ca068ffaddf22cbd0+4985",
         "servers":[
			"keep0.qr1hi.arvadosapi.com:25107",
			"keep1.qr1hi.arvadosapi.com:25108"
		 ]
	  },
	  {
		 "locator":"55ae4d45d2db0793d53f03e805f656e5+658395",
		 "servers":[
			"10.0.1.5:25107",
			"10.0.1.6:25107",
			"10.0.1.7:25108"
		 ]
	  },
	  ...
   ]

   Each pull request in the list consists of a block locator string
   and an ordered list of servers.  Keepstore should try to fetch the
   block from each server in turn.

   If the request has not been sent by the Data Manager, return 401
   Unauthorized.

   If the JSON unmarshalling fails, return 400 Bad Request.
*/

// PullRequest consists of a block locator and an ordered list of servers
type PullRequest struct {
	Locator string   `json:"locator"`
	Servers []string `json:"servers"`
}

// PullHandler processes "PUT /pull" requests for the data manager.
func PullHandler(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !IsDataManagerToken(GetApiToken(req)) {
		http.Error(resp, UnauthorizedError.Error(), UnauthorizedError.HTTPCode)
		return
	}

	// Parse the request body.
	var pr []PullRequest
	r := json.NewDecoder(req.Body)
	if err := r.Decode(&pr); err != nil {
		http.Error(resp, err.Error(), BadRequestError.HTTPCode)
		return
	}

	// We have a properly formatted pull list sent from the data
	// manager.  Report success and send the list to the pull list
	// manager for further handling.
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(
		fmt.Sprintf("Received %d pull requests\n", len(pr))))

	plist := list.New()
	for _, p := range pr {
		plist.PushBack(p)
	}
	pullq.ReplaceQueue(plist)
}

// TrashRequest consists of a block locator and it's Mtime
type TrashRequest struct {
	Locator    string `json:"locator"`
	BlockMtime int64  `json:"block_mtime"`
}

// TrashHandler processes /trash requests.
func TrashHandler(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !IsDataManagerToken(GetApiToken(req)) {
		http.Error(resp, UnauthorizedError.Error(), UnauthorizedError.HTTPCode)
		return
	}

	// Parse the request body.
	var trash []TrashRequest
	r := json.NewDecoder(req.Body)
	if err := r.Decode(&trash); err != nil {
		http.Error(resp, err.Error(), BadRequestError.HTTPCode)
		return
	}

	// We have a properly formatted trash list sent from the data
	// manager.  Report success and send the list to the trash work
	// queue for further handling.
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(
		fmt.Sprintf("Received %d trash requests\n", len(trash))))

	tlist := list.New()
	for _, t := range trash {
		tlist.PushBack(t)
	}
	trashq.ReplaceQueue(tlist)
}

// UntrashHandler processes "PUT /untrash/{hash:[0-9a-f]{32}}" requests for the data manager.
func UntrashHandler(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !IsDataManagerToken(GetApiToken(req)) {
		http.Error(resp, UnauthorizedError.Error(), UnauthorizedError.HTTPCode)
		return
	}

	hash := mux.Vars(req)["hash"]

	if len(KeepVM.AllWritable()) == 0 {
		http.Error(resp, "No writable volumes", http.StatusNotFound)
		return
	}

	var untrashedOn, failedOn []string
	var numNotFound int
	for _, vol := range KeepVM.AllWritable() {
		err := vol.Untrash(hash)

		if os.IsNotExist(err) {
			numNotFound++
		} else if err != nil {
			log.Printf("Error untrashing %v on volume %v", hash, vol.String())
			failedOn = append(failedOn, vol.String())
		} else {
			log.Printf("Untrashed %v on volume %v", hash, vol.String())
			untrashedOn = append(untrashedOn, vol.String())
		}
	}

	if numNotFound == len(KeepVM.AllWritable()) {
		http.Error(resp, "Block not found on any of the writable volumes", http.StatusNotFound)
		return
	}

	if len(failedOn) == len(KeepVM.AllWritable()) {
		http.Error(resp, "Failed to untrash on all writable volumes", http.StatusInternalServerError)
	} else {
		respBody := "Successfully untrashed on: " + strings.Join(untrashedOn, ",")
		if len(failedOn) > 0 {
			respBody += "; Failed to untrash on: " + strings.Join(failedOn, ",")
		}
		resp.Write([]byte(respBody))
	}
}

// ==============================
// GetBlock and PutBlock implement lower-level code for handling
// blocks by rooting through volumes connected to the local machine.
// Once the handler has determined that system policy permits the
// request, it calls these methods to perform the actual operation.
//
// TODO(twp): this code would probably be better located in the
// VolumeManager interface. As an abstraction, the VolumeManager
// should be the only part of the code that cares about which volume a
// block is stored on, so it should be responsible for figuring out
// which volume to check for fetching blocks, storing blocks, etc.
// ==============================

// GetBlock fetches and returns the block identified by "hash".
//
// On success, GetBlock returns a byte slice with the block data, and
// a nil error.
//
// If the block cannot be found on any volume, returns NotFoundError.
//
// If the block found does not have the correct MD5 hash, returns
// DiskHashError.
//
func GetBlock(hash string) ([]byte, error) {
	// Attempt to read the requested hash from a keep volume.
	errorToCaller := NotFoundError

	for _, vol := range KeepVM.AllReadable() {
		buf, err := vol.Get(hash)
		if err != nil {
			// IsNotExist is an expected error and may be
			// ignored. All other errors are logged. In
			// any case we continue trying to read other
			// volumes. If all volumes report IsNotExist,
			// we return a NotFoundError.
			if !os.IsNotExist(err) {
				log.Printf("%s: Get(%s): %s", vol, hash, err)
			}
			continue
		}
		// Check the file checksum.
		//
		filehash := fmt.Sprintf("%x", md5.Sum(buf))
		if filehash != hash {
			// TODO: Try harder to tell a sysadmin about
			// this.
			log.Printf("%s: checksum mismatch for request %s (actual %s)",
				vol, hash, filehash)
			errorToCaller = DiskHashError
			bufs.Put(buf)
			continue
		}
		if errorToCaller == DiskHashError {
			log.Printf("%s: checksum mismatch for request %s but a good copy was found on another volume and returned",
				vol, hash)
		}
		return buf, nil
	}
	return nil, errorToCaller
}

// PutBlock Stores the BLOCK (identified by the content id HASH) in Keep.
//
// PutBlock(block, hash)
//   Stores the BLOCK (identified by the content id HASH) in Keep.
//
//   The MD5 checksum of the block must be identical to the content id HASH.
//   If not, an error is returned.
//
//   PutBlock stores the BLOCK on the first Keep volume with free space.
//   A failure code is returned to the user only if all volumes fail.
//
//   On success, PutBlock returns nil.
//   On failure, it returns a KeepError with one of the following codes:
//
//   500 Collision
//          A different block with the same hash already exists on this
//          Keep server.
//   422 MD5Fail
//          The MD5 hash of the BLOCK does not match the argument HASH.
//   503 Full
//          There was not enough space left in any Keep volume to store
//          the object.
//   500 Fail
//          The object could not be stored for some other reason (e.g.
//          all writes failed). The text of the error message should
//          provide as much detail as possible.
//
func PutBlock(block []byte, hash string) (int, error) {
	// Check that BLOCK's checksum matches HASH.
	blockhash := fmt.Sprintf("%x", md5.Sum(block))
	if blockhash != hash {
		log.Printf("%s: MD5 checksum %s did not match request", hash, blockhash)
		return 0, RequestHashError
	}

	// If we already have this data, it's intact on disk, and we
	// can update its timestamp, return success. If we have
	// different data with the same hash, return failure.
	if n, err := CompareAndTouch(hash, block); err == nil || err == CollisionError {
		return n, err
	}

	// Choose a Keep volume to write to.
	// If this volume fails, try all of the volumes in order.
	if vol := KeepVM.NextWritable(); vol != nil {
		if err := vol.Put(hash, block); err == nil {
			return vol.Replication(), nil // success!
		}
	}

	writables := KeepVM.AllWritable()
	if len(writables) == 0 {
		log.Print("No writable volumes.")
		return 0, FullError
	}

	allFull := true
	for _, vol := range writables {
		err := vol.Put(hash, block)
		if err == nil {
			return vol.Replication(), nil // success!
		}
		if err != FullError {
			// The volume is not full but the
			// write did not succeed.  Report the
			// error and continue trying.
			allFull = false
			log.Printf("%s: Write(%s): %s", vol, hash, err)
		}
	}

	if allFull {
		log.Print("All volumes are full.")
		return 0, FullError
	}
	// Already logged the non-full errors.
	return 0, GenericError
}

// CompareAndTouch returns the current replication level if one of the
// volumes already has the given content and it successfully updates
// the relevant block's modification time in order to protect it from
// premature garbage collection. Otherwise, it returns a non-nil
// error.
func CompareAndTouch(hash string, buf []byte) (int, error) {
	var bestErr error = NotFoundError
	for _, vol := range KeepVM.AllWritable() {
		if err := vol.Compare(hash, buf); err == CollisionError {
			// Stop if we have a block with same hash but
			// different content. (It will be impossible
			// to tell which one is wanted if we have
			// both, so there's no point writing it even
			// on a different volume.)
			log.Printf("%s: Compare(%s): %s", vol, hash, err)
			return 0, err
		} else if os.IsNotExist(err) {
			// Block does not exist. This is the only
			// "normal" error: we don't log anything.
			continue
		} else if err != nil {
			// Couldn't open file, data is corrupt on
			// disk, etc.: log this abnormal condition,
			// and try the next volume.
			log.Printf("%s: Compare(%s): %s", vol, hash, err)
			continue
		}
		if err := vol.Touch(hash); err != nil {
			log.Printf("%s: Touch %s failed: %s", vol, hash, err)
			bestErr = err
			continue
		}
		// Compare and Touch both worked --> done.
		return vol.Replication(), nil
	}
	return 0, bestErr
}

var validLocatorRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// IsValidLocator returns true if the specified string is a valid Keep locator.
//   When Keep is extended to support hash types other than MD5,
//   this should be updated to cover those as well.
//
func IsValidLocator(loc string) bool {
	return validLocatorRe.MatchString(loc)
}

var authRe = regexp.MustCompile(`^OAuth2\s+(.*)`)

// GetApiToken returns the OAuth2 token from the Authorization
// header of a HTTP request, or an empty string if no matching
// token is found.
func GetApiToken(req *http.Request) string {
	if auth, ok := req.Header["Authorization"]; ok {
		if match := authRe.FindStringSubmatch(auth[0]); match != nil {
			return match[1]
		}
	}
	return ""
}

// IsExpired returns true if the given Unix timestamp (expressed as a
// hexadecimal string) is in the past, or if timestampHex cannot be
// parsed as a hexadecimal string.
func IsExpired(timestampHex string) bool {
	ts, err := strconv.ParseInt(timestampHex, 16, 0)
	if err != nil {
		log.Printf("IsExpired: %s", err)
		return true
	}
	return time.Unix(ts, 0).Before(time.Now())
}

// CanDelete returns true if the user identified by apiToken is
// allowed to delete blocks.
func CanDelete(apiToken string) bool {
	if apiToken == "" {
		return false
	}
	// Blocks may be deleted only when Keep has been configured with a
	// data manager.
	if IsDataManagerToken(apiToken) {
		return true
	}
	// TODO(twp): look up apiToken with the API server
	// return true if is_admin is true and if the token
	// has unlimited scope
	return false
}

// IsDataManagerToken returns true if apiToken represents the data
// manager's token.
func IsDataManagerToken(apiToken string) bool {
	return dataManagerToken != "" && apiToken == dataManagerToken
}
