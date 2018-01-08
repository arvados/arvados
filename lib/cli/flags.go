// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"flag"

	"git.curoverse.com/arvados.git/lib/cmd"
	"rsc.io/getopt"
)

type LegacyFlagValues struct {
	Format  string
	DryRun  bool
	Short   bool
	Verbose bool
}

func LegacyFlagSet() (cmd.FlagSet, *LegacyFlagValues) {
	values := &LegacyFlagValues{Format: "json"}
	flags := getopt.NewFlagSet("", flag.ContinueOnError)
	flags.BoolVar(&values.DryRun, "dry-run", false, "Don't actually do anything")
	flags.Alias("n", "dry-run")
	flags.StringVar(&values.Format, "format", values.Format, "Output format: json, yaml, or uuid")
	flags.Alias("f", "format")
	flags.BoolVar(&values.Short, "short", false, "Return only UUIDs (equivalent to --format=uuid)")
	flags.Alias("s", "short")
	flags.BoolVar(&values.Verbose, "verbose", false, "Print more debug/progress messages on stderr")
	flags.Alias("v", "verbose")
	return flags, values
}
