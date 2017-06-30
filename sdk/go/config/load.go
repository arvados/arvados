// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"io/ioutil"

	"github.com/ghodss/yaml"
)

// LoadFile loads configuration from the file given by configPath and
// decodes it into cfg.
//
// YAML and JSON formats are supported.
func LoadFile(cfg interface{}, configPath string) error {
	buf, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(buf, cfg)
	if err != nil {
		return fmt.Errorf("Error decoding config %q: %v", configPath, err)
	}
	return nil
}

// Dump returns a YAML representation of cfg.
func Dump(cfg interface{}) ([]byte, error) {
	return yaml.Marshal(cfg)
}
