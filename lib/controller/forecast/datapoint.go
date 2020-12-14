// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0
package forecast

import (
	"context"
	"fmt"
	"time"

	"git.arvados.org/arvados.git/sdk/go/arvados"
)

// Datapoint structure will hold a container and possible a container request
// to extract all the information needed.
// By filling ContainerRequestUUID and/or ContainerUUID the function hydrate
// will fetch the data. (usually good to make it with a gorutine)
type Datapoint struct {
	// CheckpointName is used to group the values, this is the id of the step or
	// the name of the container usually
	CheckpointName       string
	ContainerRequestUUID string
	ContainerRequest     *arvados.ContainerRequest
	ContainerUUID        string
	Container            arvados.Container

	// Scattered will be true if the name is "..._NNN". representing that is part of a scattered
	// process.

	// TODO: As the first process will have no "_NNN" as soon as we find one identifier that matches
	// scattered pattern, we also assign it to the first process. This is in TODO for now becasue  maps
	// are not thread-safe. We need to implemnt locks to access it, or use sync.Map
	Scattered bool
}

// ErrInvalidDuration is returned when Duration() or other functions are expecting a to have a container
// and is able to calculate the duration of if (i.e. a container has not finished yet)
type ErrInvalidDuration struct {
	ContainerUUID string
}

func (e *ErrInvalidDuration) Error() string {
	return fmt.Sprintf("forecast: Duration for container request '%s' can't be calculated", e.ContainerUUID)
}

// Duration returns a time Duration with the container start and stop
func (d *Datapoint) Duration() (time.Duration, error) {
	if d.Container.FinishedAt == nil {
		return time.Duration(0), &ErrInvalidDuration{ContainerUUID: d.ContainerUUID}
	}

	return d.Container.FinishedAt.Sub(*d.Container.StartedAt), nil
}

// Reuse returns a boolean based
func (d *Datapoint) Reuse() bool {

	if d.ContainerRequest == nil {
		return false
	}
	if d.Container.StartedAt == nil {
		return false
	}

	return d.ContainerRequest.CreatedAt.After(*d.Container.StartedAt)

}

// ErrNoContainer is returned when Hydrate() or other functions are expecting a container
// in a datapoint that still doesn't have it (i.e. has not been executed yet)
type ErrNoContainer struct {
	ContainerUUID string
}

func (e *ErrNoContainer) Error() string {
	return fmt.Sprintf("forecast: Container Request '%s' doesn't have a container yet", e.ContainerUUID)
}

// Hydrate will make d.containerRequest and d.container with the values from
// the cloud (or cache) based on the con
// FIXME how error should be handle?
func (d *Datapoint) Hydrate(ctx context.Context, ctrl Controller) (err error) {
	if d.ContainerUUID == "" {
		return &ErrNoContainer{ContainerUUID: d.ContainerUUID}
	}

	c, err := ctrl.parent.ContainerGet(ctx, arvados.GetOptions{UUID: d.ContainerUUID})
	if err != nil {
		return
	}

	// after the retrival we assign it to the data point.
	d.Container = c
	// TODO: implement cache at this dfunction to
	return
}

// Datapoints is map will have checkpointName as the key to fill in the data as we collect it.
// the starting point could be by parsung stderr.txt logfile from crunch-run or a container
// request and analyzing the information from the database.
type Datapoints map[string]*Datapoint
