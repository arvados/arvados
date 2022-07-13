// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package install

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"text/template"
	"time"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/lib/controller/rpc"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/auth"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"github.com/lib/pq"
)

var InitCommand cmd.Handler = &initCommand{}

type initCommand struct {
	ClusterID          string
	Domain             string
	PostgreSQLPassword string
	Login              string
	TLS                string
	AdminEmail         string
	Start              bool

	LoginPAM                bool
	LoginTest               bool
	LoginGoogle             bool
	LoginGoogleClientID     string
	LoginGoogleClientSecret string
}

func (initcmd *initCommand) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	logger := ctxlog.New(stderr, "text", "info")
	ctx := ctxlog.Context(context.Background(), logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var err error
	defer func() {
		if err != nil {
			logger.WithError(err).Info("exiting")
		}
	}()

	hostname, err := os.Hostname()
	if err != nil {
		err = fmt.Errorf("Hostname(): %w", err)
		return 1
	}

	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	flags.SetOutput(stderr)
	versionFlag := flags.Bool("version", false, "Write version information to stdout and exit 0")
	flags.StringVar(&initcmd.ClusterID, "cluster-id", "", "cluster `id`, like x1234 for a dev cluster")
	flags.StringVar(&initcmd.Domain, "domain", hostname, "cluster public DNS `name`, like x1234.arvadosapi.com")
	flags.StringVar(&initcmd.Login, "login", "", "login `backend`: test, pam, 'google {client-id} {client-secret}', or ''")
	flags.StringVar(&initcmd.AdminEmail, "admin-email", "", "give admin privileges to user with given `email`")
	flags.StringVar(&initcmd.TLS, "tls", "none", "tls certificate `source`: acme, auto, insecure, or none")
	flags.BoolVar(&initcmd.Start, "start", true, "start systemd service after creating config")
	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		return code
	} else if *versionFlag {
		return cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
	} else if !regexp.MustCompile(`^[a-z][a-z0-9]{4}`).MatchString(initcmd.ClusterID) {
		err = fmt.Errorf("cluster ID %q is invalid; must be an ASCII letter followed by 4 alphanumerics (try -help)", initcmd.ClusterID)
		return 1
	}

	if fields := strings.Fields(initcmd.Login); len(fields) == 3 && fields[0] == "google" {
		initcmd.LoginGoogle = true
		initcmd.LoginGoogleClientID = fields[1]
		initcmd.LoginGoogleClientSecret = fields[2]
	} else if initcmd.Login == "test" {
		initcmd.LoginTest = true
		if initcmd.AdminEmail == "" {
			initcmd.AdminEmail = "admin@example.com"
		}
	} else if initcmd.Login == "pam" {
		initcmd.LoginPAM = true
	} else if initcmd.Login == "" {
		// none; login will show an error page
	} else {
		err = fmt.Errorf("invalid argument to -login: %q: should be 'test', 'pam', 'google {client-id} {client-secret}', or empty", initcmd.Login)
		return 1
	}

	confdir := "/etc/arvados"
	conffile := confdir + "/config.yml"
	if _, err = os.Stat(conffile); err == nil {
		err = fmt.Errorf("config file %s already exists; delete it first if you really want to start over", conffile)
		return 1
	}

	err = initcmd.checkPort(ctx, "4440")
	err = initcmd.checkPort(ctx, "443")
	if initcmd.TLS == "auto" {
		err = initcmd.checkPort(ctx, "80")
		if err != nil {
			return 1
		}
	}

	// Do the "create extension" thing early. This way, if there's
	// no local postgresql server (a likely failure mode), we can
	// bail out without any side effects, and the user can start
	// over easily.
	fmt.Fprintln(stderr, "installing pg_trgm postgresql extension...")
	cmd := exec.CommandContext(ctx, "sudo", "-u", "postgres", "psql", "--quiet",
		"-c", `CREATE EXTENSION IF NOT EXISTS pg_trgm`)
	cmd.Dir = "/"
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("error preparing postgresql server: %w", err)
		return 1
	}
	fmt.Fprintln(stderr, "...done")

	wwwuser, err := user.Lookup("www-data")
	if err != nil {
		err = fmt.Errorf("user.Lookup(%q): %w", "www-data", err)
		return 1
	}
	wwwgid, err := strconv.Atoi(wwwuser.Gid)
	if err != nil {
		return 1
	}
	initcmd.PostgreSQLPassword = initcmd.RandomHex(32)

	fmt.Fprintln(stderr, "creating data storage directory /var/lib/arvados/keep ...")
	err = os.Mkdir("/var/lib/arvados/keep", 0600)
	if err != nil && !os.IsExist(err) {
		err = fmt.Errorf("mkdir /var/lib/arvados/keep: %w", err)
		return 1
	}
	fmt.Fprintln(stderr, "...done")

	fmt.Fprintln(stderr, "creating config file", conffile, "...")
	err = os.Mkdir(confdir, 0750)
	if err != nil && !os.IsExist(err) {
		err = fmt.Errorf("mkdir %s: %w", confdir, err)
		return 1
	}
	err = os.Chown(confdir, 0, wwwgid)
	if err != nil {
		err = fmt.Errorf("chown 0:%d %s: %w", wwwgid, confdir, err)
		return 1
	}
	f, err := os.OpenFile(conffile+".tmp", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		err = fmt.Errorf("open %s: %w", conffile+".tmp", err)
		return 1
	}
	tmpl, err := template.New("config").Parse(`Clusters:
  {{.ClusterID}}:
    Services:
      Controller:
        InternalURLs:
          "http://0.0.0.0:9000/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4440/" ) }}
      RailsAPI:
        InternalURLs:
          "http://0.0.0.0:9001/": {}
      Websocket:
        InternalURLs:
          "http://0.0.0.0:8005/": {}
        ExternalURL: {{printf "%q" ( print "wss://" .Domain ":4446/" ) }}
      Keepbalance:
        InternalURLs:
          "http://0.0.0.0:9019/": {}
      GitHTTP:
        InternalURLs:
          "http://0.0.0.0:9005/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4445/" ) }}
      DispatchCloud:
        InternalURLs:
          "http://0.0.0.0:9006/": {}
      Keepproxy:
        InternalURLs:
          "http://0.0.0.0:9007/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4447/" ) }}
      WebDAV:
        InternalURLs:
          "http://0.0.0.0:9008/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4448/" ) }}
      WebDAVDownload:
        InternalURLs:
          "http://0.0.0.0:9009/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4449/" ) }}
      Keepstore:
        InternalURLs:
          "http://0.0.0.0:9010/": {}
      Composer:
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4459/composer" ) }}
      Workbench1:
        InternalURLs:
          "http://0.0.0.0:9002/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4442/" ) }}
      Workbench2:
        InternalURLs:
          "http://0.0.0.0:9003/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain "/" ) }}
      Health:
        InternalURLs:
          "http://0.0.0.0:9011/": {}
    Collections:
      BlobSigningKey: {{printf "%q" ( .RandomHex 50 )}}
      {{if eq .TLS "insecure"}}
      TrustAllContent: true
      {{end}}
    Containers:
      DispatchPrivateKey: {{printf "%q" .GenerateSSHPrivateKey}}
      CloudVMs:
        Enable: true
        Driver: loopback
    ManagementToken: {{printf "%q" ( .RandomHex 50 )}}
    PostgreSQL:
      Connection:
        dbname: arvados
        host: localhost
        user: arvados
        password: {{printf "%q" .PostgreSQLPassword}}
    SystemRootToken: {{printf "%q" ( .RandomHex 50 )}}
    TLS:
      {{if eq .TLS "insecure"}}
      Insecure: true
      {{else if eq .TLS "auto"}}
      Automatic: true
      {{else if eq .TLS "acme"}}
      Certificate: {{printf "%q" (print "/var/lib/acme/live/" .Domain "/cert")}}
      Key: {{printf "%q" (print "/var/lib/acme/live/" .Domain "/privkey")}}
      {{else}}
      {}
      {{end}}
    Volumes:
      {{.ClusterID}}-nyw5e-000000000000000:
        Driver: Directory
        DriverParameters:
          Root: /var/lib/arvados/keep
        Replication: 2
    Workbench:
      SecretKeyBase: {{printf "%q" ( .RandomHex 50 )}}
    {{if .LoginPAM}}
    Login:
      PAM:
        Enable: true
    {{else if .LoginTest}}
    Login:
      Test:
        Enable: true
        Users:
          admin:
            Email: {{printf "%q" .AdminEmail}}
            Password: admin
    {{else if .LoginGoogle}}
    Login:
      Google:
        Enable: true
        ClientID: {{printf "%q" .LoginGoogleClientID}}
        ClientSecret: {{printf "%q" .LoginGoogleClientSecret}}
    {{end}}
    Users:
      AutoAdminUserWithEmail: {{printf "%q" .AdminEmail}}
`)
	if err != nil {
		return 1
	}
	err = tmpl.Execute(f, initcmd)
	if err != nil {
		err = fmt.Errorf("%s: tmpl.Execute: %w", conffile+".tmp", err)
		return 1
	}
	err = f.Close()
	if err != nil {
		err = fmt.Errorf("%s: close: %w", conffile+".tmp", err)
		return 1
	}
	err = os.Rename(conffile+".tmp", conffile)
	if err != nil {
		err = fmt.Errorf("rename %s -> %s: %w", conffile+".tmp", conffile, err)
		return 1
	}
	fmt.Fprintln(stderr, "...done")

	ldr := config.NewLoader(nil, logger)
	ldr.SkipLegacy = true
	ldr.Path = conffile // load the file we just wrote, even if $ARVADOS_CONFIG is set
	cfg, err := ldr.Load()
	if err != nil {
		err = fmt.Errorf("%s: %w", conffile, err)
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}

	fmt.Fprintln(stderr, "creating postresql user and database...")
	err = initcmd.createDB(ctx, cluster.PostgreSQL.Connection, stderr)
	if err != nil {
		return 1
	}
	fmt.Fprintln(stderr, "...done")

	fmt.Fprintln(stderr, "initializing database...")
	cmd = exec.CommandContext(ctx, "sudo", "-u", "www-data", "-E", "HOME=/var/www", "PATH=/var/lib/arvados/bin:"+os.Getenv("PATH"), "/var/lib/arvados/bin/bundle", "exec", "rake", "db:setup")
	cmd.Dir = "/var/lib/arvados/railsapi"
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("rake db:setup failed: %w", err)
		return 1
	}
	fmt.Fprintln(stderr, "...done")

	if initcmd.Start {
		fmt.Fprintln(stderr, "starting systemd service...")
		cmd := exec.CommandContext(ctx, "systemctl", "start", "arvados")
		cmd.Dir = "/"
		cmd.Stdout = stderr
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("%v: %w", cmd.Args, err)
			return 1
		}
		fmt.Fprintln(stderr, "...done")

		fmt.Fprintln(stderr, "checking controller API endpoint...")
		u := url.URL(cluster.Services.Controller.ExternalURL)
		conn := rpc.NewConn(cluster.ClusterID, &u, cluster.TLS.Insecure, rpc.PassthroughTokenProvider)
		ctx := auth.NewContext(context.Background(), auth.NewCredentials(cluster.SystemRootToken))
		_, err = conn.UserGetCurrent(ctx, arvados.GetOptions{})
		if err != nil {
			err = fmt.Errorf("API request failed: %w", err)
			return 1
		}
		fmt.Fprintln(stderr, "...looks good")
	}

	if out, err := exec.CommandContext(ctx, "docker", "version").CombinedOutput(); err == nil && strings.Contains(string(out), "\nServer:\n") {
		fmt.Fprintln(stderr, "loading alpine docker image for diagnostics...")
		cmd := exec.CommandContext(ctx, "docker", "pull", "alpine")
		cmd.Stdout = stderr
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("%v: %w", cmd.Args, err)
			return 1
		}
		cmd = exec.CommandContext(ctx, "arv", "sudo", "keep", "docker", "alpine")
		cmd.Stdout = stderr
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("%v: %w", cmd.Args, err)
			return 1
		}
		fmt.Fprintln(stderr, "...done")
	} else {
		fmt.Fprintln(stderr, "docker is not installed -- skipping step of downloading 'alpine' image")
	}

	fmt.Fprintln(stderr, "Setup complete. You should now be able to log in to workbench2 at", cluster.Services.Workbench2.ExternalURL.String())

	return 0
}

