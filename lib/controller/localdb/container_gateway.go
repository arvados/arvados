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
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/service"
	"git.arvados.org/arvados.git/lib/webdavfs"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	keepweb "git.arvados.org/arvados.git/services/keep-web"
	"github.com/hashicorp/yamux"
	"golang.org/x/net/webdav"
)

var (
	// forceProxyForTest enables test cases to exercise the "proxy
	// to a different controller instance" code path without
	// running a second controller instance.  If this is set, an
	// incoming request with NoForward==false is always proxied to
	// the configured controller instance that matches the
	// container gateway's tunnel endpoint, without checking
	// whether the tunnel is actually connected to the current
	// process.
	forceProxyForTest = false

	// forceInternalURLForTest is sent to the crunch-run gateway
	// when setting up a tunnel in a test suite where
	// service.URLFromContext() does not return anything.
	forceInternalURLForTest *arvados.URL
)

// ContainerRequestLog returns a WebDAV handler that reads logs from
// the indicated container request. It works by proxying the incoming
// HTTP request to
//
//   - the container gateway, if there is an associated container that
//     is running
//
//   - a different controller process, if there is a running container
//     whose gateway is accessible through a tunnel to a different
//     controller process
//
//   - keep-web, if saved logs exist and there is no gateway (or the
//     associated container is finished)
//
//   - an empty-collection stub, if there is no gateway and no saved
//     log
//
// For an incoming request
//
//	GET /arvados/v1/container_requests/{cr_uuid}/log/{c_uuid}{/c_log_path}
//
// The upstream request may be to {c_uuid}'s container gateway
//
//	GET /arvados/v1/container_requests/{cr_uuid}/log/{c_uuid}{/c_log_path}
//	X-Webdav-Prefix: /arvados/v1/container_requests/{cr_uuid}/log/{c_uuid}
//	X-Webdav-Source: /log
//
// ...or the upstream request may be to keep-web (where {cr_log_uuid}
// is the container request log collection UUID)
//
//	GET /arvados/v1/container_requests/{cr_uuid}/log/{c_uuid}{/c_log_path}
//	Host: {cr_log_uuid}.internal
//	X-Webdav-Prefix: /arvados/v1/container_requests/{cr_uuid}/log
//	X-Arvados-Container-Uuid: {c_uuid}
//
// ...or the request may be handled locally using an empty-collection
// stub.
func (conn *Conn) ContainerRequestLog(ctx context.Context, opts arvados.ContainerLogOptions) (http.Handler, error) {
	if opts.Method == "OPTIONS" && opts.Header.Get("Access-Control-Request-Method") != "" {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !keepweb.ServeCORSPreflight(w, opts.Header) {
				// Inconceivable.  We already checked
				// for the only condition where
				// ServeCORSPreflight returns false.
				httpserver.Error(w, "unhandled CORS preflight request", http.StatusInternalServerError)
			}
		}), nil
	}
	cr, err := conn.railsProxy.ContainerRequestGet(ctx, arvados.GetOptions{UUID: opts.UUID, Select: []string{"uuid", "container_uuid", "log_uuid"}})
	if err != nil {
		if se := httpserver.HTTPStatusError(nil); errors.As(err, &se) && se.HTTPStatus() == http.StatusUnauthorized {
			// Hint to WebDAV client that we accept HTTP basic auth.
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Www-Authenticate", "Basic realm=\"collections\"")
				w.WriteHeader(http.StatusUnauthorized)
			}), nil
		}
		return nil, err
	}
	ctr, err := conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{UUID: cr.ContainerUUID, Select: []string{"uuid", "state", "gateway_address"}})
	if err != nil {
		return nil, err
	}
	// .../log/{ctr.UUID} is a directory where the currently
	// assigned container's log data [will] appear (as opposed to
	// previous attempts in .../log/{previous_ctr_uuid}). Requests
	// that are outside that directory, and requests on a
	// non-running container, are proxied to keep-web instead of
	// going through the container gateway system.
	//
	// Side note: a depth>1 directory tree listing starting at
	// .../{cr_uuid}/log will only include subdirectories for
	// finished containers, i.e., will not include a subdirectory
	// with log data for a current (unfinished) container UUID.
	// In order to access live logs, a client must look up the
	// container_uuid field of the container request record, and
	// explicitly request a path under .../{cr_uuid}/log/{c_uuid}.
	if ctr.GatewayAddress == "" ||
		(ctr.State != arvados.ContainerStateLocked && ctr.State != arvados.ContainerStateRunning) ||
		!(opts.Path == "/"+ctr.UUID || strings.HasPrefix(opts.Path, "/"+ctr.UUID+"/")) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn.serveContainerRequestLogViaKeepWeb(opts, cr, w, r)
		}), nil
	}
	dial, arpc, err := conn.findGateway(ctx, ctr, opts.NoForward)
	if err != nil {
		return nil, err
	}
	if arpc != nil {
		opts.NoForward = true
		return arpc.ContainerRequestLog(ctx, opts)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(ctx)
		var proxyErr error
		gatewayProxy(dial, w, http.Header{
			"X-Arvados-Container-Gateway-Uuid": {ctr.UUID},
			"X-Webdav-Prefix":                  {"/arvados/v1/container_requests/" + cr.UUID + "/log/" + ctr.UUID},
			"X-Webdav-Source":                  {"/log"},
		}, &proxyErr).ServeHTTP(w, r)
		if proxyErr == nil {
			// proxy succeeded
			return
		}
		// If proxying to the container gateway fails, it
		// might be caused by a race where crunch-run exited
		// after we decided (above) the log was not final.
		// In that case we should proxy to keep-web.
		ctr, err := conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{
			UUID:   ctr.UUID,
			Select: []string{"uuid", "state", "gateway_address", "log"},
		})
		if err != nil {
			// Lost access to the container record?
			httpserver.Error(w, "error re-fetching container record: "+err.Error(), http.StatusServiceUnavailable)
		} else if ctr.State == arvados.ContainerStateLocked || ctr.State == arvados.ContainerStateRunning {
			// No race, proxyErr was the best we can do
			httpserver.Error(w, "proxy error: "+proxyErr.Error(), http.StatusServiceUnavailable)
		} else {
			conn.serveContainerRequestLogViaKeepWeb(opts, cr, w, r)
		}
	}), nil
}

