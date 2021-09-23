// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"context"
	"net/http"
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/stats"
	"github.com/sirupsen/logrus"
)

type contextKey struct {
	name string
}

var (
	requestTimeContextKey = contextKey{"requestTime"}
)

// HandlerWithDeadline cancels the request context if the request
// takes longer than the specified timeout.
func HandlerWithDeadline(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithDeadline(r.Context(), time.Now().Add(timeout))
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LogRequests wraps an http.Handler, logging each request and
// response.
func LogRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(wrapped http.ResponseWriter, req *http.Request) {
		w := &responseTimer{ResponseWriter: WrapResponseWriter(wrapped)}
		lgr := ctxlog.FromContext(req.Context()).WithFields(logrus.Fields{
			"RequestID":       req.Header.Get("X-Request-Id"),
			"remoteAddr":      req.RemoteAddr,
			"reqForwardedFor": req.Header.Get("X-Forwarded-For"),
			"reqMethod":       req.Method,
			"reqHost":         req.Host,
			"reqPath":         req.URL.Path[1:],
			"reqQuery":        req.URL.RawQuery,
			"reqBytes":        req.ContentLength,
		})
		ctx := req.Context()
		ctx = context.WithValue(ctx, &requestTimeContextKey, time.Now())
		ctx = ctxlog.Context(ctx, lgr)
		req = req.WithContext(ctx)

		logRequest(w, req, lgr)
		defer logResponse(w, req, lgr)
		h.ServeHTTP(rewrapResponseWriter(w, wrapped), req)
	})
}

// Rewrap w to restore additional interfaces provided by wrapped.
func rewrapResponseWriter(w http.ResponseWriter, wrapped http.ResponseWriter) http.ResponseWriter {
	if hijacker, ok := wrapped.(http.Hijacker); ok {
		return struct {
			http.ResponseWriter
			http.Hijacker
		}{w, hijacker}
	}
	return w
}

func Logger(req *http.Request) logrus.FieldLogger {
	return ctxlog.FromContext(req.Context())
}

func logRequest(w *responseTimer, req *http.Request, lgr *logrus.Entry) {
	lgr.Info("request")
}

func logResponse(w *responseTimer, req *http.Request, lgr *logrus.Entry) {
	if tStart, ok := req.Context().Value(&requestTimeContextKey).(time.Time); ok {
		tDone := time.Now()
		writeTime := w.writeTime
		if !w.wrote {
			// Empty response body. Header was sent when
			// handler exited.
			writeTime = tDone
		}
		lgr = lgr.WithFields(logrus.Fields{
			"timeTotal":     stats.Duration(tDone.Sub(tStart)),
			"timeToStatus":  stats.Duration(writeTime.Sub(tStart)),
			"timeWriteBody": stats.Duration(tDone.Sub(writeTime)),
		})
	}
	respCode := w.WroteStatus()
	if respCode == 0 {
		respCode = http.StatusOK
	}
	fields := logrus.Fields{
		"respStatusCode": respCode,
		"respStatus":     http.StatusText(respCode),
		"respBytes":      w.WroteBodyBytes(),
	}
	if respCode >= 400 {
		fields["respBody"] = string(w.Sniffed())
	}
	lgr.WithFields(fields).Info("response")
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
