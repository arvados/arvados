// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package controller

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/jmcvetta/randutil"
)

var pathPattern = `^/arvados/v1/%s(/([0-9a-z]{5})-%s-[0-9a-z]{15})?(.*)$`
var wfRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "workflows", "7fd4e"))
var containersRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "containers", "dz642"))
var containerRequestsRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "container_requests", "xvhdp"))
var collectionsRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "collections", "4zz18"))
var collectionsByPDHRe = regexp.MustCompile(`^/arvados/v1/collections/([0-9a-fA-F]{32}\+[0-9]+)+$`)
var linksRe = regexp.MustCompile(fmt.Sprintf(pathPattern, "links", "o0j2j"))

func (h *Handler) remoteClusterRequest(remoteID string, req *http.Request) (*http.Response, error) {
	remote, ok := h.Cluster.RemoteClusters[remoteID]
	if !ok {
		return nil, HTTPError{fmt.Sprintf("no proxy available for cluster %v", remoteID), http.StatusNotFound}
	}
	scheme := remote.Scheme
	if scheme == "" {
		scheme = "https"
	}
	saltedReq, err := h.saltAuthToken(req, remoteID)
	if err != nil {
		return nil, err
	}
	urlOut := &url.URL{
		Scheme:   scheme,
		Host:     remote.Host,
		Path:     saltedReq.URL.Path,
		RawPath:  saltedReq.URL.RawPath,
		RawQuery: saltedReq.URL.RawQuery,
	}
	client := h.secureClient
	if remote.Insecure {
		client = h.insecureClient
	}
	return h.proxy.Do(saltedReq, urlOut, client)
}

