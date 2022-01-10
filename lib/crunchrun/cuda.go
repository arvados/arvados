// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"os/exec"
)

// nvidiaModprobe makes sure all the nvidia kernel modules and devices
// are set up.  If we don't have all the modules/devices set up we get
// "CUDA_ERROR_UNKNOWN".
func nvidiaModprobe(writer *ThrottledLogger) {
	// The underlying problem is that when normally running
	// directly on the host, the CUDA SDK will automatically
	// detect and set up the devices on demand.  However, when
	// running inside a container, it lacks sufficient permissions
	// to do that.  So, it needs to be set up before the container
	// can be started.
	//
	// The Singularity documentation hints about this but isn't
	// very helpful with a solution.
	// https://sylabs.io/guides/3.7/user-guide/gpu.html#cuda-error-unknown-when-everything-seems-to-be-correctly-configured
	//
	// If we're running "nvidia-persistenced", it sets up most of
	// these things on system boot.
	//
	// However, it seems that doesn't include /dev/nvidia-uvm
	// We're also no guaranteed to be running
	// "nvidia-persistenced" or otherwise have the devices set up
	// for us.  So the most robust solution is to do it ourselves.
	//
	// These are idempotent operations so it is harmless in the
	// case that everything was actually already set up.

	// Running nvida-smi the first time loads the core 'nvidia'
	// kernel module creates /dev/nvidiactl the per-GPU
	// /dev/nvidia* devices
	nvidiaSmi := exec.Command("nvidia-smi", "-L")
	nvidiaSmi.Stdout = writer
	nvidiaSmi.Stderr = writer
	err := nvidiaSmi.Run()
	if err != nil {
		writer.Printf("Warning %v: %v", nvidiaSmi.Args, err)
	}

	// Load the kernel modules & devices associated with
	// /dev/nvidia-modeset, /dev/nvidia-nvlink, /dev/nvidia-uvm
	// and /dev/nvidia-uvm-tools (-m, -l and -u).  Annoyingly,
	// these don't have multiple devices but you need to supply
	// "-c0" anyway or it won't make the device file.

	// Nvswitch devices are multi-GPU interconnects for up to 16
	// GPUs.  The "-c0 -s" flag will create /dev/nvidia-nvswitch0.
	// If someone runs Arvados on a system with multiple
	// nvswitches (i.e. more than 16 GPUs) they'll have to ensure
	// that all the /dev/nvidia-nvswitch* devices exist before
	// crunch-run starts.
	for _, opt := range []string{"-m", "-l", "-u", "-s"} {
		nvmodprobe := exec.Command("nvidia-modprobe", "-c0", opt)
		nvmodprobe.Stdout = writer
		nvmodprobe.Stderr = writer
		err = nvmodprobe.Run()
		if err != nil {
			writer.Printf("Warning %v: %v", nvmodprobe.Args, err)
		}
	}
}
