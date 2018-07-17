// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

// REST handlers for Keep are implemented here.
//
// GetBlockHandler (GET /locator)
// PutBlockHandler (PUT /locator)
// IndexHandler    (GET /index, GET /index/prefix)
// StatusHandler   (GET /status.json)

import (
	"container/list"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type router struct {
	*mux.Router
	limiter httpserver.RequestCounter
}

// MakeRESTRouter returns a new router that forwards all Keep requests
// to the appropriate handlers.
func MakeRESTRouter() http.Handler {
	rtr := &router{Router: mux.NewRouter()}

	rtr.HandleFunc(
		`/{hash:[0-9a-f]{32}}`, GetBlockHandler).Methods("GET", "HEAD")
	rtr.HandleFunc(
		`/{hash:[0-9a-f]{32}}+{hints}`,
		GetBlockHandler).Methods("GET", "HEAD")

	rtr.HandleFunc(`/{hash:[0-9a-f]{32}}`, PutBlockHandler).Methods("PUT")
	rtr.HandleFunc(`/{hash:[0-9a-f]{32}}`, DeleteHandler).Methods("DELETE")
	// List all blocks stored here. Privileged client only.
	rtr.HandleFunc(`/index`, rtr.IndexHandler).Methods("GET", "HEAD")
	// List blocks stored here whose hash has the given prefix.
	// Privileged client only.
	rtr.HandleFunc(`/index/{prefix:[0-9a-f]{0,32}}`, rtr.IndexHandler).Methods("GET", "HEAD")

	// Internals/debugging info (runtime.MemStats)
	rtr.HandleFunc(`/debug.json`, rtr.DebugHandler).Methods("GET", "HEAD")

	// List volumes: path, device number, bytes used/avail.
	rtr.HandleFunc(`/status.json`, rtr.StatusHandler).Methods("GET", "HEAD")

	// List mounts: UUID, readonly, tier, device ID, ...
	rtr.HandleFunc(`/mounts`, rtr.MountsHandler).Methods("GET")
	rtr.HandleFunc(`/mounts/{uuid}/blocks`, rtr.IndexHandler).Methods("GET")
	rtr.HandleFunc(`/mounts/{uuid}/blocks/`, rtr.IndexHandler).Methods("GET")

	// Replace the current pull queue.
	rtr.HandleFunc(`/pull`, PullHandler).Methods("PUT")

	// Replace the current trash queue.
	rtr.HandleFunc(`/trash`, TrashHandler).Methods("PUT")

	// Untrash moves blocks from trash back into store
	rtr.HandleFunc(`/untrash/{hash:[0-9a-f]{32}}`, UntrashHandler).Methods("PUT")

	rtr.Handle("/_health/{check}", &health.Handler{
		Token:  theConfig.ManagementToken,
		Prefix: "/_health/",
	}).Methods("GET")

	// Any request which does not match any of these routes gets
	// 400 Bad Request.
	rtr.NotFoundHandler = http.HandlerFunc(BadRequestHandler)

	theConfig.metrics.setup()

	rtr.limiter = httpserver.NewRequestLimiter(theConfig.MaxRequests, rtr)

	mux := http.NewServeMux()
	mux.Handle("/", theConfig.metrics.Instrument(
		httpserver.AddRequestIDs(httpserver.LogRequests(nil, rtr.limiter))))
	mux.HandleFunc("/metrics.json", theConfig.metrics.exportJSON)
	mux.Handle("/metrics", theConfig.metrics.exportProm)

	return mux
}

// BadRequestHandler is a HandleFunc to address bad requests.
func BadRequestHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, BadRequestError.Error(), BadRequestError.HTTPCode)
}

