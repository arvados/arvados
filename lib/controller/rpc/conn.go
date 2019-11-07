// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package rpc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"git.curoverse.com/arvados.git/sdk/go/auth"
)

type TokenProvider func(context.Context) ([]string, error)

func PassthroughTokenProvider(ctx context.Context) ([]string, error) {
	if incoming, ok := auth.FromContext(ctx); !ok {
		return nil, errors.New("no token provided")
	} else {
		return incoming.Tokens, nil
	}
}

type Conn struct {
	SendHeader    http.Header
	clusterID     string
	httpClient    http.Client
	baseURL       url.URL
	tokenProvider TokenProvider
}

func NewConn(clusterID string, url *url.URL, insecure bool, tp TokenProvider) *Conn {
	transport := http.DefaultTransport
	if insecure {
		// It's not safe to copy *http.DefaultTransport
		// because it has a mutex (which might be locked)
		// protecting a private map (which might not be nil).
		// So we build our own, using the Go 1.12 default
		// values, ignoring any changes the application has
		// made to http.DefaultTransport.
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}
	}
	return &Conn{
		clusterID: clusterID,
		httpClient: http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse },
			Transport:     transport,
		},
		baseURL:       *url,
		tokenProvider: tp,
	}
}

func (conn *Conn) requestAndDecode(ctx context.Context, dst interface{}, ep arvados.APIEndpoint, body io.Reader, opts interface{}) error {
	aClient := arvados.Client{
		Client:     &conn.httpClient,
		Scheme:     conn.baseURL.Scheme,
		APIHost:    conn.baseURL.Host,
		SendHeader: conn.SendHeader,
	}
	tokens, err := conn.tokenProvider(ctx)
	if err != nil {
		return err
	} else if len(tokens) > 0 {
		ctx = arvados.ContextWithAuthorization(ctx, "Bearer "+tokens[0])
	} else {
		// Use a non-empty auth string to ensure we override
		// any default token set on aClient -- and to avoid
		// having the remote prompt us to send a token by
		// responding 401.
		ctx = arvados.ContextWithAuthorization(ctx, "Bearer -")
	}

	// Encode opts to JSON and decode from there to a
	// map[string]interface{}, so we can munge the query params
	// using the JSON key names specified by opts' struct tags.
	j, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("%T: requestAndDecode: Marshal opts: %s", conn, err)
	}
	var params map[string]interface{}
	err = json.Unmarshal(j, &params)
	if err != nil {
		return fmt.Errorf("%T: requestAndDecode: Unmarshal opts: %s", conn, err)
	}
	if attrs, ok := params["attrs"]; ok && ep.AttrsKey != "" {
		params[ep.AttrsKey] = attrs
		delete(params, "attrs")
	}
	if limit, ok := params["limit"].(float64); ok && limit < 0 {
		// Negative limit means "not specified" here, but some
		// servers/versions do not accept that, so we need to
		// remove it entirely.
		delete(params, "limit")
	}
	if len(tokens) > 1 {
		params["reader_tokens"] = tokens[1:]
	}
	path := ep.Path
	if strings.Contains(ep.Path, "/:uuid") {
		uuid, _ := params["uuid"].(string)
		path = strings.Replace(path, "/:uuid", "/"+uuid, 1)
		delete(params, "uuid")
	}
	return aClient.RequestAndDecodeContext(ctx, dst, ep.Method, path, body, params)
}

func (conn *Conn) BaseURL() url.URL {
	return conn.baseURL
}

func (conn *Conn) ConfigGet(ctx context.Context) (json.RawMessage, error) {
	ep := arvados.EndpointConfigGet
	var resp json.RawMessage
	err := conn.requestAndDecode(ctx, &resp, ep, nil, nil)
	return resp, err
}

func (conn *Conn) Login(ctx context.Context, options arvados.LoginOptions) (arvados.LoginResponse, error) {
	ep := arvados.EndpointLogin
	var resp arvados.LoginResponse
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	resp.RedirectLocation = conn.relativeToBaseURL(resp.RedirectLocation)
	return resp, err
}

