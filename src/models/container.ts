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
}

export interface ContainerResource extends Resource {
    kind: ResourceKind.CONTAINER;
    state: string;
    startedAt: string | null;
    finishedAt: string | null;
    log: string | null;
    environment: {};
    cwd: string;
    command: string[];
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
    exitCode: number | null;
    authUuid: string | null;
    lockedByUuid: string | null;
}