func (initcmd *initCommand) GenerateSSHPrivateKey() (string, error) {
	privkey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", err
	}
	err = privkey.Validate()
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privkey),
	})), nil
}

func (initcmd *initCommand) RandomHex(chars int) string {
	b := make([]byte, chars/2)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", b)
}

func (initcmd *initCommand) createDB(ctx context.Context, dbconn arvados.PostgreSQLConnection, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "sudo", "-u", "postgres", "psql", "--quiet",
		"-c", `CREATE USER `+pq.QuoteIdentifier(dbconn["user"])+` WITH SUPERUSER ENCRYPTED PASSWORD `+pq.QuoteLiteral(dbconn["password"]),
		"-c", `CREATE DATABASE `+pq.QuoteIdentifier(dbconn["dbname"])+` WITH TEMPLATE template0 ENCODING 'utf8'`,
	)
	cmd.Dir = "/"
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error setting up arvados user/database: %w", err)
	}
	return nil
}

// Confirm that http://{initcmd.Domain}:{port} reaches a server that
// we run on {port}.
//
// If port is "80", listening fails, and Nginx appears to be using the
// debian-packaged default configuration that listens on port 80,
// disable that Nginx config and try again.
//
// (Typically, the reason Nginx is installed is so that Arvados can
// run an Nginx child process; the default Nginx service using config
// from /etc/nginx is just an unfortunate side effect of installing
// Nginx by way of the Debian package.)
func (initcmd *initCommand) checkPort(ctx context.Context, port string) error {
	err := initcmd.checkPortOnce(ctx, port)
	if err == nil || port != "80" {
		// success, or poking Nginx in the eye won't help
		return err
	}
	d, err2 := os.Open("/etc/nginx/sites-enabled/.")
	if err2 != nil {
		return err
	}
	fis, err2 := d.Readdir(-1)
	if err2 != nil || len(fis) != 1 {
		return err
	}
	if target, err2 := os.Readlink("/etc/nginx/sites-enabled/default"); err2 != nil || target != "/etc/nginx/sites-available/default" {
		return err
	}
	err2 = os.Remove("/etc/nginx/sites-enabled/default")
	if err2 != nil {
		return err
	}
	exec.CommandContext(ctx, "nginx", "-s", "reload").Run()
	time.Sleep(time.Second)
	return initcmd.checkPortOnce(ctx, port)
}

// Start an http server on 0.0.0.0:{port} and confirm that
// http://{initcmd.Domain}:{port} reaches that server.
func (initcmd *initCommand) checkPortOnce(ctx context.Context, port string) error {
	b := make([]byte, 128)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	token := fmt.Sprintf("%x", b)

	srv := http.Server{
		Addr: net.JoinHostPort("", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, token)
		})}
	var errServe atomic.Value
	go func() {
		errServe.Store(srv.ListenAndServe())
	}()
	defer srv.Close()
	url := "http://" + net.JoinHostPort(initcmd.Domain, port) + "/probe"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if errServe, _ := errServe.Load().(error); errServe != nil {
		// If server already exited, return that error
		// (probably "can't listen"), not the request error.
		return errServe
	}
	if err != nil {
		return err
	}
	buf := make([]byte, len(token))
	n, err := io.ReadFull(resp.Body, buf)
	if string(buf[:n]) != token {
		return fmt.Errorf("listened on port %s but %s connected to something else, returned %q, err %v", port, url, buf[:n], err)
	}
	return nil
}
