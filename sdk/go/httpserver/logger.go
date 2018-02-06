// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package httpserver

import (
	"context"
	"net/http"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/stats"
	"github.com/Sirupsen/logrus"
)

type contextKey struct {
	name string
}

var requestTimeContextKey = contextKey{"requestTime"}

var Logger logrus.FieldLogger = logrus.StandardLogger()

// LogRequests wraps an http.Handler, logging each request and
// response via logrus.
func LogRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(wrapped http.ResponseWriter, req *http.Request) {
		w := &responseTimer{ResponseWriter: WrapResponseWriter(wrapped)}
		req = req.WithContext(context.WithValue(req.Context(), &requestTimeContextKey, time.Now()))
		lgr := Logger.WithFields(logrus.Fields{
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

func logRequest(w *responseTimer, req *http.Request, lgr *logrus.Entry) {
	lgr.Info("request")
}

func logResponse(w *responseTimer, req *http.Request, lgr *logrus.Entry) {
	if tStart, ok := req.Context().Value(&requestTimeContextKey).(time.Time); ok {
		tDone := time.Now()
		lgr = lgr.WithFields(logrus.Fields{
			"timeTotal":     stats.Duration(tDone.Sub(tStart)),
			"timeToStatus":  stats.Duration(w.writeTime.Sub(tStart)),
			"timeWriteBody": stats.Duration(tDone.Sub(w.writeTime)),
		})
	}
	lgr.WithFields(logrus.Fields{
		"respStatusCode": w.WroteStatus(),
		"respStatus":     http.StatusText(w.WroteStatus()),
		"respBytes":      w.WroteBodyBytes(),
	}).Info("response")
}

type responseTimer struct {
	ResponseWriter
	wrote     bool
	writeTime time.Time
}

func (rt *responseTimer) CloseNotify() <-chan bool {
	if cn, ok := rt.ResponseWriter.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}
	return nil
}

func (rt *responseTimer) WriteHeader(code int) {
	if !rt.wrote {
		rt.wrote = true
		rt.writeTime = time.Now()
	}
	rt.ResponseWriter.WriteHeader(code)
}

func (rt *responseTimer) Write(p []byte) (int, error) {
	if !rt.wrote {
		rt.wrote = true
		rt.writeTime = time.Now()
	}
	return rt.ResponseWriter.Write(p)
}
