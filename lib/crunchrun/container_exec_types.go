// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package crunchrun

import (
	"bufio"
	"io"
	"net"

	"golang.org/x/net/context"
)

// ContainerConfig holds all values needed for Docker and Singularity
// to run a container. In the case of docker is similar to
// github.com/docker/docker/api/types/container/Config
// see https://github.com/moby/moby/blob/master/api/types/container/config.go
// "It should hold only portable information about the container."
// and for Singularity TBD
type ContainerConfig struct {
	Image        string
	OpenStdin    bool
	StdinOnce    bool
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool

	Cmd        []string
	WorkingDir string
	Env        []string
	Volumes    map[string]struct{}
}

// LogConfig represents the logging configuration of the container.
type LogConfig struct {
	Type   string
	Config map[string]string
}

// Resources contains container's resources (cgroups config, ulimits...)
type Resources struct {
	Memory       int64  // Memory limit (in bytes)
	NanoCPUs     int64  `json:"NanoCpus"` // CPU quota in units of 10<sup>-9</sup> CPUs.
	CgroupParent string // Parent cgroup.
	MemorySwap   int64  // Total memory usage (memory + swap); set `-1` to enable unlimited swap
	KernelMemory int64  // Kernel memory limit (in bytes)
}

type NetworkMode string

// HostConfig holds all values needed for Docker and Singularity
// to run a container related to the host. In the case of docker is
// similar to github.com/docker/docker/api/types/container/HostConfig
// see https://github.com/moby/moby/blob/master/api/types/container/host_config.go
// "dependent of the host we are running on".
// and for Singularity TBD
type HostConfig struct {
	NetworkMode NetworkMode
	Binds       []string // List of volume bindings for this container
	//important bits:
	LogConfig LogConfig // Configuration of the logs for this container

	// Contains container's resources (cgroups, ulimits)
	Resources
}

// ---- NETROWKING STUFF
// EndpointIPAMConfig represents IPAM configurations for the endpoint
type EndpointIPAMConfig struct {
	IPv4Address  string   `json:",omitempty"`
	IPv6Address  string   `json:",omitempty"`
	LinkLocalIPs []string `json:",omitempty"`
}

// EndpointSettings stores the network endpoint details
type EndpointSettings struct {
	// Configurations
	IPAMConfig *EndpointIPAMConfig
	Links      []string
	Aliases    []string
	// Operational data
	NetworkID           string
	EndpointID          string
	Gateway             string
	IPAddress           string
	IPPrefixLen         int
	IPv6Gateway         string
	GlobalIPv6Address   string
	GlobalIPv6PrefixLen int
	MacAddress          string
	DriverOpts          map[string]string
}

// ----
// NetworkingConfig holds all values needed for Docker and Singularity
// related network. In the case of docker is similar to
// github.com/docker/docker/api/types/network/NetworkingConfig
// and for Singularity TBD.
type NetworkingConfig struct {
	EndpointsConfig map[string]*EndpointSettings
}

// ContainerCreateResponse in the case of docker will be similar to
// github.com/docker/docker/api/types/container/ContainerCreateCreatedBody
// and for Singularity TBD.
type ContainerCreateResponse struct {
	// The ID of the created container
	// Required: true
	ID string
	// Warnings encountered when creating the container
	// Required: true
	Warnings []string
}

// ContainerStartOptions in the case of docker will be similar to
// github.com/docker/docker/api/types/container/ContainerStartOptions
// and for Singularity TBD.
type ContainerStartOptions struct {
	// FIXME: do we need this in this wrapping? since we only use it's zero value
	// just to comply with Docker's ContainerStart API
	// maybe not using it will be the best
	CheckpointID  string
	CheckpointDir string
}

// ContainerRemoveOptions in the case of docker will be similar to
// github.com/docker/docker/api/types/container/ContainerRemoveOptions
// and for Singularity TBD.
type ContainerRemoveOptions struct {
	// FIXME: we *only* call it with dockertypes.ContainerRemoveOptions{Force: true})
	// may be should not be in this
	Force bool
}

// ContainerAttachOptions in the case of docker will be similar to
// github.com/docker/docker/api/types/container/ContainerAttachOptions
// and for Singularity TBD.
type ContainerAttachOptions struct {
	Stream bool
	Stdin  bool
	Stdout bool
	Stderr bool
}

// ImageRemoveOptions in the case of docker will be similar to
// github.com/docker/docker/api/types/container/ImageRemoveOptions
// and for Singularity TBD.
type ImageRemoveOptions struct {
	Force         bool
	PruneChildren bool
	//not used as far as I know
}

// ContainerInspectResponse in the case of docker will be similar to
// github.com/docker/docker/api/types/ContainerJSON
// and for Singularity TBD.
// MAYBE call it ExecRunnerContainer? since the struct is describing  a container
// from the underlying ExecRunner
type ContainerInspectResponse struct {
	//Important bits for us
	// State = current checks: (nil, Running, Created)
	State *ContainerState
}

// ImageInspectResponse  in the case of docker is similar to
//  github.com/docker/docker/api/types/ImageInspect
// and for Singularity TBD.
// MAYBE call it ExecRunnerImage? since the struct is describing an image
// from the underlying ExecRunner
type ImageInspectResponse struct {
	// we don't use the respones but we use ImageInspectWithRaw(context.TODO(), imageID)
	// to check if we already have the docker image, maybe we can do the
	// a imagePresent(id string) (bool)
	ID string
}

