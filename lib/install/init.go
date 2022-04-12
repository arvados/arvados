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
	Insecure           bool
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
	flags.StringVar(&initcmd.Login, "login", "", "login `backend`: test, pam, or ''")
	flags.BoolVar(&initcmd.Insecure, "insecure", false, "accept invalid TLS certificates and configure TrustAllContent (do not use in production!)")
	if ok, code := cmd.ParseFlags(flags, prog, args, "", stderr); !ok {
		return code
	} else if *versionFlag {
		return cmd.Version.RunCommand(prog, args, stdin, stdout, stderr)
	} else if !regexp.MustCompile(`^[a-z][a-z0-9]{4}`).MatchString(initcmd.ClusterID) {
		err = fmt.Errorf("cluster ID %q is invalid; must be an ASCII letter followed by 4 alphanumerics (try -help)", initcmd.ClusterID)
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

	err = os.Mkdir("/etc/arvados", 0750)
	if err != nil && !os.IsExist(err) {
		err = fmt.Errorf("mkdir /etc/arvados: %w", err)
		return 1
	}
	err = os.Chown("/etc/arvados", 0, wwwgid)
	f, err := os.OpenFile("/etc/arvados/config.yml", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		err = fmt.Errorf("open /etc/arvados/config.yml: %w", err)
		return 1
	}
	tmpl, err := template.New("config").Parse(`Clusters:
  {{.ClusterID}}:
    Services:
      Controller:
        InternalURLs:
          "http://0.0.0.0:8003/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4430/" ) }}
      RailsAPI:
        InternalURLs:
          "http://0.0.0.0:8004/": {}
      Websocket:
        InternalURLs:
          "http://0.0.0.0:8005/": {}
        ExternalURL: {{printf "%q" ( print "wss://" .Domain ":4435/websocket" ) }}
      Keepbalance:
        InternalURLs:
          "http://0.0.0.0:9005/": {}
      GitHTTP:
        InternalURLs:
          "http://0.0.0.0:9001/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4436/" ) }}
      DispatchCloud:
        InternalURLs:
          "http://0.0.0.0:9006/": {}
      Keepproxy:
        InternalURLs:
          "http://0.0.0.0:25108/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4437/" ) }}
      WebDAV:
        InternalURLs:
          "http://0.0.0.0:9002/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4438/" ) }}
      WebDAVDownload:
        InternalURLs:
          "http://0.0.0.0:8004/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4439/" ) }}
      Keepstore:
        InternalURLs:
          "http://0.0.0.0:25107/": {}
      Composer:
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4440/composer" ) }}
      Workbench1:
        InternalURLs:
          "http://0.0.0.0:8001/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4441/" ) }}
      Workbench2:
        InternalURLs:
          "http://0.0.0.0:8002/": {}
        ExternalURL: {{printf "%q" ( print "https://" .Domain ":4442/" ) }}
      Health:
        InternalURLs:
          "http://0.0.0.0:9007/": {}
    Collections:
      BlobSigningKey: {{printf "%q" ( .RandomHex 50 )}}
      {{if .Insecure}}
      TrustAllContent: true
      {{end}}
    Containers:
      DispatchPrivateKey: {{printf "%q" .GenerateSSHPrivateKey}}
    ManagementToken: {{printf "%q" ( .RandomHex 50 )}}
    PostgreSQL:
      Connection:
        dbname: arvados_production
        host: localhost
        user: arvados
        password: {{printf "%q" .PostgreSQLPassword}}
    SystemRootToken: {{printf "%q" ( .RandomHex 50 )}}
    {{if .Insecure}}
    TLS:
      Insecure: true
    {{end}}
    Volumes:
      {{.ClusterID}}-nyw5e-000000000000000:
        Driver: Directory
        DriverParameters:
          Root: /var/lib/arvados/keep
        Replication: 2
    Workbench:
      SecretKeyBase: {{printf "%q" ( .RandomHex 50 )}}
    Login:
      {{if eq .Login "pam"}}
      PAM:
        Enable: true
      {{else if eq .Login "test"}}
      Test:
        Enable: true
        Users:
          admin:
            Email: admin@example.com
            Password: admin
      {{else}}
      {}
      {{end}}
    Users:
      {{if eq .Login "test"}}
      AutoAdminUserWithEmail: admin@example.com
      {{else}}
      {}
      {{end}}
`)
	if err != nil {
		return 1
	}
	err = tmpl.Execute(f, initcmd)
	if err != nil {
		err = fmt.Errorf("/etc/arvados/config.yml: tmpl.Execute: %w", err)
		return 1
	}
	err = f.Close()
	if err != nil {
		err = fmt.Errorf("/etc/arvados/config.yml: close: %w", err)
		return 1
	}
	fmt.Fprintln(stderr, "created /etc/arvados/config.yml")

	ldr := config.NewLoader(nil, logger)
	ldr.SkipLegacy = true
	cfg, err := ldr.Load()
	if err != nil {
		err = fmt.Errorf("/etc/arvados/config.yml: %w", err)
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
		err = fmt.Errorf("rake db:setup: %w", err)
		return 1
	}
	fmt.Fprintln(stderr, "initialized database")

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
		cmd.Stdout = stderr
		cmd.Stderr = stderr
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error setting up arvados user/database: %w", err)
		}
	}
	return nil
}
