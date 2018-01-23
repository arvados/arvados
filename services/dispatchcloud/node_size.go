// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"errors"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var (
	ErrConstraintsNotSatisfiable  = errors.New("constraints not satisfiable by any configured instance type")
	ErrInstanceTypesNotConfigured = errors.New("site configuration does not list any instance types")
)

// ChooseInstanceType returns the cheapest available
// arvados.InstanceType big enough to run ctr.
func ChooseInstanceType(cc *arvados.Cluster, ctr *arvados.Container) (best arvados.InstanceType, err error) {
	needVCPUs := ctr.RuntimeConstraints.VCPUs
	needRAM := ctr.RuntimeConstraints.RAM + ctr.RuntimeConstraints.KeepCacheRAM

	if len(cc.InstanceTypes) == 0 {
		err = ErrInstanceTypesNotConfigured
		return
	}

	err = ErrConstraintsNotSatisfiable
	for _, it := range cc.InstanceTypes {
		switch {
		case err == nil && it.Price > best.Price:
		case it.RAM < needRAM:
		case it.VCPUs < needVCPUs:
		case it.Price == best.Price && (it.RAM < best.RAM || it.VCPUs < best.VCPUs):
			// Equal price, but worse specs
		default:
			// Lower price || (same price && better specs)
			best = it
			err = nil
		}
	}
	return
}
