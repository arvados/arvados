// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

import (
	"context"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
)

type InstanceTag string
type InstanceID string
type ImageID string

// instance is implemented by the provider-specific instance types.
type Instance interface {
	// String typically returns the cloud-provided instance ID.
	String() string
	// Cloud provider's "instance type" ID. Matches a key in
	// configured arvados.InstanceTypeMap.
	ProviderType() string
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
