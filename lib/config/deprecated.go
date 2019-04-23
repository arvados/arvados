// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"fmt"
	"os"
	"strings"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
)

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

func applyDeprecatedConfig(cfg *arvados.Config, configdata []byte, log logger) error {
	var dc deprecatedConfig
	err := yaml.Unmarshal(configdata, &dc)
	if err != nil {
		return err
	}
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
