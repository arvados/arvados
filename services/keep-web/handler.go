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
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
)

type handler struct {
	Cache     cache
	Cluster   *arvados.Cluster
	setupOnce sync.Once
	webdavLS  webdav.LockSystem
}

var urlPDHDecoder = strings.NewReplacer(" ", "+", "-", "+")

var notFoundMessage = "Not Found"
var unauthorizedMessage = "401 Unauthorized\r\n\r\nA valid Arvados token must be provided to access this resource.\r\n"

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

func (h *handler) setup() {
	keepclient.DefaultBlockCache.MaxBlocks = h.Cluster.Collections.WebDAVCache.MaxBlockEntries

	// Even though we don't accept LOCK requests, every webdav
	// handler must have a non-nil LockSystem.
	h.webdavLS = &noLockSystem{}
}

func (h *handler) serveStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(struct{ Version string }{cmd.Version.String()})
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
				var he interface{ HTTPStatus() int }
				if errors.As(uos.err, &he) {
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
		"Depth", "Destination", "If", "Lock-Token", "Overwrite", "Timeout",
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
	h.setupOnce.Do(h.setup)

	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" && xfp != "http" {
		r.URL.Scheme = xfp
	}

	w := httpserver.WrapResponseWriter(wOrig)

	if method := r.Header.Get("Access-Control-Request-Method"); method != "" && r.Method == "OPTIONS" {
		if !browserMethod[method] && !webdavMethod[method] {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Access-Control-Allow-Headers", corsAllowHeadersHeader)
		w.Header().Set("Access-Control-Allow-Methods", "COPY, DELETE, GET, LOCK, MKCOL, MOVE, OPTIONS, POST, PROPFIND, PROPPATCH, PUT, RMCOL, UNLOCK")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Max-Age", "86400")
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

	pathParts := strings.Split(r.URL.Path[1:], "/")

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

	formToken := r.FormValue("api_token")
	origin := r.Header.Get("Origin")
	cors := origin != "" && !strings.HasSuffix(origin, "://"+r.Host)
	safeAjax := cors && (r.Method == http.MethodGet || r.Method == http.MethodHead)
	safeAttachment := attachment && r.URL.Query().Get("api_token") == ""
	if formToken == "" {
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
		reqTokens = append(reqTokens, formToken)
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
			http.Error(w, errReadOnly.Error(), http.StatusMethodNotAllowed)
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

	if tokens == nil {
		tokens = reqTokens
		if h.Cluster.Users.AnonymousUserToken != "" {
			tokens = append(tokens, h.Cluster.Users.AnonymousUserToken)
		}
	}

	if tokens == nil {
		if !credentialsOK {
			http.Error(w, fmt.Sprintf("Authorization tokens are not accepted here: %v, and no anonymous user token is configured.", reasonNotAcceptingCredentials), http.StatusUnauthorized)
		} else {
			http.Error(w, fmt.Sprintf("No authorization token in request, and no anonymous user token is configured."), http.StatusUnauthorized)
		}
		return
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

	validToken := make(map[string]bool)
	var token string
	var tokenUser *arvados.User
	var sessionFS arvados.CustomFileSystem
	var session *cachedSession
	var collectionDir arvados.File
	for _, token = range tokens {
		var statusErr interface{ HTTPStatus() int }
		fs, sess, user, err := h.Cache.GetSession(token)
		if errors.As(err, &statusErr) && statusErr.HTTPStatus() == http.StatusUnauthorized {
			// bad token
			continue
		} else if err != nil {
			http.Error(w, "cache error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		f, err := fs.OpenFile(fsprefix, dirOpenMode, 0)
		if errors.As(err, &statusErr) && statusErr.HTTPStatus() == http.StatusForbidden {
			// collection id is outside token scope
			validToken[token] = true
			continue
		}
		validToken[token] = true
		if os.IsNotExist(err) {
			// collection does not exist or is not
			// readable using this token
			continue
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		collectionDir, sessionFS, session, tokenUser = f, fs, sess, user
		break
	}
	if forceReload && collectionDir != nil {
		err := collectionDir.Sync()
		if err != nil {
			var statusErr interface{ HTTPStatus() int }
			if errors.As(err, &statusErr) {
				http.Error(w, err.Error(), statusErr.HTTPStatus())
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	if session == nil {
		if pathToken || !credentialsOK {
			// Either the URL is a "secret sharing link"
			// that didn't work out (and asking the client
			// for additional credentials would just be
			// confusing), or we don't even accept
			// credentials at this path.
			http.Error(w, notFoundMessage, http.StatusNotFound)
			return
		}
		for _, t := range reqTokens {
			if validToken[t] {
				// The client provided valid token(s),
				// but the collection was not found.
				http.Error(w, notFoundMessage, http.StatusNotFound)
				return
			}
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
			// target.RawQuery = url.Values{redirkey:
			// {target}}.Encode() would be the obvious
			// thing to do here, but wb2 doesn't decode
			// this as a query param -- it takes
			// everything after "${redirkey}=" as the
			// target URL. If we encode "/" as "%2F" etc.,
			// the redirect won't work.
			target.RawQuery = redirkey + "=" + callback
			w.Header().Add("Location", target.String())
			w.WriteHeader(http.StatusSeeOther)
		} else {
			w.Header().Add("WWW-Authenticate", "Basic realm=\"collections\"")
			http.Error(w, unauthorizedMessage, http.StatusUnauthorized)
		}
		return
	}

	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		targetfnm := fsprefix + strings.Join(pathParts[stripParts:], "/")
		if fi, err := sessionFS.Stat(targetfnm); err == nil && fi.IsDir() {
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
		http.Error(w, errReadOnly.Error(), http.StatusMethodNotAllowed)
		return
	}
	if !h.userPermittedToUploadOrDownload(r.Method, tokenUser) {
		http.Error(w, "Not permitted", http.StatusForbidden)
		return
	}
	h.logUploadOrDownload(r, session.arvadosclient, sessionFS, fsprefix+strings.Join(targetPath, "/"), nil, tokenUser)

	if writeMethod[r.Method] {
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
	}
	if r.Method == http.MethodGet {
		applyContentDispositionHdr(w, r, basename, attachment)
	}
	wh := webdav.Handler{
		Prefix: "/" + strings.Join(pathParts[:stripParts], "/"),
		FileSystem: &webdavFS{
			collfs:        sessionFS,
			prefix:        fsprefix,
			writing:       writeMethod[r.Method],
			alwaysReadEOF: r.Method == "PROPFIND",
		},
		LockSystem: h.webdavLS,
		Logger: func(r *http.Request, err error) {
			if err != nil {
				ctxlog.FromContext(r.Context()).WithError(err).Error("error reported by webdav handler")
			}
		},
	}
	wh.ServeHTTP(w, r)
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
the entire directory tree with wget, try:</P>

<PRE>$ wget --mirror --no-parent --no-host --cut-dirs={{ .StripParts }} https://{{ .Request.Host }}{{ .Request.URL.Path }}</PRE>

<H2>File Listing</H2>

{{if .Files}}
<UL>
{{range .Files}}
{{if .IsDir }}
  <LI>{{" " | printf "%15s  " | nbsp}}<A href="{{print "./" .Name}}/">{{.Name}}/</A></LI>
{{else}}
  <LI>{{.Size | printf "%15d  " | nbsp}}<A href="{{print "./" .Name}}">{{.Name}}</A></LI>
{{end}}
{{end}}
</UL>
{{else}}
<P>(No files; this collection is empty.)</P>
{{end}}

<HR noshade>
<DIV class="footer">
  <P>
    About Arvados:
    Arvados is a free and open source software bioinformatics platform.
    To learn more, visit arvados.org.
    Arvados is not responsible for the files listed on this page.
  </P>
</DIV>

</BODY>
`

type fileListEnt struct {
	Name  string
	Size  int64
	IsDir bool
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
				files = append(files, fileListEnt{
					Name:  path + ent.Name(),
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
		"CollectionName": collectionName,
		"Files":          files,
		"Request":        r,
		"StripParts":     strings.Count(strings.TrimRight(r.URL.Path, "/"), "/"),
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
	if formToken := r.FormValue("api_token"); formToken != "" {
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
		http.SetCookie(w, &http.Cookie{
			Name:     "arvados_api_token",
			Value:    auth.EncodeTokenCookie([]byte(formToken)),
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
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

func (h *handler) logUploadOrDownload(
	r *http.Request,
	client *arvadosclient.ArvadosClient,
	fs arvados.CustomFileSystem,
	filepath string,
	collection *arvados.Collection,
	user *arvados.User) {

	log := ctxlog.FromContext(r.Context())
	props := make(map[string]string)
	props["reqPath"] = r.URL.Path
	var useruuid string
	if user != nil {
		log = log.WithField("user_uuid", user.UUID).
			WithField("user_full_name", user.FullName)
		useruuid = user.UUID
	} else {
		useruuid = fmt.Sprintf("%s-tpzed-anonymouspublic", h.Cluster.ClusterID)
	}
	if collection == nil && fs != nil {
		collection, filepath = h.determineCollection(fs, filepath)
	}
	if collection != nil {
		log = log.WithField("collection_file_path", filepath)
		props["collection_file_path"] = filepath
		// h.determineCollection populates the collection_uuid
		// prop with the PDH, if this collection is being
		// accessed via PDH. For logging, we use a different
		// field depending on whether it's a UUID or PDH.
		if len(collection.UUID) > 32 {
			log = log.WithField("portable_data_hash", collection.UUID)
			props["portable_data_hash"] = collection.UUID
		} else {
			log = log.WithField("collection_uuid", collection.UUID)
			props["collection_uuid"] = collection.UUID
		}
	}
	if r.Method == "PUT" || r.Method == "POST" {
		log.Info("File upload")
		if h.Cluster.Collections.WebDAVLogEvents {
			go func() {
				lr := arvadosclient.Dict{"log": arvadosclient.Dict{
					"object_uuid": useruuid,
					"event_type":  "file_upload",
					"properties":  props}}
				err := client.Create("logs", lr, nil)
				if err != nil {
					log.WithError(err).Error("Failed to create upload log event on API server")
				}
			}()
		}
	} else if r.Method == "GET" {
		if collection != nil && collection.PortableDataHash != "" {
			log = log.WithField("portable_data_hash", collection.PortableDataHash)
			props["portable_data_hash"] = collection.PortableDataHash
		}
		log.Info("File download")
		if h.Cluster.Collections.WebDAVLogEvents {
			go func() {
				lr := arvadosclient.Dict{"log": arvadosclient.Dict{
					"object_uuid": useruuid,
					"event_type":  "file_download",
					"properties":  props}}
				err := client.Create("logs", lr, nil)
				if err != nil {
					log.WithError(err).Error("Failed to create download log event on API server")
				}
			}()
		}
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
