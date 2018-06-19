// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"git.curoverse.com/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&NodeSizeSuite{})

const GiB = int64(1 << 30)

type NodeSizeSuite struct{}

func (*NodeSizeSuite) TestChooseNotConfigured(c *check.C) {
	_, err := ChooseInstanceType(&arvados.Cluster{}, &arvados.Container{
		RuntimeConstraints: arvados.RuntimeConstraints{
			RAM:   1234567890,
			VCPUs: 2,
		},
	})
	c.Check(err, check.Equals, ErrInstanceTypesNotConfigured)
}

func (*NodeSizeSuite) TestChooseUnsatisfiable(c *check.C) {
	checkUnsatisfiable := func(ctr *arvados.Container) {
		_, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: []arvados.InstanceType{
			{Price: 1.1, RAM: 1000000000, VCPUs: 2, Name: "small1"},
			{Price: 2.2, RAM: 2000000000, VCPUs: 4, Name: "small2"},
			{Price: 4.4, RAM: 4000000000, VCPUs: 8, Name: "small4", Scratch: GiB},
		}}, ctr)
		c.Check(err, check.FitsTypeOf, ConstraintsNotSatisfiableError{})
	}

	for _, rc := range []arvados.RuntimeConstraints{
		{RAM: 9876543210, VCPUs: 2},
		{RAM: 1234567890, VCPUs: 20},
		{RAM: 1234567890, VCPUs: 2, KeepCacheRAM: 9876543210},
	} {
		checkUnsatisfiable(&arvados.Container{RuntimeConstraints: rc})
	}
	checkUnsatisfiable(&arvados.Container{
		Mounts:             map[string]arvados.Mount{"/tmp": {Kind: "tmp", Capacity: 2 * GiB}},
		RuntimeConstraints: arvados.RuntimeConstraints{RAM: 12345, VCPUs: 1},
	})
}

func (*NodeSizeSuite) TestChoose(c *check.C) {
	for _, menu := range [][]arvados.InstanceType{
		{
			{Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
			{Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			{Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 2 * GiB, Name: "small"},
		},
		{
			{Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
			{Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "goodenough"},
			{Price: 2.2, RAM: 4000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			{Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 2 * GiB, Name: "small"},
		},
		{
			{Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 2 * GiB, Name: "small"},
			{Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "goodenough"},
			{Price: 2.2, RAM: 4000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			{Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
		},
		{
			{Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: GiB, Name: "small"},
			{Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: GiB, Name: "nearly"},
			{Price: 3.3, RAM: 4000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			{Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
		},
	} {
		best, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: menu}, &arvados.Container{
			Mounts: map[string]arvados.Mount{
				"/tmp": {Kind: "tmp", Capacity: 2 * GiB},
			},
			RuntimeConstraints: arvados.RuntimeConstraints{
				VCPUs:        2,
				RAM:          987654321,
				KeepCacheRAM: 123456789,
			},
		})
		c.Check(err, check.IsNil)
		c.Check(best.Name, check.Equals, "best")
		c.Check(best.RAM >= 1234567890, check.Equals, true)
		c.Check(best.VCPUs >= 2, check.Equals, true)
		c.Check(best.Scratch >= 2*GiB, check.Equals, true)
	}
}

func (*NodeSizeSuite) TestChoosePreemptible(c *check.C) {
	menu := []arvados.InstanceType{
		{Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Preemptible: true, Name: "costly"},
		{Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "almost best"},
		{Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Preemptible: true, Name: "best"},
		{Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 2 * GiB, Preemptible: true, Name: "small"},
	}
	best, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: menu}, &arvados.Container{
		Mounts: map[string]arvados.Mount{
			"/tmp": {Kind: "tmp", Capacity: 2 * GiB},
		},
		RuntimeConstraints: arvados.RuntimeConstraints{
			VCPUs:        2,
			RAM:          987654321,
			KeepCacheRAM: 123456789,
		},
		SchedulingParameters: arvados.SchedulingParameters{
			Preemptible: true,
		},
	})
	c.Check(err, check.IsNil)
	c.Check(best.Name, check.Equals, "best")
	c.Check(best.RAM >= 1234567890, check.Equals, true)
	c.Check(best.VCPUs >= 2, check.Equals, true)
	c.Check(best.Scratch >= 2*GiB, check.Equals, true)
	c.Check(best.Preemptible, check.Equals, true)
}
