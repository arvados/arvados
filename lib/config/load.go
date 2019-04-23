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

type deprRequestLimits struct {
	MaxItemsPerResponse            *int
	MultiClusterRequestConcurrency *int
}

type deprCluster struct {
	RequestLimits deprRequestLimits
	NodeProfiles  map[string]arvados.NodeProfile
}

type deprecatedConfig struct {
	Clusters map[string]deprCluster
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
		mergo.Merge(&merged, src, mergo.WithOverride)
	}
	var src map[string]interface{}
	err = yaml.Unmarshal(buf, &src)
	if err != nil {
		return nil, fmt.Errorf("loading config data: %s", err)
	}
	mergo.Merge(&merged, src, mergo.WithOverride)

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
		if dst, n := &cluster.API.MaxItemsPerResponse, dcluster.RequestLimits.MaxItemsPerResponse; n != nil && *n != *dst {
			log.Warnf("overriding Clusters.%s.API.MaxItemsPerResponse with deprecated config RequestLimits.MultiClusterRequestConcurrency = %d", id, *n)
			*dst = *n
		}
		if dst, n := &cluster.API.MaxRequestAmplification, dcluster.RequestLimits.MultiClusterRequestConcurrency; n != nil && *n != *dst {
			log.Warnf("overriding Clusters.%s.API.MaxRequestAmplification with deprecated config RequestLimits.MultiClusterRequestConcurrency = %d", id, *n)
			*dst = *n
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
