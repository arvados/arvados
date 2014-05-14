package keepclient

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
)

type KeepClient struct {
	ApiServer     string
	ApiToken      string
	ApiInsecure   bool
	Service_roots []string
	Want_replicas int
	client        *http.Client
}

type KeepDisk struct {
	Hostname string `json:"service_host"`
	Port     int    `json:"service_port"`
	SSL      bool   `json:"service_ssl_flag"`
}

func MakeKeepClient() (kc *KeepClient, err error) {
	kc = &KeepClient{
		ApiServer:   os.Getenv("ARVADOS_API_HOST"),
		ApiToken:    os.Getenv("ARVADOS_API_TOKEN"),
		ApiInsecure: (os.Getenv("ARVADOS_API_HOST_INSECURE") != "")}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: kc.ApiInsecure},
	}

	kc.client = &http.Client{Transport: tr}

	err = kc.DiscoverKeepDisks()

	return kc, err
}

func (this *KeepClient) DiscoverKeepDisks() error {
	// Construct request of keep disk list
	var req *http.Request
	var err error
	if req, err = http.NewRequest("GET", fmt.Sprintf("https://%s/arvados/v1/keep_disks", this.ApiServer), nil); err != nil {
		return err
	}

	// Add api token header
	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.ApiToken))

	// Make the request
	var resp *http.Response
	if resp, err = this.client.Do(req); err != nil {
		return err
	}

	type SvcList struct {
		Items []KeepDisk `json:"items"`
	}

	// Decode json reply
	dec := json.NewDecoder(resp.Body)
	var m SvcList
	if err := dec.Decode(&m); err != nil {
		return err
	}

	listed := make(map[string]bool)
	this.Service_roots = make([]string, 0, len(m.Items))

	for _, element := range m.Items {
		n := ""
		if element.SSL {
			n = "s"
		}

		// Construct server URL
		url := fmt.Sprintf("http%s://%s:%d", n, element.Hostname, element.Port)

		// Skip duplicates
		if !listed[url] {
			listed[url] = true
			this.Service_roots = append(this.Service_roots, url)
		}
	}

	// Must be sorted for ShuffledServiceRoots() to produce consistent
	// results.
	sort.Strings(this.Service_roots)

	return nil
}

func (this KeepClient) ShuffledServiceRoots(hash string) (pseq []string) {
	// Build an ordering with which to query the Keep servers based on the
	// contents of the hash.  "hash" is a hex-encoded number at least 8
	// digits (32 bits) long

	// seed used to calculate the next keep server from 'pool' to be added
	// to 'pseq'
	seed := hash

	// Keep servers still to be added to the ordering
	pool := make([]string, len(this.Service_roots))
	copy(pool, this.Service_roots)

	// output probe sequence
	pseq = make([]string, 0, len(this.Service_roots))

	// iterate while there are servers left to be assigned
	for len(pool) > 0 {

		if len(seed) < 8 {
			// ran out of digits in the seed
			if len(pseq) < (len(hash) / 4) {
				// the number of servers added to the probe
				// sequence is less than the number of 4-digit
				// slices in 'hash' so refill the seed with the
				// last 4 digits.
				seed = hash[len(hash)-4:]
			}
			seed += hash
		}

		// Take the next 8 digits (32 bytes) and interpret as an integer,
		// then modulus with the size of the remaining pool to get the next
		// selected server.
		probe, _ := strconv.ParseUint(seed[0:8], 16, 32)
		probe %= uint64(len(pool))

		// Append the selected server to the probe sequence and remove it
		// from the pool.
		pseq = append(pseq, pool[probe])
		pool = append(pool[:probe], pool[probe+1:]...)

		// Remove the digits just used from the seed
		seed = seed[8:]
	}
	return pseq
}

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
		log.Printf("ReadIntoBuffer doing read")
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
			log.Printf("ReadIntoBuffer sending error %d %s", n, err.Error())
			slices <- ReaderSlice{nil, err}
			return
		}

		log.Printf("ReadIntoBuffer got %d", n)

		if n > 0 {
			log.Printf("ReadIntoBuffer sending readerslice")
			// Make a slice with the contents of the read
			slices <- ReaderSlice{ptr[:n], nil}
			log.Printf("ReadIntoBuffer sent readerslice")

			// Adjust the scratch space slice
			ptr = ptr[n:]
		}
	}
}

// A read request to the Transfer() function
type ReadRequest struct {
	offset int
	p      []byte
	result chan<- ReadResult
}

