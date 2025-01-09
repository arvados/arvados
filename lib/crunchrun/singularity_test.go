// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
	"git.arvados.org/arvados.git/sdk/go/arvadostest"
	. "gopkg.in/check.v1"
)

var _ = Suite(&singularitySuite{})

type singularitySuite struct {
	executorSuite
}

func (s *singularitySuite) SetUpSuite(c *C) {
	_, err := exec.LookPath("singularity")
	if err != nil {
		c.Skip("looks like singularity is not installed")
	}
	s.newExecutor = func(c *C) {
		var err error
		s.executor, err = newSingularityExecutor(c.Logf)
		c.Assert(err, IsNil)
	}
}

func (s *singularitySuite) TearDownSuite(c *C) {
	if s.executor != nil {
		s.executor.Close()
	}
}

func (s *singularitySuite) TestEnableNetwork_Listen(c *C) {
	// With modern iptables, singularity (as of 4.2.1) cannot
	// enable networking when invoked by a regular user. Under
	// arvados-dispatch-cloud, crunch-run runs as root, so it's
	// OK. For testing, assuming tests are not running as root, we
	// use sudo -- but only if requested via environment variable.
	if os.Getuid() == 0 {
		// already root
	} else if os.Getenv("ARVADOS_TEST_PRIVESC") == "sudo" {
		c.Logf("ARVADOS_TEST_PRIVESC is 'sudo', invoking 'sudo singularity ...'")
		s.executor.(*singularityExecutor).sudo = true
	} else {
		c.Skip("test case needs to run singularity as root -- set ARVADOS_TEST_PRIVESC=sudo to enable this test")
	}
	s.executorSuite.TestEnableNetwork_Listen(c)
}

func (s *singularitySuite) TestInject(c *C) {
	path, err := exec.LookPath("nsenter")
	if err != nil || path != "/var/lib/arvados/bin/nsenter" {
		c.Skip("looks like /var/lib/arvados/bin/nsenter is not installed -- re-run `arvados-server install`?")
	}
	s.executorSuite.TestInject(c)
}

var _ = Suite(&singularityStubSuite{})

// singularityStubSuite tests don't really invoke singularity, so we
// can run them even if singularity is not installed.
type singularityStubSuite struct{}

func (s *singularityStubSuite) TestSingularityExecArgs(c *C) {
	e, err := newSingularityExecutor(c.Logf)
	c.Assert(err, IsNil)
	err = e.Create(containerSpec{
		WorkingDir:      "/WorkingDir",
		Env:             map[string]string{"FOO": "bar"},
		BindMounts:      map[string]bindmount{"/mnt": {HostPath: "/hostpath", ReadOnly: true}},
		EnableNetwork:   false,
		CUDADeviceCount: 3,
		VCPUs:           2,
		RAM:             12345678,
	})
	c.Check(err, IsNil)
	e.imageFilename = "/fake/image.sif"
	cmd := e.execCmd("./singularity")
	expectArgs := []string{"./singularity", "exec", "--containall", "--cleanenv", "--pwd=/WorkingDir", "--net", "--network=none", "--nv"}
	if cgroupSupport["cpu"] {
		expectArgs = append(expectArgs, "--cpus", "2")
	}
	if cgroupSupport["memory"] {
		expectArgs = append(expectArgs, "--memory", "12345678")
	}
	expectArgs = append(expectArgs, "--bind", "/hostpath:/mnt:ro", "/fake/image.sif")
	c.Check(cmd.Args, DeepEquals, expectArgs)
	c.Check(cmd.Env, DeepEquals, []string{
		"SINGULARITYENV_FOO=bar",
		"SINGULARITY_NO_EVAL=1",
		"XDG_RUNTIME_DIR=" + os.Getenv("XDG_RUNTIME_DIR"),
		"DBUS_SESSION_BUS_ADDRESS=" + os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
	})
}

