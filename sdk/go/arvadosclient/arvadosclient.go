// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

/* Simple Arvados Go SDK for communicating with API server. */

package arvadosclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

type StringMatcher func(string) bool

var UUIDMatch StringMatcher = arvados.UUIDMatch
var PDHMatch StringMatcher = arvados.PDHMatch

var MissingArvadosApiHost = errors.New("Missing required environment variable ARVADOS_API_HOST")
var MissingArvadosApiToken = errors.New("Missing required environment variable ARVADOS_API_TOKEN")
var ErrInvalidArgument = errors.New("Invalid argument")

// A common failure mode is to reuse a keepalive connection that has been
// terminated (in a way that we can't detect) for being idle too long.
// POST and DELETE are not safe to retry automatically, so we minimize
// such failures by always using a new or recently active socket.
var MaxIdleConnectionDuration = 30 * time.Second

var RetryDelay = 2 * time.Second

var (
	defaultInsecureHTTPClient *http.Client
	defaultSecureHTTPClient   *http.Client
	defaultHTTPClientMtx      sync.Mutex
)

// APIServerError contains an error that was returned by the API server.
type APIServerError struct {
	// Address of server returning error, of the form "host:port".
	ServerAddress string

	// Components of server response.
	HttpStatusCode    int
	HttpStatusMessage string

	// Additional error details from response body.
	ErrorDetails []string
}

func (e APIServerError) Error() string {
	if len(e.ErrorDetails) > 0 {
		return fmt.Sprintf("arvados API server error: %s (%d: %s) returned by %s",
			strings.Join(e.ErrorDetails, "; "),
			e.HttpStatusCode,
			e.HttpStatusMessage,
			e.ServerAddress)
	}
	return fmt.Sprintf("arvados API server error: %d: %s returned by %s",
		e.HttpStatusCode,
		e.HttpStatusMessage,
		e.ServerAddress)
}

// StringBool tests whether s is suggestive of true. It returns true
// if s is a mixed/uppoer/lower-case variant of "1", "yes", or "true".
func StringBool(s string) bool {
	s = strings.ToLower(s)
	return s == "1" || s == "yes" || s == "true"
}

// Dict is a helper type so we don't have to write out 'map[string]interface{}' every time.
type Dict map[string]interface{}

// ArvadosClient contains information about how to contact the Arvados server
type ArvadosClient struct {
	// https
	Scheme string

	// Arvados API server, form "host:port"
	ApiServer string

	// Arvados API token for authentication
	ApiToken string

	// Whether to require a valid SSL certificate or not
	ApiInsecure bool

	// Client object shared by client requests.  Supports HTTP KeepAlive.
	Client *http.Client

	// If true, sets the X-External-Client header to indicate
	// the client is outside the cluster.
	External bool

	// Base URIs of Keep services, e.g., {"https://host1:8443",
	// "https://host2:8443"}.  If this is nil, Keep clients will
	// use the arvados.v1.keep_services.accessible API to discover
	// available services.
	KeepServiceURIs []string

	// Discovery document
	DiscoveryDoc Dict

	lastClosedIdlesAt time.Time

	// Number of retries
	Retries int

	// X-Request-Id for outgoing requests
	RequestID string
}

var CertFiles = []string{
	"/etc/arvados/ca-certificates.crt",
	"/etc/ssl/certs/ca-certificates.crt", // Debian/Ubuntu/Gentoo etc.
	"/etc/pki/tls/certs/ca-bundle.crt",   // Fedora/RHEL
}

// MakeTLSConfig sets up TLS configuration for communicating with
// Arvados and Keep services.
func MakeTLSConfig(insecure bool) *tls.Config {
	tlsconfig := tls.Config{InsecureSkipVerify: insecure}

	if !insecure {
		// Use the first entry in CertFiles that we can read
		// certificates from. If none of those work out, use
		// the Go defaults.
		certs := x509.NewCertPool()
		for _, file := range CertFiles {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				if !os.IsNotExist(err) {
					log.Printf("proceeding without loading cert file %q: %s", file, err)
				}
				continue
			}
			if !certs.AppendCertsFromPEM(data) {
				log.Printf("unable to load any certificates from %v", file)
				continue
			}
			tlsconfig.RootCAs = certs
			break
		}
	}

	return &tlsconfig
}

// New returns an ArvadosClient using the given arvados.Client
// configuration. This is useful for callers who load arvados.Client
// fields from configuration files but still need to use the
// arvadosclient.ArvadosClient package.
func New(c *arvados.Client) (*ArvadosClient, error) {
	ac := &ArvadosClient{
		Scheme:      "https",
		ApiServer:   c.APIHost,
		ApiToken:    c.AuthToken,
		ApiInsecure: c.Insecure,
		Client: &http.Client{
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				TLSClientConfig: MakeTLSConfig(c.Insecure)},
		},
		External:          false,
		Retries:           2,
		KeepServiceURIs:   c.KeepServiceURIs,
		lastClosedIdlesAt: time.Now(),
	}

	return ac, nil
}

