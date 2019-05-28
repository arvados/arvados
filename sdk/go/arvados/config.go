// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

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

type API struct {
	MaxItemsPerResponse     int
	MaxRequestAmplification int
	RequestTimeout          Duration
}

type Cluster struct {
	ClusterID       string `json:"-"`
	ManagementToken string
	SystemRootToken string
	Services        Services
	InstanceTypes   InstanceTypeMap
	Containers      ContainersConfig
	RemoteClusters  map[string]RemoteCluster
	PostgreSQL      PostgreSQL
	API             API
	SystemLogs      SystemLogs
	TLS             TLS
}

type Services struct {
	Controller    Service
	DispatchCloud Service
	Health        Service
	Keepbalance   Service
	Keepproxy     Service
	Keepstore     Service
	Nodemanager   Service
	RailsAPI      Service
	WebDAV        Service
	Websocket     Service
	Workbench1    Service
	Workbench2    Service
}

type Service struct {
	InternalURLs map[URL]ServiceInstance `json:",omitempty"`
	ExternalURL  URL
}

// URL is a url.URL that is also usable as a JSON key/value.
type URL url.URL

// UnmarshalText implements encoding.TextUnmarshaler so URL can be
// used as a JSON key/value.
func (su *URL) UnmarshalText(text []byte) error {
	u, err := url.Parse(string(text))
	if err == nil {
		*su = URL(*u)
	}
	return err
}

func (su URL) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%s", (*url.URL)(&su).String())), nil
}

type ServiceInstance struct{}

type SystemLogs struct {
	LogLevel                string
	Format                  string
	MaxRequestLogParamsSize int
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
	Name            string
	ProviderType    string
	VCPUs           int
	RAM             ByteSize
	Scratch         ByteSize
	IncludedScratch ByteSize
	AddedScratch    ByteSize
	Price           float64
	Preemptible     bool
}

type ContainersConfig struct {
	CloudVMs           CloudVMsConfig
	DispatchPrivateKey string
	StaleLockTimeout   Duration
}

type CloudVMsConfig struct {
	Enable bool

	BootProbeCommand     string
	ImageID              string
	MaxCloudOpsPerSecond int
	MaxProbesPerSecond   int
	PollInterval         Duration
	ProbeInterval        Duration
	SSHPort              string
	SyncInterval         Duration
	TimeoutBooting       Duration
	TimeoutIdle          Duration
	TimeoutProbe         Duration
	TimeoutShutdown      Duration
	TimeoutSignal        Duration
	TimeoutTERM          Duration
	ResourceTags         map[string]string

	Driver           string
	DriverParameters json.RawMessage
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
			if t.ProviderType == "" {
				t.ProviderType = t.Name
			}
			if t.Scratch == 0 {
				t.Scratch = t.IncludedScratch + t.AddedScratch
			} else if t.AddedScratch == 0 {
				t.AddedScratch = t.Scratch - t.IncludedScratch
			} else if t.IncludedScratch == 0 {
				t.IncludedScratch = t.Scratch - t.AddedScratch
			}

			if t.Scratch != (t.IncludedScratch + t.AddedScratch) {
				return fmt.Errorf("%v: Scratch != (IncludedScratch + AddedScratch)", t.Name)
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
	// Fill in Name field (and ProviderType field, if not
	// specified) using hash key.
	*it = InstanceTypeMap(hash)
	for name, t := range *it {
		t.Name = name
		if t.ProviderType == "" {
			t.ProviderType = name
		}
		(*it)[name] = t
	}
	return nil
}

type ServiceName string

const (
	ServiceNameRailsAPI      ServiceName = "arvados-api-server"
	ServiceNameController    ServiceName = "arvados-controller"
	ServiceNameDispatchCloud ServiceName = "arvados-dispatch-cloud"
	ServiceNameHealth        ServiceName = "arvados-health"
	ServiceNameNodemanager   ServiceName = "arvados-node-manager"
	ServiceNameWorkbench1    ServiceName = "arvados-workbench1"
	ServiceNameWorkbench2    ServiceName = "arvados-workbench2"
	ServiceNameWebsocket     ServiceName = "arvados-ws"
	ServiceNameKeepbalance   ServiceName = "keep-balance"
	ServiceNameKeepweb       ServiceName = "keep-web"
	ServiceNameKeepproxy     ServiceName = "keepproxy"
	ServiceNameKeepstore     ServiceName = "keepstore"
)

// Map returns all services as a map, suitable for iterating over all
// services or looking up a service by name.
func (svcs Services) Map() map[ServiceName]Service {
	return map[ServiceName]Service{
		ServiceNameRailsAPI:      svcs.RailsAPI,
		ServiceNameController:    svcs.Controller,
		ServiceNameDispatchCloud: svcs.DispatchCloud,
		ServiceNameHealth:        svcs.Health,
		ServiceNameNodemanager:   svcs.Nodemanager,
		ServiceNameWorkbench1:    svcs.Workbench1,
		ServiceNameWorkbench2:    svcs.Workbench2,
		ServiceNameWebsocket:     svcs.Websocket,
		ServiceNameKeepbalance:   svcs.Keepbalance,
		ServiceNameKeepweb:       svcs.WebDAV,
		ServiceNameKeepproxy:     svcs.Keepproxy,
		ServiceNameKeepstore:     svcs.Keepstore,
	}
}

type TLS struct {
	Certificate string
	Key         string
	Insecure    bool
}
