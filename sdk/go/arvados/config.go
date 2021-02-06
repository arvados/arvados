// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"git.arvados.org/arvados.git/sdk/go/config"
)

var DefaultConfigFile = func() string {
	if path := os.Getenv("ARVADOS_CONFIG"); path != "" {
		return path
	}
	return "/etc/arvados/config.yml"
}()

type Config struct {
	Clusters         map[string]Cluster
	AutoReloadConfig bool
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
	cc, ok := sc.Clusters[clusterID]
	if !ok {
		return nil, fmt.Errorf("cluster %q is not configured", clusterID)
	}
	cc.ClusterID = clusterID
	return &cc, nil
}

type WebDAVCacheConfig struct {
	TTL                  Duration
	UUIDTTL              Duration
	MaxBlockEntries      int
	MaxCollectionEntries int
	MaxCollectionBytes   int64
	MaxPermissionEntries int
	MaxUUIDEntries       int
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

	API struct {
		AsyncPermissionsUpdateInterval Duration
		DisabledAPIs                   StringSet
		MaxIndexDatabaseRead           int
		MaxItemsPerResponse            int
		MaxConcurrentRequests          int
		MaxKeepBlobBuffers             int
		MaxRequestAmplification        int
		MaxRequestSize                 int
		RequestTimeout                 Duration
		SendTimeout                    Duration
		WebsocketClientEventQueue      int
		WebsocketServerEventQueue      int
		KeepServiceRequestTimeout      Duration
	}
	AuditLogs struct {
		MaxAge             Duration
		MaxDeleteBatch     int
		UnloggedAttributes StringSet
	}
	Collections struct {
		BlobSigning              bool
		BlobSigningKey           string
		BlobSigningTTL           Duration
		BlobTrash                bool
		BlobTrashLifetime        Duration
		BlobTrashCheckInterval   Duration
		BlobTrashConcurrency     int
		BlobDeleteConcurrency    int
		BlobReplicateConcurrency int
		CollectionVersioning     bool
		DefaultTrashLifetime     Duration
		DefaultReplication       int
		ManagedProperties        map[string]struct {
			Value     interface{}
			Function  string
			Protected bool
		}
		PreserveVersionIfIdle        Duration
		TrashSweepInterval           Duration
		TrustAllContent              bool
		ForwardSlashNameSubstitution string
		S3FolderObjects              bool

		BlobMissingReport        string
		BalancePeriod            Duration
		BalanceCollectionBatch   int
		BalanceCollectionBuffers int
		BalanceTimeout           Duration

		WebDAVCache WebDAVCacheConfig
	}
	Git struct {
		GitCommand   string
		GitoliteHome string
		Repositories string
	}
	Login struct {
		LDAP struct {
			Enable             bool
			URL                URL
			StartTLS           bool
			InsecureTLS        bool
			StripDomain        string
			AppendDomain       string
			SearchAttribute    string
			SearchBindUser     string
			SearchBindPassword string
			SearchBase         string
			SearchFilters      string
			EmailAttribute     string
			UsernameAttribute  string
		}
		Google struct {
			Enable                          bool
			ClientID                        string
			ClientSecret                    string
			AlternateEmailAddresses         bool
			AuthenticationRequestParameters map[string]string
		}
		OpenIDConnect struct {
			Enable                          bool
			Issuer                          string
			ClientID                        string
			ClientSecret                    string
			EmailClaim                      string
			EmailVerifiedClaim              string
			UsernameClaim                   string
			AuthenticationRequestParameters map[string]string
		}
		PAM struct {
			Enable             bool
			Service            string
			DefaultEmailDomain string
		}
		SSO struct {
			Enable            bool
			ProviderAppID     string
			ProviderAppSecret string
		}
		Test struct {
			Enable bool
			Users  map[string]TestUser
		}
		LoginCluster       string
		RemoteTokenRefresh Duration
		TokenLifetime      Duration
		TrustedClients     map[string]struct{}
	}
	Mail struct {
		MailchimpAPIKey                string
		MailchimpListID                string
		SendUserSetupNotificationEmail bool
		IssueReporterEmailFrom         string
		IssueReporterEmailTo           string
		SupportEmailAddress            string
		EmailFrom                      string
	}
	SystemLogs struct {
		LogLevel                string
		Format                  string
		MaxRequestLogParamsSize int
	}
	TLS struct {
		Certificate string
		Key         string
		Insecure    bool
	}
	Users struct {
		AnonymousUserToken                    string
		AdminNotifierEmailFrom                string
		AutoAdminFirstUser                    bool
		AutoAdminUserWithEmail                string
		AutoSetupNewUsers                     bool
		AutoSetupNewUsersWithRepository       bool
		AutoSetupNewUsersWithVmUUID           string
		AutoSetupUsernameBlacklist            StringSet
		EmailSubjectPrefix                    string
		NewInactiveUserNotificationRecipients StringSet
		NewUserNotificationRecipients         StringSet
		NewUsersAreActive                     bool
		UserNotifierEmailFrom                 string
		UserProfileNotificationAddress        string
		PreferDomainForUsername               string
		UserSetupMailText                     string
	}
	Volumes   map[string]Volume
	Workbench struct {
		ActivationContactLink            string
		APIClientConnectTimeout          Duration
		APIClientReceiveTimeout          Duration
		APIResponseCompression           bool
		ApplicationMimetypesWithViewIcon StringSet
		ArvadosDocsite                   string
		ArvadosPublicDataDocURL          string
		DefaultOpenIdPrefix              string
		EnableGettingStartedPopup        bool
		EnablePublicProjectsPage         bool
		FileViewersConfigURL             string
		LogViewerMaxBytes                ByteSize
		MultiSiteSearch                  string
		ProfilingEnabled                 bool
		Repositories                     bool
		RepositoryCache                  string
		RunningJobLogRecordsToFetch      int
		SecretKeyBase                    string
		ShowRecentCollectionsOnDashboard bool
		ShowUserAgreementInline          bool
		ShowUserNotifications            bool
		SiteName                         string
		Theme                            string
		UserProfileFormFields            map[string]struct {
			Type                 string
			FormFieldTitle       string
			FormFieldDescription string
			Required             bool
			Position             int
			Options              map[string]struct{}
		}
		UserProfileFormMessage string
		VocabularyURL          string
		WelcomePageHTML        string
		InactivePageHTML       string
		SSHHelpPageHTML        string
		SSHHelpHostSuffix      string
		IdleTimeout            Duration
	}

	ForceLegacyAPI14 bool
}

