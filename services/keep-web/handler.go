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
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/health"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
	"golang.org/x/net/webdav"
)

type handler struct {
	Config        *Config
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
	h.clientPool = arvadosclient.MakeClientPool()

	keepclient.RefreshServiceDiscoveryOnSIGHUP()

	h.healthHandler = &health.Handler{
		Token:  h.Config.ManagementToken,
		Prefix: "/_health/",
	}

	// Even though we don't accept LOCK requests, every webdav
	// handler must have a non-nil LockSystem.
	h.webdavLS = &noLockSystem{}
}

func (h *handler) serveStatus(w http.ResponseWriter, r *http.Request) {
	status := struct {
		cacheStats
		Version string
	}{
		cacheStats: h.Config.Cache.Stats(),
		Version:    version,
	}
	json.NewEncoder(w).Encode(status)
}

// updateOnSuccess wraps httpserver.ResponseWriter. If the handler
// sends an HTTP header indicating success, updateOnSuccess first
// calls the provided update func. If the update func fails, a 500
// response is sent, and the status code and body sent by the handler
// are ignored (all response writes return the update error).
type updateOnSuccess struct {
	httpserver.ResponseWriter
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
				log.Printf("update() changes response to HTTP %d: %T %q", code, uos.err, uos.err)
				http.Error(uos.ResponseWriter, uos.err.Error(), code)
				return
			}
		}
	}
	uos.ResponseWriter.WriteHeader(code)
}

