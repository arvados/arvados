// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

var pathPattern = `^/arvados/v1/%s(/([0-9a-z]{5})-%s-[0-9a-z]{15})?(.*)$`
var wfRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "workflows", "7fd4e"))
var containersRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "containers", "dz642"))
var containerRequestsRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "container_requests", "xvhdp"))
var collectionRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "collections", "4zz18"))
var collectionByPDHRe = regexp.MustCompile(`^/arvados/v1/collections/([0-9a-fA-F]{32}\+[0-9]+)+$`)

type genericFederatedRequestHandler struct {
	next    http.Handler
	handler *Handler
	matcher *regexp.Regexp
}

type collectionFederatedRequestHandler struct {
	next    http.Handler
	handler *Handler
}

func (h *Handler) remoteClusterRequest(remoteID string, w http.ResponseWriter, req *http.Request, filter ResponseFilter) {
	remote, ok := h.Cluster.RemoteClusters[remoteID]
	if !ok {
		err := fmt.Errorf("no proxy available for cluster %v", remoteID)
		if filter != nil {
			_, err = filter(nil, err)
		}
		if err != nil {
			httpserver.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}
	scheme := remote.Scheme
	if scheme == "" {
		scheme = "https"
	}
	req, err := h.saltAuthToken(req, remoteID)
	if err != nil {
		if filter != nil {
			_, err = filter(nil, err)
		}
		if err != nil {
			httpserver.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}
	urlOut := &url.URL{
		Scheme:   scheme,
		Host:     remote.Host,
		Path:     req.URL.Path,
		RawPath:  req.URL.RawPath,
		RawQuery: req.URL.RawQuery,
	}
	client := h.secureClient
	if remote.Insecure {
		client = h.insecureClient
	}
	h.proxy.Do(w, req, urlOut, client, filter)
}

// Buffer request body, parse form parameters in request, and then
// replace original body with the buffer so it can be re-read by
// downstream proxy steps.
func loadParamsFromForm(req *http.Request) error {
	var postBody *bytes.Buffer
	if req.Body != nil && req.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		var cl int64
		if req.ContentLength > 0 {
			cl = req.ContentLength
		}
		postBody = bytes.NewBuffer(make([]byte, 0, cl))
		originalBody := req.Body
		defer originalBody.Close()
		req.Body = ioutil.NopCloser(io.TeeReader(req.Body, postBody))
	}

	err := req.ParseForm()
	if err != nil {
		return err
	}

	if req.Body != nil && postBody != nil {
		req.Body = ioutil.NopCloser(postBody)
	}
	return nil
}

type multiClusterQueryResponseCollector struct {
	responses []map[string]interface{}
	error     error
	kind      string
	clusterID string
}

func (c *multiClusterQueryResponseCollector) collectResponse(resp *http.Response,
	requestError error) (newResponse *http.Response, err error) {
	if requestError != nil {
		c.error = requestError
		return nil, nil
	}

	defer resp.Body.Close()
	var loadInto struct {
		Kind   string                   `json:"kind"`
		Items  []map[string]interface{} `json:"items"`
		Errors []string                 `json:"errors"`
	}
	err = json.NewDecoder(resp.Body).Decode(&loadInto)

	if err != nil {
		c.error = fmt.Errorf("error fetching from %v (%v): %v", c.clusterID, resp.Status, err)
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		c.error = fmt.Errorf("error fetching from %v (%v): %v", c.clusterID, resp.Status, loadInto.Errors)
		return nil, nil
	}

	c.responses = loadInto.Items
	c.kind = loadInto.Kind

	return nil, nil
}

func (h *genericFederatedRequestHandler) remoteQueryUUIDs(w http.ResponseWriter,
	req *http.Request,
	clusterID string, uuids []string) (rp []map[string]interface{}, kind string, err error) {

	found := make(map[string]bool)
	prev_len_uuids := len(uuids) + 1
	// Loop while
	// (1) there are more uuids to query
	// (2) we're making progress - on each iteration the set of
	// uuids we are expecting for must shrink.
	for len(uuids) > 0 && len(uuids) < prev_len_uuids {
		var remoteReq http.Request
		remoteReq.Header = req.Header
		remoteReq.Method = "POST"
		remoteReq.URL = &url.URL{Path: req.URL.Path}
		remoteParams := make(url.Values)
		remoteParams.Set("_method", "GET")
		remoteParams.Set("count", "none")
		if req.Form.Get("select") != "" {
			remoteParams.Set("select", req.Form.Get("select"))
		}
		content, err := json.Marshal(uuids)
		if err != nil {
			return nil, "", err
		}
		remoteParams["filters"] = []string{fmt.Sprintf(`[["uuid", "in", %s]]`, content)}
		enc := remoteParams.Encode()
		remoteReq.Body = ioutil.NopCloser(bytes.NewBufferString(enc))

		rc := multiClusterQueryResponseCollector{clusterID: clusterID}

		if clusterID == h.handler.Cluster.ClusterID {
			h.handler.localClusterRequest(w, &remoteReq,
				rc.collectResponse)
		} else {
			h.handler.remoteClusterRequest(clusterID, w, &remoteReq,
				rc.collectResponse)
		}
		if rc.error != nil {
			return nil, "", rc.error
		}

		kind = rc.kind

		if len(rc.responses) == 0 {
			// We got zero responses, no point in doing
			// another query.
			return rp, kind, nil
		}

		rp = append(rp, rc.responses...)

		// Go through the responses and determine what was
		// returned.  If there are remaining items, loop
		// around and do another request with just the
		// stragglers.
		for _, i := range rc.responses {
			uuid, ok := i["uuid"].(string)
			if ok {
				found[uuid] = true
			}
		}

		l := []string{}
		for _, u := range uuids {
			if !found[u] {
				l = append(l, u)
			}
		}
		prev_len_uuids = len(uuids)
		uuids = l
	}

	return rp, kind, nil
}

func (h *genericFederatedRequestHandler) handleMultiClusterQuery(w http.ResponseWriter,
	req *http.Request, clusterId *string) bool {

	var filters [][]interface{}
	err := json.Unmarshal([]byte(req.Form.Get("filters")), &filters)
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusBadRequest)
		return true
	}

	// Split the list of uuids by prefix
	queryClusters := make(map[string][]string)
	expectCount := 0
	for _, filter := range filters {
		if len(filter) != 3 {
			return false
		}

		if lhs, ok := filter[0].(string); !ok || lhs != "uuid" {
			return false
		}

		op, ok := filter[1].(string)
		if !ok {
			return false
		}

		if op == "in" {
			if rhs, ok := filter[2].([]interface{}); ok {
				for _, i := range rhs {
					if u, ok := i.(string); ok {
						*clusterId = u[0:5]
						queryClusters[u[0:5]] = append(queryClusters[u[0:5]], u)
						expectCount += 1
					}
				}
			}
		} else if op == "=" {
			if u, ok := filter[2].(string); ok {
				*clusterId = u[0:5]
				queryClusters[u[0:5]] = append(queryClusters[u[0:5]], u)
				expectCount += 1
			}
		} else {
			return false
		}

	}

	if len(queryClusters) <= 1 {
		// Query does not search for uuids across multiple
		// clusters.
		return false
	}

	// Validations
	count := req.Form.Get("count")
	if count != "" && count != `none` && count != `"none"` {
		httpserver.Error(w, "Federated multi-object query must have 'count=none'", http.StatusBadRequest)
		return true
	}
	if req.Form.Get("limit") != "" || req.Form.Get("offset") != "" || req.Form.Get("order") != "" {
		httpserver.Error(w, "Federated multi-object may not provide 'limit', 'offset' or 'order'.", http.StatusBadRequest)
		return true
	}
	if expectCount > h.handler.Cluster.RequestLimits.GetMaxItemsPerResponse() {
		httpserver.Error(w, fmt.Sprintf("Federated multi-object request for %v objects which is more than max page size %v.",
			expectCount, h.handler.Cluster.RequestLimits.GetMaxItemsPerResponse()), http.StatusBadRequest)
		return true
	}
	if req.Form.Get("select") != "" {
		foundUUID := false
		var selects []string
		err := json.Unmarshal([]byte(req.Form.Get("select")), &selects)
		if err != nil {
			httpserver.Error(w, err.Error(), http.StatusBadRequest)
			return true
		}

		for _, r := range selects {
			if r == "uuid" {
				foundUUID = true
				break
			}
		}
		if !foundUUID {
			httpserver.Error(w, "Federated multi-object request must include 'uuid' in 'select'", http.StatusBadRequest)
			return true
		}
	}

	// Perform concurrent requests to each cluster

	// use channel as a semaphore to limit the number of concurrent
	// requests at a time
	sem := make(chan bool, h.handler.Cluster.RequestLimits.GetMultiClusterRequestConcurrency())
	defer close(sem)
	wg := sync.WaitGroup{}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	mtx := sync.Mutex{}
	errors := []error{}
	var completeResponses []map[string]interface{}
	var kind string

	for k, v := range queryClusters {
		if len(v) == 0 {
			// Nothing to query
			continue
		}

		// blocks until it can put a value into the
		// channel (which has a max queue capacity)
		sem <- true
		wg.Add(1)
		go func(k string, v []string) {
			rp, kn, err := h.remoteQueryUUIDs(w, req, k, v)
			mtx.Lock()
			if err == nil {
				completeResponses = append(completeResponses, rp...)
				kind = kn
			} else {
				errors = append(errors, err)
			}
			mtx.Unlock()
			wg.Done()
			<-sem
		}(k, v)
	}
	wg.Wait()

	if len(errors) > 0 {
		var strerr []string
		for _, e := range errors {
			strerr = append(strerr, e.Error())
		}
		httpserver.Errors(w, strerr, http.StatusBadGateway)
		return true
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	itemList := make(map[string]interface{})
	itemList["items"] = completeResponses
	itemList["kind"] = kind
	json.NewEncoder(w).Encode(itemList)

	return true
}

