package keepclient

import (
	"errors"
	"fmt"
	"io"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/manifest"
)

// A Reader implements, io.Reader, io.Seeker, and io.Closer, and has a
// Len() method that returns the total number of bytes available to
// read.
type Reader interface {
	io.Reader
	io.Seeker
	io.Closer
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

// CollectionFileReader returns a Reader that reads content from a single file
// in the collection. The filename must be relative to the root of the
// collection.  A leading prefix of "/" or "./" in the filename is ignored.
func (kc *KeepClient) CollectionFileReader(collection map[string]interface{}, filename string) (Reader, error) {
	mText, ok := collection["manifest_text"].(string)
	if !ok {
		return nil, ErrNoManifest
	}
	m := manifest.Manifest{Text: mText}
	return kc.ManifestFileReader(m, filename)
}

func (kc *KeepClient) ManifestFileReader(m manifest.Manifest, filename string) (Reader, error) {
	f := &file{
		kc: kc,
	}
	err := f.load(m, filename)
	if err != nil {
		return nil, err
	}
	return f, nil
}

type file struct {
	kc       *KeepClient
	segments []*manifest.FileSegment
	size     int64 // total file size
	offset   int64 // current read offset

	// current/latest segment accessed -- might or might not match pos
	seg           *manifest.FileSegment
	segStart      int64 // position of segment relative to file
	segData       []byte
	segNext       []*manifest.FileSegment
	readaheadDone bool
}

// Close implements io.Closer.
func (f *file) Close() error {
	f.kc = nil
	f.segments = nil
	f.segData = nil
	return nil
}

// Read implements io.Reader.
func (f *file) Read(buf []byte) (int, error) {
	if f.seg == nil || f.offset < f.segStart || f.offset >= f.segStart+int64(f.seg.Len) {
		// f.seg does not cover the current read offset
		// (f.pos).  Iterate over f.segments to find the one
		// that does.
		f.seg = nil
		f.segStart = 0
		f.segData = nil
		f.segNext = f.segments
		for len(f.segNext) > 0 {
			seg := f.segNext[0]
			f.segNext = f.segNext[1:]
			segEnd := f.segStart + int64(seg.Len)
			if segEnd > f.offset {
				f.seg = seg
				break
			}
			f.segStart = segEnd
		}
		f.readaheadDone = false
	}
	if f.seg == nil {
		return 0, io.EOF
	}
	if f.segData == nil {
		data, err := f.kc.cache().Get(f.kc, f.seg.Locator)
		if err != nil {
			return 0, err
		}
		if len(data) < f.seg.Offset+f.seg.Len {
			return 0, fmt.Errorf("invalid segment (offset %d len %d) in %d-byte block %s", f.seg.Offset, f.seg.Len, len(data), f.seg.Locator)
		}
		f.segData = data[f.seg.Offset : f.seg.Offset+f.seg.Len]
	}
	// dataOff and dataLen denote a portion of f.segData
	// corresponding to a portion of the file at f.offset.
	dataOff := int(f.offset - f.segStart)
	dataLen := f.seg.Len - dataOff

	if !f.readaheadDone && len(f.segNext) > 0 && f.offset >= 1048576 && dataOff+dataLen > len(f.segData)/16 {
		// If we have already read more than just the first
		// few bytes of this file, and we have already
		// consumed a noticeable portion of this segment, and
		// there's more data for this file in the next segment
		// ... then there's a good chance we are going to need
		// the data for that next segment soon. Start getting
		// it into the cache now.
		go f.kc.cache().Get(f.kc, f.segNext[0].Locator)
		f.readaheadDone = true
	}

	n := len(buf)
	if n > dataLen {
		n = dataLen
	}
	copy(buf[:n], f.segData[dataOff:dataOff+n])
	f.offset += int64(n)
	return n, nil
}

// Seek implements io.Seeker.
func (f *file) Seek(offset int64, whence int) (int64, error) {
	var want int64
	switch whence {
	case io.SeekStart:
		want = offset
	case io.SeekCurrent:
		want = f.offset + offset
	case io.SeekEnd:
		want = f.size + offset
	default:
		return f.offset, fmt.Errorf("invalid whence %d", whence)
	}
	if want < 0 {
		return f.offset, fmt.Errorf("attempted seek to %d", want)
	}
	if want > f.size {
		want = f.size
	}
	f.offset = want
	return f.offset, nil
}

// Len returns the file size in bytes.
func (f *file) Len() uint64 {
	return uint64(f.size)
}

func (f *file) load(m manifest.Manifest, path string) error {
	f.segments = nil
	f.size = 0
	for seg := range m.FileSegmentIterByName(path) {
		f.segments = append(f.segments, seg)
		f.size += int64(seg.Len)
	}
	if f.segments == nil {
		return os.ErrNotExist
	}
	return nil
}
