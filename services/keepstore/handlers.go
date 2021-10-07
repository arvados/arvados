// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

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
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type router struct {
	*mux.Router
	cluster     *arvados.Cluster
	logger      logrus.FieldLogger
	remoteProxy remoteProxy
	metrics     *nodeMetrics
	volmgr      *RRVolumeManager
	pullq       *WorkQueue
	trashq      *WorkQueue
}

// MakeRESTRouter returns a new router that forwards all Keep requests
// to the appropriate handlers.
func MakeRESTRouter(ctx context.Context, cluster *arvados.Cluster, reg *prometheus.Registry, volmgr *RRVolumeManager, pullq, trashq *WorkQueue) http.Handler {
	rtr := &router{
		Router:  mux.NewRouter(),
		cluster: cluster,
		logger:  ctxlog.FromContext(ctx),
		metrics: &nodeMetrics{reg: reg},
		volmgr:  volmgr,
		pullq:   pullq,
		trashq:  trashq,
	}

	rtr.HandleFunc(
		`/{hash:[0-9a-f]{32}}`, rtr.handleGET).Methods("GET", "HEAD")
	rtr.HandleFunc(
		`/{hash:[0-9a-f]{32}}+{hints}`,
		rtr.handleGET).Methods("GET", "HEAD")

	rtr.HandleFunc(`/{hash:[0-9a-f]{32}}`, rtr.handlePUT).Methods("PUT")
	rtr.HandleFunc(`/{hash:[0-9a-f]{32}}`, rtr.handleDELETE).Methods("DELETE")
	// List all blocks stored here. Privileged client only.
	rtr.HandleFunc(`/index`, rtr.handleIndex).Methods("GET", "HEAD")
	// List blocks stored here whose hash has the given prefix.
	// Privileged client only.
	rtr.HandleFunc(`/index/{prefix:[0-9a-f]{0,32}}`, rtr.handleIndex).Methods("GET", "HEAD")
	// Update timestamp on existing block. Privileged client only.
	rtr.HandleFunc(`/{hash:[0-9a-f]{32}}`, rtr.handleTOUCH).Methods("TOUCH")

	// Internals/debugging info (runtime.MemStats)
	rtr.HandleFunc(`/debug.json`, rtr.DebugHandler).Methods("GET", "HEAD")

	// List volumes: path, device number, bytes used/avail.
	rtr.HandleFunc(`/status.json`, rtr.StatusHandler).Methods("GET", "HEAD")

	// List mounts: UUID, readonly, tier, device ID, ...
	rtr.HandleFunc(`/mounts`, rtr.MountsHandler).Methods("GET")
	rtr.HandleFunc(`/mounts/{uuid}/blocks`, rtr.handleIndex).Methods("GET")
	rtr.HandleFunc(`/mounts/{uuid}/blocks/`, rtr.handleIndex).Methods("GET")

	// Replace the current pull queue.
	rtr.HandleFunc(`/pull`, rtr.handlePull).Methods("PUT")

	// Replace the current trash queue.
	rtr.HandleFunc(`/trash`, rtr.handleTrash).Methods("PUT")

	// Untrash moves blocks from trash back into store
	rtr.HandleFunc(`/untrash/{hash:[0-9a-f]{32}}`, rtr.handleUntrash).Methods("PUT")

	rtr.Handle("/_health/{check}", &health.Handler{
		Token:  cluster.ManagementToken,
		Prefix: "/_health/",
	}).Methods("GET")

	// Any request which does not match any of these routes gets
	// 400 Bad Request.
	rtr.NotFoundHandler = http.HandlerFunc(BadRequestHandler)

	rtr.metrics.setupBufferPoolMetrics(bufs)
	rtr.metrics.setupWorkQueueMetrics(rtr.pullq, "pull")
	rtr.metrics.setupWorkQueueMetrics(rtr.trashq, "trash")

	return rtr
}

// BadRequestHandler is a HandleFunc to address bad requests.
func BadRequestHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, BadRequestError.Error(), BadRequestError.HTTPCode)
}

