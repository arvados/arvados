// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package localdb

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/lib/webdavfs"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/hashicorp/yamux"
	"golang.org/x/net/webdav"
)

var (
	forceProxyForTest       = false
	forceInternalURLForTest *arvados.URL
)

// ContainerLog returns a WebDAV handler that reads logs from the
// indicated container. It works by proxying the request to
//
//   - the container gateway, if the container is running
//
//   - a different controller process, if the container is running and
//     the gateway is accessible through a tunnel to a different
//     controller process
//
//   - keep-web, if saved logs exist and there is no gateway (or the
//     container is finished)
//
//   - an empty-collection stub, if there is no gateway and no saved
//     log
func (conn *Conn) ContainerLog(ctx context.Context, opts arvados.ContainerLogOptions) (http.Handler, error) {
	ctr, err := conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{UUID: opts.UUID, Select: []string{"uuid", "state", "gateway_address", "log"}})
	if err != nil {
		return nil, err
	}
	if ctr.GatewayAddress == "" ||
		(ctr.State != arvados.ContainerStateLocked && ctr.State != arvados.ContainerStateRunning) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn.serveContainerLogViaKeepWeb(opts, ctr, w, r)
		}), nil
	}
	dial, arpc, err := conn.findGateway(ctx, ctr, opts.NoForward)
	if err != nil {
		return nil, err
	}
	if arpc != nil {
		opts.NoForward = true
		return arpc.ContainerLog(ctx, opts)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctx)
		var proxyReq *http.Request
		var proxyErr error
		var expectRespondAuth string
		proxy := &httputil.ReverseProxy{
			// Our custom Transport:
			//
			// - Uses a custom dialer to connect to the
			// gateway (either directly or through a
			// tunnel set up though ContainerTunnel)
			//
			// - Verifies the gateway's TLS certificate
			// using X-Arvados-Authorization headers.
			//
			// This involves modifying the outgoing
			// request header in DialTLSContext.
			// (ReverseProxy certainly doesn't expect us
			// to do this, but it works.)
			Transport: &http.Transport{
				DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					tlsconn, requestAuth, respondAuth, err := dial()
					if err != nil {
						return nil, err
					}
					proxyReq.Header.Set("X-Arvados-Authorization", requestAuth)
					expectRespondAuth = respondAuth
					return tlsconn, nil
				},
			},
			Director: func(r *http.Request) {
				// Scheme/host of incoming r.URL are
				// irrelevant now, and may even be
				// missing. Host is ignored by our
				// DialTLSContext, but we need a
				// generic syntactically correct URL
				// for net/http to work with.
				r.URL.Scheme = "https"
				r.URL.Host = "0.0.0.0:0"
				r.Header.Set("X-Arvados-Container-Gateway-Uuid", opts.UUID)
				proxyReq = r
			},
			ModifyResponse: func(resp *http.Response) error {
				if resp.Header.Get("X-Arvados-Authorization-Response") != expectRespondAuth {
					// Note this is how we detect
					// an attacker-in-the-middle.
					return httpserver.ErrorWithStatus(errors.New("bad X-Arvados-Authorization-Response header"), http.StatusBadGateway)
				}
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				proxyErr = err
			},
		}
		proxy.ServeHTTP(w, r)
		if proxyErr == nil {
			// proxy succeeded
			return
		}
		// If proxying to the container gateway fails, it
		// might be caused by a race where crunch-run exited
		// after we decided (above) the log was not final.
		// In that case we should proxy to keep-web.
		ctr, err := conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{
			UUID:   opts.UUID,
			Select: []string{"uuid", "state", "gateway_address", "log"},
		})
		if err != nil {
			// Lost access to the container record?
			httpserver.Error(w, "error re-fetching container record: "+err.Error(), http.StatusServiceUnavailable)
		} else if ctr.State == arvados.ContainerStateLocked || ctr.State == arvados.ContainerStateRunning {
			// No race, proxyErr was the best we can do
			httpserver.Error(w, "proxy error: "+proxyErr.Error(), http.StatusServiceUnavailable)
		} else {
			conn.serveContainerLogViaKeepWeb(opts, ctr, w, r)
		}
	}), nil
}

