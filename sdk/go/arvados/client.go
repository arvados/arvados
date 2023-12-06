// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	mathrand "math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/hashicorp/go-retryablehttp"
)

// A Client is an HTTP client with an API endpoint and a set of
// Arvados credentials.
//
// It offers methods for accessing individual Arvados APIs, and
// methods that implement common patterns like fetching multiple pages
// of results using List APIs.
type Client struct {
	// HTTP client used to make requests. If nil,
	// DefaultSecureClient or InsecureHTTPClient will be used.
	Client *http.Client `json:"-"`

	// Protocol scheme: "http", "https", or "" (https)
	Scheme string

	// Hostname (or host:port) of Arvados API server.
	APIHost string

	// User authentication token.
	AuthToken string

	// Accept unverified certificates. This works only if the
	// Client field is nil: otherwise, it has no effect.
	Insecure bool

	// Override keep service discovery with a list of base
	// URIs. (Currently there are no Client methods for
	// discovering keep services so this is just a convenience for
	// callers who use a Client to initialize an
	// arvadosclient.ArvadosClient.)
	KeepServiceURIs []string `json:",omitempty"`

	// HTTP headers to add/override in outgoing requests.
	SendHeader http.Header

	// Timeout for requests. NewClientFromConfig and
	// NewClientFromEnv return a Client with a default 5 minute
	// timeout. Within this time, retryable errors are
	// automatically retried with exponential backoff.
	//
	// To disable automatic retries, set Timeout to zero and use a
	// context deadline to establish a maximum request time.
	Timeout time.Duration

	dd *DiscoveryDocument

	defaultRequestID string

	// APIHost and AuthToken were loaded from ARVADOS_* env vars
	// (used to customize "no host/token" error messages)
	loadedFromEnv bool

	// Track/limit concurrent outgoing API calls. Note this
	// differs from an outgoing connection limit (a feature
	// provided by http.Transport) when concurrent calls are
	// multiplexed on a single http2 connection.
	//
	// getRequestLimiter() should always be used, because this can
	// be nil.
	requestLimiter *requestLimiter

	last503 atomic.Value
}

// InsecureHTTPClient is the default http.Client used by a Client with
// Insecure==true and Client==nil.
var InsecureHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true}}}

// DefaultSecureClient is the default http.Client used by a Client otherwise.
var DefaultSecureClient = &http.Client{}

// NewClientFromConfig creates a new Client that uses the endpoints in
// the given cluster.
//
// AuthToken is left empty for the caller to populate.
func NewClientFromConfig(cluster *Cluster) (*Client, error) {
	ctrlURL := cluster.Services.Controller.ExternalURL
	if ctrlURL.Host == "" {
		return nil, fmt.Errorf("no host in config Services.Controller.ExternalURL: %v", ctrlURL)
	}
	var hc *http.Client
	if srvaddr := os.Getenv("ARVADOS_SERVER_ADDRESS"); srvaddr != "" {
		// When this client is used to make a request to
		// https://{ctrlhost}:port/ (any port), it dials the
		// indicated port on ARVADOS_SERVER_ADDRESS instead.
		//
		// This is invoked by arvados-server boot to ensure
		// that server->server traffic (e.g.,
		// keepproxy->controller) only hits local interfaces,
		// even if the Controller.ExternalURL host is a load
		// balancer / gateway and not a local interface
		// address (e.g., when running on a cloud VM).
		//
		// This avoids unnecessary delay/cost of routing
		// external traffic, and also allows controller to
		// recognize other services as internal clients based
		// on the connection source address.
		divertedHost := (*url.URL)(&cluster.Services.Controller.ExternalURL).Hostname()
		var dialer net.Dialer
		hc = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: cluster.TLS.Insecure},
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					host, port, err := net.SplitHostPort(addr)
					if err == nil && network == "tcp" && host == divertedHost {
						addr = net.JoinHostPort(srvaddr, port)
					}
					return dialer.DialContext(ctx, network, addr)
				},
			},
		}
	}
	return &Client{
		Client:         hc,
		Scheme:         ctrlURL.Scheme,
		APIHost:        ctrlURL.Host,
		Insecure:       cluster.TLS.Insecure,
		Timeout:        5 * time.Minute,
		requestLimiter: &requestLimiter{maxlimit: int64(cluster.API.MaxConcurrentRequests / 4)},
	}, nil
}