func (rtr *router) handleGET(resp http.ResponseWriter, req *http.Request) {
	ctx, cancel := contextForResponse(context.TODO(), resp)
	defer cancel()

	locator := req.URL.Path[1:]
	if strings.Contains(locator, "+R") && !strings.Contains(locator, "+A") {
		rtr.remoteProxy.Get(ctx, resp, req, rtr.cluster, rtr.volmgr)
		return
	}

	if rtr.cluster.Collections.BlobSigning {
		locator := req.URL.Path[1:] // strip leading slash
		if err := VerifySignature(rtr.cluster, locator, GetAPIToken(req)); err != nil {
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

	size, err := GetBlock(ctx, rtr.volmgr, mux.Vars(req)["hash"], buf, resp)
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

func (rtr *router) handleTOUCH(resp http.ResponseWriter, req *http.Request) {
	if !rtr.isSystemAuth(GetAPIToken(req)) {
		http.Error(resp, UnauthorizedError.Error(), UnauthorizedError.HTTPCode)
		return
	}
	hash := mux.Vars(req)["hash"]
	vols := rtr.volmgr.AllWritable()
	if len(vols) == 0 {
		http.Error(resp, "no volumes", http.StatusNotFound)
		return
	}
	var err error
	for _, mnt := range vols {
		err = mnt.Touch(hash)
		if err == nil {
			break
		}
	}
	switch {
	case err == nil:
		return
	case os.IsNotExist(err):
		http.Error(resp, err.Error(), http.StatusNotFound)
	default:
		http.Error(resp, err.Error(), http.StatusInternalServerError)
	}
}

func (rtr *router) handlePUT(resp http.ResponseWriter, req *http.Request) {
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

	if len(rtr.volmgr.AllWritable()) == 0 {
		http.Error(resp, FullError.Error(), FullError.HTTPCode)
		return
	}

	var wantStorageClasses []string
	if hdr := req.Header.Get("X-Keep-Storage-Classes"); hdr != "" {
		wantStorageClasses = strings.Split(hdr, ",")
		for i, sc := range wantStorageClasses {
			wantStorageClasses[i] = strings.TrimSpace(sc)
		}
	} else {
		// none specified -- use configured default
		for class, cfg := range rtr.cluster.StorageClasses {
			if cfg.Default {
				wantStorageClasses = append(wantStorageClasses, class)
			}
		}
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

	result, err := PutBlock(ctx, rtr.volmgr, buf, hash, wantStorageClasses)
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
	if rtr.cluster.Collections.BlobSigningKey != "" && apiToken != "" {
		expiry := time.Now().Add(rtr.cluster.Collections.BlobSigningTTL.Duration())
		returnHash = SignLocator(rtr.cluster, returnHash, apiToken, expiry)
	}
	resp.Header().Set("X-Keep-Replicas-Stored", result.TotalReplication())
	resp.Header().Set("X-Keep-Storage-Classes-Confirmed", result.ClassReplication())
	resp.Write([]byte(returnHash + "\n"))
}

// IndexHandler responds to "/index", "/index/{prefix}", and
// "/mounts/{uuid}/blocks" requests.
func (rtr *router) handleIndex(resp http.ResponseWriter, req *http.Request) {
	if !rtr.isSystemAuth(GetAPIToken(req)) {
		http.Error(resp, UnauthorizedError.Error(), UnauthorizedError.HTTPCode)
		return
	}

	prefix := mux.Vars(req)["prefix"]
	if prefix == "" {
		req.ParseForm()
		prefix = req.Form.Get("prefix")
	}

	uuid := mux.Vars(req)["uuid"]

	var vols []*VolumeMount
	if uuid == "" {
		vols = rtr.volmgr.AllReadable()
	} else if mnt := rtr.volmgr.Lookup(uuid, false); mnt == nil {
		http.Error(resp, "mount not found", http.StatusNotFound)
		return
	} else {
		vols = []*VolumeMount{mnt}
	}

	for _, v := range vols {
		if err := v.IndexTo(prefix, resp); err != nil {
			// We can't send an error status/message to
			// the client because IndexTo() might have
			// already written body content. All we can do
			// is log the error in our own logs.
			//
			// The client must notice the lack of trailing
			// newline as an indication that the response
			// is incomplete.
			ctxlog.FromContext(req.Context()).WithError(err).Errorf("truncating index response after error from volume %s", v)
			return
		}
	}
	// An empty line at EOF is the only way the client can be
	// assured the entire index was received.
	resp.Write([]byte{'\n'})
}

// MountsHandler responds to "GET /mounts" requests.
func (rtr *router) MountsHandler(resp http.ResponseWriter, req *http.Request) {
	err := json.NewEncoder(resp).Encode(rtr.volmgr.Mounts())
	if err != nil {
		httpserver.Error(resp, err.Error(), http.StatusInternalServerError)
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
	data, err := json.Marshal(&ds)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Write(data)
}

// StatusHandler addresses /status.json requests.
func (rtr *router) StatusHandler(resp http.ResponseWriter, req *http.Request) {
	stLock.Lock()
	rtr.readNodeStatus(&st)
	data, err := json.Marshal(&st)
	stLock.Unlock()
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Write(data)
}

// populate the given NodeStatus struct with current values.
func (rtr *router) readNodeStatus(st *NodeStatus) {
	st.Version = version
	vols := rtr.volmgr.AllReadable()
	if cap(st.Volumes) < len(vols) {
		st.Volumes = make([]*volumeStatusEnt, len(vols))
	}
	st.Volumes = st.Volumes[:0]
	for _, vol := range vols {
		var internalStats interface{}
		if vol, ok := vol.Volume.(InternalStatser); ok {
			internalStats = vol.InternalStats()
		}
		st.Volumes = append(st.Volumes, &volumeStatusEnt{
			Label:         vol.String(),
			Status:        vol.Status(),
			InternalStats: internalStats,
			//VolumeStats: rtr.volmgr.VolumeStats(vol),
		})
	}
	st.BufferPool.Alloc = bufs.Alloc()
	st.BufferPool.Cap = bufs.Cap()
	st.BufferPool.Len = bufs.Len()
	st.PullQueue = getWorkQueueStatus(rtr.pullq)
	st.TrashQueue = getWorkQueueStatus(rtr.trashq)
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

// handleDELETE processes DELETE requests.
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
// handleDELETE deletes all copies of the specified block on local
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
func (rtr *router) handleDELETE(resp http.ResponseWriter, req *http.Request) {
	hash := mux.Vars(req)["hash"]

	// Confirm that this user is an admin and has a token with unlimited scope.
	var tok = GetAPIToken(req)
	if tok == "" || !rtr.canDelete(tok) {
		http.Error(resp, PermissionError.Error(), PermissionError.HTTPCode)
		return
	}

	if !rtr.cluster.Collections.BlobTrash {
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
	for _, vol := range rtr.volmgr.AllWritable() {
		if err := vol.Trash(hash); err == nil {
			result.Deleted++
		} else if os.IsNotExist(err) {
			continue
		} else {
			result.Failed++
			ctxlog.FromContext(req.Context()).WithError(err).Errorf("Trash(%s) failed on volume %s", hash, vol)
		}
	}
	if result.Deleted == 0 && result.Failed == 0 {
		resp.WriteHeader(http.StatusNotFound)
		return
	}
	body, err := json.Marshal(result)
	if err != nil {
		http.Error(resp, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Write(body)
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
func (rtr *router) handlePull(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !rtr.isSystemAuth(GetAPIToken(req)) {
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
	rtr.pullq.ReplaceQueue(plist)
}

// TrashRequest consists of a block locator and its Mtime
type TrashRequest struct {
	Locator    string `json:"locator"`
	BlockMtime int64  `json:"block_mtime"`

	// Target mount, or "" for "everywhere"
	MountUUID string `json:"mount_uuid"`
}

// TrashHandler processes /trash requests.
func (rtr *router) handleTrash(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !rtr.isSystemAuth(GetAPIToken(req)) {
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
	rtr.trashq.ReplaceQueue(tlist)
}

// UntrashHandler processes "PUT /untrash/{hash:[0-9a-f]{32}}" requests for the data manager.
func (rtr *router) handleUntrash(resp http.ResponseWriter, req *http.Request) {
	// Reject unauthorized requests.
	if !rtr.isSystemAuth(GetAPIToken(req)) {
		http.Error(resp, UnauthorizedError.Error(), UnauthorizedError.HTTPCode)
		return
	}

	log := ctxlog.FromContext(req.Context())
	hash := mux.Vars(req)["hash"]

	if len(rtr.volmgr.AllWritable()) == 0 {
		http.Error(resp, "No writable volumes", http.StatusNotFound)
		return
	}

	var untrashedOn, failedOn []string
	var numNotFound int
	for _, vol := range rtr.volmgr.AllWritable() {
		err := vol.Untrash(hash)

		if os.IsNotExist(err) {
			numNotFound++
		} else if err != nil {
			log.WithError(err).Errorf("Error untrashing %v on volume %s", hash, vol)
			failedOn = append(failedOn, vol.String())
		} else {
			log.Infof("Untrashed %v on volume %v", hash, vol.String())
			untrashedOn = append(untrashedOn, vol.String())
		}
	}

	if numNotFound == len(rtr.volmgr.AllWritable()) {
		http.Error(resp, "Block not found on any of the writable volumes", http.StatusNotFound)
	} else if len(failedOn) == len(rtr.volmgr.AllWritable()) {
		http.Error(resp, "Failed to untrash on all writable volumes", http.StatusInternalServerError)
	} else {
		respBody := "Successfully untrashed on: " + strings.Join(untrashedOn, ", ")
		if len(failedOn) > 0 {
			respBody += "; Failed to untrash on: " + strings.Join(failedOn, ", ")
			http.Error(resp, respBody, http.StatusInternalServerError)
		} else {
			fmt.Fprintln(resp, respBody)
		}
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
func GetBlock(ctx context.Context, volmgr *RRVolumeManager, hash string, buf []byte, resp http.ResponseWriter) (int, error) {
	log := ctxlog.FromContext(ctx)

	// Attempt to read the requested hash from a keep volume.
	errorToCaller := NotFoundError

	for _, vol := range volmgr.AllReadable() {
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
				log.WithError(err).Errorf("Get(%s) failed on %s", hash, vol)
			}
			// If some volume returns a transient error, return it to the caller
			// instead of "Not found" so it can retry.
			if err == VolumeBusyError {
				errorToCaller = err.(*KeepError)
			}
			continue
		}
		// Check the file checksum.
		filehash := fmt.Sprintf("%x", md5.Sum(buf[:size]))
		if filehash != hash {
			// TODO: Try harder to tell a sysadmin about
			// this.
			log.Error("checksum mismatch for block %s (actual %s) on %s", hash, filehash, vol)
			errorToCaller = DiskHashError
			continue
		}
		if errorToCaller == DiskHashError {
			log.Warn("after checksum mismatch for block %s on a different volume, a good copy was found on volume %s and returned", hash, vol)
		}
		return size, nil
	}
	return 0, errorToCaller
}

type putProgress struct {
	classNeeded      map[string]bool
	classTodo        map[string]bool
	mountUsed        map[*VolumeMount]bool
	totalReplication int
	classDone        map[string]int
}

// Number of distinct replicas stored. "2" can mean the block was
// stored on 2 different volumes with replication 1, or on 1 volume
// with replication 2.
func (pr putProgress) TotalReplication() string {
	return strconv.Itoa(pr.totalReplication)
}

// Number of replicas satisfying each storage class, formatted like
// "default=2; special=1".
func (pr putProgress) ClassReplication() string {
	s := ""
	for k, v := range pr.classDone {
		if len(s) > 0 {
			s += ", "
		}
		s += k + "=" + strconv.Itoa(v)
	}
	return s
}

func (pr *putProgress) Add(mnt *VolumeMount) {
	if pr.mountUsed[mnt] {
		logrus.Warnf("BUG? superfluous extra write to mount %s", mnt.UUID)
		return
	}
	pr.mountUsed[mnt] = true
	pr.totalReplication += mnt.Replication
	for class := range mnt.StorageClasses {
		pr.classDone[class] += mnt.Replication
		delete(pr.classTodo, class)
	}
}

func (pr *putProgress) Sub(mnt *VolumeMount) {
	if !pr.mountUsed[mnt] {
		logrus.Warnf("BUG? Sub called with no prior matching Add: %s", mnt.UUID)
		return
	}
	pr.mountUsed[mnt] = false
	pr.totalReplication -= mnt.Replication
	for class := range mnt.StorageClasses {
		pr.classDone[class] -= mnt.Replication
		if pr.classNeeded[class] {
			pr.classTodo[class] = true
		}
	}
}

func (pr *putProgress) Done() bool {
	return len(pr.classTodo) == 0 && pr.totalReplication > 0
}

func (pr *putProgress) Want(mnt *VolumeMount) bool {
	if pr.Done() || pr.mountUsed[mnt] {
		return false
	}
	if len(pr.classTodo) == 0 {
		// none specified == "any"
		return true
	}
	for class := range mnt.StorageClasses {
		if pr.classTodo[class] {
			return true
		}
	}
	return false
}

func (pr *putProgress) Copy() *putProgress {
	cp := putProgress{
		classNeeded:      pr.classNeeded,
		classTodo:        make(map[string]bool, len(pr.classTodo)),
		classDone:        make(map[string]int, len(pr.classDone)),
		mountUsed:        make(map[*VolumeMount]bool, len(pr.mountUsed)),
		totalReplication: pr.totalReplication,
	}
	for k, v := range pr.classTodo {
		cp.classTodo[k] = v
	}
	for k, v := range pr.classDone {
		cp.classDone[k] = v
	}
	for k, v := range pr.mountUsed {
		cp.mountUsed[k] = v
	}
	return &cp
}

func newPutProgress(classes []string) putProgress {
	pr := putProgress{
		classNeeded: make(map[string]bool, len(classes)),
		classTodo:   make(map[string]bool, len(classes)),
		classDone:   map[string]int{},
		mountUsed:   map[*VolumeMount]bool{},
	}
	for _, c := range classes {
		if c != "" {
			pr.classNeeded[c] = true
			pr.classTodo[c] = true
		}
	}
	return pr
}

// PutBlock stores the given block on one or more volumes.
//
// The MD5 checksum of the block must match the given hash.
//
// The block is written to each writable volume (ordered by priority
// and then UUID, see volume.go) until at least one replica has been
// stored in each of the requested storage classes.
//
// The returned error, if any, is a KeepError with one of the
// following codes:
//
// 500 Collision
//        A different block with the same hash already exists on this
//        Keep server.
// 422 MD5Fail
//        The MD5 hash of the BLOCK does not match the argument HASH.
// 503 Full
//        There was not enough space left in any Keep volume to store
//        the object.
// 500 Fail
//        The object could not be stored for some other reason (e.g.
//        all writes failed). The text of the error message should
//        provide as much detail as possible.
func PutBlock(ctx context.Context, volmgr *RRVolumeManager, block []byte, hash string, wantStorageClasses []string) (putProgress, error) {
	log := ctxlog.FromContext(ctx)

	// Check that BLOCK's checksum matches HASH.
	blockhash := fmt.Sprintf("%x", md5.Sum(block))
	if blockhash != hash {
		log.Printf("%s: MD5 checksum %s did not match request", hash, blockhash)
		return putProgress{}, RequestHashError
	}

	result := newPutProgress(wantStorageClasses)

	// If we already have this data, it's intact on disk, and we
	// can update its timestamp, return success. If we have
	// different data with the same hash, return failure.
	if err := CompareAndTouch(ctx, volmgr, hash, block, &result); err != nil || result.Done() {
		return result, err
	}
	if ctx.Err() != nil {
		return result, ErrClientDisconnect
	}

	writables := volmgr.NextWritable()
	if len(writables) == 0 {
		log.Error("no writable volumes")
		return result, FullError
	}

	var wg sync.WaitGroup
	var mtx sync.Mutex
	cond := sync.Cond{L: &mtx}
	// pending predicts what result will be if all pending writes
	// succeed.
	pending := result.Copy()
	var allFull atomic.Value
	allFull.Store(true)

	// We hold the lock for the duration of the "each volume" loop
	// below, except when it is released during cond.Wait().
	mtx.Lock()

	for _, mnt := range writables {
		// Wait until our decision to use this mount does not
		// depend on the outcome of pending writes.
		for result.Want(mnt) && !pending.Want(mnt) {
			cond.Wait()
		}
		if !result.Want(mnt) {
			continue
		}
		mnt := mnt
		pending.Add(mnt)
		wg.Add(1)
		go func() {
			log.Debugf("PutBlock: start write to %s", mnt.UUID)
			defer wg.Done()
			err := mnt.Put(ctx, hash, block)

			mtx.Lock()
			if err != nil {
				log.Debugf("PutBlock: write to %s failed", mnt.UUID)
				pending.Sub(mnt)
			} else {
				log.Debugf("PutBlock: write to %s succeeded", mnt.UUID)
				result.Add(mnt)
			}
			cond.Broadcast()
			mtx.Unlock()

			if err != nil && err != FullError && ctx.Err() == nil {
				// The volume is not full but the
				// write did not succeed.  Report the
				// error and continue trying.
				allFull.Store(false)
				log.WithError(err).Errorf("%s: Put(%s) failed", mnt.Volume, hash)
			}
		}()
	}
	mtx.Unlock()
	wg.Wait()
	if ctx.Err() != nil {
		return result, ErrClientDisconnect
	}
	if result.Done() {
		return result, nil
	}

	if result.totalReplication > 0 {
		// Some, but not all, of the storage classes were
		// satisfied. This qualifies as success.
		return result, nil
	} else if allFull.Load().(bool) {
		log.Error("all volumes with qualifying storage classes are full")
		return putProgress{}, FullError
	} else {
		// Already logged the non-full errors.
		return putProgress{}, GenericError
	}
}

// CompareAndTouch looks for volumes where the given content already
// exists and its modification time can be updated (i.e., it is
// protected from garbage collection), and updates result accordingly.
// It returns when the result is Done() or all volumes have been
// checked.
func CompareAndTouch(ctx context.Context, volmgr *RRVolumeManager, hash string, buf []byte, result *putProgress) error {
	log := ctxlog.FromContext(ctx)
	for _, mnt := range volmgr.AllWritable() {
		if !result.Want(mnt) {
			continue
		}
		err := mnt.Compare(ctx, hash, buf)
		if ctx.Err() != nil {
			return nil
		} else if err == CollisionError {
			// Stop if we have a block with same hash but
			// different content. (It will be impossible
			// to tell which one is wanted if we have
			// both, so there's no point writing it even
			// on a different volume.)
			log.Error("collision in Compare(%s) on volume %s", hash, mnt.Volume)
			return CollisionError
		} else if os.IsNotExist(err) {
			// Block does not exist. This is the only
			// "normal" error: we don't log anything.
			continue
		} else if err != nil {
			// Couldn't open file, data is corrupt on
			// disk, etc.: log this abnormal condition,
			// and try the next volume.
			log.WithError(err).Warnf("error in Compare(%s) on volume %s", hash, mnt.Volume)
			continue
		}
		if err := mnt.Touch(hash); err != nil {
			log.WithError(err).Errorf("error in Touch(%s) on volume %s", hash, mnt.Volume)
			continue
		}
		// Compare and Touch both worked --> done.
		result.Add(mnt)
		if result.Done() {
			return nil
		}
	}
	return nil
}

var validLocatorRe = regexp.MustCompile(`^[0-9a-f]{32}$`)

// IsValidLocator returns true if the specified string is a valid Keep locator.
//   When Keep is extended to support hash types other than MD5,
//   this should be updated to cover those as well.
//
func IsValidLocator(loc string) bool {
	return validLocatorRe.MatchString(loc)
}

var authRe = regexp.MustCompile(`^(OAuth2|Bearer)\s+(.*)`)

// GetAPIToken returns the OAuth2 token from the Authorization
// header of a HTTP request, or an empty string if no matching
// token is found.
func GetAPIToken(req *http.Request) string {
	if auth, ok := req.Header["Authorization"]; ok {
		if match := authRe.FindStringSubmatch(auth[0]); match != nil {
			return match[2]
		}
	}
	return ""
}

// canDelete returns true if the user identified by apiToken is
// allowed to delete blocks.
func (rtr *router) canDelete(apiToken string) bool {
	if apiToken == "" {
		return false
	}
	// Blocks may be deleted only when Keep has been configured with a
	// data manager.
	if rtr.isSystemAuth(apiToken) {
		return true
	}
	// TODO(twp): look up apiToken with the API server
	// return true if is_admin is true and if the token
	// has unlimited scope
	return false
}

// isSystemAuth returns true if the given token is allowed to perform
// system level actions like deleting data.
func (rtr *router) isSystemAuth(token string) bool {
	return token != "" && token == rtr.cluster.SystemRootToken
}
