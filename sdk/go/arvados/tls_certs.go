// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "os"

// Load root CAs from /etc/arvados/ca-certificates.crt if it exists
// and SSL_CERT_FILE does not already specify a different file.
func init() {
	envvar := "SSL_CERT_FILE"
	certfile := "/etc/arvados/ca-certificates.crt"
	if os.Getenv(envvar) != "" {
		// Caller has already specified SSL_CERT_FILE.
		return
	}
	if _, err := os.ReadFile(certfile); err != nil {
		// Custom cert file is not present/readable.
		return
	}
	os.Setenv(envvar, certfile)
}
