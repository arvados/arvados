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
	"strings"
)

// Errors
var MissingArvadosApiHost = errors.New("Missing required environment variable ARVADOS_API_HOST")
var MissingArvadosApiToken = errors.New("Missing required environment variable ARVADOS_API_TOKEN")

type ArvadosApiError struct {
	error
	HttpStatusCode int
	HttpStatus string
}

func (e ArvadosApiError) Error() string { return e.error.Error() }

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
}

// Create a new KeepClient, initialized with standard Arvados environment
// variables ARVADOS_API_HOST, ARVADOS_API_TOKEN, and (optionally)
// ARVADOS_API_HOST_INSECURE.
func MakeArvadosClient() (kc ArvadosClient, err error) {
	insecure := (os.Getenv("ARVADOS_API_HOST_INSECURE") == "true")
	external := (os.Getenv("ARVADOS_EXTERNAL_CLIENT") == "true")

	kc = ArvadosClient{
		ApiServer:   os.Getenv("ARVADOS_API_HOST"),
		ApiToken:    os.Getenv("ARVADOS_API_TOKEN"),
		ApiInsecure: insecure,
		Client: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}}},
		External: external}

	if kc.ApiServer == "" {
		return kc, MissingArvadosApiHost
	}
	if kc.ApiToken == "" {
		return kc, MissingArvadosApiToken
	}

	return kc, err
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

	u.Path = "/arvados/v1"

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
	errorText := fmt.Sprintf("API response: %s", resp.Status)

	// If the response body has {"errors":["reason1","reason2"]}
	// then return those reasons.
	var errInfo = Dict{}
	if err := json.NewDecoder(resp.Body).Decode(&errInfo); err == nil {
		if errorList, ok := errInfo["errors"]; ok {
			var errorStrings []string
			if errArray, ok := errorList.([]interface{}); ok {
				for _, errItem := range errArray {
					// We expect an array of strings here.
					// Non-strings will be passed along
					// JSON-encoded.
					if s, ok := errItem.(string); ok {
						errorStrings = append(errorStrings, s)
					} else if j, err := json.Marshal(errItem); err == nil {
						errorStrings = append(errorStrings, string(j))
					}
				}
				errorText = strings.Join(errorStrings, "; ")
			}
		}
	}
	return nil, ArvadosApiError{errors.New(errorText), resp.StatusCode, resp.Status}
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
