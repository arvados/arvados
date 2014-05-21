/* Internal implementation of AsyncStream.
Outline of operation:

The kernel is the transfer() goroutine.  It manages concurrent reads and
appends to the "body" slice.  "body" is a slice of "source_buffer" that
represents the segment of the buffer that is already filled in and available
for reading.

To fill in the buffer, transfer() starts the readIntoBuffer() goroutine to read
from the io.Reader source directly into source_buffer.  Each read goes into a
slice of buffer which spans the section immediately following the end of the
current "body".  Each time a Read completes, a slice representing the the
section just filled in (or any read errors/EOF) is sent over the "slices"
channel back to the transfer() function.

Meanwhile, the transfer() function selects() on two channels, the "requests"
channel and the "slices" channel.

When a message is recieved on the "slices" channel, this means the a new
section of the buffer has data, or an error is signaled.  Since the data has
been read directly into the source_buffer, it is able to simply increases the
size of the body slice to encompass the newly filled in section.  Then any
pending reads are serviced with handleReadRequest (described below).

When a message is recieved on the "requests" channel, it means a StreamReader
wants access to a slice of the buffer.  This is passed to handleReadRequest().

The handleReadRequest() function takes a sliceRequest consisting of a buffer
offset, maximum size, and channel to send the response.  If there was an error
reported from the source reader, it is returned.  If the offset is less than
the size of the body, the request can proceed, and it sends a body slice
spanning the segment from offset to min(offset+maxsize, end of the body).  If
source reader status is EOF (done filling the buffer) and the read request
offset is beyond end of the body, it responds with EOF.  Otherwise, the read
request is for a slice beyond the current size of "body" but we expect the body
to expand as more data is added, so the request gets added to a wait list.

The transfer() runs until the requests channel is closed by AsyncStream.Close()

To track readers, streamer uses the readersMonitor() goroutine.  This goroutine
chooses which channels to receive from based on the number of outstanding
readers.  When a new reader is created, it sends a message on the add_reader
channel.  If the number of readers is already at MAX_READERS, this blocks the
sender until an existing reader is closed.  When a reader is closed, it sends a
message on the subtract_reader channel.  Finally, when AsyncStream.Close() is
called, it sends a message on the wait_zero_readers channel, which will block
the sender unless there are zero readers and it is safe to shut down the
AsyncStream.
*/

package streamer

import (
	"io"
)

const MAX_READERS = 100

// A slice passed from readIntoBuffer() to transfer()
type nextSlice struct {
	slice        []byte
	reader_error error
}

// A read request to the Transfer() function
type sliceRequest struct {
	offset  int
	maxsize int
	result  chan<- sliceResult
}

// A read result from the Transfer() function
type sliceResult struct {
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
func readIntoBuffer(buffer []byte, r io.Reader, slices chan<- nextSlice) {
	defer close(slices)

	if writeto, ok := r.(io.WriterTo); ok {
		n, err := writeto.WriteTo(&bufferWriter{buffer, 0})
		if err != nil {
			slices <- nextSlice{nil, err}
		} else {
			slices <- nextSlice{buffer[:n], nil}
			slices <- nextSlice{nil, io.EOF}
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
					slices <- nextSlice{nil, io.ErrShortBuffer}
				} else {
					// Return some other error (hopefully EOF)
					slices <- nextSlice{nil, err}
				}
				return
			}

			// End on error (includes EOF)
			if err != nil {
				slices <- nextSlice{nil, err}
				return
			}

			if n > 0 {
				// Make a slice with the contents of the read
				slices <- nextSlice{ptr[:n], nil}

				// Adjust the scratch space slice
				ptr = ptr[n:]
			}
		}
	}
}

// Handle a read request.  Returns true if a response was sent, and false if
// the request should be queued.
func handleReadRequest(req sliceRequest, body []byte, reader_status error) bool {
	if (reader_status != nil) && (reader_status != io.EOF) {
		req.result <- sliceResult{nil, reader_status}
		return true
	} else if req.offset < len(body) {
		var end int
		if req.offset+req.maxsize < len(body) {
			end = req.offset + req.maxsize
		} else {
			end = len(body)
		}
		req.result <- sliceResult{body[req.offset:end], nil}
		return true
	} else if (reader_status == io.EOF) && (req.offset >= len(body)) {
		req.result <- sliceResult{nil, io.EOF}
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
func (this *AsyncStream) transfer(source_reader io.Reader) {
	source_buffer := this.buffer
	requests := this.requests

	// currently buffered data
	var body []byte

	// for receiving slices from readIntoBuffer
	var slices chan nextSlice = nil

	// indicates the status of the underlying reader
	var reader_status error = nil

	if source_reader != nil {
		// 'body' is the buffer slice representing the body content read so far
		body = source_buffer[:0]

		// used to communicate slices of the buffer as they are
		// readIntoBuffer will close 'slices' when it is done with it
		slices = make(chan nextSlice)

		// Spin it off
		go readIntoBuffer(source_buffer, source_reader, slices)
	} else {
		// use the whole buffer
		body = source_buffer[:]

		// buffer is complete
		reader_status = io.EOF
	}

	pending_requests := make([]sliceRequest, 0)

	for {
		select {
		case req, valid := <-requests:
			// Handle a buffer read request
			if valid {
				if !handleReadRequest(req, body, reader_status) {
					pending_requests = append(pending_requests, req)
				}
			} else {
				// closed 'requests' channel indicates we're done
				return
			}

		case bk, valid := <-slices:
			// Got a new slice from the reader
			if valid {
				reader_status = bk.reader_error

				if bk.slice != nil {
					// adjust body bounds now that another slice has been read
					body = source_buffer[0 : len(body)+len(bk.slice)]
				}

				// handle pending reads
				n := 0
				for n < len(pending_requests) {
					if handleReadRequest(pending_requests[n], body, reader_status) {
						// move the element from the back of the slice to
						// position 'n', then shorten the slice by one element
						pending_requests[n] = pending_requests[len(pending_requests)-1]
						pending_requests = pending_requests[0 : len(pending_requests)-1]
					} else {

						// Request wasn't handled, so keep it in the request slice
						n += 1
					}
				}
			} else {
				if reader_status == io.EOF {
					// no more reads expected, so this is ok
				} else {
					// slices channel closed without signaling EOF
					reader_status = io.ErrUnexpectedEOF
				}
				slices = nil
			}
		}
	}
}

func (this *AsyncStream) readersMonitor() {
	var readers int = 0

	for {
		if readers == 0 {
			select {
			case _, ok := <-this.wait_zero_readers:
				if ok {
					// nothing, just implicitly unblock the sender
				} else {
					return
				}
			case _, ok := <-this.add_reader:
				if ok {
					readers += 1
				} else {
					return
				}
			}
		} else if readers > 0 && readers < MAX_READERS {
			select {
			case _, ok := <-this.add_reader:
				if ok {
					readers += 1
				} else {
					return
				}

			case _, ok := <-this.subtract_reader:
				if ok {
					readers -= 1
				} else {
					return
				}
			}
		} else if readers == MAX_READERS {
			_, ok := <-this.subtract_reader
			if ok {
				readers -= 1
			} else {
				return
			}
		}
	}
}
