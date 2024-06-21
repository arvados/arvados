// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"git.arvados.org/arvados.git/lib/cmd"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
)

var Get cmd.Handler = getCmd{}

type getCmd struct{}

func (getCmd) RunCommand(prog string, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var err error
	defer func() {
		if err != nil {
			fmt.Fprintf(stderr, "%s\n", err)
		}
	}()

	flags, opts := LegacyFlagSet()
	flags.SetOutput(stderr)
	err = flags.Parse(args)
	if err != nil {
		return cmd.EXIT_INVALIDARGUMENT
	}
	if len(flags.Args()) != 1 {
		fmt.Fprintf(stderr, "usage of %s:\n", prog)
		flags.PrintDefaults()
		return cmd.EXIT_INVALIDARGUMENT
	}
	if opts.Short {
		opts.Format = "uuid"
	}

	id := flags.Args()[0]
	client := arvados.NewClientFromEnv()
	path, err := client.PathForUUID("get", id)
	if err != nil {
		return 1
	}

	var obj map[string]interface{}
	err = client.RequestAndDecode(&obj, "GET", path, nil, nil)
	if err != nil {
		err = fmt.Errorf("GET %s: %s", path, err)
		return 1
	}
	if opts.Format == "yaml" {
		var buf []byte
		buf, err = yaml.Marshal(obj)
		if err == nil {
			_, err = stdout.Write(buf)
		}
	} else if opts.Format == "uuid" {
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
