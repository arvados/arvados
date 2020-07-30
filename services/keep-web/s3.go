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
	"github.com/AdRoll/goamz/s3"
)

const s3MaxKeys = 1000

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
		for _, cmpt := range strings.Split(auth[17:], ",") {
			cmpt = strings.TrimSpace(cmpt)
			split := strings.SplitN(cmpt, "=", 2)
			if len(split) == 2 && split[0] == "Credential" {
				keyandscope := strings.Split(split[1], "/")
				if len(keyandscope[0]) > 0 {
					token = keyandscope[0]
					break
				}
			}
		}
		if token == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Println(w, "invalid V4 signature")
			return true
		}
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

	objectNameGiven := strings.Count(strings.TrimSuffix(r.URL.Path, "/"), "/") > 1

	switch {
	case r.Method == "GET" && !objectNameGiven:
		// Path is "/{uuid}" or "/{uuid}/", has no object name
		if _, ok := r.URL.Query()["versioning"]; ok {
			// GetBucketVersioning
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, xml.Header)
			fmt.Fprintln(w, `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`)
		} else {
			// ListObjects
			h.s3list(w, r, fs)
		}
		return true
	case r.Method == "GET" || r.Method == "HEAD":
		fspath := "/by_id" + r.URL.Path
		fi, err := fs.Stat(fspath)
		if r.Method == "HEAD" && !objectNameGiven {
			// HeadBucket
			if err != nil && fi.IsDir() {
				w.WriteHeader(http.StatusOK)
			} else if os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
			} else {
				http.Error(w, err.Error(), http.StatusBadGateway)
			}
			return true
		}
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
			err = fmt.Errorf("write to %q failed: close: %w", r.URL.Path, err)
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

func walkFS(fs arvados.CustomFileSystem, path string, ignoreNotFound bool, fn func(path string, fi os.FileInfo) error) error {
	f, err := fs.Open(path)
	if os.IsNotExist(err) && ignoreNotFound {
		return nil
	} else if err != nil {
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
			err = walkFS(fs, path+"/"+fi.Name(), false, fn)
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
	if mk, _ := strconv.ParseInt(r.FormValue("max-keys"), 10, 64); mk > 0 && mk < s3MaxKeys {
		params.maxKeys = int(mk)
	} else {
		params.maxKeys = s3MaxKeys
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
	if cut := strings.LastIndex(walkpath, "/"); cut >= 0 {
		walkpath = walkpath[:cut]
	} else {
		walkpath = ""
	}

	resp := s3.ListResp{
		Name:      strings.SplitN(r.URL.Path[1:], "/", 2)[0],
		Prefix:    params.prefix,
		Delimiter: params.delimiter,
		Marker:    params.marker,
		MaxKeys:   params.maxKeys,
	}
	commonPrefixes := map[string]bool{}
	err := walkFS(fs, strings.TrimSuffix(bucketdir+"/"+walkpath, "/"), true, func(path string, fi os.FileInfo) error {
		path = path[len(bucketdir)+1:]
		if len(path) <= len(params.prefix) {
			if path > params.prefix[:len(path)] {
				// with prefix "foobar", walking "fooz" means we're done
				return errDone
			}
			if path < params.prefix[:len(path)] {
				// with prefix "foobar", walking "foobag" is pointless
				return filepath.SkipDir
			}
			if fi.IsDir() && !strings.HasPrefix(params.prefix+"/", path+"/") {
				// with prefix "foo/bar", walking "fo"
				// is pointless (but walking "foo" or
				// "foo/bar" is necessary)
				return filepath.SkipDir
			}
			if len(path) < len(params.prefix) {
				// can't skip anything, and this entry
				// isn't in the results, so just
				// continue descent
				return nil
			}
		} else {
			if path[:len(params.prefix)] > params.prefix {
				// with prefix "foobar", nothing we
				// see after "foozzz" is relevant
				return errDone
			}
		}
		if path < params.marker || path < params.prefix {
			return nil
		}
		if fi.IsDir() {
			return nil
		}
		if params.delimiter != "" {
			idx := strings.Index(path[len(params.prefix):], params.delimiter)
			if idx >= 0 {
				// with prefix "foobar" and delimiter
				// "z", when we hit "foobar/baz", we
				// add "/baz" to commonPrefixes and
				// stop descending (note that even if
				// delimiter is "/" we don't add
				// anything to commonPrefixes when
				// seeing a dir: we wait until we see
				// a file, so we don't incorrectly
				// return results for empty dirs)
				commonPrefixes[path[:len(params.prefix)+idx+1]] = true
				return filepath.SkipDir
			}
		}
		if len(resp.Contents)+len(commonPrefixes) >= params.maxKeys {
			resp.IsTruncated = true
			if params.delimiter != "" {
				resp.NextMarker = path
			}
			return errDone
		}
		resp.Contents = append(resp.Contents, s3.Key{
			Key:          path,
			LastModified: fi.ModTime().UTC().Format("2006-01-02T15:04:05.999") + "Z",
			Size:         fi.Size(),
		})
		return nil
	})
	if err != nil && err != errDone {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if params.delimiter != "" {
		for prefix := range commonPrefixes {
			resp.CommonPrefixes = append(resp.CommonPrefixes, prefix)
			sort.Strings(resp.CommonPrefixes)
		}
	}
	wrappedResp := struct {
		XMLName string `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListBucketResult"`
		s3.ListResp
	}{"", resp}
	w.Header().Set("Content-Type", "application/xml")
	io.WriteString(w, xml.Header)
	if err := xml.NewEncoder(w).Encode(wrappedResp); err != nil {
		ctxlog.FromContext(r.Context()).WithError(err).Error("error writing xml response")
	}
}
