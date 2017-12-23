// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
)

func Get(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	defer func() {
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err)
		}
	}()

	flags := flag.NewFlagSet(prog, flag.ContinueOnError)
	format := flags.String("format", "json", "output format (json, yaml, or uuid)")
	flags.StringVar(format, "f", "json", "output format (json, yaml, or uuid)")
	short := flags.Bool("short", false, "equivalent to --format=uuid")
	flags.BoolVar(short, "s", false, "equivalent to --format=uuid")
	flags.Bool("dry-run", false, "dry run (ignored, for compatibility)")
	flags.Bool("n", false, "dry run (ignored, for compatibility)")
	flags.Bool("verbose", false, "verbose (ignored, for compatibility)")
	flags.Bool("v", false, "verbose (ignored, for compatibility)")
	err = flags.Parse(args)
	if err != nil {
		return 2
	}
	if len(flags.Args()) != 1 {
		flags.Usage()
		return 2
	}
	if *short {
		*format = "uuid"
	}

	id := flags.Args()[0]
	client := arvados.NewClientFromEnv()
	path, err := client.PathForUUID("show", id)
	if err != nil {
		return 1
	}

	var obj map[string]interface{}
	err = client.RequestAndDecode(&obj, "GET", path, nil, nil)
	if err != nil {
		err = fmt.Errorf("GET %s: %s", path, err)
		return 1
	}
	if *format == "yaml" {
		var buf []byte
		buf, err = yaml.Marshal(obj)
		if err == nil {
			_, err = stdout.Write(buf)
		}
	} else if *format == "uuid" {
		fmt.Fprintln(stdout, obj["uuid"])
	} else {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		err = enc.Encode(obj)
	}
	if err != nil {
		err = fmt.Errorf("encoding: %s", err)
		return 1
	}
	return 0
}
