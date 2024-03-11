// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"io"
	"math"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	mDownloadSpeed        *prometheus.HistogramVec
	mDownloadBackendSpeed *prometheus.HistogramVec
	mUploadSpeed          *prometheus.HistogramVec
	mUploadSyncDelay      *prometheus.HistogramVec
}

func newMetrics(reg *prometheus.Registry) *metrics {
	m := &metrics{
		mDownloadSpeed: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "arvados",
			Subsystem: "keepweb",
			Name:      "download_speed",
			Help:      "Download speed (bytes per second) bucketed by transfer size range",
			Buckets:   []float64{10_000, 1_000_000, 10_000_000, 100_000_000, 1_000_000_000, math.Inf(+1)},
		}, []string{"size_range"}),
		mDownloadBackendSpeed: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "arvados",
			Subsystem: "keepweb",
			Name:      "download_apparent_backend_speed",
			Help:      "Apparent download speed from the backend (bytes per second) when serving file downloads, bucketed by transfer size range (see https://dev.arvados.org/issues/15317#note-25 for explanation)",
			Buckets:   []float64{10_000, 1_000_000, 10_000_000, 100_000_000, 1_000_000_000, math.Inf(+1)},
		}, []string{"size_range"}),
		mUploadSpeed: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "arvados",
			Subsystem: "keepweb",
			Name:      "upload_speed",
			Help:      "Upload speed (bytes per second) bucketed by transfer size range",
			Buckets:   []float64{10_000, 1_000_000, 10_000_000, 100_000_000, 1_000_000_000, math.Inf(+1)},
		}, []string{"size_range"}),
		mUploadSyncDelay: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "arvados",
			Subsystem: "keepweb",
			Name:      "upload_sync_delay_seconds",
			Help:      "Upload sync delay (time from last byte received to HTTP response)",
		}, []string{"size_range"}),
	}
	reg.MustRegister(m.mDownloadSpeed)
	reg.MustRegister(m.mDownloadBackendSpeed)
	reg.MustRegister(m.mUploadSpeed)
	reg.MustRegister(m.mUploadSyncDelay)
	return m
}

// run handler(w,r) and record upload/download metrics as applicable.
func (m *metrics) track(handler http.Handler, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		dt := newDownloadTracker(w)
		handler.ServeHTTP(dt, r)
		size := dt.bytesOut
		if size == 0 {
			return
		}
		bucket := sizeRange(size)
		m.mDownloadSpeed.WithLabelValues(bucket).Observe(float64(dt.bytesOut) / time.Since(dt.t0).Seconds())
		m.mDownloadBackendSpeed.WithLabelValues(bucket).Observe(float64(size) / (dt.backendWait + time.Since(dt.lastByte)).Seconds())
	case http.MethodPut:
		ut := newUploadTracker(r)
		handler.ServeHTTP(w, r)
		d := ut.lastByte.Sub(ut.t0)
		if d <= 0 {
			// Read() was not called, or did not return
			// any data
			return
		}
		size := ut.bytesIn
		bucket := sizeRange(size)
		m.mUploadSpeed.WithLabelValues(bucket).Observe(float64(ut.bytesIn) / d.Seconds())
		m.mUploadSyncDelay.WithLabelValues(bucket).Observe(time.Since(ut.lastByte).Seconds())
	default:
		handler.ServeHTTP(w, r)
	}
}

// Assign a sizeRange based on number of bytes transferred (not the
// same as file size in the case of a Range request or interrupted
// transfer).
func sizeRange(size int64) string {
	switch {
	case size < 1_000_000:
		return "0"
	case size < 10_000_000:
		return "1M"
	case size < 100_000_000:
		return "10M"
	default:
		return "100M"
	}
}

type downloadTracker struct {
	http.ResponseWriter
	t0 time.Time

	firstByte   time.Time     // time of first call to Write
	lastByte    time.Time     // time of most recent call to Write
	bytesOut    int64         // bytes sent to client so far
	backendWait time.Duration // total of intervals between Write calls
}

func newDownloadTracker(w http.ResponseWriter) *downloadTracker {
	return &downloadTracker{ResponseWriter: w, t0: time.Now()}
}

func (dt *downloadTracker) Write(p []byte) (int, error) {
	if dt.lastByte.IsZero() {
		dt.backendWait += time.Since(dt.t0)
	} else {
		dt.backendWait += time.Since(dt.lastByte)
	}
	if dt.firstByte.IsZero() {
		dt.firstByte = time.Now()
	}
	n, err := dt.ResponseWriter.Write(p)
	dt.bytesOut += int64(n)
	dt.lastByte = time.Now()
	return n, err
}

type uploadTracker struct {
	io.ReadCloser
	t0       time.Time
	lastByte time.Time
	bytesIn  int64
}

func newUploadTracker(r *http.Request) *uploadTracker {
	now := time.Now()
	ut := &uploadTracker{ReadCloser: r.Body, t0: now}
	r.Body = ut
	return ut
}

func (ut *uploadTracker) Read(p []byte) (int, error) {
	n, err := ut.ReadCloser.Read(p)
	ut.lastByte = time.Now()
	ut.bytesIn += int64(n)
	return n, err
}
