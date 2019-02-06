// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"errors"
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

type RequestLimits struct {
	MaxItemsPerResponse            int
	MultiClusterRequestConcurrency int
}

type Cluster struct {
	ClusterID          string `json:"-"`
	ManagementToken    string
	NodeProfiles       map[string]NodeProfile
	InstanceTypes      InstanceTypeMap
	CloudVMs           CloudVMs
	Dispatch           Dispatch
	HTTPRequestTimeout Duration
	RemoteClusters     map[string]RemoteCluster
	PostgreSQL         PostgreSQL
	RequestLimits      RequestLimits
}

type PostgreSQL struct {
	Connection     PostgreSQLConnection
	ConnectionPool int
}

type PostgreSQLConnection map[string]string

type RemoteCluster struct {
	// API endpoint host or host:port; default is {id}.arvadosapi.com
	Host string
	// Perform a proxy request when a local client requests an
	// object belonging to this remote.
	Proxy bool
	// Scheme, default "https". Can be set to "http" for testing.
	Scheme string
	// Disable TLS verify. Can be set to true for testing.
	Insecure bool
}

type InstanceType struct {
	Name         string
	ProviderType string
	VCPUs        int
	RAM          ByteSize
	Scratch      ByteSize
	Price        float64
	Preemptible  bool
}

type Dispatch struct {
	// PEM encoded SSH key (RSA, DSA, or ECDSA) able to log in to
	// cloud VMs.
	PrivateKey []byte

	// Max time for workers to come up before abandoning stale
	// locks from previous run
	StaleLockTimeout Duration

	// Interval between queue polls
	PollInterval Duration

	// Interval between probes to each worker
	ProbeInterval Duration

	// Maximum total worker probes per second
	MaxProbesPerSecond int
}

type CloudVMs struct {
	// Shell command that exits zero IFF the VM is fully booted
	// and ready to run containers, e.g., "mount | grep
	// /encrypted-tmp"
	BootProbeCommand string

	// Listening port (name or number) of SSH servers on worker
	// VMs
	SSHPort string

	SyncInterval Duration

	// Maximum idle time before automatic shutdown
	TimeoutIdle Duration

	// Maximum booting time before automatic shutdown
	TimeoutBooting Duration

	// Maximum time with no successful probes before automatic shutdown
	TimeoutProbe Duration

	// Time after shutdown to retry shutdown
	TimeoutShutdown Duration

	ImageID string

	Driver           string
	DriverParameters map[string]interface{}
}

type InstanceTypeMap map[string]InstanceType

var errDuplicateInstanceTypeName = errors.New("duplicate instance type name")

// UnmarshalJSON handles old config files that provide an array of
// instance types instead of a hash.
func (it *InstanceTypeMap) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		var arr []InstanceType
		err := json.Unmarshal(data, &arr)
		if err != nil {
			return err
		}
		if len(arr) == 0 {
			*it = nil
			return nil
		}
		*it = make(map[string]InstanceType, len(arr))
		for _, t := range arr {
			if _, ok := (*it)[t.Name]; ok {
				return errDuplicateInstanceTypeName
			}
			(*it)[t.Name] = t
		}
		return nil
	}
	var hash map[string]InstanceType
	err := json.Unmarshal(data, &hash)
	if err != nil {
		return err
	}
	// Fill in Name field using hash key.
	*it = InstanceTypeMap(hash)
	for name, t := range *it {
		t.Name = name
		(*it)[name] = t
	}
	return nil
}

// GetNodeProfile returns a NodeProfile for the given hostname. An
// error is returned if the appropriate configuration can't be
// determined (e.g., this does not appear to be a system node). If
// node is empty, use the OS-reported hostname.
func (cc *Cluster) GetNodeProfile(node string) (*NodeProfile, error) {
	if node == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, err
		}
		node = hostname
	}
	if cfg, ok := cc.NodeProfiles[node]; ok {
		return &cfg, nil
	}
	// If node is not listed, but "*" gives a default system node
	// config, use the default config.
	if cfg, ok := cc.NodeProfiles["*"]; ok {
		return &cfg, nil
	}
	return nil, fmt.Errorf("config does not provision host %q as a system node", node)
}

type NodeProfile struct {
	Controller    SystemServiceInstance `json:"arvados-controller"`
	Health        SystemServiceInstance `json:"arvados-health"`
	Keepbalance   SystemServiceInstance `json:"keep-balance"`
	Keepproxy     SystemServiceInstance `json:"keepproxy"`
	Keepstore     SystemServiceInstance `json:"keepstore"`
	Keepweb       SystemServiceInstance `json:"keep-web"`
	Nodemanager   SystemServiceInstance `json:"arvados-node-manager"`
	DispatchCloud SystemServiceInstance `json:"arvados-dispatch-cloud"`
	RailsAPI      SystemServiceInstance `json:"arvados-api-server"`
	Websocket     SystemServiceInstance `json:"arvados-ws"`
	Workbench     SystemServiceInstance `json:"arvados-workbench"`
}

type ServiceName string

const (
	ServiceNameRailsAPI      ServiceName = "arvados-api-server"
	ServiceNameController    ServiceName = "arvados-controller"
	ServiceNameDispatchCloud ServiceName = "arvados-dispatch-cloud"
	ServiceNameNodemanager   ServiceName = "arvados-node-manager"
	ServiceNameWorkbench     ServiceName = "arvados-workbench"
	ServiceNameWebsocket     ServiceName = "arvados-ws"
	ServiceNameKeepbalance   ServiceName = "keep-balance"
	ServiceNameKeepweb       ServiceName = "keep-web"
	ServiceNameKeepproxy     ServiceName = "keepproxy"
	ServiceNameKeepstore     ServiceName = "keepstore"
)

// ServicePorts returns the configured listening address (or "" if
// disabled) for each service on the node.
func (np *NodeProfile) ServicePorts() map[ServiceName]string {
	return map[ServiceName]string{
		ServiceNameRailsAPI:      np.RailsAPI.Listen,
		ServiceNameController:    np.Controller.Listen,
		ServiceNameDispatchCloud: np.DispatchCloud.Listen,
		ServiceNameNodemanager:   np.Nodemanager.Listen,
		ServiceNameWorkbench:     np.Workbench.Listen,
		ServiceNameWebsocket:     np.Websocket.Listen,
		ServiceNameKeepbalance:   np.Keepbalance.Listen,
		ServiceNameKeepweb:       np.Keepweb.Listen,
		ServiceNameKeepproxy:     np.Keepproxy.Listen,
		ServiceNameKeepstore:     np.Keepstore.Listen,
	}
}

func (h RequestLimits) GetMultiClusterRequestConcurrency() int {
	if h.MultiClusterRequestConcurrency == 0 {
		return 4
	}
	return h.MultiClusterRequestConcurrency
}

func (h RequestLimits) GetMaxItemsPerResponse() int {
	if h.MaxItemsPerResponse == 0 {
		return 1000
	}
	return h.MaxItemsPerResponse
}

type SystemServiceInstance struct {
	Listen   string
	TLS      bool
	Insecure bool
}