type Volume struct {
	AccessViaHosts   map[URL]VolumeAccess
	ReadOnly         bool
	Replication      int
	StorageClasses   map[string]bool
	Driver           string
	DriverParameters json.RawMessage
}

type S3VolumeDriverParameters struct {
	IAMRole            string
	AccessKey          string
	SecretKey          string
	Endpoint           string
	Region             string
	Bucket             string
	LocationConstraint bool
	V2Signature        bool
	UseAWSS3v2Driver   bool
	IndexPageSize      int
	ConnectTimeout     Duration
	ReadTimeout        Duration
	RaceWindow         Duration
	UnsafeDelete       bool
}

type AzureVolumeDriverParameters struct {
	StorageAccountName   string
	StorageAccountKey    string
	StorageBaseURL       string
	ContainerName        string
	RequestTimeout       Duration
	ListBlobsRetryDelay  Duration
	ListBlobsMaxAttempts int
}

type DirectoryVolumeDriverParameters struct {
	Root      string
	Serialize bool
}

type VolumeAccess struct {
	ReadOnly bool
}

type Services struct {
	Composer       Service
	Controller     Service
	DispatchCloud  Service
	GitHTTP        Service
	GitSSH         Service
	Health         Service
	Keepbalance    Service
	Keepproxy      Service
	Keepstore      Service
	RailsAPI       Service
	SSO            Service
	WebDAVDownload Service
	WebDAV         Service
	WebShell       Service
	Websocket      Service
	Workbench1     Service
	Workbench2     Service
}

type Service struct {
	InternalURLs map[URL]ServiceInstance
	ExternalURL  URL
}

type TestUser struct {
	Email    string
	Password string
}

// URL is a url.URL that is also usable as a JSON key/value.
type URL url.URL

// UnmarshalText implements encoding.TextUnmarshaler so URL can be
// used as a JSON key/value.
func (su *URL) UnmarshalText(text []byte) error {
	u, err := url.Parse(string(text))
	if err == nil {
		*su = URL(*u)
		if su.Path == "" && su.Host != "" {
			// http://example really means http://example/
			su.Path = "/"
		}
	}
	return err
}

func (su URL) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%s", (*url.URL)(&su).String())), nil
}

func (su URL) String() string {
	return (*url.URL)(&su).String()
}

type ServiceInstance struct {
	Rendezvous string `json:",omitempty"`
}

type PostgreSQL struct {
	Connection     PostgreSQLConnection
	ConnectionPool int
}

type PostgreSQLConnection map[string]string

