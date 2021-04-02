// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

type authHandler struct {
	handler    http.Handler
	clientPool *arvadosclient.ClientPool
	cluster    *arvados.Cluster
	setupOnce  sync.Once
}

func (h *authHandler) setup() {
	client, err := arvados.NewClientFromConfig(h.cluster)
	if err != nil {
		log.Fatal(err)
	}

	ac, err := arvadosclient.New(client)
	if err != nil {
		log.Fatalf("Error setting up arvados client prototype %v", err)
	}

	h.clientPool = &arvadosclient.ClientPool{Prototype: ac}
}

func (h *authHandler) ServeHTTP(wOrig http.ResponseWriter, r *http.Request) {
	h.setupOnce.Do(h.setup)

	var statusCode int
	var statusText string
	var apiToken string
	var repoName string
	var validAPIToken bool

	w := httpserver.WrapResponseWriter(wOrig)

	if r.Method == "OPTIONS" {
		method := r.Header.Get("Access-Control-Request-Method")
		if method != "GET" && method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Header.Get("Origin") != "" {
		// Allow simple cross-origin requests without user
		// credentials ("user credentials" as defined by CORS,
		// i.e., cookies, HTTP authentication, and client-side
		// SSL certificates. See
		// http://www.w3.org/TR/cors/#user-credentials).
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	defer func() {
		if w.WroteStatus() == 0 {
			// Nobody has called WriteHeader yet: that
			// must be our job.
			w.WriteHeader(statusCode)
			if statusCode >= 400 {
				w.Write([]byte(statusText))
			}
		}

		// If the given password is a valid token, log the first 10 characters of the token.
		// Otherwise: log the string <invalid> if a password is given, else an empty string.
		passwordToLog := ""
		if !validAPIToken {
			if len(apiToken) > 0 {
				passwordToLog = "<invalid>"
			}
		} else {
			passwordToLog = apiToken[0:10]
		}

		httpserver.Log(r.RemoteAddr, passwordToLog, w.WroteStatus(), statusText, repoName, r.Method, r.URL.Path)
	}()

	creds := auth.CredentialsFromRequest(r)
	if len(creds.Tokens) == 0 {
		statusCode, statusText = http.StatusUnauthorized, "no credentials provided"
		w.Header().Add("WWW-Authenticate", "Basic realm=\"git\"")
		return
	}
	apiToken = creds.Tokens[0]

	// Access to paths "/foo/bar.git/*" and "/foo/bar/.git/*" are
	// protected by the permissions on the repository named
	// "foo/bar".
	pathParts := strings.SplitN(r.URL.Path[1:], ".git/", 2)
	if len(pathParts) != 2 {
		statusCode, statusText = http.StatusNotFound, "not found"
		return
	}
	repoName = pathParts[0]
	repoName = strings.TrimRight(repoName, "/")

	arv := h.clientPool.Get()
	if arv == nil {
		statusCode, statusText = http.StatusInternalServerError, "connection pool failed: "+h.clientPool.Err().Error()
		return
	}
	defer h.clientPool.Put(arv)

	// Ask API server whether the repository is readable using
	// this token (by trying to read it!)
	arv.ApiToken = apiToken
	repoUUID, err := h.lookupRepo(arv, repoName)
	if err != nil {
		statusCode, statusText = http.StatusInternalServerError, err.Error()
		return
	}
	validAPIToken = true
	if repoUUID == "" {
		statusCode, statusText = http.StatusNotFound, "not found"
		return
	}

	isWrite := strings.HasSuffix(r.URL.Path, "/git-receive-pack")
	if !isWrite {
		statusText = "read"
	} else {
		err := arv.Update("repositories", repoUUID, arvadosclient.Dict{
			"repository": arvadosclient.Dict{
				"modified_at": time.Now().String(),
			},
		}, &arvadosclient.Dict{})
		if err != nil {
			statusCode, statusText = http.StatusForbidden, err.Error()
			return
		}
		statusText = "write"
	}

	// Regardless of whether the client asked for "/foo.git" or
	// "/foo/.git", we choose whichever variant exists in our repo
	// root, and we try {uuid}.git and {uuid}/.git first. If none
	// of these exist, we 404 even though the API told us the repo
	// _should_ exist (presumably this means the repo was just
	// created, and gitolite sync hasn't run yet).
	rewrittenPath := ""
	tryDirs := []string{
		"/" + repoUUID + ".git",
		"/" + repoUUID + "/.git",
		"/" + repoName + ".git",
		"/" + repoName + "/.git",
	}
	for _, dir := range tryDirs {
		if fileInfo, err := os.Stat(h.cluster.Git.Repositories + dir); err != nil {
			if !os.IsNotExist(err) {
				statusCode, statusText = http.StatusInternalServerError, err.Error()
				return
			}
		} else if fileInfo.IsDir() {
			rewrittenPath = dir + "/" + pathParts[1]
			break
		}
	}
	if rewrittenPath == "" {
		log.Println("WARNING:", repoUUID,
			"git directory not found in", h.cluster.Git.Repositories, tryDirs)
		// We say "content not found" to disambiguate from the
		// earlier "API says that repo does not exist" error.
		statusCode, statusText = http.StatusNotFound, "content not found"
		return
	}
	r.URL.Path = rewrittenPath

	h.handler.ServeHTTP(w, r)
}

var uuidRegexp = regexp.MustCompile(`^[0-9a-z]{5}-s0uqq-[0-9a-z]{15}$`)

func (h *authHandler) lookupRepo(arv *arvadosclient.ArvadosClient, repoName string) (string, error) {
	reposFound := arvadosclient.Dict{}
	var column string
	if uuidRegexp.MatchString(repoName) {
		column = "uuid"
	} else {
		column = "name"
	}
	err := arv.List("repositories", arvadosclient.Dict{
		"filters": [][]string{{column, "=", repoName}},
	}, &reposFound)
	if err != nil {
		return "", err
	} else if avail, ok := reposFound["items_available"].(float64); !ok {
		return "", errors.New("bad list response from API")
	} else if avail < 1 {
		return "", nil
	} else if avail > 1 {
		return "", errors.New("name collision")
	}
	return reposFound["items"].([]interface{})[0].(map[string]interface{})["uuid"].(string), nil
}
