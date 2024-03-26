// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"git.arvados.org/arvados.git/sdk/go/arvados"
	check "gopkg.in/check.v1"
)

var _ = check.Suite(&NodeSizeSuite{})

const GiB = arvados.ByteSize(1 << 30)

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
		_, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: map[string]arvados.InstanceType{
			"small1": {Price: 1.1, RAM: 1000000000, VCPUs: 2, Name: "small1"},
			"small2": {Price: 2.2, RAM: 2000000000, VCPUs: 4, Name: "small2"},
			"small4": {Price: 4.4, RAM: 4000000000, VCPUs: 8, Name: "small4", Scratch: GiB},
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
		Mounts:             map[string]arvados.Mount{"/tmp": {Kind: "tmp", Capacity: int64(2 * GiB)}},
		RuntimeConstraints: arvados.RuntimeConstraints{RAM: 12345, VCPUs: 1},
	})
}

func (*NodeSizeSuite) TestChoose(c *check.C) {
	for _, menu := range []map[string]arvados.InstanceType{
		{
			"costly": {Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
			"best":   {Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			"small":  {Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 2 * GiB, Name: "small"},
		},
		{
			"costly":     {Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
			"goodenough": {Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "goodenough"},
			"best":       {Price: 2.2, RAM: 4000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			"small":      {Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 2 * GiB, Name: "small"},
		},
		{
			"small":      {Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 2 * GiB, Name: "small"},
			"goodenough": {Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "goodenough"},
			"best":       {Price: 2.2, RAM: 4000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			"costly":     {Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
		},
		{
			"small":  {Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: GiB, Name: "small"},
			"nearly": {Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: GiB, Name: "nearly"},
			"best":   {Price: 3.3, RAM: 4000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			"costly": {Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
		},
		{
			"small":  {Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: GiB, Name: "small"},
			"nearly": {Price: 2.2, RAM: 1200000000, VCPUs: 4, Scratch: 2 * GiB, Name: "nearly"},
			"best":   {Price: 3.3, RAM: 4000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
			"costly": {Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
		},
	} {
		best, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: menu, Containers: arvados.ContainersConfig{
			LocalKeepBlobBuffersPerVCPU: 1,
			ReserveExtraRAM:             268435456,
		}}, &arvados.Container{
			Mounts: map[string]arvados.Mount{
				"/tmp": {Kind: "tmp", Capacity: 2 * int64(GiB)},
			},
			RuntimeConstraints: arvados.RuntimeConstraints{
				VCPUs:        2,
				RAM:          987654321,
				KeepCacheRAM: 123456789,
			},
		})
		c.Assert(err, check.IsNil)
		c.Assert(best, check.Not(check.HasLen), 0)
		c.Check(best[0].Name, check.Equals, "best")
		c.Check(best[0].RAM >= 1234567890, check.Equals, true)
		c.Check(best[0].VCPUs >= 2, check.Equals, true)
		c.Check(best[0].Scratch >= 2*GiB, check.Equals, true)
		for i := range best {
			// If multiple instance types are returned
			// then they should all have the same price,
			// because we didn't set MaximumPriceFactor>1.
			c.Check(best[i].Price, check.Equals, best[0].Price)
		}
	}
}

func (*NodeSizeSuite) TestMaximumPriceFactor(c *check.C) {
	menu := map[string]arvados.InstanceType{
		"best+7":  {Price: 3.4, RAM: 8000000000, VCPUs: 8, Scratch: 64 * GiB, Name: "best+7"},
		"best+5":  {Price: 3.0, RAM: 8000000000, VCPUs: 8, Scratch: 16 * GiB, Name: "best+5"},
		"best+3":  {Price: 2.6, RAM: 4000000000, VCPUs: 8, Scratch: 16 * GiB, Name: "best+3"},
		"best+2":  {Price: 2.4, RAM: 4000000000, VCPUs: 8, Scratch: 4 * GiB, Name: "best+2"},
		"best+1":  {Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 4 * GiB, Name: "best+1"},
		"best":    {Price: 2.0, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
		"small+1": {Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 16 * GiB, Name: "small+1"},
		"small":   {Price: 1.0, RAM: 2000000000, VCPUs: 2, Scratch: 1 * GiB, Name: "small"},
	}
	best, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: menu, Containers: arvados.ContainersConfig{
		MaximumPriceFactor: 1.5,
	}}, &arvados.Container{
		Mounts: map[string]arvados.Mount{
			"/tmp": {Kind: "tmp", Capacity: 2 * int64(GiB)},
		},
		RuntimeConstraints: arvados.RuntimeConstraints{
			VCPUs:        2,
			RAM:          987654321,
			KeepCacheRAM: 123456789,
		},
	})
	c.Assert(err, check.IsNil)
	c.Assert(best, check.HasLen, 5)
	c.Check(best[0].Name, check.Equals, "best") // best price is $2
	c.Check(best[1].Name, check.Equals, "best+1")
	c.Check(best[2].Name, check.Equals, "best+2")
	c.Check(best[3].Name, check.Equals, "best+3")
	c.Check(best[4].Name, check.Equals, "best+5") // max price is $2 * 1.5 = $3
}

func (*NodeSizeSuite) TestChooseWithBlobBuffersOverhead(c *check.C) {
	menu := map[string]arvados.InstanceType{
		"nearly": {Price: 2.2, RAM: 4000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "small"},
		"best":   {Price: 3.3, RAM: 8000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best"},
		"costly": {Price: 4.4, RAM: 12000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly"},
	}
	best, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: menu, Containers: arvados.ContainersConfig{
		LocalKeepBlobBuffersPerVCPU: 16, // 1 GiB per vcpu => 2 GiB
		ReserveExtraRAM:             268435456,
	}}, &arvados.Container{
		Mounts: map[string]arvados.Mount{
			"/tmp": {Kind: "tmp", Capacity: 2 * int64(GiB)},
		},
		RuntimeConstraints: arvados.RuntimeConstraints{
			VCPUs:        2,
			RAM:          987654321,
			KeepCacheRAM: 123456789,
		},
	})
	c.Check(err, check.IsNil)
	c.Assert(best, check.HasLen, 1)
	c.Check(best[0].Name, check.Equals, "best")
}

func (*NodeSizeSuite) TestChoosePreemptible(c *check.C) {
	menu := map[string]arvados.InstanceType{
		"costly":      {Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Preemptible: true, Name: "costly"},
		"almost best": {Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "almost best"},
		"best":        {Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Preemptible: true, Name: "best"},
		"small":       {Price: 1.1, RAM: 1000000000, VCPUs: 2, Scratch: 2 * GiB, Preemptible: true, Name: "small"},
	}
	best, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: menu}, &arvados.Container{
		Mounts: map[string]arvados.Mount{
			"/tmp": {Kind: "tmp", Capacity: 2 * int64(GiB)},
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
	c.Assert(best, check.HasLen, 1)
	c.Check(best[0].Name, check.Equals, "best")
	c.Check(best[0].RAM >= 1234567890, check.Equals, true)
	c.Check(best[0].VCPUs >= 2, check.Equals, true)
	c.Check(best[0].Scratch >= 2*GiB, check.Equals, true)
	c.Check(best[0].Preemptible, check.Equals, true)
}

func (*NodeSizeSuite) TestScratchForDockerImage(c *check.C) {
	n := EstimateScratchSpace(&arvados.Container{
		ContainerImage: "d5025c0f29f6eef304a7358afa82a822+342",
	})
	// Actual image is 371.1 MiB (according to workbench)
	// Estimated size is 384 MiB (402653184 bytes)
	// Want to reserve 2x the estimated size, so 805306368 bytes
	c.Check(n, check.Equals, int64(805306368))

	n = EstimateScratchSpace(&arvados.Container{
		ContainerImage: "d5025c0f29f6eef304a7358afa82a822+-342",
	})
	// Parse error will return 0
	c.Check(n, check.Equals, int64(0))

	n = EstimateScratchSpace(&arvados.Container{
		ContainerImage: "d5025c0f29f6eef304a7358afa82a822+34",
	})
	// Short manifest will return 0
	c.Check(n, check.Equals, int64(0))
}

func (*NodeSizeSuite) TestChooseGPU(c *check.C) {
	menu := map[string]arvados.InstanceType{
		"costly":         {Price: 4.4, RAM: 4000000000, VCPUs: 8, Scratch: 2 * GiB, Name: "costly", CUDA: arvados.CUDAFeatures{DeviceCount: 2, HardwareCapability: "9.0", DriverVersion: "11.0"}},
		"low_capability": {Price: 2.1, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "low_capability", CUDA: arvados.CUDAFeatures{DeviceCount: 1, HardwareCapability: "8.0", DriverVersion: "11.0"}},
		"best":           {Price: 2.2, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "best", CUDA: arvados.CUDAFeatures{DeviceCount: 1, HardwareCapability: "9.0", DriverVersion: "11.0"}},
		"low_driver":     {Price: 2.1, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "low_driver", CUDA: arvados.CUDAFeatures{DeviceCount: 1, HardwareCapability: "9.0", DriverVersion: "10.0"}},
		"cheap_gpu":      {Price: 2.0, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "cheap_gpu", CUDA: arvados.CUDAFeatures{DeviceCount: 1, HardwareCapability: "8.0", DriverVersion: "10.0"}},
		"invalid_gpu":    {Price: 1.9, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "invalid_gpu", CUDA: arvados.CUDAFeatures{DeviceCount: 1, HardwareCapability: "12.0.12", DriverVersion: "12.0.12"}},
		"non_gpu":        {Price: 1.1, RAM: 2000000000, VCPUs: 4, Scratch: 2 * GiB, Name: "non_gpu"},
	}

	type GPUTestCase struct {
		CUDA             arvados.CUDARuntimeConstraints
		SelectedInstance string
	}
	cases := []GPUTestCase{
		GPUTestCase{
			CUDA: arvados.CUDARuntimeConstraints{
				DeviceCount:        1,
				HardwareCapability: "9.0",
				DriverVersion:      "11.0",
			},
			SelectedInstance: "best",
		},
		GPUTestCase{
			CUDA: arvados.CUDARuntimeConstraints{
				DeviceCount:        2,
				HardwareCapability: "9.0",
				DriverVersion:      "11.0",
			},
			SelectedInstance: "costly",
		},
		GPUTestCase{
			CUDA: arvados.CUDARuntimeConstraints{
				DeviceCount:        1,
				HardwareCapability: "8.0",
				DriverVersion:      "11.0",
			},
			SelectedInstance: "low_capability",
		},
		GPUTestCase{
			CUDA: arvados.CUDARuntimeConstraints{
				DeviceCount:        1,
				HardwareCapability: "9.0",
				DriverVersion:      "10.0",
			},
			SelectedInstance: "low_driver",
		},
		GPUTestCase{
			CUDA: arvados.CUDARuntimeConstraints{
				DeviceCount:        1,
				HardwareCapability: "",
				DriverVersion:      "10.0",
			},
			SelectedInstance: "",
		},
		GPUTestCase{
			CUDA: arvados.CUDARuntimeConstraints{
				DeviceCount:        0,
				HardwareCapability: "9.0",
				DriverVersion:      "11.0",
			},
			SelectedInstance: "non_gpu",
		},
	}

	for _, tc := range cases {
		best, err := ChooseInstanceType(&arvados.Cluster{InstanceTypes: menu}, &arvados.Container{
			Mounts: map[string]arvados.Mount{
				"/tmp": {Kind: "tmp", Capacity: 2 * int64(GiB)},
			},
			RuntimeConstraints: arvados.RuntimeConstraints{
				VCPUs:        2,
				RAM:          987654321,
				KeepCacheRAM: 123456789,
				CUDA:         tc.CUDA,
			},
		})
		if len(best) > 0 {
			c.Check(err, check.IsNil)
			c.Assert(best, check.HasLen, 1)
			c.Check(best[0].Name, check.Equals, tc.SelectedInstance)
		} else {
			c.Check(err, check.Not(check.IsNil))
		}
	}
}
