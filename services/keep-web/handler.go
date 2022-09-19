// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/health"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/webdav"
)

type handler struct {
	Config        *Config
	MetricsAPI    http.Handler
	clientPool    *arvadosclient.ClientPool
	setupOnce     sync.Once
	healthHandler http.Handler
	webdavLS      webdav.LockSystem
}

// parseCollectionIDFromDNSName returns a UUID or PDH if s begins with
// a UUID or URL-encoded PDH; otherwise "".
func parseCollectionIDFromDNSName(s string) string {
	// Strip domain.
	if i := strings.IndexRune(s, '.'); i >= 0 {
		s = s[:i]
	}
	// Names like {uuid}--collections.example.com serve the same
	// purpose as {uuid}.collections.example.com but can reduce
	// cost/effort of using [additional] wildcard certificates.
	if i := strings.Index(s, "--"); i >= 0 {
		s = s[:i]
	}
	if arvadosclient.UUIDMatch(s) {
		return s
	}
	if pdh := strings.Replace(s, "-", "+", 1); arvadosclient.PDHMatch(pdh) {
		return pdh
	}
	return ""
}

var urlPDHDecoder = strings.NewReplacer(" ", "+", "-", "+")

var notFoundMessage = "404 Not found\r\n\r\nThe requested path was not found, or you do not have permission to access it.\r"
var unauthorizedMessage = "401 Unauthorized\r\n\r\nA valid Arvados token must be provided to access this resource.\r"

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
	// Errors will be handled at the client pool.
	arv, _ := arvados.NewClientFromConfig(h.Config.cluster)
	h.clientPool = arvadosclient.MakeClientPoolWith(arv)

	keepclient.RefreshServiceDiscoveryOnSIGHUP()
	keepclient.DefaultBlockCache.MaxBlocks = h.Config.cluster.Collections.WebDAVCache.MaxBlockEntries

	h.healthHandler = &health.Handler{
		Token:  h.Config.cluster.ManagementToken,
		Prefix: "/_health/",
	}

	// Even though we don't accept LOCK requests, every webdav
	// handler must have a non-nil LockSystem.
	h.webdavLS = &noLockSystem{}
}

func (h *handler) serveStatus(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(struct{ Version string }{version})
}

