// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/lib/selfsigned"
	"git.arvados.org/arvados.git/lib/webdavfs"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/httpserver"
	"github.com/creack/pty"
	"github.com/google/shlex"
	"github.com/hashicorp/yamux"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/webdav"
)

type GatewayTarget interface {
	// Command that will execute cmd inside the container
	InjectCommand(ctx context.Context, detachKeys, username string, usingTTY bool, cmd []string) (*exec.Cmd, error)

	// IP address inside container
	IPAddress() (string, error)
}

type GatewayTargetStub struct{}

func (GatewayTargetStub) IPAddress() (string, error) { return "127.0.0.1", nil }
func (GatewayTargetStub) InjectCommand(ctx context.Context, detachKeys, username string, usingTTY bool, cmd []string) (*exec.Cmd, error) {
	return exec.CommandContext(ctx, cmd[0], cmd[1:]...), nil
}

type Gateway struct {
	ContainerUUID string
	// Caller should set Address to "", or "host:0" or "host:port"
	// where host is a known external IP address; port is a
	// desired port number to listen on; and ":0" chooses an
	// available dynamic port.
	//
	// If Address is "", Start() listens only on the loopback
	// interface (and changes Address to "127.0.0.1:port").
	// Otherwise it listens on all interfaces.
	//
	// If Address is "host:0", Start() updates Address to
	// "host:port".
	Address    string
	AuthSecret string
	Target     GatewayTarget
	Log        interface {
		Printf(fmt string, args ...interface{})
	}
	// If non-nil, set up a ContainerGatewayTunnel, so that the
	// controller can connect to us even if our external IP
	// address is unknown or not routable from controller.
	ArvadosClient *arvados.Client

	// When a tunnel is connected or reconnected, this func (if
	// not nil) will be called with the InternalURL of the
	// controller process at the other end of the tunnel.
	UpdateTunnelURL func(url string)

	// Source for serving WebDAV requests at /arvados/v1/{uuid}/log/
	LogCollection arvados.CollectionFileSystem

	sshConfig   ssh.ServerConfig
	requestAuth string
	respondAuth string
	logPath     string
}

// Start starts an http server that allows authenticated clients to open an
// interactive "docker exec" session and (in future) connect to tcp ports
// inside the docker container.
func (gw *Gateway) Start() error {
	gw.sshConfig = ssh.ServerConfig{
		NoClientAuth: true,
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "_" {
				return nil, nil
			}
			return nil, fmt.Errorf("cannot specify user %q via ssh client", c.User())
		},
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if c.User() == "_" {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("cannot specify user %q via ssh client", c.User())
		},
	}
	pvt, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	err = pvt.Validate()
	if err != nil {
		return err
	}
	signer, err := ssh.NewSignerFromKey(pvt)
	if err != nil {
		return err
	}
	gw.sshConfig.AddHostKey(signer)

	// Address (typically provided by arvados-dispatch-cloud) is
	// HOST:PORT where HOST is our IP address or hostname as seen
	// from arvados-controller, and PORT is either the desired
	// port where we should run our gateway server, or "0" if we
	// should choose an available port.
	extAddr := gw.Address
	// Generally we can't know which local interface corresponds
	// to an externally reachable IP address, so if we expect to
	// be reachable by external hosts, we listen on all
	// interfaces.
	listenHost := ""
	if extAddr == "" {
		// If the dispatcher doesn't tell us our external IP
		// address, controller will only be able to connect
		// through the tunnel (see runTunnel), so our gateway
		// server only needs to listen on the loopback
		// interface.
		extAddr = "127.0.0.1:0"
		listenHost = "127.0.0.1"
	}
	extHost, extPort, err := net.SplitHostPort(extAddr)
	if err != nil {
		return err
	}
	cert, err := selfsigned.CertGenerator{}.Generate()
	if err != nil {
		return err
	}
	h := hmac.New(sha256.New, []byte(gw.AuthSecret))
	h.Write(cert.Certificate[0])
	gw.requestAuth = fmt.Sprintf("%x", h.Sum(nil))
	h.Reset()
	h.Write([]byte(gw.requestAuth))
	gw.respondAuth = fmt.Sprintf("%x", h.Sum(nil))

	gw.logPath = "/arvados/v1/containers/" + gw.ContainerUUID + "/log"

	srv := &httpserver.Server{
		Server: http.Server{
			Handler: gw,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		},
		Addr: net.JoinHostPort(listenHost, extPort),
	}
	err = srv.Start()
	if err != nil {
		return err
	}
	go func() {
		err := srv.Wait()
		gw.Log.Printf("gateway server stopped: %s", err)
	}()
	// Get the port number we are listening on (extPort might be
	// "0" or a port name, in which case this will be different).
	_, listenPort, err := net.SplitHostPort(srv.Addr)
	if err != nil {
		return err
	}
	// When changing state to Running, the caller will want to set
	// gateway_address to a "HOST:PORT" that, if controller
	// connects to it, will reach this gateway server.
	//
	// The most likely thing to work is: HOST is our external
	// hostname/IP as provided by the caller
	// (arvados-dispatch-cloud) or 127.0.0.1 to indicate
	// non-tunnel connections aren't available; and PORT is the
	// port number we are listening on.
	gw.Address = net.JoinHostPort(extHost, listenPort)
	gw.Log.Printf("gateway server listening at %s", gw.Address)
	if gw.ArvadosClient != nil {
		go gw.maintainTunnel(gw.Address)
	}
	return nil
}

