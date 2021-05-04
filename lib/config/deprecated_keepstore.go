// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

const defaultKeepstoreConfigPath = "/etc/arvados/keepstore/keepstore.yml"

type oldKeepstoreConfig struct {
	Debug  *bool
	Listen *string

	LogFormat *string

	PIDFile *string

	MaxBuffers  *int
	MaxRequests *int

	BlobSignatureTTL    *arvados.Duration
	BlobSigningKeyFile  *string
	RequireSignatures   *bool
	SystemAuthTokenFile *string
	EnableDelete        *bool
	TrashLifetime       *arvados.Duration
	TrashCheckInterval  *arvados.Duration
	PullWorkers         *int
	TrashWorkers        *int
	EmptyTrashWorkers   *int
	TLSCertificateFile  *string
	TLSKeyFile          *string

	Volumes *oldKeepstoreVolumeList

	ManagementToken *string

	DiscoverVolumesFromMountsFile string // not a real legacy config -- just useful for tests
}

type oldKeepstoreVolumeList []oldKeepstoreVolume

type oldKeepstoreVolume struct {
	arvados.Volume
	Type string `json:",omitempty"`

	// Azure driver configs
	StorageAccountName    string           `json:",omitempty"`
	StorageAccountKeyFile string           `json:",omitempty"`
	StorageBaseURL        string           `json:",omitempty"`
	ContainerName         string           `json:",omitempty"`
	AzureReplication      int              `json:",omitempty"`
	RequestTimeout        arvados.Duration `json:",omitempty"`
	ListBlobsRetryDelay   arvados.Duration `json:",omitempty"`
	ListBlobsMaxAttempts  int              `json:",omitempty"`

	// S3 driver configs
	AccessKeyFile      string           `json:",omitempty"`
	SecretKeyFile      string           `json:",omitempty"`
	Endpoint           string           `json:",omitempty"`
	Region             string           `json:",omitempty"`
	Bucket             string           `json:",omitempty"`
	LocationConstraint bool             `json:",omitempty"`
	IndexPageSize      int              `json:",omitempty"`
	S3Replication      int              `json:",omitempty"`
	ConnectTimeout     arvados.Duration `json:",omitempty"`
	ReadTimeout        arvados.Duration `json:",omitempty"`
	RaceWindow         arvados.Duration `json:",omitempty"`
	UnsafeDelete       bool             `json:",omitempty"`

	// Directory driver configs
	Root                 string
	DirectoryReplication int
	Serialize            bool

	// Common configs
	ReadOnly       bool     `json:",omitempty"`
	StorageClasses []string `json:",omitempty"`
}

