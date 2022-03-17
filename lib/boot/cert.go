// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const stagingDirectoryURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

var errInvalidHost = errors.New("unrecognized target host in incoming TLS request")

type createCertificates struct{}

func (createCertificates) String() string {
	return "certificates"
}

func (createCertificates) Run(ctx context.Context, fail func(error), super *Supervisor) error {
	if super.cluster.TLS.Automatic {
		return bootAutoCert(ctx, fail, super)
	} else if super.cluster.TLS.Key == "" && super.cluster.TLS.Certificate == "" {
		return createSelfSignedCert(ctx, fail, super)
	} else {
		return nil
	}
}

// bootAutoCert uses Let's Encrypt to get certificates for all the
// domains appearing in ExternalURLs, writes them to files where Nginx
// can load them, and updates super.cluster.TLS fields (Key and
// Certificiate) to point to those files.
//
// It also runs a background task to keep the files up to date.
//
// After bootAutoCert returns, other service components will get the
// certificates they need by reading these files or by using a
// read-only autocert cache.
//
// Currently this only works when port 80 of every ExternalURL domain
// is routed to this host, i.e., on a single-node cluster. Wildcard
// domains [for WebDAV] are not supported.
func bootAutoCert(ctx context.Context, fail func(error), super *Supervisor) error {
	hosts := map[string]bool{}
	for _, svc := range super.cluster.Services.Map() {
		u := url.URL(svc.ExternalURL)
		if u.Scheme == "https" || u.Scheme == "wss" {
			hosts[strings.ToLower(u.Hostname())] = true
		}
	}
	mgr := &autocert.Manager{
		Cache:  autocert.DirCache(super.tempdir + "/autocert"),
		Prompt: autocert.AcceptTOS,
		HostPolicy: func(ctx context.Context, host string) error {
			if hosts[strings.ToLower(host)] {
				return nil
			} else {
				return errInvalidHost
			}
		},
	}
	if super.cluster.TLS.Staging {
		mgr.Client = &acme.Client{DirectoryURL: stagingDirectoryURL}
	}
	go func() {
		err := http.ListenAndServe(":80", mgr.HTTPHandler(nil))
		fail(fmt.Errorf("autocert http-01 challenge handler stopped: %w", err))
	}()
	u := url.URL(super.cluster.Services.Controller.ExternalURL)
	extHost := u.Hostname()
	update := func() error {
		for h := range hosts {
			cert, err := mgr.GetCertificate(&tls.ClientHelloInfo{ServerName: h})
			if err != nil {
				return err
			}
			if h == extHost {
				err = writeCert(super.tempdir, "server.key", "server.crt", cert)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	err := update()
	if err != nil {
		return err
	}
	go func() {
		for range time.NewTicker(time.Hour).C {
			err := update()
			if err != nil {
				super.logger.WithError(err).Error("error getting certificate from autocert")
			}
		}
	}()
	super.cluster.TLS.Key = "file://" + super.tempdir + "/server.key"
	super.cluster.TLS.Certificate = "file://" + super.tempdir + "/server.crt"
	return nil
}

// Save cert chain and key in a format Nginx can read.
func writeCert(outdir, keyfile, certfile string, cert *tls.Certificate) error {
	keytmp, err := os.CreateTemp(outdir, keyfile+".tmp.*")
	if err != nil {
		return err
	}
	defer keytmp.Close()
	defer os.Remove(keytmp.Name())

	certtmp, err := os.CreateTemp(outdir, certfile+".tmp.*")
	if err != nil {
		return err
	}
	defer certtmp.Close()
	defer os.Remove(certtmp.Name())

	switch privkey := cert.PrivateKey.(type) {
	case *rsa.PrivateKey:
		err = pem.Encode(keytmp, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privkey),
		})
		if err != nil {
			return err
		}
	default:
		buf, err := x509.MarshalPKCS8PrivateKey(privkey)
		if err != nil {
			return err
		}
		err = pem.Encode(keytmp, &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: buf,
		})
		if err != nil {
			return err
		}
	}
	err = keytmp.Close()
	if err != nil {
		return err
	}

	for _, cert := range cert.Certificate {
		err = pem.Encode(certtmp, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert,
		})
		if err != nil {
			return err
		}
	}
	err = certtmp.Close()
	if err != nil {
		return err
	}

	err = os.Rename(keytmp.Name(), filepath.Join(outdir, keyfile))
	if err != nil {
		return err
	}
	err = os.Rename(certtmp.Name(), filepath.Join(outdir, certfile))
	if err != nil {
		return err
	}
	return nil
}

// Create a root CA key and use it to make a new server
// certificate+key pair.
//
// In future we'll make one root CA key per host instead of one per
// cluster, so it only needs to be imported to a browser once for
// ongoing dev/test usage.
func createSelfSignedCert(ctx context.Context, fail func(error), super *Supervisor) error {
	san := "DNS:localhost,DNS:localhost.localdomain"
	if net.ParseIP(super.ListenHost) != nil {
		san += fmt.Sprintf(",IP:%s", super.ListenHost)
	} else {
		san += fmt.Sprintf(",DNS:%s", super.ListenHost)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("hostname: %w", err)
	}
	if hostname != super.ListenHost {
		san += ",DNS:" + hostname
	}

	// Generate root key
	err = super.RunProgram(ctx, super.tempdir, runOptions{}, "openssl", "genrsa", "-out", "rootCA.key", "4096")
	if err != nil {
		return err
	}
	// Generate a self-signed root certificate
	err = super.RunProgram(ctx, super.tempdir, runOptions{}, "openssl", "req", "-x509", "-new", "-nodes", "-key", "rootCA.key", "-sha256", "-days", "3650", "-out", "rootCA.crt", "-subj", "/C=US/ST=MA/O=Example Org/CN=localhost")
	if err != nil {
		return err
	}
	// Generate server key
	err = super.RunProgram(ctx, super.tempdir, runOptions{}, "openssl", "genrsa", "-out", "server.key", "2048")
	if err != nil {
		return err
	}
	// Build config file for signing request
	defaultconf, err := ioutil.ReadFile("/etc/ssl/openssl.cnf")
	if err != nil {
		return err
	}
	conf := append(defaultconf, []byte(fmt.Sprintf("\n[SAN]\nsubjectAltName=%s\n", san))...)
	err = ioutil.WriteFile(filepath.Join(super.tempdir, "server.cfg"), conf, 0644)
	if err != nil {
		return err
	}
	// Generate signing request
	err = super.RunProgram(ctx, super.tempdir, runOptions{}, "openssl", "req", "-new", "-sha256", "-key", "server.key", "-subj", "/C=US/ST=MA/O=Example Org/CN=localhost", "-reqexts", "SAN", "-config", "server.cfg", "-out", "server.csr")
	if err != nil {
		return err
	}
	// Sign certificate
	err = super.RunProgram(ctx, super.tempdir, runOptions{}, "openssl", "x509", "-req", "-in", "server.csr", "-CA", "rootCA.crt", "-CAkey", "rootCA.key", "-CAcreateserial", "-out", "server.crt", "-extfile", "server.cfg", "-extensions", "SAN", "-days", "3650", "-sha256")
	if err != nil {
		return err
	}
	super.cluster.TLS.Key = "file://" + super.tempdir + "/server.key"
	super.cluster.TLS.Certificate = "file://" + super.tempdir + "/server.crt"
	return nil
}
