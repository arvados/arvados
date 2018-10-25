// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package dispatchcloud

// A dispatcher comprises a container queue, a scheduler, a worker
// pool, a cloud provider, a stale-lock fixer, and a syncer.
// 1. Choose a provider.
// 2. Start a worker pool.
// 3. Start a container queue.
// 4. Run a stale-lock fixer.
// 5. Start a scheduler.
// 6. Start a syncer.
//
//
// A provider (cloud driver) creates new cloud VM instances and gets
// the latest list of instances. The returned instances implement
// proxies to the provider's metadata and control interfaces (get IP
// address, update tags, shutdown).
//
//
// A workerPool tracks workers' instance types and readiness states
// (available to do work now, booting, suffering a temporary network
// outage, shutting down). It loads internal state from the cloud
// provider's list of instances at startup, and syncs periodically
// after that.
//
//
// A worker maintains a multiplexed SSH connection to a cloud
// instance, retrying/reconnecting as needed, so the workerPool can
// execute commands. It asks the provider's instance to verify its SSH
// public key once when first connecting, and again later if the key
// changes.
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
// A stale-lock fixer waits for any already-locked containers (i.e.,
// locked by a prior server process) to appear on workers as the
// worker pool recovers its state. It unlocks/requeues any that still
// remain when all workers are recovered or shutdown, or its timer
// expires.
//
//
// A scheduler chooses which containers to assign to which idle
// workers, and decides what to do when there are not enough idle
// workers (including shutting down some idle nodes).
//
//
// A syncer updates state to Cancelled when a running container
// process dies without finalizing its entry in the controller
// database. It also calls the worker pool to kill containers that
// have priority=0 while locked or running.
//
//
// A provider proxy wraps a provider with rate-limiting logic. After
// the wrapped provider receives a cloud.RateLimitError, the proxy
// starts returning errors to callers immediately without calling
// through to the wrapped provider.
//
//
// TBD: Bootstrapping script via SSH, too? Future version.
//
// TBD: drain instance, keep instance alive
// TBD: metrics, diagnostics
// TBD: why dispatch token currently passed to worker?
//
// Metrics: queue size, time job has been in queued, #idle/busy/booting nodes
// Timing in each step, and end-to-end
// Metrics: boot/idle/alloc time and cost
