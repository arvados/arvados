// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"git.arvados.org/arvados.git/lib/config"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadosclient"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	"git.arvados.org/arvados.git/sdk/go/ctxlog"
	"git.arvados.org/arvados.git/sdk/go/keepclient"
	. "gopkg.in/check.v1"
)

var _ = Suite(&integrationSuite{})

type integrationSuite struct {
	engine string
	image  arvados.Collection
	input  arvados.Collection
	stdin  bytes.Buffer
	stdout bytes.Buffer
	stderr bytes.Buffer
	args   []string
	cr     arvados.ContainerRequest
	client *arvados.Client
	ac     *arvadosclient.ArvadosClient
	kc     *keepclient.KeepClient

	logCollection    arvados.Collection
	outputCollection arvados.Collection
	logFiles         map[string]string // filename => contents
}

func (s *integrationSuite) SetUpSuite(c *C) {
	_, err := exec.LookPath("docker")
	if err != nil {
		c.Skip("looks like docker is not installed")
	}

	arvadostest.StartKeep(2, true)

	out, err := exec.Command("docker", "load", "--input", arvadostest.BusyboxDockerImage(c)).CombinedOutput()
	c.Log(string(out))
	c.Assert(err, IsNil)
	out, err = exec.Command("arv-keepdocker", "--no-resume", "busybox:uclibc").Output()
	imageUUID := strings.TrimSpace(string(out))
	c.Logf("image uuid %s", imageUUID)
	if !c.Check(err, IsNil) {
		if err, ok := err.(*exec.ExitError); ok {
			c.Logf("%s", err.Stderr)
		}
		c.Fail()
	}
	err = arvados.NewClientFromEnv().RequestAndDecode(&s.image, "GET", "arvados/v1/collections/"+imageUUID, nil, nil)
	c.Assert(err, IsNil)
	c.Logf("image pdh %s", s.image.PortableDataHash)

	s.client = arvados.NewClientFromEnv()
	s.ac, err = arvadosclient.New(s.client)
	c.Assert(err, IsNil)
	s.kc = keepclient.New(s.ac)
	fs, err := s.input.FileSystem(s.client, s.kc)
	c.Assert(err, IsNil)
	f, err := fs.OpenFile("inputfile", os.O_CREATE|os.O_WRONLY, 0755)
	c.Assert(err, IsNil)
	_, err = f.Write([]byte("inputdata"))
	c.Assert(err, IsNil)
	err = f.Close()
	c.Assert(err, IsNil)
	s.input.ManifestText, err = fs.MarshalManifest(".")
	c.Assert(err, IsNil)
	err = s.client.RequestAndDecode(&s.input, "POST", "arvados/v1/collections", nil, map[string]interface{}{
		"ensure_unique_name": true,
		"collection": map[string]interface{}{
			"manifest_text": s.input.ManifestText,
		},
	})
	c.Assert(err, IsNil)
	c.Logf("input pdh %s", s.input.PortableDataHash)
}

func (s *integrationSuite) TearDownSuite(c *C) {
	os.Unsetenv("ARVADOS_KEEP_SERVICES")
	if s.client == nil {
		// didn't set up
		return
	}
	err := s.client.RequestAndDecode(nil, "POST", "database/reset", nil, nil)
	c.Check(err, IsNil)
}

func (s *integrationSuite) SetUpTest(c *C) {
	os.Unsetenv("ARVADOS_KEEP_SERVICES")
	s.engine = "docker"
	s.args = nil
	s.stdin = bytes.Buffer{}
	s.stdout = bytes.Buffer{}
	s.stderr = bytes.Buffer{}
	s.logCollection = arvados.Collection{}
	s.outputCollection = arvados.Collection{}
	s.logFiles = map[string]string{}
	s.cr = arvados.ContainerRequest{
		Priority:       1,
		State:          "Committed",
		OutputPath:     "/mnt/out",
		ContainerImage: s.image.PortableDataHash,
		Mounts: map[string]arvados.Mount{
			"/mnt/json": {
				Kind: "json",
				Content: []interface{}{
					"foo",
					map[string]string{"foo": "bar"},
					nil,
				},
			},
			"/mnt/in": {
				Kind:             "collection",
				PortableDataHash: s.input.PortableDataHash,
			},
			"/mnt/out": {
				Kind:     "tmp",
				Capacity: 1000,
			},
		},
		RuntimeConstraints: arvados.RuntimeConstraints{
			RAM:   128000000,
			VCPUs: 1,
			API:   true,
		},
	}
}

