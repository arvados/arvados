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
func (this HashCheckingReader) Read(p []byte) (n int, err error) {
	n, err = this.Reader.Read(p)
	if n > 0 {
		this.Hash.Write(p[:n])
	}
	if err == io.EOF {
		sum := this.Hash.Sum(make([]byte, 0, this.Hash.Size()))
		if fmt.Sprintf("%x", sum) != this.Check {
			err = BadChecksum
		}
	}
	return n, err
}

// WriteTo writes the entire contents of this.Reader to dest.  Returns
// BadChecksum if the checksum doesn't match.
func (this HashCheckingReader) WriteTo(dest io.Writer) (written int64, err error) {
	if writeto, ok := this.Reader.(io.WriterTo); ok {
		written, err = writeto.WriteTo(io.MultiWriter(dest, this.Hash))
	} else {
		written, err = io.Copy(io.MultiWriter(dest, this.Hash), this.Reader)
	}

	sum := this.Hash.Sum(make([]byte, 0, this.Hash.Size()))

	if fmt.Sprintf("%x", sum) != this.Check {
		err = BadChecksum
	}

	return written, err
}

// Close reads all remaining data from the underlying Reader and
// returns BadChecksum if the checksum doesn't match. It also closes
// the underlying Reader if it implements io.ReadCloser.
func (this HashCheckingReader) Close() (err error) {
	_, err = io.Copy(this.Hash, this.Reader)

	if closer, ok := this.Reader.(io.ReadCloser); ok {
		err = closer.Close()
	}

	sum := this.Hash.Sum(make([]byte, 0, this.Hash.Size()))
	if fmt.Sprintf("%x", sum) != this.Check {
		err = BadChecksum
	}

	return err
}
