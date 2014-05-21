/* AsyncStream pulls data in from a io.Reader source (such as a file or network
socket) and fans out to any number of StreamReader sinks.

Unlike io.TeeReader() or io.MultiWriter(), new StreamReaders can be created at
any point in the lifetime of the AsyncStream, and each StreamReader will read
the contents of the buffer up to the "frontier" of the buffer, at which point
the StreamReader blocks until new data is read from the source.

This is useful for minimizing readthrough latency as sinks can read and act on
data from the source without waiting for the source to be completely buffered.
It is also useful as a cache in situations where re-reading the original source
potentially is costly, since the buffer retains a copy of the source data.

Usage:

Begin reading into a buffer with maximum size 'buffersize' from 'source':
  stream := AsyncStreamFromReader(buffersize, source)

To create a new reader (this can be called multiple times, each reader starts
at the beginning of the buffer):
  reader := tr.MakeStreamReader()

Make sure to close the reader when you're done with it.
  reader.Close()

When you're done with the stream:
  stream.Close()

Alternately, if you already have a filled buffer and just want to read out from it:
  stream := AsyncStreamFromSlice(buf)

  r := tr.MakeStreamReader()

*/

package streamer

import (
	"io"
)

type AsyncStream struct {
	buffer            []byte
	requests          chan sliceRequest
	add_reader        chan bool
	subtract_reader   chan bool
	wait_zero_readers chan bool
}

// Reads from the buffer managed by the Transfer()
type StreamReader struct {
	offset    int
	stream    *AsyncStream
	responses chan sliceResult
}

func AsyncStreamFromReader(buffersize int, source io.Reader) *AsyncStream {
	t := &AsyncStream{make([]byte, buffersize), make(chan sliceRequest), make(chan bool), make(chan bool), make(chan bool)}

	go t.transfer(source)
	go t.readersMonitor()

	return t
}

func AsyncStreamFromSlice(buf []byte) *AsyncStream {
	t := &AsyncStream{buf, make(chan sliceRequest), make(chan bool), make(chan bool), make(chan bool)}

	go t.transfer(nil)
	go t.readersMonitor()

	return t
}

func (this *AsyncStream) MakeStreamReader() *StreamReader {
	this.add_reader <- true
	return &StreamReader{0, this, make(chan sliceResult)}
}

// Reads from the buffer managed by the Transfer()
func (this *StreamReader) Read(p []byte) (n int, err error) {
	this.stream.requests <- sliceRequest{this.offset, len(p), this.responses}
	rr, valid := <-this.responses
	if valid {
		this.offset += len(rr.slice)
		return copy(p, rr.slice), rr.err
	} else {
		return 0, io.ErrUnexpectedEOF
	}
}

func (this *StreamReader) WriteTo(dest io.Writer) (written int64, err error) {
	// Record starting offset in order to correctly report the number of bytes sent
	starting_offset := this.offset
	for {
		this.stream.requests <- sliceRequest{this.offset, 32 * 1024, this.responses}
		rr, valid := <-this.responses
		if valid {
			this.offset += len(rr.slice)
			if rr.err != nil {
				if rr.err == io.EOF {
					// EOF is not an error.
					return int64(this.offset - starting_offset), nil
				} else {
					return int64(this.offset - starting_offset), rr.err
				}
			} else {
				dest.Write(rr.slice)
			}
		} else {
			return int64(this.offset), io.ErrUnexpectedEOF
		}
	}
}

// Close the responses channel
func (this *StreamReader) Close() error {
	this.stream.subtract_reader <- true
	close(this.responses)
	this.stream = nil
	return nil
}

func (this *AsyncStream) Close() {
	this.wait_zero_readers <- true
	close(this.requests)
	close(this.add_reader)
	close(this.subtract_reader)
	close(this.wait_zero_readers)
}
