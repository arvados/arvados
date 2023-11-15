// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"errors"
	"math"
	"regexp"
	"sort"
	"strconv"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

var ErrInstanceTypesNotConfigured = errors.New("site configuration does not list any instance types")

var discountConfiguredRAMPercent = 5

// ConstraintsNotSatisfiableError includes a list of available instance types
// to be reported back to the user.
type ConstraintsNotSatisfiableError struct {
	error
	AvailableTypes []arvados.InstanceType
}

var pdhRegexp = regexp.MustCompile(`^[0-9a-f]{32}\+(\d+)$`)

// estimateDockerImageSize estimates how much disk space will be used
// by a Docker image, given the PDH of a collection containing a
// Docker image that was created by "arv-keepdocker".  Returns
// estimated number of bytes of disk space that should be reserved.
func estimateDockerImageSize(collectionPDH string) int64 {
	m := pdhRegexp.FindStringSubmatch(collectionPDH)
	if m == nil {
		return 0
	}
	n, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil || n < 122 {
		return 0
	}
	// To avoid having to fetch the collection, take advantage of
	// the fact that the manifest storing a container image
	// uploaded by arv-keepdocker has a predictable format, which
	// allows us to estimate the size of the image based on just
	// the size of the manifest.
	//
	// Use the following heuristic:
	// - Start with the length of the manifest (n)
	// - Subtract 80 characters for the filename and file segment
	// - Divide by 42 to get the number of block identifiers ('hash\+size\ ' is 32+1+8+1)
	// - Assume each block is full, multiply by 64 MiB
	return ((n - 80) / 42) * (64 * 1024 * 1024)
}

// EstimateScratchSpace estimates how much available disk space (in
// bytes) is needed to run the container by summing the capacity
// requested by 'tmp' mounts plus disk space required to load the
// Docker image plus arv-mount block cache.
func EstimateScratchSpace(ctr *arvados.Container) (needScratch int64) {
	for _, m := range ctr.Mounts {
		if m.Kind == "tmp" {
			needScratch += m.Capacity
		}
	}

	// Account for disk space usage by Docker, assumes the following behavior:
	// - Layer tarballs are buffered to disk during "docker load".
	// - Individual layer tarballs are extracted from buffered
	// copy to the filesystem
	dockerImageSize := estimateDockerImageSize(ctr.ContainerImage)

	// The buffer is only needed during image load, so make sure
	// the baseline scratch space at least covers dockerImageSize,
	// and assume it will be released to the job afterwards.
	if needScratch < dockerImageSize {
		needScratch = dockerImageSize
	}

	// Now reserve space for the extracted image on disk.
	needScratch += dockerImageSize

	// Now reserve space the arv-mount disk cache
	needScratch += ctr.RuntimeConstraints.KeepCacheDisk

	return
}

// compareVersion returns true if vs1 < vs2, otherwise false
func versionLess(vs1 string, vs2 string) (bool, error) {
	v1, err := strconv.ParseFloat(vs1, 64)
	if err != nil {
		return false, err
	}
	v2, err := strconv.ParseFloat(vs2, 64)
	if err != nil {
		return false, err
	}
	return v1 < v2, nil
}

// ChooseInstanceType returns the arvados.InstanceTypes eligible to
// run ctr, i.e., those that have enough RAM, VCPUs, etc., and are not
// too expensive according to cluster configuration.
//
// The returned types are sorted with lower prices first.
//
// The error is non-nil if and only if the returned slice is empty.
func ChooseInstanceType(cc *arvados.Cluster, ctr *arvados.Container) ([]arvados.InstanceType, error) {
	if len(cc.InstanceTypes) == 0 {
		return nil, ErrInstanceTypesNotConfigured
	}

	needScratch := EstimateScratchSpace(ctr)

	needVCPUs := ctr.RuntimeConstraints.VCPUs

	needRAM := ctr.RuntimeConstraints.RAM + ctr.RuntimeConstraints.KeepCacheRAM
	needRAM += int64(cc.Containers.ReserveExtraRAM)
	if cc.Containers.LocalKeepBlobBuffersPerVCPU > 0 {
		// + 200 MiB for keepstore process + 10% for GOGC=10
		needRAM += 220 << 20
		// + 64 MiB for each blob buffer + 10% for GOGC=10
		needRAM += int64(cc.Containers.LocalKeepBlobBuffersPerVCPU * needVCPUs * (1 << 26) * 11 / 10)
	}
	needRAM = (needRAM * 100) / int64(100-discountConfiguredRAMPercent)

	maxPriceFactor := math.Max(cc.Containers.MaximumPriceFactor, 1)
	var types []arvados.InstanceType
	var maxPrice float64
	for _, it := range cc.InstanceTypes {
		driverInsuff, driverErr := versionLess(it.CUDA.DriverVersion, ctr.RuntimeConstraints.CUDA.DriverVersion)
		capabilityInsuff, capabilityErr := versionLess(it.CUDA.HardwareCapability, ctr.RuntimeConstraints.CUDA.HardwareCapability)

		switch {
		// reasons to reject a node
		case maxPrice > 0 && it.Price > maxPrice: // too expensive
		case int64(it.Scratch) < needScratch: // insufficient scratch
		case int64(it.RAM) < needRAM: // insufficient RAM
		case it.VCPUs < needVCPUs: // insufficient VCPUs
		case it.Preemptible != ctr.SchedulingParameters.Preemptible: // wrong preemptable setting
		case it.CUDA.DeviceCount < ctr.RuntimeConstraints.CUDA.DeviceCount: // insufficient CUDA devices
		case ctr.RuntimeConstraints.CUDA.DeviceCount > 0 && (driverInsuff || driverErr != nil): // insufficient driver version
		case ctr.RuntimeConstraints.CUDA.DeviceCount > 0 && (capabilityInsuff || capabilityErr != nil): // insufficient hardware capability
			// Don't select this node
		default:
			// Didn't reject the node, so select it
			types = append(types, it)
			if newmax := it.Price * maxPriceFactor; newmax < maxPrice || maxPrice == 0 {
				maxPrice = newmax
			}
		}
	}
	if len(types) == 0 {
		availableTypes := make([]arvados.InstanceType, 0, len(cc.InstanceTypes))
		for _, t := range cc.InstanceTypes {
			availableTypes = append(availableTypes, t)
		}
		sort.Slice(availableTypes, func(a, b int) bool {
			return availableTypes[a].Price < availableTypes[b].Price
		})
		return nil, ConstraintsNotSatisfiableError{
			errors.New("constraints not satisfiable by any configured instance type"),
			availableTypes,
		}
	}
	sort.Slice(types, func(i, j int) bool {
		if types[i].Price != types[j].Price {
			// prefer lower price
			return types[i].Price < types[j].Price
		}
		if types[i].RAM != types[j].RAM {
			// if same price, prefer more RAM
			return types[i].RAM > types[j].RAM
		}
		if types[i].VCPUs != types[j].VCPUs {
			// if same price and RAM, prefer more VCPUs
			return types[i].VCPUs > types[j].VCPUs
		}
		if types[i].Scratch != types[j].Scratch {
			// if same price and RAM and VCPUs, prefer more scratch
			return types[i].Scratch > types[j].Scratch
		}
		// no preference, just sort the same way each time
		return types[i].Name < types[j].Name
	})
	// Truncate types at maxPrice. We rejected it.Price>maxPrice
	// in the loop above, but at that point maxPrice wasn't
	// necessarily the final (lowest) maxPrice.
	for i, it := range types {
		if i > 0 && it.Price > maxPrice {
			types = types[:i]
			break
		}
	}
	return types, nil
}
