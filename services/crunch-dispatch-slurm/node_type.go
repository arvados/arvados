// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchslurm

import (
	"log"
	"os/exec"
	"strings"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

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
		time.Sleep(2 * time.Second)
	}
}

const slurmDummyNode = "compute0"

func slurmKludge(features []string) {
	allFeatures := strings.Join(features, ",")

	cmd := exec.Command("sinfo", "--nodes="+slurmDummyNode, "--format=%f", "--noheader")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("running %q %q: %s (output was %q)", cmd.Path, cmd.Args, err, out)
		return
	}
	if string(out) == allFeatures+"\n" {
		// Already configured correctly, nothing to do.
		return
	}

	log.Printf("configuring node %q with all node type features", slurmDummyNode)
	cmd = exec.Command("scontrol", "update", "NodeName="+slurmDummyNode, "Features="+allFeatures)
	log.Printf("running: %q %q", cmd.Path, cmd.Args)
	out, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("error: scontrol: %s (output was %q)", err, out)
	}
}