// NewClientFromEnv creates a new Client that uses the default HTTP
// client, and loads API endpoint and credentials from ARVADOS_*
// environment variables (if set) and
// $HOME/.config/arvados/settings.conf (if readable).
//
// If a config exists in both locations, the environment variable is
// used.
//
// If there is an error (other than ENOENT) reading settings.conf,
// NewClientFromEnv logs the error to log.Default(), then proceeds as
// if settings.conf did not exist.
//
// Space characters are trimmed when reading the settings file, so
// these are equivalent:
//
//	ARVADOS_API_HOST=localhost\n
//	ARVADOS_API_HOST=localhost\r\n
//	ARVADOS_API_HOST = localhost \n
//	\tARVADOS_API_HOST = localhost\n
func NewClientFromEnv() *Client {
	vars := map[string]string{}
	home := os.Getenv("HOME")
	conffile := home + "/.config/arvados/settings.conf"
	if home == "" {
		// no $HOME => just use env vars
	} else if settings, err := os.ReadFile(conffile); errors.Is(err, fs.ErrNotExist) {
		// no config file => just use env vars
	} else if err != nil {
		// config file unreadable => log message, then use env vars
		log.Printf("continuing without loading %s: %s", conffile, err)
	} else {
		for _, line := range bytes.Split(settings, []byte{'\n'}) {
			kv := bytes.SplitN(line, []byte{'='}, 2)
			k := string(bytes.TrimSpace(kv[0]))
			if len(kv) != 2 || !strings.HasPrefix(k, "ARVADOS_") {
				// Same behavior as python sdk:
				// silently skip leading # (comments),
				// blank lines, typos, and non-Arvados
				// vars.
				continue
			}
			vars[k] = string(bytes.TrimSpace(kv[1]))
		}
	}
	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, "ARVADOS_") {
			continue
		}
		kv := strings.SplitN(env, "=", 2)
		if len(kv) == 2 {
			vars[kv[0]] = kv[1]
		}
	}
	var svcs []string
	for _, s := range strings.Split(vars["ARVADOS_KEEP_SERVICES"], " ") {
		if s == "" {
			continue
		} else if u, err := url.Parse(s); err != nil {
			log.Printf("ARVADOS_KEEP_SERVICES: %q: %s", s, err)
		} else if !u.IsAbs() {
			log.Printf("ARVADOS_KEEP_SERVICES: %q: not an absolute URI", s)
		} else {
			svcs = append(svcs, s)
		}
	}
	var insecure bool
	if s := strings.ToLower(vars["ARVADOS_API_HOST_INSECURE"]); s == "1" || s == "yes" || s == "true" {
		insecure = true
	}
	return &Client{
		Scheme:          "https",
		APIHost:         vars["ARVADOS_API_HOST"],
		AuthToken:       vars["ARVADOS_API_TOKEN"],
		Insecure:        insecure,
		KeepServiceURIs: svcs,
		Timeout:         5 * time.Minute,
		loadedFromEnv:   true,
	}
}

var reqIDGen = httpserver.IDGenerator{Prefix: "req-"}

var nopCancelFunc context.CancelFunc = func() {}

var reqErrorRe = regexp.MustCompile(`net/http: invalid header `)