// update config using values from an old-style keepstore config file.
func (ldr *Loader) loadOldKeepstoreConfig(cfg *arvados.Config) error {
	if ldr.KeepstorePath == "" {
		return nil
	}
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("getting hostname: %s", err)
	}

	var oc oldKeepstoreConfig
	err = ldr.loadOldConfigHelper("keepstore", ldr.KeepstorePath, &oc)
	if os.IsNotExist(err) && ldr.KeepstorePath == defaultKeepstoreConfigPath {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	myURL := arvados.URL{Scheme: "http", Path: "/"}
	if oc.TLSCertificateFile != nil && oc.TLSKeyFile != nil {
		myURL.Scheme = "https"
	}

	if v := oc.Debug; v == nil {
	} else if *v && cluster.SystemLogs.LogLevel != "debug" {
		cluster.SystemLogs.LogLevel = "debug"
	} else if !*v && cluster.SystemLogs.LogLevel != "info" {
		cluster.SystemLogs.LogLevel = "info"
	}

	if v := oc.TLSCertificateFile; v != nil {
		cluster.TLS.Certificate = "file://" + *v
	}
	if v := oc.TLSKeyFile; v != nil {
		cluster.TLS.Key = "file://" + *v
	}
	if v := oc.Listen; v != nil {
		if _, ok := cluster.Services.Keepstore.InternalURLs[arvados.URL{Scheme: myURL.Scheme, Host: *v, Path: "/"}]; ok {
			// already listed
			myURL.Host = *v
		} else if len(*v) > 1 && (*v)[0] == ':' {
			myURL.Host = net.JoinHostPort(hostname, (*v)[1:])
			cluster.Services.Keepstore.InternalURLs[myURL] = arvados.ServiceInstance{}
		} else {
			return fmt.Errorf("unable to migrate Listen value %q -- you must update Services.Keepstore.InternalURLs manually, and comment out the Listen entry in your legacy keepstore config file", *v)
		}
	} else {
		for url := range cluster.Services.Keepstore.InternalURLs {
			if host, _, _ := net.SplitHostPort(url.Host); host == hostname {
				myURL = url
				break
			}
		}
		if myURL.Host == "" {
			return fmt.Errorf("unable to migrate legacy keepstore config: no 'Listen' key, and hostname %q does not match an entry in Services.Keepstore.InternalURLs", hostname)
		}
	}

	if v := oc.LogFormat; v != nil {
		cluster.SystemLogs.Format = *v
	}
	if v := oc.MaxBuffers; v != nil {
		cluster.API.MaxKeepBlobBuffers = *v
	}
	if v := oc.MaxRequests; v != nil {
		cluster.API.MaxConcurrentRequests = *v
	}
	if v := oc.BlobSignatureTTL; v != nil {
		cluster.Collections.BlobSigningTTL = *v
	}
	if v := oc.BlobSigningKeyFile; v != nil {
		buf, err := ioutil.ReadFile(*v)
		if err != nil {
			return fmt.Errorf("error reading BlobSigningKeyFile: %s", err)
		}
		if key := strings.TrimSpace(string(buf)); key != cluster.Collections.BlobSigningKey {
			cluster.Collections.BlobSigningKey = key
		}
	}
	if v := oc.RequireSignatures; v != nil {
		cluster.Collections.BlobSigning = *v
	}
	if v := oc.SystemAuthTokenFile; v != nil {
		f, err := os.Open(*v)
		if err != nil {
			return fmt.Errorf("error opening SystemAuthTokenFile: %s", err)
		}
		defer f.Close()
		buf, err := ioutil.ReadAll(f)
		if err != nil {
			return fmt.Errorf("error reading SystemAuthTokenFile: %s", err)
		}
		if key := strings.TrimSpace(string(buf)); key != cluster.SystemRootToken {
			cluster.SystemRootToken = key
		}
	}
	if v := oc.EnableDelete; v != nil {
		cluster.Collections.BlobTrash = *v
	}
	if v := oc.TrashLifetime; v != nil {
		cluster.Collections.BlobTrashLifetime = *v
	}
	if v := oc.TrashCheckInterval; v != nil {
		cluster.Collections.BlobTrashCheckInterval = *v
	}
	if v := oc.TrashWorkers; v != nil {
		cluster.Collections.BlobTrashConcurrency = *v
	}
	if v := oc.EmptyTrashWorkers; v != nil {
		cluster.Collections.BlobDeleteConcurrency = *v
	}
	if v := oc.PullWorkers; v != nil {
		cluster.Collections.BlobReplicateConcurrency = *v
	}
	if oc.Volumes == nil || len(*oc.Volumes) == 0 {
		ldr.Logger.Warn("no volumes in legacy config; discovering local directory volumes")
		err := ldr.discoverLocalVolumes(cluster, oc.DiscoverVolumesFromMountsFile, myURL)
		if err != nil {
			return fmt.Errorf("error discovering local directory volumes: %s", err)
		}
	} else {
		err := ldr.migrateOldKeepstoreVolumes(cluster, oc, myURL)
		if err != nil {
			return err
		}
	}

	if err := ldr.checkPendingKeepstoreMigrations(cluster); err != nil {
		return err
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
}

// Merge Volumes section of old keepstore config into cluster config.
func (ldr *Loader) migrateOldKeepstoreVolumes(cluster *arvados.Cluster, oc oldKeepstoreConfig, myURL arvados.URL) error {
	for i, oldvol := range *oc.Volumes {
		var accessViaHosts map[arvados.URL]arvados.VolumeAccess
		oldUUID, found := ldr.alreadyMigrated(oldvol, cluster.Volumes, myURL)
		if found {
			accessViaHosts = cluster.Volumes[oldUUID].AccessViaHosts
			writers := false
			for _, va := range accessViaHosts {
				if !va.ReadOnly {
					writers = true
				}
			}
			if writers || len(accessViaHosts) == 0 {
				ldr.Logger.Infof("ignoring volume #%d's parameters in legacy keepstore config: using matching entry in cluster config instead", i)
				if len(accessViaHosts) > 0 {
					cluster.Volumes[oldUUID].AccessViaHosts[myURL] = arvados.VolumeAccess{ReadOnly: oldvol.ReadOnly}
				}
				continue
			}
		}
		var newvol arvados.Volume
		if found {
			ldr.Logger.Infof("ignoring volume #%d's parameters in legacy keepstore config: using matching entry in cluster config instead", i)
			newvol = cluster.Volumes[oldUUID]
			// Remove the old entry. It will be added back
			// below, possibly with a new UUID.
			delete(cluster.Volumes, oldUUID)
		} else {
			v, err := ldr.translateOldKeepstoreVolume(oldvol)
			if err != nil {
				return err
			}
			newvol = v
		}
		if accessViaHosts == nil {
			accessViaHosts = make(map[arvados.URL]arvados.VolumeAccess, 1)
		}
		accessViaHosts[myURL] = arvados.VolumeAccess{ReadOnly: oldvol.ReadOnly}
		newvol.AccessViaHosts = accessViaHosts

		volUUID := oldUUID
		if oldvol.ReadOnly {
		} else if oc.Listen == nil {
			ldr.Logger.Warn("cannot find optimal volume UUID because Listen address is not given in legacy keepstore config")
		} else if uuid, _, err := findKeepServicesItem(cluster, *oc.Listen); err != nil {
			ldr.Logger.WithError(err).Warn("cannot find optimal volume UUID: failed to find a matching keep_service listing for this legacy keepstore config")
		} else if len(uuid) != 27 {
			ldr.Logger.WithField("UUID", uuid).Warn("cannot find optimal volume UUID: keep_service UUID does not have expected format")
		} else {
			rendezvousUUID := cluster.ClusterID + "-nyw5e-" + uuid[12:]
			if _, ok := cluster.Volumes[rendezvousUUID]; ok {
				ldr.Logger.Warn("suggesting a random volume UUID because the volume ID matching our keep_service UUID is already in use")
			} else {
				volUUID = rendezvousUUID
			}
			si := cluster.Services.Keepstore.InternalURLs[myURL]
			si.Rendezvous = uuid[12:]
			cluster.Services.Keepstore.InternalURLs[myURL] = si
		}
		if volUUID == "" {
			volUUID = newUUID(cluster.ClusterID, "nyw5e")
			ldr.Logger.WithField("UUID", volUUID).Infof("suggesting a random volume UUID for volume #%d in legacy config", i)
		}
		cluster.Volumes[volUUID] = newvol
	}
	return nil
}

func (ldr *Loader) translateOldKeepstoreVolume(oldvol oldKeepstoreVolume) (arvados.Volume, error) {
	var newvol arvados.Volume
	var params interface{}
	switch oldvol.Type {
	case "S3":
		accesskeydata, err := ioutil.ReadFile(oldvol.AccessKeyFile)
		if err != nil && oldvol.AccessKeyFile != "" {
			return newvol, fmt.Errorf("error reading AccessKeyFile: %s", err)
		}
		secretkeydata, err := ioutil.ReadFile(oldvol.SecretKeyFile)
		if err != nil && oldvol.SecretKeyFile != "" {
			return newvol, fmt.Errorf("error reading SecretKeyFile: %s", err)
		}
		newvol = arvados.Volume{
			Driver:         "S3",
			ReadOnly:       oldvol.ReadOnly,
			Replication:    oldvol.S3Replication,
			StorageClasses: array2boolmap(oldvol.StorageClasses),
		}
		params = arvados.S3VolumeDriverParameters{
			AccessKeyID:        string(bytes.TrimSpace(accesskeydata)),
			SecretAccessKey:    string(bytes.TrimSpace(secretkeydata)),
			Endpoint:           oldvol.Endpoint,
			Region:             oldvol.Region,
			Bucket:             oldvol.Bucket,
			LocationConstraint: oldvol.LocationConstraint,
			IndexPageSize:      oldvol.IndexPageSize,
			ConnectTimeout:     oldvol.ConnectTimeout,
			ReadTimeout:        oldvol.ReadTimeout,
			RaceWindow:         oldvol.RaceWindow,
			UnsafeDelete:       oldvol.UnsafeDelete,
		}
	case "Azure":
		keydata, err := ioutil.ReadFile(oldvol.StorageAccountKeyFile)
		if err != nil && oldvol.StorageAccountKeyFile != "" {
			return newvol, fmt.Errorf("error reading StorageAccountKeyFile: %s", err)
		}
		newvol = arvados.Volume{
			Driver:         "Azure",
			ReadOnly:       oldvol.ReadOnly,
			Replication:    oldvol.AzureReplication,
			StorageClasses: array2boolmap(oldvol.StorageClasses),
		}
		params = arvados.AzureVolumeDriverParameters{
			StorageAccountName:   oldvol.StorageAccountName,
			StorageAccountKey:    string(bytes.TrimSpace(keydata)),
			StorageBaseURL:       oldvol.StorageBaseURL,
			ContainerName:        oldvol.ContainerName,
			RequestTimeout:       oldvol.RequestTimeout,
			ListBlobsRetryDelay:  oldvol.ListBlobsRetryDelay,
			ListBlobsMaxAttempts: oldvol.ListBlobsMaxAttempts,
		}
	case "Directory":
		newvol = arvados.Volume{
			Driver:         "Directory",
			ReadOnly:       oldvol.ReadOnly,
			Replication:    oldvol.DirectoryReplication,
			StorageClasses: array2boolmap(oldvol.StorageClasses),
		}
		params = arvados.DirectoryVolumeDriverParameters{
			Root:      oldvol.Root,
			Serialize: oldvol.Serialize,
		}
	default:
		return newvol, fmt.Errorf("unsupported volume type %q", oldvol.Type)
	}
	dp, err := json.Marshal(params)
	if err != nil {
		return newvol, err
	}
	newvol.DriverParameters = json.RawMessage(dp)
	if newvol.Replication < 1 {
		newvol.Replication = 1
	}
	return newvol, nil
}

func (ldr *Loader) alreadyMigrated(oldvol oldKeepstoreVolume, newvols map[string]arvados.Volume, myURL arvados.URL) (string, bool) {
	for uuid, newvol := range newvols {
		if oldvol.Type != newvol.Driver {
			continue
		}
		switch oldvol.Type {
		case "S3":
			var params arvados.S3VolumeDriverParameters
			if err := json.Unmarshal(newvol.DriverParameters, &params); err == nil &&
				oldvol.Endpoint == params.Endpoint &&
				oldvol.Region == params.Region &&
				oldvol.Bucket == params.Bucket &&
				oldvol.LocationConstraint == params.LocationConstraint {
				return uuid, true
			}
		case "Azure":
			var params arvados.AzureVolumeDriverParameters
			if err := json.Unmarshal(newvol.DriverParameters, &params); err == nil &&
				oldvol.StorageAccountName == params.StorageAccountName &&
				oldvol.StorageBaseURL == params.StorageBaseURL &&
				oldvol.ContainerName == params.ContainerName {
				return uuid, true
			}
		case "Directory":
			var params arvados.DirectoryVolumeDriverParameters
			if err := json.Unmarshal(newvol.DriverParameters, &params); err == nil &&
				oldvol.Root == params.Root {
				if _, ok := newvol.AccessViaHosts[myURL]; ok || len(newvol.AccessViaHosts) == 0 {
					return uuid, true
				}
			}
		}
	}
	return "", false
}

func (ldr *Loader) discoverLocalVolumes(cluster *arvados.Cluster, mountsFile string, myURL arvados.URL) error {
	if mountsFile == "" {
		mountsFile = "/proc/mounts"
	}
	f, err := os.Open(mountsFile)
	if err != nil {
		return fmt.Errorf("error opening %s: %s", mountsFile, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		args := strings.Fields(scanner.Text())
		dev, mount := args[0], args[1]
		if mount == "/" {
			continue
		}
		if dev != "tmpfs" && !strings.HasPrefix(dev, "/dev/") {
			continue
		}
		keepdir := mount + "/keep"
		if st, err := os.Stat(keepdir); err != nil || !st.IsDir() {
			continue
		}

		ro := false
		for _, fsopt := range strings.Split(args[3], ",") {
			if fsopt == "ro" {
				ro = true
			}
		}

		uuid := newUUID(cluster.ClusterID, "nyw5e")
		ldr.Logger.WithFields(logrus.Fields{
			"UUID":                       uuid,
			"Driver":                     "Directory",
			"DriverParameters.Root":      keepdir,
			"DriverParameters.Serialize": false,
			"ReadOnly":                   ro,
			"Replication":                1,
		}).Warn("adding local directory volume")

		p, err := json.Marshal(arvados.DirectoryVolumeDriverParameters{
			Root:      keepdir,
			Serialize: false,
		})
		if err != nil {
			panic(err)
		}
		cluster.Volumes[uuid] = arvados.Volume{
			Driver:           "Directory",
			DriverParameters: p,
			ReadOnly:         ro,
			Replication:      1,
			AccessViaHosts: map[arvados.URL]arvados.VolumeAccess{
				myURL: {ReadOnly: ro},
			},
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading %s: %s", mountsFile, err)
	}
	return nil
}

func array2boolmap(keys []string) map[string]bool {
	m := map[string]bool{}
	for _, k := range keys {
		m[k] = true
	}
	return m
}

func newUUID(clusterID, infix string) string {
	randint, err := rand.Int(rand.Reader, big.NewInt(0).Exp(big.NewInt(36), big.NewInt(15), big.NewInt(0)))
	if err != nil {
		panic(err)
	}
	randstr := randint.Text(36)
	for len(randstr) < 15 {
		randstr = "0" + randstr
	}
	return fmt.Sprintf("%s-%s-%s", clusterID, infix, randstr)
}

// Return the UUID and URL for the controller's keep_services listing
// corresponding to this host/process.
func findKeepServicesItem(cluster *arvados.Cluster, listen string) (uuid string, url arvados.URL, err error) {
	client, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return
	}
	client.AuthToken = cluster.SystemRootToken
	var svcList arvados.KeepServiceList
	err = client.RequestAndDecode(&svcList, "GET", "arvados/v1/keep_services", nil, nil)
	if err != nil {
		return
	}
	hostname, err := os.Hostname()
	if err != nil {
		err = fmt.Errorf("error getting hostname: %s", err)
		return
	}
	var tried []string
	for _, ks := range svcList.Items {
		if ks.ServiceType == "proxy" {
			continue
		} else if keepServiceIsMe(ks, hostname, listen) {
			return ks.UUID, keepServiceURL(ks), nil
		} else {
			tried = append(tried, fmt.Sprintf("%s:%d", ks.ServiceHost, ks.ServicePort))
		}
	}
	err = fmt.Errorf("listen address %q does not match any of the non-proxy keep_services entries %q", listen, tried)
	return
}

func keepServiceURL(ks arvados.KeepService) arvados.URL {
	url := arvados.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(ks.ServiceHost, strconv.Itoa(ks.ServicePort)),
		Path:   "/",
	}
	if ks.ServiceSSLFlag {
		url.Scheme = "https"
	}
	return url
}

var localhostOrAllInterfaces = map[string]bool{
	"localhost": true,
	"127.0.0.1": true,
	"::1":       true,
	"::":        true,
	"":          true,
}

// Return true if the given KeepService entry matches the given
// hostname and (keepstore config file) listen address.
//
// If the KeepService host is some variant of "localhost", we assume
// this is a testing or single-node environment, ignore the given
// hostname, and return true if the port numbers match.
//
// The hostname isn't assumed to be a FQDN: a hostname "foo.bar" will
// match a KeepService host "foo.bar", but also "foo.bar.example",
// "foo.bar.example.org", etc.
func keepServiceIsMe(ks arvados.KeepService, hostname string, listen string) bool {
	// Extract the port name/number from listen, and resolve it to
	// a port number to compare with ks.ServicePort.
	_, listenport, err := net.SplitHostPort(listen)
	if err != nil && strings.HasPrefix(listen, ":") {
		listenport = listen[1:]
	}
	if lp, err := net.LookupPort("tcp", listenport); err != nil {
		return false
	} else if !(lp == ks.ServicePort ||
		(lp == 0 && ks.ServicePort == 80)) {
		return false
	}

	kshost := strings.ToLower(ks.ServiceHost)
	return localhostOrAllInterfaces[kshost] || strings.HasPrefix(kshost+".", strings.ToLower(hostname)+".")
}

// Warn about pending keepstore migration tasks that haven't already
// been warned about in loadOldKeepstoreConfig() -- i.e., unmigrated
// keepstore hosts other than the present host, and obsolete content
// in the keep_services table.
func (ldr *Loader) checkPendingKeepstoreMigrations(cluster *arvados.Cluster) error {
	if cluster.Services.Controller.ExternalURL.String() == "" {
		ldr.Logger.Debug("Services.Controller.ExternalURL not configured -- skipping check for pending keepstore config migrations")
		return nil
	}
	if ldr.SkipAPICalls {
		ldr.Logger.Debug("(Loader).SkipAPICalls == true -- skipping check for pending keepstore config migrations")
		return nil
	}
	client, err := arvados.NewClientFromConfig(cluster)
	if err != nil {
		return err
	}
	client.AuthToken = cluster.SystemRootToken
	var svcList arvados.KeepServiceList
	err = client.RequestAndDecode(&svcList, "GET", "arvados/v1/keep_services", nil, nil)
	if err != nil {
		ldr.Logger.WithError(err).Warn("error retrieving keep_services list -- skipping check for pending keepstore config migrations")
		return nil
	}
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("error getting hostname: %s", err)
	}
	sawTimes := map[time.Time]bool{}
	for _, ks := range svcList.Items {
		sawTimes[ks.CreatedAt] = true
		sawTimes[ks.ModifiedAt] = true
	}
	if len(sawTimes) <= 1 {
		// If all timestamps in the arvados/v1/keep_services
		// response are identical, it's a clear sign the
		// response was generated on the fly from the cluster
		// config, rather than real database records. In that
		// case (as well as the case where none are listed at
		// all) it's pointless to look for entries that
		// haven't yet been migrated to the config file.
		return nil
	}
	needDBRows := false
	for _, ks := range svcList.Items {
		if ks.ServiceType == "proxy" {
			if len(cluster.Services.Keepproxy.InternalURLs) == 0 {
				needDBRows = true
				ldr.Logger.Warn("you should migrate your keepproxy configuration to the cluster configuration file")
			}
			continue
		}
		kshost := strings.ToLower(ks.ServiceHost)
		if localhostOrAllInterfaces[kshost] || strings.HasPrefix(kshost+".", strings.ToLower(hostname)+".") {
			// it would be confusing to recommend
			// migrating *this* host's legacy keepstore
			// config immediately after explaining that
			// very migration process in more detail.
			continue
		}
		ksurl := keepServiceURL(ks)
		if _, ok := cluster.Services.Keepstore.InternalURLs[ksurl]; ok {
			// already added to InternalURLs
			continue
		}
		ldr.Logger.Warnf("you should migrate the legacy keepstore configuration file on host %s", ks.ServiceHost)
	}
	if !needDBRows {
		ldr.Logger.Warn("you should delete all of your manually added keep_services listings using `arv --format=uuid keep_service list | xargs -n1 arv keep_service delete --uuid` -- when those are deleted, the services listed in your cluster configuration will be used instead")
	}
	return nil
}

// Warn about keepstore servers that have no volumes.
func (ldr *Loader) checkEmptyKeepstores(cluster arvados.Cluster) error {
servers:
	for url := range cluster.Services.Keepstore.InternalURLs {
		for _, vol := range cluster.Volumes {
			if len(vol.AccessViaHosts) == 0 {
				// accessible by all servers
				return nil
			}
			if _, ok := vol.AccessViaHosts[url]; ok {
				continue servers
			}
		}
		ldr.Logger.Warnf("keepstore configured at %s does not have access to any volumes", url)
	}
	return nil
}

// Warn about AccessViaHosts entries that don't correspond to any of
// the listed keepstore services.
func (ldr *Loader) checkUnlistedKeepstores(cluster arvados.Cluster) error {
	for uuid, vol := range cluster.Volumes {
		if uuid == "SAMPLE" {
			continue
		}
		for url := range vol.AccessViaHosts {
			if _, ok := cluster.Services.Keepstore.InternalURLs[url]; !ok {
				ldr.Logger.Warnf("Volumes.%s.AccessViaHosts refers to nonexistent keepstore server %s", uuid, url)
			}
		}
	}
	return nil
}
