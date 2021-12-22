// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	check "gopkg.in/check.v1"
)

type KeepstoreMigrationSuite struct {
	hostname string // blank = use test system's hostname
	ksByPort map[int]arvados.KeepService
}

var _ = check.Suite(&KeepstoreMigrationSuite{})

func (s *KeepstoreMigrationSuite) SetUpSuite(c *check.C) {
	os.Setenv("ARVADOS_API_HOST", os.Getenv("ARVADOS_TEST_API_HOST"))
	os.Setenv("ARVADOS_API_HOST_INSECURE", "1")
	os.Setenv("ARVADOS_API_TOKEN", arvadostest.AdminToken)

	// We don't need the keepstore servers, but we do need
	// keep_services listings that point to localhost, rather than
	// the apiserver fixtures that point to fictional hosts
	// keep*.zzzzz.arvadosapi.com.

	client := arvados.NewClientFromEnv()

	// Delete existing non-proxy listings.
	var svcList arvados.KeepServiceList
	err := client.RequestAndDecode(&svcList, "GET", "arvados/v1/keep_services", nil, nil)
	c.Assert(err, check.IsNil)
	for _, ks := range svcList.Items {
		if ks.ServiceType != "proxy" {
			err = client.RequestAndDecode(new(struct{}), "DELETE", "arvados/v1/keep_services/"+ks.UUID, nil, nil)
			c.Assert(err, check.IsNil)
		}
	}
	// Add new fake listings.
	s.ksByPort = map[int]arvados.KeepService{}
	for _, port := range []int{25107, 25108} {
		var ks arvados.KeepService
		err = client.RequestAndDecode(&ks, "POST", "arvados/v1/keep_services", nil, map[string]interface{}{
			"keep_service": map[string]interface{}{
				"service_type": "disk",
				"service_host": "localhost",
				"service_port": port,
			},
		})
		c.Assert(err, check.IsNil)
		s.ksByPort[port] = ks
	}
}

func (s *KeepstoreMigrationSuite) checkEquivalentWithKeepstoreConfig(c *check.C, keepstoreconfig, clusterconfig, expectedconfig string) {
	keepstorefile, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(keepstorefile.Name())
	_, err = io.WriteString(keepstorefile, keepstoreconfig)
	c.Assert(err, check.IsNil)
	err = keepstorefile.Close()
	c.Assert(err, check.IsNil)

	gotldr := testLoader(c, clusterconfig, nil)
	gotldr.KeepstorePath = keepstorefile.Name()
	expectedldr := testLoader(c, expectedconfig, nil)
	checkEquivalentLoaders(c, gotldr, expectedldr)
}

func (s *KeepstoreMigrationSuite) TestDeprecatedKeepstoreConfig(c *check.C) {
	keyfile, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(keyfile.Name())
	io.WriteString(keyfile, "blobsigningkey\n")

	hostname, err := os.Hostname()
	c.Assert(err, check.IsNil)

	s.checkEquivalentWithKeepstoreConfig(c, `
Listen: ":25107"
Debug: true
LogFormat: text
MaxBuffers: 1234
MaxRequests: 2345
BlobSignatureTTL: 123m
BlobSigningKeyFile: `+keyfile.Name()+`
Volumes:
- Type: Directory
  Root: /tmp
`, `
Clusters:
  z1111:
    SystemRootToken: `+arvadostest.AdminToken+`
    TLS: {Insecure: true}
    Services:
      Controller:
        ExternalURL: "https://`+os.Getenv("ARVADOS_API_HOST")+`/"
`, `
Clusters:
  z1111:
    SystemRootToken: `+arvadostest.AdminToken+`
    TLS: {Insecure: true}
    Services:
      Keepstore:
        InternalURLs:
          "http://`+hostname+`:25107": {Rendezvous: `+s.ksByPort[25107].UUID[12:]+`}
      Controller:
        ExternalURL: "https://`+os.Getenv("ARVADOS_API_HOST")+`/"
    SystemLogs:
      Format: text
      LogLevel: debug
    API:
      MaxKeepBlobBuffers: 1234
      MaxConcurrentRequests: 2345
    Collections:
      BlobSigningTTL: 123m
      BlobSigningKey: blobsigningkey
    Volumes:
      z1111-nyw5e-`+s.ksByPort[25107].UUID[12:]+`:
        AccessViaHosts:
          "http://`+hostname+`:25107":
            ReadOnly: false
        Driver: Directory
        DriverParameters:
          Root: /tmp
          Serialize: false
        ReadOnly: false
        Replication: 1
        StorageClasses: {}
`)
}