// If the given location is a valid URL and its origin is the same as
// conn.baseURL, return it as a relative URL. Otherwise, return it
// unmodified.
func (conn *Conn) relativeToBaseURL(location string) string {
	u, err := url.Parse(location)
	if err == nil && u.Scheme == conn.baseURL.Scheme && strings.ToLower(u.Host) == strings.ToLower(conn.baseURL.Host) {
		u.Opaque = ""
		u.Scheme = ""
		u.User = nil
		u.Host = ""
		return u.String()
	} else {
		return location
	}
}

func (conn *Conn) CollectionCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Collection, error) {
	ep := arvados.EndpointCollectionCreate
	var resp arvados.Collection
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) CollectionUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Collection, error) {
	ep := arvados.EndpointCollectionUpdate
	var resp arvados.Collection
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) CollectionGet(ctx context.Context, options arvados.GetOptions) (arvados.Collection, error) {
	ep := arvados.EndpointCollectionGet
	var resp arvados.Collection
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) CollectionList(ctx context.Context, options arvados.ListOptions) (arvados.CollectionList, error) {
	ep := arvados.EndpointCollectionList
	var resp arvados.CollectionList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) CollectionProvenance(ctx context.Context, options arvados.GetOptions) (map[string]interface{}, error) {
	ep := arvados.EndpointCollectionProvenance
	var resp map[string]interface{}
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) CollectionUsedBy(ctx context.Context, options arvados.GetOptions) (map[string]interface{}, error) {
	ep := arvados.EndpointCollectionUsedBy
	var resp map[string]interface{}
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) CollectionDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Collection, error) {
	ep := arvados.EndpointCollectionDelete
	var resp arvados.Collection
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) CollectionTrash(ctx context.Context, options arvados.DeleteOptions) (arvados.Collection, error) {
	ep := arvados.EndpointCollectionTrash
	var resp arvados.Collection
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) CollectionUntrash(ctx context.Context, options arvados.UntrashOptions) (arvados.Collection, error) {
	ep := arvados.EndpointCollectionUntrash
	var resp arvados.Collection
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Container, error) {
	ep := arvados.EndpointContainerCreate
	var resp arvados.Container
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Container, error) {
	ep := arvados.EndpointContainerUpdate
	var resp arvados.Container
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerGet(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	ep := arvados.EndpointContainerGet
	var resp arvados.Container
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerList(ctx context.Context, options arvados.ListOptions) (arvados.ContainerList, error) {
	ep := arvados.EndpointContainerList
	var resp arvados.ContainerList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Container, error) {
	ep := arvados.EndpointContainerDelete
	var resp arvados.Container
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerLock(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	ep := arvados.EndpointContainerLock
	var resp arvados.Container
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerUnlock(ctx context.Context, options arvados.GetOptions) (arvados.Container, error) {
	ep := arvados.EndpointContainerUnlock
	var resp arvados.Container
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) SpecimenCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Specimen, error) {
	ep := arvados.EndpointSpecimenCreate
	var resp arvados.Specimen
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) SpecimenUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Specimen, error) {
	ep := arvados.EndpointSpecimenUpdate
	var resp arvados.Specimen
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) SpecimenGet(ctx context.Context, options arvados.GetOptions) (arvados.Specimen, error) {
	ep := arvados.EndpointSpecimenGet
	var resp arvados.Specimen
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) SpecimenList(ctx context.Context, options arvados.ListOptions) (arvados.SpecimenList, error) {
	ep := arvados.EndpointSpecimenList
	var resp arvados.SpecimenList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) SpecimenDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Specimen, error) {
	ep := arvados.EndpointSpecimenDelete
	var resp arvados.Specimen
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) APIClientAuthorizationCurrent(ctx context.Context, options arvados.GetOptions) (arvados.APIClientAuthorization, error) {
	ep := arvados.EndpointAPIClientAuthorizationCurrent
	var resp arvados.APIClientAuthorization
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

type UserSessionCreateOptions struct {
	AuthInfo map[string]interface{} `json:"auth_info"`
	ReturnTo string                 `json:"return_to"`
}

func (conn *Conn) UserSessionCreate(ctx context.Context, options UserSessionCreateOptions) (arvados.LoginResponse, error) {
	ep := arvados.APIEndpoint{Method: "POST", Path: "auth/controller/callback"}
	var resp arvados.LoginResponse
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
