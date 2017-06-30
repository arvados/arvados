// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"os"

	"github.com/ghodss/yaml"
)

// DumpAndExit writes the given config to stdout as YAML. If an error
// occurs, that error is returned. Otherwise, the program exits 0.
//
// Example:
//
//	log.Fatal(DumpAndExit(cfg))
func DumpAndExit(cfg interface{}) error {
	y, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(y)
	if err != nil {
		return err
	}
	os.Exit(0)
	return errors.New("exit failed!?")
}