func (s *KeepstoreMigrationSuite) TestDiscoverLocalVolumes(c *check.C) {
	tmpd, err := ioutil.TempDir("", "")
	c.Assert(err, check.IsNil)
	defer os.RemoveAll(tmpd)
	err = os.Mkdir(tmpd+"/keep", 0777)
	c.Assert(err, check.IsNil)

	tmpf, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(tmpf.Name())

	// read/write
	_, err = fmt.Fprintf(tmpf, "/dev/xvdb %s ext4 rw,noexec 0 0\n", tmpd)
	c.Assert(err, check.IsNil)

	s.testDeprecatedVolume(c, "DiscoverVolumesFromMountsFile: "+tmpf.Name(), arvados.Volume{
		Driver:      "Directory",
		ReadOnly:    false,
		Replication: 1,
	}, &arvados.DirectoryVolumeDriverParameters{
		Root:      tmpd + "/keep",
		Serialize: false,
	}, &arvados.DirectoryVolumeDriverParameters{})

	// read-only
	tmpf.Seek(0, os.SEEK_SET)
	tmpf.Truncate(0)
	_, err = fmt.Fprintf(tmpf, "/dev/xvdb %s ext4 ro,noexec 0 0\n", tmpd)
	c.Assert(err, check.IsNil)

	s.testDeprecatedVolume(c, "DiscoverVolumesFromMountsFile: "+tmpf.Name(), arvados.Volume{
		Driver:      "Directory",
		ReadOnly:    true,
		Replication: 1,
	}, &arvados.DirectoryVolumeDriverParameters{
		Root:      tmpd + "/keep",
		Serialize: false,
	}, &arvados.DirectoryVolumeDriverParameters{})
}

func (s *KeepstoreMigrationSuite) TestDeprecatedVolumes(c *check.C) {
	accesskeyfile, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(accesskeyfile.Name())
	io.WriteString(accesskeyfile, "accesskeydata\n")

	secretkeyfile, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(secretkeyfile.Name())
	io.WriteString(secretkeyfile, "secretkeydata\n")

	// s3, empty/default
	s.testDeprecatedVolume(c, `
Volumes:
- Type: S3
`, arvados.Volume{
		Driver:      "S3",
		Replication: 1,
	}, &arvados.S3VolumeDriverParameters{}, &arvados.S3VolumeDriverParameters{})

	// s3, fully configured
	s.testDeprecatedVolume(c, `
Volumes:
- Type: S3
  AccessKeyFile: `+accesskeyfile.Name()+`
  SecretKeyFile: `+secretkeyfile.Name()+`
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: testbucket
  LocationConstraint: true
  IndexPageSize: 1234
  S3Replication: 4
  ConnectTimeout: 3m
  ReadTimeout: 4m
  RaceWindow: 5m
  UnsafeDelete: true
`, arvados.Volume{
		Driver:      "S3",
		Replication: 4,
	}, &arvados.S3VolumeDriverParameters{
		AccessKeyID:        "accesskeydata",
		SecretAccessKey:    "secretkeydata",
		Endpoint:           "https://storage.googleapis.com",
		Region:             "us-east-1z",
		Bucket:             "testbucket",
		LocationConstraint: true,
		IndexPageSize:      1234,
		ConnectTimeout:     arvados.Duration(time.Minute * 3),
		ReadTimeout:        arvados.Duration(time.Minute * 4),
		RaceWindow:         arvados.Duration(time.Minute * 5),
		UnsafeDelete:       true,
	}, &arvados.S3VolumeDriverParameters{})

	// azure, empty/default
	s.testDeprecatedVolume(c, `
Volumes:
- Type: Azure
`, arvados.Volume{
		Driver:      "Azure",
		Replication: 1,
	}, &arvados.AzureVolumeDriverParameters{}, &arvados.AzureVolumeDriverParameters{})

	// azure, fully configured
	s.testDeprecatedVolume(c, `
Volumes:
- Type: Azure
  ReadOnly: true
  StorageAccountName: storageacctname
  StorageAccountKeyFile: `+secretkeyfile.Name()+`
  StorageBaseURL: https://example.example/
  ContainerName: testctr
  LocationConstraint: true
  AzureReplication: 4
  RequestTimeout: 3m
  ListBlobsRetryDelay: 4m
  ListBlobsMaxAttempts: 5
`, arvados.Volume{
		Driver:      "Azure",
		ReadOnly:    true,
		Replication: 4,
	}, &arvados.AzureVolumeDriverParameters{
		StorageAccountName:   "storageacctname",
		StorageAccountKey:    "secretkeydata",
		StorageBaseURL:       "https://example.example/",
		ContainerName:        "testctr",
		RequestTimeout:       arvados.Duration(time.Minute * 3),
		ListBlobsRetryDelay:  arvados.Duration(time.Minute * 4),
		ListBlobsMaxAttempts: 5,
	}, &arvados.AzureVolumeDriverParameters{})

	// directory, empty/default
	s.testDeprecatedVolume(c, `
Volumes:
- Type: Directory
  Root: /tmp/xyzzy
`, arvados.Volume{
		Driver:      "Directory",
		Replication: 1,
	}, &arvados.DirectoryVolumeDriverParameters{
		Root: "/tmp/xyzzy",
	}, &arvados.DirectoryVolumeDriverParameters{})

	// directory, fully configured
	s.testDeprecatedVolume(c, `
Volumes:
- Type: Directory
  ReadOnly: true
  Root: /tmp/xyzzy
  DirectoryReplication: 4
  Serialize: true
`, arvados.Volume{
		Driver:      "Directory",
		ReadOnly:    true,
		Replication: 4,
	}, &arvados.DirectoryVolumeDriverParameters{
		Root:      "/tmp/xyzzy",
		Serialize: true,
	}, &arvados.DirectoryVolumeDriverParameters{})
}

