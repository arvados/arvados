package keepclient

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
)

type KeepClient struct {
	Service_roots []string
	ApiToken      string
}

type KeepDisk struct {
	Hostname string `json:"service_host"`
	Port     int    `json:"service_port"`
	SSL      bool   `json:"service_ssl_flag"`
}

func MakeKeepClient() (kc *KeepClient, err error) {
	kc := KeepClient{}
	err := kc.DiscoverKeepDisks()
	if err != nil {
		return nil, err
	}
	return &kc, nil
}

func (this *KeepClient) DiscoverKeepDisks() error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	var req *http.Request
	if req, err = http.NewRequest("GET", "https://localhost:3001/arvados/v1/keep_disks", nil); err != nil {
		return nil, err
	}

	var resp *http.Response
	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.ApiToken))
	if resp, err = client.Do(req); err != nil {
		return nil, err
	}

	type SvcList struct {
		Items []KeepDisk `json:"items"`
	}
	dec := json.NewDecoder(resp.Body)
	var m SvcList
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}

	this.service_roots = make([]string, len(m.Items))
	for index, element := range m.Items {
		n := ""
		if element.SSL {
			n = "s"
		}
		this.service_roots[index] = fmt.Sprintf("http%s://%s:%d",
			n, element.Hostname, element.Port)
	}
	sort.Strings(this.service_roots)
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

type Source <-chan ReaderSlice
type Sink chan<- ReaderSlice
type Status chan error

// Read repeatedly from the reader into the specified buffer, and report each
// read to channel 'c'.  Completes when Reader 'r' reports an error and closes
// channel 'c'.
func ReadIntoBuffer(buffer []byte, r io.Reader, c Sink) {
	defer close(c)

	// Initially use entire buffer as scratch space
	ptr := buffer[:]
	for len(ptr) > 0 {
		v // Read into the scratch space
		n, err := r.Read(ptr)

		// End on error (includes EOF)
		if err != nil {
			c <- ReaderSlice{nil, err}
			return
		}

		// Make a slice with the contents of the read
		c <- ReaderSlice{ptr[:n], nil}

		// Adjust the scratch space slice
		ptr = ptr[n:]
	}
	if len(ptr) == 0 {
		c <- ReaderSlice{nil, io.ErrShortBuffer}
	}
}

// Take slices from 'source' channel and write them to Writer 'w'.  Reports read
// or write errors on 'status'.  Completes when 'source' channel is closed.
func SinkWriter(source Source, w io.Writer, status Status) {
	can_write = true

	for {
		// Get the next block from the source
		rs, valid := <-source

		if valid {
			if rs.error != nil {
				// propagate reader status (should only be EOF)
				status <- rs.error
			} else if can_write {
				buf := rs.slice[:]
				for len(buf) > 0 {
					n, err := w.Write(buf)
					buf = buf[n:]
					if err == io.ErrShortWrite {
						// short write, so go around again
					} else if err != nil {
						// some other write error,
						// propagate error and stop
						// further writes
						status <- err
						can_write = false
					}
				}
			}
		} else {
			// source channel closed
			break
		}
	}
}

func closeSinks(sinks_slice []Sink) {
	for _, s := range sinks_slice {
		close(s)
	}
}

// Transfer data from a source (either an already-filled buffer, or a reader)
// into one or more 'sinks'.  If 'source' is valid, it will read from the
// reader into the buffer and send the data to the sinks.  Otherwise 'buffer'
// it will just send the contents of the buffer to the sinks.  Completes when
// the 'sinks' channel is closed.
func Transfer(source_buffer []byte, source_reader io.Reader, sinks <-chan Sink, reader_error chan error) {
	// currently buffered data
	var body []byte

	// for receiving slices from ReadIntoBuffer
	var slices chan []byte = nil

	// indicates whether the buffered data is complete
	var complete bool = false

	if source != nil {
		// 'body' is the buffer slice representing the body content read so far
		body = source_buffer[:0]

		// used to communicate slices of the buffer as read
		reader_slices := make(chan []ReaderSlice)

		// Spin it off
		go ReadIntoBuffer(source_buffer, source_reader, reader_slices)
	} else {
		// use the whole buffer
		body = source_buffer[:]

		// that's it
		complete = true
	}

	// list of sinks to send to
	sinks_slice := make([]Sink, 0)
	defer closeSinks(sinks_slice)

	for {
		select {
		case s, valid := <-sinks:
			if valid {
				// add to the sinks slice
				sinks_slice = append(sinks_slice, s)

				// catch up the sink with the current body contents
				if len(body) > 0 {
					s <- ReaderSlice{body, nil}
					if complete {
						s <- ReaderSlice{nil, io.EOF}
					}
				}
			} else {
				// closed 'sinks' channel indicates we're done
				return
			}

		case bk, valid := <-slices:
			if valid {
				if bk.err != nil {
					reader_error <- bk.err
					if bk.err == io.EOF {
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

				// send the new slice to the sinks
				for _, s := range sinks_slice {
					s <- bk
				}

				if complete {
					// got an EOF, so close the sinks
					closeSinks(sinks_slice)

					// truncate sinks slice
					sinks_slice = sinks_slice[:0]
				}
			} else {
				// no more reads
				slices = nil
			}
		}
	}
}

func (this KeepClient) ConnectToKeepServer(url string, sinks chan<- Sink, write_status chan<- error) {
	pipereader, pipewriter := io.Pipe()

	var req *http.Request
	if req, err = http.NewRequest("POST", url, nil); err != nil {
		write_status <- err
	}
	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.ApiToken))
	req.Body = pipereader

	// create channel to transfer slices from reader to writer
	tr := make(chan ReaderSlice)

	// start the writer goroutine
	go SinkWriter(tr, pipewriter, write_status)

	// now transfer the channel to the reader goroutine
	sinks <- tr

	var resp *http.Response

	if resp, err = client.Do(req); err != nil {
		return nil, err
	}
}

var KeepWriteError = errors.new("Could not write sufficient replicas")

func (this KeepClient) KeepPut(hash string, r io.Reader, want_replicas int) error {
	// Calculate the ordering to try writing to servers
	sv := this.ShuffledServiceRoots(hash)

	// The next server to try contacting
	n := 0

	// The number of active writers
	active := 0

	// Used to buffer reads from 'r'
	buffer := make([]byte, 64*1024*1024)

	// Used to send writers to the reader goroutine
	sinks := make(chan Sink)
	defer close(sinks)

	// Used to communicate status from the reader goroutine
	reader_status := make(chan error)

	// Start the reader goroutine
	go Transfer(buffer, r, sinks, reader_status)

	// Used to communicate status from the writer goroutines
	write_status := make(chan error)

	for want_replicas > 0 {
		for active < want_replicas {
			// Start some writers
			if n < len(sv) {
				go this.ConnectToKeepServer(sv[n], sinks, write_status)
				n += 1
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
		case status := <-write_status:
			if status == io.EOF {
				// good news!
				want_replicas -= 1
			} else {
				// writing to keep server failed for some reason.
			}
			active -= 1
		}
	}
}
