package main

import (
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/auth"
	"git.curoverse.com/arvados.git/sdk/go/httpserver"
	"git.curoverse.com/arvados.git/sdk/go/keepclient"
)

var clientPool = arvadosclient.MakeClientPool()

var anonymousTokens []string

type handler struct{}

func init() {
	// TODO(TC): Get anonymousTokens from flags
	anonymousTokens = []string{}
}

// return a UUID or PDH if s begins with a UUID or URL-encoded PDH;
// otherwise return "".
func parseCollectionIdFromDNSName(s string) string {
	// Strip domain.
	if i := strings.IndexRune(s, '.'); i >= 0 {
		s = s[:i]
	}
	// Names like {uuid}--dl.example.com serve the same purpose as
	// {uuid}.dl.example.com but can reduce cost/effort of using
	// [additional] wildcard certificates.
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

func (h *handler) ServeHTTP(wOrig http.ResponseWriter, r *http.Request) {
	var statusCode = 0
	var statusText string

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
		httpserver.Log(r.RemoteAddr, statusCode, statusText, w.WroteBodyBytes(), r.Method, r.Host, r.URL.Path, r.URL.RawQuery)
	}()

	if r.Method != "GET" && r.Method != "POST" {
		statusCode, statusText = http.StatusMethodNotAllowed, r.Method
		return
	}

	arv := clientPool.Get()
	if arv == nil {
		statusCode, statusText = http.StatusInternalServerError, "Pool failed: "+clientPool.Err().Error()
		return
	}
	defer clientPool.Put(arv)

	pathParts := strings.Split(r.URL.Path[1:], "/")

	var targetId string
	var targetPath []string
	var tokens []string
	var reqTokens []string
	var pathToken bool

	if targetId = parseCollectionIdFromDNSName(r.Host); targetId != "" {
		// "http://{id}.domain.example.com/{path}" form
		if t := r.FormValue("api_token"); t != "" {
			// ...with explicit token in query string or
			// form in POST body. We must encrypt the
			// token such that it can only be used for
			// this collection; put it in an HttpOnly
			// cookie; and redirect to the same URL with
			// the query param redacted, and method =
			// GET.
			//
			// The HttpOnly flag is necessary to prevent
			// JavaScript code (included in, or loaded by,
			// a page in the collection being served) from
			// employing the user's token beyond reading
			// other files in the same domain, i.e., same
			// the collection.
			//
			// The 303 redirect is necessary in the case
			// of a GET request to avoid exposing the
			// token in the Location bar, and in the case
			// of a POST request to avoid raising warnings
			// when the user refreshes the resulting page.
			http.SetCookie(w, &http.Cookie{
				Name:    "api_token",
				Value:   auth.EncodeTokenCookie([]byte(t)),
				Path:    "/",
				Expires: time.Now().AddDate(10,0,0),
			})
			redir := (&url.URL{Host: r.Host, Path: r.URL.Path}).String()

			w.Header().Add("Location", redir)
			statusCode, statusText = http.StatusSeeOther, redir
			w.WriteHeader(statusCode)
			io.WriteString(w, `<A href="`)
			io.WriteString(w, html.EscapeString(redir))
			io.WriteString(w, `">Continue</A>`)
			return
		} else if strings.HasPrefix(pathParts[0], "t=") {
			// ...with explicit token in path,
			// "{...}.com/t={token}/{path}".  This form
			// must only be used to pass scoped tokens
			// that give permission for a single
			// collection. See FormValue case above.
			tokens = []string{pathParts[0][2:]}
			targetPath = pathParts[1:]
			pathToken = true
		} else {
			// ...with cookie, Authorization header, or
			// no token at all
			reqTokens = auth.NewCredentialsFromHTTPRequest(r).Tokens
			tokens = append(reqTokens, anonymousTokens...)
			targetPath = pathParts
		}
	} else if len(pathParts) < 3 || pathParts[0] != "collections" || pathParts[1] == "" || pathParts[2] == "" {
		statusCode = http.StatusNotFound
		return
	} else if len(pathParts) >= 5 && pathParts[1] == "download" {
		// "/collections/download/{id}/{token}/path..." form:
		// Don't use our configured anonymous tokens,
		// Authorization headers, etc.  Just use the token in
		// the path.
		targetId = pathParts[2]
		tokens = []string{pathParts[3]}
		targetPath = pathParts[4:]
		pathToken = true
	} else {
		// "/collections/{id}/path..." form
		targetId = pathParts[1]
		reqTokens = auth.NewCredentialsFromHTTPRequest(r).Tokens
		tokens = append(reqTokens, anonymousTokens...)
		targetPath = pathParts[2:]
	}

	tokenResult := make(map[string]int)
	collection := make(map[string]interface{})
	found := false
	for _, arv.ApiToken = range tokens {
		err := arv.Get("collections", targetId, nil, &collection)
		if err == nil {
			// Success
			found = true
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
	if !found {
		if pathToken {
			// The URL is a "secret sharing link", but it
			// didn't work out. Asking the client for
			// additional credentials would just be
			// confusing.
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
		w.Header().Add("WWW-Authenticate", "Basic realm=\"dl\"")
		statusCode = http.StatusUnauthorized
		return
	}

	filename := strings.Join(targetPath, "/")
	kc, err := keepclient.MakeKeepClient(arv)
	if err != nil {
		statusCode, statusText = http.StatusInternalServerError, err.Error()
		return
	}
	rdr, err := kc.CollectionFileReader(collection, filename)
	if os.IsNotExist(err) {
		statusCode = http.StatusNotFound
		return
	} else if err != nil {
		statusCode, statusText = http.StatusBadGateway, err.Error()
		return
	}
	defer rdr.Close()

	// One or both of these can be -1 if not found:
	basenamePos := strings.LastIndex(filename, "/")
	extPos := strings.LastIndex(filename, ".")
	if extPos > basenamePos {
		// Now extPos is safely >= 0.
		if t := mime.TypeByExtension(filename[extPos:]); t != "" {
			w.Header().Set("Content-Type", t)
		}
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", rdr.Len()))

	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, rdr)
	if err != nil {
		statusCode, statusText = http.StatusBadGateway, err.Error()
	}
}
