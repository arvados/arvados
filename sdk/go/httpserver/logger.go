// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package httpserver

import (
	"context"
	"net/http"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/stats"
	log "github.com/Sirupsen/logrus"
)

type contextKey struct {
	name string
}

var requestTimeContextKey = contextKey{"requestTime"}

// LogRequests wraps an http.Handler, logging each request and
// response via logrus.
func LogRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(wrapped http.ResponseWriter, req *http.Request) {
		w := WrapResponseWriter(wrapped)
		req = req.WithContext(context.WithValue(req.Context(), &requestTimeContextKey, time.Now()))
		lgr := log.WithFields(log.Fields{
			"RequestID":       req.Header.Get("X-Request-Id"),
			"remoteAddr":      req.RemoteAddr,
			"reqForwardedFor": req.Header.Get("X-Forwarded-For"),
			"reqMethod":       req.Method,
			"reqPath":         req.URL.Path[1:],
			"reqBytes":        req.ContentLength,
		})
		logRequest(w, req, lgr)
		defer logResponse(w, req, lgr)
		h.ServeHTTP(w, req)
	})
}

func logRequest(w ResponseWriter, req *http.Request, lgr *log.Entry) {
	lgr.Info("request")
}

func logResponse(w ResponseWriter, req *http.Request, lgr *log.Entry) {
	if tStart, ok := req.Context().Value(&requestTimeContextKey).(time.Time); ok {
		tDone := time.Now()
		lgr = lgr.WithFields(log.Fields{
			"timeTotal": stats.Duration(tDone.Sub(tStart)),
			// TODO: track WriteHeader timing
			// "timeToStatus":  stats.Duration(w.sentHdr.Sub(tStart)),
			// "timeWriteBody": stats.Duration(tDone.Sub(w.sentHdr)),
		})
	}
	lgr.WithFields(log.Fields{
		"respStatusCode": w.WroteStatus(),
		"respStatus":     http.StatusText(w.WroteStatus()),
		"respBytes":      w.WroteBodyBytes(),
	}).Info("response")
}
