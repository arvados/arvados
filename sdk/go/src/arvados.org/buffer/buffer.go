package buffer

import (
	"io"
	"log"
)

type ReaderSlice struct {
	slice        []byte
	reader_error error
}

// Read repeatedly from the reader into the specified buffer, and report each
// read to channel 'c'.  Completes when Reader 'r' reports on the error channel
// and closes channel 'c'.
func ReadIntoBuffer(buffer []byte, r io.Reader, slices chan<- ReaderSlice) {
	defer close(slices)

	// Initially use entire buffer as scratch space
	ptr := buffer[:]
	for {
		var n int
		var err error
		if len(ptr) > 0 {
			// Read into the scratch space
			n, err = r.Read(ptr)
		} else {
			// Ran out of scratch space, try reading one more byte
			var b [1]byte
			n, err = r.Read(b[:])

			if n > 0 {
				// Reader has more data but we have nowhere to
				// put it, so we're stuffed
				slices <- ReaderSlice{nil, io.ErrShortBuffer}
			} else {
				// Return some other error (hopefully EOF)
				slices <- ReaderSlice{nil, err}
			}
			return
		}

		// End on error (includes EOF)
		if err != nil {
			slices <- ReaderSlice{nil, err}
			return
		}

		if n > 0 {
			// Make a slice with the contents of the read
			slices <- ReaderSlice{ptr[:n], nil}

			// Adjust the scratch space slice
			ptr = ptr[n:]
		}
	}
}

// A read request to the Transfer() function
type ReadRequest struct {
	offset  int
	maxsize int
	result  chan<- ReadResult
}

// A read result from the Transfer() function
type ReadResult struct {
	slice []byte
	err   error
}

// Reads from the buffer managed by the Transfer()
type BufferReader struct {
	offset    *int
	requests  chan<- ReadRequest
	responses chan ReadResult
}

func MakeBufferReader(requests chan<- ReadRequest) BufferReader {
	return BufferReader{new(int), requests, make(chan ReadResult)}
}

// Reads from the buffer managed by the Transfer()
func (this BufferReader) Read(p []byte) (n int, err error) {
	this.requests <- ReadRequest{*this.offset, len(p), this.responses}
	rr, valid := <-this.responses
	if valid {
		*this.offset += len(rr.slice)
		return copy(p, rr.slice), rr.err
	} else {
		return 0, io.ErrUnexpectedEOF
	}
}

func (this BufferReader) WriteTo(dest io.Writer) (written int64, err error) {
	// Record starting offset in order to correctly report the number of bytes sent
	starting_offset := *this.offset
	for {
		this.requests <- ReadRequest{*this.offset, 32 * 1024, this.responses}
		rr, valid := <-this.responses
		if valid {
			log.Printf("WriteTo slice %v %d %v", *this.offset, len(rr.slice), rr.err)
			*this.offset += len(rr.slice)
			if rr.err != nil {
				if rr.err == io.EOF {
					// EOF is not an error.
					return int64(*this.offset - starting_offset), nil
				} else {
					return int64(*this.offset - starting_offset), rr.err
				}
			} else {
				dest.Write(rr.slice)
			}
		} else {
			return int64(*this.offset), io.ErrUnexpectedEOF
		}
	}
}

// Close the responses channel
func (this BufferReader) Close() error {
	close(this.responses)
	return nil
}

// Handle a read request.  Returns true if a response was sent, and false if
// the request should be queued.
func HandleReadRequest(req ReadRequest, body []byte, complete bool) bool {
	log.Printf("HandleReadRequest %d %d %d", req.offset, req.maxsize, len(body))
	if req.offset < len(body) {
		var end int
		if req.offset+req.maxsize < len(body) {
			end = req.offset + req.maxsize
		} else {
			end = len(body)
		}
		req.result <- ReadResult{body[req.offset:end], nil}
		return true
	} else if complete && req.offset >= len(body) {
		req.result <- ReadResult{nil, io.EOF}
		return true
	} else {
		return false
	}
}

// If 'source_reader' is not nil, reads data from 'source_reader' and stores it
// in the provided buffer.  Otherwise, use the contents of 'buffer' as is.
// Accepts read requests on the buffer on the 'requests' channel.  Completes
// when 'requests' channel is closed.
func Transfer(source_buffer []byte, source_reader io.Reader, requests <-chan ReadRequest, reader_error chan error) {
	// currently buffered data
	var body []byte

	// for receiving slices from ReadIntoBuffer
	var slices chan ReaderSlice = nil

	// indicates whether the buffered data is complete
	var complete bool = false

	if source_reader != nil {
		// 'body' is the buffer slice representing the body content read so far
		body = source_buffer[:0]

		// used to communicate slices of the buffer as they are
		// ReadIntoBuffer will close 'slices' when it is done with it
		slices = make(chan ReaderSlice)

		// Spin it off
		go ReadIntoBuffer(source_buffer, source_reader, slices)
	} else {
		// use the whole buffer
		body = source_buffer[:]

		// buffer is complete
		complete = true
	}

	pending_requests := make([]ReadRequest, 0)

	for {
		select {
		case req, valid := <-requests:
			// Handle a buffer read request
			if valid {
				if !HandleReadRequest(req, body, complete) {
					pending_requests = append(pending_requests, req)
				}
			} else {
				// closed 'requests' channel indicates we're done
				return
			}

		case bk, valid := <-slices:
			// Got a new slice from the reader
			if valid {
				if bk.reader_error != nil {
					reader_error <- bk.reader_error
					if bk.reader_error == io.EOF {
						// EOF indicates the reader is done
						// sending, so our buffer is complete.
						complete = true
					} else {
						// some other reader error
						return
					}
				}

				if bk.slice != nil {
					// adjust body bounds now that another slice has been read
					body = source_buffer[0 : len(body)+len(bk.slice)]
				}

				// handle pending reads
				n := 0
				for n < len(pending_requests) {
					if HandleReadRequest(pending_requests[n], body, complete) {

						// move the element from the
						// back of the slice to
						// position 'n', then shorten
						// the slice by one element
						pending_requests[n] = pending_requests[len(pending_requests)-1]
						pending_requests = pending_requests[0 : len(pending_requests)-1]
					} else {

						// Request wasn't handled, so keep it in the request slice
						n += 1
					}
				}
			} else {
				if complete {
					// no more reads
					slices = nil
				} else {
					// reader channel closed without signaling EOF
					reader_error <- io.ErrUnexpectedEOF
					return
				}
			}
		}
	}
}
