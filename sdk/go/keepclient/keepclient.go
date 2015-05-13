/* Provides low-level Get/Put primitives for accessing Arvados Keep blocks. */
package keepclient

import (
	"crypto/md5"
	"crypto/tls"
	"errors"
	"fmt"
	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/streamer"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// A Keep "block" is 64MB.
const BLOCKSIZE = 64 * 1024 * 1024

var BlockNotFound = errors.New("Block not found")
var InsufficientReplicasError = errors.New("Could not write sufficient replicas")
var OversizeBlockError = errors.New("Exceeded maximum block size (" + strconv.Itoa(BLOCKSIZE) + ")")
var MissingArvadosApiHost = errors.New("Missing required environment variable ARVADOS_API_HOST")
var MissingArvadosApiToken = errors.New("Missing required environment variable ARVADOS_API_TOKEN")
var InvalidLocatorError = errors.New("Invalid locator")

const X_Keep_Desired_Replicas = "X-Keep-Desired-Replicas"
const X_Keep_Replicas_Stored = "X-Keep-Replicas-Stored"

// Information about Arvados and Keep servers.
type KeepClient struct {
	Arvados       *arvadosclient.ArvadosClient
	Want_replicas int
	Using_proxy   bool
	localRoots    *map[string]string
	gatewayRoots  *map[string]string
	writableRoots *map[string]string
	lock          sync.RWMutex
	Client        *http.Client
}

// Create a new KeepClient.  This will contact the API server to discover Keep
// servers.
func MakeKeepClient(arv *arvadosclient.ArvadosClient) (*KeepClient, error) {
	var matchTrue = regexp.MustCompile("^(?i:1|yes|true)$")
	insecure := matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))
	kc := &KeepClient{
		Arvados:       arv,
		Want_replicas: 2,
		Using_proxy:   false,
		Client: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}}},
	}
	return kc, kc.DiscoverKeepServers()
}

// Put a block given the block hash, a reader, and the number of bytes
// to read from the reader (which must be between 0 and BLOCKSIZE).
//
// Returns the locator for the written block, the number of replicas
// written, and an error.
//
// Returns an InsufficientReplicas error if 0 <= replicas <
// kc.Wants_replicas.
func (kc *KeepClient) PutHR(hash string, r io.Reader, dataBytes int64) (string, int, error) {
	// Buffer for reads from 'r'
	var bufsize int
	if dataBytes > 0 {
		if dataBytes > BLOCKSIZE {
			return "", 0, OversizeBlockError
		}
		bufsize = int(dataBytes)
	} else {
		bufsize = BLOCKSIZE
	}

	t := streamer.AsyncStreamFromReader(bufsize, HashCheckingReader{r, md5.New(), hash})
	defer t.Close()

	return kc.putReplicas(hash, t, dataBytes)
}

// PutHB writes a block to Keep. The hash of the bytes is given in
// hash, and the data is given in buf.
//
// Return values are the same as for PutHR.
func (kc *KeepClient) PutHB(hash string, buf []byte) (string, int, error) {
	t := streamer.AsyncStreamFromSlice(buf)
	defer t.Close()
	return kc.putReplicas(hash, t, int64(len(buf)))
}

// PutB writes a block to Keep. It computes the hash itself.
//
// Return values are the same as for PutHR.
func (kc *KeepClient) PutB(buffer []byte) (string, int, error) {
	hash := fmt.Sprintf("%x", md5.Sum(buffer))
	return kc.PutHB(hash, buffer)
}

// PutR writes a block to Keep. It first reads all data from r into a buffer
// in order to compute the hash.
//
// Return values are the same as for PutHR.
//
// If the block hash and data size are known, PutHR is more efficient.
func (kc *KeepClient) PutR(r io.Reader) (locator string, replicas int, err error) {
	if buffer, err := ioutil.ReadAll(r); err != nil {
		return "", 0, err
	} else {
		return kc.PutB(buffer)
	}
}

// Get() retrieves a block, given a locator. Returns a reader, the
// expected data length, the URL the block is being fetched from, and
// an error.
//
// If the block checksum does not match, the final Read() on the
// reader returned by this method will return a BadChecksum error
// instead of EOF.
func (kc *KeepClient) Get(locator string) (io.ReadCloser, int64, string, error) {
	var errs []string
	for _, host := range kc.getSortedRoots(locator) {
		url := host + "/" + locator
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", kc.Arvados.ApiToken))
		resp, err := kc.Client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				var respbody []byte
				if resp.Body != nil {
					respbody, _ = ioutil.ReadAll(&io.LimitedReader{resp.Body, 4096})
				}
				errs = append(errs, fmt.Sprintf("%s: %d %s",
					url, resp.StatusCode, strings.TrimSpace(string(respbody))))
			} else {
				errs = append(errs, fmt.Sprintf("%s: %v", url, err))
			}
			continue
		}
		return HashCheckingReader{
			Reader: resp.Body,
			Hash:   md5.New(),
			Check:  locator[0:32],
		}, resp.ContentLength, url, nil
	}
	log.Printf("DEBUG: GET %s failed: %v", locator, errs)
	return nil, 0, "", BlockNotFound
}

