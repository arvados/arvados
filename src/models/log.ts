// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource, ResourceWithProperties } from "./resource";
import { ResourceKind } from 'models/resource';

export enum LogEventType {
    CREATE = 'create',
    UPDATE = 'update',
    DISPATCH = 'dispatch',
    CRUNCH_RUN = 'crunch-run',
    CRUNCHSTAT = 'crunchstat',
    HOSTSTAT = 'hoststat',
    NODE_INFO = 'node-info',
    ARV_MOUNT = 'arv-mount',
    STDOUT = 'stdout',
    STDERR = 'stderr',
    CONTAINER = 'container',
}

export interface LogResource extends Resource, ResourceWithProperties {
    kind: ResourceKind.LOG;
    objectUuid: string;
    eventAt: string;
    eventType: LogEventType;
    summary: string;
}
