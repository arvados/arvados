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
  outputTtl: number;
  outputUuid: string | null;
  priority: number | null;
  requestingContainerUuid: string | null;
  runtimeConstraints: RuntimeConstraints;
  schedulingParameters: SchedulingParameters;
  state: ContainerRequestState;
  useExisting: boolean;
}