// Do augments (*http.Client)Do(): adds Authorization and X-Request-Id
// headers, delays in order to comply with rate-limiting restrictions,
// and retries failed requests when appropriate.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if auth, _ := ctx.Value(contextKeyAuthorization{}).(string); auth != "" {
		req.Header.Add("Authorization", auth)
	} else if c.AuthToken != "" {
		req.Header.Add("Authorization", "OAuth2 "+c.AuthToken)
	}

	if req.Header.Get("X-Request-Id") == "" {
		var reqid string
		if ctxreqid, _ := ctx.Value(contextKeyRequestID{}).(string); ctxreqid != "" {
			reqid = ctxreqid
		} else if c.defaultRequestID != "" {
			reqid = c.defaultRequestID
		} else {
			reqid = reqIDGen.Next()
		}
		if req.Header == nil {
			req.Header = http.Header{"X-Request-Id": {reqid}}
		} else {
			req.Header.Set("X-Request-Id", reqid)
		}
	}

	rreq, err := retryablehttp.FromRequest(req)
	if err != nil {
		return nil, err
	}

	cancel := nopCancelFunc
	var lastResp *http.Response
	var lastRespBody io.ReadCloser
	var lastErr error
	var checkRetryCalled int

	rclient := retryablehttp.NewClient()
	rclient.HTTPClient = c.httpClient()
	rclient.Backoff = exponentialBackoff
	if c.Timeout > 0 {
		rclient.RetryWaitMax = c.Timeout / 10
		rclient.RetryMax = 32
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(c.Timeout))
		rreq = rreq.WithContext(ctx)
	} else {
		rclient.RetryMax = 0
	}
	rclient.CheckRetry = func(ctx context.Context, resp *http.Response, respErr error) (bool, error) {
		checkRetryCalled++
		if c.getRequestLimiter().Report(resp, respErr) {
			c.last503.Store(time.Now())
		}
		if c.Timeout == 0 {
			return false, nil
		}
		if respErr != nil && reqErrorRe.MatchString(respErr.Error()) {
			return false, nil
		}
		retrying, err := retryablehttp.DefaultRetryPolicy(ctx, resp, respErr)
		if retrying {
			lastResp, lastRespBody, lastErr = resp, nil, respErr
			if respErr == nil {
				// Save the response and body so we
				// can return it instead of "deadline
				// exceeded". retryablehttp.Client
				// will drain and discard resp.body,
				// so we need to stash it separately.
				buf, err := ioutil.ReadAll(resp.Body)
				if err == nil {
					lastRespBody = io.NopCloser(bytes.NewReader(buf))
				} else {
					lastResp, lastErr = nil, err
				}
			}
		}
		return retrying, err
	}
	rclient.Logger = nil

	limiter := c.getRequestLimiter()
	limiter.Acquire(ctx)
	if ctx.Err() != nil {
		limiter.Release()
		cancel()
		return nil, ctx.Err()
	}
	resp, err := rclient.Do(rreq)
	if (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) && (lastResp != nil || lastErr != nil) {
		resp = lastResp
		err = lastErr
		if checkRetryCalled > 0 && err != nil {
			// Mimic retryablehttp's "giving up after X
			// attempts" message, even if we gave up
			// because of time rather than maxretries.
			err = fmt.Errorf("%s %s giving up after %d attempt(s): %w", req.Method, req.URL.String(), checkRetryCalled, err)
		}
		if resp != nil {
			resp.Body = lastRespBody
		}
	}
	if err != nil {
		limiter.Release()
		cancel()
		return nil, err
	}
	// We need to call cancel() eventually, but we can't use
	// "defer cancel()" because the context has to stay alive
	// until the caller has finished reading the response body.
	resp.Body = cancelOnClose{
		ReadCloser: resp.Body,
		cancel: func() {
			limiter.Release()
			cancel()
		},
	}
	return resp, err
}

// Last503 returns the time of the most recent HTTP 503 (Service
// Unavailable) response. Zero time indicates never.
func (c *Client) Last503() time.Time {
	t, _ := c.last503.Load().(time.Time)
	return t
}

// globalRequestLimiter entries (one for each APIHost) don't have a
// hard limit on outgoing connections, but do add a delay and reduce
// concurrency after 503 errors.
var (
	globalRequestLimiter     = map[string]*requestLimiter{}
	globalRequestLimiterLock sync.Mutex
)

