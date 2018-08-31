// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"golang.org/x/crypto/ssh"
)

// A RateLimitError should be returned by a Provider when the cloud
// service indicates it is rejecting all API calls for some time
// interval.
type RateLimitError interface {
	// Time before which the caller should expect requests to
	// fail.
	EarliestRetry() time.Time
	error
}

// A QuotaError should be returned by a Provider when the cloud
// service indicates the account cannot create more VMs than already
// exist.
type QuotaError interface {
	// If true, don't create more instances until some existing
	// instances are destroyed. If false, don't handle the error
	// as a quota error.
	IsQuotaError() bool
	error
}

type InstanceTags map[string]string
type InstanceID string
type ImageID string

// instance is implemented by the provider-specific instance types.
type Instance interface {
	// ID returns the provider's instance ID. It must be stable
	// for the life of the instance.
	ID() InstanceID

	// String typically returns the cloud-provided instance ID.
	String() string

	// Get tags
	Tags(context.Context) (InstanceTags, error)

	// Replace tags with the given tags
	SetTags(context.Context, InstanceTags) error

	// Shut down the node
	Destroy(context.Context) error

	// SSH server hostname or IP address, or empty string if unknown pending creation.
	Address() string
}

type InstanceProvider interface {
	// Create a new instance. If supported by the driver, add the
	// provided public key to /root/.ssh/authorized_keys.
	//
	// The returned error should implement RateLimitError and
	// QuotaError where applicable.
	Create(context.Context, arvados.InstanceType, ImageID, InstanceTags, ssh.PublicKey) (Instance, error)

	// Return all instances, including ones that are booting or
	// shutting down.
	//
	// An instance returned by successive calls to Instances() may
	// -- but does not need to -- be represented by the same
	// Instance object each time. Thus, the caller is responsible
	// for de-duplicating the returned instances by comparing the
	// InstanceIDs returned by the instances' ID() methods.
	Instances(context.Context) ([]Instance, error)

	// Stop any background tasks and release other resources.
	Stop()
}
