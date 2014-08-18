package main

import (
	"arvados.org/sdk"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"
)

// tokenInfo records information we get from an API token
type tokenInfo struct {
	GeneratedAt            time.Time
	User                   sdk.Dict
	ApiClientAuthorization sdk.Dict
}

var tokenCache map[string]*tokenInfo

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
	if tok := lookupToken(api_token); tok == nil {
		return false
	} else {
		return tok.User["is_admin"].(bool)
	}
}

// HasUnlimitedScope returns true if the scopes attached to this
// token's ApiClientAuthorization record include "all".
func HasUnlimitedScope(api_token string) bool {
	if tok := lookupToken(api_token); tok == nil {
		return false
	} else {
		if tok.ApiClientAuthorization["scopes"] == nil {
			return false
		}
		var scopes []interface{} = tok.ApiClientAuthorization["scopes"].([]interface{})
		for _, s := range scopes {
			if s.(string) == "all" {
				return true
			}
		}
	}
	return false
}

// lookupToken fetches information about an API token (the User and
// ApiClientAuthorization methods belonging to it). It will use a
// cached value, refreshing the cache if the token is not found or has
// expired.
//
func lookupToken(api_token string) *tokenInfo {
	var refresh bool = true
	var ti *tokenInfo
	var ok bool

	if tokenCache == nil {
		tokenCache = make(map[string]*tokenInfo)
	}
	if ti, ok = tokenCache[api_token]; ok {
		token_age := time.Now().Sub(ti.GeneratedAt)
		if token_age < time.Duration(token_cache_ttl)*time.Second {
			refresh = false
		}
	}
	if refresh {
		ti = fetchTokenInfo(api_token)
		tokenCache[api_token] = ti
	}
	return ti
}

// fetchTokenInfo issues API calls for the User and
// ApiClientAuthorization record associated with a token. It returns a
// tokenInfo with the fetched data.  If any data could
func fetchTokenInfo(api_token string) *tokenInfo {
	var ti tokenInfo
	var api_client = MakeApiClient(api_token)
	if err := api_client.List("users/current", nil, &ti.User); err != nil {
		log.Printf("fetchTokenInfo: %s\n", err)
		return nil
	}
	req := "api_client_authorizations/" + api_token
	if err := api_client.List(req, nil, &ti.ApiClientAuthorization); err != nil {
		log.Printf("fetchTokenInfo: %s\n", err)
		return nil
	}
	ti.GeneratedAt = time.Now()
	return &ti
}
