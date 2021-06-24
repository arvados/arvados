// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import { Resource } from "./resource";
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
}

export interface LogResource extends Resource {
    kind: ResourceKind.LOG;
    objectUuid: string;
    eventAt: string;
    eventType: string;
    summary: string;
    properties: any;
}