// Get this client's requestLimiter, or a global requestLimiter
// singleton for c's APIHost if this client doesn't have its own.
func (c *Client) getRequestLimiter() *requestLimiter {
	if c.requestLimiter != nil {
		return c.requestLimiter
	}
	globalRequestLimiterLock.Lock()
	defer globalRequestLimiterLock.Unlock()
	limiter := globalRequestLimiter[c.APIHost]
	if limiter == nil {
		limiter = &requestLimiter{}
		globalRequestLimiter[c.APIHost] = limiter
	}
	return limiter
}

// cancelOnClose calls a provided CancelFunc when its wrapped
// ReadCloser's Close() method is called.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (coc cancelOnClose) Close() error {
	err := coc.ReadCloser.Close()
	coc.cancel()
	return err
}

func isRedirectStatus(code int) bool {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

const minExponentialBackoffBase = time.Second

// Implements retryablehttp.Backoff using the server-provided
// Retry-After header if available, otherwise nearly-full jitter
// exponential backoff (similar to
// https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/),
// in all cases respecting the provided min and max.
func exponentialBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	if attemptNum > 0 && min < minExponentialBackoffBase {
		min = minExponentialBackoffBase
	}
	var t time.Duration
	if resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) {
		if s := resp.Header.Get("Retry-After"); s != "" {
			if sleep, err := strconv.ParseInt(s, 10, 64); err == nil {
				t = time.Second * time.Duration(sleep)
			} else if stamp, err := time.Parse(time.RFC1123, s); err == nil {
				t = stamp.Sub(time.Now())
			}
		}
	}
	if t == 0 {
		jitter := mathrand.New(mathrand.NewSource(int64(time.Now().Nanosecond()))).Float64()
		t = min + time.Duration((math.Pow(2, float64(attemptNum))*float64(min)-float64(min))*jitter)
	}
	if t < min {
		return min
	} else if t > max {
		return max
	} else {
		return t
	}
}

// DoAndDecode performs req and unmarshals the response (which must be
// JSON) into dst. Use this instead of RequestAndDecode if you need
// more control of the http.Request object.
//
// If the response status indicates an HTTP redirect, the Location
// header value is unmarshalled to dst as a RedirectLocation
// key/field.
func (c *Client) DoAndDecode(dst interface{}, req *http.Request) error {
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	switch {
	case resp.StatusCode == http.StatusNoContent:
		return nil
	case resp.StatusCode == http.StatusOK && dst == nil:
		return nil
	case resp.StatusCode == http.StatusOK:
		return json.Unmarshal(buf, dst)

	// If the caller uses a client with a custom CheckRedirect
	// func, Do() might return the 3xx response instead of
	// following it.
	case isRedirectStatus(resp.StatusCode) && dst == nil:
		return nil
	case isRedirectStatus(resp.StatusCode):
		// Copy the redirect target URL to dst.RedirectLocation.
		buf, err := json.Marshal(map[string]string{"redirect_location": resp.Header.Get("Location")})
		if err != nil {
			return err
		}
		return json.Unmarshal(buf, dst)

	default:
		return newTransactionError(req, resp, buf)
	}
}

// Convert an arbitrary struct to url.Values. For example,
//
//	Foo{Bar: []int{1,2,3}, Baz: "waz"}
//
// becomes
//
//	url.Values{`bar`:`{"a":[1,2,3]}`,`Baz`:`waz`}
//
// params itself is returned if it is already an url.Values.
func anythingToValues(params interface{}) (url.Values, error) {
	if v, ok := params.(url.Values); ok {
		return v, nil
	}
	// TODO: Do this more efficiently, possibly using
	// json.Decode/Encode, so the whole thing doesn't have to get
	// encoded, decoded, and re-encoded.
	j, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	var generic map[string]interface{}
	dec := json.NewDecoder(bytes.NewBuffer(j))
	dec.UseNumber()
	err = dec.Decode(&generic)
	if err != nil {
		return nil, err
	}
	urlValues := url.Values{}
	for k, v := range generic {
		if v, ok := v.(string); ok {
			urlValues.Set(k, v)
			continue
		}
		if v, ok := v.(json.Number); ok {
			urlValues.Set(k, v.String())
			continue
		}
		if v, ok := v.(bool); ok {
			if v {
				urlValues.Set(k, "true")
			} else {
				// "foo=false", "foo=0", and "foo="
				// are all taken as true strings, so
				// don't send false values at all --
				// rely on the default being false.
			}
			continue
		}
		j, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(j, []byte("null")) {
			// don't add it to urlValues at all
			continue
		}
		urlValues.Set(k, string(j))
	}
	return urlValues, nil
}

