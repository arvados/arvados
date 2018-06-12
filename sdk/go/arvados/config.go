// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"fmt"
	"os"

	"git.curoverse.com/arvados.git/sdk/go/config"
)

const DefaultConfigFile = "/etc/arvados/config.yml"

type Config struct {
	Clusters map[string]Cluster
}

// GetConfig returns the current system config, loading it from
// configFile if needed.
func GetConfig(configFile string) (*Config, error) {
	var cfg Config
	err := config.LoadFile(&cfg, configFile)
	return &cfg, err
}

// GetCluster returns the cluster ID and config for the given
// cluster, or the default/only configured cluster if clusterID is "".
func (sc *Config) GetCluster(clusterID string) (*Cluster, error) {
	if clusterID == "" {
		if len(sc.Clusters) == 0 {
			return nil, fmt.Errorf("no clusters configured")
		} else if len(sc.Clusters) > 1 {
			return nil, fmt.Errorf("multiple clusters configured, cannot choose")
		} else {
			for id, cc := range sc.Clusters {
				cc.ClusterID = id
				return &cc, nil
			}
		}
	}
	if cc, ok := sc.Clusters[clusterID]; !ok {
		return nil, fmt.Errorf("cluster %q is not configured", clusterID)
	} else {
		cc.ClusterID = clusterID
		return &cc, nil
	}
}

type Cluster struct {
	ClusterID       string `json:"-"`
	ManagementToken string
	SystemNodes     map[string]SystemNode
	InstanceTypes   []InstanceType
}

type InstanceType struct {
	Name         string
	ProviderType string
	VCPUs        int
	RAM          int64
	Scratch      int64
	Price        float64
}

// GetThisSystemNode returns a SystemNode for the node we're running
// on right now.
func (cc *Cluster) GetThisSystemNode() (*SystemNode, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	return cc.GetSystemNode(hostname)
}

// GetSystemNode returns a SystemNode for the given hostname. An error
// is returned if the appropriate configuration can't be determined
// (e.g., this does not appear to be a system node).
func (cc *Cluster) GetSystemNode(node string) (*SystemNode, error) {
	if cfg, ok := cc.SystemNodes[node]; ok {
		return &cfg, nil
	}
	// If node is not listed, but "*" gives a default system node
	// config, use the default config.
	if cfg, ok := cc.SystemNodes["*"]; ok {
		return &cfg, nil
	}
	return nil, fmt.Errorf("config does not provision host %q as a system node", node)
}

type SystemNode struct {
	Controller  SystemServiceInstance `json:"arvados-controller"`
	Health      SystemServiceInstance `json:"arvados-health"`
	Keepproxy   SystemServiceInstance `json:"keepproxy"`
	Keepstore   SystemServiceInstance `json:"keepstore"`
	Keepweb     SystemServiceInstance `json:"keep-web"`
	Nodemanager SystemServiceInstance `json:"arvados-node-manager"`
	RailsAPI    SystemServiceInstance `json:"arvados-api-server"`
	Websocket   SystemServiceInstance `json:"arvados-ws"`
	Workbench   SystemServiceInstance `json:"arvados-workbench"`
}

type ServiceName string

const (
	ServiceNameRailsAPI    ServiceName = "arvados-api-server"
	ServiceNameController  ServiceName = "arvados-controller"
	ServiceNameNodemanager ServiceName = "arvados-node-manager"
	ServiceNameWorkbench   ServiceName = "arvados-workbench"
	ServiceNameWebsocket   ServiceName = "arvados-ws"
	ServiceNameKeepweb     ServiceName = "keep-web"
	ServiceNameKeepproxy   ServiceName = "keepproxy"
	ServiceNameKeepstore   ServiceName = "keepstore"
)

// ServicePorts returns the configured listening address (or "" if
// disabled) for each service on the node.
func (sn *SystemNode) ServicePorts() map[ServiceName]string {
	return map[ServiceName]string{
		ServiceNameRailsAPI:    sn.RailsAPI.Listen,
		ServiceNameController:  sn.Controller.Listen,
		ServiceNameNodemanager: sn.Nodemanager.Listen,
		ServiceNameWorkbench:   sn.Workbench.Listen,
		ServiceNameWebsocket:   sn.Websocket.Listen,
		ServiceNameKeepweb:     sn.Keepweb.Listen,
		ServiceNameKeepproxy:   sn.Keepproxy.Listen,
		ServiceNameKeepstore:   sn.Keepstore.Listen,
	}
}

type SystemServiceInstance struct {
	Listen string
	TLS    bool
}
