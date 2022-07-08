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
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
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

	err = os.Mkdir("/var/lib/arvados/keep", 0600)
	if err != nil && !os.IsExist(err) {
		err = fmt.Errorf("mkdir /var/lib/arvados/keep: %w", err)
		return 1
	}
	fmt.Fprintln(stderr, "created /var/lib/arvados/keep")

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
	f, err := os.OpenFile(conffile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		err = fmt.Errorf("open %s: %w", conffile, err)
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
		err = fmt.Errorf("%s: tmpl.Execute: %w", conffile, err)
		return 1
	}
	err = f.Close()
	if err != nil {
		err = fmt.Errorf("%s: close: %w", conffile, err)
		return 1
	}
	fmt.Fprintln(stderr, "created", conffile)

	ldr := config.NewLoader(nil, logger)
	ldr.SkipLegacy = true
	cfg, err := ldr.Load()
	if err != nil {
		err = fmt.Errorf("%s: %w", conffile, err)
		return 1
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return 1
	}

	err = initcmd.createDB(ctx, cluster.PostgreSQL.Connection, stderr)
	if err != nil {
		return 1
	}

	cmd := exec.CommandContext(ctx, "sudo", "-u", "www-data", "-E", "HOME=/var/www", "PATH=/var/lib/arvados/bin:"+os.Getenv("PATH"), "/var/lib/arvados/bin/bundle", "exec", "rake", "db:setup")
	cmd.Dir = "/var/lib/arvados/railsapi"
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("rake db:setup failed: %w", err)
		return 1
	}
	fmt.Fprintln(stderr, "initialized database")

	if initcmd.Start {
		fmt.Fprintln(stderr, "starting systemd service")
		cmd := exec.CommandContext(ctx, "systemctl", "start", "--no-block", "arvados")
		cmd.Dir = "/"
		cmd.Stdout = stderr
		cmd.Stderr = stderr
		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("%v: %w", cmd.Args, err)
			return 1
		}
	}

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
	for _, sql := range []string{
		`CREATE USER ` + pq.QuoteIdentifier(dbconn["user"]) + ` WITH SUPERUSER ENCRYPTED PASSWORD ` + pq.QuoteLiteral(dbconn["password"]),
		`CREATE DATABASE ` + pq.QuoteIdentifier(dbconn["dbname"]) + ` WITH TEMPLATE template0 ENCODING 'utf8'`,
		`CREATE EXTENSION IF NOT EXISTS pg_trgm`,
	} {
		cmd := exec.CommandContext(ctx, "sudo", "-u", "postgres", "psql", "-c", sql)
		cmd.Dir = "/"
		cmd.Stdout = stderr
		cmd.Stderr = stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error setting up arvados user/database: %w", err)
		}
	}
	return nil
}