// MakeArvadosClient creates a new ArvadosClient using the standard
// environment variables ARVADOS_API_HOST, ARVADOS_API_TOKEN,
// ARVADOS_API_HOST_INSECURE, ARVADOS_EXTERNAL_CLIENT, and
// ARVADOS_KEEP_SERVICES.
func MakeArvadosClient() (ac *ArvadosClient, err error) {
	ac, err = New(arvados.NewClientFromEnv())
	if err != nil {
		return
	}
	ac.External = StringBool(os.Getenv("ARVADOS_EXTERNAL_CLIENT"))
	return
}

// CallRaw is the same as Call() but returns a Reader that reads the
// response body, instead of taking an output object.
func (c *ArvadosClient) CallRaw(method string, resourceType string, uuid string, action string, parameters Dict) (reader io.ReadCloser, err error) {
	scheme := c.Scheme
	if scheme == "" {
		scheme = "https"
	}
	if c.ApiServer == "" {
		return nil, fmt.Errorf("Arvados client is not configured (target API host is not set). Maybe env var ARVADOS_API_HOST should be set first?")
	}
	u := url.URL{
		Scheme: scheme,
		Host:   c.ApiServer}

	if resourceType != ApiDiscoveryResource {
		u.Path = "/arvados/v1"
	}

	if resourceType != "" {
		u.Path = u.Path + "/" + resourceType
	}
	if uuid != "" {
		u.Path = u.Path + "/" + uuid
	}
	if action != "" {
		u.Path = u.Path + "/" + action
	}

	if parameters == nil {
		parameters = make(Dict)
	}

	vals := make(url.Values)
	for k, v := range parameters {
		if s, ok := v.(string); ok {
			vals.Set(k, s)
		} else if m, err := json.Marshal(v); err == nil {
			vals.Set(k, string(m))
		}
	}

	retryable := false
	switch method {
	case "GET", "HEAD", "PUT", "OPTIONS", "DELETE":
		retryable = true
	}

	// Non-retryable methods such as POST are not safe to retry automatically,
	// so we minimize such failures by always using a new or recently active socket
	if !retryable {
		if time.Since(c.lastClosedIdlesAt) > MaxIdleConnectionDuration {
			c.lastClosedIdlesAt = time.Now()
			c.Client.Transport.(*http.Transport).CloseIdleConnections()
		}
	}

	// Make the request
	var req *http.Request
	var resp *http.Response

	for attempt := 0; attempt <= c.Retries; attempt++ {
		if method == "GET" || method == "HEAD" {
			u.RawQuery = vals.Encode()
			if req, err = http.NewRequest(method, u.String(), nil); err != nil {
				return nil, err
			}
		} else {
			if req, err = http.NewRequest(method, u.String(), bytes.NewBufferString(vals.Encode())); err != nil {
				return nil, err
			}
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		}

		// Add api token header
		req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", c.ApiToken))
		if c.RequestID != "" {
			req.Header.Add("X-Request-Id", c.RequestID)
		}
		if c.External {
			req.Header.Add("X-External-Client", "1")
		}

		resp, err = c.Client.Do(req)
		if err != nil {
			if retryable {
				time.Sleep(RetryDelay)
				continue
			} else {
				return nil, err
			}
		}

		if resp.StatusCode == http.StatusOK {
			return resp.Body, nil
		}

		defer resp.Body.Close()

		switch resp.StatusCode {
		case 408, 409, 422, 423, 500, 502, 503, 504:
			time.Sleep(RetryDelay)
			continue
		default:
			return nil, newAPIServerError(c.ApiServer, resp)
		}
	}

	if resp != nil {
		return nil, newAPIServerError(c.ApiServer, resp)
	}
	return nil, err
}

func newAPIServerError(ServerAddress string, resp *http.Response) APIServerError {

	ase := APIServerError{
		ServerAddress:     ServerAddress,
		HttpStatusCode:    resp.StatusCode,
		HttpStatusMessage: resp.Status}

	// If the response body has {"errors":["reason1","reason2"]}
	// then return those reasons.
	var errInfo = Dict{}
	if err := json.NewDecoder(resp.Body).Decode(&errInfo); err == nil {
		if errorList, ok := errInfo["errors"]; ok {
			if errArray, ok := errorList.([]interface{}); ok {
				for _, errItem := range errArray {
					// We expect an array of strings here.
					// Non-strings will be passed along
					// JSON-encoded.
					if s, ok := errItem.(string); ok {
						ase.ErrorDetails = append(ase.ErrorDetails, s)
					} else if j, err := json.Marshal(errItem); err == nil {
						ase.ErrorDetails = append(ase.ErrorDetails, string(j))
					}
				}
			}
		}
	}
	return ase
}

