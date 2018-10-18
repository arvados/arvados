// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/httpserver"
)

type genericFederatedRequestHandler struct {
	next    http.Handler
	handler *Handler
	matcher *regexp.Regexp
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

		var resp *http.Response
		if clusterID == h.handler.Cluster.ClusterID {
			resp, err = h.handler.localClusterRequest(&remoteReq)
		} else {
			resp, err = h.handler.remoteClusterRequest(clusterID, &remoteReq)
		}
		rc.collectResponse(resp, err)

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
		resp, err := h.handler.remoteClusterRequest(clusterId, req)
		h.handler.proxy.ForwardResponse(w, resp, err)
	}
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
