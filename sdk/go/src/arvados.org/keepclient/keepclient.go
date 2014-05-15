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

// A Keep "block" is 64MB.
const BLOCKSIZE = 64 * 1024 * 1024

var BlockNotFound = errors.New("Block not found")
var InsufficientReplicasError = errors.New("Could not write sufficient replicas")

type KeepClient struct {
	ApiServer     string
	ApiToken      string
	ApiInsecure   bool
	Service_roots []string
	Want_replicas int
	Client        *http.Client
}

type KeepDisk struct {
	Hostname string `json:"service_host"`
	Port     int    `json:"service_port"`
	SSL      bool   `json:"service_ssl_flag"`
}

func MakeKeepClient() (kc KeepClient, err error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: kc.ApiInsecure},
	}

	kc = KeepClient{
		ApiServer:     os.Getenv("ARVADOS_API_HOST"),
		ApiToken:      os.Getenv("ARVADOS_API_TOKEN"),
		ApiInsecure:   (os.Getenv("ARVADOS_API_HOST_INSECURE") != ""),
		Want_replicas: 2,
		Client:        &http.Client{Transport: tr}}

	err = (&kc).DiscoverKeepServers()

	return kc, err
}

func (this *KeepClient) DiscoverKeepServers() error {
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
	if resp, err = this.Client.Do(req); err != nil {
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

type UploadStatus struct {
	Err        error
	Url        string
	StatusCode int
}

func (this KeepClient) uploadToKeepServer(host string, hash string, body io.ReadCloser,
	upload_status chan<- UploadStatus, expectedLength int64) {

	log.Printf("Uploading to %s", host)

	var req *http.Request
	var err error
	var url = fmt.Sprintf("%s/%s", host, hash)
	if req, err = http.NewRequest("PUT", url, nil); err != nil {
		upload_status <- UploadStatus{err, url, 0}
		return
	}

	if expectedLength > 0 {
		req.ContentLength = expectedLength
	}

	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.ApiToken))
	req.Header.Add("Content-Type", "application/octet-stream")
	req.Body = body

	var resp *http.Response
	if resp, err = this.Client.Do(req); err != nil {
		upload_status <- UploadStatus{err, url, 0}
		return
	}

	if resp.StatusCode == http.StatusOK {
		upload_status <- UploadStatus{nil, url, resp.StatusCode}
	} else {
		upload_status <- UploadStatus{errors.New(resp.Status), url, resp.StatusCode}
	}
}

func (this KeepClient) putReplicas(
	hash string,
	requests chan ReadRequest,
	reader_status chan error,
	expectedLength int64) (replicas int, err error) {

	// Calculate the ordering for uploading to servers
	sv := this.ShuffledServiceRoots(hash)

	// The next server to try contacting
	next_server := 0

	// The number of active writers
	active := 0

	// Used to communicate status from the upload goroutines
	upload_status := make(chan UploadStatus)
	defer close(upload_status)

	// Desired number of replicas
	remaining_replicas := this.Want_replicas

	for remaining_replicas > 0 {
		for active < remaining_replicas {
			// Start some upload requests
			if next_server < len(sv) {
				go this.uploadToKeepServer(sv[next_server], hash, MakeBufferReader(requests), upload_status, expectedLength)
				next_server += 1
				active += 1
			} else {
				return (this.Want_replicas - remaining_replicas), InsufficientReplicasError
			}
		}

		// Now wait for something to happen.
		select {
		case status := <-reader_status:
			if status == io.EOF {
				// good news!
			} else {
				// bad news
				return (this.Want_replicas - remaining_replicas), status
			}
		case status := <-upload_status:
			if status.StatusCode == 200 {
				// good news!
				remaining_replicas -= 1
			} else {
				// writing to keep server failed for some reason
				log.Printf("Keep server put to %v failed with '%v'",
					status.Url, status.Err)
			}
			active -= 1
			log.Printf("Upload status %v %v %v", status.StatusCode, remaining_replicas, active)
		}
	}

	return (this.Want_replicas - remaining_replicas), nil
}

var OversizeBlockError = errors.New("Block too big")

func (this KeepClient) PutHR(hash string, r io.Reader, expectedLength int64) (replicas int, err error) {

	// Buffer for reads from 'r'
	var buffer []byte
	if expectedLength > 0 {
		if expectedLength > BLOCKSIZE {
			return 0, OversizeBlockError
		}
		buffer = make([]byte, expectedLength)
	} else {
		buffer = make([]byte, BLOCKSIZE)
	}

	// Read requests on Transfer() buffer
	requests := make(chan ReadRequest)
	defer close(requests)

	// Reporting reader error states
	reader_status := make(chan error)
	defer close(reader_status)

	// Start the transfer goroutine
	go Transfer(buffer, r, requests, reader_status)

	return this.putReplicas(hash, requests, reader_status, expectedLength)
}

func (this KeepClient) PutHB(hash string, buffer []byte) (replicas int, err error) {
	// Read requests on Transfer() buffer
	requests := make(chan ReadRequest)
	defer close(requests)

	// Start the transfer goroutine
	go Transfer(buffer, nil, requests, nil)

	return this.putReplicas(hash, requests, nil, int64(len(buffer)))
}

func (this KeepClient) PutB(buffer []byte) (hash string, replicas int, err error) {
	hash = fmt.Sprintf("%x", md5.Sum(buffer))
	replicas, err = this.PutHB(hash, buffer)
	return hash, replicas, err
}

func (this KeepClient) PutR(r io.Reader) (hash string, replicas int, err error) {
	if buffer, err := ioutil.ReadAll(r); err != nil {
		return "", 0, err
	} else {
		return this.PutB(buffer)
	}
}

func (this KeepClient) Get(hash string) (reader io.ReadCloser,
	contentLength int64, url string, err error) {
	return this.AuthorizedGet(hash, "", "")
}

func (this KeepClient) AuthorizedGet(hash string,
	signature string,
	timestamp string) (reader io.ReadCloser,
	contentLength int64, url string, err error) {

	// Calculate the ordering for asking servers
	sv := this.ShuffledServiceRoots(hash)

	for _, host := range sv {
		var req *http.Request
		var err error
		var url string
		if signature != "" {
			url = fmt.Sprintf("%s/%s+A%s@%s", host, hash,
				signature, timestamp)
		} else {
			url = fmt.Sprintf("%s/%s", host, hash)
		}
		if req, err = http.NewRequest("GET", url, nil); err != nil {
			continue
		}

		req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.ApiToken))

		var resp *http.Response
		if resp, err = this.Client.Do(req); err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp.Body, resp.ContentLength, url, nil
		}
	}

	return nil, 0, "", BlockNotFound
}

func (this KeepClient) Ask(hash string) (contentLength int64, url string, err error) {
	return this.AuthorizedAsk(hash, "", "")
}

func (this KeepClient) AuthorizedAsk(hash string, signature string,
	timestamp string) (contentLength int64, url string, err error) {
	// Calculate the ordering for asking servers
	sv := this.ShuffledServiceRoots(hash)

	for _, host := range sv {
		var req *http.Request
		var err error
		if signature != "" {
			url = fmt.Sprintf("%s/%s+A%s@%s", host, hash,
				signature, timestamp)
		} else {
			url = fmt.Sprintf("%s/%s", host, hash)
		}

		if req, err = http.NewRequest("HEAD", url, nil); err != nil {
			continue
		}

		req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.ApiToken))

		var resp *http.Response
		if resp, err = this.Client.Do(req); err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return resp.ContentLength, url, nil
		}
	}

	return 0, "", BlockNotFound

}
