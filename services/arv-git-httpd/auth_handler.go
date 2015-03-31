package main

import (
	"log"
	"net/http"
	"net/http/cgi"
	"os"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
)

func newArvadosClient() interface{} {
	arv, err := arvadosclient.MakeArvadosClient()
	if err != nil {
		log.Println("MakeArvadosClient:", err)
		return nil
	}
	return &arv
}

var connectionPool = &sync.Pool{New: newArvadosClient}

type spyingResponseWriter struct {
	http.ResponseWriter
	wroteStatus *int
}

func (w spyingResponseWriter) WriteHeader(s int) {
	*w.wroteStatus = s
	w.ResponseWriter.WriteHeader(s)
}

type authHandler struct {
	handler *cgi.Handler
}

func (h *authHandler) ServeHTTP(wOrig http.ResponseWriter, r *http.Request) {
	var statusCode int
	var statusText string
	var username, password string
	var repoName string
	var wroteStatus int

	w := spyingResponseWriter{wOrig, &wroteStatus}

	defer func() {
		if wroteStatus == 0 {
			// Nobody has called WriteHeader yet: that must be our job.
			w.WriteHeader(statusCode)
			w.Write([]byte(statusText))
		}
		log.Println(quoteStrings(r.RemoteAddr, username, password, wroteStatus, statusText, repoName, r.URL.Path)...)
	}()

	// HTTP request username is logged, but unused. Password is an
	// Arvados API token.
	username, password, ok := BasicAuth(r)
	if !ok || username == "" || password == "" {
		statusCode, statusText = http.StatusUnauthorized, "no credentials provided"
		w.Header().Add("WWW-Authenticate", "basic")
		return
	}

	// Access to paths "/foo/bar.git/*" and "/foo/bar/.git/*" are
	// protected by the permissions on the repository named
	// "foo/bar".
	pathParts := strings.SplitN(r.URL.Path[1:], ".git/", 2)
	if len(pathParts) != 2 {
		statusCode, statusText = http.StatusBadRequest, "bad request"
		return
	}
	repoName = pathParts[0]
	repoName = strings.TrimRight(repoName, "/")

	// Regardless of whether the client asked for "/foo.git" or
	// "/foo/.git", we choose whichever variant exists in our repo
	// root. If neither exists, we won't even bother checking
	// authentication.
	rewrittenPath := ""
	tryDirs := []string{
		"/" + repoName + ".git",
		"/" + repoName + "/.git",
	}
	for _, dir := range tryDirs {
		if fileInfo, err := os.Stat(theConfig.Root + dir); err != nil {
			if !os.IsNotExist(err) {
				statusCode, statusText = http.StatusInternalServerError, err.Error()
				return
			}
		} else if fileInfo.IsDir() {
			rewrittenPath = dir + "/" + pathParts[1]
			break
		}
	}
	if rewrittenPath == "" {
		statusCode, statusText = http.StatusNotFound, "not found"
		return
	}
	r.URL.Path = rewrittenPath

	arv, ok := connectionPool.Get().(*arvadosclient.ArvadosClient)
	if !ok || arv == nil {
		statusCode, statusText = http.StatusInternalServerError, "connection pool failed"
		return
	}
	defer connectionPool.Put(arv)

	// Ask API server whether the repository is readable using this token (by trying to read it!)
	arv.ApiToken = password
	reposFound := arvadosclient.Dict{}
	if err := arv.List("repositories", arvadosclient.Dict{
		"filters": [][]string{[]string{"name", "=", repoName}},
	}, &reposFound); err != nil {
		statusCode, statusText = http.StatusInternalServerError, err.Error()
		return
	}
	if avail, ok := reposFound["items_available"].(float64); !ok {
		statusCode, statusText = http.StatusInternalServerError, "bad list response from API"
		return
	} else if avail < 1 {
		statusCode, statusText = http.StatusNotFound, "not found"
		return
	} else if avail > 1 {
		statusCode, statusText = http.StatusInternalServerError, "name collision"
		return
	}
	isWrite := strings.HasSuffix(r.URL.Path, "/git-receive-pack")
	if !isWrite {
		statusText = "read"
	} else {
		uuid := reposFound["items"].([]interface{})[0].(map[string]interface{})["uuid"].(string)
		err := arv.Update("repositories", uuid, arvadosclient.Dict{
			"repository": arvadosclient.Dict{
				"modified_at": time.Now().String(),
			},
		}, &arvadosclient.Dict{})
		if err != nil {
			statusCode, statusText = http.StatusForbidden, err.Error()
			return
		}
		statusText = "write"
	}
	handlerCopy := *h.handler
	handlerCopy.Env = append(handlerCopy.Env, "REMOTE_USER="+r.RemoteAddr) // Should be username
	handlerCopy.ServeHTTP(&w, r)
}

var escaper = strings.NewReplacer("\"", "\\\"", "\\", "\\\\", "\n", "\\n")

// Transform strings so they are safer to write in logs (e.g.,
// 'foo"bar' becomes '"foo\"bar"'). Non-string args are left alone.
func quoteStrings(args ...interface{}) []interface{} {
	for i, arg := range args {
		if s, ok := arg.(string); ok {
			args[i] = "\"" + escaper.Replace(s) + "\""
		}
	}
	return args
}
