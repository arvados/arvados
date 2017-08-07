// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func usage() {
	c := DefaultConfig()
	c.Client.APIHost = "zzzzz.arvadosapi.com:443"
	exampleConfigFile, err := json.MarshalIndent(c, "    ", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, `

Keepproxy forwards GET and PUT requests to keepstore servers.  See
http://doc.arvados.org/install/install-keepproxy.html

Usage: keepproxy [-config path/to/keepproxy.yml]

Options:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Example config file:
    %s

Client.APIHost:

    Address (or address:port) of the Arvados API endpoint.

Client.AuthToken:

    Anonymous API token.

Client.Insecure:

    True if your Arvados API endpoint uses an unverifiable SSL/TLS
    certificate.

Listen:

    Local port to listen on. Can be "address:port" or ":port", where
    "address" is a host IP address or name and "port" is a port number
    or name.

DisableGet:

    Respond 404 to GET and HEAD requests.

DisablePut:

    Respond 404 to PUT, POST, and OPTIONS requests.

DefaultReplicas:

    Default number of replicas to write if not specified by the
    client. If this is zero or omitted, the site-wide
    defaultCollectionReplication configuration will be used.

Timeout:

    Timeout for requests to keep services, with units (e.g., "120s",
    "2m").

PIDFile:

    Path to PID file. During startup this file will be created if
    needed, and locked using flock() until keepproxy exits. If it is
    already locked, or any error is encountered while writing to it,
    keepproxy will exit immediately. If omitted or empty, no PID file
    will be used.

Debug:

    Enable debug logging.

ManagementToken:

    Authorization token to be included in all health check requests.

`, exampleConfigFile)
}