func (s *integrationSuite) setup(c *C) {
	err := s.client.RequestAndDecode(&s.cr, "POST", "arvados/v1/container_requests", nil, map[string]interface{}{"container_request": map[string]interface{}{
		"priority":            s.cr.Priority,
		"state":               s.cr.State,
		"command":             s.cr.Command,
		"output_path":         s.cr.OutputPath,
		"container_image":     s.cr.ContainerImage,
		"mounts":              s.cr.Mounts,
		"runtime_constraints": s.cr.RuntimeConstraints,
		"use_existing":        false,
	}})
	c.Assert(err, IsNil)
	c.Assert(s.cr.ContainerUUID, Not(Equals), "")
	err = s.client.RequestAndDecode(nil, "POST", "arvados/v1/containers/"+s.cr.ContainerUUID+"/lock", nil, nil)
	c.Assert(err, IsNil)
}

func (s *integrationSuite) TestRunTrivialContainerWithDocker(c *C) {
	s.engine = "docker"
	s.testRunTrivialContainer(c)
	c.Check(s.logFiles["crunch-run.txt"], Matches, `(?ms).*Using container runtime: docker Engine \d+\.\d+.*`)
}

func (s *integrationSuite) TestRunTrivialContainerWithSingularity(c *C) {
	s.engine = "singularity"
	s.testRunTrivialContainer(c)
	c.Check(s.logFiles["crunch-run.txt"], Matches, `(?ms).*Using container runtime: singularity.* version 3\.\d+.*`)
}

func (s *integrationSuite) TestRunTrivialContainerWithLocalKeepstore(c *C) {
	for _, trial := range []struct {
		logConfig           string
		matchGetReq         Checker
		matchPutReq         Checker
		matchStartupMessage Checker
	}{
		{"none", Not(Matches), Not(Matches), Not(Matches)},
		{"all", Matches, Matches, Matches},
		{"errors", Not(Matches), Not(Matches), Matches},
	} {
		c.Logf("=== testing with Containers.LocalKeepLogsToContainerLog: %q", trial.logConfig)
		s.SetUpTest(c)

		cfg, err := config.NewLoader(nil, ctxlog.TestLogger(c)).Load()
		c.Assert(err, IsNil)
		cluster, err := cfg.GetCluster("")
		c.Assert(err, IsNil)
		for uuid, volume := range cluster.Volumes {
			volume.AccessViaHosts = nil
			volume.Replication = 2
			cluster.Volumes[uuid] = volume
		}
		cluster.Containers.LocalKeepLogsToContainerLog = trial.logConfig

		s.stdin.Reset()
		err = json.NewEncoder(&s.stdin).Encode(ConfigData{
			Env:         nil,
			KeepBuffers: 1,
			Cluster:     cluster,
		})
		c.Assert(err, IsNil)

		s.engine = "docker"
		s.testRunTrivialContainer(c)

		log, logExists := s.logFiles["keepstore.txt"]
		if trial.logConfig == "none" {
			c.Check(logExists, Equals, false)
		} else {
			c.Check(log, trial.matchGetReq, `(?ms).*"reqMethod":"GET".*`)
			c.Check(log, trial.matchPutReq, `(?ms).*"reqMethod":"PUT".*,"reqPath":"0e3bcff26d51c895a60ea0d4585e134d".*`)
		}
	}

	// Check that (1) config is loaded from $ARVADOS_CONFIG when
	// not provided on stdin and (2) if a local keepstore is not
	// started, crunch-run.txt explains why not.
	s.SetUpTest(c)
	s.stdin.Reset()
	s.testRunTrivialContainer(c)
	c.Check(s.logFiles["crunch-run.txt"], Matches, `(?ms).*not starting a local keepstore process because a volume \(zzzzz-nyw5e-000000000000000\) uses AccessViaHosts\n.*`)

	// Check that config read errors are logged
	s.SetUpTest(c)
	s.args = []string{"-config", c.MkDir() + "/config-error.yaml"}
	s.stdin.Reset()
	s.testRunTrivialContainer(c)
	c.Check(s.logFiles["crunch-run.txt"], Matches, `(?ms).*could not load config file \Q`+s.args[1]+`\E:.* no such file or directory\n.*`)

	s.SetUpTest(c)
	s.args = []string{"-config", c.MkDir() + "/config-unreadable.yaml"}
	s.stdin.Reset()
	err := ioutil.WriteFile(s.args[1], []byte{}, 0)
	c.Check(err, IsNil)
	s.testRunTrivialContainer(c)
	c.Check(s.logFiles["crunch-run.txt"], Matches, `(?ms).*could not load config file \Q`+s.args[1]+`\E:.* permission denied\n.*`)

	s.SetUpTest(c)
	s.stdin.Reset()
	s.testRunTrivialContainer(c)
	c.Check(s.logFiles["crunch-run.txt"], Matches, `(?ms).*loaded config file \Q`+os.Getenv("ARVADOS_CONFIG")+`\E\n.*`)
}