// GetBlockHandler is a HandleFunc to address Get block requests.
func GetBlockHandler(resp http.ResponseWriter, req *http.Request) {
	ctx, cancel := contextForResponse(context.TODO(), resp)
	defer cancel()

	if theConfig.RequireSignatures {
		locator := req.URL.Path[1:] // strip leading slash
		if err := VerifySignature(locator, GetAPIToken(req)); err != nil {
			http.Error(resp, err.Error(), err.(*KeepError).HTTPCode)
			return
		}
	}

	// TODO: Probe volumes to check whether the block _might_
	// exist. Some volumes/types could support a quick existence
	// check without causing other operations to suffer. If all
	// volumes support that, and assure us the block definitely
	// isn't here, we can return 404 now instead of waiting for a
	// buffer.

	buf, err := getBufferWithContext(ctx, bufs, BlockSize)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer bufs.Put(buf)

	size, err := GetBlock(ctx, mux.Vars(req)["hash"], buf, resp)
	if err != nil {
		code := http.StatusInternalServerError
		if err, ok := err.(*KeepError); ok {
			code = err.HTTPCode
		}
		http.Error(resp, err.Error(), code)
		return
	}

	resp.Header().Set("Content-Length", strconv.Itoa(size))
	resp.Header().Set("Content-Type", "application/octet-stream")
	resp.Write(buf[:size])
}

// Return a new context that gets cancelled by resp's CloseNotifier.
func contextForResponse(parent context.Context, resp http.ResponseWriter) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	if cn, ok := resp.(http.CloseNotifier); ok {
		go func(c <-chan bool) {
			select {
			case <-c:
				theConfig.debugLogf("cancel context")
				cancel()
			case <-ctx.Done():
			}
		}(cn.CloseNotify())
	}
	return ctx, cancel
}

// Get a buffer from the pool -- but give up and return a non-nil
// error if ctx ends before we get a buffer.
func getBufferWithContext(ctx context.Context, bufs *bufferPool, bufSize int) ([]byte, error) {
	bufReady := make(chan []byte)
	go func() {
		bufReady <- bufs.Get(bufSize)
	}()
	select {
	case buf := <-bufReady:
		return buf, nil
	case <-ctx.Done():
		go func() {
			// Even if closeNotifier happened first, we
			// need to keep waiting for our buf so we can
			// return it to the pool.
			bufs.Put(<-bufReady)
		}()
		return nil, ErrClientDisconnect
	}
}

// PutBlockHandler is a HandleFunc to address Put block requests.
func PutBlockHandler(resp http.ResponseWriter, req *http.Request) {
	ctx, cancel := contextForResponse(context.TODO(), resp)
	defer cancel()

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

	buf, err := getBufferWithContext(ctx, bufs, int(req.ContentLength))
	if err != nil {
		http.Error(resp, err.Error(), http.StatusServiceUnavailable)
		return
	}

	_, err = io.ReadFull(req.Body, buf)
	if err != nil {
		http.Error(resp, err.Error(), 500)
		bufs.Put(buf)
		return
	}

	replication, err := PutBlock(ctx, buf, hash)
	bufs.Put(buf)

	if err != nil {
		code := http.StatusInternalServerError
		if err, ok := err.(*KeepError); ok {
			code = err.HTTPCode
		}
		http.Error(resp, err.Error(), code)
		return
	}

	// Success; add a size hint, sign the locator if possible, and
	// return it to the client.
	returnHash := fmt.Sprintf("%s+%d", hash, req.ContentLength)
	apiToken := GetAPIToken(req)
	if theConfig.blobSigningKey != nil && apiToken != "" {
		expiry := time.Now().Add(theConfig.BlobSignatureTTL.Duration())
		returnHash = SignLocator(returnHash, apiToken, expiry)
	}
	resp.Header().Set("X-Keep-Replicas-Stored", strconv.Itoa(replication))
	resp.Write([]byte(returnHash + "\n"))
}