func (h *genericFederatedRequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m := h.matcher.FindStringSubmatch(req.URL.Path)
	clusterId := ""

	if len(m) > 0 && m[2] != "" {
		clusterId = m[2]
	}

	// Get form parameters from URL and form body (if POST).
	if err := loadParamsFromForm(req); err != nil {
		httpserver.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the parameters have an explicit cluster_id
	if req.Form.Get("cluster_id") != "" {
		clusterId = req.Form.Get("cluster_id")
	}

	// Handle the POST-as-GET special case (workaround for large
	// GET requests that potentially exceed maximum URL length,
	// like multi-object queries where the filter has 100s of
	// items)
	effectiveMethod := req.Method
	if req.Method == "POST" && req.Form.Get("_method") != "" {
		effectiveMethod = req.Form.Get("_method")
	}

	if effectiveMethod == "GET" &&
		clusterId == "" &&
		req.Form.Get("filters") != "" &&
		h.handleMultiClusterQuery(w, req, &clusterId) {
		return
	}

	if clusterId == "" || clusterId == h.handler.Cluster.ClusterID {
		h.next.ServeHTTP(w, req)
	} else {
		h.handler.remoteClusterRequest(clusterId, w, req, nil)
	}
}

type rewriteSignaturesClusterId struct {
	clusterID  string
	expectHash string
}

func (rw rewriteSignaturesClusterId) rewriteSignatures(resp *http.Response, requestError error) (newResponse *http.Response, err error) {
	if requestError != nil {
		return resp, requestError
	}

	if resp.StatusCode != 200 {
		return resp, nil
	}

	originalBody := resp.Body
	defer originalBody.Close()

	var col arvados.Collection
	err = json.NewDecoder(resp.Body).Decode(&col)
	if err != nil {
		return nil, err
	}

	// rewriting signatures will make manifest text 5-10% bigger so calculate
	// capacity accordingly
	updatedManifest := bytes.NewBuffer(make([]byte, 0, int(float64(len(col.ManifestText))*1.1)))

	hasher := md5.New()
	mw := io.MultiWriter(hasher, updatedManifest)
	sz := 0

	scanner := bufio.NewScanner(strings.NewReader(col.ManifestText))
	scanner.Buffer(make([]byte, 1048576), len(col.ManifestText))
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, " ")
		if len(tokens) < 3 {
			return nil, fmt.Errorf("Invalid stream (<3 tokens): %q", line)
		}

		n, err := mw.Write([]byte(tokens[0]))
		if err != nil {
			return nil, fmt.Errorf("Error updating manifest: %v", err)
		}
		sz += n
		for _, token := range tokens[1:] {
			n, err = mw.Write([]byte(" "))
			if err != nil {
				return nil, fmt.Errorf("Error updating manifest: %v", err)
			}
			sz += n

			m := keepclient.SignedLocatorRe.FindStringSubmatch(token)
			if m != nil {
				// Rewrite the block signature to be a remote signature
				_, err = fmt.Fprintf(updatedManifest, "%s%s%s+R%s-%s%s", m[1], m[2], m[3], rw.clusterID, m[5][2:], m[8])
				if err != nil {
					return nil, fmt.Errorf("Error updating manifest: %v", err)
				}

				// for hash checking, ignore signatures
				n, err = fmt.Fprintf(hasher, "%s%s", m[1], m[2])
				if err != nil {
					return nil, fmt.Errorf("Error updating manifest: %v", err)
				}
				sz += n
			} else {
				n, err = mw.Write([]byte(token))
				if err != nil {
					return nil, fmt.Errorf("Error updating manifest: %v", err)
				}
				sz += n
			}
		}
		n, err = mw.Write([]byte("\n"))
		if err != nil {
			return nil, fmt.Errorf("Error updating manifest: %v", err)
		}
		sz += n
	}

	// Check that expected hash is consistent with
	// portable_data_hash field of the returned record
	if rw.expectHash == "" {
		rw.expectHash = col.PortableDataHash
	} else if rw.expectHash != col.PortableDataHash {
		return nil, fmt.Errorf("portable_data_hash %q on returned record did not match expected hash %q ", rw.expectHash, col.PortableDataHash)
	}

	// Certify that the computed hash of the manifest_text matches our expectation
	sum := hasher.Sum(nil)
	computedHash := fmt.Sprintf("%x+%v", sum, sz)
	if computedHash != rw.expectHash {
		return nil, fmt.Errorf("Computed manifest_text hash %q did not match expected hash %q", computedHash, rw.expectHash)
	}

	col.ManifestText = updatedManifest.String()

	newbody, err := json.Marshal(col)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(newbody)
	resp.Body = ioutil.NopCloser(buf)
	resp.ContentLength = int64(buf.Len())
	resp.Header.Set("Content-Length", fmt.Sprintf("%v", buf.Len()))

	return resp, nil
}