func (s *KeepstoreMigrationSuite) testDeprecatedVolume(c *check.C, oldconfigdata string, expectvol arvados.Volume, expectparams interface{}, paramsdst interface{}) {
	hostname := s.hostname
	if hostname == "" {
		h, err := os.Hostname()
		c.Assert(err, check.IsNil)
		hostname = h
	}

	oldconfig, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(oldconfig.Name())
	io.WriteString(oldconfig, "Listen: :12345\n"+oldconfigdata)
	if !strings.Contains(oldconfigdata, "DiscoverVolumesFromMountsFile") {
		// Prevent tests from looking at the real /proc/mounts on the test host.
		io.WriteString(oldconfig, "\nDiscoverVolumesFromMountsFile: /dev/null\n")
	}

	ldr := testLoader(c, "Clusters: {z1111: {}}", nil)
	ldr.KeepstorePath = oldconfig.Name()
	cfg, err := ldr.Load()
	c.Assert(err, check.IsNil)
	cc := cfg.Clusters["z1111"]
	c.Check(cc.Volumes, check.HasLen, 1)
	for uuid, v := range cc.Volumes {
		c.Check(uuid, check.HasLen, 27)
		c.Check(v.Driver, check.Equals, expectvol.Driver)
		c.Check(v.Replication, check.Equals, expectvol.Replication)

		avh, ok := v.AccessViaHosts[arvados.URL{Scheme: "http", Host: hostname + ":12345", Path: "/"}]
		c.Check(ok, check.Equals, true)
		c.Check(avh.ReadOnly, check.Equals, expectvol.ReadOnly)

		err := json.Unmarshal(v.DriverParameters, paramsdst)
		c.Check(err, check.IsNil)
		c.Check(paramsdst, check.DeepEquals, expectparams)
	}
}

// How we handle a volume from a legacy keepstore config file depends
// on whether it's writable, whether a volume using the same cloud
// backend already exists in the cluster config, and (if so) whether
// it already has an AccessViaHosts entry for this host.
//
// In all cases, we should end up with an AccessViaHosts entry for
// this host, to indicate that the current host's volumes have been
// migrated.

