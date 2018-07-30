// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind } from "./resource";

export enum ContainerRequestState {
    UNCOMMITTED = "Uncommitted",
    COMMITTED = "Committed",
    FINAL = "Final"
}

export interface ContainerRequestResource extends Resource {
    kind: ResourceKind.CONTAINER_REQUEST;
    name: string;
    description: string;
    properties: any;
    state: ContainerRequestState;
    requestingContainerUuid: string;
    containerUuid: string;
    containerCountMax: number;
    mounts: any;
    runtimeConstraints: any;
    schedulingParameters: any;
    containerImage: string;
    environment: any;
    cwd: string;
    command: string[];
    outputPath: string;
    outputName: string;
    outputTtl: number;
    priority: number;
    expiresAt: string;
    useExisting: boolean;
    logUuid: string;
    outputUuid: string;
    filters: string;
}
