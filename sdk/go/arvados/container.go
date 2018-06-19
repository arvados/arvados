// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Container is an arvados#container resource.
type Container struct {
	UUID                 string               `json:"uuid"`
	CreatedAt            time.Time            `json:"created_at"`
	Command              []string             `json:"command"`
	ContainerImage       string               `json:"container_image"`
	Cwd                  string               `json:"cwd"`
	Environment          map[string]string    `json:"environment"`
	LockedByUUID         string               `json:"locked_by_uuid"`
	Mounts               map[string]Mount     `json:"mounts"`
	Output               string               `json:"output"`
	OutputPath           string               `json:"output_path"`
	Priority             int                  `json:"priority"`
	RuntimeConstraints   RuntimeConstraints   `json:"runtime_constraints"`
	State                ContainerState       `json:"state"`
	SchedulingParameters SchedulingParameters `json:"scheduling_parameters"`
}

// Mount is special behavior to attach to a filesystem path or device.
type Mount struct {
	Kind              string      `json:"kind"`
	Writable          bool        `json:"writable"`
	PortableDataHash  string      `json:"portable_data_hash"`
	UUID              string      `json:"uuid"`
	DeviceType        string      `json:"device_type"`
	Path              string      `json:"path"`
	Content           interface{} `json:"content"`
	ExcludeFromOutput bool        `json:"exclude_from_output"`
	Capacity          int64       `json:"capacity"`
	Commit            string      `json:"commit"`          // only if kind=="git_tree"
	RepositoryName    string      `json:"repository_name"` // only if kind=="git_tree"
	GitURL            string      `json:"git_url"`         // only if kind=="git_tree"
}

// RuntimeConstraints specify a container's compute resources (RAM,
// CPU) and network connectivity.
type RuntimeConstraints struct {
	API          *bool
	RAM          int64 `json:"ram"`
	VCPUs        int   `json:"vcpus"`
	KeepCacheRAM int64 `json:"keep_cache_ram"`
}

// SchedulingParameters specify a container's scheduling parameters
// such as Partitions
type SchedulingParameters struct {
	Partitions  []string `json:"partitions"`
	Preemptible bool     `json:"preemptible"`
}

// ContainerList is an arvados#containerList resource.
type ContainerList struct {
	Items          []Container `json:"items"`
	ItemsAvailable int         `json:"items_available"`
	Offset         int         `json:"offset"`
	Limit          int         `json:"limit"`
}

// ContainerState is a string corresponding to a valid Container state.
type ContainerState string

const (
	ContainerStateQueued    = ContainerState("Queued")
	ContainerStateLocked    = ContainerState("Locked")
	ContainerStateRunning   = ContainerState("Running")
	ContainerStateComplete  = ContainerState("Complete")
	ContainerStateCancelled = ContainerState("Cancelled")
)
