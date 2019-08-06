// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Fprintf(os.Stderr, `
Keepproxy forwards GET and PUT requests to keepstore servers. See
http://doc.arvados.org/install/install-keepproxy.html

Usage: keepproxy [-config path/to/keepproxy.yml]

DEPRECATION WARNING: The -config parameter is deprecated. Use the
cluster config instead.

`)
}
