// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/ghodss/yaml"
)

type deprRequestLimits struct {
	MaxItemsPerResponse            *int
	MultiClusterRequestConcurrency *int
}

type deprCluster struct {
	RequestLimits deprRequestLimits
	NodeProfiles  map[string]nodeProfile
	Login         struct {
		GoogleClientID                *string
		GoogleClientSecret            *string
		GoogleAlternateEmailAddresses *bool
		ProviderAppID                 *string
		ProviderAppSecret             *string
	}
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

		// Google* moved to Google.*
		if dst, n := &cluster.Login.Google.ClientID, dcluster.Login.GoogleClientID; n != nil && *n != *dst {
			*dst = *n
			if *n != "" {
				// In old config, non-empty ClientID meant enable
				cluster.Login.Google.Enable = true
			}
		}
		if dst, n := &cluster.Login.Google.ClientSecret, dcluster.Login.GoogleClientSecret; n != nil && *n != *dst {
			*dst = *n
		}
		if dst, n := &cluster.Login.Google.AlternateEmailAddresses, dcluster.Login.GoogleAlternateEmailAddresses; n != nil && *n != *dst {
			*dst = *n
		}

		cfg.Clusters[id] = cluster
	}
	return nil
}

func (ldr *Loader) applyDeprecatedVolumeDriverParameters(cfg *arvados.Config) error {
	for clusterID, cluster := range cfg.Clusters {
		for volID, vol := range cluster.Volumes {
			if vol.Driver == "S3" {
				var params struct {
					AccessKey       string `json:",omitempty"`
					SecretKey       string `json:",omitempty"`
					AccessKeyID     string
					SecretAccessKey string
				}
				err := json.Unmarshal(vol.DriverParameters, &params)
				if err != nil {
					return fmt.Errorf("error loading %s.Volumes.%s.DriverParameters: %w", clusterID, volID, err)
				}
				if params.AccessKey != "" || params.SecretKey != "" {
					if params.AccessKeyID != "" || params.SecretAccessKey != "" {
						return fmt.Errorf("cannot use old keys (AccessKey/SecretKey) and new keys (AccessKeyID/SecretAccessKey) at the same time in %s.Volumes.%s.DriverParameters -- you must remove the old config keys", clusterID, volID)
					}
					var allparams map[string]interface{}
					err = json.Unmarshal(vol.DriverParameters, &allparams)
					if err != nil {
						return fmt.Errorf("error loading %s.Volumes.%s.DriverParameters: %w", clusterID, volID, err)
					}
					for k := range allparams {
						if lk := strings.ToLower(k); lk == "accesskey" || lk == "secretkey" {
							delete(allparams, k)
						}
					}
					ldr.Logger.Warnf("using your old config keys %s.Volumes.%s.DriverParameters.AccessKey/SecretKey -- but you should rename them to AccessKeyID/SecretAccessKey", clusterID, volID)
					allparams["AccessKeyID"] = params.AccessKey
					allparams["SecretAccessKey"] = params.SecretKey
					vol.DriverParameters, err = json.Marshal(allparams)
					if err != nil {
						return err
					}
					cluster.Volumes[volID] = vol
				}
			}
		}
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
	svc.InternalURLs[arvados.URL{Scheme: scheme, Host: host, Path: "/"}] = arvados.ServiceInstance{}
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
		cluster.Services.Controller.ExternalURL.Path = "/"
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
	ks := ""
	for i, u := range client.KeepServiceURIs {
		if i > 0 {
			ks += " "
		}
		ks += u
	}
	cluster.Containers.SLURM.SbatchEnvironmentVariables = map[string]string{"ARVADOS_KEEP_SERVICES": ks}
}

// update config using values from an crunch-dispatch-slurm config file.
func (ldr *Loader) loadOldCrunchDispatchSlurmConfig(cfg *arvados.Config) error {
	if ldr.CrunchDispatchSlurmPath == "" {
		return nil
	}
	var oc oldCrunchDispatchSlurmConfig
	err := ldr.loadOldConfigHelper("crunch-dispatch-slurm", ldr.CrunchDispatchSlurmPath, &oc)
	if os.IsNotExist(err) && (ldr.CrunchDispatchSlurmPath == defaultCrunchDispatchSlurmConfigPath) {
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

const defaultWebsocketConfigPath = "/etc/arvados/ws/ws.yml"

// update config using values from an crunch-dispatch-slurm config file.
func (ldr *Loader) loadOldWebsocketConfig(cfg *arvados.Config) error {
	if ldr.WebsocketPath == "" {
		return nil
	}
	var oc oldWsConfig
	err := ldr.loadOldConfigHelper("arvados-ws", ldr.WebsocketPath, &oc)
	if os.IsNotExist(err) && ldr.WebsocketPath == defaultWebsocketConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	loadOldClientConfig(cluster, oc.Client)

	if oc.Postgres != nil {
		cluster.PostgreSQL.Connection = *oc.Postgres
	}
	if oc.PostgresPool != nil {
		cluster.PostgreSQL.ConnectionPool = *oc.PostgresPool
	}
	if oc.Listen != nil {
		cluster.Services.Websocket.InternalURLs[arvados.URL{Host: *oc.Listen, Path: "/"}] = arvados.ServiceInstance{}
	}
	if oc.LogLevel != nil {
		cluster.SystemLogs.LogLevel = *oc.LogLevel
	}
	if oc.LogFormat != nil {
		cluster.SystemLogs.Format = *oc.LogFormat
	}
	if oc.PingTimeout != nil {
		cluster.API.SendTimeout = *oc.PingTimeout
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

type oldKeepProxyConfig struct {
	Client          *arvados.Client
	Listen          *string
	DisableGet      *bool
	DisablePut      *bool
	DefaultReplicas *int
	Timeout         *arvados.Duration
	PIDFile         *string
	Debug           *bool
	ManagementToken *string
}

const defaultKeepproxyConfigPath = "/etc/arvados/keepproxy/keepproxy.yml"

func (ldr *Loader) loadOldKeepproxyConfig(cfg *arvados.Config) error {
	if ldr.KeepproxyPath == "" {
		return nil
	}
	var oc oldKeepProxyConfig
	err := ldr.loadOldConfigHelper("keepproxy", ldr.KeepproxyPath, &oc)
	if os.IsNotExist(err) && ldr.KeepproxyPath == defaultKeepproxyConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	loadOldClientConfig(cluster, oc.Client)

	if oc.Listen != nil {
		cluster.Services.Keepproxy.InternalURLs[arvados.URL{Host: *oc.Listen, Path: "/"}] = arvados.ServiceInstance{}
	}
	if oc.DefaultReplicas != nil {
		cluster.Collections.DefaultReplication = *oc.DefaultReplicas
	}
	if oc.Timeout != nil {
		cluster.API.KeepServiceRequestTimeout = *oc.Timeout
	}
	if oc.Debug != nil {
		if *oc.Debug && cluster.SystemLogs.LogLevel != "debug" {
			cluster.SystemLogs.LogLevel = "debug"
		} else if !*oc.Debug && cluster.SystemLogs.LogLevel != "info" {
			cluster.SystemLogs.LogLevel = "info"
		}
	}
	if oc.ManagementToken != nil {
		cluster.ManagementToken = *oc.ManagementToken
	}

	// The following legacy options are no longer supported. If they are set to
	// true or PIDFile has a value, error out and notify the user
	unsupportedEntry := func(cfgEntry string) error {
		return fmt.Errorf("the keepproxy %s configuration option is no longer supported, please remove it from your configuration file", cfgEntry)
	}
	if oc.DisableGet != nil && *oc.DisableGet {
		return unsupportedEntry("DisableGet")
	}
	if oc.DisablePut != nil && *oc.DisablePut {
		return unsupportedEntry("DisablePut")
	}
	if oc.PIDFile != nil && *oc.PIDFile != "" {
		return unsupportedEntry("PIDFile")
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}

const defaultKeepWebConfigPath = "/etc/arvados/keep-web/keep-web.yml"

type oldKeepWebConfig struct {
	Client *arvados.Client

	Listen *string

	AnonymousTokens    *[]string
	AttachmentOnlyHost *string
	TrustAllContent    *bool

	Cache struct {
		TTL                  *arvados.Duration
		UUIDTTL              *arvados.Duration
		MaxCollectionEntries *int
		MaxCollectionBytes   *int64
		MaxUUIDEntries       *int
	}

	// Hack to support old command line flag, which is a bool
	// meaning "get actual token from environment".
	deprecatedAllowAnonymous *bool

	// Authorization token to be included in all health check requests.
	ManagementToken *string
}

func (ldr *Loader) loadOldKeepWebConfig(cfg *arvados.Config) error {
	if ldr.KeepWebPath == "" {
		return nil
	}
	var oc oldKeepWebConfig
	err := ldr.loadOldConfigHelper("keep-web", ldr.KeepWebPath, &oc)
	if os.IsNotExist(err) && ldr.KeepWebPath == defaultKeepWebConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	loadOldClientConfig(cluster, oc.Client)

	if oc.Listen != nil {
		cluster.Services.WebDAV.InternalURLs[arvados.URL{Host: *oc.Listen, Path: "/"}] = arvados.ServiceInstance{}
		cluster.Services.WebDAVDownload.InternalURLs[arvados.URL{Host: *oc.Listen, Path: "/"}] = arvados.ServiceInstance{}
	}
	if oc.AttachmentOnlyHost != nil {
		cluster.Services.WebDAVDownload.ExternalURL = arvados.URL{Host: *oc.AttachmentOnlyHost, Path: "/"}
	}
	if oc.ManagementToken != nil {
		cluster.ManagementToken = *oc.ManagementToken
	}
	if oc.TrustAllContent != nil {
		cluster.Collections.TrustAllContent = *oc.TrustAllContent
	}
	if oc.Cache.TTL != nil {
		cluster.Collections.WebDAVCache.TTL = *oc.Cache.TTL
	}
	if oc.Cache.UUIDTTL != nil {
		cluster.Collections.WebDAVCache.UUIDTTL = *oc.Cache.UUIDTTL
	}
	if oc.Cache.MaxCollectionEntries != nil {
		cluster.Collections.WebDAVCache.MaxCollectionEntries = *oc.Cache.MaxCollectionEntries
	}
	if oc.Cache.MaxCollectionBytes != nil {
		cluster.Collections.WebDAVCache.MaxCollectionBytes = *oc.Cache.MaxCollectionBytes
	}
	if oc.Cache.MaxUUIDEntries != nil {
		cluster.Collections.WebDAVCache.MaxUUIDEntries = *oc.Cache.MaxUUIDEntries
	}
	if oc.AnonymousTokens != nil {
		if len(*oc.AnonymousTokens) > 0 {
			cluster.Users.AnonymousUserToken = (*oc.AnonymousTokens)[0]
			if len(*oc.AnonymousTokens) > 1 {
				ldr.Logger.Warn("More than 1 anonymous tokens configured, using only the first and discarding the rest.")
			}
		}
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}

const defaultGitHttpdConfigPath = "/etc/arvados/git-httpd/git-httpd.yml"

type oldGitHttpdConfig struct {
	Client          *arvados.Client
	Listen          *string
	GitCommand      *string
	GitoliteHome    *string
	RepoRoot        *string
	ManagementToken *string
}

func (ldr *Loader) loadOldGitHttpdConfig(cfg *arvados.Config) error {
	if ldr.GitHttpdPath == "" {
		return nil
	}
	var oc oldGitHttpdConfig
	err := ldr.loadOldConfigHelper("arvados-git-httpd", ldr.GitHttpdPath, &oc)
	if os.IsNotExist(err) && ldr.GitHttpdPath == defaultGitHttpdConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	loadOldClientConfig(cluster, oc.Client)

	if oc.Listen != nil {
		cluster.Services.GitHTTP.InternalURLs[arvados.URL{Host: *oc.Listen}] = arvados.ServiceInstance{}
	}
	if oc.ManagementToken != nil {
		cluster.ManagementToken = *oc.ManagementToken
	}
	if oc.GitCommand != nil {
		cluster.Git.GitCommand = *oc.GitCommand
	}
	if oc.GitoliteHome != nil {
		cluster.Git.GitoliteHome = *oc.GitoliteHome
	}
	if oc.RepoRoot != nil {
		cluster.Git.Repositories = *oc.RepoRoot
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}

const defaultKeepBalanceConfigPath = "/etc/arvados/keep-balance/keep-balance.yml"

type oldKeepBalanceConfig struct {
	Client              *arvados.Client
	Listen              *string
	KeepServiceTypes    *[]string
	KeepServiceList     *arvados.KeepServiceList
	RunPeriod           *arvados.Duration
	CollectionBatchSize *int
	CollectionBuffers   *int
	RequestTimeout      *arvados.Duration
	LostBlocksFile      *string
	ManagementToken     *string
}

func (ldr *Loader) loadOldKeepBalanceConfig(cfg *arvados.Config) error {
	if ldr.KeepBalancePath == "" {
		return nil
	}
	var oc oldKeepBalanceConfig
	err := ldr.loadOldConfigHelper("keep-balance", ldr.KeepBalancePath, &oc)
	if os.IsNotExist(err) && ldr.KeepBalancePath == defaultKeepBalanceConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	loadOldClientConfig(cluster, oc.Client)

	if oc.Listen != nil {
		cluster.Services.Keepbalance.InternalURLs[arvados.URL{Host: *oc.Listen}] = arvados.ServiceInstance{}
	}
	if oc.ManagementToken != nil {
		cluster.ManagementToken = *oc.ManagementToken
	}
	if oc.RunPeriod != nil {
		cluster.Collections.BalancePeriod = *oc.RunPeriod
	}
	if oc.LostBlocksFile != nil {
		cluster.Collections.BlobMissingReport = *oc.LostBlocksFile
	}
	if oc.CollectionBatchSize != nil {
		cluster.Collections.BalanceCollectionBatch = *oc.CollectionBatchSize
	}
	if oc.CollectionBuffers != nil {
		cluster.Collections.BalanceCollectionBuffers = *oc.CollectionBuffers
	}
	if oc.RequestTimeout != nil {
		cluster.API.KeepServiceRequestTimeout = *oc.RequestTimeout
	}

	msg := "The %s configuration option is no longer supported. Please remove it from your configuration file. See the keep-balance upgrade notes at https://doc.arvados.org/admin/upgrading.html for more details."

	// If the keep service type provided is "disk" silently ignore it, since
	// this is what ends up being done anyway.
	if oc.KeepServiceTypes != nil {
		numTypes := len(*oc.KeepServiceTypes)
		if numTypes != 0 && !(numTypes == 1 && (*oc.KeepServiceTypes)[0] == "disk") {
			return fmt.Errorf(msg, "KeepServiceTypes")
		}
	}

	if oc.KeepServiceList != nil {
		return fmt.Errorf(msg, "KeepServiceList")
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}

func (ldr *Loader) loadOldEnvironmentVariables(cfg *arvados.Config) error {
	if os.Getenv("ARVADOS_API_TOKEN") == "" && os.Getenv("ARVADOS_API_HOST") == "" {
		return nil
	}
	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}
	if tok := os.Getenv("ARVADOS_API_TOKEN"); tok != "" && cluster.SystemRootToken == "" {
		ldr.Logger.Warn("SystemRootToken missing from cluster config, falling back to ARVADOS_API_TOKEN environment variable")
		cluster.SystemRootToken = tok
	}
	if apihost := os.Getenv("ARVADOS_API_HOST"); apihost != "" && cluster.Services.Controller.ExternalURL.Host == "" {
		ldr.Logger.Warn("Services.Controller.ExternalURL missing from cluster config, falling back to ARVADOS_API_HOST(_INSECURE) environment variables")
		u, err := url.Parse("https://" + apihost)
		if err != nil {
			return fmt.Errorf("cannot parse ARVADOS_API_HOST: %s", err)
		}
		cluster.Services.Controller.ExternalURL = arvados.URL(*u)
		if i := os.Getenv("ARVADOS_API_HOST_INSECURE"); i != "" && i != "0" {
			cluster.TLS.Insecure = true
		}
	}
	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}
