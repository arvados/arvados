// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/AdRoll/goamz/s3"
)

const (
	s3MaxKeys       = 1000
	s3SignAlgorithm = "AWS4-HMAC-SHA256"
	s3MaxClockSkew  = 5 * time.Minute
)

func hmacstring(msg string, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	io.WriteString(h, msg)
	return h.Sum(nil)
}

func hashdigest(h hash.Hash, payload string) string {
	io.WriteString(h, payload)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Signing key for given secret key and request attrs.
func s3signatureKey(key, datestamp, regionName, serviceName string) []byte {
	return hmacstring("aws4_request",
		hmacstring(serviceName,
			hmacstring(regionName,
				hmacstring(datestamp, []byte("AWS4"+key)))))
}

// Canonical query string for S3 V4 signature: sorted keys, spaces
// escaped as %20 instead of +, keyvalues joined with &.
func s3querystring(u *url.URL) string {
	keys := make([]string, 0, len(u.Query()))
	values := make(map[string]string, len(u.Query()))
	for k, vs := range u.Query() {
		k = strings.Replace(url.QueryEscape(k), "+", "%20", -1)
		keys = append(keys, k)
		for _, v := range vs {
			v = strings.Replace(url.QueryEscape(v), "+", "%20", -1)
			if values[k] != "" {
				values[k] += "&"
			}
			values[k] += k + "=" + v
		}
	}
	sort.Strings(keys)
	for i, k := range keys {
		keys[i] = values[k]
	}
	return strings.Join(keys, "&")
}

var reMultipleSlashChars = regexp.MustCompile(`//+`)

func s3stringToSign(alg, scope, signedHeaders string, r *http.Request) (string, error) {
	timefmt, timestr := "20060102T150405Z", r.Header.Get("X-Amz-Date")
	if timestr == "" {
		timefmt, timestr = time.RFC1123, r.Header.Get("Date")
	}
	t, err := time.Parse(timefmt, timestr)
	if err != nil {
		return "", fmt.Errorf("invalid timestamp %q: %s", timestr, err)
	}
	if skew := time.Now().Sub(t); skew < -s3MaxClockSkew || skew > s3MaxClockSkew {
		return "", errors.New("exceeded max clock skew")
	}

	var canonicalHeaders string
	for _, h := range strings.Split(signedHeaders, ";") {
		if h == "host" {
			canonicalHeaders += h + ":" + r.Host + "\n"
		} else {
			canonicalHeaders += h + ":" + r.Header.Get(h) + "\n"
		}
	}

	normalizedURL := *r.URL
	normalizedURL.RawPath = ""
	normalizedURL.Path = reMultipleSlashChars.ReplaceAllString(normalizedURL.Path, "/")
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", r.Method, normalizedURL.EscapedPath(), s3querystring(r.URL), canonicalHeaders, signedHeaders, r.Header.Get("X-Amz-Content-Sha256"))
	ctxlog.FromContext(r.Context()).Debugf("s3stringToSign: canonicalRequest %s", canonicalRequest)
	return fmt.Sprintf("%s\n%s\n%s\n%s", alg, r.Header.Get("X-Amz-Date"), scope, hashdigest(sha256.New(), canonicalRequest)), nil
}

func s3signature(secretKey, scope, signedHeaders, stringToSign string) (string, error) {
	// scope is {datestamp}/{region}/{service}/aws4_request
	drs := strings.Split(scope, "/")
	if len(drs) != 4 {
		return "", fmt.Errorf("invalid scope %q", scope)
	}
	key := s3signatureKey(secretKey, drs[0], drs[1], drs[2])
	return hashdigest(hmac.New(sha256.New, key), stringToSign), nil
}

// checks3signature verifies the given S3 V4 signature and returns the
// Arvados token that corresponds to the given accessKey. An error is
// returned if accessKey is not a valid token UUID or the signature
// does not match.
func (h *handler) checks3signature(r *http.Request) (string, error) {
	var key, scope, signedHeaders, signature string
	authstring := strings.TrimPrefix(r.Header.Get("Authorization"), s3SignAlgorithm+" ")
	for _, cmpt := range strings.Split(authstring, ",") {
		cmpt = strings.TrimSpace(cmpt)
		split := strings.SplitN(cmpt, "=", 2)
		switch {
		case len(split) != 2:
			// (?) ignore
		case split[0] == "Credential":
			keyandscope := strings.SplitN(split[1], "/", 2)
			if len(keyandscope) == 2 {
				key, scope = keyandscope[0], keyandscope[1]
			}
		case split[0] == "SignedHeaders":
			signedHeaders = split[1]
		case split[0] == "Signature":
			signature = split[1]
		}
	}

	client := (&arvados.Client{
		APIHost:  h.Config.cluster.Services.Controller.ExternalURL.Host,
		Insecure: h.Config.cluster.TLS.Insecure,
	}).WithRequestID(r.Header.Get("X-Request-Id"))
	var aca arvados.APIClientAuthorization
	var secret string
	var err error
	if len(key) == 27 && key[5:12] == "-gj3su-" {
		// Access key is the UUID of an Arvados token, secret
		// key is the secret part.
		ctx := arvados.ContextWithAuthorization(r.Context(), "Bearer "+h.Config.cluster.SystemRootToken)
		err = client.RequestAndDecodeContext(ctx, &aca, "GET", "arvados/v1/api_client_authorizations/"+key, nil, nil)
		secret = aca.APIToken
	} else {
		// Access key and secret key are both an entire
		// Arvados token or OIDC access token.
		ctx := arvados.ContextWithAuthorization(r.Context(), "Bearer "+key)
		err = client.RequestAndDecodeContext(ctx, &aca, "GET", "arvados/v1/api_client_authorizations/current", nil, nil)
		secret = key
	}
	if err != nil {
		ctxlog.FromContext(r.Context()).WithError(err).WithField("UUID", key).Info("token lookup failed")
		return "", errors.New("invalid access key")
	}
	stringToSign, err := s3stringToSign(s3SignAlgorithm, scope, signedHeaders, r)
	if err != nil {
		return "", err
	}
	expect, err := s3signature(secret, scope, signedHeaders, stringToSign)
	if err != nil {
		return "", err
	} else if expect != signature {
		return "", fmt.Errorf("signature does not match (scope %q signedHeaders %q stringToSign %q)", scope, signedHeaders, stringToSign)
	}
	return secret, nil
}

func s3ErrorResponse(w http.ResponseWriter, s3code string, message string, resource string, code int) {
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	var errstruct struct {
		Code      string
		Message   string
		Resource  string
		RequestId string
	}
	errstruct.Code = s3code
	errstruct.Message = message
	errstruct.Resource = resource
	errstruct.RequestId = ""
	enc := xml.NewEncoder(w)
	fmt.Fprint(w, xml.Header)
	enc.EncodeElement(errstruct, xml.StartElement{Name: xml.Name{Local: "Error"}})
}

var NoSuchKey = "NoSuchKey"
var NoSuchBucket = "NoSuchBucket"
var InvalidArgument = "InvalidArgument"
var InternalError = "InternalError"
var UnauthorizedAccess = "UnauthorizedAccess"
var InvalidRequest = "InvalidRequest"
var SignatureDoesNotMatch = "SignatureDoesNotMatch"

// serveS3 handles r and returns true if r is a request from an S3
// client, otherwise it returns false.
func (h *handler) serveS3(w http.ResponseWriter, r *http.Request) bool {
	var token string
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "AWS ") {
		split := strings.SplitN(auth[4:], ":", 2)
		if len(split) < 2 {
			s3ErrorResponse(w, InvalidRequest, "malformed Authorization header", r.URL.Path, http.StatusUnauthorized)
			return true
		}
		token = split[0]
	} else if strings.HasPrefix(auth, s3SignAlgorithm+" ") {
		t, err := h.checks3signature(r)
		if err != nil {
			s3ErrorResponse(w, SignatureDoesNotMatch, "signature verification failed: "+err.Error(), r.URL.Path, http.StatusForbidden)
			return true
		}
		token = t
	} else {
		return false
	}

	_, kc, client, release, err := h.getClients(r.Header.Get("X-Request-Id"), token)
	if err != nil {
		s3ErrorResponse(w, InternalError, "Pool failed: "+h.clientPool.Err().Error(), r.URL.Path, http.StatusInternalServerError)
		return true
	}
	defer release()

	fs := client.SiteFileSystem(kc)
	fs.ForwardSlashNameSubstitution(h.Config.cluster.Collections.ForwardSlashNameSubstitution)

	var objectNameGiven bool
	var bucketName string
	fspath := "/by_id"
	if id := parseCollectionIDFromDNSName(r.Host); id != "" {
		fspath += "/" + id
		bucketName = id
		objectNameGiven = strings.Count(strings.TrimSuffix(r.URL.Path, "/"), "/") > 0
	} else {
		bucketName = strings.SplitN(strings.TrimPrefix(r.URL.Path, "/"), "/", 2)[0]
		objectNameGiven = strings.Count(strings.TrimSuffix(r.URL.Path, "/"), "/") > 1
	}
	fspath += reMultipleSlashChars.ReplaceAllString(r.URL.Path, "/")

	switch {
	case r.Method == http.MethodGet && !objectNameGiven:
		// Path is "/{uuid}" or "/{uuid}/", has no object name
		if _, ok := r.URL.Query()["versioning"]; ok {
			// GetBucketVersioning
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, xml.Header)
			fmt.Fprintln(w, `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`)
		} else {
			// ListObjects
			h.s3list(bucketName, w, r, fs)
		}
		return true
	case r.Method == http.MethodGet || r.Method == http.MethodHead:
		fi, err := fs.Stat(fspath)
		if r.Method == "HEAD" && !objectNameGiven {
			// HeadBucket
			if err == nil && fi.IsDir() {
				w.WriteHeader(http.StatusOK)
			} else if os.IsNotExist(err) {
				s3ErrorResponse(w, NoSuchBucket, "The specified bucket does not exist.", r.URL.Path, http.StatusNotFound)
			} else {
				s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusBadGateway)
			}
			return true
		}
		if err == nil && fi.IsDir() && objectNameGiven && strings.HasSuffix(fspath, "/") && h.Config.cluster.Collections.S3FolderObjects {
			w.Header().Set("Content-Type", "application/x-directory")
			w.WriteHeader(http.StatusOK)
			return true
		}
		if os.IsNotExist(err) ||
			(err != nil && err.Error() == "not a directory") ||
			(fi != nil && fi.IsDir()) {
			s3ErrorResponse(w, NoSuchKey, "The specified key does not exist.", r.URL.Path, http.StatusNotFound)
			return true
		}
		// shallow copy r, and change URL path
		r := *r
		r.URL.Path = fspath
		http.FileServer(fs).ServeHTTP(w, &r)
		return true
	case r.Method == http.MethodPut:
		if !objectNameGiven {
			s3ErrorResponse(w, InvalidArgument, "Missing object name in PUT request.", r.URL.Path, http.StatusBadRequest)
			return true
		}
		var objectIsDir bool
		if strings.HasSuffix(fspath, "/") {
			if !h.Config.cluster.Collections.S3FolderObjects {
				s3ErrorResponse(w, InvalidArgument, "invalid object name: trailing slash", r.URL.Path, http.StatusBadRequest)
				return true
			}
			n, err := r.Body.Read(make([]byte, 1))
			if err != nil && err != io.EOF {
				s3ErrorResponse(w, InternalError, fmt.Sprintf("error reading request body: %s", err), r.URL.Path, http.StatusInternalServerError)
				return true
			} else if n > 0 {
				s3ErrorResponse(w, InvalidArgument, "cannot create object with trailing '/' char unless content is empty", r.URL.Path, http.StatusBadRequest)
				return true
			} else if strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0] != "application/x-directory" {
				s3ErrorResponse(w, InvalidArgument, "cannot create object with trailing '/' char unless Content-Type is 'application/x-directory'", r.URL.Path, http.StatusBadRequest)
				return true
			}
			// Given PUT "foo/bar/", we'll use "foo/bar/."
			// in the "ensure parents exist" block below,
			// and then we'll be done.
			fspath += "."
			objectIsDir = true
		}
		fi, err := fs.Stat(fspath)
		if err != nil && err.Error() == "not a directory" {
			// requested foo/bar, but foo is a file
			s3ErrorResponse(w, InvalidArgument, "object name conflicts with existing object", r.URL.Path, http.StatusBadRequest)
			return true
		}
		if strings.HasSuffix(r.URL.Path, "/") && err == nil && !fi.IsDir() {
			// requested foo/bar/, but foo/bar is a file
			s3ErrorResponse(w, InvalidArgument, "object name conflicts with existing object", r.URL.Path, http.StatusBadRequest)
			return true
		}
		// create missing parent/intermediate directories, if any
		for i, c := range fspath {
			if i > 0 && c == '/' {
				dir := fspath[:i]
				if strings.HasSuffix(dir, "/") {
					err = errors.New("invalid object name (consecutive '/' chars)")
					s3ErrorResponse(w, InvalidArgument, err.Error(), r.URL.Path, http.StatusBadRequest)
					return true
				}
				err = fs.Mkdir(dir, 0755)
				if err == arvados.ErrInvalidArgument {
					// Cannot create a directory
					// here.
					err = fmt.Errorf("mkdir %q failed: %w", dir, err)
					s3ErrorResponse(w, InvalidArgument, err.Error(), r.URL.Path, http.StatusBadRequest)
					return true
				} else if err != nil && !os.IsExist(err) {
					err = fmt.Errorf("mkdir %q failed: %w", dir, err)
					s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusInternalServerError)
					return true
				}
			}
		}
		if !objectIsDir {
			f, err := fs.OpenFile(fspath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			if os.IsNotExist(err) {
				f, err = fs.OpenFile(fspath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
			}
			if err != nil {
				err = fmt.Errorf("open %q failed: %w", r.URL.Path, err)
				s3ErrorResponse(w, InvalidArgument, err.Error(), r.URL.Path, http.StatusBadRequest)
				return true
			}
			defer f.Close()
			_, err = io.Copy(f, r.Body)
			if err != nil {
				err = fmt.Errorf("write to %q failed: %w", r.URL.Path, err)
				s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusBadGateway)
				return true
			}
			err = f.Close()
			if err != nil {
				err = fmt.Errorf("write to %q failed: close: %w", r.URL.Path, err)
				s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusBadGateway)
				return true
			}
		}
		err = fs.Sync()
		if err != nil {
			err = fmt.Errorf("sync failed: %w", err)
			s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusInternalServerError)
			return true
		}
		w.WriteHeader(http.StatusOK)
		return true
	case r.Method == http.MethodDelete:
		if !objectNameGiven || r.URL.Path == "/" {
			s3ErrorResponse(w, InvalidArgument, "missing object name in DELETE request", r.URL.Path, http.StatusBadRequest)
			return true
		}
		if strings.HasSuffix(fspath, "/") {
			fspath = strings.TrimSuffix(fspath, "/")
			fi, err := fs.Stat(fspath)
			if os.IsNotExist(err) {
				w.WriteHeader(http.StatusNoContent)
				return true
			} else if err != nil {
				s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusInternalServerError)
				return true
			} else if !fi.IsDir() {
				// if "foo" exists and is a file, then
				// "foo/" doesn't exist, so we say
				// delete was successful.
				w.WriteHeader(http.StatusNoContent)
				return true
			}
		} else if fi, err := fs.Stat(fspath); err == nil && fi.IsDir() {
			// if "foo" is a dir, it is visible via S3
			// only as "foo/", not "foo" -- so we leave
			// the dir alone and return 204 to indicate
			// that "foo" does not exist.
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		err = fs.Remove(fspath)
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNoContent)
			return true
		}
		if err != nil {
			err = fmt.Errorf("rm failed: %w", err)
			s3ErrorResponse(w, InvalidArgument, err.Error(), r.URL.Path, http.StatusBadRequest)
			return true
		}
		err = fs.Sync()
		if err != nil {
			err = fmt.Errorf("sync failed: %w", err)
			s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusInternalServerError)
			return true
		}
		w.WriteHeader(http.StatusNoContent)
		return true
	default:
		s3ErrorResponse(w, InvalidRequest, "method not allowed", r.URL.Path, http.StatusMethodNotAllowed)

		return true
	}
}

