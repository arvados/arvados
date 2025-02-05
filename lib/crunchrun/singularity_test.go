// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
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
	arvadostest.StartKeep(2, true)
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

func (s *singularitySuite) setupMount(c *C) (mountdir string) {
	mountdir = c.MkDir()
	cmd := exec.Command("arv-mount",
		"--foreground", "--read-write",
		"--storage-classes", "default",
		"--mount-by-pdh", "by_id", "--mount-by-id", "by_uuid",
		"--disable-event-listening",
		mountdir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	c.Assert(err, IsNil)
	return
}

func (s *singularitySuite) teardownMount(c *C, mountdir string) {
	exec.Command("arv-mount", "--unmount", mountdir).Run()
}

type singularitySuiteLoadTestSetup struct {
	containerClient   *arvados.Client
	imageCacheProject *arvados.Group
	dockerImageID     string
	collectionName    string
}

func (s *singularitySuite) setupLoadTest(c *C, e *singularityExecutor) (setup singularitySuiteLoadTestSetup) {
	// remove symlink and converted image already written by
	// (executorSuite)SetupTest
	os.Remove(e.tmpdir + "/image.tar")
	os.Remove(e.tmpdir + "/image.sif")

	setup.containerClient = arvados.NewClientFromEnv()
	setup.containerClient.AuthToken = arvadostest.ActiveTokenV2

	var err error
	setup.imageCacheProject, err = e.getImageCacheProject(arvadostest.ActiveUserUUID, setup.containerClient)
	c.Assert(err, IsNil)

	setup.dockerImageID = "sha256:388056c9a6838deea3792e8f00705b35b439cf57b3c9c2634fb4e95cfc896de6"
	setup.collectionName = fmt.Sprintf("singularity image for %s", setup.dockerImageID)

	// Remove existing cache entry, if any.
	var cl arvados.CollectionList
	err = setup.containerClient.RequestAndDecode(&cl,
		arvados.EndpointCollectionList.Method,
		arvados.EndpointCollectionList.Path,
		nil, arvados.ListOptions{Filters: []arvados.Filter{
			arvados.Filter{"owner_uuid", "=", setup.imageCacheProject.UUID},
			arvados.Filter{"name", "=", setup.collectionName},
		},
			Limit: 1})
	c.Assert(err, IsNil)
	if len(cl.Items) == 1 {
		setup.containerClient.RequestAndDecode(nil, "DELETE", "arvados/v1/collections/"+cl.Items[0].UUID, nil, nil)
	}

	return
}

func (s *singularitySuite) checkCacheCollectionExists(c *C, setup singularitySuiteLoadTestSetup) {
	var cl arvados.CollectionList
	err := setup.containerClient.RequestAndDecode(&cl,
		arvados.EndpointCollectionList.Method,
		arvados.EndpointCollectionList.Path,
		nil, arvados.ListOptions{Filters: []arvados.Filter{
			arvados.Filter{"owner_uuid", "=", setup.imageCacheProject.UUID},
			arvados.Filter{"name", "=", setup.collectionName},
		},
			Limit: 1})
	c.Assert(err, IsNil)
	if !c.Check(cl.Items, HasLen, 1) {
		return
	}
	c.Check(cl.Items[0].PortableDataHash, Not(Equals), "d41d8cd98f00b204e9800998ecf8427e+0")
}

func (s *singularitySuite) TestImageCache_New(c *C) {
	mountdir := s.setupMount(c)
	defer s.teardownMount(c, mountdir)
	e, err := newSingularityExecutor(c.Logf)
	c.Assert(err, IsNil)
	setup := s.setupLoadTest(c, e)
	err = e.LoadImage(setup.dockerImageID, arvadostest.BusyboxDockerImage(c), arvados.Container{RuntimeUserUUID: arvadostest.ActiveUserUUID}, mountdir, setup.containerClient)
	c.Check(err, IsNil)
	_, err = os.Stat(e.tmpdir + "/image.sif")
	c.Check(err, NotNil)
	c.Check(os.IsNotExist(err), Equals, true)
	s.checkCacheCollectionExists(c, setup)
}

func (s *singularitySuite) TestImageCache_SkipEmpty(c *C) {
	mountdir := s.setupMount(c)
	defer s.teardownMount(c, mountdir)
	e, err := newSingularityExecutor(c.Logf)
	c.Assert(err, IsNil)
	setup := s.setupLoadTest(c, e)

	var emptyCollection arvados.Collection
	exp := time.Now().Add(24 * 7 * 2 * time.Hour)
	err = setup.containerClient.RequestAndDecode(&emptyCollection,
		arvados.EndpointCollectionCreate.Method,
		arvados.EndpointCollectionCreate.Path,
		nil, map[string]interface{}{
			"collection": map[string]string{
				"owner_uuid": setup.imageCacheProject.UUID,
				"name":       setup.collectionName,
				"trash_at":   exp.UTC().Format(time.RFC3339),
			},
		})
	c.Assert(err, IsNil)

	err = e.LoadImage(setup.dockerImageID, arvadostest.BusyboxDockerImage(c), arvados.Container{RuntimeUserUUID: arvadostest.ActiveUserUUID}, mountdir, setup.containerClient)
	c.Check(err, IsNil)
	c.Check(e.imageFilename, Equals, e.tmpdir+"/image.sif")

	// tmpdir should contain symlink to docker image archive.
	tarListing, err := exec.Command("tar", "tvf", e.tmpdir+"/image.tar").CombinedOutput()
	c.Check(err, IsNil)
	c.Check(string(tarListing), Matches, `(?ms).*/layer.tar.*`)

	// converted singularity image should be non-empty.
	fi, err := os.Stat(e.imageFilename)
	if c.Check(err, IsNil) {
		c.Check(int(fi.Size()), Not(Equals), 0)
	}
}

func (s *singularitySuite) TestImageCache_Concurrency_1(c *C) {
	s.testImageCache(c, 1)
}

func (s *singularitySuite) TestImageCache_Concurrency_2(c *C) {
	s.testImageCache(c, 2)
}

func (s *singularitySuite) TestImageCache_Concurrency_10(c *C) {
	s.testImageCache(c, 10)
}

func (s *singularitySuite) testImageCache(c *C, concurrency int) {
	mountdirs := make([]string, concurrency)
	execs := make([]*singularityExecutor, concurrency)
	setups := make([]singularitySuiteLoadTestSetup, concurrency)
	for i := range execs {
		mountdirs[i] = s.setupMount(c)
		defer s.teardownMount(c, mountdirs[i])
		e, err := newSingularityExecutor(c.Logf)
		c.Assert(err, IsNil)
		defer e.Close()
		execs[i] = e
		setups[i] = s.setupLoadTest(c, e)
	}

	var wg sync.WaitGroup
	for i, e := range execs {
		i, e := i, e
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := e.LoadImage(setups[i].dockerImageID, arvadostest.BusyboxDockerImage(c), arvados.Container{RuntimeUserUUID: arvadostest.ActiveUserUUID}, mountdirs[i], setups[i].containerClient)
			c.Check(err, IsNil)
		}()
	}
	wg.Wait()

	for i, e := range execs {
		fusepath := strings.TrimPrefix(e.imageFilename, mountdirs[i])
		// imageFilename should be in the fuse mount, not
		// e.tmpdir.
		c.Check(fusepath, Not(Equals), execs[0].imageFilename)
		// Below fuse mountpoint, paths should all be equal.
		fusepath0 := strings.TrimPrefix(execs[0].imageFilename, mountdirs[0])
		c.Check(fusepath, Equals, fusepath0)
	}
}