func (gw *Gateway) maintainTunnel(addr string) {
	for ; ; time.Sleep(5 * time.Second) {
		err := gw.runTunnel(addr)
		gw.Log.Printf("runTunnel: %s", err)
	}
}

// runTunnel connects to controller and sets up a tunnel through
// which controller can connect to the gateway server at the given
// addr.
func (gw *Gateway) runTunnel(addr string) error {
	ctx := auth.NewContext(context.Background(), auth.NewCredentials(gw.ArvadosClient.AuthToken))
	arpc := rpc.NewConn("", &url.URL{Scheme: "https", Host: gw.ArvadosClient.APIHost}, gw.ArvadosClient.Insecure, rpc.PassthroughTokenProvider)
	tun, err := arpc.ContainerGatewayTunnel(ctx, arvados.ContainerGatewayTunnelOptions{
		UUID:       gw.ContainerUUID,
		AuthSecret: gw.AuthSecret,
	})
	if err != nil {
		return fmt.Errorf("error creating gateway tunnel: %s", err)
	}
	mux, err := yamux.Client(tun.Conn, nil)
	if err != nil {
		return fmt.Errorf("error setting up mux client end: %s", err)
	}
	if url := tun.Header.Get("X-Arvados-Internal-Url"); url != "" && gw.UpdateTunnelURL != nil {
		gw.UpdateTunnelURL(url)
	}
	for {
		muxconn, err := mux.AcceptStream()
		if err != nil {
			return err
		}
		gw.Log.Printf("tunnel connection %d started", muxconn.StreamID())
		go func() {
			defer muxconn.Close()
			gwconn, err := net.Dial("tcp", addr)
			if err != nil {
				gw.Log.Printf("tunnel connection %d: error connecting to %s: %s", muxconn.StreamID(), addr, err)
				return
			}
			defer gwconn.Close()
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				_, err := io.Copy(gwconn, muxconn)
				if err != nil {
					gw.Log.Printf("tunnel connection %d: mux end: %s", muxconn.StreamID(), err)
				}
				gwconn.Close()
			}()
			go func() {
				defer wg.Done()
				_, err := io.Copy(muxconn, gwconn)
				if err != nil {
					gw.Log.Printf("tunnel connection %d: gateway end: %s", muxconn.StreamID(), err)
				}
				muxconn.Close()
			}()
			wg.Wait()
			gw.Log.Printf("tunnel connection %d finished", muxconn.StreamID())
		}()
	}
}

