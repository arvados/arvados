package keepclient

import (
	"arvados.org/buffer"
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
	requests chan buffer.ReadRequest,
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
				go this.uploadToKeepServer(sv[next_server], hash, buffer.MakeBufferReader(requests), upload_status, expectedLength)
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
	var buf []byte
	if expectedLength > 0 {
		if expectedLength > BLOCKSIZE {
			return 0, OversizeBlockError
		}
		buf = make([]byte, expectedLength)
	} else {
		buf = make([]byte, BLOCKSIZE)
	}

	// Read requests on Transfer() buffer
	requests := make(chan buffer.ReadRequest)
	defer close(requests)

	// Reporting reader error states
	reader_status := make(chan error)
	defer close(reader_status)

	// Start the transfer goroutine
	go buffer.Transfer(buf, r, requests, reader_status)

	return this.putReplicas(hash, requests, reader_status, expectedLength)
}

func (this KeepClient) PutHB(hash string, buf []byte) (replicas int, err error) {
	// Read requests on Transfer() buffer
	requests := make(chan buffer.ReadRequest)
	defer close(requests)

	// Start the transfer goroutine
	go buffer.Transfer(buf, nil, requests, nil)

	return this.putReplicas(hash, requests, nil, int64(len(buf)))
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
