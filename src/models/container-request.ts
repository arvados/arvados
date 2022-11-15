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

// Until the api supports unselecting fields, we need a list of all other fields to omit mounts
export const containerRequestFieldsNoMounts = [
    "command",
    "container_count_max",
    "container_count",
    "container_image",
    "container_uuid",
    "created_at",
    "cwd",
    "description",
    "environment",
    "etag",
    "expires_at",
    "filters",
    "href",
    "kind",
    "log_uuid",
    "modified_at",
    "modified_by_client_uuid",
    "modified_by_user_uuid",
    "name",
    "output_name",
    "output_path",
    "output_properties",
    "output_storage_classes",
    "output_ttl",
    "output_uuid",
    "owner_uuid",
    "priority",
    "properties",
    "requesting_container_uuid",
    "runtime_constraints",
    "scheduling_parameters",
    "state",
    "use_existing",
    "uuid",
];
