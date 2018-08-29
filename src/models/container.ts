// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";

export enum ContainerState {
    QUEUED = 'Queued',
    LOCKED = 'Locked',
    RUNNING = 'Running',
    COMPLETE = 'Complete',
    CANCELLED = 'Cancelled',
}

export interface ContainerResource extends Resource {
    kind: ResourceKind.CONTAINER;
    state: string;
    startedAt: string;
    finishedAt: string;
    log: string;
    environment: {};
    cwd: string;
    command: string[];
    outputPath: string;
    mounts: {};
    runtimeConstraints: {};
    schedulingParameters: {};
    output: string;
    containerImage: string;
    progress: number;
    priority: number;
    exitCode: number;
    authUuid: string;
    lockedByUuid: string;
}
