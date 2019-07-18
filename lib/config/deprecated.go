// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"fmt"
	"io/ioutil"
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

func (ldr *Loader) applyDeprecatedConfig(cfg *arvados.Config) error {
	var dc deprecatedConfig
	err := yaml.Unmarshal(ldr.configdata, &dc)
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
			} else if ldr.Logger != nil {
				ldr.Logger.Warnf("overriding Clusters.%s.Services using Clusters.%s.NodeProfiles.%s (guessing %q is a hostname)", id, id, name, name)
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

const defaultKeepstoreConfigPath = "/etc/arvados/keepstore/keepstore.yml"

type oldKeepstoreConfig struct {
	Debug *bool
}

func (ldr *Loader) loadOldConfigHelper(component, path string, target interface{}) error {
	if path == "" {
		return nil
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	ldr.Logger.Warnf("you should remove the legacy %v config file (%s) after migrating all config keys to the cluster configuration file (%s)", component, path, ldr.Path)

	err = yaml.Unmarshal(buf, target)
	if err != nil {
		return fmt.Errorf("%s: %s", path, err)
	}
	return nil
}

// update config using values from an old-style keepstore config file.
func (ldr *Loader) loadOldKeepstoreConfig(cfg *arvados.Config) error {
	var oc oldKeepstoreConfig
	err := ldr.loadOldConfigHelper("keepstore", ldr.KeepstorePath, &oc)
	if os.IsNotExist(err) && ldr.KeepstorePath == defaultKeepstoreConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	if v := oc.Debug; v == nil {
	} else if *v && cluster.SystemLogs.LogLevel != "debug" {
		cluster.SystemLogs.LogLevel = "debug"
	} else if !*v && cluster.SystemLogs.LogLevel != "info" {
		cluster.SystemLogs.LogLevel = "info"
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}

type oldCrunchDispatchSlurmConfig struct {
	Client *arvados.Client

	SbatchArguments *[]string
	PollPeriod      *arvados.Duration
	PrioritySpread  *int64

	// crunch-run command to invoke. The container UUID will be
	// appended. If nil, []string{"crunch-run"} will be used.
	//
	// Example: []string{"crunch-run", "--cgroup-parent-subsystem=memory"}
	CrunchRunCommand *[]string

	// Extra RAM to reserve (in Bytes) for SLURM job, in addition
	// to the amount specified in the container's RuntimeConstraints
	ReserveExtraRAM *int64

	// Minimum time between two attempts to run the same container
	MinRetryPeriod *arvados.Duration

	// Batch size for container queries
	BatchSize *int64
}

const defaultCrunchDispatchSlurmConfigPath = "/etc/arvados/crunch-dispatch-slurm/crunch-dispatch-slurm.yml"

func loadOldClientConfig(cluster *arvados.Cluster, client *arvados.Client) {
	if client == nil {
		return
	}
	if client.APIHost != "" {
		cluster.Services.Controller.ExternalURL.Host = client.APIHost
	}
	if client.Scheme != "" {
		cluster.Services.Controller.ExternalURL.Scheme = client.Scheme
	} else {
		cluster.Services.Controller.ExternalURL.Scheme = "https"
	}
	if client.AuthToken != "" {
		cluster.SystemRootToken = client.AuthToken
	}
	cluster.TLS.Insecure = client.Insecure
}

// update config using values from an crunch-dispatch-slurm config file.
func (ldr *Loader) loadOldCrunchDispatchSlurmConfig(cfg *arvados.Config) error {
	var oc oldCrunchDispatchSlurmConfig
	err := ldr.loadOldConfigHelper("crunch-dispatch-slurm", ldr.CrunchDispatchSlurmPath, &oc)
	if os.IsNotExist(err) && ldr.CrunchDispatchSlurmPath == defaultCrunchDispatchSlurmConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	loadOldClientConfig(cluster, oc.Client)

	if oc.SbatchArguments != nil {
		cluster.Containers.SLURM.SbatchArgumentsList = *oc.SbatchArguments
	}
	if oc.PollPeriod != nil {
		cluster.Containers.CloudVMs.PollInterval = *oc.PollPeriod
	}
	if oc.PrioritySpread != nil {
		cluster.Containers.SLURM.PrioritySpread = *oc.PrioritySpread
	}
	if oc.CrunchRunCommand != nil {
		if len(*oc.CrunchRunCommand) >= 1 {
			cluster.Containers.CrunchRunCommand = (*oc.CrunchRunCommand)[0]
		}
		if len(*oc.CrunchRunCommand) >= 2 {
			cluster.Containers.CrunchRunArgumentsList = (*oc.CrunchRunCommand)[1:]
		}
	}
	if oc.ReserveExtraRAM != nil {
		cluster.Containers.ReserveExtraRAM = arvados.ByteSize(*oc.ReserveExtraRAM)
	}
	if oc.MinRetryPeriod != nil {
		cluster.Containers.MinRetryPeriod = *oc.MinRetryPeriod
	}
	if oc.BatchSize != nil {
		cluster.API.MaxItemsPerResponse = int(*oc.BatchSize)
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}

type oldWsConfig struct {
	Client       *arvados.Client
	Postgres     *arvados.PostgreSQLConnection
	PostgresPool *int
	Listen       *string
	LogLevel     *string
	LogFormat    *string

	PingTimeout      *arvados.Duration
	ClientEventQueue *int
	ServerEventQueue *int

	ManagementToken *string
}

const defaultWebsocketsConfigPath = "/etc/arvados/ws/ws.yml"

// update config using values from an crunch-dispatch-slurm config file.
func (ldr *Loader) loadOldWebsocketsConfig(cfg *arvados.Config) error {
	var oc oldWsConfig
	err := ldr.loadOldConfigHelper("arvados-ws", ldr.WebsocketsPath, &oc)
	if os.IsNotExist(err) && ldr.WebsocketsPath == defaultWebsocketsConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	loadOldClientConfig(cluster, oc.Client)
	fmt.Printf("Clllllllllllient %v %v", *oc.Client, cluster.Services.Controller.ExternalURL)

	if oc.Postgres != nil {
		cluster.PostgreSQL.Connection = *oc.Postgres
	}
	if oc.PostgresPool != nil {
		cluster.PostgreSQL.ConnectionPool = *oc.PostgresPool
	}
	if oc.Listen != nil {
		cluster.Services.Websocket.InternalURLs[arvados.URL{Host: *oc.Listen}] = arvados.ServiceInstance{}
	}
	if oc.LogLevel != nil {
		cluster.SystemLogs.LogLevel = *oc.LogLevel
	}
	if oc.LogFormat != nil {
		cluster.SystemLogs.Format = *oc.LogFormat
	}
	if oc.PingTimeout != nil {
		cluster.API.WebsocketKeepaliveTimeout = *oc.PingTimeout
	}
	if oc.ClientEventQueue != nil {
		cluster.API.WebsocketClientEventQueue = *oc.ClientEventQueue
	}
	if oc.ServerEventQueue != nil {
		cluster.API.WebsocketServerEventQueue = *oc.ServerEventQueue
	}
	if oc.ManagementToken != nil {
		cluster.ManagementToken = *oc.ManagementToken
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}
