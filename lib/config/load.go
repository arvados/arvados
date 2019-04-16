// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
)

type logger interface {
	Warnf(string, ...interface{})
}

type deprecatedConfig struct {
	Clusters map[string]struct {
		NodeProfiles map[string]arvados.NodeProfile
	}
}

func Load(rdr io.Reader, log logger) (*arvados.Config, error) {
	var cfg arvados.Config
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
	for id := range dummy.Clusters {
		err = yaml.Unmarshal(bytes.Replace(DefaultYAML, []byte("xxxxx"), []byte(id), -1), &cfg)
		if err != nil {
			return nil, fmt.Errorf("loading defaults for %s: %s", id, err)
		}
	}
	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return nil, err
	}

	// Check for deprecated config values, and apply them to cfg.
	var dc deprecatedConfig
	err = yaml.Unmarshal(buf, &dc)
	if err != nil {
		return nil, err
	}
	err = applyDeprecatedConfig(&cfg, &dc, log)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func applyDeprecatedConfig(cfg *arvados.Config, dc *deprecatedConfig, log logger) error {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	for id, dcluster := range dc.Clusters {
		cluster, ok := cfg.Clusters[id]
		if !ok {
			return fmt.Errorf("can't load legacy config %q that is not present in current config", id)
		}
		for name, np := range dcluster.NodeProfiles {
			if name == "*" || name == os.Getenv("ARVADOS_NODE_PROFILE") || name == hostname {
				applyDeprecatedNodeProfile(hostname, np.RailsAPI, &cluster.Services.RailsAPI)
				applyDeprecatedNodeProfile(hostname, np.Controller, &cluster.Services.Controller)
				applyDeprecatedNodeProfile(hostname, np.DispatchCloud, &cluster.Services.DispatchCloud)
			}
		}
		cfg.Clusters[id] = cluster
	}
	return nil
}

func applyDeprecatedNodeProfile(hostname string, ssi arvados.SystemServiceInstance, svc *arvados.Service) {
	scheme := "https"
	if !ssi.TLS {
		scheme = "http"
	}
	if svc.InternalURLs == nil {
		svc.InternalURLs = map[arvados.URL]arvados.ServiceInstance{}
	}
	host := ssi.Listen
	if host == "" {
		return
	}
	if strings.HasPrefix(host, ":") {
		host = hostname + host
	}
	svc.InternalURLs[arvados.URL{Scheme: scheme, Host: host}] = arvados.ServiceInstance{}
}