// Buffer request body, parse form parameters in request, and then
// replace original body with the buffer so it can be re-read by
// downstream proxy steps.
func loadParamsFromForm(req *http.Request) error {
	var postBody *bytes.Buffer
	if ct := req.Header.Get("Content-Type"); ct == "" {
		// Assume application/octet-stream, i.e., no form to parse.
	} else if ct, _, err := mime.ParseMediaType(ct); err != nil {
		return err
	} else if ct == "application/x-www-form-urlencoded" && req.Body != nil {
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

func (h *Handler) setupProxyRemoteCluster(next http.Handler) http.Handler {
	mux := http.NewServeMux()

	wfHandler := &genericFederatedRequestHandler{next, h, wfRe, nil}
	containersHandler := &genericFederatedRequestHandler{next, h, containersRe, nil}
	linksRequestsHandler := &genericFederatedRequestHandler{next, h, linksRe, nil}

	mux.Handle("/arvados/v1/workflows", wfHandler)
	mux.Handle("/arvados/v1/workflows/", wfHandler)
	mux.Handle("/arvados/v1/containers", containersHandler)
	mux.Handle("/arvados/v1/containers/", containersHandler)
	mux.Handle("/arvados/v1/links", linksRequestsHandler)
	mux.Handle("/arvados/v1/links/", linksRequestsHandler)
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

// validateAPItoken extracts the token from the provided http request,
// checks it again api_client_authorizations table in the database,
// and fills in the token scope and user UUID.  Does not handle remote
// tokens unless they are already in the database and not expired.
//
// Return values are:
//
// nil, false, non-nil -- if there was an internal error
//
// nil, false, nil -- if the token is invalid
//
// non-nil, true, nil -- if the token is valid
func (h *Handler) validateAPItoken(req *http.Request, token string) (*CurrentUser, bool, error) {
	user := CurrentUser{Authorization: arvados.APIClientAuthorization{APIToken: token}}
	db, err := h.db(req.Context())
	if err != nil {
		ctxlog.FromContext(req.Context()).WithError(err).Debugf("validateAPItoken(%s): database error", token)
		return nil, false, err
	}

	var uuid string
	if strings.HasPrefix(token, "v2/") {
		sp := strings.Split(token, "/")
		uuid = sp[1]
		token = sp[2]
	}
	user.Authorization.APIToken = token
	var scopes string
	err = db.QueryRowContext(req.Context(), `SELECT api_client_authorizations.uuid, api_client_authorizations.scopes, users.uuid FROM api_client_authorizations JOIN users on api_client_authorizations.user_id=users.id WHERE api_token=$1 AND (expires_at IS NULL OR expires_at > current_timestamp AT TIME ZONE 'UTC') LIMIT 1`, token).Scan(&user.Authorization.UUID, &scopes, &user.UUID)
	if err == sql.ErrNoRows {
		ctxlog.FromContext(req.Context()).Debugf("validateAPItoken(%s): not found in database", token)
		return nil, false, nil
	} else if err != nil {
		ctxlog.FromContext(req.Context()).WithError(err).Debugf("validateAPItoken(%s): database error", token)
		return nil, false, err
	}
	if uuid != "" && user.Authorization.UUID != uuid {
		// secret part matches, but UUID doesn't -- somewhat surprising
		ctxlog.FromContext(req.Context()).Debugf("validateAPItoken(%s): secret part found, but with different UUID: %s", token, user.Authorization.UUID)
		return nil, false, nil
	}
	err = json.Unmarshal([]byte(scopes), &user.Authorization.Scopes)
	if err != nil {
		ctxlog.FromContext(req.Context()).WithError(err).Debugf("validateAPItoken(%s): error parsing scopes from db", token)
		return nil, false, err
	}
	ctxlog.FromContext(req.Context()).Debugf("validateAPItoken(%s): ok", token)
	return &user, true, nil
}

func (h *Handler) createAPItoken(req *http.Request, userUUID string, scopes []string) (*arvados.APIClientAuthorization, error) {
	db, err := h.db(req.Context())
	if err != nil {
		return nil, err
	}
	rd, err := randutil.String(15, "abcdefghijklmnopqrstuvwxyz0123456789")
	if err != nil {
		return nil, err
	}
	uuid := fmt.Sprintf("%v-gj3su-%v", h.Cluster.ClusterID, rd)
	token, err := randutil.String(50, "abcdefghijklmnopqrstuvwxyz0123456789")
	if err != nil {
		return nil, err
	}
	if len(scopes) == 0 {
		scopes = append(scopes, "all")
	}
	scopesjson, err := json.Marshal(scopes)
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(req.Context(),
		`INSERT INTO api_client_authorizations
(uuid, api_token, expires_at, scopes,
user_id,
api_client_id, created_at, updated_at)
VALUES ($1, $2, CURRENT_TIMESTAMP AT TIME ZONE 'UTC' + INTERVAL '2 weeks', $3,
(SELECT id FROM users WHERE users.uuid=$4 LIMIT 1),
0, CURRENT_TIMESTAMP AT TIME ZONE 'UTC', CURRENT_TIMESTAMP AT TIME ZONE 'UTC')`,
		uuid, token, string(scopesjson), userUUID)

	if err != nil {
		return nil, err
	}

	return &arvados.APIClientAuthorization{
		UUID:      uuid,
		APIToken:  token,
		ExpiresAt: "",
		Scopes:    scopes}, nil
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

	ctxlog.FromContext(req.Context()).Debugf("saltAuthToken: cluster %s token %s remote %s", h.Cluster.ClusterID, creds.Tokens[0], remote)
	token, err := auth.SaltToken(creds.Tokens[0], remote)

	if err == auth.ErrObsoleteToken || err == auth.ErrTokenFormat {
		// If the token exists in our own database for our own
		// user, salt it for the remote. Otherwise, assume it
		// was issued by the remote, and pass it through
		// unmodified.
		currentUser, ok, err := h.validateAPItoken(req, creds.Tokens[0])
		if err != nil {
			return nil, err
		} else if !ok || strings.HasPrefix(currentUser.UUID, remote) {
			// Unknown, or cached + belongs to remote;
			// pass through unmodified.
			token = creds.Tokens[0]
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
		if k != "Authorization" {
			updatedReq.Header[k] = v
		}
	}
	updatedReq.Header.Set("Authorization", "Bearer "+token)

	// Remove api_token=... from the query string, in case we
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
