// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"fmt"
	"os"
)

var exampleConfigFile = []byte(`
    {
	"Client": {
	    "APIHost": "zzzzz.arvadosapi.com",
	    "AuthToken": "xyzzy",
	    "Insecure": false
	    "KeepServiceURIs": [],
	},
	"CrunchRunCommand": ["crunch-run"],
	"PollPeriod": "10s",
	"SbatchArguments": ["--partition=foo", "--exclude=node13"],
	"ReserveExtraRAM": 268435456,
    }`)

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, `
crunch-dispatch-slurm runs queued Arvados containers by submitting
SLURM batch jobs.

Options:
`)
	fs.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
Example config file:
%s
`, exampleConfigFile)
}