var webdavMethod = map[string]bool{
	"GET":      true,
	"OPTIONS":  true,
	"PROPFIND": true,
}

func (gw *Gateway) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	reqUUID := req.Header.Get("X-Arvados-Container-Gateway-Uuid")
	if reqUUID == "" {
		// older controller versions only send UUID as query param
		req.ParseForm()
		reqUUID = req.Form.Get("uuid")
	}
	if reqUUID != gw.ContainerUUID {
		http.Error(w, fmt.Sprintf("misdirected request: meant for %q but received by crunch-run %q", reqUUID, gw.ContainerUUID), http.StatusBadGateway)
		return
	}
	if req.Header.Get("X-Arvados-Authorization") != gw.requestAuth {
		http.Error(w, "bad X-Arvados-Authorization header", http.StatusUnauthorized)
		return
	}
	w.Header().Set("X-Arvados-Authorization-Response", gw.respondAuth)
	switch {
	case req.Method == "POST" && req.Header.Get("Upgrade") == "ssh":
		gw.handleSSH(w, req)
	case req.URL.Path == gw.logPath || strings.HasPrefix(req.URL.Path, gw.logPath):
		if !webdavMethod[req.Method] {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		gw.handleLogsWebDAV(w, req)
	default:
		http.Error(w, "path not found", http.StatusNotFound)
	}
}

func (gw *Gateway) handleLogsWebDAV(w http.ResponseWriter, r *http.Request) {
	if gw.LogCollection == nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	wh := webdav.Handler{
		Prefix: gw.logPath,
		FileSystem: &webdavfs.FS{
			FileSystem:    gw.LogCollection,
			Prefix:        "",
			Writing:       false,
			AlwaysReadEOF: r.Method == "PROPFIND",
		},
		LockSystem: webdavfs.NoLockSystem,
		Logger:     gw.webdavLogger,
	}
	wh.ServeHTTP(w, r)
}

func (gw *Gateway) webdavLogger(r *http.Request, err error) {
	if err != nil {
		ctxlog.FromContext(r.Context()).WithError(err).Error("error reported by webdav handler")
	}
}

// handleSSH connects to an SSH server that allows the caller to run
// interactive commands as root (or any other desired user) inside the
// container. The tunnel itself can only be created by an
// authenticated caller, so the SSH server itself is wide open (any
// password or key will be accepted).
//
// Requests must have path "/ssh" and the following headers:
//
// Connection: upgrade
// Upgrade: ssh
// X-Arvados-Target-Uuid: uuid of container
// X-Arvados-Authorization: must match
// hmac(AuthSecret,certfingerprint) (this prevents other containers
// and shell nodes from connecting directly)
//
// Optional headers:
//
// X-Arvados-Detach-Keys: argument to "docker exec --detach-keys",
// e.g., "ctrl-p,ctrl-q"
// X-Arvados-Login-Username: argument to "docker exec --user": account
// used to run command(s) inside the container.
func (gw *Gateway) handleSSH(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	detachKeys := req.Form.Get("detach_keys")
	username := req.Form.Get("login_username")
	if username == "" {
		username = "root"
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "ResponseWriter does not support connection upgrade", http.StatusInternalServerError)
		return
	}
	netconn, _, err := hj.Hijack()
	if !ok {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer netconn.Close()
	w.Header().Set("Connection", "upgrade")
	w.Header().Set("Upgrade", "ssh")
	netconn.Write([]byte("HTTP/1.1 101 Switching Protocols\r\n"))
	w.Header().Write(netconn)
	netconn.Write([]byte("\r\n"))

	ctx := req.Context()

	conn, newchans, reqs, err := ssh.NewServerConn(netconn, &gw.sshConfig)
	if err == io.EOF {
		return
	} else if err != nil {
		gw.Log.Printf("ssh.NewServerConn: %s", err)
		return
	}
	defer conn.Close()
	go ssh.DiscardRequests(reqs)
	for newch := range newchans {
		switch newch.ChannelType() {
		case "direct-tcpip":
			go gw.handleDirectTCPIP(ctx, newch)
		case "session":
			go gw.handleSession(ctx, newch, detachKeys, username)
		default:
			go newch.Reject(ssh.UnknownChannelType, fmt.Sprintf("unsupported channel type %q", newch.ChannelType()))
		}
	}
}

