// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"net/http"
)

type ResponseWriter interface {
	http.ResponseWriter
	WroteStatus() int
	WroteBodyBytes() int
}

// responseWriter wraps http.ResponseWriter and exposes the status
// sent, the number of bytes sent to the client, and the last write
// error.
type responseWriter struct {
	http.ResponseWriter
	wroteStatus    int   // Last status given to WriteHeader()
	wroteBodyBytes int   // Bytes successfully written
	err            error // Last error returned from Write()
}

func WrapResponseWriter(orig http.ResponseWriter) ResponseWriter {
	return &responseWriter{ResponseWriter: orig}
}

func (w *responseWriter) CloseNotify() <-chan bool {
	if cn, ok := w.ResponseWriter.(http.CloseNotifier); ok {
		return cn.CloseNotify()
	}
	return nil
}

func (w *responseWriter) WriteHeader(s int) {
	w.wroteStatus = s
	w.ResponseWriter.WriteHeader(s)
}

func (w *responseWriter) Write(data []byte) (n int, err error) {
	if w.wroteStatus == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err = w.ResponseWriter.Write(data)
	w.wroteBodyBytes += n
	w.err = err
	return
}

func (w *responseWriter) WroteStatus() int {
	return w.wroteStatus
}

func (w *responseWriter) WroteBodyBytes() int {
	return w.wroteBodyBytes
}

func (w *responseWriter) Err() error {
	return w.err
}
