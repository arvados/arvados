package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
)

func usage() {
	c := DefaultConfig()
	knownTypes := []string{}
	for _, vt := range VolumeTypes {
		c.Volumes = append(c.Volumes, vt().Examples()...)
		knownTypes = append(knownTypes, vt().Type())
	}
	exampleConfigFile, err := yaml.Marshal(c)
	if err != nil {
		panic(err)
	}
	sort.Strings(knownTypes)
	knownTypeList := strings.Join(knownTypes, ", ")
	fmt.Fprintf(os.Stderr, `

keepstore provides a content-addressed data store backed by a local filesystem or networked storage.

Usage: keepstore -config path/to/keepstore.yml
       keepstore [OPTIONS] -dump-config

NOTE: All options (other than -config) are deprecated in favor of YAML
      configuration. Use -dump-config to translate existing
      configurations to YAML format.

Options:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Example config file:

%s

Listen:

    Local port to listen on. Can be "address:port" or ":port", where
    "address" is a host IP address or name and "port" is a port number
    or name.

PIDFile:

   Path to write PID file during startup. This file is kept open and
   locked with LOCK_EX until keepstore exits, so "fuser -k pidfile" is
   one way to shut down. Exit immediately if there is an error
   opening, locking, or writing the PID file.

MaxBuffers:

    Maximum RAM to use for data buffers, given in multiples of block
    size (64 MiB). When this limit is reached, HTTP requests requiring
    buffers (like GET and PUT) will wait for buffer space to be
    released.

MaxRequests:

    Maximum concurrent requests. When this limit is reached, new
    requests will receive 503 responses. Note: this limit does not
    include idle connections from clients using HTTP keepalive, so it
    does not strictly limit the number of concurrent connections. If
    omitted or zero, the default is 2 * MaxBuffers.

BlobSigningKeyFile:

    Local file containing the secret blob signing key (used to
    generate and verify blob signatures).  This key should be
    identical to the API server's blob_signing_key configuration
    entry.

RequireSignatures:

    Honor read requests only if a valid signature is provided.  This
    should be true, except for development use and when migrating from
    a very old version.

BlobSignatureTTL:

    Duration for which new permission signatures (returned in PUT
    responses) will be valid.  This should be equal to the API
    server's blob_signature_ttl configuration entry.

SystemAuthTokenFile:

    Local file containing the Arvados API token used by keep-balance
    or data manager.  Delete, trash, and index requests are honored
    only for this token.

EnableDelete:

    Enable trash and delete features. If false, trash lists will be
    accepted but blocks will not be trashed or deleted.

TrashLifetime:

    Time duration after a block is trashed during which it can be
    recovered using an /untrash request.

TrashCheckInterval:

    How often to check for (and delete) trashed blocks whose
    TrashLifetime has expired.

Volumes:

    List of storage volumes. If omitted or empty, the default is to
    use all directories named "keep" that exist in the top level
    directory of a mount point at startup time.

    Volume types: %s

    (See volume configuration examples above.)

`, exampleConfigFile, knownTypeList)
}