// ImageLoadResponse returns information to the client about a load process.
type ImageLoadResponse struct {
	// Body must be closed to avoid a resource leak
	Body io.ReadCloser
	JSON bool
}

// ImageDeleteResponseItem is a reply from ImageRemove.
type ImageDeleteResponseItem struct {

	// The image ID of an image that was deleted
	Deleted string `json:"Deleted,omitempty"`

	// The image ID of an image that was untagged
	Untagged string `json:"Untagged,omitempty"`
}

// ContainerState stores container's running state
// it's part of ContainerJSONBase and will return by "inspect" command
type ContainerState struct {
	Status     string // String representation of the container state. Can be one of "created", "running", "paused", "restarting", "removing", "exited", or "dead"
	Running    bool
	Paused     bool
	Restarting bool
	OOMKilled  bool
	Dead       bool
	Pid        int
	ExitCode   int
	Error      string
	StartedAt  string
	FinishedAt string
}

// HijackedResponse holds connection information for a hijacked request.

// HijackedResponse is needed as an artifact that comes from docker.
// We need to figure out if this is the best abstraction at this level
// for now this is a copy and paste from docker package. Might evolve later.
type HijackedResponse struct {
	Conn   net.Conn
	Reader *bufio.Reader
}

// Close closes the hijacked connection and reader.
func (h *HijackedResponse) Close() {
	h.Conn.Close()
}

// CloseWriter is an interface that implements structs
// that close input streams to prevent from writing.
type CloseWriter interface {
	CloseWrite() error
}

// CloseWrite closes a readWriter for writing.
func (h *HijackedResponse) CloseWrite() error {
	if conn, ok := h.Conn.(CloseWriter); ok {
		return conn.CloseWrite()
	}
	return nil
}

//------------ End of HijackedResponse

// Similar to HijackedResponse, Waitcondtion is here and will decide later how is implemented in Singularity

// WaitCondition is a type used to specify a container state for which
// to wait.
type WaitCondition string

// Possible WaitCondition Values.
//
// WaitConditionNotRunning (default) is used to wait for any of the non-running
// states: "created", "exited", "dead", "removing", or "removed".
//
// WaitConditionNextExit is used to wait for the next time the state changes
// to a non-running state. If the state is currently "created" or "exited",
// this would cause Wait() to block until either the container runs and exits
// or is removed.
//
// WaitConditionRemoved is used to wait for the container to be removed.
const (
	WaitConditionNotRunning WaitCondition = "not-running"
	WaitConditionNextExit   WaitCondition = "next-exit"
	WaitConditionRemoved    WaitCondition = "removed"
)

//------------ End of WaitCondition

// ContainerWaitOKBody
/// yetanother copy, this time from  from docker/api/types/container/container_wait.go.
// That file is generated from swagger, so I don't think is a good idea to have it
// here. but for now, I'll finish creating the abstraction layer

// ContainerWaitOKBodyError container waiting error, if any
// swagger:model ContainerWaitOKBodyError
type ContainerWaitOKBodyError struct {

	// Details of an error
	Message string `json:"Message,omitempty"`
}

// ContainerWaitOKBody OK response to ContainerWait operation
// swagger:model ContainerWaitOKBody
type ContainerWaitOKBody struct {

	// error
	// Required: true
	Error *ContainerWaitOKBodyError `json:"Error"`

	// Exit code of the container
	// Required: true
	StatusCode int64 `json:"StatusCode"`
}

//------------ End of ContainerWaitOKBody

// ThinContainerExecRunner is the "common denominator" interface for all ExecRunners
// (either Docker or Singularity or more to come). For now is based in the
//  ThinDockerClient interface with our own objects (instead of the ones that come
// from docker)
type ThinContainerExecRunner interface {
	GetContainerConfig() (ContainerConfig, error)
	GetHostConfig() (HostConfig, error)

	GetImage() (imageID string)
	SetImage(imageID string)

	SetHostConfig(hostConfig HostConfig) error
	GetNetworkMode() (networkMode NetworkMode)
	SetNetworkMode(networkMode NetworkMode)

	ContainerAttach(ctx context.Context, container string, options ContainerAttachOptions) (HijackedResponse, error)
	ContainerCreate(ctx context.Context, config ContainerConfig, hostConfig HostConfig, networkingConfig *NetworkingConfig, containerName string) (ContainerCreateResponse, error)
	ContainerStart(ctx context.Context, container string, options ContainerStartOptions) error
	ContainerRemove(ctx context.Context, container string, options ContainerRemoveOptions) error
	ContainerWait(ctx context.Context, container string, condition WaitCondition) (<-chan ContainerWaitOKBody, <-chan error)

	ContainerInspect(ctx context.Context, id string) (ContainerInspectResponse, error)
	ImageInspectWithRaw(ctx context.Context, image string) (ImageInspectResponse, []byte, error)

	ImageLoad(ctx context.Context, input io.Reader, quiet bool) (ImageLoadResponse, error)
	ImageRemove(ctx context.Context, image string, options ImageRemoveOptions) ([]ImageDeleteResponseItem, error)
}
