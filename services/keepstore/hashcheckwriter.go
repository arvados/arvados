// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"fmt"
	"hash"
	"io"
)

type hashCheckWriter struct {
	writer       io.Writer
	hash         hash.Hash
	expectSize   int64
	expectDigest string

	offset int64
}

// newHashCheckWriter returns a writer that writes through to w, but
// stops short if the written content reaches expectSize bytes and
// does not match expectDigest according to the given hash
// function.
//
// It returns a write error if more than expectSize bytes are written.
//
// Thus, in case of a hash mismatch, fewer than expectSize will be
// written through.
func newHashCheckWriter(writer io.Writer, hash hash.Hash, expectSize int64, expectDigest string) io.Writer {
	return &hashCheckWriter{
		writer:       writer,
		hash:         hash,
		expectSize:   expectSize,
		expectDigest: expectDigest,
	}
}

func (hcw *hashCheckWriter) Write(p []byte) (int, error) {
	if todo := hcw.expectSize - hcw.offset - int64(len(p)); todo < 0 {
		// Writing beyond expected size returns a checksum
		// error without even checking the hash.
		return 0, errChecksum
	} else if todo > 0 {
		// This isn't the last write, so we pass it through.
		_, err := hcw.hash.Write(p)
		if err != nil {
			return 0, err
		}
		n, err := hcw.writer.Write(p)
		hcw.offset += int64(n)
		return n, err
	} else {
		// This is the last write, so we check the hash before
		// writing through.
		_, err := hcw.hash.Write(p)
		if err != nil {
			return 0, err
		}
		if digest := fmt.Sprintf("%x", hcw.hash.Sum(nil)); digest != hcw.expectDigest {
			return 0, errChecksum
		}
		// Ensure subsequent write will fail
		hcw.offset = hcw.expectSize + 1
		return hcw.writer.Write(p)
	}
}
