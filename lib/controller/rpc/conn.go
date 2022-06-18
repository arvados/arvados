// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package rpc

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
)

const rfc3339NanoFixed = "2006-01-02T15:04:05.000000000Z07:00"

type TokenProvider func(context.Context) ([]string, error)

func PassthroughTokenProvider(ctx context.Context) ([]string, error) {
	incoming, ok := auth.FromContext(ctx)
	if !ok {
		return nil, errors.New("no token provided")
	}
	return incoming.Tokens, nil
}

type Conn struct {
	SendHeader         http.Header
	RedactHostInErrors bool

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
	dec := json.NewDecoder(bytes.NewBuffer(j))
	dec.UseNumber()
	err = dec.Decode(&params)
	if err != nil {
		return fmt.Errorf("%T: requestAndDecode: Decode opts: %s", conn, err)
	}
	if attrs, ok := params["attrs"]; ok && ep.AttrsKey != "" {
		params[ep.AttrsKey] = attrs
		delete(params, "attrs")
	}
	if limitStr, ok := params["limit"]; ok {
		if limit, err := strconv.ParseInt(string(limitStr.(json.Number)), 10, 64); err == nil && limit < 0 {
			// Negative limit means "not specified" here, but some
			// servers/versions do not accept that, so we need to
			// remove it entirely.
			delete(params, "limit")
		}
	}

	if authinfo, ok := params["auth_info"]; ok {
		if tmp, ok2 := authinfo.(map[string]interface{}); ok2 {
			for k, v := range tmp {
				if strings.HasSuffix(k, "_at") {
					// Change zero times values to nil
					if v, ok3 := v.(string); ok3 && (strings.HasPrefix(v, "0001-01-01T00:00:00") || v == "") {
						tmp[k] = nil
					}
				}
			}
		}
	}

	if len(tokens) > 1 {
		params["reader_tokens"] = tokens[1:]
	}
	path := ep.Path
	if strings.Contains(ep.Path, "/{uuid}") {
		uuid, _ := params["uuid"].(string)
		path = strings.Replace(path, "/{uuid}", "/"+uuid, 1)
		delete(params, "uuid")
	}
	err = aClient.RequestAndDecodeContext(ctx, dst, ep.Method, path, body, params)
	if err != nil && conn.RedactHostInErrors {
		redacted := strings.Replace(err.Error(), strings.TrimSuffix(conn.baseURL.String(), "/"), "//railsapi.internal", -1)
		if strings.HasPrefix(redacted, "request failed: ") {
			redacted = strings.Replace(redacted, "request failed: ", "", -1)
		}
		if redacted != err.Error() {
			if err, ok := err.(httpStatusError); ok {
				return wrapHTTPStatusError(err, redacted)
			} else {
				return errors.New(redacted)
			}
		}
	}
	return err
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

func (conn *Conn) VocabularyGet(ctx context.Context) (arvados.Vocabulary, error) {
	ep := arvados.EndpointVocabularyGet
	var resp arvados.Vocabulary
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

func (conn *Conn) Logout(ctx context.Context, options arvados.LogoutOptions) (arvados.LogoutResponse, error) {
	ep := arvados.EndpointLogout
	var resp arvados.LogoutResponse
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
	}
	return location
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

// ContainerSSH returns a connection to the out-of-band SSH server for
// a running container. If the returned error is nil, the caller is
// responsible for closing sshconn.Conn.
func (conn *Conn) ContainerSSH(ctx context.Context, options arvados.ContainerSSHOptions) (sshconn arvados.ContainerSSHConnection, err error) {
	u, err := conn.baseURL.Parse("/" + strings.Replace(arvados.EndpointContainerSSH.Path, "{uuid}", options.UUID, -1))
	if err != nil {
		err = fmt.Errorf("url.Parse: %w", err)
		return
	}
	u.RawQuery = url.Values{
		"detach_keys":    {options.DetachKeys},
		"login_username": {options.LoginUsername},
	}.Encode()
	resp, err := conn.socket(ctx, u, "ssh", nil)
	if err != nil {
		return
	}
	return arvados.ContainerSSHConnection(resp), nil
}

// ContainerGatewayTunnel returns a connection to a yamux session on
// the controller. The caller should connect the returned resp.Conn to
// a client-side yamux session.
func (conn *Conn) ContainerGatewayTunnel(ctx context.Context, options arvados.ContainerGatewayTunnelOptions) (tunnelconn arvados.ConnectionResponse, err error) {
	u, err := conn.baseURL.Parse("/" + strings.Replace(arvados.EndpointContainerGatewayTunnel.Path, "{uuid}", options.UUID, -1))
	if err != nil {
		err = fmt.Errorf("url.Parse: %w", err)
		return
	}
	return conn.socket(ctx, u, "tunnel", url.Values{
		"auth_secret": {options.AuthSecret},
	})
}

// socket sets up a socket using the specified API endpoint and
// upgrade header.
func (conn *Conn) socket(ctx context.Context, u *url.URL, upgradeHeader string, postform url.Values) (connresp arvados.ConnectionResponse, err error) {
	addr := conn.baseURL.Host
	if strings.Index(addr, ":") < 1 || (strings.Contains(addr, "::") && addr[0] != '[') {
		// hostname or ::1 or 1::1
		addr = net.JoinHostPort(addr, "https")
	}
	insecure := false
	if tlsconf := conn.httpClient.Transport.(*http.Transport).TLSClientConfig; tlsconf != nil && tlsconf.InsecureSkipVerify {
		insecure = true
	}
	netconn, err := tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: insecure})
	if err != nil {
		err = fmt.Errorf("tls.Dial: %w", err)
		return
	}
	defer func() {
		if err != nil {
			netconn.Close()
		}
	}()
	bufr := bufio.NewReader(netconn)
	bufw := bufio.NewWriter(netconn)

	tokens, err := conn.tokenProvider(ctx)
	if err != nil {
		return
	} else if len(tokens) < 1 {
		err = httpserver.ErrorWithStatus(errors.New("unauthorized"), http.StatusUnauthorized)
		return
	}
	postdata := postform.Encode()
	bufw.WriteString("POST " + u.String() + " HTTP/1.1\r\n")
	bufw.WriteString("Authorization: Bearer " + tokens[0] + "\r\n")
	bufw.WriteString("Host: " + u.Host + "\r\n")
	bufw.WriteString("Upgrade: " + upgradeHeader + "\r\n")
	bufw.WriteString("Content-Type: application/x-www-form-urlencoded\r\n")
	fmt.Fprintf(bufw, "Content-Length: %d\r\n", len(postdata))
	bufw.WriteString("\r\n")
	if len(postdata) > 0 {
		bufw.WriteString(postdata)
	}
	bufw.Flush()
	resp, err := http.ReadResponse(bufr, &http.Request{Method: "GET"})
	if err != nil {
		err = fmt.Errorf("http.ReadResponse: %w", err)
		return
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		var message string
		var errDoc httpserver.ErrorResponse
		if err := json.Unmarshal(body, &errDoc); err == nil {
			message = strings.Join(errDoc.Errors, "; ")
		} else {
			message = fmt.Sprintf("%q", body)
		}
		err = fmt.Errorf("server did not provide a tunnel: %s (HTTP %d)", message, resp.StatusCode)
		return
	}
	if strings.ToLower(resp.Header.Get("Upgrade")) != upgradeHeader ||
		strings.ToLower(resp.Header.Get("Connection")) != "upgrade" {
		err = fmt.Errorf("bad response from server: Upgrade %q Connection %q", resp.Header.Get("Upgrade"), resp.Header.Get("Connection"))
		return
	}
	connresp.Conn = netconn
	connresp.Bufrw = &bufio.ReadWriter{Reader: bufr, Writer: bufw}
	return
}