// updateOnSuccess wraps httpserver.ResponseWriter. If the handler
// sends an HTTP header indicating success, updateOnSuccess first
// calls the provided update func. If the update func fails, a 500
// response is sent, and the status code and body sent by the handler
// are ignored (all response writes return the update error).
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
				if err, ok := uos.err.(*arvados.TransactionError); ok {
					code = err.StatusCode
				}
				uos.logger.WithError(uos.err).Errorf("update() returned error type %T, changing response to HTTP %d", uos.err, code)
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

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(wOrig http.ResponseWriter, r *http.Request) {
	h.setupOnce.Do(h.setup)

	if xfp := r.Header.Get("X-Forwarded-Proto"); xfp != "" && xfp != "http" {
		r.URL.Scheme = xfp
	}

	w := httpserver.WrapResponseWriter(wOrig)

	if strings.HasPrefix(r.URL.Path, "/_health/") && r.Method == "GET" {
		h.healthHandler.ServeHTTP(w, r)
		return
	}

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
	credentialsOK := h.Config.cluster.Collections.TrustAllContent
	reasonNotAcceptingCredentials := ""

	if r.Host != "" && stripDefaultPort(r.Host) == stripDefaultPort(h.Config.cluster.Services.WebDAVDownload.ExternalURL.Host) {
		credentialsOK = true
		attachment = true
	} else if r.FormValue("disposition") == "attachment" {
		attachment = true
	}

	if !credentialsOK {
		reasonNotAcceptingCredentials = fmt.Sprintf("vhost %q does not specify a single collection ID or match Services.WebDAVDownload.ExternalURL %q, and Collections.TrustAllContent is false",
			r.Host, h.Config.cluster.Services.WebDAVDownload.ExternalURL)
	}

	if collectionID = parseCollectionIDFromDNSName(r.Host); collectionID != "" {
		// http://ID.collections.example/PATH...
		credentialsOK = true
	} else if r.URL.Path == "/status.json" {
		h.serveStatus(w, r)
		return
	} else if strings.HasPrefix(r.URL.Path, "/metrics") {
		h.MetricsAPI.ServeHTTP(w, r)
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

	if collectionID == "" && !useSiteFS {
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return
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

	if useSiteFS {
		h.serveSiteFS(w, r, reqTokens, credentialsOK, attachment)
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

	if tokens == nil {
		tokens = reqTokens
		if h.Config.cluster.Users.AnonymousUserToken != "" {
			tokens = append(tokens, h.Config.cluster.Users.AnonymousUserToken)
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

	arv := h.clientPool.Get()
	if arv == nil {
		http.Error(w, "client pool error: "+h.clientPool.Err().Error(), http.StatusInternalServerError)
		return
	}
	defer h.clientPool.Put(arv)

	var collection *arvados.Collection
	var tokenUser *arvados.User
	tokenResult := make(map[string]int)
	for _, arv.ApiToken = range tokens {
		var err error
		collection, err = h.Config.Cache.Get(arv, collectionID, forceReload)
		if err == nil {
			// Success
			break
		}
		if srvErr, ok := err.(arvadosclient.APIServerError); ok {
			switch srvErr.HttpStatusCode {
			case 404, 401:
				// Token broken or insufficient to
				// retrieve collection
				tokenResult[arv.ApiToken] = srvErr.HttpStatusCode
				continue
			}
		}
		// Something more serious is wrong
		http.Error(w, "cache error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if collection == nil {
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
			if tokenResult[t] == 404 {
				// The client provided valid token(s), but the
				// collection was not found.
				http.Error(w, notFoundMessage, http.StatusNotFound)
				return
			}
		}
		// The client's token was invalid (e.g., expired), or
		// the client didn't even provide one.  Propagate the
		// 401 to encourage the client to use a [different]
		// token.
		//
		// TODO(TC): This response would be confusing to
		// someone trying (anonymously) to download public
		// data that has been deleted.  Allow a referrer to
		// provide this context somehow?
		w.Header().Add("WWW-Authenticate", "Basic realm=\"collections\"")
		http.Error(w, unauthorizedMessage, http.StatusUnauthorized)
		return
	}

	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		http.Error(w, "error setting up keep client: "+err.Error(), http.StatusInternalServerError)
		return
	}
	kc.RequestID = r.Header.Get("X-Request-Id")

	var basename string
	if len(targetPath) > 0 {
		basename = targetPath[len(targetPath)-1]
	}
	applyContentDispositionHdr(w, r, basename, attachment)

	client := (&arvados.Client{
		APIHost:   arv.ApiServer,
		AuthToken: arv.ApiToken,
		Insecure:  arv.ApiInsecure,
	}).WithRequestID(r.Header.Get("X-Request-Id"))

	fs, err := collection.FileSystem(client, kc)
	if err != nil {
		http.Error(w, "error creating collection filesystem: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writefs, writeOK := fs.(arvados.CollectionFileSystem)
	targetIsPDH := arvadosclient.PDHMatch(collectionID)
	if (targetIsPDH || !writeOK) && writeMethod[r.Method] {
		http.Error(w, errReadOnly.Error(), http.StatusMethodNotAllowed)
		return
	}

	// Check configured permission
	_, sess, err := h.Config.Cache.GetSession(arv.ApiToken)
	if err != nil {
		http.Error(w, "session cache: "+err.Error(), http.StatusInternalServerError)
	}
	tokenUser, err = h.Config.Cache.GetTokenUser(arv.ApiToken)
	if err != nil {
		http.Error(w, "user lookup: "+err.Error(), http.StatusInternalServerError)
	}

	if webdavMethod[r.Method] {
		if !h.userPermittedToUploadOrDownload(r.Method, tokenUser) {
			http.Error(w, "Not permitted", http.StatusForbidden)
			return
		}
		h.logUploadOrDownload(r, sess.arvadosclient, nil, strings.Join(targetPath, "/"), collection, tokenUser)

		if writeMethod[r.Method] {
			// Save the collection only if/when all
			// webdav->filesystem operations succeed --
			// and send a 500 error if the modified
			// collection can't be saved.
			w = &updateOnSuccess{
				ResponseWriter: w,
				logger:         ctxlog.FromContext(r.Context()),
				update: func() error {
					return h.Config.Cache.Update(client, *collection, writefs)
				}}
		}
		h := webdav.Handler{
			Prefix: "/" + strings.Join(pathParts[:stripParts], "/"),
			FileSystem: &webdavFS{
				collfs:        fs,
				writing:       writeMethod[r.Method],
				alwaysReadEOF: r.Method == "PROPFIND",
			},
			LockSystem: h.webdavLS,
			Logger: func(_ *http.Request, err error) {
				if err != nil {
					ctxlog.FromContext(r.Context()).WithError(err).Error("error reported by webdav handler")
				}
			},
		}
		h.ServeHTTP(w, r)
		return
	}

	openPath := "/" + strings.Join(targetPath, "/")
	f, err := fs.Open(openPath)
	if os.IsNotExist(err) {
		// Requested non-existent path
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return
	} else if err != nil {
		// Some other (unexpected) error
		http.Error(w, "open: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	if stat, err := f.Stat(); err != nil {
		// Can't get Size/IsDir (shouldn't happen with a collectionFS!)
		http.Error(w, "stat: "+err.Error(), http.StatusInternalServerError)
	} else if stat.IsDir() && !strings.HasSuffix(r.URL.Path, "/") {
		// If client requests ".../dirname", redirect to
		// ".../dirname/". This way, relative links in the
		// listing for "dirname" can always be "fnm", never
		// "dirname/fnm".
		h.seeOtherWithCookie(w, r, r.URL.Path+"/", credentialsOK)
	} else if stat.IsDir() {
		h.serveDirectory(w, r, collection.Name, fs, openPath, true)
	} else {
		if !h.userPermittedToUploadOrDownload(r.Method, tokenUser) {
			http.Error(w, "Not permitted", http.StatusForbidden)
			return
		}
		h.logUploadOrDownload(r, sess.arvadosclient, nil, strings.Join(targetPath, "/"), collection, tokenUser)

		http.ServeContent(w, r, basename, stat.ModTime(), f)
		if wrote := int64(w.WroteBodyBytes()); wrote != stat.Size() && w.WroteStatus() == http.StatusOK {
			// If we wrote fewer bytes than expected, it's
			// too late to change the real response code
			// or send an error message to the client, but
			// at least we can try to put some useful
			// debugging info in the logs.
			n, err := f.Read(make([]byte, 1024))
			ctxlog.FromContext(r.Context()).Errorf("stat.Size()==%d but only wrote %d bytes; read(1024) returns %d, %v", stat.Size(), wrote, n, err)
		}
	}
}

func (h *handler) getClients(reqID, token string) (arv *arvadosclient.ArvadosClient, kc *keepclient.KeepClient, client *arvados.Client, release func(), err error) {
	arv = h.clientPool.Get()
	if arv == nil {
		err = h.clientPool.Err()
		return
	}
	release = func() { h.clientPool.Put(arv) }
	arv.ApiToken = token
	kc, err = keepclient.MakeKeepClient(arv)
	if err != nil {
		release()
		return
	}
	kc.RequestID = reqID
	client = (&arvados.Client{
		APIHost:   arv.ApiServer,
		AuthToken: arv.ApiToken,
		Insecure:  arv.ApiInsecure,
	}).WithRequestID(reqID)
	return
}

func (h *handler) serveSiteFS(w http.ResponseWriter, r *http.Request, tokens []string, credentialsOK, attachment bool) {
	if len(tokens) == 0 {
		w.Header().Add("WWW-Authenticate", "Basic realm=\"collections\"")
		http.Error(w, unauthorizedMessage, http.StatusUnauthorized)
		return
	}
	if writeMethod[r.Method] {
		http.Error(w, errReadOnly.Error(), http.StatusMethodNotAllowed)
		return
	}

	fs, sess, err := h.Config.Cache.GetSession(tokens[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fs.ForwardSlashNameSubstitution(h.Config.cluster.Collections.ForwardSlashNameSubstitution)
	f, err := fs.Open(r.URL.Path)
	if os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	if fi, err := f.Stat(); err == nil && fi.IsDir() && r.Method == "GET" {
		if !strings.HasSuffix(r.URL.Path, "/") {
			h.seeOtherWithCookie(w, r, r.URL.Path+"/", credentialsOK)
		} else {
			h.serveDirectory(w, r, fi.Name(), fs, r.URL.Path, false)
		}
		return
	}

	tokenUser, err := h.Config.Cache.GetTokenUser(tokens[0])
	if !h.userPermittedToUploadOrDownload(r.Method, tokenUser) {
		http.Error(w, "Not permitted", http.StatusForbidden)
		return
	}
	h.logUploadOrDownload(r, sess.arvadosclient, fs, r.URL.Path, nil, tokenUser)

	if r.Method == "GET" {
		_, basename := filepath.Split(r.URL.Path)
		applyContentDispositionHdr(w, r, basename, attachment)
	}
	wh := webdav.Handler{
		Prefix: "/",
		FileSystem: &webdavFS{
			collfs:        fs,
			writing:       writeMethod[r.Method],
			alwaysReadEOF: r.Method == "PROPFIND",
		},
		LockSystem: h.webdavLS,
		Logger: func(_ *http.Request, err error) {
			if err != nil {
				ctxlog.FromContext(r.Context()).WithError(err).Error("error reported by webdav handler")
			}
		},
	}
	wh.ServeHTTP(w, r)
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
		permitUpload = h.Config.cluster.Collections.WebDAVPermission.Admin.Upload
		permitDownload = h.Config.cluster.Collections.WebDAVPermission.Admin.Download
	} else {
		permitUpload = h.Config.cluster.Collections.WebDAVPermission.User.Upload
		permitDownload = h.Config.cluster.Collections.WebDAVPermission.User.Download
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
		useruuid = fmt.Sprintf("%s-tpzed-anonymouspublic", h.Config.cluster.ClusterID)
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
		if h.Config.cluster.Collections.WebDAVLogEvents {
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
		if h.Config.cluster.Collections.WebDAVLogEvents {
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
	for {
		fi, err := fs.Stat(target)
		if err != nil {
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
		// Try parent
		cut := strings.LastIndexByte(target, '/')
		if cut < 0 {
			return nil, ""
		}
		target = target[:cut]
	}
}
