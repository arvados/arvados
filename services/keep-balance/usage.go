package main

import (
	"flag"
	"fmt"
	"os"
)

var exampleConfigFile = []byte(`
    {
	"Client": {
	    "APIHost": "zzzzz.arvadosapi.com:443",
	    "AuthToken": "xyzzy",
	    "Insecure": false
	},
	"KeepServiceTypes": [
	    "disk"
	],
	"RunPeriod": "600s",
	"CollectionBatchSize": 100000
    }`)

func usage() {
	fmt.Fprintf(os.Stderr, `

keep-balance rebalances a set of keepstore servers. It creates new
copies of underreplicated blocks, deletes excess copies of
overreplicated and unreferenced blocks, and moves blocks to better
positions (according to the rendezvous hash algorithm) so clients find
them faster.

Usage: keep-balance -config path/to/config.json [options]

Options:
`)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Example config file:
%s

    Client.AuthToken must be recognized by Arvados as an admin token,
    and must be recognized by all Keep services as a "data manager
    key".

    Client.Insecure should be true if your Arvados API endpoint uses
    an unverifiable SSL/TLS certificate.

Periodic scanning:

    By default, keep-balance operates periodically, i.e.: do a
    scan/balance operation, sleep, repeat.

    RunPeriod determines the interval between start times of
    successive scan/balance operations. If a scan/balance operation
    takes longer than RunPeriod, the next one will follow it
    immediately.

    If SIGUSR1 is received during an idle period between operations,
    the next operation will start immediately.

One-time scanning:

    Use the -once flag to do a single operation and then exit. The
    exit code will be zero if the operation was successful.

Committing:

    By default, keep-service computes and reports changes but does not
    implement them by sending pull and trash lists to the Keep
    services.

    Use the -commit-pull and -commit-trash flags to implement the
    computed changes.

Limitations:

    keep-balance does not attempt to discover whether committed pull
    and trash requests ever get carried out -- only that they are
    accepted by the Keep services. If some services are full, new
    copies of underreplicated blocks might never get made, only
    repeatedly requested.

`, exampleConfigFile)
}
