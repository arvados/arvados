// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package keepclient

import (
	"errors"
	"fmt"
	"hash"
	"io"
)

var BadChecksum = errors.New("Reader failed checksum")

// HashCheckingReader is an io.ReadCloser that checks the contents
// read from the underlying io.Reader against the provided hash.
type HashCheckingReader struct {
	// The underlying data source
	io.Reader

	// The hash function to use
	hash.Hash

	// The hash value to check against.  Must be a hex-encoded lowercase string.
	Check string
}

// Reads from the underlying reader, update the hashing function, and
// pass the results through. Returns BadChecksum (instead of EOF) on
// the last read if the checksum doesn't match.
func (hcr HashCheckingReader) Read(p []byte) (n int, err error) {
	n, err = hcr.Reader.Read(p)
	if n > 0 {
		hcr.Hash.Write(p[:n])
	}
	if err == io.EOF {
		sum := hcr.Hash.Sum(nil)
		if fmt.Sprintf("%x", sum) != hcr.Check {
			err = BadChecksum
		}
	}
	return n, err
}

// WriteTo writes the entire contents of hcr.Reader to dest. Returns
// BadChecksum if writing is successful but the checksum doesn't
// match.
func (hcr HashCheckingReader) WriteTo(dest io.Writer) (written int64, err error) {
	written, err = io.Copy(io.MultiWriter(dest, hcr.Hash), hcr.Reader)
	if err != nil {
		return written, err
	}

	sum := hcr.Hash.Sum(nil)
	if fmt.Sprintf("%x", sum) != hcr.Check {
		return written, BadChecksum
	}

	return written, nil
}

// Close reads all remaining data from the underlying Reader and
// returns BadChecksum if the checksum doesn't match. It also closes
// the underlying Reader if it implements io.ReadCloser.
func (hcr HashCheckingReader) Close() (err error) {
	_, err = io.Copy(hcr.Hash, hcr.Reader)

	if closer, ok := hcr.Reader.(io.Closer); ok {
		closeErr := closer.Close()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		return err
	}
	if fmt.Sprintf("%x", hcr.Hash.Sum(nil)) != hcr.Check {
		return BadChecksum
	}
	return nil
}