// RequestAndDecode performs an API request and unmarshals the
// response (which must be JSON) into dst. Method and body arguments
// are the same as for http.NewRequest(). The given path is added to
// the server's scheme/host/port to form the request URL. The given
// params are passed via POST form or query string.
//
// path must not contain a query string.
func (c *Client) RequestAndDecode(dst interface{}, method, path string, body io.Reader, params interface{}) error {
	return c.RequestAndDecodeContext(context.Background(), dst, method, path, body, params)
}

// RequestAndDecodeContext does the same as RequestAndDecode, but with a context
func (c *Client) RequestAndDecodeContext(ctx context.Context, dst interface{}, method, path string, body io.Reader, params interface{}) error {
	if body, ok := body.(io.Closer); ok {
		// Ensure body is closed even if we error out early
		defer body.Close()
	}
	if c.APIHost == "" {
		if c.loadedFromEnv {
			return errors.New("ARVADOS_API_HOST and/or ARVADOS_API_TOKEN environment variables are not set")
		}
		return errors.New("arvados.Client cannot perform request: APIHost is not set")
	}
	urlString := c.apiURL(path)
	urlValues, err := anythingToValues(params)
	if err != nil {
		return err
	}
	if dst == nil {
		if urlValues == nil {
			urlValues = url.Values{}
		}
		urlValues["select"] = []string{`["uuid"]`}
	}
	if urlValues == nil {
		// Nothing to send
	} else if body != nil || ((method == "GET" || method == "HEAD") && len(urlValues.Encode()) < 1000) {
		// Send params in query part of URL
		u, err := url.Parse(urlString)
		if err != nil {
			return err
		}
		u.RawQuery = urlValues.Encode()
		urlString = u.String()
	} else {
		body = strings.NewReader(urlValues.Encode())
	}
	req, err := http.NewRequest(method, urlString, body)
	if err != nil {
		return err
	}
	if (method == "GET" || method == "HEAD") && body != nil {
		req.Header.Set("X-Http-Method-Override", method)
		req.Method = "POST"
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
	for k, v := range c.SendHeader {
		req.Header[k] = v
	}
	return c.DoAndDecode(dst, req)
}

type resource interface {
	resourceName() string
}

// UpdateBody returns an io.Reader suitable for use as an http.Request
// Body for a create or update API call.
func (c *Client) UpdateBody(rsc resource) io.Reader {
	j, err := json.Marshal(rsc)
	if err != nil {
		// Return a reader that returns errors.
		r, w := io.Pipe()
		w.CloseWithError(err)
		return r
	}
	v := url.Values{rsc.resourceName(): {string(j)}}
	return bytes.NewBufferString(v.Encode())
}

// WithRequestID returns a new shallow copy of c that sends the given
// X-Request-Id value (instead of a new randomly generated one) with
// each subsequent request that doesn't provide its own via context or
// header.
func (c *Client) WithRequestID(reqid string) *Client {
	cc := *c
	cc.defaultRequestID = reqid
	return &cc
}

func (c *Client) httpClient() *http.Client {
	switch {
	case c.Client != nil:
		return c.Client
	case c.Insecure:
		return InsecureHTTPClient
	default:
		return DefaultSecureClient
	}
}

func (c *Client) apiURL(path string) string {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "https"
	}
	// Double-slash in URLs tend to cause subtle hidden problems
	// (e.g., they can behave differently when a load balancer is
	// in the picture). Here we ensure exactly one "/" regardless
	// of whether the given APIHost or path has a superfluous one.
	return scheme + "://" + strings.TrimSuffix(c.APIHost, "/") + "/" + strings.TrimPrefix(path, "/")
}

