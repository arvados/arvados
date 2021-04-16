// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"fmt"
	"time"

	"git.arvados.org/arvados.git/lib/cloud"
	"git.arvados.org/arvados.git/lib/cloud/azure"
	"git.arvados.org/arvados.git/lib/cloud/ec2"
	"git.arvados.org/arvados.git/sdk/go/arvados"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// Drivers is a map of available cloud drivers.
// Clusters.*.Containers.CloudVMs.Driver configuration values
// correspond to keys in this map.
var Drivers = map[string]cloud.Driver{
	"azure": azure.Driver,
	"ec2":   ec2.Driver,
}

func newInstanceSet(cluster *arvados.Cluster, setID cloud.InstanceSetID, logger logrus.FieldLogger, reg *prometheus.Registry) (cloud.InstanceSet, error) {
	driver, ok := Drivers[cluster.Containers.CloudVMs.Driver]
	if !ok {
		return nil, fmt.Errorf("unsupported cloud driver %q", cluster.Containers.CloudVMs.Driver)
	}
	sharedResourceTags := cloud.SharedResourceTags(cluster.Containers.CloudVMs.ResourceTags)
	is, err := driver.InstanceSet(cluster.Containers.CloudVMs.DriverParameters, setID, sharedResourceTags, logger)
	is = newInstrumentedInstanceSet(is, reg)
	if maxops := cluster.Containers.CloudVMs.MaxCloudOpsPerSecond; maxops > 0 {
		is = rateLimitedInstanceSet{
			InstanceSet: is,
			ticker:      time.NewTicker(time.Second / time.Duration(maxops)),
		}
	}
	is = defaultTaggingInstanceSet{
		InstanceSet: is,
		defaultTags: cloud.InstanceTags(cluster.Containers.CloudVMs.ResourceTags),
	}
	is = filteringInstanceSet{
		InstanceSet: is,
		logger:      logger,
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

func (inst *rateLimitedInstance) SetTags(tags cloud.InstanceTags) error {
	<-inst.ticker.C
	return inst.Instance.SetTags(tags)
}

// Adds the specified defaultTags to every Create() call.
type defaultTaggingInstanceSet struct {
	cloud.InstanceSet
	defaultTags cloud.InstanceTags
}

func (is defaultTaggingInstanceSet) Create(it arvados.InstanceType, image cloud.ImageID, tags cloud.InstanceTags, init cloud.InitCommand, pk ssh.PublicKey) (cloud.Instance, error) {
	allTags := cloud.InstanceTags{}
	for k, v := range is.defaultTags {
		allTags[k] = v
	}
	for k, v := range tags {
		allTags[k] = v
	}
	return is.InstanceSet.Create(it, image, allTags, init, pk)
}

// Filter the instances returned by the wrapped InstanceSet's
// Instances() method (in case the wrapped InstanceSet didn't do this
// itself).
type filteringInstanceSet struct {
	cloud.InstanceSet
	logger logrus.FieldLogger
}

func (is filteringInstanceSet) Instances(tags cloud.InstanceTags) ([]cloud.Instance, error) {
	instances, err := is.InstanceSet.Instances(tags)

	skipped := 0
	var returning []cloud.Instance
nextInstance:
	for _, inst := range instances {
		instTags := inst.Tags()
		for k, v := range tags {
			if instTags[k] != v {
				skipped++
				continue nextInstance
			}
		}
		returning = append(returning, inst)
	}
	is.logger.WithFields(logrus.Fields{
		"returning": len(returning),
		"skipped":   skipped,
	}).WithError(err).Debugf("filteringInstanceSet returning instances")
	return returning, err
}

func newInstrumentedInstanceSet(is cloud.InstanceSet, reg *prometheus.Registry) cloud.InstanceSet {
	cv := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "arvados",
		Subsystem: "dispatchcloud",
		Name:      "driver_operations",
		Help:      "Number of instance-create/destroy/list operations performed via cloud driver.",
	}, []string{"operation", "error"})

	// Create all counters, so they are reported with zero values
	// (instead of being missing) until they are incremented.
	for _, op := range []string{"Create", "List", "Destroy", "SetTags"} {
		for _, error := range []string{"0", "1"} {
			cv.WithLabelValues(op, error).Add(0)
		}
	}

	reg.MustRegister(cv)
	return instrumentedInstanceSet{is, cv}
}

type instrumentedInstanceSet struct {
	cloud.InstanceSet
	cv *prometheus.CounterVec
}

func (is instrumentedInstanceSet) Create(it arvados.InstanceType, image cloud.ImageID, tags cloud.InstanceTags, init cloud.InitCommand, pk ssh.PublicKey) (cloud.Instance, error) {
	inst, err := is.InstanceSet.Create(it, image, tags, init, pk)
	is.cv.WithLabelValues("Create", boolLabelValue(err != nil)).Inc()
	return instrumentedInstance{inst, is.cv}, err
}

func (is instrumentedInstanceSet) Instances(tags cloud.InstanceTags) ([]cloud.Instance, error) {
	instances, err := is.InstanceSet.Instances(tags)
	is.cv.WithLabelValues("List", boolLabelValue(err != nil)).Inc()
	var instrumented []cloud.Instance
	for _, i := range instances {
		instrumented = append(instrumented, instrumentedInstance{i, is.cv})
	}
	return instrumented, err
}

type instrumentedInstance struct {
	cloud.Instance
	cv *prometheus.CounterVec
}

func (inst instrumentedInstance) Destroy() error {
	err := inst.Instance.Destroy()
	inst.cv.WithLabelValues("Destroy", boolLabelValue(err != nil)).Inc()
	return err
}

func (inst instrumentedInstance) SetTags(tags cloud.InstanceTags) error {
	err := inst.Instance.SetTags(tags)
	inst.cv.WithLabelValues("SetTags", boolLabelValue(err != nil)).Inc()
	return err
}

func boolLabelValue(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
