// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

// serveZip handles a request for a zip archive.
func (h *handler) serveZip(w http.ResponseWriter, r *http.Request, client *arvados.Client, sitefs arvados.CustomFileSystem, path string) {
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "POST" {
		// This is a generic 400, not 405 (method not allowed)
		// because this method/URL combination is allowed,
		// just not with the Accept: application/zip header.
		http.Error(w, "zip archive can only be served via GET, HEAD, or POST", http.StatusBadRequest)
		return
	}
	coll, subdir := h.determineCollection(sitefs, path)
	if coll == nil || subdir != "" {
		http.Error(w, "zip archive can only be served from the root directory of a collection", http.StatusBadRequest)
		return
	}
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reqpaths := r.Form["files"]
	if reqpaths == nil && r.Header.Get("Content-Type") == "application/json" {
		// r.Body is always non-nil, but will return EOF
		// immediately if no body is present.
		err := json.NewDecoder(r.Body).Decode(&reqpaths)
		if err != nil && err != io.EOF {
			http.Error(w, "error reading request body: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	collfs, err := fs.Sub(arvados.FS(sitefs), strings.TrimSuffix(path, "/"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	wanted := make(map[string]bool)
	for _, path := range reqpaths {
		wanted[path] = true
		if path == "/" {
			continue
		} else if f, err := collfs.Open(strings.TrimSuffix(path, "/")); err != nil {
			http.Error(w, fmt.Sprintf("%q: file does not exist", path), http.StatusNotFound)
			return
		} else {
			f.Close()
		}
	}
	iswanted := func(path string) bool {
		if len(wanted) == 0 {
			// No reqpaths provided ==> include all files
			return true
		}
		if wanted[path] {
			// Exact filename match
			return true
		}
		if wanted["/"] {
			// Entire collection selected (special case
			// not covered by the generic "parent
			// selected" loop below)
			return true
		}
		for i := len(path) - 1; i >= 0; i-- {
			if path[i] == '/' && (wanted[path[:i]] || wanted[path[:i+1]]) {
				// Parent directory match
				return true
			}
		}
		return false
	}
	var filepaths []string
	err = fs.WalkDir(collfs, ".", func(path string, dirent fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirent.IsDir() {
			return nil
		}
		if !iswanted(path) {
			return nil
		}
		filepaths = append(filepaths, path)
		return nil
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var zipfilename string
	// Retrieve collection name if possible
	if coll.Name == "" && coll.UUID != "" {
		err = client.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+coll.UUID, nil, map[string]interface{}{
			"select": []string{"uuid", "name", "portable_data_hash", "properties"},
		})
		if err != nil {
			if he := errorWithHTTPStatus(nil); errors.As(err, &he) {
				http.Error(w, err.Error(), he.HTTPStatus())
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
		zipfilename = coll.Name
	} else if coll.Name == "" {
		zipfilename = coll.PortableDataHash
	}
	if len(filepaths) == 1 && len(reqpaths) == 1 && filepaths[0] == reqpaths[0] {
		// If the client specified a single (non-directory)
		// file, include the name of the file in the zip
		// archive name.
		_, basename := filepath.Split(filepaths[0])
		zipfilename += " - " + basename
	} else if len(wanted) > 0 && !wanted["/"] {
		// If the client specified any other subset of the
		// collection, mention the number of files that will
		// be in the archive, to make it more obvious that
		// it's not an archive of the entire collection.
		zipfilename += fmt.Sprintf(" - %d files", len(filepaths))
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": zipfilename}))
	w.Header().Set("Content-Type", "application/zip")
	zipw := zip.NewWriter(w)
	wrote := false
	err = func() error {
		if r.Form.Get("include_collection_metadata") != "" {
			m := map[string]interface{}{
				"portable_data_hash": coll.PortableDataHash,
				"properties":         coll.Properties,
			}
			if coll.UUID != "" {
				m["uuid"] = coll.UUID
				m["name"] = coll.Name
			}
			wrote = true
			zipf, err := zipw.CreateHeader(&zip.FileHeader{
				Name:   "collection.json",
				Method: zip.Store,
			})
			if err != nil {
				return err
			}
			err = json.NewEncoder(zipf).Encode(m)
		}
		for _, path := range filepaths {
			f, err := collfs.Open(path)
			if err != nil {
				f.Close()
				break
			}
			wrote = true
			w, err := zipw.CreateHeader(&zip.FileHeader{
				Name:   path,
				Method: zip.Store,
			})
			if err != nil {
				f.Close()
				break
			}
			_, err = io.Copy(w, f)
			f.Close()
			if err != nil {
				break
			}
		}
		wrote = true
		return zipw.Close()
	}()
	if err != nil {
		if wrote {
			ctxlog.FromContext(r.Context()).Errorf("error writing zip archive after sending response header: %s", err)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}