func filterLocalClusterResponse(resp *http.Response, requestError error) (newResponse *http.Response, err error) {
	if requestError != nil {
		return resp, requestError
	}

	if resp.StatusCode == 404 {
		// Suppress returning this result, because we want to
		// search the federation.
		return nil, nil
	}
	return resp, nil
}

type searchRemoteClusterForPDH struct {
	pdh           string
	remoteID      string
	mtx           *sync.Mutex
	sentResponse  *bool
	sharedContext *context.Context
	cancelFunc    func()
	errors        *[]string
	statusCode    *int
}

func (s *searchRemoteClusterForPDH) filterRemoteClusterResponse(resp *http.Response, requestError error) (newResponse *http.Response, err error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if *s.sentResponse {
		// Another request already returned a response
		return nil, nil
	}

	if requestError != nil {
		*s.errors = append(*s.errors, fmt.Sprintf("Request error contacting %q: %v", s.remoteID, requestError))
		// Record the error and suppress response
		return nil, nil
	}

	if resp.StatusCode != 200 {
		// Suppress returning unsuccessful result.  Maybe
		// another request will find it.
		// TODO collect and return error responses.
		*s.errors = append(*s.errors, fmt.Sprintf("Response from %q: %v", s.remoteID, resp.Status))
		if resp.StatusCode != 404 {
			// Got a non-404 error response, convert into BadGateway
			*s.statusCode = http.StatusBadGateway
		}
		return nil, nil
	}

	s.mtx.Unlock()

	// This reads the response body.  We don't want to hold the
	// lock while doing this because other remote requests could
	// also have made it to this point, and we don't want a
	// slow response holding the lock to block a faster response
	// that is waiting on the lock.
	newResponse, err = rewriteSignaturesClusterId{s.remoteID, s.pdh}.rewriteSignatures(resp, nil)

	s.mtx.Lock()

	if *s.sentResponse {
		// Another request already returned a response
		return nil, nil
	}

	if err != nil {
		// Suppress returning unsuccessful result.  Maybe
		// another request will be successful.
		*s.errors = append(*s.errors, fmt.Sprintf("Error parsing response from %q: %v", s.remoteID, err))
		return nil, nil
	}

	// We have a successful response.  Suppress/cancel all the
	// other requests/responses.
	*s.sentResponse = true
	s.cancelFunc()

	return newResponse, nil
}