// A race condition in previous versions (#20605) could result in
// empty collections being saved in the "auto-generated singularity
// images" project. Check that we remove/replace any such broken
// collections.
func (s *singularitySuite) TestRemoveEmptyImageCacheCollection(c *C) {
	arvMountDir := c.MkDir()
	arvMountCmd := exec.Command("arv-mount",
		"--foreground", "--read-write",
		"--storage-classes", "default",
		"--mount-by-pdh", "by_id", "--mount-by-id", "by_uuid",
		"--disable-event-listening",
		arvMountDir)
	err := arvMountCmd.Start()
	c.Assert(err, IsNil)
	defer exec.Command("arv-mount", "--unmount", arvMountDir).Run()

	dockerImageID := "sha256:388056c9a6838deea3792e8f00705b35b439cf57b3c9c2634fb4e95cfc896de6"
	containerClient := arvados.NewClientFromEnv()
	containerClient.AuthToken = arvadostest.ActiveTokenV2

	e := s.executor.(*singularityExecutor)

	// remove image already loaded by SetupTest
	os.Remove(e.tmpdir + "/image.tar")

	cacheGroup, err := e.getOrCreateProject(arvadostest.ActiveUserUUID, ".cache", containerClient)
	c.Assert(err, IsNil)
	imageGroup, err := e.getOrCreateProject(cacheGroup.UUID, "auto-generated singularity images", containerClient)
	c.Assert(err, IsNil)
	collectionName := fmt.Sprintf("singularity image for %v", dockerImageID)

	// Remove existing cache entry, if any.
	var cl arvados.CollectionList
	err = containerClient.RequestAndDecode(&cl,
		arvados.EndpointCollectionList.Method,
		arvados.EndpointCollectionList.Path,
		nil, arvados.ListOptions{Filters: []arvados.Filter{
			arvados.Filter{"owner_uuid", "=", imageGroup.UUID},
			arvados.Filter{"name", "=", collectionName},
		},
			Limit: 1})
	c.Assert(err, IsNil)
	if len(cl.Items) == 1 {
		containerClient.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+cl.Items[0].UUID, nil, nil)
	}

	// Create empty cache entry.
	var emptyCollection arvados.Collection
	exp := time.Now().Add(24 * 7 * 2 * time.Hour)
	err = containerClient.RequestAndDecode(&emptyCollection,
		arvados.EndpointCollectionCreate.Method,
		arvados.EndpointCollectionCreate.Path,
		nil, map[string]interface{}{
			"collection": map[string]string{
				"owner_uuid": imageGroup.UUID,
				"name":       collectionName,
				"trash_at":   exp.UTC().Format(time.RFC3339),
			},
			"ensure_unique_name": true,
		})
	c.Assert(err, IsNil)

	// LoadImage should detect and replace the empty cache entry.
	err = s.executor.LoadImage(dockerImageID, arvadostest.BusyboxDockerImage(c), arvados.Container{RuntimeUserUUID: arvadostest.ActiveUserUUID}, arvMountDir, containerClient)
	c.Check(err, IsNil)

	// New docker image should have been extracted to tmpdir.
	tarListing, err := exec.Command("tar", "tvf", e.tmpdir+"/image.tar").CombinedOutput()
	c.Check(err, IsNil)
	c.Check(string(tarListing), Matches, `(?ms).*/layer.tar.*`)

	// Empty collection should have been removed.
	err = containerClient.RequestAndDecode(nil, arvados.EndpointCollectionGet.Method, "arvados/v1/collections/"+emptyCollection.UUID, nil, nil)
	c.Check(err, ErrorMatches, `.*404.*`)

	// New cache collection should have been saved.
	err = containerClient.RequestAndDecode(&cl,
		arvados.EndpointCollectionList.Method,
		arvados.EndpointCollectionList.Path,
		nil, arvados.ListOptions{Filters: []arvados.Filter{
			arvados.Filter{"owner_uuid", "=", imageGroup.UUID},
			arvados.Filter{"name", "=", collectionName},
		},
			Limit: 1})
	c.Assert(err, IsNil)
	c.Assert(cl.Items, HasLen, 1)
	c.Check(cl.Items[0].PortableDataHash, Not(Equals), "d41d8cd98f00b204e9800998ecf8427e+0")
}
