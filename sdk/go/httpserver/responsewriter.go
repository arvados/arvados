// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package httpserver

import (
	"bufio"
	"net"
	"net/http"
	"time"
)

const sniffBytes = 1024

type ResponseWriter interface {
	http.ResponseWriter
	WroteStatus() int
	WroteBodyBytes() int
	Sniffed() []byte
}

// responseWriter wraps http.ResponseWriter and exposes the status
// sent, the number of bytes sent to the client, and the last write
// error.
type responseWriter struct {
	http.ResponseWriter
	wroteStatus    int   // First status given to WriteHeader()
	wroteBodyBytes int   // Bytes successfully written
	err            error // Last error returned from Write()
	sniffed        []byte
}

func WrapResponseWriter(orig http.ResponseWriter) ResponseWriter {
	return &responseWriter{ResponseWriter: orig}
}

func (w *responseWriter) WriteHeader(s int) {
	if w.wroteStatus == 0 {
		w.wroteStatus = s
	}
	// ...else it's too late to change the status seen by the
	// client -- but we call the wrapped WriteHeader() anyway so
	// it can log a warning.
	w.ResponseWriter.WriteHeader(s)
}

func (w *responseWriter) Write(data []byte) (n int, err error) {
	if w.wroteStatus == 0 {
		w.WriteHeader(http.StatusOK)
	} else if w.wroteStatus >= 400 {
		w.sniff(data)
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

func (w *responseWriter) sniff(data []byte) {
	max := sniffBytes - len(w.sniffed)
	if max <= 0 {
		return
	} else if max < len(data) {
		data = data[:max]
	}
	w.sniffed = append(w.sniffed, data...)
}

func (w *responseWriter) Sniffed() []byte {
	return w.sniffed
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// ResponseControllerShim uses a ResponseController to re-add the
// optional interface methods to a ResponseWriter that has lost them
// via wrapping by middleware.
//
// This allows us to combine old code (like x/net/websocket) with
// middleware that doesn't explicitly support the optional interfaces
// (like responseTimer and responseWriter here).
type ResponseControllerShim struct{ http.ResponseWriter }

func (s ResponseControllerShim) EnableFullDuplex() error {
	return http.NewResponseController(s.ResponseWriter).EnableFullDuplex()
}

func (s ResponseControllerShim) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(s.ResponseWriter).Hijack()
}

func (s ResponseControllerShim) SetReadDeadline(d time.Time) error {
	return http.NewResponseController(s.ResponseWriter).SetReadDeadline(d)
}

func (s ResponseControllerShim) SetWriteDeadline(d time.Time) error {
	return http.NewResponseController(s.ResponseWriter).SetWriteDeadline(d)
}

func (s ResponseControllerShim) Flush() error {
	return http.NewResponseController(s.ResponseWriter).Flush()
}
