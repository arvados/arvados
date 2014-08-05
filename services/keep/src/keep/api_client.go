package main

import (
	"arvados.org/sdk"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	_ "time"
)

// MakeApiClient returns a sdk.ArvadosClient suitable for connecting to the
// requested API server.
//
func MakeApiClient(api_token string) sdk.ArvadosClient {
	// Make an API client.
	// TODO(twp): use command line flags for ARVADOS_API_HOST, etc.
	var api_host string = os.Getenv("ARVADOS_API_HOST")
	var api_insecure bool = (os.Getenv("ARVADOS_API_HOST_INSECURE") == "true")
	return sdk.ArvadosClient{
		ApiServer:   api_host,
		ApiToken:    api_token,
		ApiInsecure: api_insecure,
		Client: &http.Client{Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: api_insecure}}},
		External: false}
}

// IsAdmin returns true if the user who submitted this request has an
// administrator's token (the "is_admin" field of their User record is
// "true").
//
func IsAdmin(api_token string) bool {
	var api_client = MakeApiClient(api_token)
	// Ask the API server whether this user is an admin.
	var userinfo sdk.Dict
	if err := api_client.List("users/current", nil, &userinfo); err != nil {
		log.Printf("IsAdmin: %s\n", err)
		return false
	}
	return userinfo["is_admin"].(bool)
}

// HasUnlimitedScope returns true if the scopes attached to this
// token's ApiClientAuthorization record include "all".
func HasUnlimitedScope(api_token string) bool {
	var api_client = MakeApiClient(api_token)
	var auth sdk.Dict
	req := "api_client_authorizations/" + api_token
	if err := api_client.List(req, nil, &auth); err != nil {
		log.Printf("HasUnlimitedScope: %s\n", err)
		return false
	}

	var scopes []interface{} = auth["scopes"].([]interface{})
	for _, s := range scopes {
		if s.(string) == "all" {
			return true
		}
	}
	return false
}
