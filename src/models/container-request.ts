// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind, ResourceWithProperties } from "./resource";
import { MountType } from "models/mount-types";
import { RuntimeConstraints } from './runtime-constraints';
import { SchedulingParameters } from './scheduling-parameters';

export enum ContainerRequestState {
    UNCOMMITTED = "Uncommitted",
    COMMITTED = "Committed",
    FINAL = "Final"
}

export interface ContainerRequestResource extends Resource, ResourceWithProperties {
    kind: ResourceKind.CONTAINER_REQUEST;
    name: string;
    description: string;
    state: ContainerRequestState;
    requestingContainerUuid: string | null;
    cumulativeCost: number;
    containerUuid: string | null;
    containerCountMax: number;
    mounts: {[path: string]: MountType};
    runtimeConstraints: RuntimeConstraints;
    schedulingParameters: SchedulingParameters;
    containerImage: string;
    environment: any;
    cwd: string;
    command: string[];
    outputPath: string;
    outputName: string;
    outputTtl: number;
    priority: number | null;
    expiresAt: string;
    useExisting: boolean;
    logUuid: string | null;
    outputUuid: string | null;
    filters: string;
    containerCount: number;
}