// serveContainerLogViaKeepWeb handles a request for saved container
// log content by proxying to one of the configured keep-web servers.
//
// It tries to choose a keep-web server that is running on this host.
func (conn *Conn) serveContainerLogViaKeepWeb(opts arvados.ContainerLogOptions, ctr arvados.Container, w http.ResponseWriter, r *http.Request) {
	if ctr.Log == "" {
		// Special case: if no log data exists yet, we serve
		// an empty collection by ourselves instead of
		// proxying to keep-web.
		conn.serveEmptyDir("/arvados/v1/containers/"+ctr.UUID+"/log", w, r)
		return
	}
	myURL, _ := service.URLFromContext(r.Context())
	u := url.URL(myURL)
	myHostname := u.Hostname()
	var webdavBase arvados.URL
	var ok bool
	for webdavBase = range conn.cluster.Services.WebDAVDownload.InternalURLs {
		ok = true
		u := url.URL(webdavBase)
		if h := u.Hostname(); h == "127.0.0.1" || h == "0.0.0.0" || h == "::1" || h == myHostname {
			// Prefer a keep-web service running on the
			// same host as us. (If we don't find one, we
			// pick one arbitrarily.)
			break
		}
	}
	if !ok {
		httpserver.Error(w, "no internalURLs configured for WebDAV service", http.StatusInternalServerError)
		return
	}
	proxy := &httputil.ReverseProxy{
		Director: func(r *http.Request) {
			r.URL = &url.URL{
				Scheme: webdavBase.Scheme,
				Host:   webdavBase.Host,
				Path:   "/by_id/" + url.PathEscape(ctr.Log) + opts.Path,
			}
			// Our outgoing Host header must match
			// WebDAVDownload.ExternalURL, otherwise
			// keep-web does not accept an auth token.
			r.Host = conn.cluster.Services.WebDAVDownload.ExternalURL.Host
			// We already checked permission on the
			// container, so we can use a root token here
			// instead of counting on the "access to log
			// via container request and container"
			// permission check, which can be racy when a
			// request gets retried with a new container.
			r.Header.Set("Authorization", "Bearer "+conn.cluster.SystemRootToken)
		},
	}
	if conn.cluster.TLS.Insecure {
		proxy.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conn.cluster.TLS.Insecure,
			},
		}
	}
	proxy.ServeHTTP(w, r)
}

// serveEmptyDir handles read-only webdav requests as if there was an
// empty collection rooted at the given path. It's equivalent to
// proxying to an empty collection in keep-web, but avoids the extra
// hop.
func (conn *Conn) serveEmptyDir(path string, w http.ResponseWriter, r *http.Request) {
	wh := webdav.Handler{
		Prefix:     path,
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdavfs.NoLockSystem,
		Logger: func(r *http.Request, err error) {
			if err != nil {
				ctxlog.FromContext(r.Context()).WithError(err).Info("webdav error on empty collection fs")
			}
		},
	}
	wh.ServeHTTP(w, r)
}

