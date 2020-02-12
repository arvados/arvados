// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package boot

import (
	"context"
	"io/ioutil"
	"path/filepath"
)

func createCertificates(ctx context.Context, boot *Booter, ready chan<- bool) error {
	// Generate root key
	err := boot.RunProgram(ctx, boot.tempdir, nil, nil, "openssl", "genrsa", "-out", "rootCA.key", "4096")
	if err != nil {
		return err
	}
	// Generate a self-signed root certificate
	err = boot.RunProgram(ctx, boot.tempdir, nil, nil, "openssl", "req", "-x509", "-new", "-nodes", "-key", "rootCA.key", "-sha256", "-days", "3650", "-out", "rootCA.crt", "-subj", "/C=US/ST=MA/O=Example Org/CN=localhost")
	if err != nil {
		return err
	}
	// Generate server key
	err = boot.RunProgram(ctx, boot.tempdir, nil, nil, "openssl", "genrsa", "-out", "server.key", "2048")
	if err != nil {
		return err
	}
	// Build config file for signing request
	defaultconf, err := ioutil.ReadFile("/etc/ssl/openssl.cnf")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(boot.tempdir, "server.cfg"), append(defaultconf, []byte(`
[SAN]
subjectAltName=DNS:localhost,DNS:localhost.localdomain
`)...), 0777)
	if err != nil {
		return err
	}
	// Generate signing request
	err = boot.RunProgram(ctx, boot.tempdir, nil, nil, "openssl", "req", "-new", "-sha256", "-key", "server.key", "-subj", "/C=US/ST=MA/O=Example Org/CN=localhost", "-reqexts", "SAN", "-config", "server.cfg", "-out", "server.csr")
	if err != nil {
		return err
	}
	// Sign certificate
	err = boot.RunProgram(ctx, boot.tempdir, nil, nil, "openssl", "x509", "-req", "-in", "server.csr", "-CA", "rootCA.crt", "-CAkey", "rootCA.key", "-CAcreateserial", "-out", "server.crt", "-days", "3650", "-sha256")
	if err != nil {
		return err
	}

	close(ready)
	<-ctx.Done()
	return nil
}