// Call an API endpoint and parse the JSON response into an object.
//
//   method - HTTP method: GET, HEAD, PUT, POST, PATCH or DELETE.
//   resourceType - the type of arvados resource to act on (e.g., "collections", "pipeline_instances").
//   uuid - the uuid of the specific item to access. May be empty.
//   action - API method name (e.g., "lock"). This is often empty if implied by method and uuid.
//   parameters - method parameters.
//   output - a map or annotated struct which is a legal target for encoding/json/Decoder.
//
// Returns a non-nil error if an error occurs making the API call, the
// API responds with a non-successful HTTP status, or an error occurs
// parsing the response body.
func (c *ArvadosClient) Call(method, resourceType, uuid, action string, parameters Dict, output interface{}) error {
	reader, err := c.CallRaw(method, resourceType, uuid, action, parameters)
	if reader != nil {
		defer reader.Close()
	}
	if err != nil {
		return err
	}

	if output != nil {
		dec := json.NewDecoder(reader)
		if err = dec.Decode(output); err != nil {
			return err
		}
	}
	return nil
}

// Create a new resource. See Call for argument descriptions.
func (c *ArvadosClient) Create(resourceType string, parameters Dict, output interface{}) error {
	return c.Call("POST", resourceType, "", "", parameters, output)
}

// Delete a resource. See Call for argument descriptions.
func (c *ArvadosClient) Delete(resource string, uuid string, parameters Dict, output interface{}) (err error) {
	return c.Call("DELETE", resource, uuid, "", parameters, output)
}

// Update attributes of a resource. See Call for argument descriptions.
func (c *ArvadosClient) Update(resourceType string, uuid string, parameters Dict, output interface{}) (err error) {
	return c.Call("PUT", resourceType, uuid, "", parameters, output)
}

// Get a resource. See Call for argument descriptions.
func (c *ArvadosClient) Get(resourceType string, uuid string, parameters Dict, output interface{}) (err error) {
	if !UUIDMatch(uuid) && !(resourceType == "collections" && PDHMatch(uuid)) {
		// No object has uuid == "": there is no need to make
		// an API call. Furthermore, the HTTP request for such
		// an API call would be "GET /arvados/v1/type/", which
		// is liable to be misinterpreted as the List API.
		return ErrInvalidArgument
	}
	return c.Call("GET", resourceType, uuid, "", parameters, output)
}

// List resources of a given type. See Call for argument descriptions.
func (c *ArvadosClient) List(resource string, parameters Dict, output interface{}) (err error) {
	return c.Call("GET", resource, "", "", parameters, output)
}

const ApiDiscoveryResource = "discovery/v1/apis/arvados/v1/rest"

// Discovery returns the value of the given parameter in the discovery
// document. Returns a non-nil error if the discovery document cannot
// be retrieved/decoded. Returns ErrInvalidArgument if the requested
// parameter is not found in the discovery document.
func (c *ArvadosClient) Discovery(parameter string) (value interface{}, err error) {
	if len(c.DiscoveryDoc) == 0 {
		c.DiscoveryDoc = make(Dict)
		err = c.Call("GET", ApiDiscoveryResource, "", "", nil, &c.DiscoveryDoc)
		if err != nil {
			return nil, err
		}
	}

	var found bool
	value, found = c.DiscoveryDoc[parameter]
	if found {
		return value, nil
	}
	return value, ErrInvalidArgument
}

// ClusterConfig returns the value of the given key in the current cluster's
// exported config. If key is an empty string, it'll return the entire config.
func (c *ArvadosClient) ClusterConfig(key string) (config interface{}, err error) {
	var clusterConfig interface{}
	err = c.Call("GET", "config", "", "", nil, &clusterConfig)
	if err != nil {
		return nil, err
	}
	if key == "" {
		return clusterConfig, nil
	}
	configData, ok := clusterConfig.(map[string]interface{})[key]
	if !ok {
		return nil, ErrInvalidArgument
	}
	return configData, nil
}

func (c *ArvadosClient) httpClient() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	cl := &defaultSecureHTTPClient
	if c.ApiInsecure {
		cl = &defaultInsecureHTTPClient
	}
	if *cl == nil {
		defaultHTTPClientMtx.Lock()
		defer defaultHTTPClientMtx.Unlock()
		*cl = &http.Client{Transport: &http.Transport{
			TLSClientConfig: MakeTLSConfig(c.ApiInsecure)}}
	}
	return *cl
}
