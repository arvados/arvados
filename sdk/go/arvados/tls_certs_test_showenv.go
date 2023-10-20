// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

//go:build ignore

// This is a test program invoked by tls_certs_test.go

package main

import (
	"fmt"
	"os"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

var _ = arvados.Client{}

func main() {
	fmt.Println(os.Getenv("SSL_CERT_FILE"))
}