func (gw *Gateway) handleDirectTCPIP(ctx context.Context, newch ssh.NewChannel) {
	ch, reqs, err := newch.Accept()
	if err != nil {
		gw.Log.Printf("accept direct-tcpip channel: %s", err)
		return
	}
	defer ch.Close()
	go ssh.DiscardRequests(reqs)

	// RFC 4254 7.2 (copy of channelOpenDirectMsg in
	// golang.org/x/crypto/ssh)
	var msg struct {
		Raddr string
		Rport uint32
		Laddr string
		Lport uint32
	}
	err = ssh.Unmarshal(newch.ExtraData(), &msg)
	if err != nil {
		fmt.Fprintf(ch.Stderr(), "unmarshal direct-tcpip extradata: %s\n", err)
		return
	}
	switch msg.Raddr {
	case "localhost", "0.0.0.0", "127.0.0.1", "::1", "::":
	default:
		fmt.Fprintf(ch.Stderr(), "cannot forward to ports on %q, only localhost\n", msg.Raddr)
		return
	}

	dstaddr, err := gw.Target.IPAddress()
	if err != nil {
		fmt.Fprintf(ch.Stderr(), "container has no IP address: %s\n", err)
		return
	} else if dstaddr == "" {
		fmt.Fprintf(ch.Stderr(), "container has no IP address\n")
		return
	}

	dst := net.JoinHostPort(dstaddr, fmt.Sprintf("%d", msg.Rport))
	tcpconn, err := net.Dial("tcp", dst)
	if err != nil {
		fmt.Fprintf(ch.Stderr(), "%s: %s\n", dst, err)
		return
	}
	go func() {
		n, _ := io.Copy(ch, tcpconn)
		ctxlog.FromContext(ctx).Debugf("tcpip: sent %d bytes\n", n)
		ch.CloseWrite()
	}()
	n, _ := io.Copy(tcpconn, ch)
	ctxlog.FromContext(ctx).Debugf("tcpip: received %d bytes\n", n)
}

