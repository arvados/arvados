package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func usage() {
	c := defaultConfig()
	c.Client.APIHost = "zzzzz.arvadosapi.com:443"
	exampleConfigFile, err := json.MarshalIndent(c, "    ", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(os.Stderr, `

arvados-git-httpd provides authenticated access to Arvados-hosted git
repositories.

See http://doc.arvados.org/install/install-arv-git-httpd.html.

Usage: arvados-git-httpd [-config path/to/arvados/git-httpd.yml]

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

    Local port to listen on. Can be "address:port" or ":port", where
    "address" is a host IP address or name and "port" is a port number
    or name.

GitCommand:

    Path to git or gitolite-shell executable. Each authenticated
    request will execute this program with the single argument
    "http-backend".

RepoRoot:

    Path to git repositories.

`, exampleConfigFile)
}
