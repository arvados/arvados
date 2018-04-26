// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

/* Provides low-level Get/Put primitives for accessing Arvados Keep blocks. */
package keepclient

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvadosclient"
	"git.curoverse.com/arvados.git/sdk/go/asyncbuf"
)

// A Keep "block" is 64MB.
const BLOCKSIZE = 64 * 1024 * 1024

var (
	DefaultRequestTimeout      = 20 * time.Second
	DefaultConnectTimeout      = 2 * time.Second
	DefaultTLSHandshakeTimeout = 4 * time.Second
	DefaultKeepAlive           = 180 * time.Second

	DefaultProxyRequestTimeout      = 300 * time.Second
	DefaultProxyConnectTimeout      = 30 * time.Second
	DefaultProxyTLSHandshakeTimeout = 10 * time.Second
	DefaultProxyKeepAlive           = 120 * time.Second
)

// Error interface with an error and boolean indicating whether the error is temporary
type Error interface {
	error
	Temporary() bool
}

// multipleResponseError is of type Error
type multipleResponseError struct {
	error
	isTemp bool
}

func (e *multipleResponseError) Temporary() bool {
	return e.isTemp
}

// BlockNotFound is a multipleResponseError where isTemp is false
var BlockNotFound = &ErrNotFound{multipleResponseError{
	error:  errors.New("Block not found"),
	isTemp: false,
}}

// ErrNotFound is a multipleResponseError where isTemp can be true or false
type ErrNotFound struct {
	multipleResponseError
}

type InsufficientReplicasError error

type OversizeBlockError error

var ErrOversizeBlock = OversizeBlockError(errors.New("Exceeded maximum block size (" + strconv.Itoa(BLOCKSIZE) + ")"))
var MissingArvadosApiHost = errors.New("Missing required environment variable ARVADOS_API_HOST")
var MissingArvadosApiToken = errors.New("Missing required environment variable ARVADOS_API_TOKEN")
var InvalidLocatorError = errors.New("Invalid locator")

// ErrNoSuchKeepServer is returned when GetIndex is invoked with a UUID with no matching keep server
var ErrNoSuchKeepServer = errors.New("No keep server matching the given UUID is found")

// ErrIncompleteIndex is returned when the Index response does not end with a new empty line
var ErrIncompleteIndex = errors.New("Got incomplete index")

const X_Keep_Desired_Replicas = "X-Keep-Desired-Replicas"
const X_Keep_Replicas_Stored = "X-Keep-Replicas-Stored"

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Information about Arvados and Keep servers.
type KeepClient struct {
	Arvados            *arvadosclient.ArvadosClient
	Want_replicas      int
	localRoots         map[string]string
	writableLocalRoots map[string]string
	gatewayRoots       map[string]string
	lock               sync.RWMutex
	HTTPClient         HTTPClient
	Retries            int
	BlockCache         *BlockCache

	// set to 1 if all writable services are of disk type, otherwise 0
	replicasPerService int

	// Any non-disk typed services found in the list of keepservers?
	foundNonDiskSvc bool

	// Disable automatic discovery of keep services
	disableDiscovery bool
}

// MakeKeepClient creates a new KeepClient, calls
// DiscoverKeepServices(), and returns when the client is ready to
// use.
func MakeKeepClient(arv *arvadosclient.ArvadosClient) (*KeepClient, error) {
	kc := New(arv)
	return kc, kc.discoverServices()
}

// New creates a new KeepClient. Service discovery will occur on the
// next read/write operation.
func New(arv *arvadosclient.ArvadosClient) *KeepClient {
	defaultReplicationLevel := 2
	value, err := arv.Discovery("defaultCollectionReplication")
	if err == nil {
		v, ok := value.(float64)
		if ok && v > 0 {
			defaultReplicationLevel = int(v)
		}
	}
	return &KeepClient{
		Arvados:       arv,
		Want_replicas: defaultReplicationLevel,
		Retries:       2,
	}
}

// Put a block given the block hash, a reader, and the number of bytes
// to read from the reader (which must be between 0 and BLOCKSIZE).
//
// Returns the locator for the written block, the number of replicas
// written, and an error.
//
// Returns an InsufficientReplicasError if 0 <= replicas <
// kc.Wants_replicas.
func (kc *KeepClient) PutHR(hash string, r io.Reader, dataBytes int64) (string, int, error) {
	// Buffer for reads from 'r'
	var bufsize int
	if dataBytes > 0 {
		if dataBytes > BLOCKSIZE {
			return "", 0, ErrOversizeBlock
		}
		bufsize = int(dataBytes)
	} else {
		bufsize = BLOCKSIZE
	}

	buf := asyncbuf.NewBuffer(make([]byte, 0, bufsize))
	go func() {
		_, err := io.Copy(buf, HashCheckingReader{r, md5.New(), hash})
		buf.CloseWithError(err)
	}()
	return kc.putReplicas(hash, buf.NewReader, dataBytes)
}

