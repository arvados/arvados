package keepclient

import (
	"errors"
	"io"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/manifest"
)

// ReadCloserWithLen extends io.ReadCloser with a Len() method that
// returns the total number of bytes available to read.
type ReadCloserWithLen interface {
	io.ReadCloser
	Len() uint64
}

const (
	// After reading a data block from Keep, cfReader slices it up
	// and sends the slices to a buffered channel to be consumed
	// by the caller via Read().
	//
	// dataSliceSize is the maximum size of the slices, and
	// therefore the maximum number of bytes that will be returned
	// by a single call to Read().
	dataSliceSize = 1 << 20
)

// ErrNoManifest indicates the given collection has no manifest
// information (e.g., manifest_text was excluded by a "select"
// parameter when retrieving the collection record).
var ErrNoManifest = errors.New("Collection has no manifest")

// CollectionFileReader returns a ReadCloserWithLen that reads file
// content from a collection. The filename must be given relative to
// the root of the collection, without a leading "./".
func (kc *KeepClient) CollectionFileReader(collection map[string]interface{}, filename string) (ReadCloserWithLen, error) {
	mText, ok := collection["manifest_text"].(string)
	if !ok {
		return nil, ErrNoManifest
	}
	m := manifest.Manifest{Text: mText}
	return kc.ManifestFileReader(m, filename)
}

func (kc *KeepClient) ManifestFileReader(m manifest.Manifest, filename string) (ReadCloserWithLen, error) {
	rdrChan := make(chan *cfReader)
	go kc.queueSegmentsToGet(m, filename, rdrChan)
	r, ok := <-rdrChan
	if !ok {
		return nil, os.ErrNotExist
	}
	return r, nil
}

// Send segments for the specified file to r.toGet. Send a *cfReader
// to rdrChan if the specified file is found (even if it's empty).
// Then, close rdrChan.
func (kc *KeepClient) queueSegmentsToGet(m manifest.Manifest, filename string, rdrChan chan *cfReader) {
	defer close(rdrChan)

	// q is a queue of FileSegments that we have received but
	// haven't yet been able to send to toGet.
	var q []*manifest.FileSegment
	var r *cfReader
	for seg := range m.FileSegmentIterByName(filename) {
		if r == nil {
			// We've just discovered that the requested
			// filename does appear in the manifest, so we
			// can return a real reader (not nil) from
			// CollectionFileReader().
			r = newCFReader(kc)
			rdrChan <- r
		}
		q = append(q, seg)
		r.totalSize += uint64(seg.Len)
		// Send toGet as many segments as we can until it
		// blocks.
	Q:
		for len(q) > 0 {
			select {
			case r.toGet <- q[0]:
				q = q[1:]
			default:
				break Q
			}
		}
	}
	if r == nil {
		// File not found.
		return
	}
	close(r.countDone)
	for _, seg := range q {
		r.toGet <- seg
	}
	close(r.toGet)
}

type cfReader struct {
	keepClient *KeepClient

	// doGet() reads FileSegments from toGet, gets the data from
	// Keep, and sends byte slices to toRead to be consumed by
	// Read().
	toGet chan *manifest.FileSegment

	// toRead is a buffered channel, sized to fit one full Keep
	// block. This lets us verify checksums without having a
	// store-and-forward delay between blocks: by the time the
	// caller starts receiving data from block N, cfReader is
	// starting to fetch block N+1. A larger buffer would be
	// useful for a caller whose read speed varies a lot.
	toRead chan []byte

	// bytes ready to send next time someone calls Read()
	buf []byte

	// Total size of the file being read. Not safe to read this
	// until countDone is closed.
	totalSize uint64
	countDone chan struct{}

	// First error encountered.
	err error

	// errNotNil is closed IFF err contains a non-nil error.
	// Receiving from it will block until an error occurs.
	errNotNil chan struct{}

	// rdrClosed is closed IFF the reader's Close() method has
	// been called. Any goroutines associated with the reader will
	// stop and free up resources when they notice this channel is
	// closed.
	rdrClosed chan struct{}
}

func (r *cfReader) Read(outbuf []byte) (int, error) {
	if r.Error() != nil {
		// Short circuit: the caller might as well find out
		// now that we hit an error, even if there's buffered
		// data we could return.
		return 0, r.Error()
	}
	for len(r.buf) == 0 {
		// Private buffer was emptied out by the last Read()
		// (or this is the first Read() and r.buf is nil).
		// Read from r.toRead until we get a non-empty slice
		// or hit an error.
		var ok bool
		r.buf, ok = <-r.toRead
		if r.Error() != nil {
			// Error encountered while waiting for bytes
			return 0, r.Error()
		} else if !ok {
			// No more bytes to read, no error encountered
			return 0, io.EOF
		}
	}
	// Copy as much as possible from our private buffer to the
	// caller's buffer
	n := len(r.buf)
	if len(r.buf) > len(outbuf) {
		n = len(outbuf)
	}
	copy(outbuf[:n], r.buf[:n])

	// Next call to Read() will continue where we left off
	r.buf = r.buf[n:]

	return n, nil
}

// Close releases resources. It returns a non-nil error if an error
// was encountered by the reader.
func (r *cfReader) Close() error {
	close(r.rdrClosed)
	return r.Error()
}

// Error returns an error if one has been encountered, otherwise
// nil. It is safe to call from any goroutine.
func (r *cfReader) Error() error {
	select {
	case <-r.errNotNil:
		return r.err
	default:
		return nil
	}
}

// Len returns the total number of bytes in the file being read. If
// necessary, it waits for manifest parsing to finish.
func (r *cfReader) Len() uint64 {
	// Wait for all segments to be counted
	<-r.countDone
	return r.totalSize
}

func (r *cfReader) doGet() {
	defer close(r.toRead)
GET:
	for fs := range r.toGet {
		rdr, _, _, err := r.keepClient.Get(fs.Locator)
		if err != nil {
			r.err = err
			close(r.errNotNil)
			return
		}
		var buf = make([]byte, fs.Offset+fs.Len)
		_, err = io.ReadFull(rdr, buf)
		if err != nil {
			r.err = err
			close(r.errNotNil)
			return
		}
		for bOff, bLen := fs.Offset, dataSliceSize; bOff < fs.Offset+fs.Len && bLen > 0; bOff += bLen {
			if bOff+bLen > fs.Offset+fs.Len {
				bLen = fs.Offset + fs.Len - bOff
			}
			select {
			case r.toRead <- buf[bOff : bOff+bLen]:
			case <-r.rdrClosed:
				// Reader is closed: no point sending
				// anything more to toRead.
				break GET
			}
		}
		// It is possible that r.rdrClosed is closed but we
		// never noticed because r.toRead was also ready in
		// every select{} above. Here we check before wasting
		// a keepclient.Get() call.
		select {
		case <-r.rdrClosed:
			break GET
		default:
		}
	}
	// In case we exited the above loop early: before returning,
	// drain the toGet channel so its sender doesn't sit around
	// blocking forever.
	for _ = range r.toGet {
	}
}

func newCFReader(kc *KeepClient) (r *cfReader) {
	r = new(cfReader)
	r.keepClient = kc
	r.rdrClosed = make(chan struct{})
	r.errNotNil = make(chan struct{})
	r.toGet = make(chan *manifest.FileSegment, 2)
	r.toRead = make(chan []byte, (BLOCKSIZE+dataSliceSize-1)/dataSliceSize)
	r.countDone = make(chan struct{})
	go r.doGet()
	return
}
