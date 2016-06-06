package arvados

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

// A Client is an HTTP client with an API endpoint and a set of
// Arvados credentials.
//
// It offers methods for accessing individual Arvados APIs, and
// methods that implement common patterns like fetching multiple pages
// of results using List APIs.
type Client struct {
	// HTTP client used to make requests. If nil,
	// http.DefaultClient or InsecureHTTPClient will be used.
	Client *http.Client

	// Hostname (or host:port) of Arvados API server.
	APIHost string

	// User authentication token.
	AuthToken string

	// Accept unverified certificates. This works only if the
	// Client field is nil: otherwise, it has no effect.
	Insecure bool
}

// The default http.Client used by a Client with Insecure==true and
// Client==nil.
var InsecureHTTPClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true}}}

// NewClientFromEnv creates a new Client that uses the default HTTP
// client with the API endpoint and credentials given by the
// ARVADOS_API_* environment variables.
func NewClientFromEnv() *Client {
	return &Client{
		APIHost:   os.Getenv("ARVADOS_API_HOST"),
		AuthToken: os.Getenv("ARVADOS_API_TOKEN"),
		Insecure:  os.Getenv("ARVADOS_API_HOST_INSECURE") != "",
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
		return fmt.Errorf("request failed (%s): %s", req.URL, resp.Status)
	}
	if dst == nil {
		return nil
	}
	return json.Unmarshal(buf, dst)
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
	var urlValues url.Values
	if v, ok := params.(url.Values); ok {
		urlValues = v
	} else if params != nil {
		// Convert an arbitrary struct to url.Values. For
		// example, Foo{Bar: []int{1,2,3}, Baz: "waz"} becomes
		// url.Values{`bar`:`{"a":[1,2,3]}`,`Baz`:`waz`}
		//
		// TODO: Do this more efficiently, possibly using
		// json.Decode/Encode, so the whole thing doesn't have
		// to get encoded, decoded, and re-encoded.
		j, err := json.Marshal(params)
		if err != nil {
			return err
		}
		var generic map[string]interface{}
		err = json.Unmarshal(j, &generic)
		if err != nil {
			return err
		}
		urlValues = url.Values{}
		for k, v := range generic {
			if v, ok := v.(string); ok {
				urlValues.Set(k, v)
				continue
			}
			j, err := json.Marshal(v)
			if err != nil {
				return err
			}
			urlValues.Set(k, string(j))
		}
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
		return http.DefaultClient
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