// PutHB writes a block to Keep. The hash of the bytes is given in
// hash, and the data is given in buf.
//
// Return values are the same as for PutHR.
func (kc *KeepClient) PutHB(hash string, buf []byte) (string, int, error) {
	newReader := func() io.Reader { return bytes.NewBuffer(buf) }
	return kc.putReplicas(hash, newReader, int64(len(buf)))
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

func (kc *KeepClient) getOrHead(method string, locator string) (io.ReadCloser, int64, string, error) {
	if strings.HasPrefix(locator, "d41d8cd98f00b204e9800998ecf8427e+0") {
		return ioutil.NopCloser(bytes.NewReader(nil)), 0, "", nil
	}

	var expectLength int64
	if parts := strings.SplitN(locator, "+", 3); len(parts) < 2 {
		expectLength = -1
	} else if n, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
		expectLength = -1
	} else {
		expectLength = n
	}

	var errs []string

	tries_remaining := 1 + kc.Retries

	serversToTry := kc.getSortedRoots(locator)

	numServers := len(serversToTry)
	count404 := 0

	var retryList []string

	for tries_remaining > 0 {
		tries_remaining -= 1
		retryList = nil

		for _, host := range serversToTry {
			url := host + "/" + locator

			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", url, err))
				continue
			}
			req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", kc.Arvados.ApiToken))
			resp, err := kc.httpClient().Do(req)
			if err != nil {
				// Probably a network error, may be transient,
				// can try again.
				errs = append(errs, fmt.Sprintf("%s: %v", url, err))
				retryList = append(retryList, host)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				var respbody []byte
				respbody, _ = ioutil.ReadAll(&io.LimitedReader{R: resp.Body, N: 4096})
				resp.Body.Close()
				errs = append(errs, fmt.Sprintf("%s: HTTP %d %q",
					url, resp.StatusCode, bytes.TrimSpace(respbody)))

				if resp.StatusCode == 408 ||
					resp.StatusCode == 429 ||
					resp.StatusCode >= 500 {
					// Timeout, too many requests, or other
					// server side failure, transient
					// error, can try again.
					retryList = append(retryList, host)
				} else if resp.StatusCode == 404 {
					count404++
				}
				continue
			}
			if expectLength < 0 {
				if resp.ContentLength < 0 {
					resp.Body.Close()
					return nil, 0, "", fmt.Errorf("error reading %q: no size hint, no Content-Length header in response", locator)
				}
				expectLength = resp.ContentLength
			} else if resp.ContentLength >= 0 && expectLength != resp.ContentLength {
				resp.Body.Close()
				return nil, 0, "", fmt.Errorf("error reading %q: size hint %d != Content-Length %d", locator, expectLength, resp.ContentLength)
			}
			// Success
			if method == "GET" {
				return HashCheckingReader{
					Reader: resp.Body,
					Hash:   md5.New(),
					Check:  locator[0:32],
				}, expectLength, url, nil
			} else {
				resp.Body.Close()
				return nil, expectLength, url, nil
			}
		}
		serversToTry = retryList
	}
	DebugPrintf("DEBUG: %s %s failed: %v", method, locator, errs)

	var err error
	if count404 == numServers {
		err = BlockNotFound
	} else {
		err = &ErrNotFound{multipleResponseError{
			error:  fmt.Errorf("%s %s failed: %v", method, locator, errs),
			isTemp: len(serversToTry) > 0,
		}}
	}
	return nil, 0, "", err
}

// Get() retrieves a block, given a locator. Returns a reader, the
// expected data length, the URL the block is being fetched from, and
// an error.
//
// If the block checksum does not match, the final Read() on the
// reader returned by this method will return a BadChecksum error
// instead of EOF.
func (kc *KeepClient) Get(locator string) (io.ReadCloser, int64, string, error) {
	return kc.getOrHead("GET", locator)
}

// ReadAt() retrieves a portion of block from the cache if it's
// present, otherwise from the network.
func (kc *KeepClient) ReadAt(locator string, p []byte, off int) (int, error) {
	return kc.cache().ReadAt(kc, locator, p, off)
}

// Ask() verifies that a block with the given hash is available and
// readable, according to at least one Keep service. Unlike Get, it
// does not retrieve the data or verify that the data content matches
// the hash specified by the locator.
//
// Returns the data size (content length) reported by the Keep service
// and the URI reporting the data size.
func (kc *KeepClient) Ask(locator string) (int64, string, error) {
	_, size, url, err := kc.getOrHead("HEAD", locator)
	return size, url, err
}

