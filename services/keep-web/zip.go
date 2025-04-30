// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"archive/zip"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

// serveZip handles a request for a zip archive.
func (h *handler) serveZip(w http.ResponseWriter, r *http.Request, session *cachedSession, sitefs arvados.CustomFileSystem, ziproot string, tokenUser *arvados.User) {
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "POST" {
		// This is a generic 400, not 405 (method not allowed)
		// because this method/URL combination is allowed,
		// just not with the Accept: application/zip header.
		http.Error(w, "zip archive can only be served via GET, HEAD, or POST", http.StatusBadRequest)
		return
	}
	// Check "GET" permission regardless of r.Method, because all
	// methods result in downloads.
	if !h.userPermittedToUploadOrDownload("GET", tokenUser) {
		http.Error(w, "Not permitted", http.StatusForbidden)
		return
	}
	coll, subdir := h.determineCollection(sitefs, ziproot)
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
	collfs, err := fs.Sub(arvados.FS(sitefs), strings.TrimSuffix(ziproot, "/"))
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
		err = session.client.RequestAndDecode(&coll, "GET", "arvados/v1/collections/"+coll.UUID, nil, map[string]interface{}{
			"select": []string{
				"created_at",
				"description",
				"modified_at",
				"modified_by_user_uuid",
				"name",
				"portable_data_hash",
				"properties",
				"uuid",
			},
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

	var user arvados.User
	if coll.ModifiedByUserUUID != "" {
		err = session.client.RequestAndDecode(&user, "GET", "arvados/v1/users/"+coll.ModifiedByUserUUID, nil, map[string]interface{}{
			"select": []string{
				"email",
				"full_name",
				"username",
				"uuid",
				// RailsAPI <= 3.1 fails if we select
				// full_name without also selecting
				// first_name and last_name.
				"first_name",
				"last_name",
			},
		})
		if he := errorWithHTTPStatus(nil); errors.As(err, &he) && he.HTTPStatus() < 500 {
			// Cannot retrieve the user record, but this
			// shouldn't prevent the download from
			// working.
			http.Error(w, err.Error(), he.HTTPStatus())
		} else if errors.As(err, &he) {
			http.Error(w, err.Error(), he.HTTPStatus())
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
		//
		// Also include a partial hash of {PDH, list of
		// filenames} so downloading different subsets of a
		// collection results in different names, even if the
		// number of files happens to be the same.  (The pdh
		// is incorporated here because otherwise the
		// existence of a hash in the filename would be a
		// strong misleading hint that identical filenames
		// signify identical content.)
		h := md5.New()
		fmt.Fprintln(h, coll.PortableDataHash)
		for _, path := range filepaths {
			fmt.Fprintln(h, path)
		}
		zipfilename += fmt.Sprintf(" - %d files (%-4.4x)", len(filepaths), h.Sum(nil))
	}
	zipfilename += ".zip"

	logpath := ""
	if len(filepaths) == 1 {
		// If downloading a zip file with exactly one file,
		// log that file as collection_file_path in the audit
		// logs.  (Otherwise, leave collection_file_path
		// empty.)
		logpath = filepaths[0]
	}
	rGET := r.Clone(r.Context())
	rGET.Method = "GET"
	h.logUploadOrDownload(rGET, session.arvadosclient, session.fs, logpath, len(filepaths), coll, tokenUser)

	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": zipfilename}))
	w.Header().Set("Content-Type", "application/zip")
	zipw := zip.NewWriter(w)
	wrote := false
	err = func() error {
		u := url.URL(h.Cluster.Services.WebDAVDownload.ExternalURL)
		if coll.UUID != "" {
			u.Path = "/by_id/" + coll.UUID + "/"
		} else {
			u.Path = "/by_id/" + coll.PortableDataHash + "/"
		}
		err := zipw.SetComment(fmt.Sprintf("Downloaded from %s", u.String()))
		if err != nil {
			return err
		}
		if r.Form.Get("include_collection_metadata") != "" {
			m := map[string]interface{}{
				"portable_data_hash": coll.PortableDataHash,
			}
			if coll.UUID != "" {
				m["uuid"] = coll.UUID
				m["name"] = coll.Name
				m["properties"] = coll.Properties
				m["created_at"] = coll.CreatedAt.Format(rfc3339NanoFixed)
				m["modified_at"] = coll.ModifiedAt.Format(rfc3339NanoFixed)
				m["description"] = coll.Description
			}
			if user.UUID != "" {
				m["modified_by_user"] = map[string]interface{}{
					"email":     user.Email,
					"full_name": user.FullName,
					"username":  user.Username,
					"uuid":      user.UUID,
				}
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
