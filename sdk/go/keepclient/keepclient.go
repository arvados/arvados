/* Provides low-level Get/Put primitives for accessing Arvados Keep blocks. */
package keepclient

import (
	"crypto/md5"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/streamer"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	Arvados       *arvadosclient.ArvadosClient
	Want_replicas int
	Using_proxy   bool
	service_roots *map[string]string
	lock          sync.Mutex
	Client        *http.Client
}

// Create a new KeepClient.  This will contact the API server to discover Keep
// servers.
func MakeKeepClient(arv *arvadosclient.ArvadosClient) (kc KeepClient, err error) {
	kc = KeepClient{
		Arvados:       arv,
		Want_replicas: 2,
		Using_proxy:   false,
		Client:        &http.Client{},
	}
	_, err = (&kc).DiscoverKeepServers()

	return kc, err
}

// Put a block given the block hash, a reader with the block data, and the
// expected length of that data.  The desired number of replicas is given in
// KeepClient.Want_replicas.  Returns the number of replicas that were written
// and if there was an error.  Note this will return InsufficientReplias
// whenever 0 <= replicas < this.Wants_replicas.
func (this KeepClient) PutHR(hash string, r io.Reader, expectedLength int64) (locator string, replicas int, err error) {

	// Buffer for reads from 'r'
	var bufsize int
	if expectedLength > 0 {
		if expectedLength > BLOCKSIZE {
			return "", 0, OversizeBlockError
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
func (this KeepClient) PutHB(hash string, buf []byte) (locator string, replicas int, err error) {
	t := streamer.AsyncStreamFromSlice(buf)
	defer t.Close()

	return this.putReplicas(hash, t, int64(len(buf)))
}

// Put a block given a buffer.  The hash will be computed.  The desired number
// of replicas is given in KeepClient.Want_replicas.  Returns the number of
// replicas that were written and if there was an error.  Note this will return
// InsufficientReplias whenever 0 <= replicas < this.Wants_replicas.
func (this KeepClient) PutB(buffer []byte) (locator string, replicas int, err error) {
	hash := fmt.Sprintf("%x", md5.Sum(buffer))
	return this.PutHB(hash, buffer)
}

// Put a block, given a Reader.  This will read the entire reader into a buffer
// to compute the hash.  The desired number of replicas is given in
// KeepClient.Want_replicas.  Returns the number of replicas that were written
// and if there was an error.  Note this will return InsufficientReplias
// whenever 0 <= replicas < this.Wants_replicas.  Also nhote that if the block
// hash and data size are available, PutHR() is more efficient.
func (this KeepClient) PutR(r io.Reader) (locator string, replicas int, err error) {
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

	// Take the hash of locator and timestamp in order to identify this
	// specific transaction in log statements.
	requestId := fmt.Sprintf("%x", md5.Sum([]byte(hash+time.Now().String())))[0:8]

	// Calculate the ordering for asking servers
	sv := NewRootSorter(this.ServiceRoots(), hash).GetSortedRoots()

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

		req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.Arvados.ApiToken))

		log.Printf("[%v] Begin download %s", requestId, url)

		var resp *http.Response
		if resp, err = this.Client.Do(req); err != nil || resp.StatusCode != http.StatusOK {
			statusCode := -1
			var respbody []byte
			if resp != nil {
				statusCode = resp.StatusCode
				if resp.Body != nil {
					respbody, _ = ioutil.ReadAll(&io.LimitedReader{resp.Body, 4096})
				}
			}
			response := strings.TrimSpace(string(respbody))
			log.Printf("[%v] Download %v status code: %v error: \"%v\" response: \"%v\"",
				requestId, url, statusCode, err, response)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			log.Printf("[%v] Download %v status code: %v", requestId, url, resp.StatusCode)
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
	sv := NewRootSorter(this.ServiceRoots(), hash).GetSortedRoots()

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

		req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.Arvados.ApiToken))

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
func (this *KeepClient) ServiceRoots() map[string]string {
	r := (*map[string]string)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&this.service_roots))))
	return *r
}

// Atomically update the service_roots field.  Enables you to update
// service_roots without disrupting any GET or PUT operations that might
// already be in progress.
func (this *KeepClient) SetServiceRoots(new_roots map[string]string) {
	roots := make(map[string]string)
	for uuid, root := range new_roots {
		roots[uuid] = root
	}
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&this.service_roots)),
		unsafe.Pointer(&roots))
}

type Locator struct {
	Hash      string
	Size      int
	Signature string
	Timestamp string
}

func MakeLocator2(hash string, hints string) (locator Locator) {
	locator.Hash = hash
	if hints != "" {
		signature_pat, _ := regexp.Compile("^A([[:xdigit:]]+)@([[:xdigit:]]{8})$")
		for _, hint := range strings.Split(hints, "+") {
			if hint != "" {
				if match, _ := regexp.MatchString("^[[:digit:]]+$", hint); match {
					fmt.Sscanf(hint, "%d", &locator.Size)
				} else if m := signature_pat.FindStringSubmatch(hint); m != nil {
					locator.Signature = m[1]
					locator.Timestamp = m[2]
				} else if match, _ := regexp.MatchString("^[:upper:]", hint); match {
					// Any unknown hint that starts with an uppercase letter is
					// presumed to be valid and ignored, to permit forward compatibility.
				} else {
					// Unknown format; not a valid locator.
					return Locator{"", 0, "", ""}
				}
			}
		}
	}
	return locator
}

func MakeLocator(path string) Locator {
	pathpattern, err := regexp.Compile("^([0-9a-f]{32})([+].*)?$")
	if err != nil {
		log.Print("Don't like regexp", err)
	}

	sm := pathpattern.FindStringSubmatch(path)
	if sm == nil {
		log.Print("Failed match ", path)
		return Locator{"", 0, "", ""}
	}

	return MakeLocator2(sm[1], sm[2])
}
