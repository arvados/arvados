// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package main

import (
	"flag"
	"fmt"
	"os"
)

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, `
crunch-dispatch-slurm runs queued Arvados containers by submitting
SLURM batch jobs.

Options:
`)
	fs.PrintDefaults()
	fmt.Fprintf(os.Stderr, `

For configuration instructions see https://doc.arvados.org/install/crunch2-slurm/install-dispatch.html
`)
}
