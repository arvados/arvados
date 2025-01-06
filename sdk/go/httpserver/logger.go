// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"context"
	"net/http"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/stats"
	"github.com/sirupsen/logrus"
)

type contextKey struct {
	name string
}

var (
	requestTimeContextKey       = contextKey{"requestTime"}
	responseLogFieldsContextKey = contextKey{"responseLogFields"}
	mutexContextKey             = contextKey{"mutex"}
	stopDeadlineTimerContextKey = contextKey{"stopDeadlineTimer"}
)

// HandlerWithDeadline cancels the request context if the request
// takes longer than the specified timeout without having its
// connection hijacked.
//
// If timeout is 0, there is no deadline: HandlerWithDeadline is a
// no-op.
func HandlerWithDeadline(timeout time.Duration, next http.Handler) http.Handler {
	if timeout == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		timer := time.AfterFunc(timeout, cancel)
		ctx = context.WithValue(ctx, stopDeadlineTimerContextKey, timer.Stop)
		next.ServeHTTP(w, r.WithContext(ctx))
		timer.Stop()
	})
}

// ExemptFromDeadline exempts the given request from the timeout set
// by HandlerWithDeadline.
//
// It is a no-op if the deadline has already passed, or none was set.
func ExemptFromDeadline(r *http.Request) {
	if stop, ok := r.Context().Value(stopDeadlineTimerContextKey).(func() bool); ok {
		stop()
	}
}

func SetResponseLogFields(ctx context.Context, fields logrus.Fields) {
	m, _ := ctx.Value(&mutexContextKey).(*sync.Mutex)
	c, _ := ctx.Value(&responseLogFieldsContextKey).(logrus.Fields)
	if m == nil || c == nil {
		return
	}
	m.Lock()
	defer m.Unlock()
	for k, v := range fields {
		c[k] = v
	}
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
		ctx = context.WithValue(ctx, &responseLogFieldsContextKey, logrus.Fields{})
		ctx = context.WithValue(ctx, &mutexContextKey, &sync.Mutex{})
		ctx = ctxlog.Context(ctx, lgr)
		req = req.WithContext(ctx)

		logRequest(w, req, lgr)
		defer logResponse(w, req, lgr)
		h.ServeHTTP(w, req)
	})
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
	if responseLogFields, ok := req.Context().Value(&responseLogFieldsContextKey).(logrus.Fields); ok {
		lgr = lgr.WithFields(responseLogFields)
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

func (rt *responseTimer) Unwrap() http.ResponseWriter {
	return rt.ResponseWriter
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