func (h *collectionFederatedRequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		// Only handle GET requests right now
		h.next.ServeHTTP(w, req)
		return
	}

	m := collectionByPDHRe.FindStringSubmatch(req.URL.Path)
	if len(m) != 2 {
		// Not a collection PDH GET request
		m = collectionRe.FindStringSubmatch(req.URL.Path)
		clusterId := ""

		if len(m) > 0 {
			clusterId = m[2]
		}

		if clusterId != "" && clusterId != h.handler.Cluster.ClusterID {
			// request for remote collection by uuid
			h.handler.remoteClusterRequest(clusterId, w, req,
				rewriteSignaturesClusterId{clusterId, ""}.rewriteSignatures)
			return
		}
		// not a collection UUID request, or it is a request
		// for a local UUID, either way, continue down the
		// handler stack.
		h.next.ServeHTTP(w, req)
		return
	}

	// Request for collection by PDH.  Search the federation.

	// First, query the local cluster.
	if h.handler.localClusterRequest(w, req, filterLocalClusterResponse) {
		return
	}

	sharedContext, cancelFunc := context.WithCancel(req.Context())
	defer cancelFunc()
	req = req.WithContext(sharedContext)

	// Create a goroutine for each cluster in the
	// RemoteClusters map.  The first valid result gets
	// returned to the client.  When that happens, all
	// other outstanding requests are cancelled or
	// suppressed.
	sentResponse := false
	mtx := sync.Mutex{}
	wg := sync.WaitGroup{}
	var errors []string
	var errorCode int = 404

	// use channel as a semaphore to limit the number of concurrent
	// requests at a time
	sem := make(chan bool, h.handler.Cluster.RequestLimits.GetMultiClusterRequestConcurrency())
	defer close(sem)
	for remoteID := range h.handler.Cluster.RemoteClusters {
		if remoteID == h.handler.Cluster.ClusterID {
			// No need to query local cluster again
			continue
		}
		// blocks until it can put a value into the
		// channel (which has a max queue capacity)
		sem <- true
		if sentResponse {
			break
		}
		search := &searchRemoteClusterForPDH{m[1], remoteID, &mtx, &sentResponse,
			&sharedContext, cancelFunc, &errors, &errorCode}
		wg.Add(1)
		go func() {
			h.handler.remoteClusterRequest(search.remoteID, w, req, search.filterRemoteClusterResponse)
			wg.Done()
			<-sem
		}()
	}
	wg.Wait()

	if sentResponse {
		return
	}

	// No successful responses, so return the error
	httpserver.Errors(w, errors, errorCode)
}

