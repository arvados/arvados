// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package keepstore

import (
	"github.com/sirupsen/logrus"
)

type putProgress struct {
	classNeeded      map[string]bool
	classTodo        map[string]bool
	mountUsed        map[*mount]bool
	totalReplication int
	classDone        map[string]int
}

func (pr *putProgress) Add(mnt *mount) {
	if pr.mountUsed[mnt] {
		logrus.Warnf("BUG? superfluous extra write to mount %s", mnt.UUID)
		return
	}
	pr.mountUsed[mnt] = true
	pr.totalReplication += mnt.Replication
	for class := range mnt.StorageClasses {
		pr.classDone[class] += mnt.Replication
		delete(pr.classTodo, class)
	}
}

func (pr *putProgress) Sub(mnt *mount) {
	if !pr.mountUsed[mnt] {
		logrus.Warnf("BUG? Sub called with no prior matching Add: %s", mnt.UUID)
		return
	}
	pr.mountUsed[mnt] = false
	pr.totalReplication -= mnt.Replication
	for class := range mnt.StorageClasses {
		pr.classDone[class] -= mnt.Replication
		if pr.classNeeded[class] {
			pr.classTodo[class] = true
		}
	}
}

func (pr *putProgress) Done() bool {
	return len(pr.classTodo) == 0 && pr.totalReplication > 0
}

func (pr *putProgress) Want(mnt *mount) bool {
	if pr.Done() || pr.mountUsed[mnt] {
		return false
	}
	if len(pr.classTodo) == 0 {
		// none specified == "any"
		return true
	}
	for class := range mnt.StorageClasses {
		if pr.classTodo[class] {
			return true
		}
	}
	return false
}

func (pr *putProgress) Copy() *putProgress {
	cp := putProgress{
		classNeeded:      pr.classNeeded,
		classTodo:        make(map[string]bool, len(pr.classTodo)),
		classDone:        make(map[string]int, len(pr.classDone)),
		mountUsed:        make(map[*mount]bool, len(pr.mountUsed)),
		totalReplication: pr.totalReplication,
	}
	for k, v := range pr.classTodo {
		cp.classTodo[k] = v
	}
	for k, v := range pr.classDone {
		cp.classDone[k] = v
	}
	for k, v := range pr.mountUsed {
		cp.mountUsed[k] = v
	}
	return &cp
}

func newPutProgress(classes []string) putProgress {
	pr := putProgress{
		classNeeded: make(map[string]bool, len(classes)),
		classTodo:   make(map[string]bool, len(classes)),
		classDone:   map[string]int{},
		mountUsed:   map[*mount]bool{},
	}
	for _, c := range classes {
		if c != "" {
			pr.classNeeded[c] = true
			pr.classTodo[c] = true
		}
	}
	return pr
}