// Ask() verifies that a block with the given hash is available and
// readable, according to at least one Keep service. Unlike Get, it
// does not retrieve the data or verify that the data content matches
// the hash specified by the locator.
//
// Returns the data size (content length) reported by the Keep service
// and the URI reporting the data size.
func (kc *KeepClient) Ask(locator string) (int64, string, error) {
	for _, host := range kc.getSortedRoots(locator) {
		url := host + "/" + locator
		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			continue
		}
		req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", kc.Arvados.ApiToken))
		if resp, err := kc.Client.Do(req); err == nil && resp.StatusCode == http.StatusOK {
			return resp.ContentLength, url, nil
		}
	}
	return 0, "", BlockNotFound
}

// LocalRoots() returns the map of local (i.e., disk and proxy) Keep
// services: uuid -> baseURI.
func (kc *KeepClient) LocalRoots() map[string]string {
	kc.lock.RLock()
	defer kc.lock.RUnlock()
	return *kc.localRoots
}

// GatewayRoots() returns the map of Keep remote gateway services:
// uuid -> baseURI.
func (kc *KeepClient) GatewayRoots() map[string]string {
	kc.lock.RLock()
	defer kc.lock.RUnlock()
	return *kc.gatewayRoots
}

// WritableRoots() returns the map of writable Keep services:
// url -> ""
func (kc *KeepClient) WritableRoots() map[string]string {
	kc.lock.RLock()
	defer kc.lock.RUnlock()
	return *kc.writableRoots
}

// SetServiceRoots updates the localRoots and gatewayRoots maps,
// without risk of disrupting operations that are already in progress.
//
// The KeepClient makes its own copy of the supplied maps, so the
// caller can reuse/modify them after SetServiceRoots returns, but
// they should not be modified by any other goroutine while
// SetServiceRoots is running.
func (kc *KeepClient) SetServiceRoots(newLocals, newGateways map[string]string, writableRoots map[string]string) {
	locals := make(map[string]string)
	for uuid, root := range newLocals {
		locals[uuid] = root
	}
	gateways := make(map[string]string)
	for uuid, root := range newGateways {
		gateways[uuid] = root
	}
	writables := make(map[string]string)
	for root, _ := range writableRoots {
		writables[root] = ""
	}
	kc.lock.Lock()
	defer kc.lock.Unlock()
	kc.localRoots = &locals
	kc.gatewayRoots = &gateways
	kc.writableRoots = &writables
}

// getSortedRoots returns a list of base URIs of Keep services, in the
// order they should be attempted in order to retrieve content for the
// given locator.
func (kc *KeepClient) getSortedRoots(locator string) []string {
	var found []string
	for _, hint := range strings.Split(locator, "+") {
		if len(hint) < 7 || hint[0:2] != "K@" {
			// Not a service hint.
			continue
		}
		if len(hint) == 7 {
			// +K@abcde means fetch from proxy at
			// keep.abcde.arvadosapi.com
			found = append(found, "https://keep."+hint[2:]+".arvadosapi.com")
		} else if len(hint) == 29 {
			// +K@abcde-abcde-abcdeabcdeabcde means fetch
			// from gateway with given uuid
			if gwURI, ok := kc.GatewayRoots()[hint[2:]]; ok {
				found = append(found, gwURI)
			}
			// else this hint is no use to us; carry on.
		}
	}
	// After trying all usable service hints, fall back to local roots.
	found = append(found, NewRootSorter(kc.LocalRoots(), locator[0:32]).GetSortedRoots()...)
	return found
}

type Locator struct {
	Hash  string
	Size  int      // -1 if data size is not known
	Hints []string // Including the size hint, if any
}

func (loc *Locator) String() string {
	s := loc.Hash
	if len(loc.Hints) > 0 {
		s = s + "+" + strings.Join(loc.Hints, "+")
	}
	return s
}

var locatorMatcher = regexp.MustCompile("^([0-9a-f]{32})([+](.*))?$")

func MakeLocator(path string) (*Locator, error) {
	sm := locatorMatcher.FindStringSubmatch(path)
	if sm == nil {
		return nil, InvalidLocatorError
	}
	loc := Locator{Hash: sm[1], Size: -1}
	if sm[2] != "" {
		loc.Hints = strings.Split(sm[3], "+")
	} else {
		loc.Hints = []string{}
	}
	if len(loc.Hints) > 0 {
		if size, err := strconv.Atoi(loc.Hints[0]); err == nil {
			loc.Size = size
		}
	}
	return &loc, nil
}