// serveContainerLogViaKeepWeb handles a request for saved container
// log content by proxying to one of the configured keep-web servers.
//
// It tries to choose a keep-web server that is running on this host.
func (conn *Conn) serveContainerRequestLogViaKeepWeb(opts arvados.ContainerLogOptions, cr arvados.ContainerRequest, w http.ResponseWriter, r *http.Request) {
	if cr.LogUUID == "" {
		// Special case: if no log data exists yet, we serve
		// an empty collection by ourselves instead of
		// proxying to keep-web.
		conn.serveEmptyDir("/arvados/v1/container_requests/"+cr.UUID+"/log", w, r)
		return
	}
	myURL, _ := service.URLFromContext(r.Context())
	u := url.URL(myURL)
	myHostname := u.Hostname()
	var webdavBase arvados.URL
	var ok bool
	for webdavBase = range conn.cluster.Services.WebDAV.InternalURLs {
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
			r.URL.Scheme = webdavBase.Scheme
			r.URL.Host = webdavBase.Host
			// Outgoing Host header specifies the
			// collection ID.
			r.Host = cr.LogUUID + ".internal"
			// We already checked permission on the
			// container, so we can use a root token here
			// instead of counting on the "access to log
			// via container request and container"
			// permission check, which can be racy when a
			// request gets retried with a new container.
			r.Header.Set("Authorization", "Bearer "+conn.cluster.SystemRootToken)
			// We can't change r.URL.Path without
			// confusing WebDAV (request body and response
			// headers refer to the same paths) so we tell
			// keep-web to map the log collection onto the
			// containers/X/log/ namespace.
			r.Header.Set("X-Webdav-Prefix", "/arvados/v1/container_requests/"+cr.UUID+"/log")
			if len(opts.Path) >= 28 && opts.Path[6:13] == "-dz642-" {
				// "/arvados/v1/container_requests/{crUUID}/log/{cUUID}..."
				// proxies to
				// "/log for container {cUUID}..."
				r.Header.Set("X-Webdav-Prefix", "/arvados/v1/container_requests/"+cr.UUID+"/log/"+opts.Path[1:28])
				r.Header.Set("X-Webdav-Source", "/log for container "+opts.Path[1:28]+"/")
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			preemptivelyDeduplicateHeaders(w.Header(), resp.Header)
			return nil
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

func gatewayProxy(dial gatewayDialer, responseWriter http.ResponseWriter, setRequestHeader http.Header, proxyErr *error) *httputil.ReverseProxy {
	var proxyReq *http.Request
	var expectRespondAuth string
	return &httputil.ReverseProxy{
		// Our custom Transport:
		//
		// - Uses a custom dialer to connect to the gateway
		// (either directly or through a tunnel set up though
		// ContainerTunnel)
		//
		// - Verifies the gateway's TLS certificate using
		// X-Arvados-Authorization headers.
		//
		// This involves modifying the outgoing request header
		// in DialTLSContext.  (ReverseProxy certainly doesn't
		// expect us to do this, but it works.)
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
			// This transport is only used for a single
			// request, so http keep-alive would
			// accumulate open sockets without providing
			// any benefit.  So, disable keep-alive.
			DisableKeepAlives: true,
			// Use stdlib defaults.
			ForceAttemptHTTP2:     http.DefaultTransport.(*http.Transport).ForceAttemptHTTP2,
			TLSHandshakeTimeout:   http.DefaultTransport.(*http.Transport).TLSHandshakeTimeout,
			ExpectContinueTimeout: http.DefaultTransport.(*http.Transport).ExpectContinueTimeout,
		},
		Director: func(r *http.Request) {
			// Scheme/host of incoming r.URL are
			// irrelevant now, and may even be
			// missing. Host is ignored by our
			// DialTLSContext, but we need a generic
			// syntactically correct URL for net/http to
			// work with.
			r.URL.Scheme = "https"
			r.URL.Host = "0.0.0.0:0"
			for k, v := range setRequestHeader {
				r.Header[k] = v
			}
			proxyReq = r
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.Header.Get("X-Arvados-Authorization-Response") != expectRespondAuth {
				// Note this is how we detect
				// an attacker-in-the-middle.
				return httpserver.ErrorWithStatus(errors.New("bad X-Arvados-Authorization-Response header"), http.StatusBadGateway)
			}
			resp.Header.Del("X-Arvados-Authorization-Response")
			preemptivelyDeduplicateHeaders(responseWriter.Header(), resp.Header)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			if proxyErr != nil {
				*proxyErr = err
			}
		},
	}
}

// httputil.ReverseProxy uses (http.Header)Add() to copy headers from
// the upstream Response to the downstream ResponseWriter. If headers
// have already been set on the downstream ResponseWriter, Add() will
// result in duplicate headers. For example, if we set CORS headers
// and then use ReverseProxy with an upstream that also sets CORS
// headers, our client will receive
//
//	Access-Control-Allow-Origin: *
//	Access-Control-Allow-Origin: *
//
// ...which is incorrect.
//
// preemptivelyDeduplicateHeaders, when called from a ModifyResponse
// hook, solves this by removing any conflicting headers from
// ResponseWriter. This way, when ReverseProxy calls Add(), it will
// assign the new values without causing duplicates.
//
// dst is the downstream ResponseWriter's Header(). src is the
// upstream resp.Header.
func preemptivelyDeduplicateHeaders(dst, src http.Header) {
	for hdr := range src {
		dst.Del(hdr)
	}
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
			if err != nil && !os.IsNotExist(err) {
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
	if !user.IsAdmin || !conn.cluster.Containers.ShellAccess.Admin {
		if !conn.cluster.Containers.ShellAccess.User {
			return sshconn, httpserver.ErrorWithStatus(errors.New("shell access is disabled in config"), http.StatusServiceUnavailable)
		}
		err = conn.checkContainerLoginPermission(ctx, user.UUID, opts.UUID)
		if err != nil {
			return sshconn, err
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
		ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{conn.cluster.SystemRootToken}})
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

// Check that userUUID is permitted to start an interactive login
// session in ctrUUID.  Any returned error has an HTTPStatus().
func (conn *Conn) checkContainerLoginPermission(ctx context.Context, userUUID, ctrUUID string) error {
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{conn.cluster.SystemRootToken}})
	crs, err := conn.railsProxy.ContainerRequestList(ctxRoot, arvados.ListOptions{Limit: -1, Filters: []arvados.Filter{{"container_uuid", "=", ctrUUID}}})
	if err != nil {
		return err
	}
	for _, cr := range crs.Items {
		if cr.ModifiedByUserUUID != userUUID {
			return httpserver.ErrorWithStatus(errors.New("permission denied: container is associated with requests submitted by other users"), http.StatusForbidden)
		}
	}
	if crs.ItemsAvailable != len(crs.Items) {
		return httpserver.ErrorWithStatus(errors.New("incomplete response while checking permission"), http.StatusInternalServerError)
	}
	return nil
}

var errUnassignedPort = httpserver.ErrorWithStatus(errors.New("unassigned port"), http.StatusGone)

// ContainerHTTPProxy proxies an incoming request through to the
// specified port on a running container, via crunch-run's container
// gateway.
func (conn *Conn) ContainerHTTPProxy(ctx context.Context, opts arvados.ContainerHTTPProxyOptions) (http.Handler, error) {
	// We'll use ctxRoot to do requests below that don't depend on
	// the supplied token.
	ctxRoot := auth.NewContext(ctx, &auth.Credentials{Tokens: []string{conn.cluster.SystemRootToken}})

	needTokenCookie := false
	if queryToken := opts.Request.URL.Query().Get("arvados_api_token"); queryToken != "" {
		// If there's a token in the query, use it when
		// looking up the container/container request.  This
		// lets us reliably determine needClearSiteData, even
		// if the container UUID is not explicitly given in
		// the request.
		ctx = auth.NewContext(ctx, &auth.Credentials{Tokens: []string{queryToken}})
		// Ensure we ultimately send a redirect response to
		// move the token from query to cookie.
		needTokenCookie = true
	}

	var targetUUID string
	var targetPort int
	if strings.HasPrefix(opts.Target, ":") {
		// Target ":1234" means "the entry in the
		// container_ports table with external_port=1234".
		extport, err := strconv.Atoi(opts.Target[1:])
		if err != nil {
			return nil, httpserver.ErrorWithStatus(fmt.Errorf("invalid port in target: %s", opts.Target), http.StatusBadRequest)
		}
		db, err := conn.getdb(ctx)
		if err != nil {
			return nil, httpserver.ErrorWithStatus(fmt.Errorf("getdb: %w", err), http.StatusBadGateway)
		}
		err = db.QueryRowContext(ctx, `select container_uuid, container_port
			from container_ports
			where external_port = $1`, extport).Scan(&targetUUID, &targetPort)
		if err == sql.ErrNoRows {
			return nil, errUnassignedPort
		} else if err != nil {
			return nil, httpserver.ErrorWithStatus(err, http.StatusBadGateway)
		}
	} else if len(opts.Target) > 28 && arvadosclient.UUIDMatch(opts.Target[:27]) && opts.Target[27] == '-' {
		targetUUID = opts.Target[:27]
		fmt.Sscanf(opts.Target[28:], "%d", &targetPort)
		if targetPort < 1 {
			return nil, httpserver.ErrorWithStatus(fmt.Errorf("cannot parse port number from vhost prefix %q", opts.Target), http.StatusBadRequest)
		}
	} else {
		links, err := conn.railsProxy.LinkList(ctxRoot, arvados.ListOptions{
			Limit: 1,
			Filters: []arvados.Filter{
				{"link_class", "=", "published_port"},
				{"name", "=", opts.Target}}})
		if err != nil {
			return nil, fmt.Errorf("lookup failed: %w", err)
		}
		if len(links.Items) == 0 {
			return nil, httpserver.ErrorWithStatus(fmt.Errorf("container web service not found: %q", opts.Target), http.StatusNotFound)
		}
		targetUUID = links.Items[0].HeadUUID
		port, ok := links.Items[0].Properties["port"].(float64)
		targetPort = int(port)
		if !ok || targetPort < 1 || targetPort > 65535 {
			return nil, httpserver.ErrorWithStatus(fmt.Errorf("invalid port in published_port link: %v", links.Items[0].Properties["port"]), http.StatusInternalServerError)
		}
	}

	var ctr arvados.Container

	// A redirect might be needed for one or two reasons: (1) to
	// avoid letting the container web service access it via
	// document.location, or showing the token in the browser's
	// location bar (even when returning an error), and/or (2) to
	// clear client-side state left over from a different
	// container that was previously available on the same
	// dynamically assigned port.
	//
	// maybeRedirect() returns (nil, nil) if the given err is nil
	// and there is no need to redirect.  Otherwise, it returns
	// suitable values for the main function to return: either
	// (nil, err), or (h, nil) where h implements a redirect.
	maybeRedirect := func(err error) (http.Handler, error) {
		needClearSiteData := false
		for _, cookie := range opts.Request.CookiesNamed("arvados_container_uuid") {
			if cookie.Value != ctr.UUID && ctr.UUID != "" {
				// The user agent might have
				// client-side state left over from a
				// different container that was
				// previously available on this port.
				needClearSiteData = true
			}
		}
		if !needClearSiteData && !needTokenCookie {
			// Redirect not needed
			return nil, err
		}
		return containerHTTPProxyRedirect(needClearSiteData), nil
	}

	// First we need to fetch the container request (or container)
	// record as root, so we can check whether the requested port
	// is marked public in published_ports.  This needs to work
	// even if the request did not provide a token at all.
	var isPublic bool
	if len(targetUUID) == 27 && targetUUID[6:11] == "xvhdp" {
		// Look up specified container request
		ctrreq, err := conn.railsProxy.ContainerRequestGet(ctxRoot, arvados.GetOptions{
			UUID:   targetUUID,
			Select: []string{"uuid", "state", "published_ports", "container_uuid"},
		})
		if err == nil && ctrreq.PublishedPorts[strconv.Itoa(targetPort)].Access == arvados.PublishedPortAccessPublic {
			isPublic = true
			targetUUID = ctrreq.ContainerUUID
		}
	} else {
		// Look up specified container
		var err error
		ctr, err = conn.railsProxy.ContainerGet(ctxRoot, arvados.GetOptions{
			UUID:   targetUUID,
			Select: []string{"uuid", "state", "gateway_address", "published_ports"},
		})
		if err == nil && ctr.PublishedPorts[strconv.Itoa(targetPort)].Access == arvados.PublishedPortAccessPublic {
			isPublic = true
		}
	}

	if !isPublic {
		// Re-fetch the container request record, this time as
		// the authenticated user instead of root.  This lets
		// us return 404 if the container is not readable by
		// this user, for example.
		if len(targetUUID) == 27 && targetUUID[6:11] == "xvhdp" {
			ctrreq, err := conn.railsProxy.ContainerRequestGet(ctxRoot, arvados.GetOptions{
				UUID:   targetUUID,
				Select: []string{"uuid", "state", "published_ports", "container_uuid"},
			})
			if err != nil {
				return maybeRedirect(fmt.Errorf("container request lookup error: %w", err))
			}
			if ctrreq.ContainerUUID == "" {
				return maybeRedirect(httpserver.ErrorWithStatus(errors.New("container request does not have an assigned container"), http.StatusBadRequest))
			}
			targetUUID = ctrreq.ContainerUUID
		}
		var err error
		ctr, err = conn.railsProxy.ContainerGet(ctx, arvados.GetOptions{UUID: targetUUID, Select: []string{"uuid", "state", "gateway_address"}})
		if err != nil {
			return maybeRedirect(fmt.Errorf("container lookup failed: %w", err))
		}
		user, err := conn.railsProxy.UserGetCurrent(ctx, arvados.GetOptions{})
		if err != nil {
			return maybeRedirect(err)
		}
		if !user.IsAdmin {
			// For non-public ports, access is only granted to
			// admins and the user who submitted all of the
			// container requests that reference this container.
			err = conn.checkContainerLoginPermission(ctx, user.UUID, ctr.UUID)
			if err != nil {
				return maybeRedirect(err)
			}
		}
	} else if ctr.UUID == "" {
		// isPublic, but we don't have the container record
		// yet because the request specified a container
		// request UUID.
		var err error
		ctr, err = conn.railsProxy.ContainerGet(ctxRoot, arvados.GetOptions{UUID: targetUUID, Select: []string{"uuid", "state", "gateway_address"}})
		if err != nil {
			return maybeRedirect(fmt.Errorf("container lookup failed: %w", err))
		}
	}
	dial, arpc, err := conn.findGateway(ctx, ctr, opts.NoForward)
	if err != nil {
		return maybeRedirect(fmt.Errorf("cannot find gateway: %w", err))
	}
	if arpc != nil {
		if h, err := maybeRedirect(nil); h != nil || err != nil {
			return h, err
		}
		opts.NoForward = true
		return arpc.ContainerHTTPProxy(ctx, opts)
	}

	// Redirect if needed to clear site data and/or move the token
	// from the query to a cookie.
	if h, err := maybeRedirect(nil); h != nil || err != nil {
		return h, err
	}

	// Remove arvados_api_token cookie to ensure the http service
	// in the container does not see it.
	cookies := opts.Request.Cookies()
	opts.Request.Header.Del("Cookie")
	for _, cookie := range cookies {
		if cookie.Name != "arvados_api_token" {
			opts.Request.AddCookie(cookie)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "arvados_container_uuid", Value: ctr.UUID})
		var proxyErr error
		gatewayProxy(dial, w, http.Header{
			"X-Arvados-Container-Gateway-Uuid": {targetUUID},
			"X-Arvados-Container-Target-Port":  {strconv.Itoa(targetPort)},
		}, &proxyErr).ServeHTTP(w, opts.Request)
		if proxyErr != nil {
			httpserver.Error(w, "proxy error: "+proxyErr.Error(), http.StatusBadGateway)
		}
	}), nil
}

