// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/gorilla/mux"
)

type router struct {
	http.Handler
	keepstore *keepstore
	puller    *puller
	trasher   *trasher
}

func newRouter(keepstore *keepstore, puller *puller, trasher *trasher) service.Handler {
	rtr := &router{
		keepstore: keepstore,
		puller:    puller,
		trasher:   trasher,
	}
	adminonly := func(h http.HandlerFunc) http.HandlerFunc {
		return auth.RequireLiteralToken(keepstore.cluster.SystemRootToken, h).ServeHTTP
	}

	r := mux.NewRouter()
	locatorPath := `/{locator:[0-9a-f]{32}.*}`
	get := r.Methods(http.MethodGet, http.MethodHead).Subrouter()
	get.HandleFunc(locatorPath, rtr.handleBlockRead)
	get.HandleFunc(`/index`, adminonly(rtr.handleIndex))
	get.HandleFunc(`/index/{prefix:[0-9a-f]{0,32}}`, adminonly(rtr.handleIndex))
	get.HandleFunc(`/mounts`, adminonly(rtr.handleMounts))
	get.HandleFunc(`/mounts/{uuid}/blocks`, adminonly(rtr.handleIndex))
	get.HandleFunc(`/mounts/{uuid}/blocks/{prefix:[0-9a-f]{0,32}}`, adminonly(rtr.handleIndex))
	put := r.Methods(http.MethodPut).Subrouter()
	put.HandleFunc(locatorPath, rtr.handleBlockWrite)
	put.HandleFunc(`/pull`, adminonly(rtr.handlePullList))
	put.HandleFunc(`/trash`, adminonly(rtr.handleTrashList))
	put.HandleFunc(`/untrash`+locatorPath, adminonly(rtr.handleUntrash))
	touch := r.Methods("TOUCH").Subrouter()
	touch.HandleFunc(locatorPath, adminonly(rtr.handleBlockTouch))
	delete := r.Methods(http.MethodDelete).Subrouter()
	delete.HandleFunc(locatorPath, adminonly(rtr.handleBlockTrash))
	r.NotFoundHandler = http.HandlerFunc(rtr.handleBadRequest)
	r.MethodNotAllowedHandler = http.HandlerFunc(rtr.handleBadRequest)
	rtr.Handler = auth.LoadToken(r)
	return rtr
}

func (rtr *router) CheckHealth() error {
	return nil
}

func (rtr *router) Done() <-chan struct{} {
	return nil
}

func (rtr *router) handleBlockRead(w http.ResponseWriter, req *http.Request) {
	// Intervening proxies must not return a cached GET response
	// to a prior request if a X-Keep-Signature request header has
	// been added or changed.
	w.Header().Add("Vary", "X-Keep-Signature")
	var localLocator func(string)
	if strings.SplitN(req.Header.Get("X-Keep-Signature"), ",", 2)[0] == "local" {
		localLocator = func(locator string) {
			w.Header().Set("X-Keep-Locator", locator)
		}
	}
	out := w
	if req.Method == http.MethodHead {
		out = discardWrite{ResponseWriter: w}
	} else if li, err := parseLocator(mux.Vars(req)["locator"]); err != nil {
		rtr.handleError(w, req, err)
		return
	} else if li.size == 0 && li.hash != "d41d8cd98f00b204e9800998ecf8427e" {
		// GET {hash} (with no size hint) is not allowed
		// because we can't report md5 mismatches.
		rtr.handleError(w, req, errMethodNotAllowed)
		return
	}
	n, err := rtr.keepstore.BlockRead(req.Context(), arvados.BlockReadOptions{
		Locator:      mux.Vars(req)["locator"],
		WriteTo:      out,
		LocalLocator: localLocator,
	})
	if err != nil && (n == 0 || req.Method == http.MethodHead) {
		rtr.handleError(w, req, err)
		return
	}
}

func (rtr *router) handleBlockWrite(w http.ResponseWriter, req *http.Request) {
	dataSize, _ := strconv.Atoi(req.Header.Get("Content-Length"))
	replicas, _ := strconv.Atoi(req.Header.Get("X-Arvados-Replicas-Desired"))
	resp, err := rtr.keepstore.BlockWrite(req.Context(), arvados.BlockWriteOptions{
		Hash:           mux.Vars(req)["locator"],
		Reader:         req.Body,
		DataSize:       dataSize,
		RequestID:      req.Header.Get("X-Request-Id"),
		StorageClasses: trimSplit(req.Header.Get("X-Keep-Storage-Classes"), ","),
		Replicas:       replicas,
	})
	if err != nil {
		rtr.handleError(w, req, err)
		return
	}
	w.Header().Set("X-Keep-Replicas-Stored", fmt.Sprintf("%d", resp.Replicas))
	scc := ""
	for k, n := range resp.StorageClasses {
		if n > 0 {
			if scc != "" {
				scc += "; "
			}
			scc += fmt.Sprintf("%s=%d", k, n)
		}
	}
	w.Header().Set("X-Keep-Storage-Classes-Confirmed", scc)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, resp.Locator)
}

