// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";
import { MountType } from 'models/mount-types';
import { RuntimeConstraints } from "models/runtime-constraints";
import { SchedulingParameters } from './scheduling-parameters';
import { RuntimeStatus } from "./runtime-status";

export enum ContainerState {
    QUEUED = 'Queued',
    LOCKED = 'Locked',
    RUNNING = 'Running',
    COMPLETE = 'Complete',
    CANCELLED = 'Cancelled',
    UNCOMMITTED = 'Uncommitted',
}

/**
 * Schema for published service ports
 * camelcase is not used due to canonical mapKeys behavior with certain nested structures
 * base_url, initial_url, and external_port are observed to not always be present
 */
export type PublishedPort = {
    access: 'public' | 'private';
    label: string;
    base_url?: string;
    initial_path: string;
    initial_url?: string;
    external_port?: number;
};

export interface ContainerResource extends Resource {
    kind: ResourceKind.CONTAINER;
    state: string;
    startedAt: string | null;
    finishedAt: string | null;
    log: string | null;
    environment: {};
    cwd: string;
    command: string[];
    cost: number;
    outputPath: string;
    mounts: MountType[];
    runtimeConstraints: RuntimeConstraints;
    runtimeStatus: RuntimeStatus;
    runtimeUserUuid: string;
    schedulingParameters: SchedulingParameters;
    output: string | null;
    containerImage: string;
    progress: number;
    priority: number;
    publishedPorts: Record<string, PublishedPort>;
    exitCode: number | null;
    authUuid: string | null;
    lockedByUuid: string | null;
}