// GetIndex retrieves a list of blocks stored on the given server whose hashes
// begin with the given prefix. The returned reader will return an error (other
// than EOF) if the complete index cannot be retrieved.
//
// This is meant to be used only by system components and admin tools.
// It will return an error unless the client is using a "data manager token"
// recognized by the Keep services.
func (kc *KeepClient) GetIndex(keepServiceUUID, prefix string) (io.Reader, error) {
	url := kc.LocalRoots()[keepServiceUUID]
	if url == "" {
		return nil, ErrNoSuchKeepServer
	}

	url += "/index"
	if prefix != "" {
		url += "/" + prefix
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", kc.Arvados.ApiToken))
	resp, err := kc.httpClient().Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Got http status code: %d", resp.StatusCode)
	}

	var respBody []byte
	respBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Got index; verify that it is complete
	// The response should be "\n" if no locators matched the prefix
	// Else, it should be a list of locators followed by a blank line
	if !bytes.Equal(respBody, []byte("\n")) && !bytes.HasSuffix(respBody, []byte("\n\n")) {
		return nil, ErrIncompleteIndex
	}

	// Got complete index; strip the trailing newline and send
	return bytes.NewReader(respBody[0 : len(respBody)-1]), nil
}

// LocalRoots() returns the map of local (i.e., disk and proxy) Keep
// services: uuid -> baseURI.
func (kc *KeepClient) LocalRoots() map[string]string {
	kc.discoverServices()
	kc.lock.RLock()
	defer kc.lock.RUnlock()
	return kc.localRoots
}

// GatewayRoots() returns the map of Keep remote gateway services:
// uuid -> baseURI.
func (kc *KeepClient) GatewayRoots() map[string]string {
	kc.discoverServices()
	kc.lock.RLock()
	defer kc.lock.RUnlock()
	return kc.gatewayRoots
}

// WritableLocalRoots() returns the map of writable local Keep services:
// uuid -> baseURI.
func (kc *KeepClient) WritableLocalRoots() map[string]string {
	kc.discoverServices()
	kc.lock.RLock()
	defer kc.lock.RUnlock()
	return kc.writableLocalRoots
}

// SetServiceRoots disables service discovery and updates the
// localRoots and gatewayRoots maps, without disrupting operations
// that are already in progress.
//
// The supplied maps must not be modified after calling
// SetServiceRoots.
func (kc *KeepClient) SetServiceRoots(locals, writables, gateways map[string]string) {
	kc.disableDiscovery = true
	kc.setServiceRoots(locals, writables, gateways)
}

func (kc *KeepClient) setServiceRoots(locals, writables, gateways map[string]string) {
	kc.lock.Lock()
	defer kc.lock.Unlock()
	kc.localRoots = locals
	kc.writableLocalRoots = writables
	kc.gatewayRoots = gateways
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

func (kc *KeepClient) cache() *BlockCache {
	if kc.BlockCache != nil {
		return kc.BlockCache
	} else {
		return DefaultBlockCache
	}
}

func (kc *KeepClient) ClearBlockCache() {
	kc.cache().Clear()
}

var (
	// There are four global http.Client objects for the four
	// possible permutations of TLS behavior (verify/skip-verify)
	// and timeout settings (proxy/non-proxy).
	defaultClient = map[bool]map[bool]HTTPClient{
		// defaultClient[false] is used for verified TLS reqs
		false: {},
		// defaultClient[true] is used for unverified
		// (insecure) TLS reqs
		true: {},
	}
	defaultClientMtx sync.Mutex
)

// httpClient returns the HTTPClient field if it's not nil, otherwise
// whichever of the four global http.Client objects is suitable for
// the current environment (i.e., TLS verification on/off, keep
// services are/aren't proxies).
func (kc *KeepClient) httpClient() HTTPClient {
	if kc.HTTPClient != nil {
		return kc.HTTPClient
	}
	defaultClientMtx.Lock()
	defer defaultClientMtx.Unlock()
	if c, ok := defaultClient[kc.Arvados.ApiInsecure][kc.foundNonDiskSvc]; ok {
		return c
	}

	var requestTimeout, connectTimeout, keepAlive, tlsTimeout time.Duration
	if kc.foundNonDiskSvc {
		// Use longer timeouts when connecting to a proxy,
		// because this usually means the intervening network
		// is slower.
		requestTimeout = DefaultProxyRequestTimeout
		connectTimeout = DefaultProxyConnectTimeout
		tlsTimeout = DefaultProxyTLSHandshakeTimeout
		keepAlive = DefaultProxyKeepAlive
	} else {
		requestTimeout = DefaultRequestTimeout
		connectTimeout = DefaultConnectTimeout
		tlsTimeout = DefaultTLSHandshakeTimeout
		keepAlive = DefaultKeepAlive
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if ok {
		copy := *transport
		transport = &copy
	} else {
		// Evidently the application has replaced
		// http.DefaultTransport with a different type, so we
		// need to build our own from scratch using the Go 1.8
		// defaults.
		transport = &http.Transport{
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			ExpectContinueTimeout: time.Second,
		}
	}
	transport.DialContext = (&net.Dialer{
		Timeout:   connectTimeout,
		KeepAlive: keepAlive,
		DualStack: true,
	}).DialContext
	transport.TLSHandshakeTimeout = tlsTimeout
	transport.TLSClientConfig = arvadosclient.MakeTLSConfig(kc.Arvados.ApiInsecure)
	c := &http.Client{
		Timeout:   requestTimeout,
		Transport: transport,
	}
	defaultClient[kc.Arvados.ApiInsecure][kc.foundNonDiskSvc] = c
	return c
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
