// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0

package arvados

import "time"

// Container is an arvados#container resource.
type Container struct {
	UUID                      string                 `json:"uuid"`
	Etag                      string                 `json:"etag"`
	CreatedAt                 time.Time              `json:"created_at"`
	ModifiedByClientUUID      string                 `json:"modified_by_client_uuid"`
	ModifiedByUserUUID        string                 `json:"modified_by_user_uuid"`
	ModifiedAt                time.Time              `json:"modified_at"`
	Command                   []string               `json:"command"`
	ContainerImage            string                 `json:"container_image"`
	Cwd                       string                 `json:"cwd"`
	Environment               map[string]string      `json:"environment"`
	LockedByUUID              string                 `json:"locked_by_uuid"`
	Mounts                    map[string]Mount       `json:"mounts"`
	Output                    string                 `json:"output"`
	OutputPath                string                 `json:"output_path"`
	Priority                  int64                  `json:"priority"`
	RuntimeConstraints        RuntimeConstraints     `json:"runtime_constraints"`
	State                     ContainerState         `json:"state"`
	SchedulingParameters      SchedulingParameters   `json:"scheduling_parameters"`
	ExitCode                  int                    `json:"exit_code"`
	RuntimeStatus             map[string]interface{} `json:"runtime_status"`
	StartedAt                 *time.Time             `json:"started_at"`  // nil if not yet started
	FinishedAt                *time.Time             `json:"finished_at"` // nil if not yet finished
	GatewayAddress            string                 `json:"gateway_address"`
	InteractiveSessionStarted bool                   `json:"interactive_session_started"`
}

// ContainerRequest is an arvados#container_request resource.
type ContainerRequest struct {
	UUID                    string                 `json:"uuid"`
	OwnerUUID               string                 `json:"owner_uuid"`
	CreatedAt               time.Time              `json:"created_at"`
	ModifiedByClientUUID    string                 `json:"modified_by_client_uuid"`
	ModifiedByUserUUID      string                 `json:"modified_by_user_uuid"`
	ModifiedAt              time.Time              `json:"modified_at"`
	Href                    string                 `json:"href"`
	Etag                    string                 `json:"etag"`
	Name                    string                 `json:"name"`
	Description             string                 `json:"description"`
	Properties              map[string]interface{} `json:"properties"`
	State                   ContainerRequestState  `json:"state"`
	RequestingContainerUUID string                 `json:"requesting_container_uuid"`
	ContainerUUID           string                 `json:"container_uuid"`
	ContainerCountMax       int                    `json:"container_count_max"`
	Mounts                  map[string]Mount       `json:"mounts"`
	RuntimeConstraints      RuntimeConstraints     `json:"runtime_constraints"`
	SchedulingParameters    SchedulingParameters   `json:"scheduling_parameters"`
	ContainerImage          string                 `json:"container_image"`
	Environment             map[string]string      `json:"environment"`
	Cwd                     string                 `json:"cwd"`
	Command                 []string               `json:"command"`
	OutputPath              string                 `json:"output_path"`
	OutputName              string                 `json:"output_name"`
	OutputTTL               int                    `json:"output_ttl"`
	Priority                int                    `json:"priority"`
	UseExisting             bool                   `json:"use_existing"`
	LogUUID                 string                 `json:"log_uuid"`
	OutputUUID              string                 `json:"output_uuid"`
	RuntimeToken            string                 `json:"runtime_token"`
	ExpiresAt               time.Time              `json:"expires_at"`
	Filters                 []Filter               `json:"filters"`
	ContainerCount          int                    `json:"container_count"`
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
	API          bool  `json:"API"`
	RAM          int64 `json:"ram"`
	VCPUs        int   `json:"vcpus"`
	KeepCacheRAM int64 `json:"keep_cache_ram"`
}

// SchedulingParameters specify a container's scheduling parameters
// such as Partitions
type SchedulingParameters struct {
	Partitions  []string `json:"partitions"`
	Preemptible bool     `json:"preemptible"`
	MaxRunTime  int      `json:"max_run_time"`
}

// ContainerList is an arvados#containerList resource.
type ContainerList struct {
	Items          []Container `json:"items"`
	ItemsAvailable int         `json:"items_available"`
	Offset         int         `json:"offset"`
	Limit          int         `json:"limit"`
}

// ContainerRequestList is an arvados#containerRequestList resource.
type ContainerRequestList struct {
	Items          []ContainerRequest `json:"items"`
	ItemsAvailable int                `json:"items_available"`
	Offset         int                `json:"offset"`
	Limit          int                `json:"limit"`
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

// ContainerRequestState is a string corresponding to a valid Container Request state.
type ContainerRequestState string

const (
	ContainerRequestStateUncomitted = ContainerRequestState("Uncommitted")
	ContainerRequestStateCommitted  = ContainerRequestState("Committed")
	ContainerRequestStateFinal      = ContainerRequestState("Final")
)