// Same backend already referenced in cluster config, this host
// already listed in AccessViaHosts --> no change, except possibly
// updating the ReadOnly flag on the AccessViaHosts entry.
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_AlreadyMigrated(c *check.C) {
	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :12345
Volumes:
- Type: S3
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: alreadymigrated
  S3Replication: 3
`)
	checkEqualYAML(c, after, before)
}

// Writable volume, same cloud backend already referenced in cluster
// config --> change UUID to match this keepstore's UUID.
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_UpdateUUID(c *check.C) {
	port, expectUUID := s.getTestKeepstorePortAndMatchingVolumeUUID(c)

	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :`+strconv.Itoa(port)+`
Volumes:
- Type: S3
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: readonlyonother
  S3Replication: 3
`)
	c.Check(after, check.HasLen, len(before))
	newuuids := s.findAddedVolumes(c, before, after, 1)
	newvol := after[newuuids[0]]

	var params arvados.S3VolumeDriverParameters
	json.Unmarshal(newvol.DriverParameters, &params)
	c.Check(params.Bucket, check.Equals, "readonlyonother")
	c.Check(newuuids[0], check.Equals, expectUUID)
}

// Writable volume, same cloud backend not yet referenced --> add a
// new volume, with UUID to match this keepstore's UUID.
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_AddCloudVolume(c *check.C) {
	port, expectUUID := s.getTestKeepstorePortAndMatchingVolumeUUID(c)

	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :`+strconv.Itoa(port)+`
Volumes:
- Type: S3
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: bucket-to-migrate
  S3Replication: 3
`)
	newuuids := s.findAddedVolumes(c, before, after, 1)
	newvol := after[newuuids[0]]

	var params arvados.S3VolumeDriverParameters
	json.Unmarshal(newvol.DriverParameters, &params)
	c.Check(params.Bucket, check.Equals, "bucket-to-migrate")
	c.Check(newvol.Replication, check.Equals, 3)

	c.Check(newuuids[0], check.Equals, expectUUID)
}

// Writable volume, same filesystem backend already referenced in
// cluster config, but this host isn't in AccessViaHosts --> add a new
// volume, with UUID to match this keepstore's UUID (filesystem-backed
// volumes are assumed to be different on different hosts, even if
// paths are the same).
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_AddLocalVolume(c *check.C) {
	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :12345
Volumes:
- Type: Directory
  Root: /data/sdd
  DirectoryReplication: 2
`)
	newuuids := s.findAddedVolumes(c, before, after, 1)
	newvol := after[newuuids[0]]

	var params arvados.DirectoryVolumeDriverParameters
	json.Unmarshal(newvol.DriverParameters, &params)
	c.Check(params.Root, check.Equals, "/data/sdd")
	c.Check(newvol.Replication, check.Equals, 2)
}

// Writable volume, same filesystem backend already referenced in
// cluster config, and this host is already listed in AccessViaHosts
// --> already migrated, don't change anything.
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_LocalVolumeAlreadyMigrated(c *check.C) {
	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :12345
Volumes:
- Type: Directory
  Root: /data/sde
  DirectoryReplication: 2
`)
	checkEqualYAML(c, after, before)
}

// Multiple writable cloud-backed volumes --> one of them will get a
// UUID matching this keepstore's UUID.
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_AddMultipleCloudVolumes(c *check.C) {
	port, expectUUID := s.getTestKeepstorePortAndMatchingVolumeUUID(c)

	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :`+strconv.Itoa(port)+`
Volumes:
- Type: S3
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: first-bucket-to-migrate
  S3Replication: 3
- Type: S3
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: second-bucket-to-migrate
  S3Replication: 3
`)
	newuuids := s.findAddedVolumes(c, before, after, 2)
	// Sort by bucket name (so "first" comes before "second")
	params := map[string]arvados.S3VolumeDriverParameters{}
	for _, uuid := range newuuids {
		var p arvados.S3VolumeDriverParameters
		json.Unmarshal(after[uuid].DriverParameters, &p)
		params[uuid] = p
	}
	sort.Slice(newuuids, func(i, j int) bool { return params[newuuids[i]].Bucket < params[newuuids[j]].Bucket })
	newvol0, newvol1 := after[newuuids[0]], after[newuuids[1]]
	params0, params1 := params[newuuids[0]], params[newuuids[1]]

	c.Check(params0.Bucket, check.Equals, "first-bucket-to-migrate")
	c.Check(newvol0.Replication, check.Equals, 3)

	c.Check(params1.Bucket, check.Equals, "second-bucket-to-migrate")
	c.Check(newvol1.Replication, check.Equals, 3)

	// Don't care which one gets the special UUID
	if newuuids[0] != expectUUID {
		c.Check(newuuids[1], check.Equals, expectUUID)
	}
}

