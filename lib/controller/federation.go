// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bufio"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

var wfRe = regexp.MustCompile(`^/arvados/v1/workflows/([0-9a-z]{5})-[^/]+$`)
var collectionRe = regexp.MustCompile(`^/arvados/v1/collections/([0-9a-z]{5})-[^/]+$`)
var collectionByPDHRe = regexp.MustCompile(`^/arvados/v1/collections/([0-9a-fA-F]{32}\+[0-9]+)+$`)

type genericFederatedRequestHandler struct {
	next    http.Handler
	handler *Handler
}

type collectionFederatedRequestHandler struct {
	next    http.Handler
	handler *Handler
}

func (h *Handler) remoteClusterRequest(remoteID string, w http.ResponseWriter, req *http.Request, filter ResponseFilter) {
	remote, ok := h.Cluster.RemoteClusters[remoteID]
	if !ok {
		httpserver.Error(w, "no proxy available for cluster "+remoteID, http.StatusNotFound)
		return
	}
	scheme := remote.Scheme
	if scheme == "" {
		scheme = "https"
	}
	err := h.saltAuthToken(req, remoteID)
	if err != nil {
		httpserver.Error(w, err.Error(), http.StatusBadRequest)
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

func (h *genericFederatedRequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m := wfRe.FindStringSubmatch(req.URL.Path)
	if len(m) < 2 || m[1] == h.handler.Cluster.ClusterID {
		h.next.ServeHTTP(w, req)
		return
	}
	h.handler.remoteClusterRequest(m[1], w, req, nil)
}

type rewriteSignaturesClusterId string

func (clusterId rewriteSignaturesClusterId) rewriteSignatures(resp *http.Response) (newResponse *http.Response, err error) {
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

	scanner := bufio.NewScanner(strings.NewReader(col.ManifestText))
	scanner.Buffer(make([]byte, 1048576), len(col.ManifestText))
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, " ")
		if len(tokens) < 3 {
			return nil, fmt.Errorf("Invalid stream (<3 tokens): %q", line)
		}

		updatedManifest.WriteString(tokens[0])
		for _, token := range tokens[1:] {
			updatedManifest.WriteString(" ")
			m := keepclient.SignedLocatorRe.FindStringSubmatch(token)
			if m != nil {
				// Rewrite the block signature to be a remote signature
				fmt.Fprintf(updatedManifest, "%s%s%s+R%s-%s%s", m[1], m[2], m[3], clusterId, m[5][2:], m[8])
			} else {
				updatedManifest.WriteString(token)
			}

		}
		updatedManifest.WriteString("\n")
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

type searchLocalClusterForPDH struct {
	needSearchFederation bool
}

func (s *searchLocalClusterForPDH) filterLocalClusterResponse(resp *http.Response) (newResponse *http.Response, err error) {
	if resp.StatusCode == 404 {
		// Suppress returning this result, because we want to
		// search the federation.
		s.needSearchFederation = true
		return nil, nil
	}
	return resp, nil
}

func (h *collectionFederatedRequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	m := collectionByPDHRe.FindStringSubmatch(req.URL.Path)
	if len(m) == 2 {
		urlOut, insecure, err := findRailsAPI(h.handler.Cluster, h.handler.NodeProfile)
		if err != nil {
			httpserver.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		urlOut = &url.URL{
			Scheme:   urlOut.Scheme,
			Host:     urlOut.Host,
			Path:     req.URL.Path,
			RawPath:  req.URL.RawPath,
			RawQuery: req.URL.RawQuery,
		}
		client := h.handler.secureClient
		if insecure {
			client = h.handler.insecureClient
		}
		sf := &searchLocalClusterForPDH{false}
		h.handler.proxy.Do(w, req, urlOut, client, sf.filterLocalClusterResponse)
		if !sf.needSearchFederation {
			// a response was sent
			return
		}
	}

	m = collectionRe.FindStringSubmatch(req.URL.Path)
	if len(m) < 2 || m[1] == h.handler.Cluster.ClusterID {
		h.next.ServeHTTP(w, req)
		return
	}
	h.handler.remoteClusterRequest(m[1], w, req,
		rewriteSignaturesClusterId(m[1]).rewriteSignatures)
}

func (h *Handler) setupProxyRemoteCluster(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/arvados/v1/workflows", next)
	mux.Handle("/arvados/v1/workflows/", &genericFederatedRequestHandler{next, h})
	mux.Handle("/arvados/v1/collections", next)
	mux.Handle("/arvados/v1/collections/", &collectionFederatedRequestHandler{next, h})
	mux.Handle("/", next)

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
func (h *Handler) saltAuthToken(req *http.Request, remote string) error {
	creds := auth.NewCredentials()
	creds.LoadTokensFromHTTPRequest(req)
	if len(creds.Tokens) == 0 && req.Header.Get("Content-Type") == "application/x-www-form-encoded" {
		// Override ParseForm's 10MiB limit by ensuring
		// req.Body is a *http.maxBytesReader.
		req.Body = http.MaxBytesReader(nil, req.Body, 1<<28) // 256MiB. TODO: use MaxRequestSize from discovery doc or config.
		if err := creds.LoadTokensFromHTTPRequestBody(req); err != nil {
			return err
		}
		// Replace req.Body with a buffer that re-encodes the
		// form without api_token, in case we end up
		// forwarding the request.
		if req.PostForm != nil {
			req.PostForm.Del("api_token")
		}
		req.Body = ioutil.NopCloser(bytes.NewBufferString(req.PostForm.Encode()))
	}
	if len(creds.Tokens) == 0 {
		return nil
	}
	token, err := auth.SaltToken(creds.Tokens[0], remote)
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
			return err
		} else {
			// Found; make V2 version and salt it.
			token, err = auth.SaltToken(currentUser.Authorization.TokenV2(), remote)
			if err != nil {
				return err
			}
		}
	} else if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Remove api_token=... from the the query string, in case we
	// end up forwarding the request.
	if values, err := url.ParseQuery(req.URL.RawQuery); err != nil {
		return err
	} else if _, ok := values["api_token"]; ok {
		delete(values, "api_token")
		req.URL.RawQuery = values.Encode()
	}
	return nil
}
