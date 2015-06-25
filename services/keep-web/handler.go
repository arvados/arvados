package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

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

func (h *handler) ServeHTTP(wOrig http.ResponseWriter, r *http.Request) {
	var statusCode int
	var statusText string

	w := httpserver.WrapResponseWriter(wOrig)
	defer func() {
		if statusCode > 0 {
			if w.WroteStatus() == 0 {
				w.WriteHeader(statusCode)
			} else {
				httpserver.Log(r.RemoteAddr, "WARNING",
					fmt.Sprintf("Our status changed from %d to %d after we sent headers", w.WroteStatus(), statusCode))
			}
		}
		if statusText == "" {
			statusText = http.StatusText(statusCode)
		}
		httpserver.Log(r.RemoteAddr, statusCode, statusText, w.WroteBodyBytes(), r.Method, r.URL.Path)
	}()

	arv := clientPool.Get()
	if arv == nil {
		statusCode, statusText = http.StatusInternalServerError, "Pool failed: "+clientPool.Err().Error()
		return
	}
	defer clientPool.Put(arv)

	pathParts := strings.Split(r.URL.Path[1:], "/")

	if len(pathParts) < 3 || pathParts[0] != "collections" || pathParts[1] == "" || pathParts[2] == "" {
		statusCode = http.StatusNotFound
		return
	}

	var targetId string
	var targetPath []string
	var tokens []string
	var reqTokens []string
	var pathToken bool
	if len(pathParts) >= 5 && pathParts[1] == "download" {
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
		httpserver.Log(err)
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
		statusCode = http.StatusUnauthorized
		w.Header().Add("WWW-Authenticate", "Basic realm=\"dl\"")
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

	_, err = io.Copy(w, rdr)
	if err != nil {
		statusCode, statusText = http.StatusBadGateway, err.Error()
	}
}
