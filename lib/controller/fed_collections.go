// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

type collectionFederatedRequestHandler struct {
	next    http.Handler
	handler *Handler
}

func rewriteSignatures(clusterID string, expectHash string,
	resp *http.Response, requestError error) (newResponse *http.Response, err error) {

	if requestError != nil {
		return resp, requestError
	}

	if resp.StatusCode != http.StatusOK {
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
				_, err = fmt.Fprintf(updatedManifest, "%s%s%s+R%s-%s%s", m[1], m[2], m[3], clusterID, m[5][2:], m[8])
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
	if expectHash == "" {
		expectHash = col.PortableDataHash
	} else if expectHash != col.PortableDataHash {
		return nil, fmt.Errorf("portable_data_hash %q on returned record did not match expected hash %q ", expectHash, col.PortableDataHash)
	}

	// Certify that the computed hash of the manifest_text matches our expectation
	sum := hasher.Sum(nil)
	computedHash := fmt.Sprintf("%x+%v", sum, sz)
	if computedHash != expectHash {
		return nil, fmt.Errorf("Computed manifest_text hash %q did not match expected hash %q", computedHash, expectHash)
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

	if resp.StatusCode == http.StatusNotFound {
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
			resp, err := h.handler.remoteClusterRequest(clusterId, req)
			newResponse, err := rewriteSignatures(clusterId, "", resp, err)
			h.handler.proxy.ForwardResponse(w, newResponse, err)
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
	resp, err := h.handler.localClusterRequest(req)
	newResp, err := filterLocalClusterResponse(resp, err)
	if newResp != nil || err != nil {
		h.handler.proxy.ForwardResponse(w, newResp, err)
		return
	}

	// Create a goroutine for each cluster in the
	// RemoteClusters map.  The first valid result gets
	// returned to the client.  When that happens, all
	// other outstanding requests are cancelled
	sharedContext, cancelFunc := context.WithCancel(req.Context())
	req = req.WithContext(sharedContext)
	wg := sync.WaitGroup{}
	pdh := m[1]
	success := make(chan *http.Response)
	errorChan := make(chan error, len(h.handler.Cluster.RemoteClusters))

	// use channel as a semaphore to limit the number of concurrent
	// requests at a time
	sem := make(chan bool, h.handler.Cluster.RequestLimits.GetMultiClusterRequestConcurrency())

	defer cancelFunc()

	for remoteID := range h.handler.Cluster.RemoteClusters {
		if remoteID == h.handler.Cluster.ClusterID {
			// No need to query local cluster again
			continue
		}

		wg.Add(1)
		go func(remote string) {
			defer wg.Done()
			// blocks until it can put a value into the
			// channel (which has a max queue capacity)
			sem <- true
			select {
			case <-sharedContext.Done():
				return
			default:
			}

			resp, err := h.handler.remoteClusterRequest(remote, req)
			wasSuccess := false
			defer func() {
				if resp != nil && !wasSuccess {
					resp.Body.Close()
				}
			}()
			if err != nil {
				errorChan <- err
				return
			}
			if resp.StatusCode != http.StatusOK {
				errorChan <- HTTPError{resp.Status, resp.StatusCode}
				return
			}
			select {
			case <-sharedContext.Done():
				return
			default:
			}

			newResponse, err := rewriteSignatures(remote, pdh, resp, nil)
			if err != nil {
				errorChan <- err
				return
			}
			select {
			case <-sharedContext.Done():
			case success <- newResponse:
				wasSuccess = true
			}
			<-sem
		}(remoteID)
	}
	go func() {
		wg.Wait()
		close(errorChan)
		cancelFunc()
	}()

	errorCode := http.StatusNotFound

	for {
		select {
		case newResp = <-success:
			h.handler.proxy.ForwardResponse(w, newResp, nil)
			return
		case <-sharedContext.Done():
			var errors []string
			for err := range errorChan {
				if httperr, ok := err.(HTTPError); ok {
					if httperr.Code != http.StatusNotFound {
						errorCode = http.StatusBadGateway
					}
				}
				errors = append(errors, err.Error())
			}
			httpserver.Errors(w, errors, errorCode)
			return
		}
	}
}