func (s *integrationSuite) testRunTrivialContainer(c *C) {
	if err := exec.Command("which", s.engine).Run(); err != nil {
		c.Skip(fmt.Sprintf("%s: %s", s.engine, err))
	}
	if s.engine == "docker" && os.Getenv("ENABLE_DOCKER_TESTS") == "" {
		c.Skip("docker tests temporarily disabled if ENABLE_DOCKER_TESTS is not set, see https://dev.arvados.org/issues/15370#note-31")
	}
	s.cr.Command = []string{"sh", "-c", "cat /mnt/in/inputfile >/mnt/out/inputfile && cat /mnt/json >/mnt/out/json && ! touch /mnt/in/shouldbereadonly && mkdir /mnt/out/emptydir"}
	s.setup(c)

	args := []string{
		"-runtime-engine=" + s.engine,
		"-enable-memory-limit=false",
	}
	if s.stdin.Len() > 0 {
		args = append(args, "-stdin-config=true")
	}
	args = append(args, s.args...)
	args = append(args, s.cr.ContainerUUID)
	code := command{}.RunCommand("crunch-run", args, &s.stdin, io.MultiWriter(&s.stdout, os.Stderr), io.MultiWriter(&s.stderr, os.Stderr))
	c.Logf("\n===== stdout =====\n%s", s.stdout.String())
	c.Logf("\n===== stderr =====\n%s", s.stderr.String())
	c.Check(code, Equals, 0)
	err := s.client.RequestAndDecode(&s.cr, "GET", "arvados/v1/container_requests/"+s.cr.UUID, nil, nil)
	c.Assert(err, IsNil)
	c.Logf("Finished container request: %#v", s.cr)

	var log arvados.Collection
	err = s.client.RequestAndDecode(&log, "GET", "arvados/v1/collections/"+s.cr.LogUUID, nil, nil)
	c.Assert(err, IsNil)
	fs, err := log.FileSystem(s.client, s.kc)
	c.Assert(err, IsNil)
	if d, err := fs.Open("/"); c.Check(err, IsNil) {
		fis, err := d.Readdir(-1)
		c.Assert(err, IsNil)
		for _, fi := range fis {
			if fi.IsDir() {
				continue
			}
			f, err := fs.Open(fi.Name())
			c.Assert(err, IsNil)
			buf, err := ioutil.ReadAll(f)
			c.Assert(err, IsNil)
			c.Logf("\n===== %s =====\n%s", fi.Name(), buf)
			s.logFiles[fi.Name()] = string(buf)
		}
	}
	s.logCollection = log

	var output arvados.Collection
	err = s.client.RequestAndDecode(&output, "GET", "arvados/v1/collections/"+s.cr.OutputUUID, nil, nil)
	c.Assert(err, IsNil)
	fs, err = output.FileSystem(s.client, s.kc)
	c.Assert(err, IsNil)
	if f, err := fs.Open("inputfile"); c.Check(err, IsNil) {
		defer f.Close()
		buf, err := ioutil.ReadAll(f)
		c.Check(err, IsNil)
		c.Check(string(buf), Equals, "inputdata")
	}
	if f, err := fs.Open("json"); c.Check(err, IsNil) {
		defer f.Close()
		buf, err := ioutil.ReadAll(f)
		c.Check(err, IsNil)
		c.Check(string(buf), Equals, `["foo",{"foo":"bar"},null]`)
	}
	if fi, err := fs.Stat("emptydir"); c.Check(err, IsNil) {
		c.Check(fi.IsDir(), Equals, true)
	}
	if d, err := fs.Open("emptydir"); c.Check(err, IsNil) {
		defer d.Close()
		fis, err := d.Readdir(-1)
		c.Assert(err, IsNil)
		// crunch-run still saves a ".keep" file to preserve
		// empty dirs even though that shouldn't be
		// necessary. Ideally we would do:
		// c.Check(fis, HasLen, 0)
		for _, fi := range fis {
			c.Check(fi.Name(), Equals, ".keep")
		}
	}
	s.outputCollection = output
}
