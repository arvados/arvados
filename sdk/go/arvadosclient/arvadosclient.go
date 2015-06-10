/* Simple Arvados Go SDK for communicating with API server. */

package arvadosclient

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

// Errors
var MissingArvadosApiHost = errors.New("Missing required environment variable ARVADOS_API_HOST")
var MissingArvadosApiToken = errors.New("Missing required environment variable ARVADOS_API_TOKEN")

// Indicates an error that was returned by the API server.
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
	} else {
		return fmt.Sprintf("arvados API server error: %d: %s returned by %s",
			e.HttpStatusCode,
			e.HttpStatusMessage,
			e.ServerAddress)
	}
}

// Helper type so we don't have to write out 'map[string]interface{}' every time.
type Dict map[string]interface{}

// Information about how to contact the Arvados server
type ArvadosClient struct {
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

	// Discovery document
	DiscoveryDoc Dict
}

// Create a new ArvadosClient, initialized with standard Arvados environment
// variables ARVADOS_API_HOST, ARVADOS_API_TOKEN, and (optionally)
// ARVADOS_API_HOST_INSECURE.
func MakeArvadosClient() (ac ArvadosClient, err error) {
	var matchTrue = regexp.MustCompile("^(?i:1|yes|true)$")
	insecure := matchTrue.MatchString(os.Getenv("ARVADOS_API_HOST_INSECURE"))
	external := matchTrue.MatchString(os.Getenv("ARVADOS_EXTERNAL_CLIENT"))

	ac = ArvadosClient{
		ApiServer:   os.Getenv("ARVADOS_API_HOST"),
		ApiToken:    os.Getenv("ARVADOS_API_TOKEN"),
		ApiInsecure: insecure,
		Client: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}}},
		External: external}

	if ac.ApiServer == "" {
		return ac, MissingArvadosApiHost
	}
	if ac.ApiToken == "" {
		return ac, MissingArvadosApiToken
	}

	return ac, err
}

// Low-level access to a resource.
//
//   method - HTTP method, one of GET, HEAD, PUT, POST or DELETE
//   resource - the arvados resource to act on
//   uuid - the uuid of the specific item to access (may be empty)
//   action - sub-action to take on the resource or uuid (may be empty)
//   parameters - method parameters
//
// return
//   reader - the body reader, or nil if there was an error
//   err - error accessing the resource, or nil if no error
func (this ArvadosClient) CallRaw(method string, resource string, uuid string, action string, parameters Dict) (reader io.ReadCloser, err error) {
	var req *http.Request

	u := url.URL{
		Scheme: "https",
		Host:   this.ApiServer}

	if resource != API_DISCOVERY_RESOURCE {
		u.Path = "/arvados/v1"
	}

	if resource != "" {
		u.Path = u.Path + "/" + resource
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

	parameters["format"] = "json"

	vals := make(url.Values)
	for k, v := range parameters {
		m, err := json.Marshal(v)
		if err == nil {
			vals.Set(k, string(m))
		}
	}

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
	req.Header.Add("Authorization", fmt.Sprintf("OAuth2 %s", this.ApiToken))
	if this.External {
		req.Header.Add("X-External-Client", "1")
	}

	// Make the request
	var resp *http.Response
	if resp, err = this.Client.Do(req); err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}

	defer resp.Body.Close()
	return nil, newAPIServerError(this.ApiServer, resp)
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

// Access to a resource.
//
//   method - HTTP method, one of GET, HEAD, PUT, POST or DELETE
//   resource - the arvados resource to act on
//   uuid - the uuid of the specific item to access (may be empty)
//   action - sub-action to take on the resource or uuid (may be empty)
//   parameters - method parameters
//   output - a map or annotated struct which is a legal target for encoding/json/Decoder
// return
//   err - error accessing the resource, or nil if no error
func (this ArvadosClient) Call(method string, resource string, uuid string, action string, parameters Dict, output interface{}) (err error) {
	var reader io.ReadCloser
	reader, err = this.CallRaw(method, resource, uuid, action, parameters)
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

// Create a new instance of a resource.
//
//   resource - the arvados resource on which to create an item
//   parameters - method parameters
//   output - a map or annotated struct which is a legal target for encoding/json/Decoder
// return
//   err - error accessing the resource, or nil if no error
func (this ArvadosClient) Create(resource string, parameters Dict, output interface{}) (err error) {
	return this.Call("POST", resource, "", "", parameters, output)
}

// Delete an instance of a resource.
//
//   resource - the arvados resource on which to delete an item
//   uuid - the item to delete
//   parameters - method parameters
//   output - a map or annotated struct which is a legal target for encoding/json/Decoder
// return
//   err - error accessing the resource, or nil if no error
func (this ArvadosClient) Delete(resource string, uuid string, parameters Dict, output interface{}) (err error) {
	return this.Call("DELETE", resource, uuid, "", parameters, output)
}

// Update fields of an instance of a resource.
//
//   resource - the arvados resource on which to update the item
//   uuid - the item to update
//   parameters - method parameters
//   output - a map or annotated struct which is a legal target for encoding/json/Decoder
// return
//   err - error accessing the resource, or nil if no error
func (this ArvadosClient) Update(resource string, uuid string, parameters Dict, output interface{}) (err error) {
	return this.Call("PUT", resource, uuid, "", parameters, output)
}

// List the instances of a resource
//
//   resource - the arvados resource on which to list
//   parameters - method parameters
//   output - a map or annotated struct which is a legal target for encoding/json/Decoder
// return
//   err - error accessing the resource, or nil if no error
func (this ArvadosClient) List(resource string, parameters Dict, output interface{}) (err error) {
	return this.Call("GET", resource, "", "", parameters, output)
}

// API Discovery
//
//   parameter - name of parameter to be discovered
// return
//   valueMap - Dict key value pair of the discovered parameter
//   err - error accessing the resource, or nil if no error
var API_DISCOVERY_RESOURCE string = "discovery/v1/apis/arvados/v1/rest"

func (this *ArvadosClient) Discovery(parameter string) (value interface{}, err error) {
	if len(this.DiscoveryDoc) == 0 {
		this.DiscoveryDoc = make(Dict)
		err = this.Call("GET", API_DISCOVERY_RESOURCE, "", "", nil, &this.DiscoveryDoc)
		if err != nil {
			return nil, err
		}
	}

	var found bool
	value, found = this.DiscoveryDoc[parameter]
	if found {
		return value, nil
	} else {
		return value, errors.New("Not found")
	}
}