// Non-writable volume, same cloud backend already referenced in
// cluster config --> add this host to AccessViaHosts with
// ReadOnly==true
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_UpdateWithReadOnly(c *check.C) {
	port, _ := s.getTestKeepstorePortAndMatchingVolumeUUID(c)
	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :`+strconv.Itoa(port)+`
Volumes:
- Type: S3
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: readonlyonother
  S3Replication: 3
  ReadOnly: true
`)
	hostname, err := os.Hostname()
	c.Assert(err, check.IsNil)
	url := arvados.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", hostname, port),
		Path:   "/",
	}
	_, ok := before["zzzzz-nyw5e-readonlyonother"].AccessViaHosts[url]
	c.Check(ok, check.Equals, false)
	_, ok = after["zzzzz-nyw5e-readonlyonother"].AccessViaHosts[url]
	c.Check(ok, check.Equals, true)
}

// Writable volume, same cloud backend already writable by another
// keepstore server --> add this host to AccessViaHosts with
// ReadOnly==true
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_UpdateAlreadyWritable(c *check.C) {
	port, _ := s.getTestKeepstorePortAndMatchingVolumeUUID(c)
	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :`+strconv.Itoa(port)+`
Volumes:
- Type: S3
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: writableonother
  S3Replication: 3
  ReadOnly: false
`)
	hostname, err := os.Hostname()
	c.Assert(err, check.IsNil)
	url := arvados.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", hostname, port),
		Path:   "/",
	}
	_, ok := before["zzzzz-nyw5e-writableonother"].AccessViaHosts[url]
	c.Check(ok, check.Equals, false)
	_, ok = after["zzzzz-nyw5e-writableonother"].AccessViaHosts[url]
	c.Check(ok, check.Equals, true)
}

// Non-writable volume, same cloud backend not already referenced in
// cluster config --> assign a new random volume UUID.
func (s *KeepstoreMigrationSuite) TestIncrementalVolumeMigration_AddReadOnly(c *check.C) {
	port, _ := s.getTestKeepstorePortAndMatchingVolumeUUID(c)
	before, after := s.loadWithKeepstoreConfig(c, `
Listen: :`+strconv.Itoa(port)+`
Volumes:
- Type: S3
  Endpoint: https://storage.googleapis.com
  Region: us-east-1z
  Bucket: differentbucket
  S3Replication: 3
`)
	newuuids := s.findAddedVolumes(c, before, after, 1)
	newvol := after[newuuids[0]]

	var params arvados.S3VolumeDriverParameters
	json.Unmarshal(newvol.DriverParameters, &params)
	c.Check(params.Bucket, check.Equals, "differentbucket")

	hostname, err := os.Hostname()
	c.Assert(err, check.IsNil)
	_, ok := newvol.AccessViaHosts[arvados.URL{Scheme: "http", Host: fmt.Sprintf("%s:%d", hostname, port), Path: "/"}]
	c.Check(ok, check.Equals, true)
}

