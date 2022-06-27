// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/AdRoll/goamz/s3"
)

const (
	s3MaxKeys       = 1000
	s3SignAlgorithm = "AWS4-HMAC-SHA256"
	s3MaxClockSkew  = 5 * time.Minute
)

type commonPrefix struct {
	Prefix string
}

type listV1Resp struct {
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

type listV2Resp struct {
	XMLName               string `xml:"http://s3.amazonaws.com/doc/2006-03-01/ ListBucketResult"`
	IsTruncated           bool
	Contents              []s3.Key
	Name                  string
	Prefix                string
	Delimiter             string
	MaxKeys               int
	CommonPrefixes        []commonPrefix
	EncodingType          string `xml:",omitempty"`
	KeyCount              int
	ContinuationToken     string `xml:",omitempty"`
	NextContinuationToken string `xml:",omitempty"`
	StartAfter            string `xml:",omitempty"`
}

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

	normalizedPath := normalizePath(r.URL.Path)
	ctxlog.FromContext(r.Context()).Debugf("normalizedPath %q", normalizedPath)
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s", r.Method, normalizedPath, s3querystring(r.URL), canonicalHeaders, signedHeaders, r.Header.Get("X-Amz-Content-Sha256"))
	ctxlog.FromContext(r.Context()).Debugf("s3stringToSign: canonicalRequest %s", canonicalRequest)
	return fmt.Sprintf("%s\n%s\n%s\n%s", alg, r.Header.Get("X-Amz-Date"), scope, hashdigest(sha256.New(), canonicalRequest)), nil
}

