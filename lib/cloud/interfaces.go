// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package cloud

import (
	"context"
	"io"
	"time"

	"git.curoverse.com/arvados.git/sdk/go/arvados"
	"golang.org/x/crypto/ssh"
)

// A RateLimitError should be returned by an InstanceSet when the
// cloud service indicates it is rejecting all API calls for some time
// interval.
type RateLimitError interface {
	// Time before which the caller should expect requests to
	// fail.
	EarliestRetry() time.Time
	error
}

// A QuotaError should be returned by an InstanceSet when the cloud
// service indicates the account cannot create more VMs than already
// exist.
type QuotaError interface {
	// If true, don't create more instances until some existing
	// instances are destroyed. If false, don't handle the error
	// as a quota error.
	IsQuotaError() bool
	error
}

type InstanceSetID string
type InstanceTags map[string]string
type InstanceID string
type ImageID string

// An Executor executes commands on an ExecutorTarget.
type Executor interface {
	// Update the set of private keys used to authenticate to
	// targets.
	SetSigners(...ssh.Signer)

	// Set the target used for subsequent command executions.
	SetTarget(ExecutorTarget)

	// Return the current target.
	Target() ExecutorTarget

	// Execute a shell command and return the resulting stdout and
	// stderr. stdin can be nil.
	Execute(cmd string, stdin io.Reader) (stdout, stderr []byte, err error)
}

// An ExecutorTarget is a remote command execution service.
type ExecutorTarget interface {
	// SSH server hostname or IP address, or empty string if
	// unknown while instance is booting.
	Address() string

	// Return nil if the given public key matches the instance's
	// SSH server key. If the provided Dialer is not nil,
	// VerifyHostKey can use it to make outgoing network
	// connections from the instance -- e.g., to use the cloud's
	// "this instance's metadata" API.
	VerifyHostKey(context.Context, ssh.PublicKey, *ssh.Client) error
}

// Instance is implemented by the provider-specific instance types.
type Instance interface {
	ExecutorTarget

	// ID returns the provider's instance ID. It must be stable
	// for the life of the instance.
	ID() InstanceID

	// String typically returns the cloud-provided instance ID.
	String() string

	// Cloud provider's "instance type" ID. Matches a ProviderType
	// in the cluster's InstanceTypes configuration.
	ProviderType() string

	// Get current tags
	Tags() InstanceTags

	// Replace tags with the given tags
	SetTags(context.Context, InstanceTags) error

	// Shut down the node
	Destroy(context.Context) error
}

// An InstanceSet manages a set of VM instances created by an elastic
// cloud provider like AWS, GCE, or Azure.
//
// All public methods of an InstanceSet, and all public methods of the
// instances it returns, are goroutine safe.
type InstanceSet interface {
	// Create a new instance. If supported by the driver, add the
	// provided public key to /root/.ssh/authorized_keys.
	//
	// The returned error should implement RateLimitError and
	// QuotaError where applicable.
	Create(context.Context, arvados.InstanceType, ImageID, InstanceTags, ssh.PublicKey) (Instance, error)

	// Return all instances, including ones that are booting or
	// shutting down. Optionally, filter out nodes that don't have
	// all of the given InstanceTags (the caller will ignore these
	// anyway).
	//
	// An instance returned by successive calls to Instances() may
	// -- but does not need to -- be represented by the same
	// Instance object each time. Thus, the caller is responsible
	// for de-duplicating the returned instances by comparing the
	// InstanceIDs returned by the instances' ID() methods.
	Instances(context.Context, InstanceTags) ([]Instance, error)

	// Stop any background tasks and release other resources.
	Stop()
}

// A Driver returns an InstanceSet that uses the given InstanceSetID
// and driver-dependent configuration parameters.
//
// The supplied id will be of the form "zzzzz-zzzzz-zzzzzzzzzzzzzzz"
// where each z can be any alphanum. The returned InstanceSet must use
// this id to tag long-lived cloud resources that it creates, and must
// assume control of any existing resources that are tagged with the
// same id. Tagging can be accomplished by including the ID in
// resource names, using the cloud provider's tagging feature, or any
// other mechanism. The tags must be visible to another instance of
// the same driver running on a different host.
//
// The returned InstanceSet must ignore existing resources that are
// visible but not tagged with the given id, except that it should log
// a summary of such resources -- only once -- when it starts
// up. Thus, two identically configured InstanceSets running on
// different hosts with different ids should log about the existence
// of each other's resources at startup, but will not interfere with
// each other.
//
// Example:
//
//	type exampleInstanceSet struct {
//		ownID     string
//		AccessKey string
//	}
//
//	type exampleDriver struct {}
//
//	func (*exampleDriver) InstanceSet(config map[string]interface{}, id InstanceSetID) (InstanceSet, error) {
//		var is exampleInstanceSet
//		if err := mapstructure.Decode(config, &is); err != nil {
//			return nil, err
//		}
//		is.ownID = id
//		return &is, nil
//	}
//
//	var _ = registerCloudDriver("example", &exampleDriver{})
type Driver interface {
	InstanceSet(config map[string]interface{}, id InstanceSetID) (InstanceSet, error)
}

// DriverFunc makes a Driver using the provided function as its
// InstanceSet method. This is similar to http.HandlerFunc.
func DriverFunc(fn func(config map[string]interface{}, id InstanceSetID) (InstanceSet, error)) Driver {
	return driverFunc(fn)
}

type driverFunc func(config map[string]interface{}, id InstanceSetID) (InstanceSet, error)

func (df driverFunc) InstanceSet(config map[string]interface{}, id InstanceSetID) (InstanceSet, error) {
	return df(config, id)
}