// Ensure logs mention unmigrated servers.
func (s *KeepstoreMigrationSuite) TestPendingKeepstoreMigrations(c *check.C) {
	client := arvados.NewClientFromEnv()
	for _, host := range []string{"keep0", "keep1"} {
		err := client.RequestAndDecode(new(struct{}), "POST", "arvados/v1/keep_services", nil, map[string]interface{}{
			"keep_service": map[string]interface{}{
				"service_type": "disk",
				"service_host": host + ".zzzzz.example.com",
				"service_port": 25107,
			},
		})
		c.Assert(err, check.IsNil)
	}

	port, _ := s.getTestKeepstorePortAndMatchingVolumeUUID(c)
	logs := s.logsWithKeepstoreConfig(c, `
Listen: :`+strconv.Itoa(port)+`
Volumes:
- Type: S3
  Endpoint: https://storage.googleapis.com
  Bucket: foo
`)
	c.Check(logs, check.Matches, `(?ms).*you should remove the legacy keepstore config file.*`)
	c.Check(logs, check.Matches, `(?ms).*you should migrate the legacy keepstore configuration file on host keep1.zzzzz.example.com.*`)
	c.Check(logs, check.Not(check.Matches), `(?ms).*should migrate.*keep0.zzzzz.example.com.*`)
	c.Check(logs, check.Matches, `(?ms).*keepstore configured at http://keep2.zzzzz.example.com:25107/ does not have access to any volumes.*`)
	c.Check(logs, check.Matches, `(?ms).*Volumes.zzzzz-nyw5e-possconfigerror.AccessViaHosts refers to nonexistent keepstore server http://keep00.zzzzz.example.com:25107.*`)
}

const clusterConfigForKeepstoreMigrationTest = `
Clusters:
  zzzzz:
    SystemRootToken: ` + arvadostest.AdminToken + `
    Services:
      Keepstore:
        InternalURLs:
          "http://{{.hostname}}:12345": {}
          "http://keep0.zzzzz.example.com:25107": {}
          "http://keep2.zzzzz.example.com:25107": {}
      Controller:
        ExternalURL: "https://{{.controller}}"
    TLS:
      Insecure: true
    Volumes:

      zzzzz-nyw5e-alreadymigrated:
        AccessViaHosts:
          "http://{{.hostname}}:12345": {}
        Driver: S3
        DriverParameters:
          Endpoint: https://storage.googleapis.com
          Region: us-east-1z
          Bucket: alreadymigrated
        Replication: 3

      zzzzz-nyw5e-readonlyonother:
        AccessViaHosts:
          "http://keep0.zzzzz.example.com:25107": {ReadOnly: true}
        Driver: S3
        DriverParameters:
          Endpoint: https://storage.googleapis.com
          Region: us-east-1z
          Bucket: readonlyonother
        Replication: 3

      zzzzz-nyw5e-writableonother:
        AccessViaHosts:
          "http://keep0.zzzzz.example.com:25107": {}
        Driver: S3
        DriverParameters:
          Endpoint: https://storage.googleapis.com
          Region: us-east-1z
          Bucket: writableonother
        Replication: 3

      zzzzz-nyw5e-localfilesystem:
        AccessViaHosts:
          "http://keep0.zzzzz.example.com:25107": {}
        Driver: Directory
        DriverParameters:
          Root: /data/sdd
        Replication: 1

      zzzzz-nyw5e-localismigrated:
        AccessViaHosts:
          "http://{{.hostname}}:12345": {}
        Driver: Directory
        DriverParameters:
          Root: /data/sde
        Replication: 1

      zzzzz-nyw5e-possconfigerror:
        AccessViaHosts:
          "http://keep00.zzzzz.example.com:25107": {}
        Driver: Directory
        DriverParameters:
          Root: /data/sdf
        Replication: 1
`

// Determine the effect of combining the given legacy keepstore config
// YAML (just the "Volumes" entries of an old keepstore config file)
// with the example clusterConfigForKeepstoreMigrationTest config.
//
// Return two Volumes configs -- one without loading keepstoreYAML
// ("before") and one with ("after") -- for the caller to compare.
func (s *KeepstoreMigrationSuite) loadWithKeepstoreConfig(c *check.C, keepstoreYAML string) (before, after map[string]arvados.Volume) {
	ldr := testLoader(c, s.clusterConfigYAML(c), nil)
	cBefore, err := ldr.Load()
	c.Assert(err, check.IsNil)

	keepstoreconfig, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(keepstoreconfig.Name())
	io.WriteString(keepstoreconfig, keepstoreYAML)

	ldr = testLoader(c, s.clusterConfigYAML(c), nil)
	ldr.KeepstorePath = keepstoreconfig.Name()
	cAfter, err := ldr.Load()
	c.Assert(err, check.IsNil)

	return cBefore.Clusters["zzzzz"].Volumes, cAfter.Clusters["zzzzz"].Volumes
}

