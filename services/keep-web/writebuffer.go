// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepweb

import (
	"errors"
	"io"
	"net/http"
	"sync/atomic"
)

type writeBuffer struct {
	out       io.Writer
	buf       []byte
	writesize int
	wpos      atomic.Int64  // index in buf where writer (Write()) will write to next
	wsignal   chan struct{} // receives a value after wpos or closed changes
	rpos      atomic.Int64  // index in buf where reader (flush()) will read from next
	rsignal   chan struct{} // receives a value after rpos or err changes
	err       error         // error encountered by flush
	closed    atomic.Bool
	flushed   chan struct{} // closes when flush() is finished
}

func newWriteBuffer(w io.Writer, size int) *writeBuffer {
	wb := &writeBuffer{
		out:       w,
		buf:       make([]byte, size),
		writesize: (size + 63) / 64,
		wsignal:   make(chan struct{}, 1),
		rsignal:   make(chan struct{}, 1),
		flushed:   make(chan struct{}),
	}
	go wb.flush()
	return wb
}

func (wb *writeBuffer) Close() error {
	if wb.closed.Load() {
		return errors.New("writeBuffer: already closed")
	}
	wb.closed.Store(true)
	// wake up flush()
	select {
	case wb.wsignal <- struct{}{}:
	default:
	}
	// wait for flush() to finish
	<-wb.flushed
	return wb.err
}

func (wb *writeBuffer) Write(p []byte) (int, error) {
	if len(wb.buf) < 2 {
		// Our buffer logic doesn't work with size<2, and such
		// a tiny buffer has no purpose anyway, so just write
		// through unbuffered.
		return wb.out.Write(p)
	}
	todo := p
	wpos := int(wb.wpos.Load())
	rpos := int(wb.rpos.Load())
	for len(todo) > 0 {
		for rpos == (wpos+1)%len(wb.buf) {
			select {
			case <-wb.flushed:
				if wb.err == nil {
					return 0, errors.New("Write called on closed writeBuffer")
				}
				return 0, wb.err
			case <-wb.rsignal:
				rpos = int(wb.rpos.Load())
			}
		}
		var avail []byte
		if rpos == 0 {
			avail = wb.buf[wpos : len(wb.buf)-1]
		} else if wpos >= rpos {
			avail = wb.buf[wpos:]
		} else {
			avail = wb.buf[wpos : rpos-1]
		}
		n := copy(avail, todo)
		wpos = (wpos + n) % len(wb.buf)
		wb.wpos.Store(int64(wpos))
		// wake up flush()
		select {
		case wb.wsignal <- struct{}{}:
		default:
		}
		todo = todo[n:]
	}
	return len(p), nil
}

func (wb *writeBuffer) flush() {
	defer close(wb.flushed)
	rpos := 0
	wpos := 0
	closed := false
	for {
		for rpos == wpos {
			if closed {
				return
			}
			<-wb.wsignal
			closed = wb.closed.Load()
			wpos = int(wb.wpos.Load())
		}
		var ready []byte
		if rpos < wpos {
			ready = wb.buf[rpos:wpos]
		} else {
			ready = wb.buf[rpos:]
		}
		if len(ready) > wb.writesize {
			ready = ready[:wb.writesize]
		}
		_, wb.err = wb.out.Write(ready)
		if wb.err != nil {
			return
		}
		rpos = (rpos + len(ready)) % len(wb.buf)
		wb.rpos.Store(int64(rpos))
		select {
		case wb.rsignal <- struct{}{}:
		default:
		}
	}
}

type responseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (rwc responseWriter) Write(p []byte) (int, error) {
	return rwc.Writer.Write(p)
}