// IndexHandler responds to "/index", "/index/{prefix}", and
// "/mounts/{uuid}/blocks" requests.
func (rtr *router) IndexHandler(resp http.ResponseWriter, req *http.Request) {
	if !IsSystemAuth(GetAPIToken(req)) {
		http.Error(resp, UnauthorizedError.Error(), UnauthorizedError.HTTPCode)
		return
	}

	prefix := mux.Vars(req)["prefix"]
	if prefix == "" {
		req.ParseForm()
		prefix = req.Form.Get("prefix")
	}

	uuid := mux.Vars(req)["uuid"]

	var vols []Volume
	if uuid == "" {
		vols = KeepVM.AllReadable()
	} else if v := KeepVM.Lookup(uuid, false); v == nil {
		http.Error(resp, "mount not found", http.StatusNotFound)
		return
	} else {
		vols = []Volume{v}
	}

	for _, v := range vols {
		if err := v.IndexTo(prefix, resp); err != nil {
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

// MountsHandler responds to "GET /mounts" requests.
func (rtr *router) MountsHandler(resp http.ResponseWriter, req *http.Request) {
	err := json.NewEncoder(resp).Encode(KeepVM.Mounts())
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}
}

// PoolStatus struct
type PoolStatus struct {
	Alloc uint64 `json:"BytesAllocatedCumulative"`
	Cap   int    `json:"BuffersMax"`
	Len   int    `json:"BuffersInUse"`
}

type volumeStatusEnt struct {
	Label         string
	Status        *VolumeStatus `json:",omitempty"`
	VolumeStats   *ioStats      `json:",omitempty"`
	InternalStats interface{}   `json:",omitempty"`
}

// NodeStatus struct
type NodeStatus struct {
	Volumes         []*volumeStatusEnt
	BufferPool      PoolStatus
	PullQueue       WorkQueueStatus
	TrashQueue      WorkQueueStatus
	RequestsCurrent int
	RequestsMax     int
	Version         string
}

var st NodeStatus
var stLock sync.Mutex

// DebugHandler addresses /debug.json requests.
func (rtr *router) DebugHandler(resp http.ResponseWriter, req *http.Request) {
	type debugStats struct {
		MemStats runtime.MemStats
	}
	var ds debugStats
	runtime.ReadMemStats(&ds.MemStats)
	err := json.NewEncoder(resp).Encode(&ds)
	if err != nil {
		http.Error(resp, err.Error(), 500)
	}
}

// StatusHandler addresses /status.json requests.
func (rtr *router) StatusHandler(resp http.ResponseWriter, req *http.Request) {
	stLock.Lock()
	rtr.readNodeStatus(&st)
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
func (rtr *router) readNodeStatus(st *NodeStatus) {
	st.Version = version
	vols := KeepVM.AllReadable()
	if cap(st.Volumes) < len(vols) {
		st.Volumes = make([]*volumeStatusEnt, len(vols))
	}
	st.Volumes = st.Volumes[:0]
	for _, vol := range vols {
		var internalStats interface{}
		if vol, ok := vol.(InternalStatser); ok {
			internalStats = vol.InternalStats()
		}
		st.Volumes = append(st.Volumes, &volumeStatusEnt{
			Label:         vol.String(),
			Status:        vol.Status(),
			InternalStats: internalStats,
			//VolumeStats: KeepVM.VolumeStats(vol),
		})
	}
	st.BufferPool.Alloc = bufs.Alloc()
	st.BufferPool.Cap = bufs.Cap()
	st.BufferPool.Len = bufs.Len()
	st.PullQueue = getWorkQueueStatus(pullq)
	st.TrashQueue = getWorkQueueStatus(trashq)
	if rtr.limiter != nil {
		st.RequestsCurrent = rtr.limiter.Current()
		st.RequestsMax = rtr.limiter.Max()
	}
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
	var tok = GetAPIToken(req)
	if tok == "" || !CanDelete(tok) {
		http.Error(resp, PermissionError.Error(), PermissionError.HTTPCode)
		return
	}

	if !theConfig.EnableDelete {
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

	// Destination mount, or "" for "anywhere"
	MountUUID string `json:"mount_uuid"`
}

// PullHandler processes "PUT /pull" requests for the data manager.
func PullHandler(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !IsSystemAuth(GetAPIToken(req)) {
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

// TrashRequest consists of a block locator and its Mtime
type TrashRequest struct {
	Locator    string `json:"locator"`
	BlockMtime int64  `json:"block_mtime"`

	// Target mount, or "" for "everywhere"
	MountUUID string `json:"mount_uuid"`
}

// TrashHandler processes /trash requests.
func TrashHandler(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !IsSystemAuth(GetAPIToken(req)) {
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
	if !IsSystemAuth(GetAPIToken(req)) {
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

// GetBlock fetches the block identified by "hash" into the provided
// buf, and returns the data size.
//
// If the block cannot be found on any volume, returns NotFoundError.
//
// If the block found does not have the correct MD5 hash, returns
// DiskHashError.
//
func GetBlock(ctx context.Context, hash string, buf []byte, resp http.ResponseWriter) (int, error) {
	// Attempt to read the requested hash from a keep volume.
	errorToCaller := NotFoundError

	for _, vol := range KeepVM.AllReadable() {
		size, err := vol.Get(ctx, hash, buf)
		select {
		case <-ctx.Done():
			return 0, ErrClientDisconnect
		default:
		}
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
		filehash := fmt.Sprintf("%x", md5.Sum(buf[:size]))
		if filehash != hash {
			// TODO: Try harder to tell a sysadmin about
			// this.
			log.Printf("%s: checksum mismatch for request %s (actual %s)",
				vol, hash, filehash)
			errorToCaller = DiskHashError
			continue
		}
		if errorToCaller == DiskHashError {
			log.Printf("%s: checksum mismatch for request %s but a good copy was found on another volume and returned",
				vol, hash)
		}
		return size, nil
	}
	return 0, errorToCaller
}

// PutBlock Stores the BLOCK (identified by the content id HASH) in Keep.
//
// PutBlock(ctx, block, hash)
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
func PutBlock(ctx context.Context, block []byte, hash string) (int, error) {
	// Check that BLOCK's checksum matches HASH.
	blockhash := fmt.Sprintf("%x", md5.Sum(block))
	if blockhash != hash {
		log.Printf("%s: MD5 checksum %s did not match request", hash, blockhash)
		return 0, RequestHashError
	}

	// If we already have this data, it's intact on disk, and we
	// can update its timestamp, return success. If we have
	// different data with the same hash, return failure.
	if n, err := CompareAndTouch(ctx, hash, block); err == nil || err == CollisionError {
		return n, err
	} else if ctx.Err() != nil {
		return 0, ErrClientDisconnect
	}

	// Choose a Keep volume to write to.
	// If this volume fails, try all of the volumes in order.
	if vol := KeepVM.NextWritable(); vol != nil {
		if err := vol.Put(ctx, hash, block); err == nil {
			return vol.Replication(), nil // success!
		}
		if ctx.Err() != nil {
			return 0, ErrClientDisconnect
		}
	}

	writables := KeepVM.AllWritable()
	if len(writables) == 0 {
		log.Print("No writable volumes.")
		return 0, FullError
	}

	allFull := true
	for _, vol := range writables {
		err := vol.Put(ctx, hash, block)
		if ctx.Err() != nil {
			return 0, ErrClientDisconnect
		}
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
func CompareAndTouch(ctx context.Context, hash string, buf []byte) (int, error) {
	var bestErr error = NotFoundError
	for _, vol := range KeepVM.AllWritable() {
		err := vol.Compare(ctx, hash, buf)
		if ctx.Err() != nil {
			return 0, ctx.Err()
		} else if err == CollisionError {
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

// GetAPIToken returns the OAuth2 token from the Authorization
// header of a HTTP request, or an empty string if no matching
// token is found.
func GetAPIToken(req *http.Request) string {
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
	if IsSystemAuth(apiToken) {
		return true
	}
	// TODO(twp): look up apiToken with the API server
	// return true if is_admin is true and if the token
	// has unlimited scope
	return false
}

// IsSystemAuth returns true if the given token is allowed to perform
// system level actions like deleting data.
func IsSystemAuth(token string) bool {
	return token != "" && token == theConfig.systemAuthToken
}