// ContainerSSH returns a connection to the SSH server in the
// appropriate crunch-run process on the worker node where the
// specified container is running.
//
// If the returned error is nil, the caller is responsible for closing
// sshconn.Conn.
func (conn *Conn) ContainerSSH(ctx context.Context, opts arvados.ContainerSSHOptions) (sshconn arvados.ConnectionResponse, err error) {
	user, err := conn.railsProxy.UserGetCurrent(ctx, arvados.GetOptions{})
	if err != nil {
		return sshconn, err
	}
	ctr, err := conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{UUID: opts.UUID, Select: []string{"uuid", "state", "gateway_address", "interactive_session_started"}})
	if err != nil {
		return sshconn, err
	}
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{conn.cluster.SystemRootToken}})
	if !user.IsAdmin || !conn.cluster.Containers.ShellAccess.Admin {
		if !conn.cluster.Containers.ShellAccess.User {
			return sshconn, httpserver.ErrorWithStatus(errors.New("shell access is disabled in config"), http.StatusServiceUnavailable)
		}
		crs, err := conn.railsProxy.ContainerRequestList(ctxRoot, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"container_uuid", "=", opts.UUID}}})
		if err != nil {
			return sshconn, err
		}
		for _, cr := range crs.Items {
			if cr.ModifiedByUserUUID != user.UUID {
				return sshconn, httpserver.ErrorWithStatus(errors.New("permission denied: container is associated with requests submitted by other users"), http.StatusForbidden)
			}
		}
		if crs.ItemsAvailable != len(crs.Items) {
			return sshconn, httpserver.ErrorWithStatus(errors.New("incomplete response while checking permission"), http.StatusInternalServerError)
		}
	}

	if ctr.State == arvados.ContainerStateQueued || ctr.State == arvados.ContainerStateLocked {
		return sshconn, httpserver.ErrorWithStatus(fmt.Errorf("container is not running yet (state is %q)", ctr.State), http.StatusServiceUnavailable)
	} else if ctr.State != arvados.ContainerStateRunning {
		return sshconn, httpserver.ErrorWithStatus(fmt.Errorf("container has ended (state is %q)", ctr.State), http.StatusGone)
	}

	dial, arpc, err := conn.findGateway(ctx, ctr, opts.NoForward)
	if err != nil {
		return sshconn, err
	}
	if arpc != nil {
		opts.NoForward = true
		return arpc.ContainerSSH(ctx, opts)
	}

	tlsconn, requestAuth, respondAuth, err := dial()
	if err != nil {
		return sshconn, err
	}
	bufr := bufio.NewReader(tlsconn)
	bufw := bufio.NewWriter(tlsconn)

	u := url.URL{
		Scheme: "http",
		Host:   tlsconn.RemoteAddr().String(),
		Path:   "/ssh",
	}
	postform := url.Values{
		// uuid is only needed for older crunch-run versions
		// (current version uses X-Arvados-* header below)
		"uuid":           {opts.UUID},
		"detach_keys":    {opts.DetachKeys},
		"login_username": {opts.LoginUsername},
		"no_forward":     {fmt.Sprintf("%v", opts.NoForward)},
	}
	postdata := postform.Encode()
	bufw.WriteString("POST " + u.String() + " HTTP/1.1\r\n")
	bufw.WriteString("Host: " + u.Host + "\r\n")
	bufw.WriteString("Upgrade: ssh\r\n")
	bufw.WriteString("X-Arvados-Container-Gateway-Uuid: " + opts.UUID + "\r\n")
	bufw.WriteString("X-Arvados-Authorization: " + requestAuth + "\r\n")
	bufw.WriteString("Content-Type: application/x-www-form-urlencoded\r\n")
	fmt.Fprintf(bufw, "Content-Length: %d\r\n", len(postdata))
	bufw.WriteString("\r\n")
	bufw.WriteString(postdata)
	bufw.Flush()
	resp, err := http.ReadResponse(bufr, &http.Request{Method: "POST"})
	if err != nil {
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(fmt.Errorf("error reading http response from gateway: %w", err), http.StatusBadGateway)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := ioutil.ReadAll(io.LimitReader(resp.Body, 1000))
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(fmt.Errorf("unexpected status %s %q", resp.Status, body), http.StatusBadGateway)
	}
	if strings.ToLower(resp.Header.Get("Upgrade")) != "ssh" ||
		strings.ToLower(resp.Header.Get("Connection")) != "upgrade" {
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(errors.New("bad upgrade"), http.StatusBadGateway)
	}
	if resp.Header.Get("X-Arvados-Authorization-Response") != respondAuth {
		tlsconn.Close()
		return sshconn, httpserver.ErrorWithStatus(errors.New("bad X-Arvados-Authorization-Response header"), http.StatusBadGateway)
	}

	if !ctr.InteractiveSessionStarted {
		_, err = conn.railsProxy.ContainerUpdate(ctxRoot, arvados.UpdateOptions{
			UUID: opts.UUID,
			Attrs: map[string]interface{}{
				"interactive_session_started": true,
			},
		})
		if err != nil {
			tlsconn.Close()
			return sshconn, httpserver.ErrorWithStatus(err, http.StatusInternalServerError)
		}
	}

	sshconn.Conn = tlsconn
	sshconn.Bufrw = &bufio.ReadWriter{Reader: bufr, Writer: bufw}
	sshconn.Logger = ctxlog.FromContext(ctx)
	sshconn.Header = http.Header{"Upgrade": {"ssh"}}
	return sshconn, nil
}