// A read result from the Transfer() function
type ReadResult struct {
	n   int
	err error
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
	this.requests <- ReadRequest{*this.offset, p, this.responses}
	rr, valid := <-this.responses
	if valid {
		*this.offset += rr.n
		return rr.n, rr.err
	} else {
		return 0, io.ErrUnexpectedEOF
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
	log.Printf("HandleReadRequest %d %d %t", req.offset, len(body), complete)
	if req.offset < len(body) {
		req.result <- ReadResult{copy(req.p, body[req.offset:]), nil}
		return true
	} else if complete && req.offset >= len(body) {
		req.result <- ReadResult{0, io.EOF}
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
		log.Printf("Doing select")
		select {
		case req, valid := <-requests:
			log.Printf("Got read request")
			// Handle a buffer read request
			if valid {
				if !HandleReadRequest(req, body, complete) {
					log.Printf("Queued")
					pending_requests = append(pending_requests, req)
				}
			} else {
				// closed 'requests' channel indicates we're done
				return
			}

		case bk, valid := <-slices:
			// Got a new slice from the reader
			if valid {
				log.Printf("Got readerslice %d", len(bk.slice))

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
						log.Printf("ReadRequest handled")

						// move the element from the
						// back of the slice to
						// position 'n', then shorten
						// the slice by one element
						pending_requests[n] = pending_requests[len(pending_requests)-1]
						pending_requests = pending_requests[0 : len(pending_requests)-1]
					} else {
						log.Printf("ReadRequest re-queued")

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

type UploadError struct {
	err error
	url string
}

func (this KeepClient) uploadToKeepServer(host string, hash string, body io.ReadCloser, upload_status chan<- UploadError) {
	var req *http.Request
	var err error
	var url = fmt.Sprintf("%s/%s", host, hash)
	if req, err = http.NewRequest("PUT", url, nil); err != nil {
		upload_status <- UploadError{err, url}
		return
	}

	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.ApiToken))
	req.Body = body

	var resp *http.Response
	if resp, err = this.client.Do(req); err != nil {
		upload_status <- UploadError{err, url}
	}

	if resp.StatusCode == http.StatusOK {
		upload_status <- UploadError{io.EOF, url}
	}
}

var KeepWriteError = errors.New("Could not write sufficient replicas")

func (this KeepClient) putReplicas(
	hash string,
	requests chan ReadRequest,
	reader_status chan error) error {

	// Calculate the ordering for uploading to servers
	sv := this.ShuffledServiceRoots(hash)

	// The next server to try contacting
	next_server := 0

	// The number of active writers
	active := 0

	// Used to communicate status from the upload goroutines
	upload_status := make(chan UploadError)
	defer close(upload_status)

	// Desired number of replicas
	want_replicas := this.Want_replicas

	for want_replicas > 0 {
		for active < want_replicas {
			// Start some upload requests
			if next_server < len(sv) {
				go this.uploadToKeepServer(sv[next_server], hash, MakeBufferReader(requests), upload_status)
				next_server += 1
				active += 1
			} else {
				return KeepWriteError
			}
		}

		// Now wait for something to happen.
		select {
		case status := <-reader_status:
			if status == io.EOF {
				// good news!
			} else {
				// bad news
				return status
			}
		case status := <-upload_status:
			if status.err == io.EOF {
				// good news!
				want_replicas -= 1
			} else {
				// writing to keep server failed for some reason
				log.Printf("Got error %s uploading to %s", status.err, status.url)
			}
			active -= 1
		}
	}

	return nil
}

func (this KeepClient) PutHR(hash string, r io.Reader) error {

	// Buffer for reads from 'r'
	buffer := make([]byte, 64*1024*1024)

	// Read requests on Transfer() buffer
	requests := make(chan ReadRequest)
	defer close(requests)

	// Reporting reader error states
	reader_status := make(chan error)

	// Start the transfer goroutine
	go Transfer(buffer, r, requests, reader_status)

	return this.putReplicas(hash, requests, reader_status)
}

func (this KeepClient) PutHB(hash string, buffer []byte) error {
	// Read requests on Transfer() buffer
	requests := make(chan ReadRequest)
	defer close(requests)

	// Start the transfer goroutine
	go Transfer(buffer, nil, requests, nil)

	return this.putReplicas(hash, requests, nil)
}

func (this KeepClient) PutB(buffer []byte) error {
	return this.PutHB(fmt.Sprintf("%x", md5.Sum(buffer)), buffer)
}

func (this KeepClient) PutR(r io.Reader) error {
	if buffer, err := ioutil.ReadAll(r); err != nil {
		return err
	} else {
		return this.PutB(buffer)
	}
}