func (rtr *router) handleBlockTouch(w http.ResponseWriter, req *http.Request) {
	err := rtr.keepstore.BlockTouch(req.Context(), mux.Vars(req)["locator"])
	rtr.handleError(w, req, err)
}

func (rtr *router) handleBlockTrash(w http.ResponseWriter, req *http.Request) {
	err := rtr.keepstore.BlockTrash(req.Context(), mux.Vars(req)["locator"])
	rtr.handleError(w, req, err)
}

func (rtr *router) handleMounts(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode(rtr.keepstore.Mounts())
}

func (rtr *router) handleIndex(w http.ResponseWriter, req *http.Request) {
	prefix := req.FormValue("prefix")
	if prefix == "" {
		prefix = mux.Vars(req)["prefix"]
	}
	cw := &countingWriter{writer: w}
	err := rtr.keepstore.Index(req.Context(), indexOptions{
		MountUUID: mux.Vars(req)["uuid"],
		Prefix:    prefix,
		WriteTo:   cw,
	})
	if err != nil && cw.n.Load() == 0 {
		// Nothing was written, so it's not too late to report
		// an error via http response header. (Otherwise, all
		// we can do is omit the trailing newline below to
		// indicate something went wrong.)
		rtr.handleError(w, req, err)
		return
	}
	if err == nil {
		// A trailing blank line signals to the caller that
		// the response is complete.
		w.Write([]byte("\n"))
	}
}

func (rtr *router) handlePullList(w http.ResponseWriter, req *http.Request) {
	var pl []PullListItem
	err := json.NewDecoder(req.Body).Decode(&pl)
	if err != nil {
		rtr.handleError(w, req, err)
		return
	}
	req.Body.Close()
	if len(pl) > 0 && len(pl[0].Locator) == 32 {
		rtr.handleError(w, req, httpserver.ErrorWithStatus(errors.New("rejecting pull list containing a locator without a size hint -- this probably means keep-balance needs to be upgraded"), http.StatusBadRequest))
		return
	}
	rtr.puller.SetPullList(pl)
}

func (rtr *router) handleTrashList(w http.ResponseWriter, req *http.Request) {
	var tl []TrashListItem
	err := json.NewDecoder(req.Body).Decode(&tl)
	if err != nil {
		rtr.handleError(w, req, err)
		return
	}
	req.Body.Close()
	rtr.trasher.SetTrashList(tl)
}

func (rtr *router) handleUntrash(w http.ResponseWriter, req *http.Request) {
	err := rtr.keepstore.BlockUntrash(req.Context(), mux.Vars(req)["locator"])
	rtr.handleError(w, req, err)
}

func (rtr *router) handleBadRequest(w http.ResponseWriter, req *http.Request) {
	http.Error(w, "Bad Request", http.StatusBadRequest)
}

func (rtr *router) handleError(w http.ResponseWriter, req *http.Request, err error) {
	if req.Context().Err() != nil {
		w.WriteHeader(499)
		return
	}
	if err == nil {
		return
	} else if os.IsNotExist(err) {
		w.WriteHeader(http.StatusNotFound)
	} else if statusErr := interface{ HTTPStatus() int }(nil); errors.As(err, &statusErr) {
		w.WriteHeader(statusErr.HTTPStatus())
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	fmt.Fprintln(w, err.Error())
}

type countingWriter struct {
	writer io.Writer
	n      atomic.Int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.writer.Write(p)
	cw.n.Add(int64(n))
	return n, err
}

// Split s by sep, trim whitespace from each part, and drop empty
// parts.
func trimSplit(s, sep string) []string {
	var r []string
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			r = append(r, part)
		}
	}
	return r
}

// setSizeOnWrite sets the Content-Length header to the given size on
// first write.
type setSizeOnWrite struct {
	http.ResponseWriter
	size  int
	wrote bool
}

func (ss *setSizeOnWrite) Write(p []byte) (int, error) {
	if !ss.wrote {
		ss.Header().Set("Content-Length", fmt.Sprintf("%d", ss.size))
		ss.wrote = true
	}
	return ss.ResponseWriter.Write(p)
}

type discardWrite struct {
	http.ResponseWriter
}

func (discardWrite) Write(p []byte) (int, error) {
	return len(p), nil
}
