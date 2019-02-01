// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package test

import (
	"fmt"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

// ContainerUUID returns a fake container UUID.
func ContainerUUID(i int) string {
	return fmt.Sprintf("zzzzz-dz642-%015d", i)
}

// InstanceType returns a fake arvados.InstanceType called "type{i}"
// with i CPUs and i GiB of memory.
func InstanceType(i int) arvados.InstanceType {
	return arvados.InstanceType{
		Name:         fmt.Sprintf("type%d", i),
		ProviderType: fmt.Sprintf("providertype%d", i),
		VCPUs:        i,
		RAM:          arvados.ByteSize(i) << 30,
		Price:        float64(i) * 0.123,
	}
}
