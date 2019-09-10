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

	"git.curoverse.com/arvados.git/sdk/go/arvados"
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
	if os.IsNotExist(err) && (ldr.KeepstorePath == defaultKeepstoreConfigPath) {
		return nil
	} else if err != nil {
		return err
	}

	cluster, err := cfg.GetCluster("")
	if err != nil {
		return err
	}

	myURL := arvados.URL{Scheme: "http"}
	if oc.TLSCertificateFile != nil && oc.TLSKeyFile != nil {
		myURL.Scheme = "https"
	}

	if v := oc.Debug; v == nil {
	} else if *v && cluster.SystemLogs.LogLevel != "debug" {
		cluster.SystemLogs.LogLevel = "debug"
	} else if !*v && cluster.SystemLogs.LogLevel != "info" {
		cluster.SystemLogs.LogLevel = "info"
	}

	if v := oc.TLSCertificateFile; v != nil && "file://"+*v != cluster.TLS.Certificate {
		cluster.TLS.Certificate = "file://" + *v
	}
	if v := oc.TLSKeyFile; v != nil && "file://"+*v != cluster.TLS.Key {
		cluster.TLS.Key = "file://" + *v
	}
	if v := oc.Listen; v != nil {
		if _, ok := cluster.Services.Keepstore.InternalURLs[arvados.URL{Scheme: myURL.Scheme, Host: *v}]; ok {
			// already listed
			myURL.Host = *v
		} else if len(*v) > 1 && (*v)[0] == ':' {
			myURL.Host = net.JoinHostPort(hostname, (*v)[1:])
			cluster.Services.Keepstore.InternalURLs[myURL] = arvados.ServiceInstance{}
		} else {
			return fmt.Errorf("unable to migrate Listen value %q from legacy keepstore config file -- remove after configuring Services.Keepstore.InternalURLs.", *v)
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

	if v := oc.LogFormat; v != nil && *v != cluster.SystemLogs.Format {
		cluster.SystemLogs.Format = *v
	}
	if v := oc.MaxBuffers; v != nil && *v != cluster.API.MaxKeepBlockBuffers {
		cluster.API.MaxKeepBlockBuffers = *v
	}
	if v := oc.MaxRequests; v != nil && *v != cluster.API.MaxConcurrentRequests {
		cluster.API.MaxConcurrentRequests = *v
	}
	if v := oc.BlobSignatureTTL; v != nil && *v != cluster.Collections.BlobSigningTTL {
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
	if v := oc.RequireSignatures; v != nil && *v != cluster.Collections.BlobSigning {
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
	if v := oc.EnableDelete; v != nil && *v != cluster.Collections.BlobTrash {
		cluster.Collections.BlobTrash = *v
	}
	if v := oc.TrashLifetime; v != nil && *v != cluster.Collections.BlobTrashLifetime {
		cluster.Collections.BlobTrashLifetime = *v
	}
	if v := oc.TrashCheckInterval; v != nil && *v != cluster.Collections.BlobTrashCheckInterval {
		cluster.Collections.BlobTrashCheckInterval = *v
	}
	if v := oc.TrashWorkers; v != nil && *v != cluster.Collections.BlobReplicateConcurrency {
		cluster.Collections.BlobTrashConcurrency = *v
	}
	if v := oc.EmptyTrashWorkers; v != nil && *v != cluster.Collections.BlobReplicateConcurrency {
		cluster.Collections.BlobDeleteConcurrency = *v
	}
	if v := oc.PullWorkers; v != nil && *v != cluster.Collections.BlobReplicateConcurrency {
		cluster.Collections.BlobReplicateConcurrency = *v
	}
	if v := oc.Volumes; v == nil {
		ldr.Logger.Warn("no volumes in legacy config; discovering local directory volumes")
		err := ldr.discoverLocalVolumes(cluster, oc.DiscoverVolumesFromMountsFile, myURL)
		if err != nil {
			return fmt.Errorf("error discovering local directory volumes: %s", err)
		}
	} else {
		for i, oldvol := range *v {
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
				// Remove the old entry. It will be
				// added back below, possibly with a
				// new UUID.
				delete(cluster.Volumes, oldUUID)
			} else {
				var params interface{}
				switch oldvol.Type {
				case "S3":
					accesskeydata, err := ioutil.ReadFile(oldvol.AccessKeyFile)
					if err != nil && oldvol.AccessKeyFile != "" {
						return fmt.Errorf("error reading AccessKeyFile: %s", err)
					}
					secretkeydata, err := ioutil.ReadFile(oldvol.SecretKeyFile)
					if err != nil && oldvol.SecretKeyFile != "" {
						return fmt.Errorf("error reading SecretKeyFile: %s", err)
					}
					newvol = arvados.Volume{
						Driver:         "S3",
						ReadOnly:       oldvol.ReadOnly,
						Replication:    oldvol.S3Replication,
						StorageClasses: array2boolmap(oldvol.StorageClasses),
					}
					params = arvados.S3VolumeDriverParameters{
						AccessKey:          string(bytes.TrimSpace(accesskeydata)),
						SecretKey:          string(bytes.TrimSpace(secretkeydata)),
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
						return fmt.Errorf("error reading StorageAccountKeyFile: %s", err)
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
					return fmt.Errorf("unsupported volume type %q", oldvol.Type)
				}
				dp, err := json.Marshal(params)
				if err != nil {
					return err
				}
				newvol.DriverParameters = json.RawMessage(dp)
				if newvol.Replication < 1 {
					newvol.Replication = 1
				}
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
	}

	cfg.Clusters[cluster.ClusterID] = *cluster
	return nil
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
				if _, ok := newvol.AccessViaHosts[myURL]; ok {
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
			url := arvados.URL{
				Scheme: "http",
				Host:   net.JoinHostPort(ks.ServiceHost, strconv.Itoa(ks.ServicePort)),
			}
			if ks.ServiceSSLFlag {
				url.Scheme = "https"
			}
			return ks.UUID, url, nil
		} else {
			tried = append(tried, fmt.Sprintf("%s:%d", ks.ServiceHost, ks.ServicePort))
		}
	}
	err = fmt.Errorf("listen address %q does not match any of the non-proxy keep_services entries %q", listen, tried)
	return
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
