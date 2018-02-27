// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"bytes"
	"errors"
	"log"
	"os/exec"
	"strings"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

var (
	ErrConstraintsNotSatisfiable  = errors.New("constraints not satisfiable by any configured instance type")
	ErrInstanceTypesNotConfigured = errors.New("site configuration does not list any instance types")
	discountConfiguredRAMPercent  = 5
)

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

	err = ErrConstraintsNotSatisfiable
	for _, it := range cc.InstanceTypes {
		switch {
		case err == nil && it.Price > best.Price:
		case it.Scratch < needScratch:
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

// SlurmNodeTypeFeatureKludge ensures SLURM accepts every instance
// type name as a valid feature name, even if no instances of that
// type have appeared yet.
//
// It takes advantage of some SLURM peculiarities:
//
// (1) A feature is valid after it has been offered by a node, even if
// it is no longer offered by any node. So, to make a feature name
// valid, we can add it to a dummy node ("compute0"), then remove it.
//
// (2) To test whether a set of feature names are valid without
// actually submitting a job, we can call srun --test-only with the
// desired features.
//
// SlurmNodeTypeFeatureKludge does a test-and-fix operation
// immediately, and then periodically, in case slurm restarts and
// forgets the list of valid features. It never returns (unless there
// are no node types configured, in which case it returns
// immediately), so it should generally be invoked with "go".
func SlurmNodeTypeFeatureKludge(cc *arvados.Cluster) {
	if len(cc.InstanceTypes) == 0 {
		return
	}
	var features []string
	for _, it := range cc.InstanceTypes {
		features = append(features, "instancetype="+it.Name)
	}
	for {
		slurmKludge(features)
		time.Sleep(time.Minute)
	}
}

var (
	slurmDummyNode     = "compute0"
	slurmErrBadFeature = "Invalid feature"
)

func slurmKludge(features []string) {
	cmd := exec.Command("srun", "--test-only", "--constraint="+strings.Join(features, "&"), "false")
	out, err := cmd.CombinedOutput()
	switch {
	case err == nil:
		// Evidently our node-type feature names are all valid.

	case bytes.Contains(out, []byte(slurmErrBadFeature)):
		log.Printf("temporarily configuring node %q with all node type features", slurmDummyNode)
		for _, nodeFeatures := range []string{strings.Join(features, ","), ""} {
			cmd = exec.Command("scontrol", "update", "NodeName="+slurmDummyNode, "Features="+nodeFeatures)
			log.Printf("running: %q %q", cmd.Path, cmd.Args)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("error: scontrol: %s (output was %q)", err, out)
			}
		}

	default:
		log.Printf("warning: expected srun error %q or success, but output was %q", slurmErrBadFeature, out)
	}
}