// ContainerGatewayTunnel sets up a tunnel enabling us (controller) to
// connect to the caller's (crunch-run's) gateway server.
func (conn *Conn) ContainerGatewayTunnel(ctx context.Context, opts arvados.ContainerGatewayTunnelOptions) (resp arvados.ConnectionResponse, err error) {
	h := hmac.New(sha256.New, []byte(conn.cluster.SystemRootToken))
	fmt.Fprint(h, opts.UUID)
	authSecret := fmt.Sprintf("%x", h.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(authSecret), []byte(opts.AuthSecret)) != 1 {
		ctxlog.FromContext(ctx).Info("received incorrect auth_secret")
		return resp, httpserver.ErrorWithStatus(errors.New("authentication error"), http.StatusUnauthorized)
	}

	muxconn, clientconn := net.Pipe()
	tunnel, err := yamux.Server(muxconn, nil)
	if err != nil {
		clientconn.Close()
		return resp, httpserver.ErrorWithStatus(err, http.StatusInternalServerError)
	}

	conn.gwTunnelsLock.Lock()
	if conn.gwTunnels == nil {
		conn.gwTunnels = map[string]*yamux.Session{opts.UUID: tunnel}
	} else {
		conn.gwTunnels[opts.UUID] = tunnel
	}
	conn.gwTunnelsLock.Unlock()

	go func() {
		<-tunnel.CloseChan()
		conn.gwTunnelsLock.Lock()
		if conn.gwTunnels[opts.UUID] == tunnel {
			delete(conn.gwTunnels, opts.UUID)
		}
		conn.gwTunnelsLock.Unlock()
	}()

	// Assuming we're acting as the backend of an http server,
	// lib/controller/router will call resp's ServeHTTP handler,
	// which upgrades the incoming http connection to a raw socket
	// and connects it to our yamux.Server through our net.Pipe().
	resp.Conn = clientconn
	resp.Bufrw = &bufio.ReadWriter{Reader: bufio.NewReader(&bytes.Buffer{}), Writer: bufio.NewWriter(&bytes.Buffer{})}
	resp.Logger = ctxlog.FromContext(ctx)
	resp.Header = http.Header{"Upgrade": {"tunnel"}}
	if u, ok := service.URLFromContext(ctx); ok {
		resp.Header.Set("X-Arvados-Internal-Url", u.String())
	} else if forceInternalURLForTest != nil {
		resp.Header.Set("X-Arvados-Internal-Url", forceInternalURLForTest.String())
	}
	return
}

type gatewayDialer func() (conn net.Conn, requestAuth, respondAuth string, err error)

// findGateway figures out how to connect to ctr's gateway.
//
// If the gateway can be contacted directly or through a tunnel on
// this instance, the first return value is a non-nil dialer.
//
// If the gateway is only accessible through a tunnel through a
// different controller process, the second return value is a non-nil
// *rpc.Conn for that controller.
func (conn *Conn) findGateway(ctx context.Context, ctr arvados.Container, noForward bool) (gatewayDialer, *rpc.Conn, error) {
	conn.gwTunnelsLock.Lock()
	tunnel := conn.gwTunnels[ctr.UUID]
	conn.gwTunnelsLock.Unlock()

	myURL, _ := service.URLFromContext(ctx)

	if host, _, splitErr := net.SplitHostPort(ctr.GatewayAddress); splitErr == nil && host != "" && host != "127.0.0.1" {
		// If crunch-run provided a GatewayAddress like
		// "ipaddr:port", that means "ipaddr" is one of the
		// external interfaces where the gateway is
		// listening. In that case, it's the most
		// reliable/direct option, so we use it even if a
		// tunnel might also be available.
		return func() (net.Conn, string, string, error) {
			rawconn, err := (&net.Dialer{}).DialContext(ctx, "tcp", ctr.GatewayAddress)
			if err != nil {
				err = httpserver.ErrorWithStatus(err, http.StatusServiceUnavailable)
			}
			return conn.dialGatewayTLS(ctx, ctr, rawconn)
		}, nil, nil
	}
	if tunnel != nil && !(forceProxyForTest && !noForward) {
		// If we can't connect directly, and the gateway has
		// established a yamux tunnel with us, connect through
		// the tunnel.
		//
		// ...except: forceProxyForTest means we are emulating
		// a situation where the gateway has established a
		// yamux tunnel with controller B, and the
		// ContainerSSH request arrives at controller A. If
		// noForward==false then we are acting as A, so
		// we pretend not to have a tunnel, and fall through
		// to the "tunurl" case below. If noForward==true
		// then the client is A and we are acting as B, so we
		// connect to our tunnel.
		return func() (net.Conn, string, string, error) {
			rawconn, err := tunnel.Open()
			if err != nil {
				err = httpserver.ErrorWithStatus(err, http.StatusServiceUnavailable)
			}
			return conn.dialGatewayTLS(ctx, ctr, rawconn)
		}, nil, nil
	}
	if tunurl := strings.TrimPrefix(ctr.GatewayAddress, "tunnel "); tunurl != ctr.GatewayAddress &&
		tunurl != "" &&
		tunurl != myURL.String() &&
		!noForward {
		// If crunch-run provided a GatewayAddress like
		// "tunnel https://10.0.0.10:1010/", that means the
		// gateway has established a yamux tunnel with the
		// controller process at the indicated InternalURL
		// (which isn't us, otherwise we would have had
		// "tunnel != nil" above). We need to proxy through to
		// the other controller process in order to use the
		// tunnel.
		for u := range conn.cluster.Services.Controller.InternalURLs {
			if u.String() == tunurl {
				ctxlog.FromContext(ctx).Debugf("connecting to container gateway through other controller at %s", u)
				u := url.URL(u)
				return nil, rpc.NewConn(conn.cluster.ClusterID, &u, conn.cluster.TLS.Insecure, rpc.PassthroughTokenProvider), nil
			}
		}
		ctxlog.FromContext(ctx).Warnf("container gateway provided a tunnel endpoint %s that is not one of Services.Controller.InternalURLs", tunurl)
		return nil, nil, httpserver.ErrorWithStatus(errors.New("container gateway is running but tunnel endpoint is invalid"), http.StatusServiceUnavailable)
	}
	if ctr.GatewayAddress == "" {
		return nil, nil, httpserver.ErrorWithStatus(errors.New("container is running but gateway is not available"), http.StatusServiceUnavailable)
	} else {
		return nil, nil, httpserver.ErrorWithStatus(errors.New("container is running but tunnel is down"), http.StatusServiceUnavailable)
	}
}