// Call fn on the given path (directory) and its contents, in
// lexicographic order.
//
// If isRoot==true and path is not a directory, return nil.
//
// If fn returns filepath.SkipDir when called on a directory, don't
// descend into that directory.
func walkFS(fs arvados.CustomFileSystem, path string, isRoot bool, fn func(path string, fi os.FileInfo) error) error {
	if isRoot {
		fi, err := fs.Stat(path)
		if os.IsNotExist(err) || (err == nil && !fi.IsDir()) {
			return nil
		} else if err != nil {
			return err
		}
		err = fn(path, fi)
		if err == filepath.SkipDir {
			return nil
		} else if err != nil {
			return err
		}
	}
	f, err := fs.Open(path)
	if os.IsNotExist(err) && isRoot {
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

func (h *handler) s3list(bucket string, w http.ResponseWriter, r *http.Request, fs arvados.CustomFileSystem) {
	var params struct {
		delimiter string
		marker    string
		maxKeys   int
		prefix    string
	}
	params.delimiter = r.FormValue("delimiter")
	params.marker = r.FormValue("marker")
	if mk, _ := strconv.ParseInt(r.FormValue("max-keys"), 10, 64); mk > 0 && mk < s3MaxKeys {
		params.maxKeys = int(mk)
	} else {
		params.maxKeys = s3MaxKeys
	}
	params.prefix = r.FormValue("prefix")

	bucketdir := "by_id/" + bucket
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

	type commonPrefix struct {
		Prefix string
	}
	type listResp struct {
		XMLName string `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListBucketResult"`
		s3.ListResp
		// s3.ListResp marshals an empty tag when
		// CommonPrefixes is nil, which confuses some clients.
		// Fix by using this nested struct instead.
		CommonPrefixes []commonPrefix
		// Similarly, we need omitempty here, because an empty
		// tag confuses some clients (e.g.,
		// github.com/aws/aws-sdk-net never terminates its
		// paging loop).
		NextMarker string `xml:"NextMarker,omitempty"`
		// ListObjectsV2 has a KeyCount response field.
		KeyCount int
	}
	resp := listResp{
		ListResp: s3.ListResp{
			Name:      bucket,
			Prefix:    params.prefix,
			Delimiter: params.delimiter,
			Marker:    params.marker,
			MaxKeys:   params.maxKeys,
		},
	}
	commonPrefixes := map[string]bool{}
	err := walkFS(fs, strings.TrimSuffix(bucketdir+"/"+walkpath, "/"), true, func(path string, fi os.FileInfo) error {
		if path == bucketdir {
			return nil
		}
		path = path[len(bucketdir)+1:]
		filesize := fi.Size()
		if fi.IsDir() {
			path += "/"
			filesize = 0
		}
		if len(path) <= len(params.prefix) {
			if path > params.prefix[:len(path)] {
				// with prefix "foobar", walking "fooz" means we're done
				return errDone
			}
			if path < params.prefix[:len(path)] {
				// with prefix "foobar", walking "foobag" is pointless
				return filepath.SkipDir
			}
			if fi.IsDir() && !strings.HasPrefix(params.prefix+"/", path) {
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
		if fi.IsDir() && !h.Config.cluster.Collections.S3FolderObjects {
			// Note we don't add anything to
			// commonPrefixes here even if delimiter is
			// "/". We descend into the directory, and
			// return a commonPrefix only if we end up
			// finding a regular file inside it.
			return nil
		}
		if params.delimiter != "" {
			idx := strings.Index(path[len(params.prefix):], params.delimiter)
			if idx >= 0 {
				// with prefix "foobar" and delimiter
				// "z", when we hit "foobar/baz", we
				// add "/baz" to commonPrefixes and
				// stop descending.
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
			Size:         filesize,
		})
		return nil
	})
	if err != nil && err != errDone {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if params.delimiter != "" {
		resp.CommonPrefixes = make([]commonPrefix, 0, len(commonPrefixes))
		for prefix := range commonPrefixes {
			resp.CommonPrefixes = append(resp.CommonPrefixes, commonPrefix{prefix})
		}
		sort.Slice(resp.CommonPrefixes, func(i, j int) bool { return resp.CommonPrefixes[i].Prefix < resp.CommonPrefixes[j].Prefix })
	}
	resp.KeyCount = len(resp.Contents)
	w.Header().Set("Content-Type", "application/xml")
	io.WriteString(w, xml.Header)
	if err := xml.NewEncoder(w).Encode(resp); err != nil {
		ctxlog.FromContext(r.Context()).WithError(err).Error("error writing xml response")
	}
}