func (h *Handler) setupProxyRemoteCluster(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/arvados/v1/workflows", &genericFederatedRequestHandler{next, h, wfRe})
	mux.Handle("/arvados/v1/workflows/", &genericFederatedRequestHandler{next, h, wfRe})
	mux.Handle("/arvados/v1/containers", &genericFederatedRequestHandler{next, h, containersRe})
	mux.Handle("/arvados/v1/containers/", &genericFederatedRequestHandler{next, h, containersRe})
	mux.Handle("/arvados/v1/container_requests", &genericFederatedRequestHandler{next, h, containerRequestsRe})
	mux.Handle("/arvados/v1/container_requests/", &genericFederatedRequestHandler{next, h, containerRequestsRe})
	mux.Handle("/arvados/v1/collections", next)
	mux.Handle("/arvados/v1/collections/", &collectionFederatedRequestHandler{next, h})
	mux.Handle("/", next)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		parts := strings.Split(req.Header.Get("Authorization"), "/")
		alreadySalted := (len(parts) == 3 && parts[0] == "Bearer v2" && len(parts[2]) == 40)

		if alreadySalted ||
			strings.Index(req.Header.Get("Via"), "arvados-controller") != -1 {
			// The token is already salted, or this is a
			// request from another instance of
			// arvados-controller.  In either case, we
			// don't want to proxy this query, so just
			// continue down the instance handler stack.
			next.ServeHTTP(w, req)
			return
		}

		mux.ServeHTTP(w, req)
	})

	return mux
}

