// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

// streamWriterAt translates random-access writes to sequential
// writes. The caller is expected to use an arbitrary sequence of
// non-overlapping WriteAt calls covering all positions between 0 and
// N, for any N < len(buf), then call Close.
//
// streamWriterAt writes the data to the provided io.Writer in
// sequential order.
//
// Close returns when all data has been written through.
type streamWriterAt struct {
	writer     io.Writer
	buf        []byte
	partsize   int         // size of each part written through to writer
	endpos     int         // portion of buf actually used, judging by WriteAt calls so far
	partfilled []int       // number of bytes written to each part so far
	partready  chan []byte // parts of buf fully written / waiting for writer goroutine
	partnext   int         // index of next part we will send to partready when it's ready
	wroteAt    int         // bytes we copied to buf in WriteAt
	wrote      int         // bytes successfully written through to writer
	errWrite   chan error  // final outcome of writer goroutine
	closed     bool        // streamWriterAt has been closed
	mtx        sync.Mutex  // guard internal fields during concurrent calls to WriteAt and Close
}

// newStreamWriterAt creates a new streamWriterAt.
func newStreamWriterAt(w io.Writer, partsize int, buf []byte) *streamWriterAt {
	if partsize == 0 {
		partsize = 65536
	}
	nparts := (len(buf) + partsize - 1) / partsize
	swa := &streamWriterAt{
		writer:     w,
		partsize:   partsize,
		buf:        buf,
		partfilled: make([]int, nparts),
		partready:  make(chan []byte, nparts),
		errWrite:   make(chan error, 1),
	}
	go swa.writeToWriter()
	return swa
}

// Wrote returns the number of bytes written through to the
// io.Writer.
//
// Wrote must not be called until after Close.
func (swa *streamWriterAt) Wrote() int {
	return swa.wrote
}

// Wrote returns the number of bytes passed to WriteAt, regardless of
// whether they were written through to the io.Writer.
func (swa *streamWriterAt) WroteAt() int {
	swa.mtx.Lock()
	defer swa.mtx.Unlock()
	return swa.wroteAt
}

func (swa *streamWriterAt) writeToWriter() {
	defer close(swa.errWrite)
	for p := range swa.partready {
		n, err := swa.writer.Write(p)
		if err != nil {
			swa.errWrite <- err
			return
		}
		swa.wrote += n
	}
}

// WriteAt implements io.WriterAt.
func (swa *streamWriterAt) WriteAt(p []byte, offset int64) (int, error) {
	pos := int(offset)
	n := 0
	if pos <= len(swa.buf) {
		n = copy(swa.buf[pos:], p)
	}
	if n < len(p) {
		return n, fmt.Errorf("write beyond end of buffer: offset %d len %d buf %d", offset, len(p), len(swa.buf))
	}
	endpos := pos + n

	swa.mtx.Lock()
	defer swa.mtx.Unlock()
	swa.wroteAt += len(p)
	if swa.endpos < endpos {
		swa.endpos = endpos
	}
	if swa.closed {
		return 0, errors.New("invalid use of closed streamWriterAt")
	}
	// Track the number of bytes that landed in each of our
	// (output) parts.
	for i := pos; i < endpos; {
		j := i + swa.partsize - (i % swa.partsize)
		if j > endpos {
			j = endpos
		}
		pf := swa.partfilled[i/swa.partsize]
		pf += j - i
		if pf > swa.partsize {
			return 0, errors.New("streamWriterAt: overlapping WriteAt calls")
		}
		swa.partfilled[i/swa.partsize] = pf
		i = j
	}
	// Flush filled parts to partready.
	for swa.partnext < len(swa.partfilled) && swa.partfilled[swa.partnext] == swa.partsize {
		offset := swa.partnext * swa.partsize
		swa.partready <- swa.buf[offset : offset+swa.partsize]
		swa.partnext++
	}
	return len(p), nil
}

// Close flushes all buffered data through to the io.Writer.
func (swa *streamWriterAt) Close() error {
	swa.mtx.Lock()
	defer swa.mtx.Unlock()
	if swa.closed {
		return errors.New("invalid use of closed streamWriterAt")
	}
	swa.closed = true
	// Flush last part if needed. If the input doesn't end on a
	// part boundary, the last part never appears "filled" when we
	// check in WriteAt.  But here, we know endpos is the end of
	// the stream, so we can check whether the last part is ready.
	if offset := swa.partnext * swa.partsize; offset < swa.endpos && offset+swa.partfilled[swa.partnext] == swa.endpos {
		swa.partready <- swa.buf[offset:swa.endpos]
		swa.partnext++
	}
	close(swa.partready)
	err := <-swa.errWrite
	if err != nil {
		return err
	}
	if swa.wrote != swa.wroteAt {
		return fmt.Errorf("streamWriterAt: detected hole in input: wrote %d but flushed %d", swa.wroteAt, swa.wrote)
	}
	return nil
}
