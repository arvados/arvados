// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/tmp/GOPATH/src/github.com/AdRoll/goamz/s3"
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

	fs := client.SiteFileSystem(kc)
	fs.ForwardSlashNameSubstitution(h.Config.cluster.Collections.ForwardSlashNameSubstitution)

	switch {
	case r.Method == "GET" && strings.Count(strings.TrimSuffix(r.URL.Path, "/"), "/") == 1:
		// Path is "/{uuid}" or "/{uuid}/", has no object name
		h.s3list(w, r, fs)
		return true
	case r.Method == "GET":
		fspath := "/by_id" + r.URL.Path
		fi, err := fs.Stat(fspath)
		if os.IsNotExist(err) ||
			(err != nil && err.Error() == "not a directory") ||
			(fi != nil && fi.IsDir()) {
			http.Error(w, "not found", http.StatusNotFound)
			return true
		}
		// shallow copy r, and change URL path
		r := *r
		r.URL.Path = fspath
		http.FileServer(fs).ServeHTTP(w, &r)
		return true
	case r.Method == "PUT":
		if strings.HasSuffix(r.URL.Path, "/") {
			http.Error(w, "invalid object name (trailing '/' char)", http.StatusBadRequest)
			return true
		}
		fspath := "by_id" + r.URL.Path
		_, err = fs.Stat(fspath)
		if err != nil && err.Error() == "not a directory" {
			// requested foo/bar, but foo is a file
			http.Error(w, "object name conflicts with existing object", http.StatusBadRequest)
			return true
		}
		f, err := fs.OpenFile(fspath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
		if os.IsNotExist(err) {
			// create missing intermediate directories, then try again
			for i, c := range fspath {
				if i > 0 && c == '/' {
					dir := fspath[:i]
					if strings.HasSuffix(dir, "/") {
						err = errors.New("invalid object name (consecutive '/' chars)")
						http.Error(w, err.Error(), http.StatusBadRequest)
						return true
					}
					err := fs.Mkdir(dir, 0755)
					if err != nil && err != os.ErrExist {
						err = fmt.Errorf("mkdir %q failed: %w", dir, err)
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return true
					}
				}
			}
			f, err = fs.OpenFile(fspath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
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

func walkFS(fs arvados.CustomFileSystem, path string, fn func(path string, fi os.FileInfo) error) error {
	f, err := fs.Open(path)
	if err != nil {
		return fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()
	if path == "/" {
		path = ""
	}
	fis, err := f.Readdir(-1)
	if err != nil {
		return err
	}
	sort.Slice(fis, func(i, j int) bool { return fis[i].Name() < fis[j].Name() })
	for _, fi := range fis {
		err = fn(path+"/"+fi.Name(), fi)
		if err == filepath.SkipDir {
			continue
		} else if err != nil {
			return err
		}
		if fi.IsDir() {
			err = walkFS(fs, path+"/"+fi.Name(), fn)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

var errDone = errors.New("done")

func (h *handler) s3list(w http.ResponseWriter, r *http.Request, fs arvados.CustomFileSystem) {
	var params struct {
		bucket    string
		delimiter string
		marker    string
		maxKeys   int
		prefix    string
	}
	params.bucket = strings.SplitN(r.URL.Path[1:], "/", 2)[0]
	params.delimiter = r.FormValue("delimiter")
	params.marker = r.FormValue("marker")
	if mk, _ := strconv.ParseInt(r.FormValue("max-keys"), 10, 64); mk > 0 {
		params.maxKeys = int(mk)
	} else {
		params.maxKeys = 100
	}
	params.prefix = r.FormValue("prefix")

	bucketdir := "by_id/" + params.bucket
	// walkpath is the directory (relative to bucketdir) we need
	// to walk: the innermost directory that is guaranteed to
	// contain all paths that have the requested prefix. Examples:
	// prefix "foo/bar"  => walkpath "foo"
	// prefix "foo/bar/" => walkpath "foo/bar"
	// prefix "foo"      => walkpath ""
	// prefix ""         => walkpath ""
	walkpath := params.prefix
	if !strings.HasSuffix(walkpath, "/") {
		walkpath, _ = filepath.Split(walkpath)
	}
	walkpath = strings.TrimSuffix(walkpath, "/")

	type commonPrefix struct {
		Prefix string
	}
	type serverListResponse struct {
		s3.ListResp
		CommonPrefixes []commonPrefix
	}
	resp := serverListResponse{ListResp: s3.ListResp{
		Name:      strings.SplitN(r.URL.Path[1:], "/", 2)[0],
		Prefix:    params.prefix,
		Delimiter: params.delimiter,
		Marker:    params.marker,
		MaxKeys:   params.maxKeys,
	}}
	err := walkFS(fs, strings.TrimSuffix(bucketdir+"/"+walkpath, "/"), func(path string, fi os.FileInfo) error {
		path = path[len(bucketdir)+1:]
		if !strings.HasPrefix(path, params.prefix) {
			return filepath.SkipDir
		}
		if fi.IsDir() {
			return nil
		}
		if path < params.marker {
			return nil
		}
		// TODO: check delimiter, roll up common prefixes
		if len(resp.Contents)+len(resp.CommonPrefixes) >= params.maxKeys {
			resp.IsTruncated = true
			if params.delimiter == "" {
				resp.NextMarker = path
			}
			return errDone
		}
		resp.ListResp.Contents = append(resp.ListResp.Contents, s3.Key{
			Key: path,
		})
		return nil
	})
	if err != nil && err != errDone {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := xml.NewEncoder(w).Encode(resp); err != nil {
		ctxlog.FromContext(r.Context()).WithError(err).Error("error writing xml response")
	}
}
