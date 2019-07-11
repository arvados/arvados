// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
)

type logger interface {
	Warnf(string, ...interface{})
}

func loadFileOrStdin(path string, stdin io.Reader, log logger) (*arvados.Config, error) {
	if path == "-" {
		return load(stdin, log, true)
	} else {
		return LoadFile(path, log)
	}
}

func LoadFile(path string, log logger) (*arvados.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Load(f, log)
}

func Load(rdr io.Reader, log logger) (*arvados.Config, error) {
	return load(rdr, log, true)
}

func load(rdr io.Reader, log logger, useDeprecated bool) (*arvados.Config, error) {
	buf, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, err
	}

	// Load the config into a dummy map to get the cluster ID
	// keys, discarding the values; then set up defaults for each
	// cluster ID; then load the real config on top of the
	// defaults.
	var dummy struct {
		Clusters map[string]struct{}
	}
	err = yaml.Unmarshal(buf, &dummy)
	if err != nil {
		return nil, err
	}
	if len(dummy.Clusters) == 0 {
		return nil, errors.New("config does not define any clusters")
	}

	// We can't merge deep structs here; instead, we unmarshal the
	// default & loaded config files into generic maps, merge
	// those, and then json-encode+decode the result into the
	// config struct type.
	var merged map[string]interface{}
	for id := range dummy.Clusters {
		var src map[string]interface{}
		err = yaml.Unmarshal(bytes.Replace(DefaultYAML, []byte(" xxxxx:"), []byte(" "+id+":"), -1), &src)
		if err != nil {
			return nil, fmt.Errorf("loading defaults for %s: %s", id, err)
		}
		err = mergo.Merge(&merged, src, mergo.WithOverride)
		if err != nil {
			return nil, fmt.Errorf("merging defaults for %s: %s", id, err)
		}
	}
	var src map[string]interface{}
	err = yaml.Unmarshal(buf, &src)
	if err != nil {
		return nil, fmt.Errorf("loading config data: %s", err)
	}
	logExtraKeys(log, merged, src, "")
	removeSampleKeys(merged)
	err = mergo.Merge(&merged, src, mergo.WithOverride)
	if err != nil {
		return nil, fmt.Errorf("merging config data: %s", err)
	}

	// map[string]interface{} => json => arvados.Config
	var cfg arvados.Config
	var errEnc error
	pr, pw := io.Pipe()
	go func() {
		errEnc = json.NewEncoder(pw).Encode(merged)
		pw.Close()
	}()
	err = json.NewDecoder(pr).Decode(&cfg)
	if errEnc != nil {
		err = errEnc
	}
	if err != nil {
		return nil, fmt.Errorf("transcoding config data: %s", err)
	}

	if useDeprecated {
		err = applyDeprecatedConfig(&cfg, buf, log)
		if err != nil {
			return nil, err
		}
	}

	// Check for known mistakes
	for id, cc := range cfg.Clusters {
		err = checkKeyConflict(fmt.Sprintf("Clusters.%s.PostgreSQL.Connection", id), cc.PostgreSQL.Connection)
		if err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

func checkKeyConflict(label string, m map[string]string) error {
	saw := map[string]bool{}
	for k := range m {
		k = strings.ToLower(k)
		if saw[k] {
			return fmt.Errorf("%s: multiple entries for %q (fix by using same capitalization as default/example file)", label, k)
		}
		saw[k] = true
	}
	return nil
}

func removeSampleKeys(m map[string]interface{}) {
	delete(m, "SAMPLE")
	for _, v := range m {
		if v, _ := v.(map[string]interface{}); v != nil {
			removeSampleKeys(v)
		}
	}
}

func logExtraKeys(log logger, expected, supplied map[string]interface{}, prefix string) {
	if log == nil {
		return
	}
	allowed := map[string]interface{}{}
	for k, v := range expected {
		allowed[strings.ToLower(k)] = v
	}
	for k, vsupp := range supplied {
		if k == "SAMPLE" {
			// entry will be dropped in removeSampleKeys anyway
			continue
		}
		vexp, ok := allowed[strings.ToLower(k)]
		if expected["SAMPLE"] != nil {
			vexp = expected["SAMPLE"]
		} else if !ok {
			log.Warnf("deprecated or unknown config entry: %s%s", prefix, k)
			continue
		}
		if vsupp, ok := vsupp.(map[string]interface{}); !ok {
			// if vsupp is a map but vexp isn't map, this
			// will be caught elsewhere; see TestBadType.
			continue
		} else if vexp, ok := vexp.(map[string]interface{}); !ok {
			log.Warnf("unexpected object in config entry: %s%s", prefix, k)
		} else {
			logExtraKeys(log, vexp, vsupp, prefix+k+".")
		}
	}
}