type RemoteCluster struct {
	Host          string
	Proxy         bool
	Scheme        string
	Insecure      bool
	ActivateUsers bool
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
	CloudVMs                    CloudVMsConfig
	CrunchRunCommand            string
	CrunchRunArgumentsList      []string
	DefaultKeepCacheRAM         ByteSize
	DispatchPrivateKey          string
	LogReuseDecisions           bool
	MaxComputeVMs               int
	MaxDispatchAttempts         int
	MaxRetryAttempts            int
	MinRetryPeriod              Duration
	ReserveExtraRAM             ByteSize
	StaleLockTimeout            Duration
	SupportedDockerImageFormats StringSet
	UsePreemptibleInstances     bool

	JobsAPI struct {
		Enable         string
		GitInternalDir string
	}
	Logging struct {
		MaxAge                       Duration
		LogBytesPerEvent             int
		LogSecondsBetweenEvents      Duration
		LogThrottlePeriod            Duration
		LogThrottleBytes             int
		LogThrottleLines             int
		LimitLogBytesPerJob          int
		LogPartialLineThrottlePeriod Duration
		LogUpdatePeriod              Duration
		LogUpdateSize                ByteSize
	}
	ShellAccess struct {
		Admin bool
		User  bool
	}
	SLURM struct {
		PrioritySpread             int64
		SbatchArgumentsList        []string
		SbatchEnvironmentVariables map[string]string
		Managed                    struct {
			DNSServerConfDir       string
			DNSServerConfTemplate  string
			DNSServerReloadCommand string
			DNSServerUpdateCommand string
			ComputeNodeDomain      string
			ComputeNodeNameservers StringSet
			AssignNodeHostname     string
		}
	}
}

type CloudVMsConfig struct {
	Enable bool

	BootProbeCommand               string
	DeployRunnerBinary             string
	ImageID                        string
	MaxCloudOpsPerSecond           int
	MaxProbesPerSecond             int
	MaxConcurrentInstanceCreateOps int
	PollInterval                   Duration
	ProbeInterval                  Duration
	SSHPort                        string
	SyncInterval                   Duration
	TimeoutBooting                 Duration
	TimeoutIdle                    Duration
	TimeoutProbe                   Duration
	TimeoutShutdown                Duration
	TimeoutSignal                  Duration
	TimeoutStaleRunLock            Duration
	TimeoutTERM                    Duration
	ResourceTags                   map[string]string
	TagKeyPrefix                   string

	Driver           string
	DriverParameters json.RawMessage
}

type InstanceTypeMap map[string]InstanceType

var errDuplicateInstanceTypeName = errors.New("duplicate instance type name")

// UnmarshalJSON handles old config files that provide an array of
// instance types instead of a hash.
func (it *InstanceTypeMap) UnmarshalJSON(data []byte) error {
	fixup := func(t InstanceType) (InstanceType, error) {
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
			return t, fmt.Errorf("InstanceType %q: Scratch != (IncludedScratch + AddedScratch)", t.Name)
		}
		return t, nil
	}

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
			t, err := fixup(t)
			if err != nil {
				return err
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
		t, err := fixup(t)
		if err != nil {
			return err
		}
		(*it)[name] = t
	}
	return nil
}

type StringSet map[string]struct{}

// UnmarshalJSON handles old config files that provide an array of
// instance types instead of a hash.
func (ss *StringSet) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '[' {
		var arr []string
		err := json.Unmarshal(data, &arr)
		if err != nil {
			return err
		}
		if len(arr) == 0 {
			*ss = nil
			return nil
		}
		*ss = make(map[string]struct{}, len(arr))
		for _, t := range arr {
			(*ss)[t] = struct{}{}
		}
		return nil
	}
	var hash map[string]struct{}
	err := json.Unmarshal(data, &hash)
	if err != nil {
		return err
	}
	*ss = make(map[string]struct{}, len(hash))
	for t := range hash {
		(*ss)[t] = struct{}{}
	}

	return nil
}

type ServiceName string

const (
	ServiceNameRailsAPI      ServiceName = "arvados-api-server"
	ServiceNameController    ServiceName = "arvados-controller"
	ServiceNameDispatchCloud ServiceName = "arvados-dispatch-cloud"
	ServiceNameHealth        ServiceName = "arvados-health"
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
		ServiceNameWorkbench1:    svcs.Workbench1,
		ServiceNameWorkbench2:    svcs.Workbench2,
		ServiceNameWebsocket:     svcs.Websocket,
		ServiceNameKeepbalance:   svcs.Keepbalance,
		ServiceNameKeepweb:       svcs.WebDAV,
		ServiceNameKeepproxy:     svcs.Keepproxy,
		ServiceNameKeepstore:     svcs.Keepstore,
	}
}
