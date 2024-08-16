// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/webdavfs"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
)

type handler struct {
	Cache   cache
	Cluster *arvados.Cluster
	metrics *metrics

	lockMtx    sync.Mutex
	lock       map[string]*sync.RWMutex
	lockTidied time.Time

	fileEventLogs         map[fileEventLog]time.Time
	fileEventLogsMtx      sync.Mutex
	fileEventLogsNextTidy time.Time

	s3SecretCache         map[string]*cachedS3Secret
	s3SecretCacheMtx      sync.Mutex
	s3SecretCacheNextTidy time.Time
}

var urlPDHDecoder = strings.NewReplacer(" ", "+", "-", "+")

var notFoundMessage = "Not Found"
var unauthorizedMessage = "401 Unauthorized\n\nA valid Arvados token must be provided to access this resource."

// parseCollectionIDFromURL returns a UUID or PDH if s is a UUID or a
// PDH (even if it is a PDH with "+" replaced by " " or "-");
// otherwise "".
func parseCollectionIDFromURL(s string) string {
	if arvadosclient.UUIDMatch(s) {
		return s
	}
	if pdh := urlPDHDecoder.Replace(s); arvadosclient.PDHMatch(pdh) {
		return pdh
	}
	return ""
}

func (h *handler) serveStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(struct{ Version string }{cmd.Version.String()})
}

type errorWithHTTPStatus interface {
	HTTPStatus() int
}

// updateOnSuccess wraps httpserver.ResponseWriter. If the handler
// sends an HTTP header indicating success, updateOnSuccess first
// calls the provided update func. If the update func fails, an error
// response is sent (using the error's HTTP status or 500 if none),
// and the status code and body sent by the handler are ignored (all
// response writes return the update error).
type updateOnSuccess struct {
	httpserver.ResponseWriter
	logger     logrus.FieldLogger
	update     func() error
	sentHeader bool
	err        error
}

func (uos *updateOnSuccess) Write(p []byte) (int, error) {
	if !uos.sentHeader {
		uos.WriteHeader(http.StatusOK)
	}
	if uos.err != nil {
		return 0, uos.err
	}
	return uos.ResponseWriter.Write(p)
}

func (uos *updateOnSuccess) WriteHeader(code int) {
	if !uos.sentHeader {
		uos.sentHeader = true
		if code >= 200 && code < 400 {
			if uos.err = uos.update(); uos.err != nil {
				code := http.StatusInternalServerError
				if he := errorWithHTTPStatus(nil); errors.As(uos.err, &he) {
					code = he.HTTPStatus()
				}
				uos.logger.WithError(uos.err).Errorf("update() returned %T error, changing response to HTTP %d", uos.err, code)
				http.Error(uos.ResponseWriter, uos.err.Error(), code)
				return
			}
		}
	}
	uos.ResponseWriter.WriteHeader(code)
}

var (
	corsAllowHeadersHeader = strings.Join([]string{
		"Authorization", "Content-Type", "Range",
		// WebDAV request headers:
		"Depth", "Destination", "If", "Lock-Token", "Overwrite", "Timeout", "Cache-Control",
	}, ", ")
	writeMethod = map[string]bool{
		"COPY":      true,
		"DELETE":    true,
		"LOCK":      true,
		"MKCOL":     true,
		"MOVE":      true,
		"PROPPATCH": true,
		"PUT":       true,
		"RMCOL":     true,
		"UNLOCK":    true,
	}
	webdavMethod = map[string]bool{
		"COPY":      true,
		"DELETE":    true,
		"LOCK":      true,
		"MKCOL":     true,
		"MOVE":      true,
		"OPTIONS":   true,
		"PROPFIND":  true,
		"PROPPATCH": true,
		"PUT":       true,
		"RMCOL":     true,
		"UNLOCK":    true,
	}
	browserMethod = map[string]bool{
		"GET":  true,
		"HEAD": true,
		"POST": true,
	}
	// top-level dirs to serve with siteFS
	siteFSDir = map[string]bool{
		"":      true, // root directory
		"by_id": true,
		"users": true,
	}
)

func stripDefaultPort(host string) string {
	// Will consider port 80 and port 443 to be the same vhost.  I think that's fine.
	u := &url.URL{Host: host}
	if p := u.Port(); p == "80" || p == "443" {
		return strings.ToLower(u.Hostname())
	} else {
		return strings.ToLower(host)
	}
}

// CheckHealth implements service.Handler.
func (h *handler) CheckHealth() error {
	return nil
}