// dialGatewayTLS negotiates a TLS connection to a container gateway
// over the given raw connection.
func (conn *Conn) dialGatewayTLS(ctx context.Context, ctr arvados.Container, rawconn net.Conn) (*tls.Conn, string, string, error) {
	// crunch-run uses a self-signed / unverifiable TLS
	// certificate, so we use the following scheme to ensure we're
	// not talking to an attacker-in-the-middle.
	//
	// 1. Compute ctrKey = HMAC-SHA256(sysRootToken,ctrUUID) --
	// this will be the same ctrKey that a-d-c supplied to
	// crunch-run in the GatewayAuthSecret env var.
	//
	// 2. Compute requestAuth = HMAC-SHA256(ctrKey,serverCert) and
	// send it to crunch-run as the X-Arvados-Authorization
	// header, proving that we know ctrKey. (Note a MITM cannot
	// replay the proof to a real crunch-run server, because the
	// real crunch-run server would have a different cert.)
	//
	// 3. Compute respondAuth = HMAC-SHA256(ctrKey,requestAuth)
	// and ensure the server returns it in the
	// X-Arvados-Authorization-Response header, proving that the
	// server knows ctrKey.
	var requestAuth, respondAuth string
	tlsconn := tls.Client(rawconn, &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(rawCerts) == 0 {
				return errors.New("no certificate received, cannot compute authorization header")
			}
			h := hmac.New(sha256.New, []byte(conn.cluster.SystemRootToken))
			fmt.Fprint(h, ctr.UUID)
			authKey := fmt.Sprintf("%x", h.Sum(nil))
			h = hmac.New(sha256.New, []byte(authKey))
			h.Write(rawCerts[0])
			requestAuth = fmt.Sprintf("%x", h.Sum(nil))
			h.Reset()
			h.Write([]byte(requestAuth))
			respondAuth = fmt.Sprintf("%x", h.Sum(nil))
			return nil
		},
	})
	err := tlsconn.HandshakeContext(ctx)
	if err != nil {
		return nil, "", "", httpserver.ErrorWithStatus(fmt.Errorf("TLS handshake failed: %w", err), http.StatusBadGateway)
	}
	if respondAuth == "" {
		tlsconn.Close()
		return nil, "", "", httpserver.ErrorWithStatus(errors.New("BUG: no respondAuth"), http.StatusInternalServerError)
	}
	return tlsconn, requestAuth, respondAuth, nil
}