// DiscoveryDocument is the Arvados server's description of itself.
type DiscoveryDocument struct {
	BasePath                     string              `json:"basePath"`
	DefaultCollectionReplication int                 `json:"defaultCollectionReplication"`
	BlobSignatureTTL             int64               `json:"blobSignatureTtl"`
	GitURL                       string              `json:"gitUrl"`
	Schemas                      map[string]Schema   `json:"schemas"`
	Resources                    map[string]Resource `json:"resources"`
	Revision                     string              `json:"revision"`
}

type Resource struct {
	Methods map[string]ResourceMethod `json:"methods"`
}

type ResourceMethod struct {
	HTTPMethod string         `json:"httpMethod"`
	Path       string         `json:"path"`
	Response   MethodResponse `json:"response"`
}

type MethodResponse struct {
	Ref string `json:"$ref"`
}

type Schema struct {
	UUIDPrefix string `json:"uuidPrefix"`
}

// DiscoveryDocument returns a *DiscoveryDocument. The returned object
// should not be modified: the same object may be returned by
// subsequent calls.
func (c *Client) DiscoveryDocument() (*DiscoveryDocument, error) {
	if c.dd != nil {
		return c.dd, nil
	}
	var dd DiscoveryDocument
	err := c.RequestAndDecode(&dd, "GET", "discovery/v1/apis/arvados/v1/rest", nil, nil)
	if err != nil {
		return nil, err
	}
	c.dd = &dd
	return c.dd, nil
}

var pdhRegexp = regexp.MustCompile(`^[0-9a-f]{32}\+\d+$`)

func (c *Client) modelForUUID(dd *DiscoveryDocument, uuid string) (string, error) {
	if pdhRegexp.MatchString(uuid) {
		return "Collection", nil
	}
	if len(uuid) != 27 {
		return "", fmt.Errorf("invalid UUID: %q", uuid)
	}
	infix := uuid[6:11]
	var model string
	for m, s := range dd.Schemas {
		if s.UUIDPrefix == infix {
			model = m
			break
		}
	}
	if model == "" {
		return "", fmt.Errorf("unrecognized type portion %q in UUID %q", infix, uuid)
	}
	return model, nil
}

func (c *Client) KindForUUID(uuid string) (string, error) {
	dd, err := c.DiscoveryDocument()
	if err != nil {
		return "", err
	}
	model, err := c.modelForUUID(dd, uuid)
	if err != nil {
		return "", err
	}
	return "arvados#" + strings.ToLower(model[:1]) + model[1:], nil
}

func (c *Client) PathForUUID(method, uuid string) (string, error) {
	dd, err := c.DiscoveryDocument()
	if err != nil {
		return "", err
	}
	model, err := c.modelForUUID(dd, uuid)
	if err != nil {
		return "", err
	}
	var resource string
	for r, rsc := range dd.Resources {
		if rsc.Methods["get"].Response.Ref == model {
			resource = r
			break
		}
	}
	if resource == "" {
		return "", fmt.Errorf("no resource for model: %q", model)
	}
	m, ok := dd.Resources[resource].Methods[method]
	if !ok {
		return "", fmt.Errorf("no method %q for resource %q", method, resource)
	}
	path := dd.BasePath + strings.Replace(m.Path, "{uuid}", uuid, -1)
	if path[0] == '/' {
		path = path[1:]
	}
	return path, nil
}

var maxUUIDInt = (&big.Int{}).Exp(big.NewInt(36), big.NewInt(15), nil)

func RandomUUID(clusterID, infix string) string {
	n, err := rand.Int(rand.Reader, maxUUIDInt)
	if err != nil {
		panic(err)
	}
	nstr := n.Text(36)
	for len(nstr) < 15 {
		nstr = "0" + nstr
	}
	return clusterID + "-" + infix + "-" + nstr
}