type CurrentUser struct {
	Authorization arvados.APIClientAuthorization
	UUID          string
}

func (h *Handler) validateAPItoken(req *http.Request, user *CurrentUser) error {
	db, err := h.db(req)
	if err != nil {
		return err
	}
	return db.QueryRowContext(req.Context(), `SELECT api_client_authorizations.uuid, users.uuid FROM api_client_authorizations JOIN users on api_client_authorizations.user_id=users.id WHERE api_token=$1 AND (expires_at IS NULL OR expires_at > current_timestamp) LIMIT 1`, user.Authorization.APIToken).Scan(&user.Authorization.UUID, &user.UUID)
}

// Extract the auth token supplied in req, and replace it with a
// salted token for the remote cluster.
func (h *Handler) saltAuthToken(req *http.Request, remote string) (updatedReq *http.Request, err error) {
	updatedReq = (&http.Request{
		Method:        req.Method,
		URL:           req.URL,
		Header:        req.Header,
		Body:          req.Body,
		ContentLength: req.ContentLength,
		Host:          req.Host,
	}).WithContext(req.Context())

	creds := auth.NewCredentials()
	creds.LoadTokensFromHTTPRequest(updatedReq)
	if len(creds.Tokens) == 0 && updatedReq.Header.Get("Content-Type") == "application/x-www-form-encoded" {
		// Override ParseForm's 10MiB limit by ensuring
		// req.Body is a *http.maxBytesReader.
		updatedReq.Body = http.MaxBytesReader(nil, updatedReq.Body, 1<<28) // 256MiB. TODO: use MaxRequestSize from discovery doc or config.
		if err := creds.LoadTokensFromHTTPRequestBody(updatedReq); err != nil {
			return nil, err
		}
		// Replace req.Body with a buffer that re-encodes the
		// form without api_token, in case we end up
		// forwarding the request.
		if updatedReq.PostForm != nil {
			updatedReq.PostForm.Del("api_token")
		}
		updatedReq.Body = ioutil.NopCloser(bytes.NewBufferString(updatedReq.PostForm.Encode()))
	}
	if len(creds.Tokens) == 0 {
		return updatedReq, nil
	}

	token, err := auth.SaltToken(creds.Tokens[0], remote)

	log.Printf("Salting %q %q to get %q %q", creds.Tokens[0], remote, token, err)
	if err == auth.ErrObsoleteToken {
		// If the token exists in our own database, salt it
		// for the remote. Otherwise, assume it was issued by
		// the remote, and pass it through unmodified.
		currentUser := CurrentUser{Authorization: arvados.APIClientAuthorization{APIToken: creds.Tokens[0]}}
		err = h.validateAPItoken(req, &currentUser)
		if err == sql.ErrNoRows {
			// Not ours; pass through unmodified.
			token = currentUser.Authorization.APIToken
		} else if err != nil {
			return nil, err
		} else {
			// Found; make V2 version and salt it.
			token, err = auth.SaltToken(currentUser.Authorization.TokenV2(), remote)
			if err != nil {
				return nil, err
			}
		}
	} else if err != nil {
		return nil, err
	}
	updatedReq.Header = http.Header{}
	for k, v := range req.Header {
		if k == "Authorization" {
			updatedReq.Header[k] = []string{"Bearer " + token}
		} else {
			updatedReq.Header[k] = v
		}
	}

	log.Printf("Salted %q %q to get %q", creds.Tokens[0], remote, token)

	// Remove api_token=... from the the query string, in case we
	// end up forwarding the request.
	if values, err := url.ParseQuery(updatedReq.URL.RawQuery); err != nil {
		return nil, err
	} else if _, ok := values["api_token"]; ok {
		delete(values, "api_token")
		updatedReq.URL = &url.URL{
			Scheme:   req.URL.Scheme,
			Host:     req.URL.Host,
			Path:     req.URL.Path,
			RawPath:  req.URL.RawPath,
			RawQuery: values.Encode(),
		}
	}
	return updatedReq, nil
}
