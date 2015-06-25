package keepclient

import (
	"errors"
	"io"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/manifest"
)

// ErrNoManifest indicates the given collection has no manifest
// information (e.g., manifest_text was excluded by a "select"
// parameter when retrieving the collection record).
var ErrNoManifest = errors.New("Collection has no manifest")

// CollectionFileReader returns an io.Reader that reads file content
// from a collection. The filename must be given relative to the root
// of the collection, without a leading "./".
func (kc *KeepClient) CollectionFileReader(collection map[string]interface{}, filename string) (*cfReader, error) {
	mText, ok := collection["manifest_text"].(string)
	if !ok {
		return nil, ErrNoManifest
	}
	m := manifest.Manifest{Text: mText}
	rdrChan := make(chan *cfReader)
	go func() {
		// q is a queue of FileSegments that we have received but
		// haven't yet been able to send to toGet.
		var q []*manifest.FileSegment
		var r *cfReader
		for seg := range m.FileSegmentIterByName(filename) {
			if r == nil {
				// We've just discovered that the
				// requested filename does appear in
				// the manifest, so we can return a
				// real reader (not nil) from
				// CollectionFileReader().
				r = newCFReader(kc)
				rdrChan <- r
			}
			q = append(q, seg)
			r.totalSize += uint64(seg.Len)
			// Send toGet whatever it's ready to receive.
			Q: for len(q) > 0 {
				select {
				case r.toGet <- q[0]:
					q = q[1:]
				default:
					break Q
				}
			}
		}
		if r == nil {
			// File not found
			rdrChan <- nil
			return
		}
		close(r.countDone)
		for _, seg := range q {
			r.toGet <- seg
		}
		close(r.toGet)
	}()
	// Before returning a reader, wait until we know whether the
	// file exists here:
	r := <-rdrChan
	if r == nil {
		return nil, os.ErrNotExist
	}
	return r, nil
}

type cfReader struct {
	keepClient *KeepClient
	// doGet() reads FileSegments from toGet, gets the data from
	// Keep, and sends byte slices to toRead to be consumed by
	// Read().
	toGet        chan *manifest.FileSegment
	toRead       chan []byte
	// bytes ready to send next time someone calls Read()
	buf          []byte
	// Total size of the file being read. Not safe to read this
	// until countDone is closed.
	totalSize    uint64
	countDone    chan struct{}
	// First error encountered.
	err          error
}

func (r *cfReader) Read(outbuf []byte) (n int, err error) {
	if r.err != nil {
		return 0, r.err
	}
	for r.buf == nil || len(r.buf) == 0 {
		var ok bool
		r.buf, ok = <-r.toRead
		if r.err != nil {
			return 0, r.err
		} else if !ok {
			return 0, io.EOF
		}
	}
	if len(r.buf) > len(outbuf) {
		n = len(outbuf)
	} else {
		n = len(r.buf)
	}
	copy(outbuf[:n], r.buf[:n])
	r.buf = r.buf[n:]
	return
}

func (r *cfReader) Close() error {
	_, _ = <-r.countDone
	for _ = range r.toGet {
	}
	for _ = range r.toRead {
	}
	return r.err
}

func (r *cfReader) Len() uint64 {
	// Wait for all segments to be counted
	_, _ = <-r.countDone
	return r.totalSize
}

func (r *cfReader) doGet() {
	defer close(r.toRead)
	for fs := range r.toGet {
		rdr, _, _, err := r.keepClient.Get(fs.Locator)
		if err != nil {
			r.err = err
			return
		}
		var buf = make([]byte, fs.Offset+fs.Len)
		_, err = io.ReadFull(rdr, buf)
		if err != nil {
			r.err = err
			return
		}
		for bOff, bLen := fs.Offset, 1<<20; bOff <= fs.Offset+fs.Len && bLen > 0; bOff += bLen {
			if bOff+bLen > fs.Offset+fs.Len {
				bLen = fs.Offset + fs.Len - bOff
			}
			r.toRead <- buf[bOff : bOff+bLen]
		}
	}
}

func newCFReader(kc *KeepClient) (r *cfReader) {
	r = new(cfReader)
	r.keepClient = kc
	r.toGet = make(chan *manifest.FileSegment, 2)
	r.toRead = make(chan []byte)
	r.countDone = make(chan struct{})
	go r.doGet()
	return
}
