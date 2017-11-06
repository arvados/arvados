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
	c.AnonymousTokens = []string{"xxxxxxxxxxxxxxxxxxxxxxx"}
	c.Client.APIHost = "zzzzz.arvadosapi.com:443"
	exampleConfigFile, err := json.MarshalIndent(c, "    ", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, `

Keep-web provides read-only HTTP access to files stored in Keep; see
https://godoc.org/github.com/curoverse/arvados/services/keep-web and
http://doc.arvados.org/install/install-keep-web.html

Usage: keep-web -config path/to/keep-web.yml

Options:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Example config file:
    %s

Client.APIHost:

    Address (or address:port) of the Arvados API endpoint.

Client.AuthToken:

    Unused. Normally empty, or omitted entirely.

Client.Insecure:

    True if your Arvados API endpoint uses an unverifiable SSL/TLS
    certificate.

Listen:

    Local port to listen on. Can be "address", "address:port", or
    ":port", where "address" is a host IP address or name and "port"
    is a port number or name.

AnonymousTokens:

    Array of tokens to try when a client does not provide a token.

AttachmentOnlyHost:

    Accept credentials, and add "Content-Disposition: attachment"
    response headers, for requests at this hostname:port.

    This prohibits inline display, which makes it possible to serve
    untrusted and non-public content from a single origin, i.e.,
    without wildcard DNS or SSL.

TrustAllContent:

    Serve non-public content from a single origin. Dangerous: read
    docs before using!

Cache.TTL:

    Maximum time to cache manifests and permission checks.

Cache.UUIDTTL:

    Maximum time to cache collection state.

Cache.MaxCollectionEntries:

    Maximum number of collection cache entries.

Cache.MaxCollectionBytes:

    Approximate memory limit for collection cache.

Cache.MaxPermissionEntries:

    Maximum number of permission cache entries.

Cache.MaxUUIDEntries:

    Maximum number of UUID cache entries.

`, exampleConfigFile)
}
