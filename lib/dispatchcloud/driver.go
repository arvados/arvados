// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"fmt"
	"time"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/lib/cloud/azure"
	"git.curoverse.com/arvados.git/lib/cloud/ec2"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

var drivers = map[string]cloud.Driver{
	"azure": azure.Driver,
	"ec2":   ec2.Driver,
}

func newInstanceSet(cluster *arvados.Cluster, setID cloud.InstanceSetID, logger logrus.FieldLogger) (cloud.InstanceSet, error) {
	driver, ok := drivers[cluster.Containers.CloudVMs.Driver]
	if !ok {
		return nil, fmt.Errorf("unsupported cloud driver %q", cluster.Containers.CloudVMs.Driver)
	}
	is, err := driver.InstanceSet(cluster.Containers.CloudVMs.DriverParameters, setID, logger)
	if maxops := cluster.Containers.CloudVMs.MaxCloudOpsPerSecond; maxops > 0 {
		is = &rateLimitedInstanceSet{
			InstanceSet: is,
			ticker:      time.NewTicker(time.Second / time.Duration(maxops)),
		}
	}
	return is, err
}

type rateLimitedInstanceSet struct {
	cloud.InstanceSet
	ticker *time.Ticker
}

func (is rateLimitedInstanceSet) Create(it arvados.InstanceType, image cloud.ImageID, tags cloud.InstanceTags, init cloud.InitCommand, pk ssh.PublicKey) (cloud.Instance, error) {
	<-is.ticker.C
	inst, err := is.InstanceSet.Create(it, image, tags, init, pk)
	return &rateLimitedInstance{inst, is.ticker}, err
}

type rateLimitedInstance struct {
	cloud.Instance
	ticker *time.Ticker
}

func (inst *rateLimitedInstance) Destroy() error {
	<-inst.ticker.C
	return inst.Instance.Destroy()
}
