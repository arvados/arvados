// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"os/exec"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	. "gopkg.in/check.v1"
)

var _ = Suite(&dockerSuite{})

type dockerSuite struct {
	executorSuite
}

func (s *dockerSuite) SetUpSuite(c *C) {
	_, err := exec.LookPath("docker")
	if err != nil {
		c.Skip("looks like docker is not installed")
	}
	s.newExecutor = func(c *C) {
		exec.Command("docker", "rm", "zzzzz-zzzzz-zzzzzzzzzzzzzzz").Run()
		var err error
		s.executor, err = newDockerExecutor("zzzzz-zzzzz-zzzzzzzzzzzzzzz", c.Logf, time.Second/2)
		c.Assert(err, IsNil)
	}
}

var _ = Suite(&dockerStubSuite{})

// dockerStubSuite tests don't really connect to the docker service,
// so we can run them even if docker is not installed.
type dockerStubSuite struct{}

func (s *dockerStubSuite) TestDockerContainerConfig(c *C) {
	e, err := newDockerExecutor("zzzzz-zzzzz-zzzzzzzzzzzzzzz", c.Logf, time.Second/2)
	c.Assert(err, IsNil)
	cfg, hostCfg := e.config(containerSpec{
		VCPUs:           4,
		RAM:             123123123,
		WorkingDir:      "/WorkingDir",
		Env:             map[string]string{"FOO": "bar"},
		BindMounts:      map[string]bindmount{"/mnt": {HostPath: "/hostpath", ReadOnly: true}},
		EnableNetwork:   false,
		CUDADeviceCount: 3,
	})
	c.Check(cfg.WorkingDir, Equals, "/WorkingDir")
	c.Check(cfg.Env, DeepEquals, []string{"FOO=bar"})
	c.Check(hostCfg.NetworkMode, Equals, dockercontainer.NetworkMode("none"))
	c.Check(hostCfg.Resources.NanoCPUs, Equals, int64(4000000000))
	c.Check(hostCfg.Resources.Memory, Equals, int64(123123123))
	c.Check(hostCfg.Resources.MemorySwap, Equals, int64(123123123))
	c.Check(hostCfg.Resources.KernelMemory, Equals, int64(123123123))
	c.Check(hostCfg.Resources.DeviceRequests, DeepEquals, []dockercontainer.DeviceRequest{{
		Driver:       "nvidia",
		Count:        3,
		Capabilities: [][]string{{"gpu", "nvidia", "compute", "utility"}},
	}})
}
