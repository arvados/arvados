// Lightweight implementation of io.ReadCloser that checks the contents read
// from the underlying io.Reader a against checksum hash.  To avoid reading the
// entire contents into a buffer up front, the hash is updated with each read,
// and the actual checksum is not checked until the underlying reader returns
// EOF.
package keepclient

import (
	"errors"
	"fmt"
	"hash"
	"io"
)

var BadChecksum = errors.New("Reader failed checksum")

type HashCheckingReader struct {
	// The underlying data source
	io.Reader

	// The hashing function to use
	hash.Hash

	// The hash value to check against.  Must be a hex-encoded lowercase string.
	Check string
}

// Read from the underlying reader, update the hashing function, and pass the
// results through.  Will return BadChecksum on the last read instead of EOF if
// the checksum doesn't match.
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

// Write entire contents of this.Reader to 'dest'.  Returns BadChecksum if the
// data written to 'dest' doesn't match the hash code of this.Check.
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

// Close() the underlying Reader if it is castable to io.ReadCloser.  This will
// drain the underlying reader of any remaining data and check the checksum.
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