// Done implements service.Handler.
func (h *handler) Done() <-chan struct{} {
	return nil
}

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(wOrig http.ResponseWriter, r *http.Request) {
	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" && xfp != "http" {
		r.URL.Scheme = xfp
	}

	wbuffer := newWriteBuffer(wOrig, int(h.Cluster.Collections.WebDAVOutputBuffer))
	defer wbuffer.Close()
	w := httpserver.WrapResponseWriter(responseWriter{
		Writer:         wbuffer,
		ResponseWriter: wOrig,
	})

	if r.Method == "OPTIONS" && ServeCORSPreflight(w, r.Header) {
		return
	}

	if !browserMethod[r.Method] && !webdavMethod[r.Method] {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Origin") != "" {
		// Allow simple cross-origin requests without user
		// credentials ("user credentials" as defined by CORS,
		// i.e., cookies, HTTP authentication, and client-side
		// SSL certificates. See
		// http://www.w3.org/TR/cors/#user-credentials).
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Range")
	}

	if h.serveS3(w, r) {
		return
	}

	webdavPrefix := ""
	arvPath := r.URL.Path
	if prefix := r.Header.Get("X-Webdav-Prefix"); prefix != "" {
		// Enable a proxy (e.g., container log handler in
		// controller) to satisfy a request for path
		// "/foo/bar/baz.txt" using content from
		// "//abc123-4.internal/bar/baz.txt", by adding a
		// request header "X-Webdav-Prefix: /foo"
		if !strings.HasPrefix(arvPath, prefix) {
			http.Error(w, "X-Webdav-Prefix header is not a prefix of the requested path", http.StatusBadRequest)
			return
		}
		arvPath = r.URL.Path[len(prefix):]
		if arvPath == "" {
			arvPath = "/"
		}
		w.Header().Set("Vary", "X-Webdav-Prefix, "+w.Header().Get("Vary"))
		webdavPrefix = prefix
	}
	pathParts := strings.Split(arvPath[1:], "/")

	var stripParts int
	var collectionID string
	var tokens []string
	var reqTokens []string
	var pathToken bool
	var attachment bool
	var useSiteFS bool
	credentialsOK := h.Cluster.Collections.TrustAllContent
	reasonNotAcceptingCredentials := ""

	if r.Host != "" && stripDefaultPort(r.Host) == stripDefaultPort(h.Cluster.Services.WebDAVDownload.ExternalURL.Host) {
		credentialsOK = true
		attachment = true
	} else if r.FormValue("disposition") == "attachment" {
		attachment = true
	}

	if !credentialsOK {
		reasonNotAcceptingCredentials = fmt.Sprintf("vhost %q does not specify a single collection ID or match Services.WebDAVDownload.ExternalURL %q, and Collections.TrustAllContent is false",
			r.Host, h.Cluster.Services.WebDAVDownload.ExternalURL)
	}

	if collectionID = arvados.CollectionIDFromDNSName(r.Host); collectionID != "" {
		// http://ID.collections.example/PATH...
		credentialsOK = true
	} else if r.URL.Path == "/status.json" {
		h.serveStatus(w, r)
		return
	} else if siteFSDir[pathParts[0]] {
		useSiteFS = true
	} else if len(pathParts) >= 1 && strings.HasPrefix(pathParts[0], "c=") {
		// /c=ID[/PATH...]
		collectionID = parseCollectionIDFromURL(pathParts[0][2:])
		stripParts = 1
	} else if len(pathParts) >= 2 && pathParts[0] == "collections" {
		if len(pathParts) >= 4 && pathParts[1] == "download" {
			// /collections/download/ID/TOKEN/PATH...
			collectionID = parseCollectionIDFromURL(pathParts[2])
			tokens = []string{pathParts[3]}
			stripParts = 4
			pathToken = true
		} else {
			// /collections/ID/PATH...
			collectionID = parseCollectionIDFromURL(pathParts[1])
			stripParts = 2
			// This path is only meant to work for public
			// data. Tokens provided with the request are
			// ignored.
			credentialsOK = false
			reasonNotAcceptingCredentials = "the '/collections/UUID/PATH' form only works for public data"
		}
	}

	forceReload := false
	if cc := r.Header.Get("Cache-Control"); strings.Contains(cc, "no-cache") || strings.Contains(cc, "must-revalidate") {
		forceReload = true
	}

	if credentialsOK {
		reqTokens = auth.CredentialsFromRequest(r).Tokens
	}

	r.ParseForm()
	origin := r.Header.Get("Origin")
	cors := origin != "" && !strings.HasSuffix(origin, "://"+r.Host)
	safeAjax := cors && (r.Method == http.MethodGet || r.Method == http.MethodHead)
	// Important distinction: safeAttachment checks whether api_token exists
	// as a query parameter. haveFormTokens checks whether api_token exists
	// as request form data *or* a query parameter. Different checks are
	// necessary because both the request disposition and the location of
	// the API token affect whether or not the request needs to be
	// redirected. The different branch comments below explain further.
	safeAttachment := attachment && !r.URL.Query().Has("api_token")
	if formTokens, haveFormTokens := r.Form["api_token"]; !haveFormTokens {
		// No token to use or redact.
	} else if safeAjax || safeAttachment {
		// If this is a cross-origin request, the URL won't
		// appear in the browser's address bar, so
		// substituting a clipboard-safe URL is pointless.
		// Redirect-with-cookie wouldn't work anyway, because
		// it's not safe to allow third-party use of our
		// cookie.
		//
		// If we're supplying an attachment, we don't need to
		// convert POST to GET to avoid the "really resubmit
		// form?" problem, so provided the token isn't
		// embedded in the URL, there's no reason to do
		// redirect-with-cookie in this case either.
		for _, tok := range formTokens {
			reqTokens = append(reqTokens, tok)
		}
	} else if browserMethod[r.Method] {
		// If this is a page view, and the client provided a
		// token via query string or POST body, we must put
		// the token in an HttpOnly cookie, and redirect to an
		// equivalent URL with the query param redacted and
		// method = GET.
		h.seeOtherWithCookie(w, r, "", credentialsOK)
		return
	}

	targetPath := pathParts[stripParts:]
	if tokens == nil && len(targetPath) > 0 && strings.HasPrefix(targetPath[0], "t=") {
		// http://ID.example/t=TOKEN/PATH...
		// /c=ID/t=TOKEN/PATH...
		//
		// This form must only be used to pass scoped tokens
		// that give permission for a single collection. See
		// FormValue case above.
		tokens = []string{targetPath[0][2:]}
		pathToken = true
		targetPath = targetPath[1:]
		stripParts++
	}

	fsprefix := ""
	if useSiteFS {
		if writeMethod[r.Method] {
			http.Error(w, webdavfs.ErrReadOnly.Error(), http.StatusMethodNotAllowed)
			return
		}
		if len(reqTokens) == 0 {
			w.Header().Add("WWW-Authenticate", "Basic realm=\"collections\"")
			http.Error(w, unauthorizedMessage, http.StatusUnauthorized)
			return
		}
		tokens = reqTokens
	} else if collectionID == "" {
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return
	} else {
		fsprefix = "by_id/" + collectionID + "/"
	}

	if src := r.Header.Get("X-Webdav-Source"); strings.HasPrefix(src, "/") && !strings.Contains(src, "//") && !strings.Contains(src, "/../") {
		fsprefix += src[1:]
	}

	if tokens == nil {
		tokens = reqTokens
		if h.Cluster.Users.AnonymousUserToken != "" {
			tokens = append(tokens, h.Cluster.Users.AnonymousUserToken)
		}
	}

	if len(targetPath) > 0 && targetPath[0] == "_" {
		// If a collection has a directory called "t=foo" or
		// "_", it can be served at
		// //collections.example/_/t=foo/ or
		// //collections.example/_/_/ respectively:
		// //collections.example/t=foo/ won't work because
		// t=foo will be interpreted as a token "foo".
		targetPath = targetPath[1:]
		stripParts++
	}

	dirOpenMode := os.O_RDONLY
	if writeMethod[r.Method] {
		dirOpenMode = os.O_RDWR
	}

	var tokenValid bool
	var tokenScopeProblem bool
	var token string
	var tokenUser *arvados.User
	var sessionFS arvados.CustomFileSystem
	var session *cachedSession
	var collectionDir arvados.File
	for _, token = range tokens {
		var statusErr errorWithHTTPStatus
		fs, sess, user, err := h.Cache.GetSession(token)
		if errors.As(err, &statusErr) && statusErr.HTTPStatus() == http.StatusUnauthorized {
			// bad token
			continue
		} else if err != nil {
			http.Error(w, "cache error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if token != h.Cluster.Users.AnonymousUserToken {
			tokenValid = true
		}
		f, err := fs.OpenFile(fsprefix, dirOpenMode, 0)
		if errors.As(err, &statusErr) &&
			statusErr.HTTPStatus() == http.StatusForbidden &&
			token != h.Cluster.Users.AnonymousUserToken {
			// collection id is outside scope of supplied
			// token
			tokenScopeProblem = true
			sess.Release()
			continue
		} else if os.IsNotExist(err) {
			// collection does not exist or is not
			// readable using this token
			sess.Release()
			continue
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			sess.Release()
			return
		}
		defer f.Close()

		collectionDir, sessionFS, session, tokenUser = f, fs, sess, user
		break
	}

	// releaseSession() is equivalent to session.Release() except
	// that it's a no-op if (1) session is nil, or (2) it has
	// already been called.
	//
	// This way, we can do a defer call here to ensure it gets
	// called in all code paths, and also call it inline (see
	// below) in the cases where we want to release the lock
	// before returning.
	releaseSession := func() {}
	if session != nil {
		var releaseSessionOnce sync.Once
		releaseSession = func() { releaseSessionOnce.Do(func() { session.Release() }) }
	}
	defer releaseSession()

	if forceReload && collectionDir != nil {
		err := collectionDir.Sync()
		if err != nil {
			if he := errorWithHTTPStatus(nil); errors.As(err, &he) {
				http.Error(w, err.Error(), he.HTTPStatus())
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	if session == nil {
		if pathToken {
			// The URL is a "secret sharing link" that
			// didn't work out.  Asking the client for
			// additional credentials would just be
			// confusing.
			http.Error(w, notFoundMessage, http.StatusNotFound)
			return
		}
		if tokenValid {
			// The client provided valid token(s), but the
			// collection was not found.
			http.Error(w, notFoundMessage, http.StatusNotFound)
			return
		}
		if tokenScopeProblem {
			// The client provided a valid token but
			// fetching a collection returned 401, which
			// means the token scope doesn't permit
			// fetching that collection.
			http.Error(w, notFoundMessage, http.StatusForbidden)
			return
		}
		// The client's token was invalid (e.g., expired), or
		// the client didn't even provide one.  Redirect to
		// workbench2's login-and-redirect-to-download url if
		// this is a browser navigation request. (The redirect
		// flow can't preserve the original method if it's not
		// GET, and doesn't make sense if the UA is a
		// command-line tool, is trying to load an inline
		// image, etc.; in these cases, there's nothing we can
		// do, so return 401 unauthorized.)
		//
		// Note Sec-Fetch-Mode is sent by all non-EOL
		// browsers, except Safari.
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Sec-Fetch-Mode
		//
		// TODO(TC): This response would be confusing to
		// someone trying (anonymously) to download public
		// data that has been deleted.  Allow a referrer to
		// provide this context somehow?
		if r.Method == http.MethodGet && r.Header.Get("Sec-Fetch-Mode") == "navigate" {
			target := url.URL(h.Cluster.Services.Workbench2.ExternalURL)
			redirkey := "redirectToPreview"
			if attachment {
				redirkey = "redirectToDownload"
			}
			callback := "/c=" + collectionID + "/" + strings.Join(targetPath, "/")
			query := url.Values{redirkey: {callback}}
			queryString := query.Encode()
			// Note: Encode (and QueryEscape function) turns space
			// into plus sign (+) rather than %20 (the plus sign
			// becomes %2B); that is the rule for web forms data
			// sent in URL query part via GET, but we're not
			// emulating forms here. Client JS APIs
			// (URLSearchParam#get, decodeURIComponent) will
			// decode %20, but while the former also expects the
			// form-specific encoding, the latter doesn't.
			// Encode() almost encodes everything; RFC3986 sec. 3.4
			// says "it is sometimes better for usability" to not
			// encode / and ? when passing URI reference in query.
			// This is also legal according to WHATWG URL spec and
			// can be desirable for debugging webapp.
			// We can let slash / appear in the encoded query, and
			// equality-sign = too, but exempting ? is not very
			// useful.
			// Plus-sign, hash, and ampersand are never exempt.
			r := strings.NewReplacer("+", "%20", "%2F", "/", "%3D", "=")
			target.RawQuery = r.Replace(queryString)
			w.Header().Add("Location", target.String())
			w.WriteHeader(http.StatusSeeOther)
			return
		}
		if !credentialsOK {
			http.Error(w, fmt.Sprintf("Authorization tokens are not accepted here: %v, and no anonymous user token is configured.", reasonNotAcceptingCredentials), http.StatusUnauthorized)
			return
		}
		// If none of the above cases apply, suggest the
		// user-agent (which is either a non-browser agent
		// like wget, or a browser that can't redirect through
		// a login flow) prompt the user for credentials.
		w.Header().Add("WWW-Authenticate", "Basic realm=\"collections\"")
		http.Error(w, unauthorizedMessage, http.StatusUnauthorized)
		return
	}

	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		targetfnm := fsprefix + strings.Join(pathParts[stripParts:], "/")
		if fi, err := sessionFS.Stat(targetfnm); err == nil && fi.IsDir() {
			releaseSession() // because we won't be writing anything
			if !strings.HasSuffix(r.URL.Path, "/") {
				h.seeOtherWithCookie(w, r, r.URL.Path+"/", credentialsOK)
			} else {
				h.serveDirectory(w, r, fi.Name(), sessionFS, targetfnm, !useSiteFS)
			}
			return
		}
	}

	var basename string
	if len(targetPath) > 0 {
		basename = targetPath[len(targetPath)-1]
	}
	if arvadosclient.PDHMatch(collectionID) && writeMethod[r.Method] {
		http.Error(w, webdavfs.ErrReadOnly.Error(), http.StatusMethodNotAllowed)
		return
	}
	if !h.userPermittedToUploadOrDownload(r.Method, tokenUser) {
		http.Error(w, "Not permitted", http.StatusForbidden)
		return
	}
	h.logUploadOrDownload(r, session.arvadosclient, sessionFS, fsprefix+strings.Join(targetPath, "/"), nil, tokenUser)

	writing := writeMethod[r.Method]
	locker := h.collectionLock(collectionID, writing)
	defer locker.Unlock()

	if writing {
		// Save the collection only if/when all
		// webdav->filesystem operations succeed --
		// and send a 500 error if the modified
		// collection can't be saved.
		//
		// Perform the write in a separate sitefs, so
		// concurrent read operations on the same
		// collection see the previous saved
		// state. After the write succeeds and the
		// collection record is updated, we reset the
		// session so the updates are visible in
		// subsequent read requests.
		client := session.client.WithRequestID(r.Header.Get("X-Request-Id"))
		sessionFS = client.SiteFileSystem(session.keepclient)
		writingDir, err := sessionFS.OpenFile(fsprefix, os.O_RDONLY, 0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer writingDir.Close()
		w = &updateOnSuccess{
			ResponseWriter: w,
			logger:         ctxlog.FromContext(r.Context()),
			update: func() error {
				err := writingDir.Sync()
				var te arvados.TransactionError
				if errors.As(err, &te) {
					err = te
				}
				if err != nil {
					return err
				}
				// Sync the changes to the persistent
				// sessionfs for this token.
				snap, err := writingDir.Snapshot()
				if err != nil {
					return err
				}
				collectionDir.Splice(snap)
				return nil
			}}
	} else {
		// When writing, we need to block session renewal
		// until we're finished, in order to guarantee the
		// effect of the write is visible in future responses.
		// But if we're not writing, we can release the lock
		// early.  This enables us to keep renewing sessions
		// and processing more requests even if a slow client
		// takes a long time to download a large file.
		releaseSession()
	}
	if r.Method == http.MethodGet {
		applyContentDispositionHdr(w, r, basename, attachment)
	}
	if webdavPrefix == "" {
		webdavPrefix = "/" + strings.Join(pathParts[:stripParts], "/")
	}
	wh := &webdav.Handler{
		Prefix: webdavPrefix,
		FileSystem: &webdavfs.FS{
			FileSystem:    sessionFS,
			Prefix:        fsprefix,
			Writing:       writeMethod[r.Method],
			AlwaysReadEOF: r.Method == "PROPFIND",
		},
		LockSystem: webdavfs.NoLockSystem,
		Logger: func(r *http.Request, err error) {
			if err != nil && !os.IsNotExist(err) {
				ctxlog.FromContext(r.Context()).WithError(err).Error("error reported by webdav handler")
			}
		},
	}
	h.metrics.track(wh, w, r)
	if r.Method == http.MethodGet && w.WroteStatus() == http.StatusOK {
		wrote := int64(w.WroteBodyBytes())
		fnm := strings.Join(pathParts[stripParts:], "/")
		fi, err := wh.FileSystem.Stat(r.Context(), fnm)
		if err == nil && fi.Size() != wrote {
			var n int
			f, err := wh.FileSystem.OpenFile(r.Context(), fnm, os.O_RDONLY, 0)
			if err == nil {
				n, err = f.Read(make([]byte, 1024))
				f.Close()
			}
			ctxlog.FromContext(r.Context()).Errorf("stat.Size()==%d but only wrote %d bytes; read(1024) returns %d, %v", fi.Size(), wrote, n, err)
		}
	}
}

var dirListingTemplate = `<!DOCTYPE HTML>
<HTML><HEAD>
  <META name="robots" content="NOINDEX">
  <TITLE>{{ .CollectionName }}</TITLE>
  <STYLE type="text/css">
    body {
      margin: 1.5em;
    }
    pre {
      background-color: #D9EDF7;
      border-radius: .25em;
      padding: .75em;
      overflow: auto;
    }
    .footer p {
      font-size: 82%;
    }
    hr {
      border: 1px solid #808080;
    }
    ul {
      padding: 0;
    }
    ul li {
      font-family: monospace;
      list-style: none;
    }
  </STYLE>
</HEAD>
<BODY>

<H1>{{ .CollectionName }}</H1>

<P>This collection of data files is being shared with you through
Arvados.  You can download individual files listed below.  To download
the entire directory tree with <CODE>wget</CODE>, try:</P>

<PRE id="wget-example">$ wget --mirror --no-parent --no-host --cut-dirs={{ .StripParts }} {{ .QuotedUrlForWget }}</PRE>

<H2>File Listing</H2>

{{if .Files}}
<UL>
{{range .Files}}
{{if .IsDir }}
  <LI>{{" " | printf "%15s  " | nbsp}}<A class="item" href="{{ .Href }}/">{{ .Name }}/</A></LI>
{{else}}
  <LI>{{.Size | printf "%15d  " | nbsp}}<A class="item" href="{{ .Href }}">{{ .Name }}</A></LI>
{{end}}
{{end}}
</UL>
{{else}}
<P>(No files; this collection is empty.)</P>
{{end}}

<HR>
<DIV class="footer">
  <P>
    About Arvados:
    Arvados is a free and open source software bioinformatics platform.
    To learn more, visit arvados.org.
    Arvados is not responsible for the files listed on this page.
  </P>
</DIV>

</BODY>
</HTML>
`

type fileListEnt struct {
	Name  string
	Href  string
	Size  int64
	IsDir bool
}

// Given a filesystem path like `foo/"bar baz"`, return an escaped
// (percent-encoded) relative path like `./foo/%22bar%20%baz%22`.
//
// Note the result may contain html-unsafe characters like '&'. These
// will be handled separately by the HTML templating engine as needed.
func relativeHref(path string) string {
	u := &url.URL{Path: path}
	return "./" + u.EscapedPath()
}

// Return a shell-quoted URL suitable for pasting to a command line
// ("wget ...") to repeat the given HTTP request.
func makeQuotedUrlForWget(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "http" || scheme == "https" {
		// use protocol reported by load balancer / proxy
	} else if r.TLS != nil {
		scheme = "https"
	} else {
		scheme = "http"
	}
	p := r.URL.EscapedPath()
	// An escaped path may still contain single quote chars, which
	// would interfere with our shell quoting. Avoid this by
	// escaping them as %27.
	return fmt.Sprintf("'%s://%s%s'", scheme, r.Host, strings.Replace(p, "'", "%27", -1))
}

func (h *handler) serveDirectory(w http.ResponseWriter, r *http.Request, collectionName string, fs http.FileSystem, base string, recurse bool) {
	var files []fileListEnt
	var walk func(string) error
	if !strings.HasSuffix(base, "/") {
		base = base + "/"
	}
	walk = func(path string) error {
		dirname := base + path
		if dirname != "/" {
			dirname = strings.TrimSuffix(dirname, "/")
		}
		d, err := fs.Open(dirname)
		if err != nil {
			return err
		}
		ents, err := d.Readdir(-1)
		if err != nil {
			return err
		}
		for _, ent := range ents {
			if recurse && ent.IsDir() {
				err = walk(path + ent.Name() + "/")
				if err != nil {
					return err
				}
			} else {
				listingName := path + ent.Name()
				files = append(files, fileListEnt{
					Name:  listingName,
					Href:  relativeHref(listingName),
					Size:  ent.Size(),
					IsDir: ent.IsDir(),
				})
			}
		}
		return nil
	}
	if err := walk(""); err != nil {
		http.Error(w, "error getting directory listing: "+err.Error(), http.StatusInternalServerError)
		return
	}

	funcs := template.FuncMap{
		"nbsp": func(s string) template.HTML {
			return template.HTML(strings.Replace(s, " ", "&nbsp;", -1))
		},
	}
	tmpl, err := template.New("dir").Funcs(funcs).Parse(dirListingTemplate)
	if err != nil {
		http.Error(w, "error parsing template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})
	w.WriteHeader(http.StatusOK)
	tmpl.Execute(w, map[string]interface{}{
		"CollectionName":   collectionName,
		"Files":            files,
		"Request":          r,
		"StripParts":       strings.Count(strings.TrimRight(r.URL.Path, "/"), "/"),
		"QuotedUrlForWget": makeQuotedUrlForWget(r),
	})
}

func applyContentDispositionHdr(w http.ResponseWriter, r *http.Request, filename string, isAttachment bool) {
	disposition := "inline"
	if isAttachment {
		disposition = "attachment"
	}
	if strings.ContainsRune(r.RequestURI, '?') {
		// Help the UA realize that the filename is just
		// "filename.txt", not
		// "filename.txt?disposition=attachment".
		//
		// TODO(TC): Follow advice at RFC 6266 appendix D
		disposition += "; filename=" + strconv.QuoteToASCII(filename)
	}
	if disposition != "inline" {
		w.Header().Set("Content-Disposition", disposition)
	}
}

func (h *handler) seeOtherWithCookie(w http.ResponseWriter, r *http.Request, location string, credentialsOK bool) {
	if formTokens, haveFormTokens := r.Form["api_token"]; haveFormTokens {
		if !credentialsOK {
			// It is not safe to copy the provided token
			// into a cookie unless the current vhost
			// (origin) serves only a single collection or
			// we are in TrustAllContent mode.
			http.Error(w, "cannot serve inline content at this URL (possible configuration error; see https://doc.arvados.org/install/install-keep-web.html#dns)", http.StatusBadRequest)
			return
		}

		// The HttpOnly flag is necessary to prevent
		// JavaScript code (included in, or loaded by, a page
		// in the collection being served) from employing the
		// user's token beyond reading other files in the same
		// domain, i.e., same collection.
		//
		// The 303 redirect is necessary in the case of a GET
		// request to avoid exposing the token in the Location
		// bar, and in the case of a POST request to avoid
		// raising warnings when the user refreshes the
		// resulting page.
		for _, tok := range formTokens {
			if tok == "" {
				continue
			}
			http.SetCookie(w, &http.Cookie{
				Name:     "arvados_api_token",
				Value:    auth.EncodeTokenCookie([]byte(tok)),
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
			break
		}
	}

	// Propagate query parameters (except api_token) from
	// the original request.
	redirQuery := r.URL.Query()
	redirQuery.Del("api_token")

	u := r.URL
	if location != "" {
		newu, err := u.Parse(location)
		if err != nil {
			http.Error(w, "error resolving redirect target: "+err.Error(), http.StatusInternalServerError)
			return
		}
		u = newu
	}
	redir := (&url.URL{
		Scheme:   r.URL.Scheme,
		Host:     r.Host,
		Path:     u.Path,
		RawQuery: redirQuery.Encode(),
	}).String()

	w.Header().Add("Location", redir)
	w.WriteHeader(http.StatusSeeOther)
	io.WriteString(w, `<A href="`)
	io.WriteString(w, html.EscapeString(redir))
	io.WriteString(w, `">Continue</A>`)
}

func (h *handler) userPermittedToUploadOrDownload(method string, tokenUser *arvados.User) bool {
	var permitDownload bool
	var permitUpload bool
	if tokenUser != nil && tokenUser.IsAdmin {
		permitUpload = h.Cluster.Collections.WebDAVPermission.Admin.Upload
		permitDownload = h.Cluster.Collections.WebDAVPermission.Admin.Download
	} else {
		permitUpload = h.Cluster.Collections.WebDAVPermission.User.Upload
		permitDownload = h.Cluster.Collections.WebDAVPermission.User.Download
	}
	if (method == "PUT" || method == "POST") && !permitUpload {
		// Disallow operations that upload new files.
		// Permit webdav operations that move existing files around.
		return false
	} else if method == "GET" && !permitDownload {
		// Disallow downloading file contents.
		// Permit webdav operations like PROPFIND that retrieve metadata
		// but not file contents.
		return false
	}
	return true
}

type fileEventLog struct {
	requestPath  string
	eventType    string
	userUUID     string
	userFullName string
	collUUID     string
	collPDH      string
	collFilePath string
	clientAddr   string
	clientToken  string
}

func newFileEventLog(
	h *handler,
	r *http.Request,
	filepath string,
	collection *arvados.Collection,
	user *arvados.User,
	token string,
) *fileEventLog {
	var eventType string
	switch r.Method {
	case "POST", "PUT":
		eventType = "file_upload"
	case "GET":
		eventType = "file_download"
	default:
		return nil
	}

	// We want to log the address of the proxy closest to keep-web—the last
	// value in the X-Forwarded-For list—or the client address if there is no
	// valid proxy.
	var clientAddr string
	// 1. Build a slice of proxy addresses from X-Forwarded-For.
	xff := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
	addrs := strings.Split(xff, ",")
	// 2. Reverse the slice so it's in our most preferred order for logging.
	slices.Reverse(addrs)
	// 3. Append the client address to that slice.
	if addr, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		addrs = append(addrs, addr)
	}
	// 4. Use the first valid address in the slice.
	for _, addr := range addrs {
		if ip := net.ParseIP(strings.TrimSpace(addr)); ip != nil {
			clientAddr = ip.String()
			break
		}
	}

	ev := &fileEventLog{
		requestPath: r.URL.Path,
		eventType:   eventType,
		clientAddr:  clientAddr,
		clientToken: token,
	}

	if user != nil {
		ev.userUUID = user.UUID
		ev.userFullName = user.FullName
	} else {
		ev.userUUID = fmt.Sprintf("%s-tpzed-anonymouspublic", h.Cluster.ClusterID)
	}

	if collection != nil {
		ev.collFilePath = filepath
		// h.determineCollection populates the collection_uuid
		// prop with the PDH, if this collection is being
		// accessed via PDH. For logging, we use a different
		// field depending on whether it's a UUID or PDH.
		if len(collection.UUID) > 32 {
			ev.collPDH = collection.UUID
		} else {
			ev.collPDH = collection.PortableDataHash
			ev.collUUID = collection.UUID
		}
	}

	return ev
}

func (ev *fileEventLog) shouldLogPDH() bool {
	return ev.eventType == "file_download" && ev.collPDH != ""
}

func (ev *fileEventLog) asDict() arvadosclient.Dict {
	props := arvadosclient.Dict{
		"reqPath":              ev.requestPath,
		"collection_uuid":      ev.collUUID,
		"collection_file_path": ev.collFilePath,
	}
	if ev.shouldLogPDH() {
		props["portable_data_hash"] = ev.collPDH
	}
	return arvadosclient.Dict{
		"object_uuid": ev.userUUID,
		"event_type":  ev.eventType,
		"properties":  props,
	}
}

func (ev *fileEventLog) asFields() logrus.Fields {
	fields := logrus.Fields{
		"collection_file_path": ev.collFilePath,
		"collection_uuid":      ev.collUUID,
		"user_uuid":            ev.userUUID,
	}
	if ev.shouldLogPDH() {
		fields["portable_data_hash"] = ev.collPDH
	}
	if !strings.HasSuffix(ev.userUUID, "-tpzed-anonymouspublic") {
		fields["user_full_name"] = ev.userFullName
	}
	return fields
}

func (h *handler) shouldLogEvent(event *fileEventLog, t time.Time) bool {
	if event == nil {
		return false
	} else if event.eventType != "file_download" ||
		h.Cluster.Collections.WebDAVLogDownloadInterval == 0 {
		return true
	}
	td := h.Cluster.Collections.WebDAVLogDownloadInterval.Duration()
	cutoff := t.Add(-td)
	ev := *event
	h.fileEventLogsMtx.Lock()
	defer h.fileEventLogsMtx.Unlock()
	if h.fileEventLogs == nil {
		h.fileEventLogs = make(map[fileEventLog]time.Time)
	}
	shouldLog := h.fileEventLogs[ev].Before(cutoff)
	if shouldLog {
		h.fileEventLogs[ev] = t
	}
	if t.After(h.fileEventLogsNextTidy) {
		for key, logTime := range h.fileEventLogs {
			if logTime.Before(cutoff) {
				delete(h.fileEventLogs, key)
			}
		}
		h.fileEventLogsNextTidy = t.Add(td)
	}
	return shouldLog
}

func (h *handler) logUploadOrDownload(
	r *http.Request,
	client *arvadosclient.ArvadosClient,
	fs arvados.CustomFileSystem,
	filepath string,
	collection *arvados.Collection,
	user *arvados.User,
) {
	if collection == nil && fs != nil {
		collection, filepath = h.determineCollection(fs, filepath)
	}
	event := newFileEventLog(h, r, filepath, collection, user, client.ApiToken)
	if !h.shouldLogEvent(event, time.Now()) {
		return
	}
	log := ctxlog.FromContext(r.Context()).WithFields(event.asFields())
	log.Info(strings.Replace(event.eventType, "file_", "File ", 1))
	if h.Cluster.Collections.WebDAVLogEvents {
		go func() {
			logReq := arvadosclient.Dict{"log": event.asDict()}
			err := client.Create("logs", logReq, nil)
			if err != nil {
				log.WithError(err).Errorf("Failed to create %s log event on API server", event.eventType)
			}
		}()
	}
}

func (h *handler) determineCollection(fs arvados.CustomFileSystem, path string) (*arvados.Collection, string) {
	target := strings.TrimSuffix(path, "/")
	for cut := len(target); cut >= 0; cut = strings.LastIndexByte(target, '/') {
		target = target[:cut]
		fi, err := fs.Stat(target)
		if os.IsNotExist(err) {
			// creating a new file/dir, or download
			// destined to fail
			continue
		} else if err != nil {
			return nil, ""
		}
		switch src := fi.Sys().(type) {
		case *arvados.Collection:
			return src, strings.TrimPrefix(path[len(target):], "/")
		case *arvados.Group:
			return nil, ""
		default:
			if _, ok := src.(error); ok {
				return nil, ""
			}
		}
	}
	return nil, ""
}

var lockTidyInterval = time.Minute * 10

// Lock the specified collection for reading or writing. Caller must
// call Unlock() on the returned Locker when the operation is
// finished.
func (h *handler) collectionLock(collectionID string, writing bool) sync.Locker {
	h.lockMtx.Lock()
	defer h.lockMtx.Unlock()
	if time.Since(h.lockTidied) > lockTidyInterval {
		// Periodically delete all locks that aren't in use.
		h.lockTidied = time.Now()
		for id, locker := range h.lock {
			if locker.TryLock() {
				locker.Unlock()
				delete(h.lock, id)
			}
		}
	}
	locker := h.lock[collectionID]
	if locker == nil {
		locker = new(sync.RWMutex)
		if h.lock == nil {
			h.lock = map[string]*sync.RWMutex{}
		}
		h.lock[collectionID] = locker
	}
	if writing {
		locker.Lock()
		return locker
	} else {
		locker.RLock()
		return locker.RLocker()
	}
}

func ServeCORSPreflight(w http.ResponseWriter, header http.Header) bool {
	method := header.Get("Access-Control-Request-Method")
	if method == "" {
		return false
	}
	if !browserMethod[method] && !webdavMethod[method] {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return true
	}
	w.Header().Set("Access-Control-Allow-Headers", corsAllowHeadersHeader)
	w.Header().Set("Access-Control-Allow-Methods", "COPY, DELETE, GET, LOCK, MKCOL, MOVE, OPTIONS, POST, PROPFIND, PROPPATCH, PUT, RMCOL, UNLOCK")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Max-Age", "86400")
	return true
}
