// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceKind, ResourceWithProperties } from './resource';
import { MountType } from 'models/mount-types';
import { RuntimeConstraints } from './runtime-constraints';
import { SchedulingParameters } from './scheduling-parameters';

export enum ContainerRequestState {
  UNCOMMITTED = 'Uncommitted',
  COMMITTED = 'Committed',
  FINAL = 'Final',
}

export interface ContainerRequestResource
  extends Resource,
    ResourceWithProperties {
  command: string[];
  containerCountMax: number;
  containerCount: number;
  containerImage: string;
  containerUuid: string | null;
  cumulativeCost: number;
  cwd: string;
  description: string;
  environment: any;
  expiresAt: string;
  filters: string;
  kind: ResourceKind.CONTAINER_REQUEST;
  logUuid: string | null;
  mounts: { [path: string]: MountType };
  name: string;
  outputName: string;
  outputPath: string;
  outputProperties: any;
  outputStorageClasses: string[];
  outputTtl: number;
  outputUuid: string | null;
  priority: number | null;
  requestingContainerUuid: string | null;
  runtimeConstraints: RuntimeConstraints;
  schedulingParameters: SchedulingParameters;
  state: ContainerRequestState;
  useExisting: boolean;
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