// containerHTTPProxyRedirect returns a redirect handler that (1) if
// there is a token in the query, moves it to a cookie, and (2) if
// needClearSiteData is true, clears all other client-side state.
func containerHTTPProxyRedirect(needClearSiteData bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redir := *r.URL
		query := redir.Query()
		needTokenCookie := query.Get("arvados_api_token")
		if needTokenCookie != "" {
			delete(query, "arvados_api_token")
			redir.RawQuery = query.Encode()
		}
		if needTokenCookie != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "arvados_api_token",
				Value:    auth.EncodeTokenCookie([]byte(needTokenCookie)),
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
		if needClearSiteData {
			if r.Method != http.MethodHead && r.Method != http.MethodGet {
				w.WriteHeader(http.StatusGone)
				return
			}
			// We cannot use `Clear-Site-Data: "cookies"`
			// to clear cookies, because that applies to
			// all origins in the entire registered
			// domain.  We only want to clear cookies for
			// this dynamically assigned origin.
			for _, cookie := range r.Cookies() {
				if cookie.Name != "arvados_api_token" {
					cookie.MaxAge = -1
					cookie.Expires = time.Time{}
					cookie.Value = ""
					http.SetCookie(w, cookie)
				}
			}
			// Unlike the "cookies" directive, "cache" and
			// "storage" clear data for the current origin
			// only.
			w.Header().Set("Clear-Site-Data", `"cache", "storage"`)
		}
		w.Header().Set("Location", redir.String())
		w.WriteHeader(http.StatusSeeOther)
	})
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

	if host, _, _ := net.SplitHostPort(ctr.GatewayAddress); host != "" &&
		(host != "127.0.0.1" || conn.cluster.Containers.CloudVMs.Driver == "loopback") {
		// If crunch-run provided a GatewayAddress like
		// host:port or [host]:port, and host is realistic
		// (127.0.0.1 is realistic only if we're running the
		// loopback driver), that means "ipaddr" is one of the
		// external interfaces where the gateway is
		// listening. In that case, it's the most
		// reliable/direct option, so we use it even if a
		// tunnel might also be available.
		return func() (net.Conn, string, string, error) {
			rawconn, err := (&net.Dialer{}).DialContext(ctx, "tcp", ctr.GatewayAddress)
			if err != nil {
				return nil, "", "", httpserver.ErrorWithStatus(err, http.StatusServiceUnavailable)
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
				return nil, "", "", httpserver.ErrorWithStatus(err, http.StatusServiceUnavailable)
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
