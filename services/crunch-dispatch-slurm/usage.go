// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

import (
	"flag"
	"fmt"
)

func usage(fs *flag.FlagSet) {
	fmt.Fprintf(fs.Output(), `
crunch-dispatch-slurm runs queued Arvados containers by submitting
SLURM batch jobs.

Options:
`)
	fs.PrintDefaults()
	fmt.Fprintf(fs.Output(), `

For configuration instructions see https://doc.arvados.org/install/crunch2-slurm/install-dispatch.html
`)
}