func (conn *Conn) ContainerRequestCreate(ctx context.Context, options arvados.CreateOptions) (arvados.ContainerRequest, error) {
	ep := arvados.EndpointContainerRequestCreate
	var resp arvados.ContainerRequest
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerRequestUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.ContainerRequest, error) {
	ep := arvados.EndpointContainerRequestUpdate
	var resp arvados.ContainerRequest
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerRequestGet(ctx context.Context, options arvados.GetOptions) (arvados.ContainerRequest, error) {
	ep := arvados.EndpointContainerRequestGet
	var resp arvados.ContainerRequest
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerRequestList(ctx context.Context, options arvados.ListOptions) (arvados.ContainerRequestList, error) {
	ep := arvados.EndpointContainerRequestList
	var resp arvados.ContainerRequestList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) ContainerRequestDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.ContainerRequest, error) {
	ep := arvados.EndpointContainerRequestDelete
	var resp arvados.ContainerRequest
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Group, error) {
	ep := arvados.EndpointGroupCreate
	var resp arvados.Group
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Group, error) {
	ep := arvados.EndpointGroupUpdate
	var resp arvados.Group
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupGet(ctx context.Context, options arvados.GetOptions) (arvados.Group, error) {
	ep := arvados.EndpointGroupGet
	var resp arvados.Group
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupList(ctx context.Context, options arvados.ListOptions) (arvados.GroupList, error) {
	ep := arvados.EndpointGroupList
	var resp arvados.GroupList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupContents(ctx context.Context, options arvados.GroupContentsOptions) (arvados.ObjectList, error) {
	ep := arvados.EndpointGroupContents
	var resp arvados.ObjectList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupShared(ctx context.Context, options arvados.ListOptions) (arvados.GroupList, error) {
	ep := arvados.EndpointGroupShared
	var resp arvados.GroupList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Group, error) {
	ep := arvados.EndpointGroupDelete
	var resp arvados.Group
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupTrash(ctx context.Context, options arvados.DeleteOptions) (arvados.Group, error) {
	ep := arvados.EndpointGroupTrash
	var resp arvados.Group
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) GroupUntrash(ctx context.Context, options arvados.UntrashOptions) (arvados.Group, error) {
	ep := arvados.EndpointGroupUntrash
	var resp arvados.Group
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) LinkCreate(ctx context.Context, options arvados.CreateOptions) (arvados.Link, error) {
	ep := arvados.EndpointLinkCreate
	var resp arvados.Link
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) LinkUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.Link, error) {
	ep := arvados.EndpointLinkUpdate
	var resp arvados.Link
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) LinkGet(ctx context.Context, options arvados.GetOptions) (arvados.Link, error) {
	ep := arvados.EndpointLinkGet
	var resp arvados.Link
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) LinkList(ctx context.Context, options arvados.ListOptions) (arvados.LinkList, error) {
	ep := arvados.EndpointLinkList
	var resp arvados.LinkList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) LinkDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.Link, error) {
	ep := arvados.EndpointLinkDelete
	var resp arvados.Link
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

func (conn *Conn) SysTrashSweep(ctx context.Context, options struct{}) (struct{}, error) {
	ep := arvados.EndpointSysTrashSweep
	var resp struct{}
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) UserCreate(ctx context.Context, options arvados.CreateOptions) (arvados.User, error) {
	ep := arvados.EndpointUserCreate
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.User, error) {
	ep := arvados.EndpointUserUpdate
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserMerge(ctx context.Context, options arvados.UserMergeOptions) (arvados.User, error) {
	ep := arvados.EndpointUserMerge
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserActivate(ctx context.Context, options arvados.UserActivateOptions) (arvados.User, error) {
	ep := arvados.EndpointUserActivate
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserSetup(ctx context.Context, options arvados.UserSetupOptions) (map[string]interface{}, error) {
	ep := arvados.EndpointUserSetup
	var resp map[string]interface{}
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserUnsetup(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	ep := arvados.EndpointUserUnsetup
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserGet(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	ep := arvados.EndpointUserGet
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserGetCurrent(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	ep := arvados.EndpointUserGetCurrent
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserGetSystem(ctx context.Context, options arvados.GetOptions) (arvados.User, error) {
	ep := arvados.EndpointUserGetSystem
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserList(ctx context.Context, options arvados.ListOptions) (arvados.UserList, error) {
	ep := arvados.EndpointUserList
	var resp arvados.UserList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) UserDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.User, error) {
	ep := arvados.EndpointUserDelete
	var resp arvados.User
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) APIClientAuthorizationCurrent(ctx context.Context, options arvados.GetOptions) (arvados.APIClientAuthorization, error) {
	ep := arvados.EndpointAPIClientAuthorizationCurrent
	var resp arvados.APIClientAuthorization
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) APIClientAuthorizationCreate(ctx context.Context, options arvados.CreateOptions) (arvados.APIClientAuthorization, error) {
	ep := arvados.EndpointAPIClientAuthorizationCreate
	var resp arvados.APIClientAuthorization
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) APIClientAuthorizationUpdate(ctx context.Context, options arvados.UpdateOptions) (arvados.APIClientAuthorization, error) {
	ep := arvados.EndpointAPIClientAuthorizationUpdate
	var resp arvados.APIClientAuthorization
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) APIClientAuthorizationDelete(ctx context.Context, options arvados.DeleteOptions) (arvados.APIClientAuthorization, error) {
	ep := arvados.EndpointAPIClientAuthorizationDelete
	var resp arvados.APIClientAuthorization
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) APIClientAuthorizationList(ctx context.Context, options arvados.ListOptions) (arvados.APIClientAuthorizationList, error) {
	ep := arvados.EndpointAPIClientAuthorizationList
	var resp arvados.APIClientAuthorizationList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}
func (conn *Conn) APIClientAuthorizationGet(ctx context.Context, options arvados.GetOptions) (arvados.APIClientAuthorization, error) {
	ep := arvados.EndpointAPIClientAuthorizationGet
	var resp arvados.APIClientAuthorization
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

type UserSessionAuthInfo struct {
	UserUUID        string    `json:"user_uuid"`
	Email           string    `json:"email"`
	AlternateEmails []string  `json:"alternate_emails"`
	FirstName       string    `json:"first_name"`
	LastName        string    `json:"last_name"`
	Username        string    `json:"username"`
	ExpiresAt       time.Time `json:"expires_at"`
}

type UserSessionCreateOptions struct {
	AuthInfo UserSessionAuthInfo `json:"auth_info"`
	ReturnTo string              `json:"return_to"`
}

func (conn *Conn) UserSessionCreate(ctx context.Context, options UserSessionCreateOptions) (arvados.LoginResponse, error) {
	ep := arvados.APIEndpoint{Method: "POST", Path: "auth/controller/callback"}
	var resp arvados.LoginResponse
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) UserBatchUpdate(ctx context.Context, options arvados.UserBatchUpdateOptions) (arvados.UserList, error) {
	ep := arvados.EndpointUserBatchUpdate
	var resp arvados.UserList
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

func (conn *Conn) UserAuthenticate(ctx context.Context, options arvados.UserAuthenticateOptions) (arvados.APIClientAuthorization, error) {
	ep := arvados.EndpointUserAuthenticate
	var resp arvados.APIClientAuthorization
	err := conn.requestAndDecode(ctx, &resp, ep, nil, options)
	return resp, err
}

// httpStatusError is an error with an HTTP status code that can be
// propagated by lib/controller/router, etc.
type httpStatusError interface {
	error
	HTTPStatus() int
}

// wrappedHTTPStatusError is used to augment/replace an error message
// while preserving the HTTP status code indicated by the original
// error.
type wrappedHTTPStatusError struct {
	httpStatusError
	message string
}

func wrapHTTPStatusError(err httpStatusError, message string) httpStatusError {
	return wrappedHTTPStatusError{err, message}
}

func (err wrappedHTTPStatusError) Error() string {
	return err.message
}
