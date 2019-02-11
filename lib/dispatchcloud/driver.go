// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"fmt"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/lib/cloud/azure"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"github.com/sirupsen/logrus"
)

var drivers = map[string]cloud.Driver{
	"azure": cloud.DriverFunc(azure.NewAzureInstanceSet),
}

func newInstanceSet(cluster *arvados.Cluster, setID cloud.InstanceSetID, logger logrus.FieldLogger) (cloud.InstanceSet, error) {
	driver, ok := drivers[cluster.CloudVMs.Driver]
	if !ok {
		return nil, fmt.Errorf("unsupported cloud driver %q", cluster.CloudVMs.Driver)
	}
	return driver.InstanceSet(cluster.CloudVMs.DriverParameters, setID, logger)
}
