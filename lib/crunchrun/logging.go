// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"time"
)

// rfc3339NanoFixed is a fixed-width version of time.RFC3339Nano.
const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

// prefixer wraps an io.Writer, inserting a string returned by
// prefixFunc at the beginning of each line.
type prefixer struct {
	writer     io.Writer
	prefixFunc func() string
	unfinished bool // true if the most recent write ended with a non-newline char
}

// newTimestamper wraps an io.Writer, inserting an RFC3339NanoFixed
// timestamp at the beginning of each line.
func newTimestamper(w io.Writer) *prefixer {
	return &prefixer{
		writer:     w,
		prefixFunc: func() string { return time.Now().UTC().Format(rfc3339NanoFixed + " ") },
	}
}

// newStringPrefixer wraps an io.Writer, inserting the given string at
// the beginning of each line. The given string should include a
// trailing space for readability.
func newStringPrefixer(w io.Writer, s string) *prefixer {
	return &prefixer{
		writer:     w,
		prefixFunc: func() string { return s },
	}
}

func (tp *prefixer) Write(p []byte) (n int, err error) {
	for len(p) > 0 && err == nil {
		if !tp.unfinished {
			_, err = io.WriteString(tp.writer, tp.prefixFunc())
			if err != nil {
				return
			}
		}
		newline := bytes.IndexRune(p, '\n')
		var nn int
		if newline < 0 {
			tp.unfinished = true
			nn, err = tp.writer.Write(p)
			p = nil
		} else {
			tp.unfinished = false
			nn, err = tp.writer.Write(p[:newline+1])
			p = p[nn:]
		}
		n += nn
	}
	return
}

// logWriter adds log.Logger methods to an io.Writer.
type logWriter struct {
	io.Writer
	*log.Logger
}

func newLogWriter(w io.Writer) *logWriter {
	return &logWriter{
		Writer: w,
		Logger: log.New(w, "", 0),
	}
}

var crunchLogUpdatePeriod = time.Hour / 2
var crunchLogUpdateSize = int64(1 << 25)

// load the rate limit discovery config parameters
func loadLogThrottleParams(clnt IArvadosClient) {
	loadDuration := func(dst *time.Duration, key string) {
		if param, err := clnt.Discovery(key); err != nil {
			return
		} else if d, ok := param.(float64); !ok {
			return
		} else {
			*dst = time.Duration(d) * time.Second
		}
	}
	loadInt64 := func(dst *int64, key string) {
		if param, err := clnt.Discovery(key); err != nil {
			return
		} else if val, ok := param.(float64); !ok {
			return
		} else {
			*dst = int64(val)
		}
	}

	loadInt64(&crunchLogUpdateSize, "crunchLogUpdateSize")
	loadDuration(&crunchLogUpdatePeriod, "crunchLogUpdatePeriod")

}

type filterKeepstoreErrorsOnly struct {
	io.WriteCloser
	buf []byte
}

func (f *filterKeepstoreErrorsOnly) Write(p []byte) (int, error) {
	log.Printf("filterKeepstoreErrorsOnly: write %q", p)
	f.buf = append(f.buf, p...)
	start := 0
	for i := len(f.buf) - len(p); i < len(f.buf); i++ {
		if f.buf[i] == '\n' {
			if f.check(f.buf[start:i]) {
				_, err := f.WriteCloser.Write(f.buf[start : i+1])
				if err != nil {
					return 0, err
				}
			}
			start = i + 1
		}
	}
	if start > 0 {
		copy(f.buf, f.buf[start:])
		f.buf = f.buf[:len(f.buf)-start]
	}
	return len(p), nil
}

func (f *filterKeepstoreErrorsOnly) check(line []byte) bool {
	if len(line) == 0 {
		return false
	}
	if line[0] != '{' {
		return true
	}
	var m map[string]interface{}
	err := json.Unmarshal(line, &m)
	if err != nil {
		return true
	}
	if m["msg"] == "request" {
		return false
	}
	if m["msg"] == "response" {
		if code, _ := m["respStatusCode"].(float64); code >= 200 && code < 300 {
			return false
		}
	}
	return true
}
