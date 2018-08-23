// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
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

type InstanceTag string
type InstanceID string
type ImageID string

// instance is implemented by the provider-specific instance types.
type Instance interface {
	// String typically returns the cloud-provided instance ID.
	String() string
	// Configured Arvados instance type
	InstanceType() arvados.InstanceType
	// Get tags
	GetTags() ([]InstanceTag, error)
	// Replace tags with the given tags
	SetTags([]InstanceTag) error
	// Shut down the node
	Destroy(ctx context.Context) error
	// SSH server hostname or IP address, or empty string if unknown pending creation.
	Address() string
}

type Provider interface {
	Create(context.Context, arvados.InstanceType, ImageID, []InstanceTag) (Instance, error)
	Instances(context.Context) ([]Instance, error)
}