// Return the log messages emitted when loading keepstoreYAML along
// with clusterConfigForKeepstoreMigrationTest.
func (s *KeepstoreMigrationSuite) logsWithKeepstoreConfig(c *check.C, keepstoreYAML string) string {
	var logs bytes.Buffer

	keepstoreconfig, err := ioutil.TempFile("", "")
	c.Assert(err, check.IsNil)
	defer os.Remove(keepstoreconfig.Name())
	io.WriteString(keepstoreconfig, keepstoreYAML)

	ldr := testLoader(c, s.clusterConfigYAML(c), &logs)
	ldr.KeepstorePath = keepstoreconfig.Name()
	_, err = ldr.Load()
	c.Assert(err, check.IsNil)

	return logs.String()
}

func (s *KeepstoreMigrationSuite) clusterConfigYAML(c *check.C) string {
	hostname, err := os.Hostname()
	c.Assert(err, check.IsNil)

	tmpl := template.Must(template.New("config").Parse(clusterConfigForKeepstoreMigrationTest))

	var clusterconfigdata bytes.Buffer
	err = tmpl.Execute(&clusterconfigdata, map[string]interface{}{
		"hostname":   hostname,
		"controller": os.Getenv("ARVADOS_API_HOST"),
	})
	c.Assert(err, check.IsNil)

	return clusterconfigdata.String()
}

// Return the uuids of volumes that appear in "after" but not
// "before".
//
// Assert the returned slice has at least minAdded entries.
func (s *KeepstoreMigrationSuite) findAddedVolumes(c *check.C, before, after map[string]arvados.Volume, minAdded int) (uuids []string) {
	for uuid := range after {
		if _, ok := before[uuid]; !ok {
			uuids = append(uuids, uuid)
		}
	}
	if len(uuids) < minAdded {
		c.Assert(uuids, check.HasLen, minAdded)
	}
	return
}

func (s *KeepstoreMigrationSuite) getTestKeepstorePortAndMatchingVolumeUUID(c *check.C) (int, string) {
	for port, ks := range s.ksByPort {
		c.Assert(ks.UUID, check.HasLen, 27)
		return port, "zzzzz-nyw5e-" + ks.UUID[12:]
	}
	c.Fatal("s.ksByPort is empty")
	return 0, ""
}

func (s *KeepstoreMigrationSuite) TestKeepServiceIsMe(c *check.C) {
	for i, trial := range []struct {
		match       bool
		hostname    string
		listen      string
		serviceHost string
		servicePort int
	}{
		{true, "keep0", "keep0", "keep0", 80},
		{true, "keep0", "[::1]:http", "keep0", 80},
		{true, "keep0", "[::]:http", "keep0", 80},
		{true, "keep0", "keep0:25107", "keep0", 25107},
		{true, "keep0", ":25107", "keep0", 25107},
		{true, "keep0.domain", ":25107", "keep0.domain.example", 25107},
		{true, "keep0.domain.example", ":25107", "keep0.domain.example", 25107},
		{true, "keep0", ":25107", "keep0.domain.example", 25107},
		{true, "keep0", ":25107", "Keep0.domain.example", 25107},
		{true, "keep0", ":http", "keep0.domain.example", 80},
		{true, "keep0", ":25107", "localhost", 25107},
		{true, "keep0", ":25107", "::1", 25107},
		{false, "keep0", ":25107", "keep0", 1111},              // different port
		{false, "keep0", ":25107", "localhost", 1111},          // different port
		{false, "keep0", ":http", "keep0.domain.example", 443}, // different port
		{false, "keep0", ":bogussss", "keep0", 25107},          // unresolvable port
		{false, "keep0", ":25107", "keep1", 25107},             // different hostname
		{false, "keep1", ":25107", "keep10", 25107},            // different hostname (prefix, but not on a "." boundary)
	} {
		c.Check(keepServiceIsMe(arvados.KeepService{ServiceHost: trial.serviceHost, ServicePort: trial.servicePort}, trial.hostname, trial.listen), check.Equals, trial.match, check.Commentf("trial #%d: %#v", i, trial))
	}
}
