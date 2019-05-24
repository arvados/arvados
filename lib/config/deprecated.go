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
	NodeProfiles  map[string]nodeProfile
}

type deprecatedConfig struct {
	Clusters map[string]deprCluster
}

type nodeProfile struct {
	Controller    systemServiceInstance `json:"arvados-controller"`
	Health        systemServiceInstance `json:"arvados-health"`
	Keepbalance   systemServiceInstance `json:"keep-balance"`
	Keepproxy     systemServiceInstance `json:"keepproxy"`
	Keepstore     systemServiceInstance `json:"keepstore"`
	Keepweb       systemServiceInstance `json:"keep-web"`
	Nodemanager   systemServiceInstance `json:"arvados-node-manager"`
	DispatchCloud systemServiceInstance `json:"arvados-dispatch-cloud"`
	RailsAPI      systemServiceInstance `json:"arvados-api-server"`
	Websocket     systemServiceInstance `json:"arvados-ws"`
	Workbench1    systemServiceInstance `json:"arvados-workbench"`
}

type systemServiceInstance struct {
	Listen   string
	TLS      bool
	Insecure bool
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
				name = "localhost"
			} else if log != nil {
				log.Warnf("overriding Clusters.%s.Services using Clusters.%s.NodeProfiles.%s (guessing %q is a hostname)", id, id, name, name)
			}
			applyDeprecatedNodeProfile(name, np.RailsAPI, &cluster.Services.RailsAPI)
			applyDeprecatedNodeProfile(name, np.Controller, &cluster.Services.Controller)
			applyDeprecatedNodeProfile(name, np.DispatchCloud, &cluster.Services.DispatchCloud)
		}
		if dst, n := &cluster.API.MaxItemsPerResponse, dcluster.RequestLimits.MaxItemsPerResponse; n != nil && *n != *dst {
			*dst = *n
		}
		if dst, n := &cluster.API.MaxRequestAmplification, dcluster.RequestLimits.MultiClusterRequestConcurrency; n != nil && *n != *dst {
			*dst = *n
		}
		cfg.Clusters[id] = cluster
	}
	return nil
}

func applyDeprecatedNodeProfile(hostname string, ssi systemServiceInstance, svc *arvados.Service) {
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