var (
	writeMethod = map[string]bool{
		"COPY":   true,
		"DELETE": true,
		"MKCOL":  true,
		"MOVE":   true,
		"PUT":    true,
		"RMCOL":  true,
	}
	webdavMethod = map[string]bool{
		"COPY":     true,
		"DELETE":   true,
		"MKCOL":    true,
		"MOVE":     true,
		"OPTIONS":  true,
		"PROPFIND": true,
		"PUT":      true,
		"RMCOL":    true,
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

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(wOrig http.ResponseWriter, r *http.Request) {
	h.setupOnce.Do(h.setup)

	var statusCode = 0
	var statusText string

	remoteAddr := r.RemoteAddr
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		remoteAddr = xff + "," + remoteAddr
	}

	w := httpserver.WrapResponseWriter(wOrig)
	defer func() {
		if statusCode == 0 {
			statusCode = w.WroteStatus()
		} else if w.WroteStatus() == 0 {
			w.WriteHeader(statusCode)
		} else if w.WroteStatus() != statusCode {
			httpserver.Log(r.RemoteAddr, "WARNING",
				fmt.Sprintf("Our status changed from %d to %d after we sent headers", w.WroteStatus(), statusCode))
		}
		if statusText == "" {
			statusText = http.StatusText(statusCode)
		}
		httpserver.Log(remoteAddr, statusCode, statusText, w.WroteBodyBytes(), r.Method, r.Host, r.URL.Path, r.URL.RawQuery)
	}()

	if strings.HasPrefix(r.URL.Path, "/_health/") && r.Method == "GET" {
		h.healthHandler.ServeHTTP(w, r)
		return
	}

	if method := r.Header.Get("Access-Control-Request-Method"); method != "" && r.Method == "OPTIONS" {
		if !browserMethod[method] && !webdavMethod[method] {
			statusCode = http.StatusMethodNotAllowed
			return
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Range")
		w.Header().Set("Access-Control-Allow-Methods", "COPY, DELETE, GET, MKCOL, MOVE, OPTIONS, POST, PROPFIND, PUT, RMCOL")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Max-Age", "86400")
		statusCode = http.StatusOK
		return
	}

	if !browserMethod[r.Method] && !webdavMethod[r.Method] {
		statusCode, statusText = http.StatusMethodNotAllowed, r.Method
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

	pathParts := strings.Split(r.URL.Path[1:], "/")

	var stripParts int
	var collectionID string
	var tokens []string
	var reqTokens []string
	var pathToken bool
	var attachment bool
	var useSiteFS bool
	credentialsOK := h.Config.TrustAllContent

	if r.Host != "" && r.Host == h.Config.AttachmentOnlyHost {
		credentialsOK = true
		attachment = true
	} else if r.FormValue("disposition") == "attachment" {
		attachment = true
	}

	if collectionID = parseCollectionIDFromDNSName(r.Host); collectionID != "" {
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
			tokens = h.Config.AnonymousTokens
			stripParts = 2
		}
	}

	if collectionID == "" && !useSiteFS {
		statusCode = http.StatusNotFound
		return
	}

	forceReload := false
	if cc := r.Header.Get("Cache-Control"); strings.Contains(cc, "no-cache") || strings.Contains(cc, "must-revalidate") {
		forceReload = true
	}

	formToken := r.FormValue("api_token")
	if formToken != "" && r.Header.Get("Origin") != "" && attachment && r.URL.Query().Get("api_token") == "" {
		// The client provided an explicit token in the POST
		// body. The Origin header indicates this *might* be
		// an AJAX request, in which case redirect-with-cookie
		// won't work: we should just serve the content in the
		// POST response. This is safe because:
		//
		// * We're supplying an attachment, not inline
		//   content, so we don't need to convert the POST to
		//   a GET and avoid the "really resubmit form?"
		//   problem.
		//
		// * The token isn't embedded in the URL, so we don't
		//   need to worry about bookmarks and copy/paste.
		tokens = append(tokens, formToken)
	} else if formToken != "" && browserMethod[r.Method] {
		// The client provided an explicit token in the query
		// string, or a form in POST body. We must put the
		// token in an HttpOnly cookie, and redirect to the
		// same URL with the query param redacted and method =
		// GET.
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

	if tokens == nil {
		if credentialsOK {
			reqTokens = auth.NewCredentialsFromHTTPRequest(r).Tokens
		}
		tokens = append(reqTokens, h.Config.AnonymousTokens...)
	}

	if useSiteFS {
		h.serveSiteFS(w, r, tokens, credentialsOK, attachment)
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
		statusCode, statusText = http.StatusInternalServerError, "Pool failed: "+h.clientPool.Err().Error()
		return
	}
	defer h.clientPool.Put(arv)

	var collection *arvados.Collection
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
		statusCode, statusText = http.StatusInternalServerError, err.Error()
		return
	}
	if collection == nil {
		if pathToken || !credentialsOK {
			// Either the URL is a "secret sharing link"
			// that didn't work out (and asking the client
			// for additional credentials would just be
			// confusing), or we don't even accept
			// credentials at this path.
			statusCode = http.StatusNotFound
			return
		}
		for _, t := range reqTokens {
			if tokenResult[t] == 404 {
				// The client provided valid token(s), but the
				// collection was not found.
				statusCode = http.StatusNotFound
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
		statusCode = http.StatusUnauthorized
		return
	}

	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		statusCode, statusText = http.StatusInternalServerError, err.Error()
		return
	}

	var basename string
	if len(targetPath) > 0 {
		basename = targetPath[len(targetPath)-1]
	}
	applyContentDispositionHdr(w, r, basename, attachment)

	client := &arvados.Client{
		APIHost:   arv.ApiServer,
		AuthToken: arv.ApiToken,
		Insecure:  arv.ApiInsecure,
	}

	fs, err := collection.FileSystem(client, kc)
	if err != nil {
		statusCode, statusText = http.StatusInternalServerError, err.Error()
		return
	}

	writefs, writeOK := fs.(arvados.CollectionFileSystem)
	targetIsPDH := arvadosclient.PDHMatch(collectionID)
	if (targetIsPDH || !writeOK) && writeMethod[r.Method] {
		statusCode, statusText = http.StatusMethodNotAllowed, errReadOnly.Error()
		return
	}

	if webdavMethod[r.Method] {
		if writeMethod[r.Method] {
			// Save the collection only if/when all
			// webdav->filesystem operations succeed --
			// and send a 500 error if the modified
			// collection can't be saved.
			w = &updateOnSuccess{
				ResponseWriter: w,
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
					log.Printf("error from webdav handler: %q", err)
				}
			},
		}
		h.ServeHTTP(w, r)
		return
	}

	openPath := "/" + strings.Join(targetPath, "/")
	if f, err := fs.Open(openPath); os.IsNotExist(err) {
		// Requested non-existent path
		statusCode = http.StatusNotFound
	} else if err != nil {
		// Some other (unexpected) error
		statusCode, statusText = http.StatusInternalServerError, err.Error()
	} else if stat, err := f.Stat(); err != nil {
		// Can't get Size/IsDir (shouldn't happen with a collectionFS!)
		statusCode, statusText = http.StatusInternalServerError, err.Error()
	} else if stat.IsDir() && !strings.HasSuffix(r.URL.Path, "/") {
		// If client requests ".../dirname", redirect to
		// ".../dirname/". This way, relative links in the
		// listing for "dirname" can always be "fnm", never
		// "dirname/fnm".
		h.seeOtherWithCookie(w, r, r.URL.Path+"/", credentialsOK)
	} else if stat.IsDir() {
		h.serveDirectory(w, r, collection.Name, fs, openPath, true)
	} else {
		http.ServeContent(w, r, basename, stat.ModTime(), f)
		if r.Header.Get("Range") == "" && int64(w.WroteBodyBytes()) != stat.Size() {
			// If we wrote fewer bytes than expected, it's
			// too late to change the real response code
			// or send an error message to the client, but
			// at least we can try to put some useful
			// debugging info in the logs.
			n, err := f.Read(make([]byte, 1024))
			statusCode, statusText = http.StatusInternalServerError, fmt.Sprintf("f.Size()==%d but only wrote %d bytes; read(1024) returns %d, %s", stat.Size(), w.WroteBodyBytes(), n, err)

		}
	}
}

func (h *handler) serveSiteFS(w http.ResponseWriter, r *http.Request, tokens []string, credentialsOK, attachment bool) {
	if len(tokens) == 0 {
		w.Header().Add("WWW-Authenticate", "Basic realm=\"collections\"")
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if writeMethod[r.Method] {
		http.Error(w, errReadOnly.Error(), http.StatusMethodNotAllowed)
		return
	}
	arv := h.clientPool.Get()
	if arv == nil {
		http.Error(w, "Pool failed: "+h.clientPool.Err().Error(), http.StatusInternalServerError)
		return
	}
	defer h.clientPool.Put(arv)
	arv.ApiToken = tokens[0]

	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := &arvados.Client{
		APIHost:   arv.ApiServer,
		AuthToken: arv.ApiToken,
		Insecure:  arv.ApiInsecure,
	}
	fs := client.SiteFileSystem(kc)
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
				log.Printf("error from webdav handler: %q", err)
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
  <LI>{{" " | printf "%15s  " | nbsp}}<A href="{{.Name}}/">{{.Name}}/</A></LI>
{{else}}
  <LI>{{.Size | printf "%15d  " | nbsp}}<A href="{{.Name}}">{{.Name}}</A></LI>
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	funcs := template.FuncMap{
		"nbsp": func(s string) template.HTML {
			return template.HTML(strings.Replace(s, " ", "&nbsp;", -1))
		},
	}
	tmpl, err := template.New("dir").Funcs(funcs).Parse(dirListingTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
			w.WriteHeader(http.StatusBadRequest)
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		u = newu
	}
	redir := (&url.URL{
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
