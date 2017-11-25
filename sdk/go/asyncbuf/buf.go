// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package asyncbuf

import (
	"bytes"
	"io"
	"sync"
)

// A Buffer is an io.Writer that distributes written data
// asynchronously to multiple concurrent readers.
//
// NewReader() can be called at any time. In all cases, every returned
// io.Reader reads all data written to the Buffer.
//
// Behavior is undefined if Write is called after Close or
// CloseWithError.
type Buffer interface {
	io.WriteCloser

	// NewReader() returns an io.Reader that reads all data
	// written to the Buffer.
	NewReader() io.Reader

	// Close, but return the given error (instead of io.EOF) to
	// all readers when they reach the end of the buffer.
	//
	// CloseWithError(nil) is equivalent to
	// CloseWithError(io.EOF).
	CloseWithError(error) error
}

type buffer struct {
	data *bytes.Buffer
	cond sync.Cond
	err  error // nil if there might be more writes
}

// NewBuffer creates a new Buffer using buf as its initial
// contents. The new Buffer takes ownership of buf, and the caller
// should not use buf after this call.
func NewBuffer(buf []byte) Buffer {
	return &buffer{
		data: bytes.NewBuffer(buf),
		cond: sync.Cond{L: &sync.Mutex{}},
	}
}

func (b *buffer) Write(p []byte) (int, error) {
	defer b.cond.Broadcast()
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	if b.err != nil {
		return 0, b.err
	}
	return b.data.Write(p)
}

func (b *buffer) Close() error {
	return b.CloseWithError(nil)
}

func (b *buffer) CloseWithError(err error) error {
	defer b.cond.Broadcast()
	b.cond.L.Lock()
	defer b.cond.L.Unlock()
	if err == nil {
		b.err = io.EOF
	} else {
		b.err = err
	}
	return nil
}

func (b *buffer) NewReader() io.Reader {
	return &reader{b: b}
}

type reader struct {
	b    *buffer
	read int // # bytes already read
}

func (r *reader) Read(p []byte) (int, error) {
	r.b.cond.L.Lock()
	for {
		switch {
		case r.read < r.b.data.Len():
			buf := r.b.data.Bytes()
			r.b.cond.L.Unlock()
			n := copy(p, buf[r.read:])
			r.read += n
			return n, nil
		case r.b.err != nil || len(p) == 0:
			// r.b.err != nil means we reached EOF.  And
			// even if we're not at EOF, there's no need
			// to block if len(p)==0.
			err := r.b.err
			r.b.cond.L.Unlock()
			return 0, err
		default:
			r.b.cond.Wait()
		}
	}
}