func (gw *Gateway) handleSession(ctx context.Context, newch ssh.NewChannel, detachKeys, username string) {
	ch, reqs, err := newch.Accept()
	if err != nil {
		gw.Log.Printf("error accepting session channel: %s", err)
		return
	}
	defer ch.Close()

	var pty0, tty0 *os.File
	// Where to send errors/messages for the client to see
	logw := io.Writer(ch.Stderr())
	// How to end lines when sending errors/messages to the client
	// (changes to \r\n when using a pty)
	eol := "\n"
	// Env vars to add to child process
	termEnv := []string(nil)

	started := 0
	wantClose := make(chan struct{})
	for {
		var req *ssh.Request
		select {
		case r, ok := <-reqs:
			if !ok {
				return
			}
			req = r
		case <-wantClose:
			return
		}
		ok := false
		switch req.Type {
		case "shell", "exec":
			if started++; started != 1 {
				// RFC 4254 6.5: "Only one of these
				// requests can succeed per channel."
				break
			}
			ok = true
			var payload struct {
				Command string
			}
			ssh.Unmarshal(req.Payload, &payload)
			execargs, err := shlex.Split(payload.Command)
			if err != nil {
				fmt.Fprintf(logw, "error parsing supplied command: %s"+eol, err)
				return
			}
			if len(execargs) == 0 {
				execargs = []string{"/bin/bash", "-login"}
			}
			go func() {
				var resp struct {
					Status uint32
				}
				defer func() {
					ch.SendRequest("exit-status", false, ssh.Marshal(&resp))
					close(wantClose)
				}()

				cmd, err := gw.Target.InjectCommand(ctx, detachKeys, username, tty0 != nil, execargs)
				if err != nil {
					fmt.Fprintln(ch.Stderr(), err)
					ch.CloseWrite()
					resp.Status = 1
					return
				}
				if tty0 != nil {
					cmd.Stdin = tty0
					cmd.Stdout = tty0
					cmd.Stderr = tty0
					go io.Copy(ch, pty0)
					go io.Copy(pty0, ch)
					// Send our own debug messages to tty as well.
					logw = tty0
				} else {
					// StdinPipe may seem
					// superfluous here, but it's
					// not: it causes cmd.Run() to
					// return when the subprocess
					// exits. Without it, Run()
					// waits for stdin to close,
					// which causes "ssh ... echo
					// ok" (with the client's
					// stdin connected to a
					// terminal or something) to
					// hang.
					stdin, err := cmd.StdinPipe()
					if err != nil {
						fmt.Fprintln(ch.Stderr(), err)
						ch.CloseWrite()
						resp.Status = 1
						return
					}
					go func() {
						io.Copy(stdin, ch)
						stdin.Close()
					}()
					cmd.Stdout = ch
					cmd.Stderr = ch.Stderr()
				}
				cmd.SysProcAttr = &syscall.SysProcAttr{
					Setctty: tty0 != nil,
					Setsid:  true,
				}
				cmd.Env = append(os.Environ(), termEnv...)
				err = cmd.Run()
				if exiterr, ok := err.(*exec.ExitError); ok {
					if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
						resp.Status = uint32(status.ExitStatus())
					}
				} else if err != nil {
					// Propagate errors like `exec: "docker": executable file not found in $PATH`
					fmt.Fprintln(ch.Stderr(), err)
				}
				errClose := ch.CloseWrite()
				if resp.Status == 0 && (err != nil || errClose != nil) {
					resp.Status = 1
				}
			}()
		case "pty-req":
			eol = "\r\n"
			p, t, err := pty.Open()
			if err != nil {
				fmt.Fprintf(ch.Stderr(), "pty failed: %s"+eol, err)
				break
			}
			defer p.Close()
			defer t.Close()
			pty0, tty0 = p, t
			ok = true
			var payload struct {
				Term string
				Cols uint32
				Rows uint32
				X    uint32
				Y    uint32
			}
			ssh.Unmarshal(req.Payload, &payload)
			termEnv = []string{"TERM=" + payload.Term, "USE_TTY=1"}
			err = pty.Setsize(pty0, &pty.Winsize{Rows: uint16(payload.Rows), Cols: uint16(payload.Cols), X: uint16(payload.X), Y: uint16(payload.Y)})
			if err != nil {
				fmt.Fprintf(logw, "pty-req: setsize failed: %s"+eol, err)
			}
		case "window-change":
			var payload struct {
				Cols uint32
				Rows uint32
				X    uint32
				Y    uint32
			}
			ssh.Unmarshal(req.Payload, &payload)
			err := pty.Setsize(pty0, &pty.Winsize{Rows: uint16(payload.Rows), Cols: uint16(payload.Cols), X: uint16(payload.X), Y: uint16(payload.Y)})
			if err != nil {
				fmt.Fprintf(logw, "window-change: setsize failed: %s"+eol, err)
				break
			}
			ok = true
		case "env":
			// TODO: implement "env"
			// requests by setting env
			// vars in the docker-exec
			// command (not docker-exec's
			// own environment, which
			// would be a gaping security
			// hole).
		default:
			// fmt.Fprintf(logw, "declined request %q on ssh channel"+eol, req.Type)
		}
		if req.WantReply {
			req.Reply(ok, nil)
		}
	}
}
