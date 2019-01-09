// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"fmt"

	"git.curoverse.com/arvados.git/lib/cloud"
	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var drivers = map[string]cloud.Driver{
	"azure": cloud.DriverFunc(cloud.NewAzureInstanceSet),
}

func newInstanceSet(cluster *arvados.Cluster, setID cloud.InstanceSetID) (cloud.InstanceSet, error) {
	driver, ok := drivers[cluster.CloudVMs.Driver]
	if !ok {
		return nil, fmt.Errorf("unsupported cloud driver %q", cluster.CloudVMs.Driver)
	}
	return driver.InstanceSet(cluster.CloudVMs.DriverParameters, setID)
}
