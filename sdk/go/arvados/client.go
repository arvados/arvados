package arvados

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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
	Client *http.Client

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
	KeepServiceURIs []string
}

// The default http.Client used by a Client with Insecure==true and
// Client==nil.
var InsecureHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true}},
	Timeout: 5 * time.Minute}

// The default http.Client used by a Client otherwise.
var DefaultSecureClient = &http.Client{
	Timeout: 5 * time.Minute}

// NewClientFromEnv creates a new Client that uses the default HTTP
// client with the API endpoint and credentials given by the
// ARVADOS_API_* environment variables.
func NewClientFromEnv() *Client {
	var svcs []string
	if s := os.Getenv("ARVADOS_KEEP_SERVICES"); s != "" {
		svcs = strings.Split(s, " ")
	}
	return &Client{
		APIHost:         os.Getenv("ARVADOS_API_HOST"),
		AuthToken:       os.Getenv("ARVADOS_API_TOKEN"),
		Insecure:        os.Getenv("ARVADOS_API_HOST_INSECURE") != "",
		KeepServiceURIs: svcs,
	}
}

// Do adds authentication headers and then calls (*http.Client)Do().
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.AuthToken != "" {
		req.Header.Add("Authorization", "OAuth2 "+c.AuthToken)
	}
	return c.httpClient().Do(req)
}

// DoAndDecode performs req and unmarshals the response (which must be
// JSON) into dst. Use this instead of RequestAndDecode if you need
// more control of the http.Request object.
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
	if resp.StatusCode != 200 {
		return newTransactionError(req, resp, buf)
	}
	if dst == nil {
		return nil
	}
	return json.Unmarshal(buf, dst)
}

// Convert an arbitrary struct to url.Values. For example,
//
//     Foo{Bar: []int{1,2,3}, Baz: "waz"}
//
// becomes
//
//     url.Values{`bar`:`{"a":[1,2,3]}`,`Baz`:`waz`}
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
	err = json.Unmarshal(j, &generic)
	if err != nil {
		return nil, err
	}
	urlValues := url.Values{}
	for k, v := range generic {
		if v, ok := v.(string); ok {
			urlValues.Set(k, v)
			continue
		}
		if v, ok := v.(float64); ok {
			// Unmarshal decodes all numbers as float64,
			// which can be written as 1.2345e4 in JSON,
			// but this form is not accepted for ints in
			// url params. If a number fits in an int64,
			// encode it as int64 rather than float64.
			if v, frac := math.Modf(v); frac == 0 && v <= math.MaxInt64 && v >= math.MinInt64 {
				urlValues.Set(k, fmt.Sprintf("%d", int64(v)))
				continue
			}
		}
		j, err := json.Marshal(v)
		if err != nil {
			return nil, err
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
	urlString := c.apiURL(path)
	urlValues, err := anythingToValues(params)
	if err != nil {
		return err
	}
	if (method == "GET" || body != nil) && urlValues != nil {
		// FIXME: what if params don't fit in URL
		u, err := url.Parse(urlString)
		if err != nil {
			return err
		}
		u.RawQuery = urlValues.Encode()
		urlString = u.String()
	}
	req, err := http.NewRequest(method, urlString, body)
	if err != nil {
		return err
	}
	return c.DoAndDecode(dst, req)
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
	return "https://" + c.APIHost + "/" + path
}

// DiscoveryDocument is the Arvados server's description of itself.
type DiscoveryDocument struct {
	DefaultCollectionReplication int   `json:"defaultCollectionReplication"`
	BlobSignatureTTL             int64 `json:"blobSignatureTtl"`
}

// DiscoveryDocument returns a *DiscoveryDocument. The returned object
// should not be modified: the same object may be returned by
// subsequent calls.
func (c *Client) DiscoveryDocument() (*DiscoveryDocument, error) {
	var dd DiscoveryDocument
	return &dd, c.RequestAndDecode(&dd, "GET", "discovery/v1/apis/arvados/v1/rest", nil, nil)
}