func normalizePath(s string) string {
	// (url.URL).EscapedPath() would be incorrect here. AWS
	// documentation specifies the URL path should be normalized
	// according to RFC 3986, i.e., unescaping ALPHA / DIGIT / "-"
	// / "." / "_" / "~". The implication is that everything other
	// than those chars (and "/") _must_ be percent-encoded --
	// even chars like ";" and "," that are not normally
	// percent-encoded in paths.
	out := ""
	for _, c := range []byte(reMultipleSlashChars.ReplaceAllString(s, "/")) {
		if (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' ||
			c == '.' ||
			c == '_' ||
			c == '~' ||
			c == '/' {
			out += string(c)
		} else {
			out += fmt.Sprintf("%%%02X", c)
		}
	}
	return out
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

var v2tokenUnderscore = regexp.MustCompile(`^v2_[a-z0-9]{5}-gj3su-[a-z0-9]{15}_`)

func unescapeKey(key string) string {
	if v2tokenUnderscore.MatchString(key) {
		// Entire Arvados token, with "/" replaced by "_" to
		// avoid colliding with the Authorization header
		// format.
		return strings.Replace(key, "_", "/", -1)
	} else if s, err := url.PathUnescape(key); err == nil {
		return s
	} else {
		return key
	}
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
		APIHost:  h.Cluster.Services.Controller.ExternalURL.Host,
		Insecure: h.Cluster.TLS.Insecure,
	}).WithRequestID(r.Header.Get("X-Request-Id"))
	var aca arvados.APIClientAuthorization
	var secret string
	var err error
	if len(key) == 27 && key[5:12] == "-gj3su-" {
		// Access key is the UUID of an Arvados token, secret
		// key is the secret part.
		ctx := arvados.ContextWithAuthorization(r.Context(), "Bearer "+h.Cluster.SystemRootToken)
		err = client.RequestAndDecodeContext(ctx, &aca, "GET", "arvados/v1/api_client_authorizations/"+key, nil, nil)
		secret = aca.APIToken
	} else {
		// Access key and secret key are both an entire
		// Arvados token or OIDC access token.
		ctx := arvados.ContextWithAuthorization(r.Context(), "Bearer "+unescapeKey(key))
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
	return aca.TokenV2(), nil
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

var reRawQueryIndicatesAPI = regexp.MustCompile(`^[a-z]+(&|$)`)

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
		token = unescapeKey(split[0])
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

	var err error
	var fs arvados.CustomFileSystem
	var arvclient *arvadosclient.ArvadosClient
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		// Use a single session (cached FileSystem) across
		// multiple read requests.
		var sess *cachedSession
		fs, sess, err = h.Cache.GetSession(token)
		if err != nil {
			s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusInternalServerError)
			return true
		}
		arvclient = sess.arvadosclient
	} else {
		// Create a FileSystem for this request, to avoid
		// exposing incomplete write operations to concurrent
		// requests.
		var kc *keepclient.KeepClient
		var release func()
		var client *arvados.Client
		arvclient, kc, client, release, err = h.getClients(r.Header.Get("X-Request-Id"), token)
		if err != nil {
			s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusInternalServerError)
			return true
		}
		defer release()
		fs = client.SiteFileSystem(kc)
		fs.ForwardSlashNameSubstitution(h.Cluster.Collections.ForwardSlashNameSubstitution)
	}

	var objectNameGiven bool
	var bucketName string
	fspath := "/by_id"
	if id := arvados.CollectionIDFromDNSName(r.Host); id != "" {
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
		} else if _, ok = r.URL.Query()["location"]; ok {
			// GetBucketLocation
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, xml.Header)
			fmt.Fprintln(w, `<LocationConstraint><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">`+
				h.Cluster.ClusterID+
				`</LocationConstraint></LocationConstraint>`)
		} else if reRawQueryIndicatesAPI.MatchString(r.URL.RawQuery) {
			// GetBucketWebsite ("GET /bucketid/?website"), GetBucketTagging, etc.
			s3ErrorResponse(w, InvalidRequest, "API not supported", r.URL.Path+"?"+r.URL.RawQuery, http.StatusBadRequest)
		} else {
			// ListObjects
			h.s3list(bucketName, w, r, fs)
		}
		return true
	case r.Method == http.MethodGet || r.Method == http.MethodHead:
		if reRawQueryIndicatesAPI.MatchString(r.URL.RawQuery) {
			// GetObjectRetention ("GET /bucketid/objectid?retention&versionID=..."), etc.
			s3ErrorResponse(w, InvalidRequest, "API not supported", r.URL.Path+"?"+r.URL.RawQuery, http.StatusBadRequest)
			return true
		}
		fi, err := fs.Stat(fspath)
		if r.Method == "HEAD" && !objectNameGiven {
			// HeadBucket
			if err == nil && fi.IsDir() {
				setFileInfoHeaders(w.Header(), fs, fspath)
				w.WriteHeader(http.StatusOK)
			} else if os.IsNotExist(err) {
				s3ErrorResponse(w, NoSuchBucket, "The specified bucket does not exist.", r.URL.Path, http.StatusNotFound)
			} else {
				s3ErrorResponse(w, InternalError, err.Error(), r.URL.Path, http.StatusBadGateway)
			}
			return true
		}
		if err == nil && fi.IsDir() && objectNameGiven && strings.HasSuffix(fspath, "/") && h.Cluster.Collections.S3FolderObjects {
			setFileInfoHeaders(w.Header(), fs, fspath)
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

		tokenUser, err := h.Cache.GetTokenUser(token)
		if !h.userPermittedToUploadOrDownload(r.Method, tokenUser) {
			http.Error(w, "Not permitted", http.StatusForbidden)
			return true
		}
		h.logUploadOrDownload(r, arvclient, fs, fspath, nil, tokenUser)

		// shallow copy r, and change URL path
		r := *r
		r.URL.Path = fspath
		setFileInfoHeaders(w.Header(), fs, fspath)
		http.FileServer(fs).ServeHTTP(w, &r)
		return true
	case r.Method == http.MethodPut:
		if reRawQueryIndicatesAPI.MatchString(r.URL.RawQuery) {
			// PutObjectAcl ("PUT /bucketid/objectid?acl&versionID=..."), etc.
			s3ErrorResponse(w, InvalidRequest, "API not supported", r.URL.Path+"?"+r.URL.RawQuery, http.StatusBadRequest)
			return true
		}
		if !objectNameGiven {
			s3ErrorResponse(w, InvalidArgument, "Missing object name in PUT request.", r.URL.Path, http.StatusBadRequest)
			return true
		}
		var objectIsDir bool
		if strings.HasSuffix(fspath, "/") {
			if !h.Cluster.Collections.S3FolderObjects {
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
				if errors.Is(err, arvados.ErrInvalidArgument) || errors.Is(err, arvados.ErrInvalidOperation) {
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

			tokenUser, err := h.Cache.GetTokenUser(token)
			if !h.userPermittedToUploadOrDownload(r.Method, tokenUser) {
				http.Error(w, "Not permitted", http.StatusForbidden)
				return true
			}
			h.logUploadOrDownload(r, arvclient, fs, fspath, nil, tokenUser)

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
		// Ensure a subsequent read operation will see the changes.
		h.Cache.ResetSession(token)
		w.WriteHeader(http.StatusOK)
		return true
	case r.Method == http.MethodDelete:
		if reRawQueryIndicatesAPI.MatchString(r.URL.RawQuery) {
			// DeleteObjectTagging ("DELETE /bucketid/objectid?tagging&versionID=..."), etc.
			s3ErrorResponse(w, InvalidRequest, "API not supported", r.URL.Path+"?"+r.URL.RawQuery, http.StatusBadRequest)
			return true
		}
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
		// Ensure a subsequent read operation will see the changes.
		h.Cache.ResetSession(token)
		w.WriteHeader(http.StatusNoContent)
		return true
	default:
		s3ErrorResponse(w, InvalidRequest, "method not allowed", r.URL.Path, http.StatusMethodNotAllowed)
		return true
	}
}

func setFileInfoHeaders(header http.Header, fs arvados.CustomFileSystem, path string) {
	path = strings.TrimSuffix(path, "/")
	var props map[string]interface{}
	for {
		fi, err := fs.Stat(path)
		if err != nil {
			return
		}
		switch src := fi.Sys().(type) {
		case *arvados.Collection:
			props = src.Properties
		case *arvados.Group:
			props = src.Properties
		default:
			// Try parent
			cut := strings.LastIndexByte(path, '/')
			if cut < 0 {
				return
			}
			path = path[:cut]
			continue
		}
		break
	}
	for k, v := range props {
		if !validMIMEHeaderKey(k) {
			continue
		}
		k = "x-amz-meta-" + k
		if s, ok := v.(string); ok {
			header.Set(k, s)
		} else if j, err := json.Marshal(v); err == nil {
			header.Set(k, string(j))
		}
	}
}

func validMIMEHeaderKey(k string) bool {
	check := "z-" + k
	return check != textproto.CanonicalMIMEHeaderKey(check)
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
		v2                bool
		delimiter         string
		maxKeys           int
		prefix            string
		marker            string // decoded continuationToken (v2) or provided by client (v1)
		startAfter        string // v2
		continuationToken string // v2
		encodingTypeURL   bool   // v2
	}
	params.delimiter = r.FormValue("delimiter")
	if mk, _ := strconv.ParseInt(r.FormValue("max-keys"), 10, 64); mk > 0 && mk < s3MaxKeys {
		params.maxKeys = int(mk)
	} else {
		params.maxKeys = s3MaxKeys
	}
	params.prefix = r.FormValue("prefix")
	switch r.FormValue("list-type") {
	case "":
	case "2":
		params.v2 = true
	default:
		http.Error(w, "invalid list-type parameter", http.StatusBadRequest)
		return
	}
	if params.v2 {
		params.continuationToken = r.FormValue("continuation-token")
		marker, err := base64.StdEncoding.DecodeString(params.continuationToken)
		if err != nil {
			http.Error(w, "invalid continuation token", http.StatusBadRequest)
			return
		}
		params.marker = string(marker)
		params.startAfter = r.FormValue("start-after")
		switch r.FormValue("encoding-type") {
		case "":
		case "url":
			params.encodingTypeURL = true
		default:
			http.Error(w, "invalid encoding-type parameter", http.StatusBadRequest)
			return
		}
	} else {
		params.marker = r.FormValue("marker")
	}

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

	resp := listV2Resp{
		Name:              bucket,
		Prefix:            params.prefix,
		Delimiter:         params.delimiter,
		MaxKeys:           params.maxKeys,
		ContinuationToken: r.FormValue("continuation-token"),
		StartAfter:        params.startAfter,
	}
	nextMarker := ""

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
		if path < params.marker || path < params.prefix || path <= params.startAfter {
			return nil
		}
		if fi.IsDir() && !h.Cluster.Collections.S3FolderObjects {
			// Note we don't add anything to
			// commonPrefixes here even if delimiter is
			// "/". We descend into the directory, and
			// return a commonPrefix only if we end up
			// finding a regular file inside it.
			return nil
		}
		if len(resp.Contents)+len(commonPrefixes) >= params.maxKeys {
			resp.IsTruncated = true
			if params.delimiter != "" || params.v2 {
				nextMarker = path
			}
			return errDone
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
	var respV1orV2 interface{}

	if params.encodingTypeURL {
		// https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListObjectsV2.html
		// "If you specify the encoding-type request
		// parameter, Amazon S3 includes this element in the
		// response, and returns encoded key name values in
		// the following response elements:
		//
		// Delimiter, Prefix, Key, and StartAfter.
		//
		// 	Type: String
		//
		// Valid Values: url"
		//
		// This is somewhat vague but in practice it appears
		// to mean x-www-form-urlencoded as in RFC1866 8.2.1
		// para 1 (encode space as "+") rather than straight
		// percent-encoding as in RFC1738 2.2.  Presumably,
		// the intent is to allow the client to decode XML and
		// then paste the strings directly into another URI
		// query or POST form like "https://host/path?foo=" +
		// foo + "&bar=" + bar.
		resp.EncodingType = "url"
		resp.Delimiter = url.QueryEscape(resp.Delimiter)
		resp.Prefix = url.QueryEscape(resp.Prefix)
		resp.StartAfter = url.QueryEscape(resp.StartAfter)
		for i, ent := range resp.Contents {
			ent.Key = url.QueryEscape(ent.Key)
			resp.Contents[i] = ent
		}
		for i, ent := range resp.CommonPrefixes {
			ent.Prefix = url.QueryEscape(ent.Prefix)
			resp.CommonPrefixes[i] = ent
		}
	}

	if params.v2 {
		resp.NextContinuationToken = base64.StdEncoding.EncodeToString([]byte(nextMarker))
		respV1orV2 = resp
	} else {
		respV1orV2 = listV1Resp{
			CommonPrefixes: resp.CommonPrefixes,
			NextMarker:     nextMarker,
			KeyCount:       resp.KeyCount,
			ListResp: s3.ListResp{
				IsTruncated: resp.IsTruncated,
				Name:        bucket,
				Prefix:      params.prefix,
				Delimiter:   params.delimiter,
				Marker:      params.marker,
				MaxKeys:     params.maxKeys,
				Contents:    resp.Contents,
			},
		}
	}

	w.Header().Set("Content-Type", "application/xml")
	io.WriteString(w, xml.Header)
	if err := xml.NewEncoder(w).Encode(respV1orV2); err != nil {
		ctxlog.FromContext(r.Context()).WithError(err).Error("error writing xml response")
	}
}
