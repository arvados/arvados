/* Provides low-level Get/Put primitives for accessing Arvados Keep blocks. */
package keepclient

import (
	"arvados.org/streamer"
	"crypto/md5"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"unsafe"
)

// A Keep "block" is 64MB.
const BLOCKSIZE = 64 * 1024 * 1024

var BlockNotFound = errors.New("Block not found")
var InsufficientReplicasError = errors.New("Could not write sufficient replicas")
var OversizeBlockError = errors.New("Block too big")
var MissingArvadosApiHost = errors.New("Missing required environment variable ARVADOS_API_HOST")
var MissingArvadosApiToken = errors.New("Missing required environment variable ARVADOS_API_TOKEN")

const X_Keep_Desired_Replicas = "X-Keep-Desired-Replicas"
const X_Keep_Replicas_Stored = "X-Keep-Replicas-Stored"

// Information about Arvados and Keep servers.
type KeepClient struct {
	ApiServer     string
	ApiToken      string
	ApiInsecure   bool
	Want_replicas int
	Client        *http.Client
	Using_proxy   bool
	External      bool
	service_roots *[]string
	lock          sync.Mutex
}

// Create a new KeepClient, initialized with standard Arvados environment
// variables ARVADOS_API_HOST, ARVADOS_API_TOKEN, and (optionally)
// ARVADOS_API_HOST_INSECURE.  This will contact the API server to discover
// Keep servers.
func MakeKeepClient() (kc KeepClient, err error) {
	insecure := (os.Getenv("ARVADOS_API_HOST_INSECURE") == "true")
	external := (os.Getenv("ARVADOS_EXTERNAL_CLIENT") == "true")

	kc = KeepClient{
		ApiServer:     os.Getenv("ARVADOS_API_HOST"),
		ApiToken:      os.Getenv("ARVADOS_API_TOKEN"),
		ApiInsecure:   insecure,
		Want_replicas: 2,
		Client: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}}},
		Using_proxy: false,
		External:    external}

	if os.Getenv("ARVADOS_API_HOST") == "" {
		return kc, MissingArvadosApiHost
	}
	if os.Getenv("ARVADOS_API_TOKEN") == "" {
		return kc, MissingArvadosApiToken
	}

	err = (&kc).DiscoverKeepServers()

	return kc, err
}

// Put a block given the block hash, a reader with the block data, and the
// expected length of that data.  The desired number of replicas is given in
// KeepClient.Want_replicas.  Returns the number of replicas that were written
// and if there was an error.  Note this will return InsufficientReplias
// whenever 0 <= replicas < this.Wants_replicas.
func (this KeepClient) PutHR(hash string, r io.Reader, expectedLength int64) (replicas int, err error) {

	// Buffer for reads from 'r'
	var bufsize int
	if expectedLength > 0 {
		if expectedLength > BLOCKSIZE {
			return 0, OversizeBlockError
		}
		bufsize = int(expectedLength)
	} else {
		bufsize = BLOCKSIZE
	}

	t := streamer.AsyncStreamFromReader(bufsize, HashCheckingReader{r, md5.New(), hash})
	defer t.Close()

	return this.putReplicas(hash, t, expectedLength)
}

// Put a block given the block hash and a byte buffer.  The desired number of
// replicas is given in KeepClient.Want_replicas.  Returns the number of
// replicas that were written and if there was an error.  Note this will return
// InsufficientReplias whenever 0 <= replicas < this.Wants_replicas.
func (this KeepClient) PutHB(hash string, buf []byte) (replicas int, err error) {
	t := streamer.AsyncStreamFromSlice(buf)
	defer t.Close()

	return this.putReplicas(hash, t, int64(len(buf)))
}

// Put a block given a buffer.  The hash will be computed.  The desired number
// of replicas is given in KeepClient.Want_replicas.  Returns the number of
// replicas that were written and if there was an error.  Note this will return
// InsufficientReplias whenever 0 <= replicas < this.Wants_replicas.
func (this KeepClient) PutB(buffer []byte) (hash string, replicas int, err error) {
	hash = fmt.Sprintf("%x", md5.Sum(buffer))
	replicas, err = this.PutHB(hash, buffer)
	return hash, replicas, err
}

// Put a block, given a Reader.  This will read the entire reader into a buffer
// to computed the hash.  The desired number of replicas is given in
// KeepClient.Want_replicas.  Returns the number of replicas that were written
// and if there was an error.  Note this will return InsufficientReplias
// whenever 0 <= replicas < this.Wants_replicas.  Also nhote that if the block
// hash and data size are available, PutHR() is more efficient.
func (this KeepClient) PutR(r io.Reader) (hash string, replicas int, err error) {
	if buffer, err := ioutil.ReadAll(r); err != nil {
		return "", 0, err
	} else {
		return this.PutB(buffer)
	}
}

// Get a block given a hash.  Return a reader, the expected data length, the
// URL the block was fetched from, and if there was an error.  If the block
// checksum does not match, the final Read() on the reader returned by this
// method will return a BadChecksum error instead of EOF.
func (this KeepClient) Get(hash string) (reader io.ReadCloser,
	contentLength int64, url string, err error) {
	return this.AuthorizedGet(hash, "", "")
}

// Get a block given a hash, with additional authorization provided by
// signature and timestamp.  Return a reader, the expected data length, the URL
// the block was fetched from, and if there was an error.  If the block
// checksum does not match, the final Read() on the reader returned by this
// method will return a BadChecksum error instead of EOF.
func (this KeepClient) AuthorizedGet(hash string,
	signature string,
	timestamp string) (reader io.ReadCloser,
	contentLength int64, url string, err error) {

	// Calculate the ordering for asking servers
	sv := this.shuffledServiceRoots(hash)

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
			return HashCheckingReader{resp.Body, md5.New(), hash}, resp.ContentLength, url, nil
		}
	}

	return nil, 0, "", BlockNotFound
}

// Determine if a block with the given hash is available and readable, but does
// not return the block contents.
func (this KeepClient) Ask(hash string) (contentLength int64, url string, err error) {
	return this.AuthorizedAsk(hash, "", "")
}

// Determine if a block with the given hash is available and readable with the
// given signature and timestamp, but does not return the block contents.
func (this KeepClient) AuthorizedAsk(hash string, signature string,
	timestamp string) (contentLength int64, url string, err error) {
	// Calculate the ordering for asking servers
	sv := this.shuffledServiceRoots(hash)

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

// Atomically read the service_roots field.
func (this *KeepClient) ServiceRoots() []string {
	r := (*[]string)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&this.service_roots))))
	return *r
}

// Atomically update the service_roots field.  Enables you to update
// service_roots without disrupting any GET or PUT operations that might
// already be in progress.
func (this *KeepClient) SetServiceRoots(svc []string) {
	// Must be sorted for ShuffledServiceRoots() to produce consistent
	// results.
	roots := make([]string, len(svc))
	copy(roots, svc)
	sort.Strings(roots)
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&this.service_roots)),
		unsafe.Pointer(&roots))
}
