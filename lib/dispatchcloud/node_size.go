// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"errors"
	"sort"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var ErrInstanceTypesNotConfigured = errors.New("site configuration does not list any instance types")

var discountConfiguredRAMPercent = 5

// ConstraintsNotSatisfiableError includes a list of available instance types
// to be reported back to the user.
type ConstraintsNotSatisfiableError struct {
	error
	AvailableTypes []arvados.InstanceType
}

// ChooseInstanceType returns the cheapest available
// arvados.InstanceType big enough to run ctr.
func ChooseInstanceType(cc *arvados.Cluster, ctr *arvados.Container) (best arvados.InstanceType, err error) {
	if len(cc.InstanceTypes) == 0 {
		err = ErrInstanceTypesNotConfigured
		return
	}

	needScratch := int64(0)
	for _, m := range ctr.Mounts {
		if m.Kind == "tmp" {
			needScratch += m.Capacity
		}
	}

	needVCPUs := ctr.RuntimeConstraints.VCPUs

	needRAM := ctr.RuntimeConstraints.RAM + ctr.RuntimeConstraints.KeepCacheRAM
	needRAM = (needRAM * 100) / int64(100-discountConfiguredRAMPercent)

	ok := false
	for _, it := range cc.InstanceTypes {
		switch {
		case ok && it.Price > best.Price:
		case int64(it.Scratch) < needScratch:
		case int64(it.RAM) < needRAM:
		case it.VCPUs < needVCPUs:
		case it.Preemptible != ctr.SchedulingParameters.Preemptible:
		case it.Price == best.Price && (it.RAM < best.RAM || it.VCPUs < best.VCPUs):
			// Equal price, but worse specs
		default:
			// Lower price || (same price && better specs)
			best = it
			ok = true
		}
	}
	if !ok {
		availableTypes := make([]arvados.InstanceType, 0, len(cc.InstanceTypes))
		for _, t := range cc.InstanceTypes {
			availableTypes = append(availableTypes, t)
		}
		sort.Slice(availableTypes, func(a, b int) bool {
			return availableTypes[a].Price < availableTypes[b].Price
		})
		err = ConstraintsNotSatisfiableError{
			errors.New("constraints not satisfiable by any configured instance type"),
			availableTypes,
		}
		return
	}
	return
}
