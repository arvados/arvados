// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// serveS3 handles r and returns true if r is a request from an S3
// client, otherwise it returns false.
func (h *handler) serveS3(w http.ResponseWriter, r *http.Request) bool {
	var token string
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "AWS ") {
		split := strings.SplitN(auth[4:], ":", 2)
		if len(split) < 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return true
		}
		token = split[0]
	} else if strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Println(w, "V4 signature is not supported")
		return true
	} else {
		return false
	}

	_, kc, client, release, err := h.getClients(r.Header.Get("X-Request-Id"), token)
	if err != nil {
		http.Error(w, "Pool failed: "+h.clientPool.Err().Error(), http.StatusInternalServerError)
		return true
	}
	defer release()

	r.URL.Path = "/by_id" + r.URL.Path

	fs := client.SiteFileSystem(kc)
	fs.ForwardSlashNameSubstitution(h.Config.cluster.Collections.ForwardSlashNameSubstitution)

	switch r.Method {
	case "GET":
		fi, err := fs.Stat(r.URL.Path)
		if os.IsNotExist(err) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return true
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		} else if fi.IsDir() {
			http.Error(w, "not found", http.StatusNotFound)
		}
		http.FileServer(fs).ServeHTTP(w, r)
		return true
	case "PUT":
		f, err := fs.OpenFile(r.URL.Path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if os.IsNotExist(err) {
			// create missing intermediate directories, then try again
			for i, c := range r.URL.Path {
				if i > 0 && c == '/' {
					dir := r.URL.Path[:i]
					err := fs.Mkdir(dir, 0755)
					if err != nil && err != os.ErrExist {
						err = fmt.Errorf("mkdir %q failed: %w", dir, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return true
					}
				}
			}
			f, err = fs.OpenFile(r.URL.Path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		}
		if err != nil {
			err = fmt.Errorf("open %q failed: %w", r.URL.Path, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return true
		}
		defer f.Close()
		_, err = io.Copy(f, r.Body)
		if err != nil {
			err = fmt.Errorf("write to %q failed: %w", r.URL.Path, err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return true
		}
		err = f.Close()
		if err != nil {
			err = fmt.Errorf("write to %q failed: %w", r.URL.Path, err)
			http.Error(w, err.Error(), http.StatusBadGateway)
			return true
		}
		err = fs.Sync()
		if err != nil {
			err = fmt.Errorf("sync failed: %w", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		w.WriteHeader(http.StatusOK)
		return true
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return true
	}
}
