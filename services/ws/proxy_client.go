package main

import (
	"net/http"
	"net/url"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type proxyClient struct {
	*arvados.Client
}

func NewProxyClient(ac arvados.Client) *proxyClient {
	ac.AuthToken = ""
	return &proxyClient{
		Client: &ac,
	}
}

func (pc *proxyClient) SetToken(token string) {
	pc.Client.AuthToken = token
}

func (pc *proxyClient) CheckReadPermission(uuid string) (bool, error) {
	var buf map[string]interface{}
	path, err := pc.PathForUUID("get", uuid)
	if err != nil {
		return false, err
	}
	err = pc.RequestAndDecode(&buf, "GET", path, nil, url.Values{
		"select": {`["uuid"]`},
	})
	if err, ok := err.(arvados.TransactionError); ok && err.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
