package streamer

import (
	"io"
	"log"
)

// A slice passed from readIntoBuffer() to transfer()
type readerSlice struct {
	slice        []byte
	reader_error error
}

// A read request to the Transfer() function
type readRequest struct {
	offset  int
	maxsize int
	result  chan<- readResult
}

// A read result from the Transfer() function
type readResult struct {
	slice []byte
	err   error
}

// Supports writing into a buffer
type bufferWriter struct {
	buf []byte
	ptr int
}

// Copy p into this.buf, increment pointer and return number of bytes read.
func (this *bufferWriter) Write(p []byte) (n int, err error) {
	n = copy(this.buf[this.ptr:], p)
	this.ptr += n
	return n, nil
}

// Read repeatedly from the reader and write sequentially into the specified
// buffer, and report each read to channel 'c'.  Completes when Reader 'r'
// reports on the error channel and closes channel 'c'.
func readIntoBuffer(buffer []byte, r io.Reader, slices chan<- readerSlice) {
	defer close(slices)

	if writeto, ok := r.(io.WriterTo); ok {
		n, err := writeto.WriteTo(&bufferWriter{buffer, 0})
		if err != nil {
			slices <- readerSlice{nil, err}
		} else {
			slices <- readerSlice{buffer[:n], nil}
			slices <- readerSlice{nil, io.EOF}
		}
		return
	} else {
		// Initially entire buffer is available
		ptr := buffer[:]
		for {
			var n int
			var err error
			if len(ptr) > 0 {
				const readblock = 64 * 1024
				// Read 64KiB into the next part of the buffer
				if len(ptr) > readblock {
					n, err = r.Read(ptr[:readblock])
				} else {
					n, err = r.Read(ptr)
				}
			} else {
				// Ran out of buffer space, try reading one more byte
				var b [1]byte
				n, err = r.Read(b[:])

				if n > 0 {
					// Reader has more data but we have nowhere to
					// put it, so we're stuffed
					slices <- readerSlice{nil, io.ErrShortBuffer}
				} else {
					// Return some other error (hopefully EOF)
					slices <- readerSlice{nil, err}
				}
				return
			}

			// End on error (includes EOF)
			if err != nil {
				slices <- readerSlice{nil, err}
				return
			}

			if n > 0 {
				// Make a slice with the contents of the read
				slices <- readerSlice{ptr[:n], nil}

				// Adjust the scratch space slice
				ptr = ptr[n:]
			}
		}
	}
}

// Handle a read request.  Returns true if a response was sent, and false if
// the request should be queued.
func handleReadRequest(req readRequest, body []byte, complete bool) bool {
	log.Printf("HandlereadRequest %d %d %d", req.offset, req.maxsize, len(body))
	if req.offset < len(body) {
		var end int
		if req.offset+req.maxsize < len(body) {
			end = req.offset + req.maxsize
		} else {
			end = len(body)
		}
		req.result <- readResult{body[req.offset:end], nil}
		return true
	} else if complete && req.offset >= len(body) {
		req.result <- readResult{nil, io.EOF}
		return true
	} else {
		return false
	}
}

// Mediates between reads and appends.
// If 'source_reader' is not nil, reads data from 'source_reader' and stores it
// in the provided buffer.  Otherwise, use the contents of 'buffer' as is.
// Accepts read requests on the buffer on the 'requests' channel.  Completes
// when 'requests' channel is closed.
func transfer(source_buffer []byte, source_reader io.Reader, requests <-chan readRequest, reader_error chan error) {
	// currently buffered data
	var body []byte

	// for receiving slices from readIntoBuffer
	var slices chan readerSlice = nil

	// indicates whether the buffered data is complete
	var complete bool = false

	if source_reader != nil {
		// 'body' is the buffer slice representing the body content read so far
		body = source_buffer[:0]

		// used to communicate slices of the buffer as they are
		// readIntoBuffer will close 'slices' when it is done with it
		slices = make(chan readerSlice)

		// Spin it off
		go readIntoBuffer(source_buffer, source_reader, slices)
	} else {
		// use the whole buffer
		body = source_buffer[:]

		// buffer is complete
		complete = true
	}

	pending_requests := make([]readRequest, 0)

	for {
		select {
		case req, valid := <-requests:
			// Handle a buffer read request
			if valid {
				if !handleReadRequest(req, body, complete) {
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
					if handleReadRequest(pending_requests[n], body, complete) {

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
