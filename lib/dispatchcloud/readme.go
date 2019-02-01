// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

// A dispatcher comprises a container queue, a scheduler, a worker
// pool, a remote command executor, and a cloud driver.
// 1. Choose a provider.
// 2. Start a worker pool.
// 3. Start a container queue.
// 4. Run the scheduler's stale-lock fixer.
// 5. Run the scheduler's mapper.
// 6. Run the scheduler's syncer.
// 7. Wait for updates to the container queue or worker pool.
// 8. Repeat from 5.
//
//
// A cloud driver creates new cloud VM instances and gets the latest
// list of instances. The returned instances are caches/proxies for
// the provider's metadata and control interfaces (get IP address,
// update tags, shutdown).
//
//
// A worker pool tracks workers' instance types and readiness states
// (available to do work now, booting, suffering a temporary network
// outage, shutting down). It loads internal state from the cloud
// provider's list of instances at startup, and syncs periodically
// after that.
//
//
// An executor maintains a multiplexed SSH connection to a cloud
// instance, retrying/reconnecting as needed, so the worker pool can
// execute commands. It asks the cloud driver's instance to verify its
// SSH public key once when first connecting, and again later if the
// key changes.
//
//
// A container queue tracks the known state (according to
// arvados-controller) of each container of interest -- i.e., queued,
// or locked/running using our own dispatch token. It also proxies the
// dispatcher's lock/unlock/cancel requests to the controller. It
// handles concurrent refresh and update operations without exposing
// out-of-order updates to its callers. (It drops any new information
// that might have originated before its own most recent
// lock/unlock/cancel operation.)
//
//
// The scheduler's stale-lock fixer waits for any already-locked
// containers (i.e., locked by a prior dispatcher process) to appear
// on workers as the worker pool recovers its state. It
// unlocks/requeues any that still remain when all workers are
// recovered or shutdown, or its timer expires.
//
//
// The scheduler's mapper chooses which containers to assign to which
// idle workers, and decides what to do when there are not enough idle
// workers (including shutting down some idle nodes).
//
//
// The scheduler's syncer updates state to Cancelled when a running
// container process dies without finalizing its entry in the
// controller database. It also calls the worker pool to kill
// containers that have priority=0 while locked or running.
//
//
// An instance set proxy wraps a driver's instance set with
// rate-limiting logic. After the wrapped instance set receives a
// cloud.RateLimitError, the proxy starts returning errors to callers
// immediately without calling through to the wrapped instance set.
